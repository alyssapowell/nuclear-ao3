package works

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/platform"
	"nuclear-ao3/shared/models"
)

// GetWorkWithAuthors returns a work and its authors. Legacy numeric IDs
// redirect to the UUID URL.
func (ws *WorkService) GetWorkWithAuthors(c *gin.Context) {
	workIDParam := c.Param("work_id")
	ctx := c.Request.Context()

	// Parse work ID (UUID or legacy)
	workID, isRedirect, redirectURL, unknownLegacy := ws.parseWorkID(workIDParam)
	if unknownLegacy {
		// A well-formed AO3 number we have no work for (an old link)
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	if isRedirect {
		c.Redirect(http.StatusMovedPermanently, redirectURL)
		return
	}
	if workID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID format"})
		return
	}

	// An app sees the work as any logged-in reader would (see
	// platform.ReaderViewer): creator-only exceptions stay off.
	userID, asReader := platform.ReaderViewer(c)
	work, err := ws.fetchWorkFromDB(ctx, workID, platform.CreatorOnly(userID, asReader))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve work"})
		}
		return
	}

	// Apply privacy filters
	if !ws.canViewWork(ctx, workID, userID, asReader) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}

	ws.recordReading(c, workID, 0)

	// A reader who is blocked from this work's creators still reads it, but
	// sees comments as disabled and no comment count. (Not on a work that
	// hides its creators: there the block is silent, blockedFromWork is false,
	// and the reader sees exactly what everyone else sees.)
	if blocked, err := ws.blockedFromWork(c, workID, userID); err != nil {
		log.Printf("blocked_from_work(%s): %v", workID, err)
	} else if blocked {
		work.CommentsDisabledForYou = true
		work.Comments = 0
	}

	// Fetch authors (what the viewer may see depends on who they are)
	authors, err := ws.fetchWorkAuthors(ctx, workID, userID)
	if err != nil {
		log.Printf("Failed to fetch authors for work %s: %v", workID, err)
		// Continue without authors rather than failing
		authors = []models.WorkAuthor{}
	}

	// Return work with authors in expected format
	response := gin.H{
		"work":    work,
		"authors": authors,
	}
	c.JSON(http.StatusOK, response)
}

// GetWorkStats returns a work's counts.
func (ws *WorkService) GetWorkStats(c *gin.Context) {
	workIDParam := c.Param("work_id")
	ctx := c.Request.Context()

	workID, err := uuid.Parse(workIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	viewer, asReader := platform.ReaderViewer(c)
	if !ws.canViewWork(ctx, workID, viewer, asReader) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}

	stats, err := ws.fetchWorkStatsFromDB(ctx, workID, platform.CreatorOnly(viewer, asReader))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve work stats"})
		return
	}
	// A reader blocked from this work's creators sees no comment count.
	if blocked, _ := ws.blockedFromWork(c, workID, viewer); blocked {
		stats["comments"] = 0
	}

	c.JSON(http.StatusOK, stats)
}

// Helper functions

func (ws *WorkService) parseWorkID(workIDParam string) (workID uuid.UUID, isRedirect bool, redirectURL string, unknownLegacy bool) {
	// Try UUID first
	if workID, err := uuid.Parse(workIDParam); err == nil {
		return workID, false, "", false
	}

	// Try legacy ID (an AO3 number; negatives are not numbers)
	if legacyID, err := strconv.Atoi(workIDParam); err == nil && legacyID >= 0 {
		// Look up UUID and redirect
		var workUUID uuid.UUID
		err = ws.db.QueryRow("SELECT id FROM works WHERE legacy_id = $1", legacyID).Scan(&workUUID)
		if err != nil {
			return uuid.Nil, false, "", true
		}
		redirectURL := fmt.Sprintf("/api/v1/work/%s", workUUID.String())
		return uuid.Nil, true, redirectURL, false
	}

	return uuid.Nil, false, "", false
}

