package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Rate limiting types and constants
type RateLimitTier string

const (
	RateLimitTierAnonymous  RateLimitTier = "anonymous"
	RateLimitTierPublic     RateLimitTier = "public"
	RateLimitTierTrusted    RateLimitTier = "trusted"
	RateLimitTierFirstParty RateLimitTier = "first_party"
	RateLimitTierAdmin      RateLimitTier = "admin"
)

type RateLimitConfig struct {
	Tier     RateLimitTier `json:"tier"`
	Requests int           `json:"requests"`
	Window   time.Duration `json:"window"`
	Burst    int           `json:"burst"`
}

type ClientRateLimitInfo struct {
	ClientID     string        `json:"client_id"`
	Tier         RateLimitTier `json:"tier"`
	IsFirstParty bool          `json:"is_first_party"`
	IsTrusted    bool          `json:"is_trusted"`
	IsAdmin      bool          `json:"is_admin"`
	Scopes       []string      `json:"scopes"`
	UserID       string        `json:"user_id,omitempty"`
}

type RateLimitHeaders struct {
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
	Reset     int64  `json:"reset"`
	Tier      string `json:"tier"`
}

type RateLimitManager struct {
	redisClient *redis.Client
	serviceName string
}

// Helper functions for rate limiting
func GetDefaultRateLimitConfigs() map[RateLimitTier]RateLimitConfig {
	return map[RateLimitTier]RateLimitConfig{
		RateLimitTierAnonymous: {
			Tier:     RateLimitTierAnonymous,
			Requests: 100,
			Window:   time.Minute,
			Burst:    20,
		},
		RateLimitTierPublic: {
			Tier:     RateLimitTierPublic,
			Requests: 1000,
			Window:   time.Minute,
			Burst:    100,
		},
		RateLimitTierTrusted: {
			Tier:     RateLimitTierTrusted,
			Requests: 5000,
			Window:   time.Minute,
			Burst:    500,
		},
		RateLimitTierFirstParty: {
			Tier:     RateLimitTierFirstParty,
			Requests: 10000,
			Window:   time.Minute,
			Burst:    1000,
		},
		RateLimitTierAdmin: {
			Tier:     RateLimitTierAdmin,
			Requests: 50000,
			Window:   time.Minute,
			Burst:    5000,
		},
	}
}

func (info *ClientRateLimitInfo) DetermineRateLimitTier() RateLimitTier {
	if info.IsAdmin || containsScope(info.Scopes, "admin") || containsScope(info.Scopes, "tags:wrangle") {
		return RateLimitTierAdmin
	}
	if info.IsFirstParty {
		return RateLimitTierFirstParty
	}
	if info.IsTrusted {
		return RateLimitTierTrusted
	}
	if info.ClientID != "" {
		return RateLimitTierPublic
	}
	return RateLimitTierAnonymous
}

func (info *ClientRateLimitInfo) GetRateLimitConfig() RateLimitConfig {
	configs := GetDefaultRateLimitConfigs()
	tier := info.DetermineRateLimitTier()
	return configs[tier]
}

func containsScope(scopes []string, target string) bool {
	for _, scope := range scopes {
		if scope == target {
			return true
		}
	}
	return false
}

func (h *RateLimitHeaders) ToHeaders() map[string]string {
	return map[string]string{
		"X-RateLimit-Limit":     fmt.Sprintf("%d", h.Limit),
		"X-RateLimit-Remaining": fmt.Sprintf("%d", h.Remaining),
		"X-RateLimit-Reset":     fmt.Sprintf("%d", h.Reset),
		"X-RateLimit-Tier":      h.Tier,
	}
}

func ExtractOAuthInfo(r *http.Request) *ClientRateLimitInfo {
	info := &ClientRateLimitInfo{
		Tier: RateLimitTierAnonymous,
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return info
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return info
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return info
	}

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

func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	ip := r.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	return ip
}

func (rlm *RateLimitManager) CheckRateLimit(clientInfo *ClientRateLimitInfo, clientIP string) (*RateLimitHeaders, error) {
	config := clientInfo.GetRateLimitConfig()

	var key string
	if clientInfo.DetermineRateLimitTier() == RateLimitTierAnonymous {
		key = fmt.Sprintf("rate_limit:%s:%s:%s", rlm.serviceName, string(config.Tier), clientIP)
	} else {
		key = fmt.Sprintf("rate_limit:%s:%s:%s", rlm.serviceName, string(config.Tier), clientInfo.ClientID)
	}

	return rlm.checkLimitWithConfig(key, config)
}

