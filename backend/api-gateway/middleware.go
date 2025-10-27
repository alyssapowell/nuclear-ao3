package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"nuclear-ao3/shared/models"
)

// =============================================================================
// MIDDLEWARE IMPLEMENTATIONS
// =============================================================================

// CORSMiddleware handles Cross-Origin Resource Sharing with production-ready configuration
func CORSMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Get environment-based configuration
		env := getEnv("GO_ENV", "development")
		allowAllOrigins := getEnv("CORS_ALLOW_ALL", "false") == "true"
		wildcardMode := getEnv("CORS_WILDCARD", "false") == "true"

		var isAllowed bool

		if wildcardMode || allowAllOrigins {
			// Wildcard mode - Allow all origins (for platforms and APIs)
			isAllowed = true
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			} else {
				c.Header("Access-Control-Allow-Origin", "*")
			}
		} else {
			// Specific origin checking
			allowedOrigins := getAllowedOrigins()
			isAllowed = isOriginAllowed(origin, allowedOrigins)

			if isAllowed && origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		}

		// Set comprehensive CORS headers for all requests
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With, X-API-Key")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if isAllowed || wildcardMode || allowAllOrigins {
				c.AbortWithStatus(204)
			} else {
				c.AbortWithStatus(403)
			}
			return
		}

		// Log CORS issues in development for debugging
		if !isAllowed && env == "development" {
			log.Printf("CORS: Origin %s not in allowed list", origin)
		}

		c.Next()
	})
}

// getAllowedOrigins returns the list of allowed origins based on environment
func getAllowedOrigins() []string {
	env := getEnv("GO_ENV", "development")

	// Base allowed origins
	origins := []string{
		// Production domains (from environment variables)
		getEnv("FRONTEND_URL", ""),
		getEnv("ADMIN_URL", ""),

		// Trusted external origins
		"https://ao3.org",
		"https://archiveofourown.org",
	}

	// Add development origins only in dev/test environments
	if env != "production" {
		devOrigins := []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:3002",
			"http://localhost:3003",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:3001",
			"http://127.0.0.1:3002",
			"http://127.0.0.1:3003",
			"http://0.0.0.0:3000",
			"http://0.0.0.0:3001",
		}
		origins = append(origins, devOrigins...)
	}

	// Add deployment platform origins
	platformOrigins := []string{
		getEnv("VERCEL_URL", ""),
		getEnv("NETLIFY_URL", ""),
		getEnv("HEROKU_URL", ""),
	}
	origins = append(origins, platformOrigins...)

	// Filter out empty origins
	var filtered []string
	for _, origin := range origins {
		if origin != "" {
			filtered = append(filtered, origin)
		}
	}

	return filtered
}

// isOriginAllowed checks if an origin is in the allowed list or matches patterns
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}

	// Check exact matches first
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}

	// Check deployment platform patterns
	trustedPatterns := []string{
		".vercel.app",
		".netlify.app",
		".herokuapp.com",
		".railway.app",
		".fly.dev",
	}

	for _, pattern := range trustedPatterns {
		if strings.HasSuffix(origin, pattern) {
			// Additional security: ensure HTTPS for deployment platforms
			if strings.HasPrefix(origin, "https://") {
				return true
			}
		}
	}

	return false
}

// LoggingMiddleware provides structured logging for all requests
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[GATEWAY] %v | %3d | %13v | %15s | %-7s %#v\n%s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	})
}

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// XSS Protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Strict Transport Security (HTTPS only)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Content Security Policy for GraphQL Playground
		if strings.Contains(c.Request.URL.Path, "/graphql") && c.Request.Method == "GET" {
			c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com")
		} else {
			c.Header("Content-Security-Policy", "default-src 'self'")
		}

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// MetricsMiddleware tracks request metrics
func MetricsMiddleware(metrics *GatewayMetrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)
		metrics.RecordRequest(c.Request.Method, c.FullPath(), c.Writer.Status(), duration)
	}
}

// RateLimitMiddleware returns a middleware function for OAuth-aware rate limiting
func (gw *APIGateway) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if gw.rateLimiter == nil {
			c.Next()
			return
		}

		// Extract OAuth client information for intelligent rate limiting
		clientInfo := gw.extractClientRateLimitInfo(c)

		// Get rate limit configuration for this client
		config := clientInfo.GetRateLimitConfig()

		// Generate appropriate rate limit key
		var rateLimitKey string
		if clientInfo.Tier == models.RateLimitTierAnonymous {
			rateLimitKey = clientInfo.GenerateRateLimitKey() + ":" + c.ClientIP()
		} else {
			rateLimitKey = clientInfo.GenerateRateLimitKey()
		}

		// Check rate limit using the client-specific configuration
		allowed, remaining, resetTime := gw.rateLimiter.CheckLimitWithConfig(
			c.Request.Context(),
			rateLimitKey,
			config.Requests,
			config.Window,
		)

		// Set comprehensive rate limit headers
		headers := &models.RateLimitHeaders{
			Limit:     config.Requests,
			Remaining: remaining,
			Reset:     resetTime.Unix(),
			Tier:      string(clientInfo.Tier),
		}

		for key, value := range headers.ToHeaders() {
			c.Header(key, value)
		}

		if !allowed {
			// Record rate limit hit for monitoring
			if gw.metrics != nil {
				gw.metrics.RecordRateLimitHit()
			}

			c.JSON(429, gin.H{
				"error":       "Rate limit exceeded",
				"tier":        clientInfo.Tier,
				"limit":       config.Requests,
				"window":      config.Window.String(),
				"retry_after": time.Until(resetTime).Seconds(),
			})
			c.Abort()
			return
		}

		// Add client info to context for downstream services
		c.Set("rate_limit_tier", clientInfo.Tier)
		c.Set("client_info", clientInfo)

		c.Next()
	}
}

