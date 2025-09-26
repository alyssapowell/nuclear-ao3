package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthClaims represents JWT token claims
type AuthClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Roles    []string  `json:"roles"`
	jwt.RegisteredClaims
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Remember bool   `json:"remember"` // For longer-lived refresh tokens
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username        string `json:"username" validate:"required,min=3,max=50"`
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
	DisplayName     string `json:"display_name" validate:"max=100"`
	AcceptTOS       bool   `json:"accept_tos" validate:"required"`
}

// AuthResponse represents successful authentication response
type AuthResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	TokenType    string `json:"token_type"`
}

// RefreshTokenRequest for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ChangePasswordRequest for password changes
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

// ResetPasswordRequest for password reset initiation
type ResetPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordConfirmRequest for password reset completion
type ResetPasswordConfirmRequest struct {
	Token           string `json:"token" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

// VerifyEmailRequest for email verification
type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

// UpdateProfileRequest for profile updates
type UpdateProfileRequest struct {
	DisplayName string                 `json:"display_name" validate:"max=100"`
	Bio         string                 `json:"bio" validate:"max=1000"`
	Location    string                 `json:"location" validate:"max=100"`
	Website     string                 `json:"website" validate:"omitempty,url"`
	Preferences map[string]interface{} `json:"preferences"`
}

// Permission represents a user permission
type Permission struct {
	Resource string `json:"resource"` // "work", "comment", "bookmark", "tag"
	Action   string `json:"action"`   // "create", "read", "update", "delete"
	Scope    string `json:"scope"`    // "own", "any", "published"
}

// RolePermissions maps roles to their permissions
var RolePermissions = map[string][]Permission{
	"user": {
		{Resource: "work", Action: "create", Scope: "own"},
		{Resource: "work", Action: "read", Scope: "published"},
		{Resource: "work", Action: "update", Scope: "own"},
		{Resource: "work", Action: "delete", Scope: "own"},
		{Resource: "comment", Action: "create", Scope: "any"},
		{Resource: "comment", Action: "update", Scope: "own"},
		{Resource: "comment", Action: "delete", Scope: "own"},
		{Resource: "bookmark", Action: "create", Scope: "own"},
		{Resource: "bookmark", Action: "update", Scope: "own"},
		{Resource: "bookmark", Action: "delete", Scope: "own"},
	},
	"tag_wrangler": {
		// Includes all user permissions plus:
		{Resource: "tag", Action: "create", Scope: "any"},
		{Resource: "tag", Action: "update", Scope: "any"},
		{Resource: "tag", Action: "delete", Scope: "any"},
		{Resource: "work", Action: "update", Scope: "any"}, // For tag corrections
	},
	"moderator": {
		// Includes all user permissions plus:
		{Resource: "work", Action: "read", Scope: "any"}, // Including drafts
		{Resource: "work", Action: "update", Scope: "any"},
		{Resource: "work", Action: "delete", Scope: "any"},
		{Resource: "comment", Action: "read", Scope: "any"},
		{Resource: "comment", Action: "update", Scope: "any"},
		{Resource: "comment", Action: "delete", Scope: "any"},
		{Resource: "user", Action: "read", Scope: "any"},
		{Resource: "user", Action: "update", Scope: "any"}, // For suspensions
	},
	"admin": {
		// Full permissions
		{Resource: "*", Action: "*", Scope: "*"},
	},
}

// HasPermission checks if a user with given roles has a specific permission
func HasPermission(userRoles []string, resource, action, scope string) bool {
	for _, role := range userRoles {
		permissions, exists := RolePermissions[role]
		if !exists {
			continue
		}
		
		for _, perm := range permissions {
			// Check for wildcard permissions (admin)
			if perm.Resource == "*" && perm.Action == "*" && perm.Scope == "*" {
				return true
			}
			
			// Check specific permissions
			if (perm.Resource == resource || perm.Resource == "*") &&
				(perm.Action == action || perm.Action == "*") &&
				(perm.Scope == scope || perm.Scope == "*") {
				return true
			}
		}
	}
	return false
}

// SecurityEvent for tracking suspicious activities
type SecurityEvent struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	UserID      *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	EventType   string                 `json:"event_type" db:"event_type"` // "failed_login", "suspicious_location", etc.
	Severity    string                 `json:"severity" db:"severity"`     // "low", "medium", "high", "critical"
	IPAddress   string                 `json:"ip_address" db:"ip_address"`
	UserAgent   string                 `json:"user_agent" db:"user_agent"`
	Location    string                 `json:"location" db:"location"`
	Description string                 `json:"description" db:"description"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}