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
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"` // Never serialize actual token
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
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
	Theme                string   `json:"theme"`                     // "light", "dark", "auto"
	Language             string   `json:"language"`                  // "en", "es", "fr", etc.
	EmailNotifications   bool     `json:"email_notifications"`       // Enable email notifications
	WorksPerPage         int      `json:"works_per_page"`            // Number of works to show per page
	DefaultRating        string   `json:"default_rating"`            // Default rating for new works
	HiddenWarnings       []string `json:"hidden_warnings"`           // Warnings to hide by default
	PreferredCategories  []string `json:"preferred_categories"`      // Categories user is interested in
	PrivateBookmarks     bool     `json:"private_bookmarks"`         // Make bookmarks private by default
	ShowWordCounts       bool     `json:"show_word_counts"`          // Show word counts in listings
	EnableKeyboardShorts bool     `json:"enable_keyboard_shortcuts"` // Enable keyboard navigation

	// Enhanced AO3-style preferences
	ProfileVisibility  string     `json:"profile_visibility"`  // "public", "registered_users", "friends", "private"
	WorkVisibility     string     `json:"work_visibility"`     // "public", "registered_users", "private"
	CommentPermissions string     `json:"comment_permissions"` // "all", "registered_users", "friends", "none"
	SkinTheme          string     `json:"skin_theme"`          // Custom theme/skin
	Timezone           string     `json:"timezone"`            // User's timezone
	BirthDate          *time.Time `json:"birth_date"`          // Optional birth date
}

// ============================================================================
// ENHANCED USER PROFILE MODELS
// ============================================================================

// UserProfile represents an enhanced user profile with all AO3-style features
type UserProfile struct {
	User
	WorksCount         int        `json:"works_count" db:"works_count"`
	SeriesCount        int        `json:"series_count" db:"series_count"`
	BookmarksCount     int        `json:"bookmarks_count" db:"bookmarks_count"`
	CommentsCount      int        `json:"comments_count" db:"comments_count"`
	KudosGivenCount    int        `json:"kudos_given_count" db:"kudos_given_count"`
	KudosReceivedCount int        `json:"kudos_received_count" db:"kudos_received_count"`
	WordsWritten       int        `json:"words_written" db:"words_written"`
	LastWorkDate       *time.Time `json:"last_work_date" db:"last_work_date"`
	Pseudonyms         []string   `json:"pseudonyms" db:"pseudonyms"`
	FriendsCount       int        `json:"friends_count" db:"friends_count"`
}

// UserPseudonym represents a pen name/pseudonym that users can write under
type UserPseudonym struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	IsDefault   bool      `json:"is_default" db:"is_default"`
	Description *string   `json:"description" db:"description"`
	IconURL     *string   `json:"icon_url" db:"icon_url"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// UserPseudonymRequest represents a request to create or update a pseudonym
type UserPseudonymRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=50"`
	IsDefault   bool    `json:"is_default"`
	Description *string `json:"description" validate:"max=500"`
	IconURL     *string `json:"icon_url" validate:"url"`
}

