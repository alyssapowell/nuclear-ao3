package models

import (
	"fmt"
	"time"
)

// RateLimitTier represents different rate limiting tiers based on client trust level
type RateLimitTier string

const (
	// RateLimitTierAnonymous for unauthenticated requests (IP-based)
	RateLimitTierAnonymous RateLimitTier = "anonymous"

	// RateLimitTierPublic for public OAuth clients (third-party apps)
	RateLimitTierPublic RateLimitTier = "public"

	// RateLimitTierTrusted for trusted OAuth clients (verified third-party apps)
	RateLimitTierTrusted RateLimitTier = "trusted"

	// RateLimitTierFirstParty for first-party AO3 applications (web app, mobile app)
	RateLimitTierFirstParty RateLimitTier = "first_party"

	// RateLimitTierAdmin for administrative operations
	RateLimitTierAdmin RateLimitTier = "admin"
)

// RateLimitConfig defines rate limiting parameters for each tier
type RateLimitConfig struct {
	Tier     RateLimitTier `json:"tier"`
	Requests int           `json:"requests"` // Number of requests allowed
	Window   time.Duration `json:"window"`   // Time window for rate limiting
	Burst    int           `json:"burst"`    // Additional burst capacity
}

// GetDefaultRateLimitConfigs returns the standard rate limiting configuration
// This is the single source of truth for rate limits across all services
func GetDefaultRateLimitConfigs() map[RateLimitTier]RateLimitConfig {
	return map[RateLimitTier]RateLimitConfig{
		RateLimitTierAnonymous: {
			Tier:     RateLimitTierAnonymous,
			Requests: 100, // 100 requests/minute
			Window:   time.Minute,
			Burst:    20, // Allow small bursts
		},
		RateLimitTierPublic: {
			Tier:     RateLimitTierPublic,
			Requests: 1000, // 1,000 requests/minute
			Window:   time.Minute,
			Burst:    100, // Larger burst for apps
		},
		RateLimitTierTrusted: {
			Tier:     RateLimitTierTrusted,
			Requests: 5000, // 5,000 requests/minute
			Window:   time.Minute,
			Burst:    500, // Significant burst capacity
		},
		RateLimitTierFirstParty: {
			Tier:     RateLimitTierFirstParty,
			Requests: 10000, // 10,000 requests/minute
			Window:   time.Minute,
			Burst:    1000, // High burst for first-party apps
		},
		RateLimitTierAdmin: {
			Tier:     RateLimitTierAdmin,
			Requests: 50000, // 50,000 requests/minute
			Window:   time.Minute,
			Burst:    5000, // Very high limits for admin operations
		},
	}
}

// ClientRateLimitInfo contains information needed for rate limiting decisions
type ClientRateLimitInfo struct {
	ClientID     string        `json:"client_id"`
	Tier         RateLimitTier `json:"tier"`
	IsFirstParty bool          `json:"is_first_party"`
	IsTrusted    bool          `json:"is_trusted"`
	IsAdmin      bool          `json:"is_admin"`
	Scopes       []string      `json:"scopes"`
	UserID       string        `json:"user_id,omitempty"`
}

// DetermineRateLimitTier calculates the appropriate rate limit tier for a client
func (info *ClientRateLimitInfo) DetermineRateLimitTier() RateLimitTier {
	// Admin scopes get highest priority
	if info.IsAdmin || containsScope(info.Scopes, "admin") || containsScope(info.Scopes, "tags:wrangle") {
		return RateLimitTierAdmin
	}

	// First-party applications get high limits
	if info.IsFirstParty {
		return RateLimitTierFirstParty
	}

	// Trusted third-party clients get elevated limits
	if info.IsTrusted {
		return RateLimitTierTrusted
	}

	// Public OAuth clients get standard limits
	if info.ClientID != "" {
		return RateLimitTierPublic
	}

	// Default to anonymous limits
	return RateLimitTierAnonymous
}

// GetRateLimitConfig returns the rate limit configuration for this client
func (info *ClientRateLimitInfo) GetRateLimitConfig() RateLimitConfig {
	configs := GetDefaultRateLimitConfigs()
	tier := info.DetermineRateLimitTier()
	return configs[tier]
}

// GenerateRateLimitKey creates a consistent rate limiting key for Redis
func (info *ClientRateLimitInfo) GenerateRateLimitKey() string {
	tier := info.DetermineRateLimitTier()

	switch tier {
	case RateLimitTierFirstParty, RateLimitTierTrusted, RateLimitTierPublic:
		return string(tier) + ":" + info.ClientID
	case RateLimitTierAdmin:
		// Admin limits per user to prevent abuse
		if info.UserID != "" {
			return string(tier) + ":user:" + info.UserID
		}
		return string(tier) + ":" + info.ClientID
	default:
		// Anonymous limits per IP will be handled by the calling code
		return string(tier) + ":ip"
	}
}

// containsScope checks if a scope exists in the scopes array
func containsScope(scopes []string, target string) bool {
	for _, scope := range scopes {
		if scope == target {
			return true
		}
	}
	return false
}

// RateLimitHeaders contains HTTP headers to return to clients
type RateLimitHeaders struct {
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
	Reset     int64  `json:"reset"`
	Tier      string `json:"tier"`
}

// ToHeaders converts the rate limit info to HTTP headers
func (h *RateLimitHeaders) ToHeaders() map[string]string {
	return map[string]string{
		"X-RateLimit-Limit":     fmt.Sprintf("%d", h.Limit),
		"X-RateLimit-Remaining": fmt.Sprintf("%d", h.Remaining),
		"X-RateLimit-Reset":     fmt.Sprintf("%d", h.Reset),
		"X-RateLimit-Tier":      h.Tier,
	}
}
