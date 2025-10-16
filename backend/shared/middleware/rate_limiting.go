package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"nuclear-ao3/shared/models"
)

// RateLimitManager handles OAuth-aware rate limiting for any service
type RateLimitManager struct {
	redisClient *redis.Client
	serviceName string
}

// NewRateLimitManager creates a new rate limiting manager
func NewRateLimitManager(redisClient *redis.Client, serviceName string) *RateLimitManager {
	return &RateLimitManager{
		redisClient: redisClient,
		serviceName: serviceName,
	}
}

// CheckRateLimit performs OAuth-aware rate limiting check
func (rlm *RateLimitManager) CheckRateLimit(clientInfo *models.ClientRateLimitInfo, clientIP string) (*models.RateLimitHeaders, error) {
	config := clientInfo.GetRateLimitConfig()

	// Generate the appropriate Redis key
	var key string
	if clientInfo.DetermineRateLimitTier() == models.RateLimitTierAnonymous {
		// For anonymous requests, use IP-based limiting
		key = fmt.Sprintf("rate_limit:%s:%s:%s", rlm.serviceName, string(config.Tier), clientIP)
	} else {
		// For OAuth clients, use client-based limiting
		limitKey := clientInfo.GenerateRateLimitKey()
		key = fmt.Sprintf("rate_limit:%s:%s", rlm.serviceName, limitKey)
	}

	return rlm.checkLimitWithConfig(key, config)
}

// checkLimitWithConfig performs the actual Redis-based rate limiting
func (rlm *RateLimitManager) checkLimitWithConfig(key string, config models.RateLimitConfig) (*models.RateLimitHeaders, error) {
	ctx := context.Background()
	now := time.Now()
	windowStart := now.Truncate(config.Window)
	windowEnd := windowStart.Add(config.Window)

	// Use Redis pipeline for atomic operations
	pipe := rlm.redisClient.Pipeline()

	// Get current count
	countCmd := pipe.Get(ctx, key)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		log.Printf("Redis error in rate limiting: %v", err)
		// Fail open - allow request if Redis is down
		return &models.RateLimitHeaders{
			Limit:     config.Requests,
			Remaining: config.Requests - 1,
			Reset:     windowEnd.Unix(),
			Tier:      string(config.Tier),
		}, nil
	}

	// Get current count
	currentCount := 0
	if countStr, err := countCmd.Result(); err == nil {
		if count, parseErr := strconv.Atoi(countStr); parseErr == nil {
			currentCount = count
		}
	}

	// Check if limit exceeded
	if currentCount >= config.Requests {
		return &models.RateLimitHeaders{
			Limit:     config.Requests,
			Remaining: 0,
			Reset:     windowEnd.Unix(),
			Tier:      string(config.Tier),
		}, fmt.Errorf("rate limit exceeded")
	}

	// Increment counter
	pipe = rlm.redisClient.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, config.Window)
	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Printf("Redis error incrementing rate limit: %v", err)
		// Fail open
	}

	return &models.RateLimitHeaders{
		Limit:     config.Requests,
		Remaining: config.Requests - currentCount - 1,
		Reset:     windowEnd.Unix(),
		Tier:      string(config.Tier),
	}, nil
}

// ExtractOAuthInfo extracts OAuth information from request headers
func ExtractOAuthInfo(r *http.Request) *models.ClientRateLimitInfo {
	// Default to anonymous
	info := &models.ClientRateLimitInfo{
		Tier: models.RateLimitTierAnonymous,
	}

	// Check for Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return info
	}

	// Extract Bearer token
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return info
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return info
	}

	// TODO: In a real implementation, you would:
	// 1. Validate the JWT/OAuth token
	// 2. Extract client_id and scopes from the token
	// 3. Look up client details from database
	// 4. Determine trust level and first-party status

	// For now, we'll extract from custom headers that the API Gateway sets
	// when it validates tokens (this is a common pattern)

	if clientID := r.Header.Get("X-Client-ID"); clientID != "" {
		info.ClientID = clientID
	}

	if userID := r.Header.Get("X-User-ID"); userID != "" {
		info.UserID = userID
	}

	if scopes := r.Header.Get("X-OAuth-Scopes"); scopes != "" {
		info.Scopes = strings.Split(scopes, ",")
	}

	if isFirstParty := r.Header.Get("X-Client-First-Party"); isFirstParty == "true" {
		info.IsFirstParty = true
	}

	if isTrusted := r.Header.Get("X-Client-Trusted"); isTrusted == "true" {
		info.IsTrusted = true
	}

	if isAdmin := r.Header.Get("X-Client-Admin"); isAdmin == "true" {
		info.IsAdmin = true
	}

	return info
}

// GetClientIP extracts the real client IP from request headers
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (set by load balancers/proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, use the first one
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header (set by some proxies)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	return ip
}

// RateLimitMiddleware creates an HTTP middleware for OAuth-aware rate limiting
func (rlm *RateLimitManager) RateLimitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract OAuth information from request
			clientInfo := ExtractOAuthInfo(r)
			clientIP := GetClientIP(r)

			// Check rate limit
			headers, err := rlm.CheckRateLimit(clientInfo, clientIP)
			if err != nil {
				// Add rate limit headers even on error
				for key, value := range headers.ToHeaders() {
					w.Header().Set(key, value)
				}

				// Return 429 Too Many Requests
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "Rate limit exceeded",
					"limit": headers.Limit,
					"reset": headers.Reset,
					"tier":  headers.Tier,
				})
				return
			}

			// Add rate limit headers to response
			for key, value := range headers.ToHeaders() {
				w.Header().Set(key, value)
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// GinRateLimitMiddleware creates a Gin middleware for OAuth-aware rate limiting
// This is useful for services that use the Gin framework
func (rlm *RateLimitManager) GinRateLimitMiddleware() func(c interface{}) {
	return func(c interface{}) {
		// This would be implemented if we're using Gin framework
		// For now, we'll focus on standard HTTP middleware
		panic("Gin middleware not implemented yet - use RateLimitMiddleware() for standard HTTP")
	}
}

// RateLimitCheck is a simple function that services can call directly
// without using middleware (useful for background jobs, etc.)
func (rlm *RateLimitManager) RateLimitCheck(clientID string, userID string, scopes []string, isFirstParty bool, isTrusted bool, clientIP string) (*models.RateLimitHeaders, error) {
	clientInfo := &models.ClientRateLimitInfo{
		ClientID:     clientID,
		UserID:       userID,
		Scopes:       scopes,
		IsFirstParty: isFirstParty,
		IsTrusted:    isTrusted,
		IsAdmin:      false, // Check scopes for admin
	}

	// Check for admin scopes
	for _, scope := range scopes {
		if scope == "admin" || scope == "tags:wrangle" {
			clientInfo.IsAdmin = true
			break
		}
	}

	return rlm.CheckRateLimit(clientInfo, clientIP)
}
