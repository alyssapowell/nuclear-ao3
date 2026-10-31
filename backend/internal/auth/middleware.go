package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

// RateLimitManager counts requests per client in memory. Each copy of the
// app enforces the limits on its own.
type RateLimitManager struct {
	counter     *windowCounter
	serviceName string
}

func newRateLimitManager(serviceName string) *RateLimitManager {
	return &RateLimitManager{counter: newWindowCounter(maxRateLimitKeys), serviceName: serviceName}
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

// rateLimitInfo works out who is making a request, for rate limiting. It
// trusts only a verified login token: logged-in users get their own bucket,
// everyone else is limited per IP. Client-supplied headers such as
// X-Client-Admin are ignored.
func (as *AuthService) rateLimitInfo(r *http.Request) *ClientRateLimitInfo {
	info := &ClientRateLimitInfo{Tier: RateLimitTierAnonymous}
	if token := extractBearerToken(r.Header.Get("Authorization")); token != "" {
		if userID, err := as.ValidateUserToken(token); err == nil {
			info.UserID = userID.String()
			info.ClientID = "user:" + info.UserID
		}
	}
	return info
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
	used, windowEnd, allowed := rlm.counter.hit(key, config.Requests, config.Window)
	headers := &RateLimitHeaders{
		Limit:     config.Requests,
		Remaining: config.Requests - used - 1,
		Reset:     windowEnd.Unix(),
		Tier:      string(config.Tier),
	}
	if !allowed {
		headers.Remaining = 0
		return headers, fmt.Errorf("rate limit exceeded")
	}
	return headers, nil
}

// JWTAuthMiddleware validates JWT tokens
func JWTAuthMiddleware(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		userID, sessionID, err := authService.validateSession(c, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "Token validation failed",
			})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("user_uuid", userID)
		c.Set("session_id", sessionID)

		c.Next()
	}
}

// RateLimitMiddleware limits how fast each user (or, for anonymous
// visitors, each IP) can make requests. The IP comes from gin's ClientIP,
// which only believes X-Forwarded-For from trusted proxies (see main).
func (as *AuthService) RateLimitMiddleware() gin.HandlerFunc {
	rateLimitManager := newRateLimitManager("app")

	return func(c *gin.Context) {
		if gin.Mode() == gin.TestMode {
			c.Next()
			return
		}

		headers, err := rateLimitManager.CheckRateLimit(as.rateLimitInfo(c.Request), c.ClientIP())
		for key, value := range headers.ToHeaders() {
			c.Header(key, value)
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":             "rate_limit_exceeded",
				"error_description": "Too many requests. Please try again later.",
				"limit":             headers.Limit,
				"reset":             headers.Reset,
				"tier":              headers.Tier,
			})
			return
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
