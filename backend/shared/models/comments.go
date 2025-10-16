package models

import (
	"time"

	"github.com/google/uuid"
)

// Comment represents a comment on a work or chapter with full threading support
type Comment struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	WorkID          *uuid.UUID `json:"work_id" db:"work_id"`
	ChapterID       *uuid.UUID `json:"chapter_id" db:"chapter_id"`
	UserID          *uuid.UUID `json:"user_id" db:"user_id"`
	PseudonymID     *uuid.UUID `json:"pseudonym_id" db:"pseudonym_id"`
	ParentCommentID *uuid.UUID `json:"parent_comment_id" db:"parent_comment_id"`
	Content         string     `json:"content" db:"content"`
	GuestName       *string    `json:"guest_name" db:"guest_name"`
	GuestEmail      *string    `json:"guest_email" db:"guest_email"`
	IPAddress       *string    `json:"ip_address" db:"ip_address"`
	IsDeleted       bool       `json:"is_deleted" db:"is_deleted"`
	IsModerated     bool       `json:"is_moderated" db:"is_moderated"`
	IsSpam          bool       `json:"is_spam" db:"is_spam"`
	ThreadLevel     int        `json:"thread_level" db:"thread_level"`
	KudosCount      int        `json:"kudos_count" db:"kudos_count"`
	ReplyCount      int        `json:"reply_count" db:"reply_count"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	EditedAt        *time.Time `json:"edited_at" db:"edited_at"`
}

// CommentWithDetails includes author information and work context
type CommentWithDetails struct {
	Comment
	AuthorName        string               `json:"author_name" db:"author_name"`
	AuthorUserID      *uuid.UUID           `json:"author_user_id" db:"author_user_id"`
	AuthorPseudonymID *uuid.UUID           `json:"author_pseudonym_id" db:"author_pseudonym_id"`
	AuthorType        string               `json:"author_type" db:"author_type"` // 'user', 'guest', 'unknown'
	WorkTitle         *string              `json:"work_title" db:"work_title"`
	WorkAuthorID      *uuid.UUID           `json:"work_author_id" db:"work_author_id"`
	ParentContent     *string              `json:"parent_content" db:"parent_content"`
	ParentAuthorName  *string              `json:"parent_author_name" db:"parent_author_name"`
	Replies           []CommentWithDetails `json:"replies,omitempty"` // For nested display
}

// CommentCreateRequest represents the data needed to create a new comment
type CommentCreateRequest struct {
	WorkID          *uuid.UUID `json:"work_id"`
	ChapterID       *uuid.UUID `json:"chapter_id"`
	ParentCommentID *uuid.UUID `json:"parent_comment_id"`
	Content         string     `json:"content" validate:"required,min=1,max=10000"`
	PseudonymID     *uuid.UUID `json:"pseudonym_id"` // For registered users
	GuestName       *string    `json:"guest_name"`   // For guest comments
	GuestEmail      *string    `json:"guest_email"`  // For guest comments
}

// CommentUpdateRequest represents data for updating an existing comment
type CommentUpdateRequest struct {
	Content string `json:"content" validate:"required,min=1,max=10000"`
}

// CommentKudos represents a kudos/like on a comment
type CommentKudos struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	CommentID    uuid.UUID  `json:"comment_id" db:"comment_id"`
	UserID       *uuid.UUID `json:"user_id" db:"user_id"`
	PseudonymID  *uuid.UUID `json:"pseudonym_id" db:"pseudonym_id"`
	GuestSession *string    `json:"guest_session" db:"guest_session"`
	IPAddress    *string    `json:"ip_address" db:"ip_address"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// CommentReport represents a report on an inappropriate comment
