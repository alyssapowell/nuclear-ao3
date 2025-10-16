package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"nuclear-ao3/shared/models"
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
		req.Content, req.GuestName, req.GuestEmail, ipParam,
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

	// Validate the request
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment data"})
		return
	}

	// Get user information from context
	var userID *uuid.UUID
	var pseudonymID *uuid.UUID

	if userIDStr := c.GetString("user_id"); userIDStr != "" {
		if parsedUserID, err := uuid.Parse(userIDStr); err == nil {
			userID = &parsedUserID
			// If user is authenticated, ensure they have a pseudonym
			if req.PseudonymID == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Pseudonym is required for authenticated users"})
				return
			}
			pseudonymID = req.PseudonymID
		}
	}

	// For guest comments, ensure guest name is provided
	if userID == nil && req.GuestName == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Guest name is required for anonymous comments"})
		return
	}

	// Verify the work or chapter exists
	if req.WorkID != nil {
		var exists bool
		err := ws.db.QueryRow("SELECT EXISTS(SELECT 1 FROM works WHERE id = $1)", req.WorkID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
			return
		}
	}

	if req.ChapterID != nil {
		var exists bool
		err := ws.db.QueryRow("SELECT EXISTS(SELECT 1 FROM chapters WHERE id = $1)", req.ChapterID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chapter not found"})
			return
		}
	}

	// Verify parent comment exists if provided
	if req.ParentCommentID != nil {
		var exists bool
		err := ws.db.QueryRow("SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1)", req.ParentCommentID).Scan(&exists)
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
		req.Content, req.GuestName, req.GuestEmail, ipParam,
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

// triggerCommentNotification sends a notification event to the notification service
func (ws *WorkService) triggerCommentNotification(comment *models.CommentWithDetails, eventType string) {
	// Get notification service URL from environment
	notificationServiceURL := getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8004")

	// Determine the notification event type
	var notificationEventType string
	if comment.ParentCommentID != nil && *comment.ParentCommentID != uuid.Nil {
		notificationEventType = "comment_replied"
	} else {
		notificationEventType = "comment_received"
	}

	// Create event data
	eventData := map[string]interface{}{
		"type":        notificationEventType,
		"source_id":   comment.WorkID,
		"source_type": "work",
		"title":       fmt.Sprintf("New comment on work"),
		"description": fmt.Sprintf("%s left a comment on your work", comment.AuthorName),
		"action_url":  fmt.Sprintf("/works/%s/comments/%s", comment.WorkID, comment.ID),
		"actor_id":    comment.AuthorUserID,
		"actor_name":  comment.AuthorName,
		"extra_data": map[string]interface{}{
			"comment_id":        comment.ID,
			"work_id":           comment.WorkID,
			"work_title":        comment.WorkTitle,
			"comment_content":   comment.Content,
			"parent_comment_id": comment.ParentCommentID,
		},
	}

	// Send to notification service
	jsonData, err := json.Marshal(eventData)
	if err != nil {
		fmt.Printf("Failed to marshal notification event: %v\n", err)
		return
	}

	resp, err := http.Post(
		notificationServiceURL+"/api/v1/process-event",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		fmt.Printf("Failed to send notification event: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Notification service returned status: %d\n", resp.StatusCode)
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

	_, err = ws.db.Exec(updateQuery, req.Content, commentID)
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
