package works

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/html"
	"nuclear-ao3/shared/models"
	"nuclear-ao3/shared/notifications"
)

// Comment handlers for the work service

// GetWorkComments retrieves comments for a specific work with threading
func (ws *WorkService) GetWorkComments(c *gin.Context) {
	workID := c.Param("id")
	if workID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Work ID is required"})
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "asc")

	// Validate sort parameters
	validSortFields := map[string]bool{
		"created_at":   true,
		"updated_at":   true,
		"kudos_count":  true,
		"reply_count":  true,
		"thread_level": true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "asc"
	}

	offset := (page - 1) * limit

	// Query to get comments with author details
	query := `
		SELECT 
			c.id, c.work_id, c.chapter_id, c.user_id, c.pseudonym_id, c.parent_comment_id,
			c.content, c.guest_name, c.guest_email, c.is_deleted, c.is_moderated, c.is_spam,
			c.thread_level, c.kudos_count, c.reply_count, c.created_at, c.updated_at, c.edited_at,
			COALESCE(up.name, u.username, c.guest_name) as author_name,
			u.id as author_user_id,
			up.id as author_pseudonym_id,
			CASE 
				WHEN c.guest_name IS NOT NULL THEN 'guest'
				WHEN u.id IS NOT NULL THEN 'user'
				ELSE 'unknown'
			END as author_type,
			w.title as work_title,
			w.user_id as work_author_id
		FROM comments c
		LEFT JOIN users u ON c.user_id = u.id
		LEFT JOIN user_pseudonyms up ON c.pseudonym_id = up.id
		LEFT JOIN works w ON c.work_id = w.id
		WHERE c.work_id = $1 AND c.is_deleted = false
		ORDER BY ` + sortBy + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3
	`

	rows, err := ws.db.Query(query, workID, limit, offset)
	if err != nil {
		// Log error and return
		// s.logger.Error("Failed to get work comments", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve comments"})
		return
	}
	defer rows.Close()

	var comments []models.CommentWithDetails

	for rows.Next() {
		var comment models.CommentWithDetails
		var pseudonymID, userID, parentCommentID sql.NullString
		var chapterID sql.NullString
		var editedAt sql.NullTime

		err := rows.Scan(
			&comment.ID, &comment.WorkID, &chapterID, &userID, &pseudonymID, &parentCommentID,
			&comment.Content, &comment.GuestName, &comment.GuestEmail, &comment.IsDeleted,
			&comment.IsModerated, &comment.IsSpam, &comment.ThreadLevel, &comment.KudosCount,
			&comment.ReplyCount, &comment.CreatedAt, &comment.UpdatedAt, &editedAt,
			&comment.AuthorName, &comment.AuthorUserID, &comment.AuthorPseudonymID,
			&comment.AuthorType, &comment.WorkTitle, &comment.WorkAuthorID,
		)
		if err != nil {
			// Log error and continue
			continue
		}

		// Handle nullable fields
		if chapterID.Valid {
			chapterUUID, _ := uuid.Parse(chapterID.String)
			comment.ChapterID = &chapterUUID
		}
		if userID.Valid {
			userUUID, _ := uuid.Parse(userID.String)
			comment.UserID = &userUUID
		}
		if pseudonymID.Valid {
			pseudUUID, _ := uuid.Parse(pseudonymID.String)
			comment.PseudonymID = &pseudUUID
		}
		if parentCommentID.Valid {
			parentUUID, _ := uuid.Parse(parentCommentID.String)
			comment.ParentCommentID = &parentUUID
		}
		if editedAt.Valid {
			comment.EditedAt = &editedAt.Time
		}

		comments = append(comments, comment)
	}

	// Get total count for pagination
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM comments WHERE work_id = $1 AND is_deleted = false`
	err = ws.db.QueryRow(countQuery, workID).Scan(&totalCount)
	if err != nil {
		// Log error and use comments length as fallback
		totalCount = len(comments)
	}

	// Build comment tree if requested
	threaded := c.Query("threaded") == "true"
	if threaded {
		// Get current user for permissions
		var currentUserID *uuid.UUID
		if userIDValue := c.GetString("user_id"); userIDValue != "" {
			if parsedID, err := uuid.Parse(userIDValue); err == nil {
				currentUserID = &parsedID
			}
		}

		commentTree := models.BuildCommentTree(comments, currentUserID)

		c.JSON(http.StatusOK, gin.H{
			"comments":    commentTree,
			"total_count": totalCount,
			"page":        page,
			"limit":       limit,
			"total_pages": (totalCount + limit - 1) / limit,
			"threaded":    true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comments":    comments,
		"total_count": totalCount,
		"page":        page,
		"limit":       limit,
		"total_pages": (totalCount + limit - 1) / limit,
		"threaded":    false,
	})
}

