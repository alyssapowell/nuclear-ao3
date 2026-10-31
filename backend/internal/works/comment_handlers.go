package works

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/audit"
	"nuclear-ao3/internal/html"
	"nuclear-ao3/internal/moderation"
	"nuclear-ao3/shared/models"
	"nuclear-ao3/shared/notifications"
)

// Comment handlers for the work service

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
	// restricted when logged out).
	var canView bool
	if err := ws.db.QueryRow("SELECT can_user_view_work($1, $2)", req.WorkID, userID).Scan(&canView); err != nil || !canView {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	// ...and a chapter you're allowed to see (not another draft chapter).
	if req.ChapterID != nil {
		var canViewChapter bool
		if err := ws.db.QueryRow("SELECT can_user_view_chapter($1, $2)", req.ChapterID, userID).Scan(&canViewChapter); err != nil || !canViewChapter {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chapter not found"})
			return
		}
	}

	// A block between the commenter and one of the work's creators closes
	// comments and replies on it (AO3 comment.rb:49-55). This comes before
	// anything that looks at other comments, which the commenter can't see.
	//
	// On a work that hides its creators nothing is refused, because saying
	// "blocked" would tell the commenter who wrote it. The comment is saved
	// as shadowed instead (silent below): the commenter and moderators can
	// see it, and nobody else is told about it.
	silent, err := ws.blockedSilently(c, *req.WorkID, userID)
	if err != nil {
		log.Printf("create comment: blocked_silently: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}
	if blocked, err := ws.blockedFromWork(c, *req.WorkID, userID); err != nil {
		log.Printf("create comment: blocked_from_work: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	} else if blocked {
		c.JSON(http.StatusForbidden, commentsDisabledBody())
		return
	}

	// Nobody can comment on a work or chapter an admin has hidden, not even
	// its creator (AO3 comments_controller.rb:76 and :158).
	var workHidden, chapterHidden bool
	if err := ws.db.QueryRow("SELECT hidden_by_admin OR spam FROM works WHERE id = $1", req.WorkID).Scan(&workHidden); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	if req.ChapterID != nil {
		if err := ws.db.QueryRow("SELECT hidden_by_admin FROM chapters WHERE id = $1", req.ChapterID).Scan(&chapterHidden); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chapter not found"})
			return
		}
	}
	if workHidden || chapterHidden {
		c.JSON(http.StatusForbidden, gin.H{"error": "work_hidden",
			"message": "You can't add comments to something an admin has hidden."})
		return
	}

	// A reply must be to a comment on the same work, that the replier can
	// see, and that can still take replies (AO3 comments_controller.rb:144-165).
	if req.ParentCommentID != nil {
		var exists, visible, iced, unreviewed, spam, hidden bool
		err := ws.db.QueryRow(`SELECT true, can_user_view_comment(c.id, $3), c.iced, c.unreviewed, c.is_spam, c.hidden_by_admin
			FROM comments c LEFT JOIN chapters ch ON ch.id = c.chapter_id
			WHERE c.id = $1 AND COALESCE(c.work_id, ch.work_id) = $2`,
			req.ParentCommentID, req.WorkID, userID).Scan(&exists, &visible, &iced, &unreviewed, &spam, &hidden)
		if err != nil || !exists || !visible {
			c.JSON(http.StatusNotFound, gin.H{"error": "Parent comment not found"})
			return
		}
		// Nobody can reply to a comment by an account they're blocked with
		// (AO3 comment.rb:40-46). Both ways, though a blocker can't see the
		// blocked person's comments, so that case already ended above.
		if userID != nil && !silent {
			var replyBlocked bool
			if err := ws.db.QueryRow(`SELECT accounts_blocked(c.user_id, $2) FROM comments c WHERE c.id = $1`,
				req.ParentCommentID, *userID).Scan(&replyBlocked); err != nil {
				log.Printf("create comment: reply block: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
				return
			}
			if replyBlocked {
				c.JSON(http.StatusForbidden, gin.H{"error": "replies_disabled", "message": "You can't reply to this comment."})
				return
			}
		}
		switch {
		case iced:
			c.JSON(http.StatusForbidden, gin.H{"error": "thread_frozen", "message": "This thread is frozen; it can't take new replies."})
			return
		case unreviewed:
			c.JSON(http.StatusForbidden, gin.H{"error": "comment_unreviewed", "message": "You can't reply to a comment that hasn't been approved yet."})
			return
		case spam:
			c.JSON(http.StatusForbidden, gin.H{"error": "comment_is_spam", "message": "You can't reply to a comment marked as spam."})
			return
		case hidden:
			c.JSON(http.StatusForbidden, gin.H{"error": "comment_hidden", "message": "You can't reply to a comment an admin has hidden."})
			return
		}
	}

	// With moderated commenting on, a comment waits for the work's creator to
	// approve it, unless a creator wrote it (AO3 comment.rb:403-409).
	var unreviewed bool
	if err := ws.db.QueryRow(`SELECT COALESCE(w.moderate_comments, false) AND NOT is_work_creator(w.id, $2) FROM works w WHERE w.id = $1`,
		req.WorkID, userID).Scan(&unreviewed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
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
			content, guest_name, guest_email, ip_address, unreviewed, shadowed, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW()
		)
	`

	_, err = ws.db.Exec(query,
		commentID, req.WorkID, req.ChapterID, userID, pseudonymID, req.ParentCommentID,
		html.Clean(req.Content), req.GuestName, req.GuestEmail, ipParam, unreviewed, silent,
	)

	if err != nil {
		log.Printf("create comment: %v", err)
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

	// Trigger notification for comment creation. A shadowed comment tells
	// nobody (the database trigger and the notifier both skip it).
	if !silent {
		go ws.triggerCommentNotification(comment, "comment_created")
	}

	c.JSON(http.StatusCreated, comment)
}

// triggerCommentNotification tells the work's subscribers about a new comment.
func (ws *WorkService) triggerCommentNotification(comment *models.CommentWithDetails, eventType string) {
	if ws.notificationService == nil || comment.WorkID == nil {
		return
	}
	// A comment waiting for review, marked as spam or on a hidden work is
	// not announced; AO3 tells people once the creator approves it.
	var quiet bool
	if err := ws.db.QueryRow(`SELECT c.unreviewed OR c.is_spam OR c.shadowed OR c.hidden_by_admin OR w.hidden_by_admin OR w.spam
		FROM comments c JOIN works w ON w.id = $2 WHERE c.id = $1`, comment.ID, comment.WorkID).Scan(&quiet); err != nil || quiet {
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
		ActorHidden: comment.AnonymousCreator,
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
	commentID, err := uuid.Parse(c.Param("comment_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	var req models.CommentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}
	content := html.Clean(req.Content)
	if strings.TrimSpace(content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment cannot be empty"})
		return
	}

	userID := ws.getUserIDFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var authorID sql.NullString
	var iced, hidden bool
	var visible, hasReplies, blocked, silent bool
	err = ws.db.QueryRowContext(c.Request.Context(), `
		SELECT c.user_id, c.iced, c.hidden_by_admin,
		       can_user_view_comment(c.id, $2),
		       EXISTS (SELECT 1 FROM comments r WHERE r.parent_comment_id = c.id AND NOT r.is_deleted),
		       blocked_from_work(COALESCE(c.work_id, ch.work_id), $2),
		       blocked_silently(COALESCE(c.work_id, ch.work_id), $2)
		FROM comments c LEFT JOIN chapters ch ON ch.id = c.chapter_id
		WHERE c.id = $1 AND NOT c.is_deleted`,
		commentID, *userID).Scan(&authorID, &iced, &hidden, &visible, &hasReplies, &blocked, &silent)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify comment"})
		return
	}

	// The order matters. First the comment has to exist for this person, and
	// be theirs; only then is the block looked at. If the block came first, a
	// blocked person could tell a comment they can't see from one that
	// doesn't exist, and could learn about a block from a comment id.
	own := authorID.Valid && authorID.String == userID.String()
	// A comment the viewer cannot see does not exist to them. (A blocked
	// person can't see any comment on the work, but their own comment is
	// theirs: they get the "comments are disabled" answer below.)
	if !visible && !(own && blocked) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	// AO3 check_ownership: only the comment's own author may edit.
	if !own {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only edit your own comments"})
		return
	}
	// A block closes editing too (AO3 comments_controller.rb:37). On a work
	// that hides its creators the edit goes through instead, and the edited
	// comment becomes shadowed (silent), as a new comment from them would be.
	if blocked {
		c.JSON(http.StatusForbidden, commentsDisabledBody())
		return
	}
	// AO3 check_permission_to_edit (comments_controller.rb:207): a frozen
	// comment cannot be edited, and neither can one that has replies.
	if iced {
		c.JSON(http.StatusForbidden, gin.H{"error": "comment_frozen", "message": "This comment is frozen and cannot be edited."})
		return
	}
	if hasReplies {
		c.JSON(http.StatusForbidden, gin.H{"error": "comment_has_replies", "message": "Comments with replies cannot be edited."})
		return
	}
	if hidden {
		c.JSON(http.StatusForbidden, gin.H{"error": "comment_hidden", "message": "This comment has been hidden by an admin and cannot be edited."})
		return
	}

	if _, err = ws.db.ExecContext(c.Request.Context(), `
		UPDATE comments SET content = $1, edited_at = NOW(), updated_at = NOW(),
			shadowed = shadowed OR $3
		WHERE id = $2`, content, commentID, silent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update comment"})
		return
	}

	updatedComment, err := ws.getCommentByID(commentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Comment updated but failed to retrieve details"})
		return
	}

	c.JSON(http.StatusOK, updatedComment)
}

// DeleteComment soft-deletes a comment. As in AO3, the comment's author and
// the creators of the work it is on may delete it. Staff delete through the
// admin endpoint instead, which records the action in the audit log.
func (ws *WorkService) DeleteComment(c *gin.Context) {
	commentID := c.Param("comment_id")
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

	// Verify the comment exists, whether the caller created its work, and
	// whether they can see it.
	var existingComment models.Comment
	var ownsWork, visible, shadowed bool
	query := `SELECT id, user_id, COALESCE(is_work_creator(work_id, $2), false),
			can_user_view_comment(id, $2), shadowed
		FROM comments WHERE id = $1 AND is_deleted = false`

	err = ws.db.QueryRow(query, commentID, userID).Scan(&existingComment.ID, &existingComment.UserID, &ownsWork, &visible, &shadowed)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	if err != nil {
		// Log error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify comment"})
		return
	}

	// A comment the caller can't see doesn't exist to them: 404, not 403.
	// A work's creators can still delete what they can't list, such as spam
	// they marked (AO3 lets them), but never a shadowed comment, which they
	// must not learn exists.
	own := existingComment.UserID != nil && *existingComment.UserID == userID
	if !own && !visible && !(ownsWork && !shadowed) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	canDelete := ownsWork || (existingComment.UserID != nil && *existingComment.UserID == userID)
	if !canDelete {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own comments, or comments on your own works"})
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

// Helper function to get a comment by ID with all details
func (ws *WorkService) getCommentByID(commentID uuid.UUID) (*models.CommentWithDetails, error) {
	query := `
		SELECT 
			c.id, c.work_id, c.chapter_id, c.user_id, c.pseudonym_id, c.parent_comment_id,
			c.content, c.guest_name, c.guest_email, c.is_deleted, c.is_moderated, c.is_spam,
			c.thread_level, c.kudos_count, c.reply_count, c.created_at, c.updated_at, c.edited_at,
			CASE WHEN ` + creatorHiddenSQL + ` THEN '` + anonymousCreatorLabel + `'
			     ELSE COALESCE(up.name, c.guest_name) END as author_name,
			u.id as author_user_id,
			CASE WHEN ` + creatorHiddenSQL + ` THEN NULL ELSE up.id END as author_pseudonym_id,
			CASE 
				WHEN c.guest_name IS NOT NULL THEN 'guest'
				WHEN u.id IS NOT NULL THEN 'user'
				ELSE 'unknown'
			END as author_type,
			w.title as work_title,
			w.user_id as work_author_id,
			` + creatorHiddenSQL + ` as creator_hidden
		FROM comments c
		LEFT JOIN users u ON c.user_id = u.id
		LEFT JOIN pseuds up ON c.pseudonym_id = up.id
		LEFT JOIN works w ON c.work_id = w.id
		WHERE c.id = $1
	`

	var comment models.CommentWithDetails
	var pseudonymID, userID, parentCommentID sql.NullString
	var chapterID sql.NullString
	var editedAt sql.NullTime
	var creatorHidden bool

	err := ws.db.QueryRow(query, commentID).Scan(
		&comment.ID, &comment.WorkID, &chapterID, &userID, &pseudonymID, &parentCommentID,
		&comment.Content, &comment.GuestName, &comment.GuestEmail, &comment.IsDeleted,
		&comment.IsModerated, &comment.IsSpam, &comment.ThreadLevel, &comment.KudosCount,
		&comment.ReplyCount, &comment.CreatedAt, &comment.UpdatedAt, &editedAt,
		&comment.AuthorName, &comment.AuthorUserID, &comment.AuthorPseudonymID,
		&comment.AuthorType, &comment.WorkTitle, &comment.WorkAuthorID,
		&creatorHidden,
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
	shapeCommentPseud(&comment, creatorHidden)

	return &comment, nil
}

// creatorHiddenSQL is true for a comment written under a pseud of one of the
// creators of a work that hides its creators (anonymous, in an anonymous
// collection, or in a collection that hasn't revealed them yet). AO3 shows
// those as "Anonymous Creator"; showing the pseud would unmask the creator.
// It needs the comment's pseud as up and the work as w.
const anonymousCreatorLabel = "Anonymous Creator"

const creatorHiddenSQL = `(up.id IS NOT NULL AND (w.is_anonymous OR w.in_anon_collection OR w.in_unrevealed_collection)
				AND EXISTS (SELECT 1 FROM creatorships cr JOIN pseuds cp ON cp.id = cr.pseud_id
					WHERE cr.creation_id = w.id AND cr.creation_type = 'Work' AND cp.user_id = up.user_id))`

// shapeCommentPseud makes the pseud in a comment's answer what the query
// allowed (nothing, for an anonymous creator) and fills in the {id, name}
// form the API shows in place of any account detail.
//
// creatorHidden comes from the query (creatorHiddenSQL), never from the
// display name: a pseud may not be named "Anonymous Creator", and even if one
// got that name it must not pass as a hidden creator.
func shapeCommentPseud(c *models.CommentWithDetails, creatorHidden bool) {
	c.PseudonymID = c.AuthorPseudonymID
	c.AnonymousCreator = creatorHidden
	if c.AuthorPseudonymID != nil {
		c.Pseud = &models.PseudSummary{ID: c.AuthorPseudonymID.String(), Name: c.AuthorName}
	}
}

var errNotYourPseud = errors.New("pseud belongs to someone else")

// commentPseud returns the pseud a user comments under: the one they asked
// for, if it's theirs, or else their default (created from their username if
// they don't have one yet).
func (ws *WorkService) commentPseud(userID uuid.UUID, requested *uuid.UUID) (uuid.UUID, error) {
	if requested != nil {
		var owner uuid.UUID
		err := ws.db.QueryRow("SELECT user_id FROM pseuds WHERE id = $1", *requested).Scan(&owner)
		if err == sql.ErrNoRows || (err == nil && owner != userID) {
			return uuid.Nil, errNotYourPseud
		}
		return *requested, err
	}
	var id uuid.UUID
	err := ws.db.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1
		ORDER BY is_default DESC, created_at LIMIT 1`, userID).Scan(&id)
	if err != sql.ErrNoRows {
		return id, err
	}
	err = ws.db.QueryRow(`
		INSERT INTO pseuds (user_id, name, is_default)
		SELECT id, left(username::text, 40), true FROM users WHERE id = $1
		ON CONFLICT (user_id, lower(name)) DO UPDATE SET is_default = pseuds.is_default
		RETURNING id`, userID).Scan(&id)
	return id, err
}

// commentRecipients are the people a new comment is for: the work's authors,
// and for a reply, whoever wrote the comment being replied to (on the same
// work). Anyone who has blocked the commenter's account is left out, whichever
// of its pseuds wrote the comment.
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
			WHERE b.blocker_id = r.user_id AND b.blocked_id = $3)`,
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

// FreezeComment freezes a comment and every reply under it, so nobody can
// reply. AO3 lets the work's creator and admins do this
// (comments_controller.rb:217-231). When an admin who is not a creator does
// it, the action goes in the audit log.
func (ws *WorkService) FreezeComment(c *gin.Context)   { ws.setFrozen(c, true) }
func (ws *WorkService) UnfreezeComment(c *gin.Context) { ws.setFrozen(c, false) }

func (ws *WorkService) setFrozen(c *gin.Context, frozen bool) {
	commentID, err := uuid.Parse(c.Param("comment_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}
	userID := ws.getUserIDFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	ctx := c.Request.Context()
	var isCreator, isMod, visible, shadowed bool
	err = ws.db.QueryRowContext(ctx, `
		SELECT is_work_creator(COALESCE(c.work_id, ch.work_id), $2), is_site_moderator($2),
		       can_user_view_comment(c.id, $2), c.shadowed
		FROM comments c LEFT JOIN chapters ch ON ch.id = c.chapter_id
		WHERE c.id = $1`, commentID, userID).Scan(&isCreator, &isMod, &visible, &shadowed)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to freeze comment"})
		return
	}
	// A comment the caller can't see doesn't exist to them (404 before 403),
	// and a creator never learns of a shadowed one.
	if (!isCreator && !isMod && !visible) || (shadowed && !isMod) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	if !isCreator && !isMod {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the work's creators and admins can freeze or unfreeze comments"})
		return
	}
	if isMod && !isCreator {
		actor, ok := moderation.Actor(c, ws.db)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}
		if err := moderation.New(ws.db).SetCommentFrozen(ctx, actor, commentID, frozen, ""); err != nil {
			moderation.Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"frozen": frozen})
		return
	}
	tx, err := ws.db.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to freeze comment"})
		return
	}
	defer tx.Rollback()
	n, err := moderation.FreezeThread(ctx, tx, commentID, frozen)
	if err != nil || tx.Commit() != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to freeze comment"})
		return
	}
	if n == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "no_change", "message": "It is already in that state."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"frozen": frozen})
}

// auditAction records an admin action from one of the older admin endpoints
// in the audit log, inside the transaction that makes the change.
func (ws *WorkService) auditAction(c *gin.Context, tx *sql.Tx, action, targetType string, target uuid.UUID, reason string, details any) error {
	actor, ok := moderation.Actor(c, ws.db)
	if !ok {
		return errors.New("audit: no logged-in administrator")
	}
	return audit.Record(c.Request.Context(), tx, actor, action, targetType, target.String(), reason, details)
}