// extractClientRateLimitInfo extracts OAuth client information for rate limiting decisions
func (gw *APIGateway) extractClientRateLimitInfo(c *gin.Context) *models.ClientRateLimitInfo {
	// Initialize with anonymous defaults
	info := &models.ClientRateLimitInfo{
		Tier: models.RateLimitTierAnonymous,
	}

	// Try to extract OAuth token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return info // Anonymous request
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return info // Empty token
	}

	// Look up OAuth client information
	// In a production system, this would validate the token and look up client info
	// For now, we'll implement a simplified version
	clientInfo := gw.lookupOAuthClientFromToken(token)
	if clientInfo != nil {
		info.ClientID = clientInfo.ClientID
		info.IsFirstParty = clientInfo.IsFirstParty
		info.IsTrusted = clientInfo.IsTrusted
		info.Scopes = clientInfo.Scopes
		info.UserID = clientInfo.UserID

		// Check for admin scopes
		info.IsAdmin = containsAdminScope(clientInfo.Scopes)
	}

	return info
}

// lookupOAuthClientFromToken looks up OAuth client information from a token
// This is a simplified implementation - in production, this would validate the token
// and query the database for client information
func (gw *APIGateway) lookupOAuthClientFromToken(token string) *models.ClientRateLimitInfo {
	// TODO: Implement proper OAuth token validation and client lookup
	// For now, return nil to indicate anonymous access
	// This will need to integrate with the auth service or shared OAuth validation
	return nil
}

// containsAdminScope checks if the scopes contain admin privileges
func containsAdminScope(scopes []string) bool {
	adminScopes := []string{"admin", "tags:wrangle", "moderation"}
	for _, scope := range scopes {
		for _, adminScope := range adminScopes {
			if scope == adminScope {
				return true
			}
		}
	}
	return false
}

// JWTAuthMiddleware validates JWT tokens and extracts user information
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for certain endpoints
		path := c.Request.URL.Path
		if isPublicEndpoint(path) {
			c.Next()
			return
		}

		// Extract Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Allow anonymous access for public endpoints
			if isOptionalAuthEndpoint(path) {
				c.Next()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing authorization header",
			})
			c.Abort()
			return
		}

		// Extract Bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Empty bearer token",
			})
			c.Abort()
			return
		}

		// Validate token with auth service
		userID, err := validateTokenWithAuthService(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid token",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// Set user_id in context for downstream services
		c.Set("user_id", userID)
		c.Next()
	}
}

// isPublicEndpoint checks if an endpoint is public and doesn't require authentication
func isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/health",
		"/metrics",
		"/status",
		"/graphql", // GraphQL playground
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/token",
		"/api/v1/auth/refresh",
		"/api/v1/auth/jwks",
		"/api/v1/auth/oauth",
		"/api/v1/tags/search", // Public tag search
		"/api/v1/search",      // Public search
	}

	for _, publicPath := range publicPaths {
		if strings.HasPrefix(path, publicPath) {
			return true
		}
	}

	return false
}

// isOptionalAuthEndpoint checks if an endpoint allows both authenticated and anonymous access
func isOptionalAuthEndpoint(path string) bool {
	optionalAuthPaths := []string{
		"/api/v1/works",  // Public works viewing
		"/api/v1/tags",   // Public tag browsing
		"/api/v1/search", // Public search
	}

	for _, optionalPath := range optionalAuthPaths {
		if strings.HasPrefix(path, optionalPath) {
			return true
		}
	}

	return false
}

// validateTokenWithAuthService validates a JWT token by calling the auth service
func validateTokenWithAuthService(tokenString string) (string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Create request to auth service /me endpoint which validates the token
	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8081")
	req, err := http.NewRequest("GET", authServiceURL+"/api/v1/auth/me", nil)
	if err != nil {
		return "", err
	}

	// Set Authorization header for validation
	req.Header.Set("Authorization", "Bearer "+tokenString)

	// Make request to auth service
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to validate token with auth service: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth service returned status %d", resp.StatusCode)
	}

	// Parse response to extract user ID
	var meResponse struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&meResponse); err != nil {
		return "", fmt.Errorf("failed to parse auth service response: %v", err)
	}

	return meResponse.UserID, nil
}