type CommentReport struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	CommentID   uuid.UUID  `json:"comment_id" db:"comment_id"`
	ReporterID  *uuid.UUID `json:"reporter_id" db:"reporter_id"`
	ReporterIP  string     `json:"reporter_ip" db:"reporter_ip"`
	Reason      string     `json:"reason" db:"reason" validate:"oneof=spam harassment off_topic inappropriate hate_speech doxxing other"`
	Description string     `json:"description" db:"description"`
	Status      string     `json:"status" db:"status" validate:"oneof=pending resolved dismissed escalated"`
	ReviewedBy  *uuid.UUID `json:"reviewed_by" db:"reviewed_by"`
	ReviewedAt  *time.Time `json:"reviewed_at" db:"reviewed_at"`
	Resolution  string     `json:"resolution" db:"resolution"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// CommentReportRequest represents a request to report a comment
type CommentReportRequest struct {
	Reason      string  `json:"reason" validate:"required,oneof=spam harassment off_topic inappropriate copyright other"`
	Description *string `json:"description" validate:"max=1000"`
}

// CommentThread represents a threaded comment structure for display
type CommentThread struct {
	Comment  CommentWithDetails `json:"comment"`
	Replies  []CommentThread    `json:"replies"`
	CanReply bool               `json:"can_reply"`
	CanEdit  bool               `json:"can_edit"`
	CanFlag  bool               `json:"can_flag"`
}

// CommentSearchParams represents parameters for searching/filtering comments
type CommentSearchParams struct {
	WorkID      *uuid.UUID `form:"work_id"`
	ChapterID   *uuid.UUID `form:"chapter_id"`
	UserID      *uuid.UUID `form:"user_id"`
	AuthorName  *string    `form:"author_name"`
	Content     *string    `form:"content"`
	DateFrom    *time.Time `form:"date_from"`
	DateTo      *time.Time `form:"date_to"`
	IsModerated *bool      `form:"is_moderated"`
	IsSpam      *bool      `form:"is_spam"`
	ThreadLevel *int       `form:"thread_level"`
	SortBy      string     `form:"sort_by" validate:"oneof=created_at updated_at kudos_count reply_count"`
	SortOrder   string     `form:"sort_order" validate:"oneof=asc desc"`
	Page        int        `form:"page" validate:"min=1"`
	Limit       int        `form:"limit" validate:"min=1,max=100"`
}

// CommentStats represents statistics about comments
type CommentStats struct {
	TotalComments    int                   `json:"total_comments" db:"total_comments"`
	TodayComments    int                   `json:"today_comments" db:"today_comments"`
	WeekComments     int                   `json:"week_comments" db:"week_comments"`
	MonthComments    int                   `json:"month_comments" db:"month_comments"`
	AveragePerWork   float64               `json:"average_per_work" db:"average_per_work"`
	MostActiveUsers  []UserCommentActivity `json:"most_active_users"`
	ThreadDepthStats ThreadDepthStats      `json:"thread_depth_stats"`
}

// UserCommentActivity represents user activity in comments
type UserCommentActivity struct {
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	Username      string    `json:"username" db:"username"`
	DisplayName   *string   `json:"display_name" db:"display_name"`
	CommentCount  int       `json:"comment_count" db:"comment_count"`
	KudosReceived int       `json:"kudos_received" db:"kudos_received"`
}

// ThreadDepthStats represents statistics about comment threading
type ThreadDepthStats struct {
	MaxDepth        int     `json:"max_depth" db:"max_depth"`
	AverageDepth    float64 `json:"average_depth" db:"average_depth"`
	ThreadedPercent float64 `json:"threaded_percent" db:"threaded_percent"`
}

// CommentPermissions represents what a user can do with comments
type CommentPermissions struct {
	CanComment    bool `json:"can_comment"`
	CanReply      bool `json:"can_reply"`
	CanEdit       bool `json:"can_edit"`
	CanDelete     bool `json:"can_delete"`
	CanModerate   bool `json:"can_moderate"`
	CanReport     bool `json:"can_report"`
	CanViewHidden bool `json:"can_view_hidden"`
}

// Validation methods

// Validate checks if the comment create request is valid
func (ccr *CommentCreateRequest) Validate() error {
	// Must have either work_id or chapter_id
	if ccr.WorkID == nil && ccr.ChapterID == nil {
		return ErrInvalidInput
	}

	// Must have content
	if ccr.Content == "" {
		return ErrInvalidInput
	}

	// For authenticated users, must have pseudonym_id
	// For guest users, must have guest_name
	if ccr.PseudonymID == nil && ccr.GuestName == nil {
		return ErrInvalidInput
	}

	return nil
}

// IsGuest returns true if this is a guest comment
func (c *Comment) IsGuest() bool {
	return c.GuestName != nil && c.UserID == nil
}

// IsAuthenticated returns true if this is from a logged-in user
func (c *Comment) IsAuthenticated() bool {
	return c.UserID != nil && c.PseudonymID != nil
}

// CanBeEditedBy checks if the given user can edit this comment
func (c *Comment) CanBeEditedBy(userID uuid.UUID) bool {
	if c.UserID == nil {
		return false // Guest comments can't be edited
	}
	return *c.UserID == userID
}

// IsThreaded returns true if this comment is part of a thread
func (c *Comment) IsThreaded() bool {
	return c.ParentCommentID != nil || c.ThreadLevel > 0
}

// GetDisplayName returns the appropriate display name for the comment author
func (cwd *CommentWithDetails) GetDisplayName() string {
	if cwd.AuthorName != "" {
		return cwd.AuthorName
	}
	if cwd.GuestName != nil {
		return *cwd.GuestName
	}
	return "Anonymous"
}

// BuildCommentTree converts a flat list of comments into a threaded tree structure
func BuildCommentTree(comments []CommentWithDetails, userID *uuid.UUID) []CommentThread {
	// Create a map for quick lookup
	commentMap := make(map[uuid.UUID]*CommentWithDetails)
	for i := range comments {
		commentMap[comments[i].ID] = &comments[i]
	}

	// Build the tree
	var rootComments []CommentThread

	for _, comment := range comments {
		thread := CommentThread{
			Comment:  comment,
			Replies:  []CommentThread{},
			CanReply: userID != nil, // Simple permission check
			CanEdit:  userID != nil && comment.UserID != nil && *comment.UserID == *userID,
			CanFlag:  userID != nil && (comment.UserID == nil || *comment.UserID != *userID),
		}

		if comment.ParentCommentID == nil {
			// This is a root comment
			thread.Replies = buildReplies(comment.ID, commentMap, userID)
			rootComments = append(rootComments, thread)
		}
	}

	return rootComments
}

// buildReplies recursively builds the reply tree for a comment
func buildReplies(parentID uuid.UUID, commentMap map[uuid.UUID]*CommentWithDetails, userID *uuid.UUID) []CommentThread {
	var replies []CommentThread

	for _, comment := range commentMap {
		if comment.ParentCommentID != nil && *comment.ParentCommentID == parentID {
			thread := CommentThread{
				Comment:  *comment,
				Replies:  buildReplies(comment.ID, commentMap, userID),
				CanReply: userID != nil,
				CanEdit:  userID != nil && comment.UserID != nil && *comment.UserID == *userID,
				CanFlag:  userID != nil && (comment.UserID == nil || *comment.UserID != *userID),
			}
			replies = append(replies, thread)
		}
	}

	return replies
}