// CreateGuestComment creates a guest comment without any auth middleware
func (ws *WorkService) CreateGuestComment(c *gin.Context) {
	var req models.CommentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// For guest comments, require guest name
	if req.GuestName == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Guest name is required for guest comments"})
		return
	}

	// Get work ID from URL parameter
	workIDStr := c.Param("work_id")
	workID, err := uuid.Parse(workIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}
	req.WorkID = &workID

	// Validate the request
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment data"})
		return
	}

	// Verify the work exists
	var exists bool
	err = ws.db.QueryRow("SELECT EXISTS(SELECT 1 FROM works WHERE id = $1)", req.WorkID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}

	// Create the comment using same logic as CreateComment
	commentID := uuid.New()
	ipAddress := c.ClientIP()

	// Handle empty IP address for test environments
	var ipParam interface{}
	if ipAddress == "" {
		ipParam = nil
	} else {
		ipParam = ipAddress
	}

	query := `
		INSERT INTO comments (
			id, work_id, chapter_id, user_id, pseudonym_id, parent_comment_id,
			content, guest_name, guest_email, ip_address, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW()
		)
	`

	_, err = ws.db.Exec(query,
		commentID, req.WorkID, req.ChapterID, nil, nil, req.ParentCommentID,
		html.Clean(req.Content), req.GuestName, req.GuestEmail, ipParam,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}

	// Retrieve the created comment with details
	comment, err := ws.getCommentByID(commentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Comment created but failed to retrieve details"})
		return
	}

	// Trigger notification for comment creation
	go ws.triggerCommentNotification(comment, "comment_created")

	c.JSON(http.StatusCreated, comment)
}

// CreateComment creates a new comment on a work or chapter
func (ws *WorkService) CreateComment(c *gin.Context) {
	var req models.CommentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Extract work ID from URL parameter if not provided in request
	if req.WorkID == nil {
		workIDStr := c.Param("work_id")
		if workIDStr != "" {
			if workID, err := uuid.Parse(workIDStr); err == nil {
				req.WorkID = &workID
			}
		}
	}

	// Who is commenting: a logged-in user under one of their own pseuds, or a
	// guest with a name.
	var userID *uuid.UUID
	var pseudonymID *uuid.UUID
	if u := ws.getUserIDFromContext(c); u != nil {
		userID = u
		pid, err := ws.commentPseud(*u, req.PseudonymID)
		if err == errNotYourPseud {
			c.JSON(http.StatusForbidden, gin.H{"error": "That pseud isn't yours"})
			return
		}
		if err != nil {
			log.Printf("create comment: pseud: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
			return
		}
		pseudonymID = &pid
		req.PseudonymID = &pid
		req.GuestName, req.GuestEmail = nil, nil
	} else if req.GuestName == nil || strings.TrimSpace(*req.GuestName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Guest name is required for anonymous comments"})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment data"})
		return
	}

	// A chapter comment belongs to that chapter's work.
	if req.ChapterID != nil {
		var chapterWork uuid.UUID
		err := ws.db.QueryRow("SELECT work_id FROM chapters WHERE id = $1", req.ChapterID).Scan(&chapterWork)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chapter not found"})
			return
		}
		if req.WorkID != nil && *req.WorkID != chapterWork {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chapter is not part of this work"})
			return
		}
		req.WorkID = &chapterWork
	}

	// You can only comment on a work you're allowed to see (not a draft, not
	// restricted when logged out, not from an author who blocked you).
	var canView bool
	if err := ws.db.QueryRow("SELECT can_user_view_work($1, $2)", req.WorkID, userID).Scan(&canView); err != nil || !canView {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}

	// A reply must be to a comment on the same work.
	if req.ParentCommentID != nil {
		var exists bool
		err := ws.db.QueryRow("SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1 AND work_id = $2)",
			req.ParentCommentID, req.WorkID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Parent comment not found"})
			return
		}
	}

	// Create the comment
	commentID := uuid.New()
	ipAddress := c.ClientIP()

	// Handle empty IP address for test environments
	var ipParam interface{}
	if ipAddress == "" {
		ipParam = nil
	} else {
		ipParam = ipAddress
	}

	query := `
		INSERT INTO comments (
			id, work_id, chapter_id, user_id, pseudonym_id, parent_comment_id,
			content, guest_name, guest_email, ip_address, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW()
		)
	`

	_, err := ws.db.Exec(query,
		commentID, req.WorkID, req.ChapterID, userID, pseudonymID, req.ParentCommentID,
		html.Clean(req.Content), req.GuestName, req.GuestEmail, ipParam,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}

	// Retrieve the created comment with details
	comment, err := ws.getCommentByID(commentID)
	if err != nil {
		// Log error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Comment created but failed to retrieve details"})
		return
	}

	// Trigger notification for comment creation
	go ws.triggerCommentNotification(comment, "comment_created")

	c.JSON(http.StatusCreated, comment)
}