// fetchWorkFromDB loads a work for the given viewer (nil for a guest). The
// kudos and comment counts include the viewer's own shadowed kudos and
// comments, so a silently blocked person never sees their own post go
// uncounted.
func (ws *WorkService) fetchWorkFromDB(ctx context.Context, workID uuid.UUID, viewer *uuid.UUID) (*models.Work, error) {
	var work models.Work

	// Handle nullable fields with sql.NullString or use database default values
	var legacyID sql.NullInt32
	var maxChapters sql.NullInt32
	var publishedAt sql.NullTime

	// Handle array fields that might be NULL
	var fandoms, characters, relationships, freeformTags pq.StringArray

	err := ws.db.QueryRowContext(ctx, `
		SELECT 
			w.id, w.legacy_id, w.title, 
			COALESCE(w.summary, '') as summary, 
			COALESCE(w.notes, '') as notes, 
			w.user_id, w.language, w.rating, 
			COALESCE(w.word_count, 0) as word_count, 
			COALESCE(w.chapter_count, 1) as chapter_count, 
			w.max_chapters,
			COALESCE(w.status, 'draft') as status, 
			(COALESCE(w.restricted, false) OR COALESCE(w.restricted_to_users, false)) as restricted_to_users,
			COALESCE(w.restricted_to_adults, false) as restricted_to_adults,
			COALESCE(w.comment_policy, 'open') as comment_policy,
			COALESCE(w.moderate_comments, false) as moderate_comments,
			COALESCE(w.disable_comments, false) as disable_comments,
			COALESCE(w.in_anon_collection, false) as in_anon_collection,
			COALESCE(w.in_unrevealed_collection, false) as in_unrevealed_collection,
			COALESCE(w.is_anonymous, false) as is_anonymous,
			COALESCE(w.fandoms, '{}') as fandoms,
			COALESCE(w.characters, '{}') as characters,
			COALESCE(w.relationships, '{}') as relationships,
			COALESCE(w.freeform_tags, '{}') as freeform_tags,
			w.published_at, w.updated_at, w.created_at,
			w.hidden_by_admin, w.spam,
			COALESCE(w.hit_count, 0),
			COALESCE(w.kudos_count, 0) + own_shadowed_kudos(w.id, $2::uuid),
			COALESCE(w.comment_count, 0) + own_shadowed_comments(w.id, $2::uuid),
			COALESCE(w.bookmark_count, 0)
		FROM works w
		JOIN users u ON w.user_id = u.id
		WHERE w.id = $1
	`, workID, viewer).Scan(
		&work.ID, &legacyID, &work.Title, &work.Summary, &work.Notes,
		&work.UserID, &work.Language, &work.Rating,
		&work.WordCount, &work.ChapterCount, &maxChapters, &work.Status,
		&work.RestrictedToUsers, &work.RestrictedToAdults, &work.CommentPolicy,
		&work.ModerateComments, &work.DisableComments, &work.InAnonCollection,
		&work.InUnrevealedCollection, &work.IsAnonymous,
		&fandoms, &characters, &relationships, &freeformTags,
		&publishedAt, &work.UpdatedAt, &work.CreatedAt,
		&work.HiddenByAdmin, &work.Spam,
		&work.Hits, &work.Kudos, &work.Comments, &work.Bookmarks,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err // Return sql.ErrNoRows specifically
		}
		log.Printf("fetch work %s: %v", workID, err)
		return nil, err
	}

	// Handle nullable fields
	if legacyID.Valid {
		legacyIDInt := int(legacyID.Int32)
		work.LegacyID = &legacyIDInt
	}

	if maxChapters.Valid {
		maxChap := int(maxChapters.Int32)
		work.MaxChapters = &maxChap
	}

	if publishedAt.Valid {
		work.PublishedAt = &publishedAt.Time
	}

	// Convert arrays to slices
	work.Fandoms = []string(fandoms)
	work.Characters = []string(characters)
	work.Relationships = []string(relationships)
	work.FreeformTags = []string(freeformTags)

	// Handle warnings - could be TEXT or TEXT[] depending on migration state
	// For now, set empty array and let it be populated later if needed
	work.Warnings = []string{}
	work.Category = []string{}

	// Set computed field - work is complete if status is not draft
	work.IsComplete = work.Status != "draft"

	return &work, nil
}

func (ws *WorkService) fetchWorkAuthors(ctx context.Context, workID uuid.UUID, userID *uuid.UUID) ([]models.WorkAuthor, error) {
	// Try the advanced creatorship system first
	rows, err := ws.db.QueryContext(ctx, "SELECT * FROM get_work_authors($1, $2)", workID, userID)
	if err != nil {
		log.Printf("fetch authors of work %s: %v", workID, err)
		return nil, err
	}
	defer rows.Close()

	authors := []models.WorkAuthor{}
	for rows.Next() {
		var author models.WorkAuthor
		err := rows.Scan(&author.PseudID, &author.PseudName, &author.UserID, &author.IsAnonymous)
		if err != nil {
			log.Printf("fetch authors of work %s: scan: %v", workID, err)
			return nil, err
		}
		authors = append(authors, author)
	}

	// No creatorship means no author to show. Naming the account that owns
	// the work instead would give away the creator of an anonymous work, and
	// would name an account rather than a pseud.
	return authors, nil
}

// fetchWorkStatsFromDB returns a work's counts as the viewer sees them: the
// public counts plus the viewer's own shadowed kudos and comments.
func (ws *WorkService) fetchWorkStatsFromDB(ctx context.Context, workID uuid.UUID, viewer *uuid.UUID) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Fetch various stats
	var hits, kudos, comments, bookmarks int
	err := ws.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(hit_count, 0) as hits,
			COALESCE(kudos_count, 0) + own_shadowed_kudos(id, $2::uuid) as kudos,
			COALESCE(comment_count, 0) + own_shadowed_comments(id, $2::uuid) as comments,
			COALESCE(bookmark_count, 0) as bookmarks
		FROM works 
		WHERE id = $1
	`, workID, viewer).Scan(&hits, &kudos, &comments, &bookmarks)

	if err != nil {
		return nil, err
	}

	stats["hits"] = hits
	stats["kudos"] = kudos
	stats["comments"] = comments
	stats["bookmarks"] = bookmarks

	return stats, nil
}

func (ws *WorkService) getUserIDFromContext(c *gin.Context) *uuid.UUID {
	userID, hasUser := c.Get("user_id")
	if !hasUser {
		return nil
	}

	userIDStr := userID.(string)
	if userVal, err := uuid.Parse(userIDStr); err == nil {
		return &userVal
	}

	return nil
}

// canViewWork asks the database, which knows about drafts, co-authors,
// restricted works, blocks and mutes. asReader is true for an app: then the
// creator and staff exceptions are off (migration 050).
func (ws *WorkService) canViewWork(ctx context.Context, workID uuid.UUID, userID *uuid.UUID, asReader bool) bool {
	var ok bool
	if err := ws.db.QueryRowContext(ctx, "SELECT can_user_view_work($1, $2, $3)", workID, userID, asReader).Scan(&ok); err != nil {
		log.Printf("can_user_view_work(%s): %v", workID, err)
		return false
	}
	return ok
}

// ownerArg is platform.OwnerArg, shortened for the queries in this package.
func ownerArg(viewerN, readerN int) string { return platform.OwnerArg(viewerN, readerN) }
