package main

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

	"nuclear-ao3/shared/cache"
	"nuclear-ao3/shared/models"
)

// CachedGetWork handles work retrieval with Redis caching
func (ws *WorkService) CachedGetWork(c *gin.Context) {
	workIDParam := c.Param("work_id")
	ctx := c.Request.Context()
	log.Printf("=== CACHEDGETWORK HANDLER CALLED for work_id: %s ===", workIDParam)

	// Parse work ID (UUID or legacy)
	workID, isRedirect, redirectURL := ws.parseWorkID(workIDParam)
	if isRedirect {
		log.Printf("DEBUG: Redirecting to: %s", redirectURL)
		c.Redirect(http.StatusMovedPermanently, redirectURL)
		return
	}
	if workID == uuid.Nil {
		log.Printf("DEBUG: Invalid work ID format: %s", workIDParam)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID format"})
		return
	}

	// Check cache first
	cacheKey := fmt.Sprintf("work:%s", workID.String())
	var cachedWork models.Work

	err := ws.cache.GetOrSet(ctx, cacheKey, &cachedWork, cache.MediumTTL, func() (interface{}, error) {
		// Cache miss - fetch from database
		return ws.fetchWorkFromDB(ctx, workID)
	})

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve work"})
		}
		return
	}

	// Apply privacy filters (this needs to be done per-request)
	userID := ws.getUserIDFromContext(c)
	if !ws.canViewWork(&cachedWork, userID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}

	// Fetch authors (not cached as it depends on viewer's permissions)
	authors, err := ws.fetchWorkAuthors(ctx, workID, userID)
	if err != nil {
		log.Printf("Failed to fetch authors for work %s: %v", workID, err)
		// Continue without authors rather than failing
		authors = []models.WorkAuthor{}
	}

	// Return work with authors in expected format
	response := gin.H{
		"work":    cachedWork,
		"authors": authors,
	}
	c.JSON(http.StatusOK, response)
}