// triggerCommentNotification tells the work's subscribers about a new comment.
func (ws *WorkService) triggerCommentNotification(comment *models.CommentWithDetails, eventType string) {
	if ws.notificationService == nil || comment.WorkID == nil {
		return
	}

	notificationEventType := models.EventCommentReceived
	if comment.ParentCommentID != nil && *comment.ParentCommentID != uuid.Nil {
		notificationEventType = models.EventCommentReplied
	}

	event := &notifications.EventData{
		Type:        notificationEventType,
		SourceID:    *comment.WorkID,
		SourceType:  "work",
		Title:       "New comment on work",
		Description: fmt.Sprintf("%s left a comment on your work", comment.AuthorName),
		ActionURL:   fmt.Sprintf("/works/%s/comments/%s", comment.WorkID, comment.ID),
		ActorID:     comment.AuthorUserID,
		ActorName:   comment.AuthorName,
		ExtraData: map[string]interface{}{
			"comment_id":        comment.ID,
			"work_id":           comment.WorkID,
			"work_title":        comment.WorkTitle,
			"comment_content":   comment.Content,
			"parent_comment_id": comment.ParentCommentID,
		},
	}
	event.Recipients = ws.commentRecipients(*comment.WorkID, comment.ParentCommentID, comment.AuthorUserID)
	if err := ws.notificationService.ProcessEvent(context.Background(), event); err != nil {
		log.Printf("Failed to process comment notification for work %s: %v", comment.WorkID, err)
	}
}

// UpdateComment updates an existing comment
func (ws *WorkService) UpdateComment(c *gin.Context) {
	commentID := c.Param("commentId")
	if commentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment ID is required"})
		return
	}

	var req models.CommentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get user information from context
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Verify the comment exists and user owns it
	var existingComment models.Comment
	query := `SELECT id, user_id, content FROM comments WHERE id = $1 AND is_deleted = false`

	err = ws.db.QueryRow(query, commentID).Scan(
		&existingComment.ID, &existingComment.UserID, &existingComment.Content,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	if err != nil {
		// Log error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify comment"})
		return
	}

	// Check if user owns the comment
	if existingComment.UserID == nil || *existingComment.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only edit your own comments"})
		return
	}

	// Update the comment
	updateQuery := `
		UPDATE comments 
		SET content = $1, edited_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`

	_, err = ws.db.Exec(updateQuery, html.Clean(req.Content), commentID)
	if err != nil {
		// Log error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update comment"})
		return
	}

	// Retrieve the updated comment
	updatedComment, err := ws.getCommentByID(uuid.MustParse(commentID))
	if err != nil {
		// Log error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Comment updated but failed to retrieve details"})
		return
	}

	c.JSON(http.StatusOK, updatedComment)
}

