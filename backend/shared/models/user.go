package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a registered user in the system
type User struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	Username     string                 `json:"username" db:"username" validate:"required,min=3,max=50"`
	Email        string                 `json:"email" db:"email" validate:"required,email"`
	PasswordHash string                 `json:"-" db:"password_hash"` // Never serialize password
	DisplayName  string                 `json:"display_name" db:"display_name"`
	Bio          string                 `json:"bio" db:"bio"`
	Location     string                 `json:"location" db:"location"`
	Website      string                 `json:"website" db:"website" validate:"omitempty,url"`
	Preferences  map[string]interface{} `json:"preferences" db:"preferences"`
	IsActive     bool                   `json:"is_active" db:"is_active"`
	IsVerified   bool                   `json:"is_verified" db:"is_verified"`
	LastLoginAt  *time.Time             `json:"last_login_at" db:"last_login_at"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
	Roles        []string               `json:"roles"` // Loaded separately
}

// PublicUser returns a user struct safe for public consumption
func (u *User) PublicUser() *User {
	return &User{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Bio:         u.Bio,
		Location:    u.Location,
		Website:     u.Website,
		IsVerified:  u.IsVerified,
		CreatedAt:   u.CreatedAt,
	}
}

// UserRole represents a role assigned to a user
type UserRole struct {
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Role      string    `json:"role" db:"role"`
	GrantedAt time.Time `json:"granted_at" db:"granted_at"`
	GrantedBy uuid.UUID `json:"granted_by" db:"granted_by"`
}

// RefreshToken represents a JWT refresh token
type RefreshToken struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	TokenHash string    `json:"-" db:"token_hash"` // Never serialize actual token
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// UserSession tracks user login sessions for security
type UserSession struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	IPAddress string    `json:"ip_address" db:"ip_address"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	Location  string    `json:"location" db:"location"` // Geo-location
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	LastSeen  time.Time `json:"last_seen" db:"last_seen"`
	IsActive  bool      `json:"is_active" db:"is_active"`
}

// PasswordResetToken represents a password reset request
type PasswordResetToken struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// EmailVerificationToken for new account verification
type EmailVerificationToken struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// UserPreferences defines the structure of user preferences
type UserPreferences struct {
	Theme                string   `json:"theme"`                 // "light", "dark", "auto"
	Language             string   `json:"language"`              // "en", "es", "fr", etc.
	EmailNotifications   bool     `json:"email_notifications"`   // Enable email notifications
	WorksPerPage         int      `json:"works_per_page"`        // Number of works to show per page
	DefaultRating        string   `json:"default_rating"`        // Default rating for new works
	HiddenWarnings       []string `json:"hidden_warnings"`       // Warnings to hide by default
	PreferredCategories  []string `json:"preferred_categories"`  // Categories user is interested in
	PrivateBookmarks     bool     `json:"private_bookmarks"`     // Make bookmarks private by default
	ShowWordCounts       bool     `json:"show_word_counts"`      // Show word counts in listings
	EnableKeyboardShorts bool     `json:"enable_keyboard_shortcuts"` // Enable keyboard navigation
}