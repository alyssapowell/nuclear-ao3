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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve work"})
		return
	}

	// Apply privacy filters (this needs to be done per-request)
	userID := ws.getUserIDFromContext(c)
	if !ws.canViewWork(&cachedWork, userID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}

	c.JSON(http.StatusOK, cachedWork)
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

	// Full query to get work details
	var categoryStr, warningStr, notesStr, summaryStr sql.NullString
	err := ws.db.QueryRowContext(ctx, `
		SELECT w.id, w.title, w.summary, w.notes, w.user_id, u.username, 
			   w.language, w.rating, w.category, w.archive_warning,
			   w.word_count, w.chapter_count, w.expected_chapters, w.is_complete, w.status,
			   w.published_at, w.updated_at, w.created_at,
			   COALESCE(w.hit_count, 0) as hits, COALESCE(w.kudos_count, 0) as kudos,
			   COALESCE(w.comment_count, 0) as comments, COALESCE(w.bookmark_count, 0) as bookmarks,
			   w.restricted, w.restricted_to_adults, w.comment_policy,
			   w.moderate_comments, w.disable_comments, w.in_anon_collection,
			   w.in_unrevealed_collection, w.is_anonymous
		FROM works w
		JOIN users u ON w.user_id = u.id
		WHERE w.id = $1 AND w.status != 'draft'
	`, workID).Scan(
		&work.ID, &work.Title, &summaryStr, &notesStr, &work.UserID, &work.Username,
		&work.Language, &work.Rating, &categoryStr, &warningStr,
		&work.WordCount, &work.ChapterCount, &work.MaxChapters, &work.IsComplete, &work.Status,
		&work.PublishedAt, &work.UpdatedAt, &work.CreatedAt,
		&work.Hits, &work.Kudos, &work.Comments, &work.Bookmarks,
		&work.RestrictedToUsers, &work.RestrictedToAdults, &work.CommentPolicy,
		&work.ModerateComments, &work.DisableComments, &work.InAnonCollection,
		&work.InUnrevealedCollection, &work.IsAnonymous,
	)

	if err != nil {
		log.Printf("DEBUG: fetchWorkFromDB database query error: %v", err)
		return nil, err
	}

	log.Printf("DEBUG: fetchWorkFromDB successfully fetched work %s", work.Title)

	// Convert nullable fields
	if summaryStr.Valid {
		work.Summary = summaryStr.String
	}
	if notesStr.Valid {
		work.Notes = notesStr.String
	}

	// Convert category and warning to arrays
	if categoryStr.Valid && categoryStr.String != "" {
		work.Category = []string{categoryStr.String}
	}
	if warningStr.Valid && warningStr.String != "" {
		work.Warnings = []string{warningStr.String}
	}

	if err != nil {
		return nil, err
	}

	// Convert category and warning to arrays
	if categoryStr.Valid && categoryStr.String != "" {
		work.Category = []string{categoryStr.String}
	}
	if warningStr.Valid && warningStr.String != "" {
		work.Warnings = []string{warningStr.String}
	}

	// Load tags from work_tags relationship table
	work.Fandoms, work.Characters, work.Relationships, work.FreeformTags = ws.loadWorkTags(workID.String())

	return work, nil
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
	// Implement privacy logic here
	// For now, simplified version - check if work is published
	// You'd implement proper privacy checks based on your work model

	// For demo, assume all works are viewable
	// In real implementation, check work status, privacy settings, etc.
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