// DeleteComment soft-deletes a comment
func (ws *WorkService) DeleteComment(c *gin.Context) {
	commentID := c.Param("commentId")
	if commentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment ID is required"})
		return
	}

	// Get user information from context
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check user roles for moderation permissions
	userRoles := c.GetStringSlice("user_roles")
	isModerator := false
	for _, role := range userRoles {
		if role == "moderator" || role == "admin" {
			isModerator = true
			break
		}
	}

	// Verify the comment exists
	var existingComment models.Comment
	query := `SELECT id, user_id FROM comments WHERE id = $1 AND is_deleted = false`

	err = ws.db.QueryRow(query, commentID).Scan(&existingComment.ID, &existingComment.UserID)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	if err != nil {
		// Log error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify comment"})
		return
	}

	// Check permissions: user owns comment OR user is moderator
	canDelete := isModerator || (existingComment.UserID != nil && *existingComment.UserID == userID)
	if !canDelete {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own comments"})
		return
	}

	// Soft delete the comment
	updateQuery := `UPDATE comments SET is_deleted = true, updated_at = NOW() WHERE id = $1`

	_, err = ws.db.Exec(updateQuery, commentID)
	if err != nil {
		// Log error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}

// GiveCommentKudos allows a user to give kudos to a comment
func (ws *WorkService) GiveCommentKudos(c *gin.Context) {
	commentID := c.Param("commentId")
	if commentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment ID is required"})
		return
	}

	// Get user information from context
	userIDStr := c.GetString("user_id")
	var userID *uuid.UUID
	var pseudonymID *uuid.UUID
	var guestSession *string

	if userIDStr != "" {
		// Authenticated user
		parsedUserID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		userID = &parsedUserID

		// Get the user's default pseudonym
		var defaultPseudonymID uuid.UUID
		err = ws.db.QueryRow(
			"SELECT id FROM user_pseudonyms WHERE user_id = $1 AND is_default = true LIMIT 1",
			userID,
		).Scan(&defaultPseudonymID)
		if err == nil {
			pseudonymID = &defaultPseudonymID
		}
	} else {
		// Guest user - create a session identifier
		sessionID := fmt.Sprintf("guest_%s_%d", c.ClientIP(), time.Now().Unix())
		guestSession = &sessionID
	}

	// Verify the comment exists
	var exists bool
	err := ws.db.QueryRow("SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1 AND is_deleted = false)", commentID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	// Check if user already gave kudos to this comment
	checkQuery := `
		SELECT EXISTS(
			SELECT 1 FROM comment_kudos 
			WHERE comment_id = $1 AND (
				(user_id = $2 AND $2 IS NOT NULL) OR 
				(guest_session = $3 AND $3 IS NOT NULL)
			)
		)
	`

	var alreadyGaveKudos bool
	err = ws.db.QueryRow(checkQuery, commentID, userID, guestSession).Scan(&alreadyGaveKudos)
	if err != nil {
		// Log error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify kudos status"})
		return
	}

	if alreadyGaveKudos {
		c.JSON(http.StatusConflict, gin.H{"error": "You have already given kudos to this comment"})
		return
	}

	// Give kudos
	kudosID := uuid.New()
	ipAddress := c.ClientIP()

	// Handle empty IP address for test environments
	var ipParam interface{}
	if ipAddress == "" {
		ipParam = nil
	} else {
		ipParam = ipAddress
	}

	insertQuery := `
		INSERT INTO comment_kudos (
			id, comment_id, user_id, pseudonym_id, guest_session, ip_address, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, NOW()
		)
	`

	_, err = ws.db.Exec(insertQuery, kudosID, commentID, userID, pseudonymID, guestSession, ipParam)
	if err != nil {
		// Log error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to give kudos"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Kudos given successfully",
		"kudos_id": kudosID,
	})
}