// UserRelationship represents friendships and social connections between users
type UserRelationship struct {
	ID          uuid.UUID `json:"id" db:"id"`
	RequesterID uuid.UUID `json:"requester_id" db:"requester_id"`
	AddresseeID uuid.UUID `json:"addressee_id" db:"addressee_id"`
	Status      string    `json:"status" db:"status"` // 'pending', 'accepted', 'blocked', 'rejected'
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// UserStatistics represents aggregated user activity statistics
type UserStatistics struct {
	UserID             uuid.UUID  `json:"user_id" db:"user_id"`
	WorksCount         int        `json:"works_count" db:"works_count"`
	SeriesCount        int        `json:"series_count" db:"series_count"`
	BookmarksCount     int        `json:"bookmarks_count" db:"bookmarks_count"`
	CommentsCount      int        `json:"comments_count" db:"comments_count"`
	KudosGivenCount    int        `json:"kudos_given_count" db:"kudos_given_count"`
	KudosReceivedCount int        `json:"kudos_received_count" db:"kudos_received_count"`
	WordsWritten       int        `json:"words_written" db:"words_written"`
	LastWorkDate       *time.Time `json:"last_work_date" db:"last_work_date"`
	JoinDate           time.Time  `json:"join_date" db:"join_date"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// UserDashboard represents a user's dashboard view with all relevant information
type UserDashboard struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	Username            string     `json:"username" db:"username"`
	DisplayName         *string    `json:"display_name" db:"display_name"`
	UnreadNotifications int        `json:"unread_notifications" db:"unread_notifications"`
	PublishedWorks      int        `json:"published_works" db:"published_works"`
	DraftWorks          int        `json:"draft_works" db:"draft_works"`
	Bookmarks           int        `json:"bookmarks" db:"bookmarks"`
	Series              int        `json:"series" db:"series"`
	TotalHits           int        `json:"total_hits" db:"total_hits"`
	TotalKudos          int        `json:"total_kudos" db:"total_kudos"`
	TotalComments       int        `json:"total_comments" db:"total_comments"`
	LastPublished       *time.Time `json:"last_published" db:"last_published"`
}

// UserActivityLog represents logged user actions for analytics and security
type UserActivityLog struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	Action     string     `json:"action" db:"action"`
	EntityType *string    `json:"entity_type" db:"entity_type"`
	EntityID   *uuid.UUID `json:"entity_id" db:"entity_id"`
	Metadata   string     `json:"metadata" db:"metadata"` // JSON string
	IPAddress  *string    `json:"ip_address" db:"ip_address"`
	UserAgent  *string    `json:"user_agent" db:"user_agent"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// UserProfileUpdateRequest represents a request to update user profile
type UserProfileUpdateRequest struct {
	DisplayName        *string    `json:"display_name" validate:"max=100"`
	Bio                *string    `json:"bio" validate:"max=2000"`
	Location           *string    `json:"location" validate:"max=100"`
	Website            *string    `json:"website" validate:"url,max=500"`
	BirthDate          *time.Time `json:"birth_date"`
	Timezone           *string    `json:"timezone" validate:"max=50"`
	ProfileVisibility  *string    `json:"profile_visibility" validate:"oneof=public registered_users friends private"`
	WorkVisibility     *string    `json:"work_visibility" validate:"oneof=public registered_users private"`
	CommentPermissions *string    `json:"comment_permissions" validate:"oneof=all registered_users friends none"`
	SkinTheme          *string    `json:"skin_theme" validate:"max=50"`
}

// UserSearchParams represents parameters for searching users
type UserSearchParams struct {
	Query      *string    `form:"q"`
	Username   *string    `form:"username"`
	Location   *string    `form:"location"`
	JoinedFrom *time.Time `form:"joined_from"`
	JoinedTo   *time.Time `form:"joined_to"`
	IsActive   *bool      `form:"is_active"`
	SortBy     string     `form:"sort_by" validate:"oneof=username created_at last_login works_count"`
	SortOrder  string     `form:"sort_order" validate:"oneof=asc desc"`
	Page       int        `form:"page" validate:"min=1"`
	Limit      int        `form:"limit" validate:"min=1,max=100"`
}

// ============================================================================
// VALIDATION AND UTILITY METHODS
// ============================================================================

// Validate checks if the pseudonym request is valid
func (upr *UserPseudonymRequest) Validate() error {
	if upr.Name == "" {
		return ErrInvalidInput
	}
	if len(upr.Name) > 50 {
		return ErrInvalidInput
	}
	return nil
}

// IsValidStatus checks if a relationship status is valid
func (ur *UserRelationship) IsValidStatus() bool {
	validStatuses := []string{"pending", "accepted", "blocked", "rejected"}
	for _, status := range validStatuses {
		if ur.Status == status {
			return true
		}
	}
	return false
}

// IsPending returns true if the relationship is pending approval
func (ur *UserRelationship) IsPending() bool {
	return ur.Status == "pending"
}

// IsAccepted returns true if the relationship is accepted (friends)
func (ur *UserRelationship) IsAccepted() bool {
	return ur.Status == "accepted"
}

// IsBlocked returns true if the relationship is blocked
func (ur *UserRelationship) IsBlocked() bool {
	return ur.Status == "blocked"
}

// CanViewProfile checks if a user can view another user's profile based on visibility settings
func CanViewProfile(viewerID *uuid.UUID, profileOwnerID uuid.UUID, visibility string, areFriends bool) bool {
	// Profile owner can always view their own profile
	if viewerID != nil && *viewerID == profileOwnerID {
		return true
	}

	switch visibility {
	case "public":
		return true
	case "registered_users":
		return viewerID != nil
	case "friends":
		return viewerID != nil && (*viewerID == profileOwnerID || areFriends)
	case "private":
		return viewerID != nil && *viewerID == profileOwnerID
	default:
		return false
	}
}

// CanComment checks if a user can comment based on comment permissions
func CanComment(commenterID *uuid.UUID, workOwnerID uuid.UUID, permissions string, areFriends bool) bool {
	switch permissions {
	case "all":
		return true
	case "registered_users":
		return commenterID != nil
	case "friends":
		return commenterID != nil && (*commenterID == workOwnerID || areFriends)
	case "none":
		return commenterID != nil && *commenterID == workOwnerID // Only work owner can comment
	default:
		return false
	}
}

// ============================================================================
// USER RELATIONSHIPS AND BLOCKING
// ============================================================================

// UserBlock represents a user blocking another user
type UserBlock struct {
	ID        uuid.UUID `json:"id" db:"id"`
	BlockerID uuid.UUID `json:"blocker_id" db:"blocker_id"`
	BlockedID uuid.UUID `json:"blocked_id" db:"blocked_id"`
	BlockType string    `json:"block_type" db:"block_type" validate:"oneof=full comments works"`
	Reason    string    `json:"reason" db:"reason"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