// CachedGetWorkStats provides cached work statistics
func (ws *WorkService) CachedGetWorkStats(c *gin.Context) {
	workIDParam := c.Param("work_id")
	ctx := c.Request.Context()

	workID, err := uuid.Parse(workIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	cacheKey := fmt.Sprintf("work_stats:%s", workID.String())
	var stats map[string]interface{}

	err = ws.cache.GetOrSet(ctx, cacheKey, &stats, cache.ShortTTL, func() (interface{}, error) {
		return ws.fetchWorkStatsFromDB(ctx, workID)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve work stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CachedSearchWorks provides cached search results for common queries
func (ws *WorkService) CachedSearchWorks(c *gin.Context) {
	// Create cache key from query parameters
	cacheKey := ws.buildSearchCacheKey(c.Request.URL.Query())
	ctx := c.Request.Context()

	var results map[string]interface{}

	err := ws.cache.GetOrSet(ctx, cacheKey, &results, cache.ShortTTL, func() (interface{}, error) {
		// Cache miss - perform actual search
		return ws.performSearchFromDB(ctx, c.Request.URL.Query())
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, results)
}

// Helper functions

func (ws *WorkService) parseWorkID(workIDParam string) (uuid.UUID, bool, string) {
	// Try UUID first
	if workID, err := uuid.Parse(workIDParam); err == nil {
		return workID, false, ""
	}

	// Try legacy ID
	if legacyID, err := strconv.Atoi(workIDParam); err == nil {
		// Look up UUID and redirect
		var workUUID uuid.UUID
		err = ws.db.QueryRow("SELECT id FROM works WHERE legacy_id = $1", legacyID).Scan(&workUUID)
		if err != nil {
			return uuid.Nil, false, ""
		}
		redirectURL := fmt.Sprintf("/api/v1/work/%s", workUUID.String())
		return uuid.Nil, true, redirectURL
	}

	return uuid.Nil, false, ""
}

func (ws *WorkService) fetchWorkFromDB(ctx context.Context, workID uuid.UUID) (interface{}, error) {
	log.Printf("DEBUG: fetchWorkFromDB called for work %s", workID.String())
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
			w.user_id, u.username, w.language, w.rating, 
			COALESCE(w.word_count, 0) as word_count, 
			COALESCE(w.chapter_count, 1) as chapter_count, 
			w.max_chapters,
			COALESCE(w.status, 'draft') as status, 
			COALESCE(w.restricted_to_users, false) as restricted_to_users,
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
			w.published_at, w.updated_at, w.created_at
		FROM works w
		JOIN users u ON w.user_id = u.id
		WHERE w.id = $1
	`, workID).Scan(
		&work.ID, &legacyID, &work.Title, &work.Summary, &work.Notes,
		&work.UserID, &work.Username, &work.Language, &work.Rating,
		&work.WordCount, &work.ChapterCount, &maxChapters, &work.Status,
		&work.RestrictedToUsers, &work.RestrictedToAdults, &work.CommentPolicy,
		&work.ModerateComments, &work.DisableComments, &work.InAnonCollection,
		&work.InUnrevealedCollection, &work.IsAnonymous,
		&fandoms, &characters, &relationships, &freeformTags,
		&publishedAt, &work.UpdatedAt, &work.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("DEBUG: Work %s not found in database", workID.String())
			return nil, err // Return sql.ErrNoRows specifically
		}
		log.Printf("DEBUG: fetchWorkFromDB database query error: %v", err)
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

	log.Printf("DEBUG: fetchWorkFromDB successfully fetched work %s - Status: %s", work.Title, work.Status)

	return work, nil
}

func (ws *WorkService) fetchWorkAuthors(ctx context.Context, workID uuid.UUID, userID *uuid.UUID) ([]models.WorkAuthor, error) {
	log.Printf("DEBUG: fetchWorkAuthors called for work %s, user %v", workID.String(), userID)

	// Try the advanced creatorship system first
	rows, err := ws.db.QueryContext(ctx, "SELECT * FROM get_work_authors($1, $2)", workID, userID)
	if err != nil {
		log.Printf("DEBUG: Failed to query get_work_authors: %v", err)
		return nil, err
	}
	defer rows.Close()

	var authors []models.WorkAuthor
	for rows.Next() {
		var author models.WorkAuthor
		err := rows.Scan(&author.PseudID, &author.PseudName, &author.UserID, &author.Username, &author.IsAnonymous)
		if err != nil {
			log.Printf("DEBUG: Failed to scan author: %v", err)
			return nil, err
		}
		authors = append(authors, author)
	}

	// If no authors found via creatorship system, fall back to work.user_id
	if len(authors) == 0 {
		log.Printf("DEBUG: No authors found in creatorship table, falling back to work.user_id")
		var workUserID uuid.UUID
		var username string
		var isAnonymous bool

		err := ws.db.QueryRowContext(ctx, `
			SELECT w.user_id, u.username, COALESCE(w.is_anonymous, false)
			FROM works w 
			JOIN users u ON w.user_id = u.id 
			WHERE w.id = $1`, workID).Scan(&workUserID, &username, &isAnonymous)

		if err != nil {
			log.Printf("DEBUG: Failed to get work user: %v", err)
			return authors, nil // Return empty array rather than error
		}

		// Create a simple author entry
		fallbackAuthor := models.WorkAuthor{
			PseudID:     nil, // No pseud system fallback
			PseudName:   username,
			UserID:      &workUserID,
			Username:    username,
			IsAnonymous: isAnonymous,
		}
		authors = append(authors, fallbackAuthor)
		log.Printf("DEBUG: Added fallback author: %s", username)
	}

	log.Printf("DEBUG: fetchWorkAuthors returning %d authors", len(authors))
	return authors, nil
}

func (ws *WorkService) fetchWorkStatsFromDB(ctx context.Context, workID uuid.UUID) (interface{}, error) {
	stats := make(map[string]interface{})

	// Fetch various stats
	var hits, kudos, comments, bookmarks int
	err := ws.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(hit_count, 0) as hits,
			COALESCE(kudos_count, 0) as kudos,
			COALESCE(comment_count, 0) as comments,
			COALESCE(bookmark_count, 0) as bookmarks
		FROM works 
		WHERE id = $1
	`, workID).Scan(&hits, &kudos, &comments, &bookmarks)

	if err != nil {
		return nil, err
	}

	stats["hits"] = hits
	stats["kudos"] = kudos
	stats["comments"] = comments
	stats["bookmarks"] = bookmarks

	return stats, nil
}

func (ws *WorkService) performSearchFromDB(ctx context.Context, params map[string][]string) (interface{}, error) {
	// Placeholder for search implementation
	results := map[string]interface{}{
		"works": []interface{}{},
		"total": 0,
		"page":  1,
	}
	return results, nil
}

func (ws *WorkService) buildSearchCacheKey(params map[string][]string) string {
	// Build deterministic cache key from search parameters
	// For now, simplified version
	var query, page string
	if q, ok := params["q"]; ok && len(q) > 0 {
		query = q[0]
	}
	if p, ok := params["page"]; ok && len(p) > 0 {
		page = p[0]
	} else {
		page = "1"
	}
	return fmt.Sprintf("search:%s:page:%s", query, page)
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

func (ws *WorkService) canViewWork(work *models.Work, userID *uuid.UUID) bool {
	// Check if work is in draft status
	if work.Status == "draft" {
		// Only the author can view their own drafts
		if userID == nil || *userID != work.UserID {
			return false
		}
	}

	// Check other privacy settings
	if work.RestrictedToUsers && userID == nil {
		return false
	}

	// For now, allow all other access
	// In real implementation, check additional privacy settings, collections, etc.
	return true
}

// Cache invalidation helpers

func (ws *WorkService) InvalidateWorkCache(workID uuid.UUID) error {
	ctx := context.Background()

	// Invalidate work cache
	cacheKey := fmt.Sprintf("work:%s", workID.String())
	if err := ws.cache.Delete(ctx, cacheKey); err != nil {
		return err
	}

	// Invalidate stats cache
	statsKey := fmt.Sprintf("work_stats:%s", workID.String())
	if err := ws.cache.Delete(ctx, statsKey); err != nil {
		return err
	}

	// Invalidate related search caches
	return ws.cache.DeletePattern(ctx, "search:*")
}

func (ws *WorkService) InvalidateUserCache(userID uuid.UUID) error {
	ctx := context.Background()

	// Invalidate user-specific caches
	pattern := fmt.Sprintf("user:%s:*", userID.String())
	return ws.cache.DeletePattern(ctx, pattern)
}