func (rlm *RateLimitManager) checkLimitWithConfig(key string, config RateLimitConfig) (*RateLimitHeaders, error) {
	ctx := context.Background()
	now := time.Now()
	windowStart := now.Truncate(config.Window)
	windowEnd := windowStart.Add(config.Window)

	pipe := rlm.redisClient.Pipeline()
	countCmd := pipe.Get(ctx, key)
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		log.Printf("Redis error in rate limiting: %v", err)
		return &RateLimitHeaders{
			Limit:     config.Requests,
			Remaining: config.Requests - 1,
			Reset:     windowEnd.Unix(),
			Tier:      string(config.Tier),
		}, nil
	}

	currentCount := 0
	if countStr, err := countCmd.Result(); err == nil {
		if count, parseErr := strconv.Atoi(countStr); parseErr == nil {
			currentCount = count
		}
	}

	if currentCount >= config.Requests {
		return &RateLimitHeaders{
			Limit:     config.Requests,
			Remaining: 0,
			Reset:     windowEnd.Unix(),
			Tier:      string(config.Tier),
		}, fmt.Errorf("rate limit exceeded")
	}

	pipe = rlm.redisClient.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, config.Window)
	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Printf("Redis error incrementing rate limit: %v", err)
	}

	return &RateLimitHeaders{
		Limit:     config.Requests,
		Remaining: config.Requests - currentCount - 1,
		Reset:     windowEnd.Unix(),
		Tier:      string(config.Tier),
	}, nil
}

// JWTAuthMiddleware validates JWT tokens
func JWTAuthMiddleware(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for test user header (for testing only)
		if testUserID := c.GetHeader("X-Test-User-ID"); testUserID != "" && gin.Mode() == gin.TestMode {
			if userID, err := uuid.Parse(testUserID); err == nil {
				c.Set("user_id", userID)
				c.Next()
				return
			}
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":             "missing_authorization_header",
				"error_description": "Authorization header is required",
			})
			c.Abort()
			return
		}

		tokenString := extractBearerToken(authHeader)
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_authorization_header",
				"error_description": "Bearer token required",
			})
			c.Abort()
			return
		}

		claims, err := authService.jwt.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "Token validation failed",
			})
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "Invalid user ID in token",
			})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("token_claims", claims)
		c.Next()
	}
}

// RequireRoleMiddleware checks if user has required role
func RequireRoleMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":             "unauthorized",
				"error_description": "User authentication required",
			})
			c.Abort()
			return
		}

		// TODO: Check user roles from database
		// For now, just allow admin users for testing
		if requiredRole == "admin" {
			// In a real implementation, you'd query the database for user roles
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":             "insufficient_permissions",
			"error_description": fmt.Sprintf("Role '%s' required", requiredRole),
		})
		c.Abort()
	}
}

// RateLimitMiddleware implements OAuth-aware rate limiting using the DRY helper
func RateLimitMiddleware(redis *redis.Client) gin.HandlerFunc {
	// Create our DRY rate limit manager
	rateLimitManager := &RateLimitManager{
		redisClient: redis,
		serviceName: "auth-service",
	}

	return func(c *gin.Context) {
		if gin.Mode() == gin.TestMode {
			// Skip rate limiting in test mode
			c.Next()
			return
		}

		// Extract OAuth information from request headers
		clientInfo := ExtractOAuthInfo(c.Request)
		clientIP := GetClientIP(c.Request)

		// Check rate limit using DRY helper
		headers, err := rateLimitManager.CheckRateLimit(clientInfo, clientIP)
		if err != nil {
			// Add rate limit headers even on error
			for key, value := range headers.ToHeaders() {
				c.Header(key, value)
			}

			// Return 429 Too Many Requests with OAuth-aware messaging
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":             "rate_limit_exceeded",
				"error_description": "Too many requests. Please try again later.",
				"limit":             headers.Limit,
				"reset":             headers.Reset,
				"tier":              headers.Tier,
			})
			c.Abort()
			return
		}

		// Add rate limit headers to response
		for key, value := range headers.ToHeaders() {
			c.Header(key, value)
		}

		c.Next()
	}
}

// Helper function to extract bearer token from Authorization header
func extractBearerToken(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}
