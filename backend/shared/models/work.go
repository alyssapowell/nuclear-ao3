package models

import (
	"time"

	"github.com/google/uuid"
)

// Work represents a fanfiction work
type Work struct {
	ID                     uuid.UUID  `json:"id" db:"id"`
	LegacyID               *int       `json:"legacy_id,omitempty" db:"legacy_id"` // Original AO3 numeric ID for migration
	Title                  string     `json:"title" db:"title" validate:"required,min=1,max=200"`
	Summary                string     `json:"summary" db:"summary"`
	Notes                  string     `json:"notes" db:"notes"`
	UserID                 uuid.UUID  `json:"user_id" db:"user_id"`
	Username               string     `json:"username"` // Loaded from join
	SeriesID               *uuid.UUID `json:"series_id" db:"series_id"`
	Language               string     `json:"language" db:"language" validate:"required,len=2"`
	Rating                 string     `json:"rating" db:"rating" validate:"required,oneof=general teen mature explicit"`
	Category               []string   `json:"category" db:"category"`           // JSON array
	Warnings               []string   `json:"warnings" db:"warnings"`           // JSON array
	Fandoms                []string   `json:"fandoms" db:"fandoms"`             // JSON array
	Characters             []string   `json:"characters" db:"characters"`       // JSON array
	Relationships          []string   `json:"relationships" db:"relationships"` // JSON array
	FreeformTags           []string   `json:"freeform_tags" db:"freeform_tags"` // JSON array
	WordCount              int        `json:"word_count" db:"word_count"`
	ChapterCount           int        `json:"chapter_count" db:"chapter_count"`
	MaxChapters            *int       `json:"max_chapters" db:"max_chapters"` // nil if unknown
	IsComplete             bool       `json:"is_complete" db:"is_complete"`
	Status                 string     `json:"status" db:"status" validate:"oneof=draft posted hidden"`
	RestrictedToUsers      bool       `json:"restricted_to_users" db:"restricted_to_users"`
	RestrictedToAdults     bool       `json:"restricted_to_adults" db:"restricted_to_adults"`
	CommentPolicy          string     `json:"comment_policy" db:"comment_policy" validate:"oneof=open users_only disabled"`
	ModerateComments       bool       `json:"moderate_comments" db:"moderate_comments"`
	DisableComments        bool       `json:"disable_comments" db:"disable_comments"`
	InAnonCollection       bool       `json:"in_anon_collection" db:"in_anon_collection"`
	InUnrevealedCollection bool       `json:"in_unrevealed_collection" db:"in_unrevealed_collection"`
	IsAnonymous            bool       `json:"is_anonymous" db:"is_anonymous"`
	PublishedAt            *time.Time `json:"published_at" db:"published_at"`
	UpdatedAt              time.Time  `json:"updated_at" db:"updated_at"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
	// Statistics (loaded separately)
	Hits        int `json:"hits"`
	Kudos       int `json:"kudos"`
	Comments    int `json:"comments"`
	Bookmarks   int `json:"bookmarks"`
	Collections int `json:"collections"`
}

// Chapter represents a chapter within a work
type Chapter struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	WorkID      uuid.UUID  `json:"work_id" db:"work_id"`
	Number      int        `json:"number" db:"number" validate:"min=1"`
	Title       string     `json:"title" db:"title"`
	Summary     string     `json:"summary" db:"summary"`
	Notes       string     `json:"notes" db:"notes"`
	EndNotes    string     `json:"end_notes" db:"end_notes"`
	Content     string     `json:"content" db:"content" validate:"required"`
	WordCount   int        `json:"word_count" db:"word_count"`
	Status      string     `json:"status" db:"status" validate:"oneof=draft posted"`
	PublishedAt *time.Time `json:"published_at" db:"published_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// Series represents a collection of related works
type Series struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Title      string    `json:"title" db:"title" validate:"required,min=1,max=200"`
	Summary    string    `json:"summary" db:"summary"`
	Notes      string    `json:"notes" db:"notes"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	Username   string    `json:"username"` // Loaded from join
	IsComplete bool      `json:"is_complete" db:"is_complete"`
	WorkCount  int       `json:"work_count" db:"work_count"`
	WordCount  int       `json:"word_count"` // Calculated from works
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// WorkStatistics tracks engagement metrics for a work
type WorkStatistics struct {
	WorkID      uuid.UUID `json:"work_id" db:"work_id"`
	Hits        int       `json:"hits" db:"hits"`
	Kudos       int       `json:"kudos" db:"kudos"`
	Comments    int       `json:"comments" db:"comments"`
	Bookmarks   int       `json:"bookmarks" db:"bookmarks"`
	Collections int       `json:"collections" db:"collections"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// WorkKudos tracks who gave kudos to which work
type WorkKudos struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	WorkID    uuid.UUID  `json:"work_id" db:"work_id"`
	UserID    *uuid.UUID `json:"user_id" db:"user_id"` // Nil for anonymous kudos
	IPAddress string     `json:"ip_address" db:"ip_address"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// WorkComment represents a comment on a work or chapter
type WorkComment struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	WorkID           uuid.UUID  `json:"work_id" db:"work_id"`
	ChapterID        *uuid.UUID `json:"chapter_id" db:"chapter_id"`       // Nil for work-level comments
	UserID           *uuid.UUID `json:"user_id" db:"user_id"`             // Nil for anonymous comments
	Username         string     `json:"username"`                         // Display name for comment
	ParentID         *uuid.UUID `json:"parent_id" db:"parent_comment_id"` // For threaded comments
	Content          string     `json:"content" db:"content" validate:"required,max=10000"`
	Status           string     `json:"status" db:"status" validate:"oneof=published pending deleted spam hidden"`
	ModerationReason string     `json:"moderation_reason" db:"moderation_reason"`
	ModeratedBy      *uuid.UUID `json:"moderated_by" db:"moderated_by"`
	ModeratedAt      *time.Time `json:"moderated_at" db:"moderated_at"`
	IsAnonymous      bool       `json:"is_anonymous" db:"is_anonymous"`
	IPAddress        string     `json:"ip_address" db:"ip_address"`
	IsDeleted        bool       `json:"is_deleted" db:"is_deleted"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// UserMute represents a user muting another user (matching AO3's implementation)
type UserMute struct {
	ID        uuid.UUID `json:"id" db:"id"`
	MuterID   uuid.UUID `json:"muter_id" db:"muter_id"`
	MutedID   uuid.UUID `json:"muted_id" db:"muted_id"`
	Reason    string    `json:"reason" db:"reason"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Pseud represents a user pseudonym (matching AO3's implementation)
type Pseud struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name" validate:"required,min=1,max=40"`
	Description string    `json:"description" db:"description"`
	IsDefault   bool      `json:"is_default" db:"is_default"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Creatorship represents the relationship between works and their creators
type Creatorship struct {
	ID           uuid.UUID `json:"id" db:"id"`
	CreationID   uuid.UUID `json:"creation_id" db:"creation_id"`
	CreationType string    `json:"creation_type" db:"creation_type" validate:"oneof=Work Series Chapter"`
	PseudID      uuid.UUID `json:"pseud_id" db:"pseud_id"`
	Approved     bool      `json:"approved" db:"approved"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Gift represents a work gift (matching AO3's implementation)
type Gift struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	WorkID        uuid.UUID  `json:"work_id" db:"work_id"`
	PseudID       *uuid.UUID `json:"pseud_id" db:"pseud_id"`
	RecipientName string     `json:"recipient_name" db:"recipient_name"`
	Rejected      bool       `json:"rejected" db:"rejected"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// WorkAuthor represents an author of a work (for API responses)
type WorkAuthor struct {
	PseudID     *uuid.UUID `json:"pseud_id"`
	PseudName   string     `json:"pseud_name"`
	UserID      *uuid.UUID `json:"user_id"`
	Username    string     `json:"username"`
	IsAnonymous bool       `json:"is_anonymous"`
}

// Bookmark represents a user's bookmark of a work
type Bookmark struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	WorkID    uuid.UUID `json:"work_id" db:"work_id"`
	IsPrivate bool      `json:"is_private" db:"is_private"`
	Notes     string    `json:"notes" db:"notes"`
	Tags      []string  `json:"tags" db:"tags"` // User's own tags for the bookmark
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Collection represents a themed collection of works
type Collection struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name" validate:"required,min=1,max=100"`
	Title       string    `json:"title" db:"title" validate:"required,min=1,max=200"`
	Description string    `json:"description" db:"description"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"` // Collection maintainer
	IsOpen      bool      `json:"is_open" db:"is_open"` // Can anyone add works?
	IsModerated bool      `json:"is_moderated" db:"is_moderated"`
	IsAnonymous bool      `json:"is_anonymous" db:"is_anonymous"`
	WorkCount   int       `json:"work_count" db:"work_count"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CollectionItem represents a work in a collection
type CollectionItem struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	CollectionID uuid.UUID  `json:"collection_id" db:"collection_id"`
	WorkID       uuid.UUID  `json:"work_id" db:"work_id"`
	AddedBy      uuid.UUID  `json:"added_by" db:"added_by"`
	IsApproved   bool       `json:"is_approved" db:"is_approved"`
	AddedAt      time.Time  `json:"added_at" db:"added_at"`
	ApprovedAt   *time.Time `json:"approved_at" db:"approved_at"`
}

// CreateWorkRequest represents the request to create a new work
type CreateWorkRequest struct {
	Title         string     `json:"title" validate:"required,min=1,max=200"`
	Summary       string     `json:"summary"`
	Notes         string     `json:"notes"`
	SeriesID      *uuid.UUID `json:"series_id"`
	Language      string     `json:"language" validate:"required,len=2"`
	Rating        string     `json:"rating" validate:"required,oneof=general teen mature explicit"`
	Category      []string   `json:"category"`
	Warnings      []string   `json:"warnings"`
	Fandoms       []string   `json:"fandoms" validate:"required,min=1"`
	Characters    []string   `json:"characters"`
	Relationships []string   `json:"relationships"`
	FreeformTags  []string   `json:"freeform_tags"`
	MaxChapters   *int       `json:"max_chapters"`
	// First chapter data
	ChapterTitle    string `json:"chapter_title"`
	ChapterSummary  string `json:"chapter_summary"`
	ChapterNotes    string `json:"chapter_notes"`
	ChapterEndNotes string `json:"chapter_end_notes"`
	ChapterContent  string `json:"chapter_content" validate:"required"`
}

// UpdateWorkRequest represents the request to update work metadata
type UpdateWorkRequest struct {
	Title                  *string    `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Summary                *string    `json:"summary,omitempty"`
	Notes                  *string    `json:"notes,omitempty"`
	SeriesID               *uuid.UUID `json:"series_id,omitempty"`
	Rating                 *string    `json:"rating,omitempty" validate:"omitempty,oneof=general teen mature explicit"`
	Category               []string   `json:"category,omitempty"`
	Warnings               []string   `json:"warnings,omitempty"`
	Fandoms                []string   `json:"fandoms,omitempty"`
	Characters             []string   `json:"characters,omitempty"`
	Relationships          []string   `json:"relationships,omitempty"`
	FreeformTags           []string   `json:"freeform_tags,omitempty"`
	MaxChapters            *int       `json:"max_chapters,omitempty"`
	IsComplete             *bool      `json:"is_complete,omitempty"`
	Status                 *string    `json:"status,omitempty" validate:"omitempty,oneof=draft posted hidden"`
	RestrictedToUsers      *bool      `json:"restricted_to_users,omitempty"`
	RestrictedToAdults     *bool      `json:"restricted_to_adults,omitempty"`
	CommentPolicy          *string    `json:"comment_policy,omitempty" validate:"omitempty,oneof=open users_only disabled"`
	ModerateComments       *bool      `json:"moderate_comments,omitempty"`
	DisableComments        *bool      `json:"disable_comments,omitempty"`
	IsAnonymous            *bool      `json:"is_anonymous,omitempty"`
	InAnonCollection       *bool      `json:"in_anon_collection,omitempty"`
	InUnrevealedCollection *bool      `json:"in_unrevealed_collection,omitempty"`
}

// WorkReport represents a report on inappropriate work content
type WorkReport struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	WorkID      uuid.UUID  `json:"work_id" db:"work_id"`
	ReporterID  *uuid.UUID `json:"reporter_id" db:"reporter_id"`
	ReporterIP  string     `json:"reporter_ip" db:"reporter_ip"`
	Reason      string     `json:"reason" db:"reason" validate:"oneof=copyright plagiarism harassment inappropriate_content wrong_rating missing_warnings spam other"`
	Description string     `json:"description" db:"description"`
	Status      string     `json:"status" db:"status" validate:"oneof=pending resolved dismissed escalated"`
	ReviewedBy  *uuid.UUID `json:"reviewed_by" db:"reviewed_by"`
	ReviewedAt  *time.Time `json:"reviewed_at" db:"reviewed_at"`
	Resolution  string     `json:"resolution" db:"resolution"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// UserPrivacySettings represents a user's default privacy preferences
type UserPrivacySettings struct {
	ID                      uuid.UUID `json:"id" db:"id"`
	UserID                  uuid.UUID `json:"user_id" db:"user_id"`
	DefaultWorkVisibility   string    `json:"default_work_visibility" db:"default_work_visibility" validate:"oneof=public users_only private"`
	DefaultCommentPolicy    string    `json:"default_comment_policy" db:"default_comment_policy" validate:"oneof=open users_only disabled"`
	DefaultModerateComments bool      `json:"default_moderate_comments" db:"default_moderate_comments"`
	HideEmail               bool      `json:"hide_email" db:"hide_email"`
	HideProfile             bool      `json:"hide_profile" db:"hide_profile"`
	AllowGuestComments      bool      `json:"allow_guest_comments" db:"allow_guest_comments"`
	AllowAnonymousKudos     bool      `json:"allow_anonymous_kudos" db:"allow_anonymous_kudos"`
	NotifyComments          bool      `json:"notify_comments" db:"notify_comments"`
	NotifyKudos             bool      `json:"notify_kudos" db:"notify_kudos"`
	NotifyBookmarks         bool      `json:"notify_bookmarks" db:"notify_bookmarks"`
	NotifyFollows           bool      `json:"notify_follows" db:"notify_follows"`
	CreatedAt               time.Time `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time `json:"updated_at" db:"updated_at"`
}

// CreateChapterRequest represents the request to create a new chapter
type CreateChapterRequest struct {
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Notes    string `json:"notes"`
	EndNotes string `json:"end_notes"`
	Content  string `json:"content" validate:"required"`
	Status   string `json:"status" validate:"oneof=draft posted"`
}

// UpdateChapterRequest represents the request to update an existing chapter
type UpdateChapterRequest struct {
	Title    *string `json:"title,omitempty"`
	Summary  *string `json:"summary,omitempty"`
	Notes    *string `json:"notes,omitempty"`
	EndNotes *string `json:"end_notes,omitempty"`
	Content  *string `json:"content,omitempty"`
	Status   *string `json:"status,omitempty" validate:"omitempty,oneof=draft posted"`
}