// Helper function to get a comment by ID with all details
func (ws *WorkService) getCommentByID(commentID uuid.UUID) (*models.CommentWithDetails, error) {
	query := `
		SELECT 
			c.id, c.work_id, c.chapter_id, c.user_id, c.pseudonym_id, c.parent_comment_id,
			c.content, c.guest_name, c.guest_email, c.is_deleted, c.is_moderated, c.is_spam,
			c.thread_level, c.kudos_count, c.reply_count, c.created_at, c.updated_at, c.edited_at,
			COALESCE(up.name, u.username, c.guest_name) as author_name,
			u.id as author_user_id,
			up.id as author_pseudonym_id,
			CASE 
				WHEN c.guest_name IS NOT NULL THEN 'guest'
				WHEN u.id IS NOT NULL THEN 'user'
				ELSE 'unknown'
			END as author_type,
			w.title as work_title,
			w.user_id as work_author_id
		FROM comments c
		LEFT JOIN users u ON c.user_id = u.id
		LEFT JOIN user_pseudonyms up ON c.pseudonym_id = up.id
		LEFT JOIN works w ON c.work_id = w.id
		WHERE c.id = $1
	`

	var comment models.CommentWithDetails
	var pseudonymID, userID, parentCommentID sql.NullString
	var chapterID sql.NullString
	var editedAt sql.NullTime

	err := ws.db.QueryRow(query, commentID).Scan(
		&comment.ID, &comment.WorkID, &chapterID, &userID, &pseudonymID, &parentCommentID,
		&comment.Content, &comment.GuestName, &comment.GuestEmail, &comment.IsDeleted,
		&comment.IsModerated, &comment.IsSpam, &comment.ThreadLevel, &comment.KudosCount,
		&comment.ReplyCount, &comment.CreatedAt, &comment.UpdatedAt, &editedAt,
		&comment.AuthorName, &comment.AuthorUserID, &comment.AuthorPseudonymID,
		&comment.AuthorType, &comment.WorkTitle, &comment.WorkAuthorID,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if chapterID.Valid {
		chapterUUID, _ := uuid.Parse(chapterID.String)
		comment.ChapterID = &chapterUUID
	}
	if userID.Valid {
		userUUID, _ := uuid.Parse(userID.String)
		comment.UserID = &userUUID
	}
	if pseudonymID.Valid {
		pseudUUID, _ := uuid.Parse(pseudonymID.String)
		comment.PseudonymID = &pseudUUID
	}
	if parentCommentID.Valid {
		parentUUID, _ := uuid.Parse(parentCommentID.String)
		comment.ParentCommentID = &parentUUID
	}
	if editedAt.Valid {
		comment.EditedAt = &editedAt.Time
	}

	return &comment, nil
}

var errNotYourPseud = errors.New("pseud belongs to someone else")

// commentPseud returns the pseud a user comments under: the one they asked
// for, if it's theirs, or else their default (created from their username if
// they don't have one yet).
func (ws *WorkService) commentPseud(userID uuid.UUID, requested *uuid.UUID) (uuid.UUID, error) {
	if requested != nil {
		var owner uuid.UUID
		err := ws.db.QueryRow("SELECT user_id FROM user_pseudonyms WHERE id = $1", *requested).Scan(&owner)
		if err == sql.ErrNoRows || (err == nil && owner != userID) {
			return uuid.Nil, errNotYourPseud
		}
		return *requested, err
	}
	var id uuid.UUID
	err := ws.db.QueryRow(`SELECT id FROM user_pseudonyms WHERE user_id = $1
		ORDER BY is_default DESC, created_at LIMIT 1`, userID).Scan(&id)
	if err != sql.ErrNoRows {
		return id, err
	}
	err = ws.db.QueryRow(`
		INSERT INTO user_pseudonyms (user_id, name, is_default)
		SELECT id, username, true FROM users WHERE id = $1
		ON CONFLICT (user_id, name) DO UPDATE SET is_default = user_pseudonyms.is_default
		RETURNING id`, userID).Scan(&id)
	return id, err
}

// commentRecipients are the people a new comment is for: the work's authors,
// and for a reply, whoever wrote the comment being replied to (on the same
// work). Anyone who has blocked the commenter is left out.
func (ws *WorkService) commentRecipients(workID uuid.UUID, parentID, commenterID *uuid.UUID) []uuid.UUID {
	rows, err := ws.db.Query(`
		SELECT r.user_id FROM (
			SELECT p.user_id FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id
			WHERE c.creation_id = $1 AND c.creation_type = 'Work' AND c.approved
			UNION
			SELECT user_id FROM comments WHERE id = $2 AND work_id = $1 AND user_id IS NOT NULL
		) r
		WHERE $3::uuid IS NULL OR NOT EXISTS (
			SELECT 1 FROM user_blocks b
			WHERE b.blocker_id = r.user_id AND b.blocked_id = $3 AND b.block_type IN ('full', 'comments'))`,
		workID, parentID, commenterID)
	if err != nil {
		log.Printf("comment recipients for work %s: %v", workID, err)
		return nil
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
