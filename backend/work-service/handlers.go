package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"nuclear-ao3/shared/models"
	"nuclear-ao3/shared/notifications"
)

// Work CRUD operations

func (ws *WorkService) CreateWork(c *gin.Context) {
	log.Printf("DEBUG: Using REGULAR CreateWork handler (NO auto-indexing)")
	var req models.CreateWorkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Get user ID from JWT token (would be set by auth middleware)
	userID, exists := c.Get("user_id")
	log.Printf("DEBUG: user_id from context: %v, exists: %v", userID, exists)
	if !exists {
		// For development, use a default user ID if not set by middleware
		userID = "672471fe-daa0-422d-8eea-4f9e4d1f285c" // testuser ID
		log.Printf("Warning: Using default user ID for development: %v", userID)
	}
	log.Printf("DEBUG: Final user_id to use: %v", userID)

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Create work
	workID := uuid.New()
	now := time.Now()

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Set defaults for required fields if not provided
	language := req.Language
	if language == "" {
		language = "en"
	}

	rating := req.Rating
	if rating == "" {
		rating = "Not Rated"
	}

	work := &models.Work{
		ID:                     workID,
		Title:                  req.Title,
		Summary:                req.Summary,
		Notes:                  req.Notes,
		UserID:                 userUUID,
		SeriesID:               req.SeriesID,
		Language:               language,
		Rating:                 rating,
		Category:               req.Category,
		Warnings:               req.Warnings,
		Fandoms:                req.Fandoms,
		Characters:             req.Characters,
		Relationships:          req.Relationships,
		FreeformTags:           req.FreeformTags,
		MaxChapters:            req.MaxChapters,
		ChapterCount:           1,
		IsComplete:             req.MaxChapters != nil && *req.MaxChapters == 1,
		Status:                 "draft",
		RestrictedToUsers:      false,  // Default to public
		CommentPolicy:          "open", // Default to open comments
		ModerateComments:       false,
		DisableComments:        false,
		IsAnonymous:            false, // Default to non-anonymous
		InAnonCollection:       false,
		InUnrevealedCollection: false,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	// Insert work with user_id (matching actual database schema)
	query := `
		INSERT INTO works (id, title, summary, notes, user_id, language, rating, 
			warnings, fandoms, characters, relationships, freeform_tags, 
			expected_chapters, chapter_count, is_complete, status, 
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`

	log.Printf("DEBUG: About to insert work with ChapterCount=%d, MaxChapters=%v", work.ChapterCount, work.MaxChapters)
	_, err = tx.Exec(query,
		work.ID, work.Title, work.Summary, work.Notes, work.UserID,
		work.Language, work.Rating, pq.Array(work.Warnings),
		pq.Array(work.Fandoms), pq.Array(work.Characters), pq.Array(work.Relationships),
		pq.Array(work.FreeformTags), work.MaxChapters, work.ChapterCount,
		work.IsComplete, work.Status, work.CreatedAt, work.UpdatedAt)
	log.Printf("DEBUG: Work insert result - error: %v", err)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create work", "details": err.Error()})
		return
	}

	// Get user's default pseud or create one if none exists
	var defaultPseudID uuid.UUID
	err = tx.QueryRow(`
		SELECT id FROM user_pseudonyms 
		WHERE user_id = $1 AND is_default = true 
		LIMIT 1`, userID).Scan(&defaultPseudID)

	if err != nil {
		// If no default pseud exists, get the username and create one
		var username string
		err = tx.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}

		// Create default pseud
		defaultPseudID = uuid.New()
		_, err = tx.Exec(`
			INSERT INTO user_pseudonyms (id, user_id, name, is_default, created_at)
			VALUES ($1, $2, $3, true, $4)`,
			defaultPseudID, userID, username, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create default pseud"})
			return
		}
	}

	// Note: Work authorship is established via user_id in works table
	// Creatorship table doesn't exist in current schema

	// Create first chapter
	chapterID := uuid.New()
	wordCount := countWords(req.ChapterContent)

	chapter := &models.Chapter{
		ID:        chapterID,
		WorkID:    workID,
		Number:    1,
		Title:     req.ChapterTitle,
		Summary:   req.ChapterSummary,
		Notes:     req.ChapterNotes,
		EndNotes:  req.ChapterEndNotes,
		Content:   req.ChapterContent,
		WordCount: wordCount,
		Status:    "published", // Make chapter published so it gets counted by trigger
		CreatedAt: now,
		UpdatedAt: now,
	}

	chapterQuery := `
		INSERT INTO chapters (id, work_id, chapter_number, title, summary, notes, end_notes, 
			content, word_count, is_draft, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err = tx.Exec(chapterQuery,
		chapter.ID, chapter.WorkID, chapter.Number, chapter.Title, chapter.Summary,
		chapter.Notes, chapter.EndNotes, chapter.Content, chapter.WordCount,
		chapter.Status == "draft", chapter.CreatedAt, chapter.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chapter", "details": err.Error()})
		return
	}

	// Update work word count
	_, err = tx.Exec("UPDATE works SET word_count = $1 WHERE id = $2", wordCount, workID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update word count"})
		return
	}

	// Work statistics are automatically initialized by the sync_work_statistics trigger

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	work.WordCount = wordCount

	// Index work in search service asynchronously
	go ws.indexWorkInSearch(workID, work)

	// Trigger notification for new work
	go func() {
		ctx := context.Background()
		// For new works, we might want to notify author subscribers
		// The triggerWorkNotification function handles work-specific subscriptions,
		// but we might also want author-level notifications here
		ws.triggerWorkNotification(ctx, workID, models.EventNewWork, work.Title, "New work has been published")
	}()

	c.JSON(http.StatusCreated, gin.H{"work": work, "first_chapter": chapter})
}

func (ws *WorkService) GetWork(c *gin.Context) {
	workIDParam := c.Param("work_id")
	var workID uuid.UUID
	var err error

	// Try to parse as UUID first (modern format)
	workID, err = uuid.Parse(workIDParam)
	if err != nil {
		// If UUID parsing fails, try parsing as integer (legacy ID)
		if legacyID, parseErr := strconv.Atoi(workIDParam); parseErr == nil {
			// Look up work by legacy_id and redirect to UUID route
			var workUUID uuid.UUID
			err = ws.db.QueryRow("SELECT id FROM works WHERE legacy_id = $1", legacyID).Scan(&workUUID)
			if err != nil {
				if err == sql.ErrNoRows {
					c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
				}
				return
			}

			// 301 permanent redirect to UUID-based URL
			newURL := fmt.Sprintf("/api/v1/work/%s", workUUID.String())
			c.Redirect(http.StatusMovedPermanently, newURL)
			return
		}

		// Neither UUID nor integer - invalid format
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID format"})
		return
	}

	// Query work details with all fields
	var work models.Work
	var legacyID sql.NullInt64
	var categoryStr, warningsStr sql.NullString
	var fandoms, characters, relationships, freeformTags pq.StringArray
	var summary sql.NullString
	var publishedAt sql.NullTime
	var status sql.NullString
	var maxChapters sql.NullInt64

	// Get user ID for privacy checks
	userID, hasUser := c.Get("user_id")
	var userUUID *uuid.UUID
	if hasUser {
		userIDStr := userID.(string)
		if userVal, err := uuid.Parse(userIDStr); err == nil {
			userUUID = &userVal
		}
	}

	// Check if user can view this work
	var canView bool
	err = ws.db.QueryRow("SELECT can_user_view_work($1, $2)", workID, userUUID).Scan(&canView)
	if err != nil || !canView {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view this work"})
		return
	}

	// Query work details with privacy controls
	query := `
		SELECT w.id, w.legacy_id, w.title, w.summary, w.language, w.rating,
			w.category, w.warnings, w.fandoms, w.characters, w.relationships, w.freeform_tags,
			w.word_count, w.chapter_count, w.max_chapters, w.is_complete, w.status,
			w.restricted, w.restricted_to_adults, w.comment_policy, w.moderate_comments, w.disable_comments,
			w.is_anonymous, w.in_anon_collection, w.in_unrevealed_collection,
			w.published_at, w.updated_at, w.created_at,
			COALESCE(w.hit_count, 0) as hits, COALESCE(w.kudos_count, 0) as kudos,
			COALESCE(w.comment_count, 0) as comments, COALESCE(w.bookmark_count, 0) as bookmarks
		FROM works w
		WHERE w.id = $1`

	// If no user, only allow non-draft, non-restricted works
	if !hasUser {
		query += " AND w.restricted = false"
	}

	// Execute query
	err = ws.db.QueryRow(query, workID).Scan(
		&work.ID, &legacyID, &work.Title, &summary,
		&work.Language, &work.Rating, &categoryStr, &warningsStr,
		&fandoms, &characters, &relationships, &freeformTags,
		&work.WordCount, &work.ChapterCount, &maxChapters,
		&work.IsComplete, &status, &work.RestrictedToUsers, &work.RestrictedToAdults,
		&work.CommentPolicy, &work.ModerateComments, &work.DisableComments,
		&work.IsAnonymous, &work.InAnonCollection, &work.InUnrevealedCollection,
		&publishedAt, &work.UpdatedAt, &work.CreatedAt,
		&work.Hits, &work.Kudos, &work.Comments, &work.Bookmarks,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	if err != nil {
		fmt.Printf("Database error in GetWork: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch work", "details": err.Error()})
		return
	}

	// Handle nullable fields exactly like SearchWorks
	if summary.Valid {
		work.Summary = summary.String
	}
	if publishedAt.Valid {
		work.PublishedAt = &publishedAt.Time
	}
	if maxChapters.Valid {
		maxChapInt := int(maxChapters.Int64)
		work.MaxChapters = &maxChapInt
	}
	if status.Valid {
		work.Status = status.String
	}
	if legacyID.Valid {
		legacyInt := int(legacyID.Int64)
		work.LegacyID = &legacyInt
	}

	// Convert string fields to arrays exactly like SearchWorks
	if categoryStr.Valid && categoryStr.String != "" {
		work.Category = []string{categoryStr.String}
	}
	if warningsStr.Valid && warningsStr.String != "" {
		work.Warnings = []string{warningsStr.String}
	}
	work.Fandoms = []string(fandoms)
	work.Characters = []string(characters)
	work.Relationships = []string(relationships)
	work.FreeformTags = []string(freeformTags)

	// Get work authors using the new co-authorship system
	authorsRows, err := ws.db.Query("SELECT * FROM get_work_authors($1, $2)", workID, userID)
	if err != nil {
		fmt.Printf("Failed to get work authors: %v\n", err)
	} else {
		defer authorsRows.Close()
		authors := []models.WorkAuthor{}
		for authorsRows.Next() {
			var author models.WorkAuthor
			err := authorsRows.Scan(&author.PseudID, &author.PseudName, &author.UserID, &author.Username, &author.IsAnonymous)
			if err != nil {
				fmt.Printf("Failed to scan author: %v\n", err)
				continue
			}
			authors = append(authors, author)
		}

		// Return work with authors
		c.JSON(http.StatusOK, gin.H{
			"work":    work,
			"authors": authors,
		})
		return
	}

	// Fallback to old response format if authors query fails
	c.JSON(http.StatusOK, work)

}

func (ws *WorkService) UpdateWork(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.UpdateWorkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Verify ownership using creatorship system
	var isAuthor bool
	err = ws.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM creatorships c
			JOIN pseuds p ON c.pseud_id = p.id
			WHERE c.creation_id = $1 AND c.creation_type = 'Work' 
			AND c.approved = true AND p.user_id = $2
		)`, workID, userID).Scan(&isAuthor)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	if !isAuthor {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to modify this work"})
		return
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, *req.Title)
		argIndex++
	}
	if req.Summary != nil {
		updates = append(updates, fmt.Sprintf("summary = $%d", argIndex))
		args = append(args, *req.Summary)
		argIndex++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argIndex))
		args = append(args, *req.Notes)
		argIndex++
	}
	if req.Rating != nil {
		updates = append(updates, fmt.Sprintf("rating = $%d", argIndex))
		args = append(args, *req.Rating)
		argIndex++
	}
	if req.Category != nil {
		updates = append(updates, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, pq.Array(req.Category))
		argIndex++
	}
	if req.Warnings != nil {
		updates = append(updates, fmt.Sprintf("warnings = $%d", argIndex))
		args = append(args, pq.Array(req.Warnings))
		argIndex++
	}
	if req.Fandoms != nil {
		updates = append(updates, fmt.Sprintf("fandoms = $%d", argIndex))
		args = append(args, pq.Array(req.Fandoms))
		argIndex++
	}
	if req.Characters != nil {
		updates = append(updates, fmt.Sprintf("characters = $%d", argIndex))
		args = append(args, pq.Array(req.Characters))
		argIndex++
	}
	if req.Relationships != nil {
		updates = append(updates, fmt.Sprintf("relationships = $%d", argIndex))
		args = append(args, pq.Array(req.Relationships))
		argIndex++
	}
	if req.FreeformTags != nil {
		updates = append(updates, fmt.Sprintf("freeform_tags = $%d", argIndex))
		args = append(args, pq.Array(req.FreeformTags))
		argIndex++
	}
	if req.MaxChapters != nil {
		updates = append(updates, fmt.Sprintf("max_chapters = $%d", argIndex))
		args = append(args, req.MaxChapters)
		argIndex++
	}
	if req.IsComplete != nil {
		updates = append(updates, fmt.Sprintf("is_complete = $%d", argIndex))
		args = append(args, *req.IsComplete)
		argIndex++

		// When marking a work as complete, automatically set max_chapters to current chapter_count
		if *req.IsComplete {
			var currentChapterCount int
			err := ws.db.QueryRow("SELECT chapter_count FROM works WHERE id = $1", workID).Scan(&currentChapterCount)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current chapter count", "details": err.Error()})
				return
			}

			// Set max_chapters to current chapter_count
			updates = append(updates, fmt.Sprintf("max_chapters = $%d", argIndex))
			args = append(args, currentChapterCount)
			argIndex++
		}
	}
	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *req.Status)

		// If publishing for first time, set published_at
		if *req.Status == "posted" {
			argIndex++
			updates = append(updates, fmt.Sprintf("published_at = $%d", argIndex))
			args = append(args, time.Now())
		}
		argIndex++
	}
	if req.RestrictedToUsers != nil {
		updates = append(updates, fmt.Sprintf("restricted = $%d", argIndex))
		args = append(args, *req.RestrictedToUsers)
		argIndex++
	}
	if req.RestrictedToAdults != nil {
		updates = append(updates, fmt.Sprintf("restricted_to_adults = $%d", argIndex))
		args = append(args, *req.RestrictedToAdults)
		argIndex++
	}
	if req.CommentPolicy != nil {
		updates = append(updates, fmt.Sprintf("comment_policy = $%d", argIndex))
		args = append(args, *req.CommentPolicy)
		argIndex++
	}
	if req.ModerateComments != nil {
		updates = append(updates, fmt.Sprintf("moderate_comments = $%d", argIndex))
		args = append(args, *req.ModerateComments)
		argIndex++
	}
	if req.DisableComments != nil {
		updates = append(updates, fmt.Sprintf("disable_comments = $%d", argIndex))
		args = append(args, *req.DisableComments)
		argIndex++
	}
	if req.IsAnonymous != nil {
		updates = append(updates, fmt.Sprintf("is_anonymous = $%d", argIndex))
		args = append(args, *req.IsAnonymous)
		argIndex++
	}
	if req.InAnonCollection != nil {
		updates = append(updates, fmt.Sprintf("in_anon_collection = $%d", argIndex))
		args = append(args, *req.InAnonCollection)
		argIndex++
	}
	if req.InUnrevealedCollection != nil {
		updates = append(updates, fmt.Sprintf("in_unrevealed_collection = $%d", argIndex))
		args = append(args, *req.InUnrevealedCollection)
		argIndex++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No updates provided"})
		return
	}

	// Add updated_at
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add work ID for WHERE clause
	args = append(args, workID)

	query := fmt.Sprintf("UPDATE works SET %s WHERE id = $%d", strings.Join(updates, ", "), argIndex)

	_, err = ws.db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update work", "details": err.Error()})
		return
	}

	// Clear cache
	cacheKey := fmt.Sprintf("work:%s", workID)
	ws.redis.Del(c.Request.Context(), cacheKey)

	// Fetch updated work
	work, err := ws.getWorkByID(workID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated work"})
		return
	}

	// Trigger notification for work update
	go func() {
		ctx := context.Background()
		ws.triggerWorkNotification(ctx, workID, models.EventWorkUpdated, work.Title, "Work has been updated")
	}()

	c.JSON(http.StatusOK, gin.H{"work": work})
}

func (ws *WorkService) DeleteWork(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Verify ownership using creatorship system
	var isAuthor bool
	err = ws.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM creatorships c
			JOIN pseuds p ON c.pseud_id = p.id
			WHERE c.creation_id = $1 AND c.creation_type = 'Work' 
			AND c.approved = true AND p.user_id = $2
		)`, workID, userID).Scan(&isAuthor)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	if !isAuthor {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this work"})
		return
	}

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Delete related data (cascading)
	tables := []string{"gifts", "creatorships", "work_comments", "work_kudos", "bookmarks", "work_statistics", "chapters", "works"}
	for _, table := range tables {
		var query string
		if table == "works" {
			query = fmt.Sprintf("DELETE FROM %s WHERE id = $1", table)
		} else if table == "creatorships" || table == "gifts" {
			if table == "creatorships" {
				query = fmt.Sprintf("DELETE FROM %s WHERE creation_id = $1 AND creation_type = 'Work'", table)
			} else {
				query = fmt.Sprintf("DELETE FROM %s WHERE work_id = $1", table)
			}
		} else {
			query = fmt.Sprintf("DELETE FROM %s WHERE work_id = $1", table)
		}

		_, err = tx.Exec(query, workID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete work data"})
			return
		}
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Clear cache
	cacheKey := fmt.Sprintf("work:%s", workID)
	ws.redis.Del(c.Request.Context(), cacheKey)

	c.JSON(http.StatusOK, gin.H{"message": "Work deleted successfully"})
}

// loadWorkTags loads tags for a work from the work_tags relationship table
func (ws *WorkService) loadWorkTags(workID string) (fandoms, characters, relationships, freeformTags []string) {
	log.Printf("=== LOADING TAGS FOR WORK %s ===", workID)
	query := `
		SELECT t.name, t.type 
		FROM tags t 
		JOIN work_tags wt ON t.id = wt.tag_id 
		WHERE wt.work_id = $1 
		ORDER BY t.type, t.name`

	rows, err := ws.db.Query(query, workID)
	if err != nil {
		log.Printf("Error loading tags for work %s: %v", workID, err)
		return nil, nil, nil, nil
	}
	defer rows.Close()

	tagCount := 0
	for rows.Next() {
		tagCount++
		var name, tagType string
		if err := rows.Scan(&name, &tagType); err != nil {
			log.Printf("Error scanning tag: %v", err)
			continue
		}

		log.Printf("DEBUG: Found tag %s (type: %s) for work %s", name, tagType, workID)
		switch tagType {
		case "fandom":
			fandoms = append(fandoms, name)
		case "character":
			characters = append(characters, name)
		case "relationship":
			relationships = append(relationships, name)
		case "freeform", "additional":
			freeformTags = append(freeformTags, name)
		case "warning":
			// Warnings are handled separately in the main work query
		}
	}

	log.Printf("DEBUG: Loaded %d tags for work %s. Fandoms: %v, Characters: %v, Relationships: %v, Freeform: %v",
		tagCount, workID, fandoms, characters, relationships, freeformTags)
	return fandoms, characters, relationships, freeformTags
}

// SearchTags provides enhanced tag search with partial matching
func (ws *WorkService) SearchTags(c *gin.Context) {
	query := c.DefaultQuery("q", "")
	tagType := c.Query("type") // Get optional type filter
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit > 100 {
		limit = 100
	}

	log.Printf("=== SEARCHING TAGS: q='%s', type='%s', limit=%d, offset=%d ===", query, tagType, limit, offset)

	// Build SQL query with case-insensitive partial matching and optional type filtering
	var sqlQuery string
	var queryArgs []interface{}
	searchPattern := "%" + query + "%"

	if tagType != "" {
		sqlQuery = `
			SELECT id, name, canonical_name, type, description, is_canonical, is_filterable, 
				   use_count, created_at, updated_at
			FROM tags 
			WHERE LOWER(name) LIKE LOWER($1) AND type = $2
			ORDER BY use_count DESC, name ASC
			LIMIT $3 OFFSET $4`
		queryArgs = []interface{}{searchPattern, tagType, limit, offset}
	} else {
		sqlQuery = `
			SELECT id, name, canonical_name, type, description, is_canonical, is_filterable, 
				   use_count, created_at, updated_at
			FROM tags 
			WHERE LOWER(name) LIKE LOWER($1)
			ORDER BY use_count DESC, name ASC
			LIMIT $2 OFFSET $3`
		queryArgs = []interface{}{searchPattern, limit, offset}
	}

	rows, err := ws.db.Query(sqlQuery, queryArgs...)
	if err != nil {
		log.Printf("ERROR: Tag search query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}
	defer rows.Close()

	tags := []map[string]interface{}{}
	for rows.Next() {
		var tag struct {
			ID            string    `json:"id"`
			Name          string    `json:"name"`
			CanonicalName *string   `json:"canonical_name"`
			Type          string    `json:"type"`
			Description   *string   `json:"description"`
			IsCanonical   bool      `json:"is_canonical"`
			IsFilterable  bool      `json:"is_filterable"`
			UseCount      int       `json:"use_count"`
			CreatedAt     time.Time `json:"created_at"`
			UpdatedAt     time.Time `json:"updated_at"`
		}

		err := rows.Scan(&tag.ID, &tag.Name, &tag.CanonicalName, &tag.Type, &tag.Description,
			&tag.IsCanonical, &tag.IsFilterable, &tag.UseCount, &tag.CreatedAt, &tag.UpdatedAt)
		if err != nil {
			log.Printf("ERROR: Scanning tag row: %v", err)
			continue
		}

		tagMap := map[string]interface{}{
			"id":             tag.ID,
			"name":           tag.Name,
			"canonical_name": tag.CanonicalName,
			"type":           tag.Type,
			"description":    tag.Description,
			"is_canonical":   tag.IsCanonical,
			"is_filterable":  tag.IsFilterable,
			"use_count":      tag.UseCount,
			"created_at":     tag.CreatedAt,
			"updated_at":     tag.UpdatedAt,
		}
		tags = append(tags, tagMap)
	}

	// Get total count with same filtering as main query
	var countQuery string
	var countArgs []interface{}

	if tagType != "" {
		countQuery = `SELECT COUNT(*) FROM tags WHERE LOWER(name) LIKE LOWER($1) AND type = $2`
		countArgs = []interface{}{searchPattern, tagType}
	} else {
		countQuery = `SELECT COUNT(*) FROM tags WHERE LOWER(name) LIKE LOWER($1)`
		countArgs = []interface{}{searchPattern}
	}

	var total int
	err = ws.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		total = len(tags) // Fallback
	}

	log.Printf("=== TAG SEARCH RESULTS: found %d tags (total %d) ===", len(tags), total)

	response := gin.H{
		"limit":  limit,
		"offset": offset,
		"total":  total,
	}

	if len(tags) > 0 {
		response["tags"] = tags
	} else {
		response["tags"] = nil
	}

	c.JSON(http.StatusOK, response)
}

func (ws *WorkService) SearchWorks(c *gin.Context) {
	log.Printf("=== SEARCHWORKS HANDLER CALLED! ===")
	// Parse query parameters
	query := c.DefaultQuery("q", "")
	fandoms := c.QueryArray("fandom")
	characters := c.QueryArray("character")
	relationships := c.QueryArray("relationship")
	tags := c.QueryArray("tag")
	rating := c.QueryArray("rating")
	category := c.QueryArray("category")
	warnings := c.QueryArray("warning")

	sortBy := c.DefaultQuery("sort", "updated_at")
	sortOrder := c.DefaultQuery("order", "desc")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Get user ID for privacy filtering
	_, hasUser := c.Get("user_id")

	// Build SQL query - only show published works, not drafts
	// Note: Remove the empty array columns, we'll load tags separately from work_tags table
	baseQuery := `
		SELECT w.id, w.title, w.summary, w.user_id, u.username, w.language, w.rating,
			w.category, w.archive_warning,
			w.word_count, w.chapter_count, w.expected_chapters, w.is_complete, 
			CASE WHEN w.is_draft THEN 'draft' WHEN w.is_complete THEN 'complete' ELSE 'in_progress' END as status,
			w.published_at, w.updated_at, w.created_at,
			COALESCE(w.hit_count, 0) as hits, COALESCE(w.kudos_count, 0) as kudos,
			COALESCE(w.comment_count, 0) as comments, COALESCE(w.bookmark_count, 0) as bookmarks
		FROM works w
		JOIN users u ON w.user_id = u.id
		WHERE w.is_draft = false AND w.published_at IS NOT NULL`

	args := []interface{}{}
	argIndex := 1

	// If no user is logged in, exclude user-restricted works
	if !hasUser {
		baseQuery += " AND w.restricted = false"
	}

	conditions := []string{}

	if query != "" {
		conditions = append(conditions, fmt.Sprintf("(w.title ILIKE $%d OR w.summary ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+query+"%")
		argIndex++
	}

	// Tag filtering using work_tags relationship table
	if len(fandoms) > 0 {
		tagPlaceholders := []string{}
		for _, fandom := range fandoms {
			tagPlaceholders = append(tagPlaceholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, fandom)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf(`w.id IN (
			SELECT DISTINCT wt.work_id FROM work_tags wt 
			JOIN tags t ON wt.tag_id = t.id 
			WHERE t.type = 'fandom' AND t.name IN (%s)
		)`, strings.Join(tagPlaceholders, ",")))
	}

	if len(characters) > 0 {
		tagPlaceholders := []string{}
		for _, character := range characters {
			tagPlaceholders = append(tagPlaceholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, character)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf(`w.id IN (
			SELECT DISTINCT wt.work_id FROM work_tags wt 
			JOIN tags t ON wt.tag_id = t.id 
			WHERE t.type = 'character' AND t.name IN (%s)
		)`, strings.Join(tagPlaceholders, ",")))
	}

	if len(relationships) > 0 {
		tagPlaceholders := []string{}
		for _, relationship := range relationships {
			tagPlaceholders = append(tagPlaceholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, relationship)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf(`w.id IN (
			SELECT DISTINCT wt.work_id FROM work_tags wt 
			JOIN tags t ON wt.tag_id = t.id 
			WHERE t.type = 'relationship' AND t.name IN (%s)
		)`, strings.Join(tagPlaceholders, ",")))
	}

	if len(tags) > 0 {
		tagPlaceholders := []string{}
		for _, tag := range tags {
			tagPlaceholders = append(tagPlaceholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, tag)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf(`w.id IN (
			SELECT DISTINCT wt.work_id FROM work_tags wt 
			JOIN tags t ON wt.tag_id = t.id 
			WHERE t.type IN ('freeform', 'additional') AND t.name IN (%s)
		)`, strings.Join(tagPlaceholders, ",")))
	}

	if len(rating) > 0 {
		placeholders := []string{}
		for _, r := range rating {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, r)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("w.rating IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(category) > 0 {
		placeholders := []string{}
		for _, c := range category {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, c)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("w.category IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(warnings) > 0 {
		placeholders := []string{}
		for _, w := range warnings {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, w)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("w.warnings IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ordering
	allowedSort := map[string]bool{
		"title": true, "updated_at": true, "created_at": true, "published_at": true,
		"word_count": true, "hits": true, "kudos": true, "comments": true, "bookmarks": true,
	}
	if !allowedSort[sortBy] {
		sortBy = "updated_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	baseQuery += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", sortBy, sortOrder, argIndex, argIndex+1)
	args = append(args, limit, offset)

	fmt.Printf("FINAL QUERY: %s\n", baseQuery)
	fmt.Printf("ARGS: %v\n", args)

	// Test simple query first
	var testCount int
	testErr := ws.db.QueryRow("SELECT COUNT(*) FROM works").Scan(&testCount)
	fmt.Printf("TEST QUERY: SELECT COUNT(*) FROM works = %d, err=%v\n", testCount, testErr)

	fmt.Printf("About to execute main query...\n")
	rows, err := ws.db.Query(baseQuery, args...)
	fmt.Printf("Query result: rows=%v, err=%v\n", rows != nil, err)
	if err != nil {
		fmt.Printf("Query failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search works", "details": err.Error()})
		return
	}
	defer rows.Close()
	fmt.Printf("Query executed successfully\n")

	works := []models.Work{}
	fmt.Printf("Starting to scan rows\n")
	for rows.Next() {
		fmt.Printf("Processing row\n")
		var work models.Work
		var categoryStr sql.NullString
		var warningsStr sql.NullString
		var summaryStr sql.NullString

		err := rows.Scan(
			&work.ID, &work.Title, &summaryStr, &work.UserID, &work.Username,
			&work.Language, &work.Rating, &categoryStr, &warningsStr,
			&work.WordCount, &work.ChapterCount, &work.MaxChapters,
			&work.IsComplete, &work.Status, &work.PublishedAt, &work.UpdatedAt, &work.CreatedAt,
			&work.Hits, &work.Kudos, &work.Comments, &work.Bookmarks)

		if err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			continue
		}

		// Convert to model fields
		if summaryStr.Valid {
			work.Summary = summaryStr.String
		}
		if categoryStr.Valid && categoryStr.String != "" {
			work.Category = []string{categoryStr.String}
		}
		if warningsStr.Valid && warningsStr.String != "" {
			work.Warnings = []string{warningsStr.String}
		}

		// Load tags from work_tags relationship table
		log.Printf("DEBUG: Work ID is: %s, type: %T", work.ID, work.ID)
		log.Printf("=== ABOUT TO LOAD TAGS FOR WORK %s ===", work.ID.String())
		work.Fandoms, work.Characters, work.Relationships, work.FreeformTags = ws.loadWorkTags(work.ID.String())
		log.Printf("=== FINISHED LOADING TAGS FOR WORK %s ===", work.ID.String())
		if err != nil {
			fmt.Printf("SCAN ERROR: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan work data", "details": err.Error()})
			return
		}
		works = append(works, work)
		fmt.Printf("Successfully scanned work: %s\n", work.Title)
	}
	fmt.Printf("Finished scanning. Found %d works\n", len(works))

	// Get total count
	countQuery := strings.Replace(baseQuery, "SELECT w.id, w.title, w.summary, w.user_id, u.username, w.language, w.rating, w.category, w.warnings, w.fandoms, w.characters, w.relationships, w.freeform_tags, w.word_count, w.chapter_count, w.max_chapters, w.is_complete, w.status, w.published_at, w.updated_at, w.created_at, COALESCE(ws.hits, 0) as hits, COALESCE(ws.kudos, 0) as kudos, COALESCE(ws.comments, 0) as comments, COALESCE(ws.bookmarks, 0) as bookmarks", "SELECT COUNT(*)", 1)
	countQuery = strings.Split(countQuery, "ORDER BY")[0] // Remove ORDER BY and LIMIT

	var total int
	err = ws.db.QueryRow(countQuery, args[:len(args)-2]...).Scan(&total) // Remove LIMIT and OFFSET args
	if err != nil {
		total = len(works) // Fallback
	}

	c.JSON(http.StatusOK, gin.H{
		"works": works,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + limit - 1) / limit,
		},
	})
}

// Helper functions

func (ws *WorkService) getWorkByID(workID uuid.UUID) (*models.Work, error) {
	query := `
		SELECT w.id, w.title, w.summary, w.notes, w.user_id, u.username,
			w.language, w.rating, w.category, w.warnings, w.fandoms, w.characters, 
			w.relationships, w.freeform_tags, w.word_count, w.chapter_count, w.max_chapters,
			w.is_complete, w.status, w.published_at, w.updated_at, w.created_at,
			COALESCE(ws.hits, 0) as hits, COALESCE(ws.kudos, 0) as kudos,
			COALESCE(ws.comments, 0) as comments, COALESCE(ws.bookmarks, 0) as bookmarks
		FROM works w
		JOIN users u ON w.user_id = u.id
		LEFT JOIN work_statistics ws ON w.id = ws.work_id
		WHERE w.id = $1`

	var work models.Work
	var categoryArray, warningsArray, fandomsArray, charactersArray, relationshipsArray, freeformArray pq.StringArray

	err := ws.db.QueryRow(query, workID).Scan(
		&work.ID, &work.Title, &work.Summary, &work.Notes, &work.UserID, &work.Username,
		&work.Language, &work.Rating, &categoryArray,
		&warningsArray, &fandomsArray, &charactersArray,
		&relationshipsArray, &freeformArray, &work.WordCount,
		&work.ChapterCount, &work.MaxChapters, &work.IsComplete, &work.Status,
		&work.PublishedAt, &work.UpdatedAt, &work.CreatedAt,
		&work.Hits, &work.Kudos, &work.Comments, &work.Bookmarks)

	if err != nil {
		return nil, fmt.Errorf("DEBUG: Query error in getWorkByID: %v", err)
	}

	// Convert arrays to slices
	work.Category = []string(categoryArray)
	work.Warnings = []string(warningsArray)
	work.Fandoms = []string(fandomsArray)
	work.Characters = []string(charactersArray)
	work.Relationships = []string(relationshipsArray)
	work.FreeformTags = []string(freeformArray)

	// Set SeriesID to nil since it's not stored in works table
	work.SeriesID = nil

	return &work, nil
}

func (ws *WorkService) incrementHits(workID uuid.UUID) {
	// Increment hit counter asynchronously
	go func() {
		_, err := ws.db.Exec(`
			INSERT INTO work_statistics (work_id, hits, kudos, comments, bookmarks, collections, updated_at)
			VALUES ($1, 1, 0, 0, 0, 0, NOW())
			ON CONFLICT (work_id)
			DO UPDATE SET hits = work_statistics.hits + 1, updated_at = NOW()`,
			workID)
		if err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to increment hits for work %s: %v\n", workID, err)
		}
	}()
}

func countWords(text string) int {
	// Simple word counting - would be more sophisticated in production
	fields := strings.Fields(strings.TrimSpace(text))
	return len(fields)
}

// Placeholder implementations for other handlers
// These would need full implementations in a real application

func (ws *WorkService) GetChapters(c *gin.Context) {
	log.Printf("GetChapters called for work_id: %s", c.Param("work_id"))

	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	// TODO: Re-enable permission check after debugging
	// Check if user can view this work
	// userID, hasUser := c.Get("user_id")
	// var userUUID *uuid.UUID
	// if hasUser {
	// 	userVal := userID.(uuid.UUID)
	// 	userUUID = &userVal
	// }

	// var canView bool
	// err = ws.db.QueryRow("SELECT can_user_view_work($1, $2)", workID, userUUID).Scan(&canView)
	// if err != nil {
	// 	log.Printf("Error checking work permissions for %s: %v", workID, err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Permission check failed"})
	// 	return
	// }
	// if !canView {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view this work"})
	// 	return
	// }

	log.Printf("About to query chapters for work %s", workID)
	rows, err := ws.db.Query(`
		SELECT id, work_id, chapter_number, 
			COALESCE(title, '') as title, 
			COALESCE(summary, '') as summary, 
			COALESCE(notes, '') as notes, 
			COALESCE(end_notes, '') as end_notes, 
			COALESCE(content, '') as content, 
			COALESCE(word_count, 0) as word_count, 
			CASE WHEN is_draft THEN 'draft' ELSE 'posted' END as status, 
			published_at, created_at, updated_at
		FROM chapters 
		WHERE work_id = $1 
		ORDER BY chapter_number`, workID)
	if err != nil {
		log.Printf("Failed to fetch chapters for work %s: %v", workID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chapters", "details": err.Error()})
		return
	}
	defer rows.Close()

	chapters := []models.Chapter{}
	for rows.Next() {
		var chapter models.Chapter
		var publishedAt sql.NullTime
		err := rows.Scan(
			&chapter.ID, &chapter.WorkID, &chapter.Number, &chapter.Title, &chapter.Summary,
			&chapter.Notes, &chapter.EndNotes, &chapter.Content, &chapter.WordCount,
			&chapter.Status, &publishedAt, &chapter.CreatedAt, &chapter.UpdatedAt)
		if err != nil {
			log.Printf("Failed to scan chapter for work %s: %v", workID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan chapter", "details": err.Error()})
			return
		}
		if publishedAt.Valid {
			chapter.PublishedAt = &publishedAt.Time
		}
		chapters = append(chapters, chapter)
	}

	c.JSON(http.StatusOK, gin.H{"chapters": chapters})
}

func (ws *WorkService) GetChapter(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	chapterNumber, err := strconv.Atoi(c.Param("chapter_number"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chapter number"})
		return
	}

	// Check if user can view this work
	userID, hasUser := c.Get("user_id")
	var userUUID *uuid.UUID
	if hasUser {
		userIDStr := userID.(string)
		if userVal, err := uuid.Parse(userIDStr); err == nil {
			userUUID = &userVal
		}
	}

	var canView bool
	err = ws.db.QueryRow("SELECT can_user_view_work($1, $2)", workID, userUUID).Scan(&canView)
	if err != nil || !canView {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view this work"})
		return
	}

	var chapter models.Chapter
	var publishedAt sql.NullTime

	err = ws.db.QueryRow(`
		SELECT id, work_id, chapter_number, title, summary, notes, end_notes, 
			content, word_count, CASE WHEN is_draft THEN 'draft' ELSE 'posted' END as status, 
			published_at, created_at, updated_at
		FROM chapters 
		WHERE work_id = $1 AND chapter_number = $2`, workID, chapterNumber).Scan(
		&chapter.ID, &chapter.WorkID, &chapter.Number, &chapter.Title, &chapter.Summary,
		&chapter.Notes, &chapter.EndNotes, &chapter.Content, &chapter.WordCount,
		&chapter.Status, &publishedAt, &chapter.CreatedAt, &chapter.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chapter not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chapter"})
		return
	}

	if publishedAt.Valid {
		chapter.PublishedAt = &publishedAt.Time
	}

	// Increment work hit count when chapter is viewed
	ws.incrementHits(workID)

	c.JSON(http.StatusOK, gin.H{"chapter": chapter})
}

func (ws *WorkService) CreateChapter(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Title    string `json:"title"`
		Summary  string `json:"summary"`
		Notes    string `json:"notes"`
		EndNotes string `json:"end_notes"`
		Content  string `json:"content" validate:"required"`
		Status   string `json:"status" validate:"oneof=draft posted"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Verify ownership using creatorship system
	var isAuthor bool
	err = ws.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM creatorships cr
			JOIN pseuds p ON cr.pseud_id = p.id
			WHERE cr.creation_id = $1 AND cr.creation_type = 'Work' 
			AND cr.approved = true AND p.user_id = $2
		)`, workID, userID).Scan(&isAuthor)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	if !isAuthor {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to add chapters to this work"})
		return
	}

	// Get next chapter number
	var nextNumber int
	err = ws.db.QueryRow("SELECT COALESCE(MAX(chapter_number), 0) + 1 FROM chapters WHERE work_id = $1", workID).Scan(&nextNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get next chapter number"})
		return
	}

	// Create chapter
	chapterID := uuid.New()
	now := time.Now()
	wordCount := countWords(req.Content)

	chapter := &models.Chapter{
		ID:        chapterID,
		WorkID:    workID,
		Number:    nextNumber,
		Title:     req.Title,
		Summary:   req.Summary,
		Notes:     req.Notes,
		EndNotes:  req.EndNotes,
		Content:   req.Content,
		WordCount: wordCount,
		Status:    req.Status,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if req.Status == "posted" {
		chapter.PublishedAt = &now
	}

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO chapters (id, work_id, chapter_number, title, summary, notes, end_notes, 
			content, word_count, is_draft, published_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		chapter.ID, chapter.WorkID, chapter.Number, chapter.Title, chapter.Summary,
		chapter.Notes, chapter.EndNotes, chapter.Content, chapter.WordCount,
		chapter.Status == "draft", chapter.PublishedAt, chapter.CreatedAt, chapter.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chapter", "details": err.Error()})
		return
	}

	// Update work statistics
	_, err = tx.Exec(`
		UPDATE works SET 
			chapter_count = (SELECT COUNT(*) FROM chapters WHERE work_id = $1),
			word_count = (SELECT COALESCE(SUM(word_count), 0) FROM chapters WHERE work_id = $1),
			updated_at = $2
		WHERE id = $1`, workID, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update work statistics"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"chapter": chapter})
}

func (ws *WorkService) UpdateChapter(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	chapterID, err := uuid.Parse(c.Param("chapter_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chapter ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Verify ownership using creatorship system
	var isAuthor bool
	err = ws.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM creatorships c
			JOIN pseuds p ON c.pseud_id = p.id
			WHERE c.creation_id = $1 AND c.creation_type = 'Work' 
			AND c.approved = true AND p.user_id = $2
		)`, workID, userID).Scan(&isAuthor)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	if !isAuthor {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to modify this chapter"})
		return
	}

	var req models.UpdateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Verify chapter belongs to this work
	var existingChapter models.Chapter
	err = ws.db.QueryRow(`
		SELECT id, work_id, chapter_number, title, summary, notes, end_notes, 
			content, word_count, CASE WHEN is_draft THEN 'draft' ELSE 'posted' END as status
		FROM chapters 
		WHERE id = $1 AND work_id = $2`, chapterID, workID).Scan(
		&existingChapter.ID, &existingChapter.WorkID, &existingChapter.Number,
		&existingChapter.Title, &existingChapter.Summary, &existingChapter.Notes,
		&existingChapter.EndNotes, &existingChapter.Content, &existingChapter.WordCount,
		&existingChapter.Status)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chapter not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chapter"})
		return
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, *req.Title)
		argIndex++
	}
	if req.Summary != nil {
		updates = append(updates, fmt.Sprintf("summary = $%d", argIndex))
		args = append(args, *req.Summary)
		argIndex++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argIndex))
		args = append(args, *req.Notes)
		argIndex++
	}
	if req.EndNotes != nil {
		updates = append(updates, fmt.Sprintf("end_notes = $%d", argIndex))
		args = append(args, *req.EndNotes)
		argIndex++
	}
	if req.Content != nil {
		updates = append(updates, fmt.Sprintf("content = $%d", argIndex))
		args = append(args, *req.Content)
		argIndex++
		// Recalculate word count when content changes
		wordCount := countWords(*req.Content)
		updates = append(updates, fmt.Sprintf("word_count = $%d", argIndex))
		args = append(args, wordCount)
		argIndex++
	}
	if req.Status != nil {
		isDraft := *req.Status == "draft"
		updates = append(updates, fmt.Sprintf("is_draft = $%d", argIndex))
		args = append(args, isDraft)
		argIndex++

		// Set published_at when status changes to posted
		if *req.Status == "posted" && existingChapter.Status == "draft" {
			updates = append(updates, fmt.Sprintf("published_at = $%d", argIndex))
			args = append(args, time.Now())
			argIndex++
		}
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Always update the updated_at timestamp
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add WHERE clause parameters
	args = append(args, chapterID, workID)

	query := fmt.Sprintf(`
		UPDATE chapters 
		SET %s 
		WHERE id = $%d AND work_id = $%d`,
		strings.Join(updates, ", "), argIndex, argIndex+1)

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chapter"})
		return
	}

	// Update work's updated_at timestamp and word count if content changed
	if req.Content != nil {
		// Recalculate total work word count
		var totalWordCount int
		err = tx.QueryRow(`
			SELECT COALESCE(SUM(word_count), 0) 
			FROM chapters 
			WHERE work_id = $1 AND is_draft = false`, workID).Scan(&totalWordCount)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate word count"})
			return
		}

		_, err = tx.Exec(`
			UPDATE works 
			SET word_count = $1, updated_at = $2 
			WHERE id = $3`, totalWordCount, time.Now(), workID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update work word count"})
			return
		}
	} else {
		// Just update work's updated_at timestamp
		_, err = tx.Exec(`
			UPDATE works 
			SET updated_at = $1 
			WHERE id = $2`, time.Now(), workID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update work timestamp"})
			return
		}
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Clear cache
	cacheKey := fmt.Sprintf("work:%s", workID)
	ws.redis.Del(c.Request.Context(), cacheKey)
	chapterCacheKey := fmt.Sprintf("chapter:%s", chapterID)
	ws.redis.Del(c.Request.Context(), chapterCacheKey)

	// Trigger notification for chapter update
	go func() {
		ctx := context.Background()
		// Get work title for notification
		var workTitle string
		err := ws.db.QueryRow("SELECT title FROM works WHERE id = $1", workID).Scan(&workTitle)
		if err != nil {
			log.Printf("Failed to get work title for notification: %v", err)
			workTitle = "Unknown Work"
		}
		ws.triggerWorkNotification(ctx, workID, models.EventWorkUpdated, workTitle, "New chapter has been posted")
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Chapter updated successfully"})
}

func (ws *WorkService) DeleteChapter(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	chapterID, err := uuid.Parse(c.Param("chapter_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chapter ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Verify ownership using creatorship system
	var isAuthor bool
	err = ws.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM creatorships c
			JOIN pseuds p ON c.pseud_id = p.id
			WHERE c.creation_id = $1 AND c.creation_type = 'Work' 
			AND c.approved = true AND p.user_id = $2
		)`, workID, userID).Scan(&isAuthor)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	if !isAuthor {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete chapters from this work"})
		return
	}

	// Verify chapter belongs to this work and get chapter info
	var chapter models.Chapter
	err = ws.db.QueryRow(`
		SELECT id, work_id, chapter_number, word_count
		FROM chapters 
		WHERE id = $1 AND work_id = $2`, chapterID, workID).Scan(
		&chapter.ID, &chapter.WorkID, &chapter.Number, &chapter.WordCount)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chapter not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chapter"})
		return
	}

	// Check if this is the only chapter - cannot delete the last chapter
	var chapterCount int
	err = ws.db.QueryRow("SELECT COUNT(*) FROM chapters WHERE work_id = $1", workID).Scan(&chapterCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count chapters"})
		return
	}

	if chapterCount <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete the only chapter of a work. Delete the work instead."})
		return
	}

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Delete the chapter
	_, err = tx.Exec("DELETE FROM chapters WHERE id = $1 AND work_id = $2", chapterID, workID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chapter"})
		return
	}

	// Renumber remaining chapters to close gaps
	_, err = tx.Exec(`
		UPDATE chapters 
		SET chapter_number = new_numbers.new_number, updated_at = $3
		FROM (
			SELECT id, ROW_NUMBER() OVER (ORDER BY chapter_number) as new_number 
			FROM chapters 
			WHERE work_id = $1
		) as new_numbers
		WHERE chapters.id = new_numbers.id AND chapters.work_id = $1`, workID, workID, time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to renumber chapters"})
		return
	}

	// Update work statistics
	var newChapterCount int
	var newWordCount int
	err = tx.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(word_count), 0) 
		FROM chapters 
		WHERE work_id = $1`, workID).Scan(&newChapterCount, &newWordCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate new work statistics"})
		return
	}

	// Update work with new counts
	_, err = tx.Exec(`
		UPDATE works 
		SET chapter_count = $1, word_count = $2, updated_at = $3
		WHERE id = $4`, newChapterCount, newWordCount, time.Now(), workID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update work statistics"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Clear cache
	cacheKey := fmt.Sprintf("work:%s", workID)
	ws.redis.Del(c.Request.Context(), cacheKey)
	chapterCacheKey := fmt.Sprintf("chapter:%s", chapterID)
	ws.redis.Del(c.Request.Context(), chapterCacheKey)

	c.JSON(http.StatusOK, gin.H{
		"message":                "Chapter deleted successfully",
		"deleted_chapter_number": chapter.Number,
		"new_chapter_count":      newChapterCount,
		"new_word_count":         newWordCount,
	})
}

func (ws *WorkService) GetComments(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	// Get user ID for moderation checks
	userID, hasUser := c.Get("user_id")
	var userUUID *uuid.UUID
	if hasUser {
		userIDStr := userID.(string)
		if userVal, err := uuid.Parse(userIDStr); err == nil {
			userUUID = &userVal
		}
	}

	// Check if user can view this work
	var canView bool
	err = ws.db.QueryRow("SELECT can_user_view_work($1, $2)", workID, userUUID).Scan(&canView)
	if err != nil || !canView {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view this work"})
		return
	}

	// Get work owner for moderation check
	var authorID uuid.UUID
	err = ws.db.QueryRow("SELECT user_id FROM works WHERE id = $1", workID).Scan(&authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get work info"})
		return
	}

	isAuthor := userUUID != nil && *userUUID == authorID

	// Build query - show different comments based on user role
	baseQuery := `
		SELECT c.id, c.work_id, c.chapter_id, c.user_id, c.parent_comment_id, c.content,
			c.status, c.is_anonymous, c.created_at, c.updated_at,
			COALESCE(u.username, 'Anonymous') as username
		FROM comments c
		LEFT JOIN users u ON c.user_id = u.id AND c.is_anonymous = false
		WHERE c.work_id = $1`

	// Authors can see all comments, others only see published ones
	if !isAuthor {
		baseQuery += " AND c.status = 'published'"
	}

	baseQuery += " ORDER BY c.created_at ASC"

	rows, err := ws.db.Query(baseQuery, workID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
		return
	}
	defer rows.Close()

	comments := []models.WorkComment{}
	for rows.Next() {
		var comment models.WorkComment
		err := rows.Scan(
			&comment.ID, &comment.WorkID, &comment.ChapterID, &comment.UserID, &comment.ParentID,
			&comment.Content, &comment.Status, &comment.IsAnonymous, &comment.CreatedAt, &comment.UpdatedAt,
			&comment.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan comment"})
			return
		}
		comments = append(comments, comment)
	}

	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

func (ws *WorkService) GetKudos(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	// Get user ID to check if current user has given kudos
	userID, hasUser := c.Get("user_id")
	var userUUID *uuid.UUID
	if hasUser {
		userIDStr := userID.(string)
		if userVal, err := uuid.Parse(userIDStr); err == nil {
			userUUID = &userVal
		}
	}

	// Get client IP for guest kudos checking
	clientIP := c.ClientIP()

	// First, get total kudos count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM kudos WHERE work_id = $1`
	err = ws.db.QueryRow(countQuery, workID).Scan(&totalCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch kudos count"})
		return
	}

	// Check if current user has given kudos
	hasGivenKudos := false
	if userUUID != nil {
		var exists bool
		userKudosQuery := `SELECT EXISTS(SELECT 1 FROM kudos WHERE work_id = $1 AND user_id = $2)`
		err = ws.db.QueryRow(userKudosQuery, workID, *userUUID).Scan(&exists)
		if err == nil {
			hasGivenKudos = exists
		}
	} else {
		// Check for guest kudos by IP
		var exists bool
		guestKudosQuery := `SELECT EXISTS(SELECT 1 FROM kudos WHERE work_id = $1 AND ip_address = $2 AND user_id IS NULL)`
		err = ws.db.QueryRow(guestKudosQuery, workID, clientIP).Scan(&exists)
		if err == nil {
			hasGivenKudos = exists
		}
	}

	// Get recent kudos for display (limit to 20 most recent)
	query := `
		SELECT k.id, k.created_at, COALESCE(u.username, 'Guest') as username
		FROM kudos k
		LEFT JOIN users u ON k.user_id = u.id
		WHERE k.work_id = $1
		ORDER BY k.created_at DESC
		LIMIT 20
	`

	rows, err := ws.db.Query(query, workID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch kudos list"})
		return
	}
	defer rows.Close()

	var kudosList []map[string]interface{}
	for rows.Next() {
		var kudosID uuid.UUID
		var createdAt time.Time
		var username string

		err := rows.Scan(&kudosID, &createdAt, &username)
		if err != nil {
			continue
		}

		kudosItem := map[string]interface{}{
			"id":         kudosID,
			"username":   username,
			"created_at": createdAt,
		}

		kudosList = append(kudosList, kudosItem)
	}

	c.JSON(http.StatusOK, gin.H{
		"kudos":           kudosList,
		"has_given_kudos": hasGivenKudos,
		"total_count":     totalCount,
	})
}

func (ws *WorkService) GiveKudos(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	// Get user ID (could be nil for anonymous kudos)
	userID, hasUser := c.Get("user_id")
	var userUUID *uuid.UUID
	if hasUser {
		userIDStr := userID.(string)
		if userVal, err := uuid.Parse(userIDStr); err == nil {
			userUUID = &userVal
		}
	}

	// Check if user can view this work
	var canView bool
	err = ws.db.QueryRow("SELECT can_user_view_work($1, $2)", workID, userUUID).Scan(&canView)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check work permissions"})
		}
		return
	}
	if !canView {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view this work"})
		return
	}

	// Check if work allows kudos from this user
	var allowKudos bool
	if userUUID != nil {
		// Logged in user - check if they've already given kudos
		err = ws.db.QueryRow(`
			SELECT NOT EXISTS(
				SELECT 1 FROM kudos 
				WHERE work_id = $1 AND user_id = $2
			)`, workID, *userUUID).Scan(&allowKudos)
	} else {
		// Anonymous user - check IP address
		clientIP := c.ClientIP()
		err = ws.db.QueryRow(`
			SELECT NOT EXISTS(
				SELECT 1 FROM kudos 
				WHERE work_id = $1 AND ip_address = $2 AND user_id IS NULL
			)`, workID, clientIP).Scan(&allowKudos)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check kudos eligibility"})
		return
	}

	if !allowKudos {
		c.JSON(http.StatusConflict, gin.H{"error": "You have already given kudos to this work"})
		return
	}

	// Check if user is the author (can't give kudos to own work)
	if userUUID != nil {
		var isAuthor bool
		err = ws.db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM creatorships cr
				JOIN pseuds p ON cr.pseud_id = p.id
				WHERE cr.creation_id = $1 AND cr.creation_type = 'Work' 
				AND cr.approved = true AND p.user_id = $2
			)`, workID, *userUUID).Scan(&isAuthor)

		if err == nil && isAuthor {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot give kudos to your own work"})
			return
		}
	}

	// Give kudos
	kudosID := uuid.New()
	now := time.Now()
	clientIP := c.ClientIP()

	_, err = ws.db.Exec(`
		INSERT INTO kudos (id, work_id, user_id, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		kudosID, workID, userUUID, clientIP, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to give kudos"})
		return
	}

	// Update work kudos count
	_, err = ws.db.Exec(`
		UPDATE works SET 
			kudos_count = (SELECT COUNT(*) FROM kudos WHERE work_id = $1),
			updated_at = $2
		WHERE id = $1`, workID, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update kudos count"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Kudos given successfully"})
}

func (ws *WorkService) RemoveKudos(c *gin.Context) {
	// TODO: Implement kudos removal
	c.JSON(http.StatusOK, gin.H{"message": "Kudos removed"})
}

func (ws *WorkService) GetStats(c *gin.Context) {
	workIDParam := c.Param("work_id")
	var workID uuid.UUID
	var err error

	// Try to parse as UUID first (modern format)
	workID, err = uuid.Parse(workIDParam)
	if err != nil {
		// If UUID parsing fails, try parsing as integer (legacy ID)
		if legacyID, parseErr := strconv.Atoi(workIDParam); parseErr == nil {
			// Look up work by legacy_id
			err = ws.db.QueryRow("SELECT id FROM works WHERE legacy_id = $1", legacyID).Scan(&workID)
			if err != nil {
				if err == sql.ErrNoRows {
					c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
				}
				return
			}
		} else {
			// Neither UUID nor integer - invalid format
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID format"})
			return
		}
	}

	// Get user ID for privacy checks
	userID, hasUser := c.Get("user_id")
	var userUUID *uuid.UUID
	if hasUser {
		userIDStr := userID.(string)
		if userVal, err := uuid.Parse(userIDStr); err == nil {
			userUUID = &userVal
		}
	}

	// Check if user can view this work
	var canView bool
	err = ws.db.QueryRow("SELECT can_user_view_work($1, $2)", workID, userUUID).Scan(&canView)
	if err != nil || !canView {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view this work"})
		return
	}

	// Get comprehensive work statistics
	var stats struct {
		WorkID      uuid.UUID  `json:"work_id"`
		Title       string     `json:"title"`
		PublishedAt *time.Time `json:"published_at"`
		UpdatedAt   time.Time  `json:"updated_at"`

		// Basic counts
		WordCount    int  `json:"word_count"`
		ChapterCount int  `json:"chapter_count"`
		MaxChapters  *int `json:"max_chapters"`
		IsComplete   bool `json:"is_complete"`

		// Engagement statistics
		Hits          int `json:"hits"`
		Kudos         int `json:"kudos"`
		Comments      int `json:"comments"`
		Bookmarks     int `json:"bookmarks"`
		Subscriptions int `json:"subscriptions"`
		Collections   int `json:"collections"`

		// Time-based statistics
		DailyHits []struct {
			Date  string `json:"date"`
			Count int    `json:"count"`
		} `json:"daily_hits,omitempty"`

		MonthlyHits []struct {
			Month string `json:"month"`
			Count int    `json:"count"`
		} `json:"monthly_hits,omitempty"`
	}

	// Get basic work info and current statistics
	var publishedAt sql.NullTime
	var maxChapters sql.NullInt64

	err = ws.db.QueryRow(`
		SELECT w.id, w.title, w.published_at, w.updated_at, w.word_count, w.chapter_count, 
			w.max_chapters, w.is_complete,
			COALESCE(w.hit_count, 0) as hits, COALESCE(w.kudos_count, 0) as kudos,
			COALESCE(w.comment_count, 0) as comments, COALESCE(w.bookmark_count, 0) as bookmarks
		FROM works w 
		WHERE w.id = $1`, workID).Scan(
		&stats.WorkID, &stats.Title, &publishedAt, &stats.UpdatedAt,
		&stats.WordCount, &stats.ChapterCount, &maxChapters, &stats.IsComplete,
		&stats.Hits, &stats.Kudos, &stats.Comments, &stats.Bookmarks)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch work statistics"})
		return
	}

	// Handle nullable fields
	if publishedAt.Valid {
		stats.PublishedAt = &publishedAt.Time
	}
	if maxChapters.Valid {
		maxChapInt := int(maxChapters.Int64)
		stats.MaxChapters = &maxChapInt
	}

	// Get subscription count
	err = ws.db.QueryRow(`
		SELECT COUNT(*) FROM subscriptions 
		WHERE subscribable_type = 'Work' AND subscribable_id = $1`, workID).Scan(&stats.Subscriptions)
	if err != nil {
		stats.Subscriptions = 0 // Default to 0 if query fails
	}

	// Get collection count
	err = ws.db.QueryRow(`
		SELECT COUNT(*) FROM collection_items 
		WHERE item_type = 'Work' AND item_id = $1`, workID).Scan(&stats.Collections)
	if err != nil {
		stats.Collections = 0 // Default to 0 if query fails
	}

	// Check if this is the work owner - only show detailed analytics to owners
	var isOwner bool
	if userUUID != nil {
		err = ws.db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM creatorships cr
				JOIN pseuds p ON cr.pseud_id = p.id
				WHERE cr.creation_id = $1 AND cr.creation_type = 'Work' 
				AND cr.approved = true AND p.user_id = $2
			)`, workID, *userUUID).Scan(&isOwner)

		if err != nil {
			isOwner = false
		}
	}

	// If user is the owner, provide detailed analytics
	if isOwner {
		// Get daily hits for the last 30 days
		rows, err := ws.db.Query(`
			SELECT DATE(hit_date) as date, COUNT(*) as count
			FROM work_hits 
			WHERE work_id = $1 AND hit_date >= CURRENT_DATE - INTERVAL '30 days'
			GROUP BY DATE(hit_date)
			ORDER BY date DESC`, workID)

		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var dailyHit struct {
					Date  string `json:"date"`
					Count int    `json:"count"`
				}
				err := rows.Scan(&dailyHit.Date, &dailyHit.Count)
				if err == nil {
					stats.DailyHits = append(stats.DailyHits, dailyHit)
				}
			}
		}

		// Get monthly hits for the last 12 months
		monthlyRows, err := ws.db.Query(`
			SELECT TO_CHAR(hit_date, 'YYYY-MM') as month, COUNT(*) as count
			FROM work_hits 
			WHERE work_id = $1 AND hit_date >= CURRENT_DATE - INTERVAL '12 months'
			GROUP BY TO_CHAR(hit_date, 'YYYY-MM')
			ORDER BY month DESC`, workID)

		if err == nil {
			defer monthlyRows.Close()
			for monthlyRows.Next() {
				var monthlyHit struct {
					Month string `json:"month"`
					Count int    `json:"count"`
				}
				err := monthlyRows.Scan(&monthlyHit.Month, &monthlyHit.Count)
				if err == nil {
					stats.MonthlyHits = append(stats.MonthlyHits, monthlyHit)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (ws *WorkService) SearchSeries(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		// Return recent series if no search query
		query = "%"
	} else {
		query = "%" + query + "%"
	}

	// Build pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Search series by title and summary
	searchQuery := `
		SELECT s.id, s.title, s.description, s.notes, s.user_id, s.is_complete,
			s.work_count, s.created_at, s.updated_at, u.username
		FROM series s
		JOIN users u ON s.user_id = u.id
		WHERE (s.title ILIKE $1 OR s.description ILIKE $1)
		ORDER BY s.updated_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := ws.db.Query(searchQuery, query, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search series"})
		return
	}
	defer rows.Close()

	var series []models.Series
	for rows.Next() {
		var s models.Series
		err := rows.Scan(
			&s.ID, &s.Title, &s.Summary, &s.Notes, &s.UserID, &s.IsComplete,
			&s.WorkCount, &s.CreatedAt, &s.UpdatedAt, &s.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan series"})
			return
		}

		// Calculate word count for each series
		ws.db.QueryRow(`
			SELECT COALESCE(SUM(w.word_count), 0) 
			FROM works w 
			JOIN series_works sw ON w.id = sw.work_id
			WHERE sw.series_id = $1 AND w.status != 'draft'`, s.ID).Scan(&s.WordCount)

		series = append(series, s)
	}

	// Get total count for pagination
	var total int
	countQuery := `
		SELECT COUNT(*) 
		FROM series s
		WHERE (s.title ILIKE $1 OR s.description ILIKE $1)`
	err = ws.db.QueryRow(countQuery, query).Scan(&total)
	if err != nil {
		total = len(series) // Fallback
	}

	c.JSON(http.StatusOK, gin.H{
		"series": series,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

func (ws *WorkService) GetSeries(c *gin.Context) {
	seriesID, err := uuid.Parse(c.Param("series_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid series ID"})
		return
	}

	var series models.Series
	err = ws.db.QueryRow(`
		SELECT s.id, s.title, s.summary, s.notes, s.user_id, s.is_complete, 
			s.work_count, s.created_at, s.updated_at, u.username
		FROM series s
		JOIN users u ON s.user_id = u.id
		WHERE s.id = $1`, seriesID).Scan(
		&series.ID, &series.Title, &series.Summary, &series.Notes, &series.UserID,
		&series.IsComplete, &series.WorkCount, &series.CreatedAt, &series.UpdatedAt,
		&series.Username)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch series"})
		return
	}

	// Calculate total word count from works
	err = ws.db.QueryRow(`
		SELECT COALESCE(SUM(w.word_count), 0)
		FROM works w
		JOIN series_works sw ON w.id = sw.work_id
		WHERE sw.series_id = $1 AND w.status != 'draft'`, seriesID).Scan(&series.WordCount)

	if err != nil {
		series.WordCount = 0 // Fallback
	}

	c.JSON(http.StatusOK, gin.H{"series": series})
}

func (ws *WorkService) GetSeriesWorks(c *gin.Context) {
	seriesID, err := uuid.Parse(c.Param("series_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid series ID"})
		return
	}

	// Get user ID for privacy checks
	userID, hasUser := c.Get("user_id")
	var userUUID *uuid.UUID
	if hasUser {
		userIDStr := userID.(string)
		if userVal, err := uuid.Parse(userIDStr); err == nil {
			userUUID = &userVal
		}
	}

	baseQuery := `
		SELECT w.id, w.title, w.summary, w.language, w.rating,
			w.category, w.warnings, w.fandoms, w.characters, w.relationships, w.freeform_tags,
			w.word_count, w.chapter_count, w.max_chapters, w.is_complete, w.status,
			w.published_at, w.updated_at, w.created_at,
			COALESCE(w.hit_count, 0) as hits, COALESCE(w.kudos_count, 0) as kudos,
			COALESCE(w.comment_count, 0) as comments, COALESCE(w.bookmark_count, 0) as bookmarks,
			sw.position
		FROM works w
		JOIN series_works sw ON w.id = sw.work_id
		WHERE sw.series_id = $1`

	// If no user, only show non-draft, non-restricted works
	if !hasUser {
		baseQuery += " AND w.status != 'draft' AND w.restricted = false"
	}

	baseQuery += " ORDER BY sw.position"

	rows, err := ws.db.Query(baseQuery, seriesID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch series works"})
		return
	}
	defer rows.Close()

	works := []interface{}{}
	for rows.Next() {
		var work models.Work
		var categoryStr, warningsStr sql.NullString
		var fandoms, characters, relationships, freeformTags pq.StringArray
		var summary sql.NullString
		var publishedAt sql.NullTime
		var status sql.NullString
		var maxChapters sql.NullInt64
		var position int

		err := rows.Scan(
			&work.ID, &work.Title, &summary,
			&work.Language, &work.Rating, &categoryStr, &warningsStr,
			&fandoms, &characters, &relationships, &freeformTags,
			&work.WordCount, &work.ChapterCount, &maxChapters,
			&work.IsComplete, &status, &publishedAt, &work.UpdatedAt, &work.CreatedAt,
			&work.Hits, &work.Kudos, &work.Comments, &work.Bookmarks,
			&position)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan work"})
			return
		}

		// Handle nullable fields
		if summary.Valid {
			work.Summary = summary.String
		}
		if publishedAt.Valid {
			work.PublishedAt = &publishedAt.Time
		}
		if maxChapters.Valid {
			maxChapInt := int(maxChapters.Int64)
			work.MaxChapters = &maxChapInt
		}
		if status.Valid {
			work.Status = status.String
		}

		// Convert string fields to arrays
		if categoryStr.Valid && categoryStr.String != "" {
			work.Category = []string{categoryStr.String}
		}
		if warningsStr.Valid && warningsStr.String != "" {
			work.Warnings = []string{warningsStr.String}
		}
		work.Fandoms = []string(fandoms)
		work.Characters = []string(characters)
		work.Relationships = []string(relationships)
		work.FreeformTags = []string(freeformTags)

		// Check if user can view this specific work
		if userUUID != nil {
			var canView bool
			err = ws.db.QueryRow("SELECT can_user_view_work($1, $2)", work.ID, *userUUID).Scan(&canView)
			if err != nil || !canView {
				continue // Skip this work
			}
		}

		workData := gin.H{
			"work":     work,
			"position": position,
		}

		works = append(works, workData)
	}

	c.JSON(http.StatusOK, gin.H{"works": works})
}

func (ws *WorkService) CreateSeries(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Title      string   `json:"title" validate:"required,min=1,max=500"`
		Summary    string   `json:"summary"`
		Notes      string   `json:"notes"`
		IsComplete bool     `json:"is_complete"`
		WorkIDs    []string `json:"work_ids"` // Works to add to series
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	userIDStr := userID.(string)
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Verify user owns all specified works
	if len(req.WorkIDs) > 0 {
		for _, workIDStr := range req.WorkIDs {
			workID, err := uuid.Parse(workIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID format"})
				return
			}

			var isAuthor bool
			err = ws.db.QueryRow(`
				SELECT EXISTS(
					SELECT 1 FROM creatorships cr
					JOIN pseuds p ON cr.pseud_id = p.id
					WHERE cr.creation_id = $1 AND cr.creation_type = 'Work' 
					AND cr.approved = true AND p.user_id = $2
				)`, workID, userUUID).Scan(&isAuthor)

			if err != nil || !isAuthor {
				c.JSON(http.StatusForbidden, gin.H{"error": "You can only add your own works to a series"})
				return
			}
		}
	}

	// Create series
	seriesID := uuid.New()
	now := time.Now()

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	series := &models.Series{
		ID:         seriesID,
		Title:      req.Title,
		Summary:    req.Summary,
		Notes:      req.Notes,
		UserID:     userUUID,
		IsComplete: req.IsComplete,
		WorkCount:  len(req.WorkIDs),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err = tx.Exec(`
		INSERT INTO series (id, title, summary, notes, user_id, is_complete, work_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		series.ID, series.Title, series.Summary, series.Notes, series.UserID,
		series.IsComplete, series.WorkCount, series.CreatedAt, series.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create series"})
		return
	}

	// Add works to series
	for i, workIDStr := range req.WorkIDs {
		workID, _ := uuid.Parse(workIDStr)
		_, err = tx.Exec(`
			INSERT INTO series_works (series_id, work_id, position, created_at)
			VALUES ($1, $2, $3, $4)`,
			seriesID, workID, i+1, now)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add work to series"})
			return
		}

		// Update work to reference series
		_, err = tx.Exec("UPDATE works SET series_id = $1, updated_at = $2 WHERE id = $3", seriesID, now, workID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update work series reference"})
			return
		}
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"series": series})
}

func (ws *WorkService) UpdateSeries(c *gin.Context) {
	seriesID, err := uuid.Parse(c.Param("series_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid series ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Title      string `json:"title" binding:"required,max=500"`
		Summary    string `json:"summary"`
		Notes      string `json:"notes"`
		IsComplete bool   `json:"is_complete"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Verify user owns the series
	var ownerID uuid.UUID
	err = ws.db.QueryRow("SELECT user_id FROM series WHERE id = $1", seriesID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify series ownership"})
		return
	}

	if ownerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own series"})
		return
	}

	// Update series
	now := time.Now()
	_, err = ws.db.Exec(`
		UPDATE series 
		SET title = $1, description = $2, notes = $3, is_complete = $4, updated_at = $5
		WHERE id = $6`,
		req.Title, req.Summary, req.Notes, req.IsComplete, now, seriesID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update series"})
		return
	}

	// Fetch updated series
	var series models.Series
	var username string
	err = ws.db.QueryRow(`
		SELECT s.id, s.title, s.description, s.notes, s.user_id, s.is_complete,
			s.work_count, s.created_at, s.updated_at, u.username
		FROM series s
		JOIN users u ON s.user_id = u.id
		WHERE s.id = $1`, seriesID).Scan(
		&series.ID, &series.Title, &series.Summary, &series.Notes, &series.UserID,
		&series.IsComplete, &series.WorkCount, &series.CreatedAt, &series.UpdatedAt, &username)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated series"})
		return
	}

	series.Username = username
	c.JSON(http.StatusOK, gin.H{"series": series})
}

func (ws *WorkService) DeleteSeries(c *gin.Context) {
	seriesID, err := uuid.Parse(c.Param("series_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid series ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Verify user owns the series
	var ownerID uuid.UUID
	err = ws.db.QueryRow("SELECT user_id FROM series WHERE id = $1", seriesID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify series ownership"})
		return
	}

	if ownerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own series"})
		return
	}

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Remove series reference from works
	_, err = tx.Exec("UPDATE works SET series_id = NULL WHERE series_id = $1", seriesID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update works"})
		return
	}

	// Delete series (series_works will be deleted by CASCADE)
	result, err := tx.Exec("DELETE FROM series WHERE id = $1", seriesID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete series"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Series deleted successfully"})
}

func (ws *WorkService) AddWorkToSeries(c *gin.Context) {
	seriesID, err := uuid.Parse(c.Param("series_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid series ID"})
		return
	}

	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Position int `json:"position"` // Optional, will be appended if not provided
	}
	c.ShouldBindJSON(&req)

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Verify user owns the series
	var seriesOwnerID uuid.UUID
	err = tx.QueryRow("SELECT user_id FROM series WHERE id = $1", seriesID).Scan(&seriesOwnerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify series ownership"})
		return
	}

	// Verify user owns the work
	var workOwnerID uuid.UUID
	err = tx.QueryRow("SELECT user_id FROM works WHERE id = $1", workID).Scan(&workOwnerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify work ownership"})
		return
	}

	if seriesOwnerID != userID || workOwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only add your own works to your own series"})
		return
	}

	// Check if work is already in series
	var existingCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM series_works WHERE series_id = $1 AND work_id = $2", seriesID, workID).Scan(&existingCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing relationship"})
		return
	}
	if existingCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Work is already in this series"})
		return
	}

	// Determine position
	var position int
	if req.Position > 0 {
		position = req.Position
		// Shift existing works to make room
		_, err = tx.Exec("UPDATE series_works SET position = position + 1 WHERE series_id = $1 AND position >= $2", seriesID, position)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update work positions"})
			return
		}
	} else {
		// Append to end
		err = tx.QueryRow("SELECT COALESCE(MAX(position), 0) + 1 FROM series_works WHERE series_id = $1", seriesID).Scan(&position)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to determine position"})
			return
		}
	}

	// Add work to series
	now := time.Now()
	_, err = tx.Exec("INSERT INTO series_works (series_id, work_id, position, created_at) VALUES ($1, $2, $3, $4)",
		seriesID, workID, position, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add work to series"})
		return
	}

	// Update work's series reference
	_, err = tx.Exec("UPDATE works SET series_id = $1, updated_at = $2 WHERE id = $3", seriesID, now, workID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update work series reference"})
		return
	}

	// Update series work count
	_, err = tx.Exec("UPDATE series SET work_count = (SELECT COUNT(*) FROM series_works WHERE series_id = $1), updated_at = $2 WHERE id = $1", seriesID, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update series work count"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Work added to series successfully", "position": position})
}

func (ws *WorkService) RemoveWorkFromSeries(c *gin.Context) {
	seriesID, err := uuid.Parse(c.Param("series_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid series ID"})
		return
	}

	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Verify user owns the series and work
	var seriesOwnerID, workOwnerID uuid.UUID
	var position int
	err = tx.QueryRow(`
		SELECT s.user_id, w.user_id, sw.position 
		FROM series s, works w, series_works sw 
		WHERE s.id = $1 AND w.id = $2 AND sw.series_id = $1 AND sw.work_id = $2`,
		seriesID, workID).Scan(&seriesOwnerID, &workOwnerID, &position)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found in this series"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	if seriesOwnerID != userID || workOwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only remove your own works from your own series"})
		return
	}

	// Remove work from series
	_, err = tx.Exec("DELETE FROM series_works WHERE series_id = $1 AND work_id = $2", seriesID, workID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove work from series"})
		return
	}

	// Update work's series reference
	now := time.Now()
	_, err = tx.Exec("UPDATE works SET series_id = NULL, updated_at = $1 WHERE id = $2", now, workID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update work series reference"})
		return
	}

	// Reorder remaining works to close the gap
	_, err = tx.Exec("UPDATE series_works SET position = position - 1 WHERE series_id = $1 AND position > $2", seriesID, position)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder works"})
		return
	}

	// Update series work count
	_, err = tx.Exec("UPDATE series SET work_count = (SELECT COUNT(*) FROM series_works WHERE series_id = $1), updated_at = $2 WHERE id = $1", seriesID, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update series work count"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Work removed from series successfully"})
}

func (ws *WorkService) SearchCollections(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		// Return recent collections if no search query
		query = "%"
	} else {
		query = "%" + query + "%"
	}

	// Build pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Search collections by name, title and description
	searchQuery := `
		SELECT c.id, c.name, c.title, c.description, c.user_id, c.is_open,
			c.is_moderated, c.is_anonymous, c.work_count, c.created_at, c.updated_at,
			u.username
		FROM collections c
		JOIN users u ON c.user_id = u.id
		WHERE (c.name ILIKE $1 OR c.title ILIKE $1 OR c.description ILIKE $1)
		ORDER BY c.updated_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := ws.db.Query(searchQuery, query, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search collections"})
		return
	}
	defer rows.Close()

	var collections []gin.H
	for rows.Next() {
		var collection models.Collection
		var username string
		err := rows.Scan(
			&collection.ID, &collection.Name, &collection.Title, &collection.Description,
			&collection.UserID, &collection.IsOpen, &collection.IsModerated,
			&collection.IsAnonymous, &collection.WorkCount, &collection.CreatedAt,
			&collection.UpdatedAt, &username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan collection"})
			return
		}

		collections = append(collections, gin.H{
			"collection": collection,
			"maintainer": username,
		})
	}

	// Get total count for pagination
	var total int
	countQuery := `
		SELECT COUNT(*) 
		FROM collections c
		WHERE (c.name ILIKE $1 OR c.title ILIKE $1 OR c.description ILIKE $1)`
	err = ws.db.QueryRow(countQuery, query).Scan(&total)
	if err != nil {
		total = len(collections) // Fallback
	}

	c.JSON(http.StatusOK, gin.H{
		"collections": collections,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

func (ws *WorkService) GetCollection(c *gin.Context) {
	collectionID, err := uuid.Parse(c.Param("collection_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	var collection models.Collection
	var username string
	err = ws.db.QueryRow(`
		SELECT c.id, c.name, c.title, c.description, c.user_id, c.is_open, 
			c.is_moderated, c.is_anonymous, c.work_count, c.created_at, c.updated_at,
			u.username
		FROM collections c
		JOIN users u ON c.user_id = u.id
		WHERE c.id = $1`, collectionID).Scan(
		&collection.ID, &collection.Name, &collection.Title, &collection.Description,
		&collection.UserID, &collection.IsOpen, &collection.IsModerated, &collection.IsAnonymous,
		&collection.WorkCount, &collection.CreatedAt, &collection.UpdatedAt, &username)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch collection"})
		return
	}

	response := gin.H{
		"collection": collection,
		"maintainer": username,
	}

	c.JSON(http.StatusOK, response)
}

func (ws *WorkService) GetCollectionWorks(c *gin.Context) {
	collectionID, err := uuid.Parse(c.Param("collection_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	// Check if collection exists
	var collectionExists bool
	err = ws.db.QueryRow("SELECT EXISTS(SELECT 1 FROM collections WHERE id = $1)", collectionID).Scan(&collectionExists)
	if err != nil || !collectionExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}

	// Get user ID for privacy checks
	userID, hasUser := c.Get("user_id")
	var userUUID *uuid.UUID
	if hasUser {
		userIDStr := userID.(string)
		if userVal, err := uuid.Parse(userIDStr); err == nil {
			userUUID = &userVal
		}
	}

	// Build pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Get works in the collection
	baseQuery := `
		SELECT w.id, w.title, w.summary, w.language, w.rating,
			w.category, w.warnings, w.fandoms, w.characters, w.relationships, w.freeform_tags,
			w.word_count, w.chapter_count, w.max_chapters, w.is_complete, w.status,
			w.published_at, w.updated_at, w.created_at,
			COALESCE(w.hit_count, 0) as hits, COALESCE(w.kudos_count, 0) as kudos,
			COALESCE(w.comment_count, 0) as comments, COALESCE(w.bookmark_count, 0) as bookmarks,
			ci.added_at, ci.is_approved
		FROM works w
		JOIN collection_items ci ON w.id = ci.work_id
		WHERE ci.collection_id = $1`

	// Only show approved items unless user is collection maintainer
	var isOwner bool
	if userUUID != nil {
		err = ws.db.QueryRow("SELECT user_id = $1 FROM collections WHERE id = $2", *userUUID, collectionID).Scan(&isOwner)
		if err != nil {
			isOwner = false
		}
	}

	if !isOwner {
		baseQuery += " AND ci.is_approved = true"
	}

	// If no user, only show non-draft, non-restricted works
	if !hasUser {
		baseQuery += " AND w.status != 'draft' AND w.restricted = false"
	}

	baseQuery += " ORDER BY ci.added_at DESC LIMIT $2 OFFSET $3"

	rows, err := ws.db.Query(baseQuery, collectionID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch collection works"})
		return
	}
	defer rows.Close()

	works := []gin.H{}
	for rows.Next() {
		var work models.Work
		var categoryStr, warningsStr sql.NullString
		var fandoms, characters, relationships, freeformTags pq.StringArray
		var summary sql.NullString
		var publishedAt sql.NullTime
		var status sql.NullString
		var maxChapters sql.NullInt64
		var addedAt time.Time
		var isApproved bool

		err := rows.Scan(
			&work.ID, &work.Title, &summary,
			&work.Language, &work.Rating, &categoryStr, &warningsStr,
			&fandoms, &characters, &relationships, &freeformTags,
			&work.WordCount, &work.ChapterCount, &maxChapters,
			&work.IsComplete, &status, &publishedAt, &work.UpdatedAt, &work.CreatedAt,
			&work.Hits, &work.Kudos, &work.Comments, &work.Bookmarks,
			&addedAt, &isApproved)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan work"})
			return
		}

		// Handle nullable fields
		if summary.Valid {
			work.Summary = summary.String
		}
		if publishedAt.Valid {
			work.PublishedAt = &publishedAt.Time
		}
		if maxChapters.Valid {
			maxChapInt := int(maxChapters.Int64)
			work.MaxChapters = &maxChapInt
		}
		if status.Valid {
			work.Status = status.String
		}

		// Convert string fields to arrays
		if categoryStr.Valid && categoryStr.String != "" {
			work.Category = []string{categoryStr.String}
		}
		if warningsStr.Valid && warningsStr.String != "" {
			work.Warnings = []string{warningsStr.String}
		}
		work.Fandoms = []string(fandoms)
		work.Characters = []string(characters)
		work.Relationships = []string(relationships)
		work.FreeformTags = []string(freeformTags)

		// Check if user can view this specific work
		if userUUID != nil {
			var canView bool
			err = ws.db.QueryRow("SELECT can_user_view_work($1, $2)", work.ID, *userUUID).Scan(&canView)
			if err != nil || !canView {
				continue // Skip this work
			}
		}

		workData := gin.H{
			"work":        work,
			"added_at":    addedAt,
			"is_approved": isApproved,
		}

		works = append(works, workData)
	}

	// Get total count for pagination
	countQuery := `
		SELECT COUNT(*) 
		FROM collection_items ci
		JOIN works w ON ci.work_id = w.id
		WHERE ci.collection_id = $1`

	args := []interface{}{collectionID}
	if !isOwner {
		countQuery += " AND ci.is_approved = true"
	}
	if !hasUser {
		countQuery += " AND w.status != 'draft' AND w.restricted = false"
	}

	var total int
	err = ws.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		total = len(works) // Fallback
	}

	c.JSON(http.StatusOK, gin.H{
		"works": works,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

func (ws *WorkService) CreateCollection(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Name        string `json:"name" validate:"required,min=1,max=100"`
		Title       string `json:"title" validate:"required,min=1,max=200"`
		Description string `json:"description"`
		IsOpen      bool   `json:"is_open"`
		IsModerated bool   `json:"is_moderated"`
		IsAnonymous bool   `json:"is_anonymous"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	userIDStr := userID.(string)
	userUUID, parseErr := uuid.Parse(userIDStr)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if collection name is unique
	var existingID uuid.UUID
	err := ws.db.QueryRow("SELECT id FROM collections WHERE name = $1", req.Name).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Collection name already exists"})
		return
	}

	// Create collection
	collectionID := uuid.New()
	now := time.Now()

	collection := &models.Collection{
		ID:          collectionID,
		Name:        req.Name,
		Title:       req.Title,
		Description: req.Description,
		UserID:      userUUID,
		IsOpen:      req.IsOpen,
		IsModerated: req.IsModerated,
		IsAnonymous: req.IsAnonymous,
		WorkCount:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = ws.db.Exec(`
		INSERT INTO collections (id, name, title, description, user_id, is_open, is_moderated, is_anonymous, work_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		collection.ID, collection.Name, collection.Title, collection.Description, collection.UserID,
		collection.IsOpen, collection.IsModerated, collection.IsAnonymous, collection.WorkCount,
		collection.CreatedAt, collection.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create collection"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"collection": collection})
}

func (ws *WorkService) UpdateCollection(c *gin.Context) {
	collectionID, err := uuid.Parse(c.Param("collection_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		IsOpen      *bool   `json:"is_open"`
		IsModerated *bool   `json:"is_moderated"`
		IsAnonymous *bool   `json:"is_anonymous"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Verify user owns the collection
	var ownerID uuid.UUID
	err = ws.db.QueryRow("SELECT user_id FROM collections WHERE id = $1", collectionID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify collection ownership"})
		return
	}

	userUUID, parseErr := uuid.Parse(userID.(string))
	if parseErr != nil || ownerID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own collections"})
		return
	}

	// Check if new name is unique (if name is being changed)
	if req.Name != nil {
		var existingID uuid.UUID
		err := ws.db.QueryRow("SELECT id FROM collections WHERE name = $1 AND id != $2", *req.Name, collectionID).Scan(&existingID)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Collection name already exists"})
			return
		}
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}
	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, *req.Title)
		argIndex++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *req.Description)
		argIndex++
	}
	if req.IsOpen != nil {
		updates = append(updates, fmt.Sprintf("is_open = $%d", argIndex))
		args = append(args, *req.IsOpen)
		argIndex++
	}
	if req.IsModerated != nil {
		updates = append(updates, fmt.Sprintf("is_moderated = $%d", argIndex))
		args = append(args, *req.IsModerated)
		argIndex++
	}
	if req.IsAnonymous != nil {
		updates = append(updates, fmt.Sprintf("is_anonymous = $%d", argIndex))
		args = append(args, *req.IsAnonymous)
		argIndex++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Add updated_at
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add collection ID for WHERE clause
	args = append(args, collectionID)

	query := fmt.Sprintf("UPDATE collections SET %s WHERE id = $%d", strings.Join(updates, ", "), argIndex)

	_, err = ws.db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update collection", "details": err.Error()})
		return
	}

	// Fetch updated collection
	var collection models.Collection
	var username string
	err = ws.db.QueryRow(`
		SELECT c.id, c.name, c.title, c.description, c.user_id, c.is_open, 
			c.is_moderated, c.is_anonymous, c.work_count, c.created_at, c.updated_at,
			u.username
		FROM collections c
		JOIN users u ON c.user_id = u.id
		WHERE c.id = $1`, collectionID).Scan(
		&collection.ID, &collection.Name, &collection.Title, &collection.Description,
		&collection.UserID, &collection.IsOpen, &collection.IsModerated, &collection.IsAnonymous,
		&collection.WorkCount, &collection.CreatedAt, &collection.UpdatedAt, &username)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated collection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"collection": collection,
		"maintainer": username,
	})
}

func (ws *WorkService) DeleteCollection(c *gin.Context) {
	collectionID, err := uuid.Parse(c.Param("collection_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Verify user owns the collection
	var ownerID uuid.UUID
	var workCount int
	err = ws.db.QueryRow("SELECT user_id, work_count FROM collections WHERE id = $1", collectionID).Scan(&ownerID, &workCount)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify collection ownership"})
		return
	}

	userUUID, parseErr := uuid.Parse(userID.(string))
	if parseErr != nil || ownerID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own collections"})
		return
	}

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Delete collection items first (foreign key constraint)
	_, err = tx.Exec("DELETE FROM collection_items WHERE collection_id = $1", collectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete collection items"})
		return
	}

	// Delete the collection
	_, err = tx.Exec("DELETE FROM collections WHERE id = $1", collectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete collection"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":               "Collection deleted successfully",
		"deleted_collection_id": collectionID,
		"removed_work_count":    workCount,
	})
}

func (ws *WorkService) AddWorkToCollection(c *gin.Context) {
	collectionID, err := uuid.Parse(c.Param("collection_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr := userID.(string)
	userUUID, parseErr := uuid.Parse(userIDStr)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get collection settings
	var collection models.Collection
	err = ws.db.QueryRow(`
		SELECT id, user_id, is_open, is_moderated
		FROM collections WHERE id = $1`, collectionID).Scan(
		&collection.ID, &collection.UserID, &collection.IsOpen, &collection.IsModerated)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch collection"})
		return
	}

	// Check permissions - either collection maintainer, work author, or open collection
	var canAdd bool
	var isWorkAuthor bool

	// Check if user is collection maintainer
	isMaintainer := collection.UserID == userUUID

	// Check if user is work author
	err = ws.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM creatorships cr
			JOIN pseuds p ON cr.pseud_id = p.id
			WHERE cr.creation_id = $1 AND cr.creation_type = 'Work' 
			AND cr.approved = true AND p.user_id = $2
		)`, workID, userUUID).Scan(&isWorkAuthor)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check work authorship"})
		return
	}

	canAdd = isMaintainer || (collection.IsOpen && isWorkAuthor)

	if !canAdd {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot add work to this collection"})
		return
	}

	// Check if work is already in collection
	var existingItemID uuid.UUID
	err = ws.db.QueryRow("SELECT id FROM collection_items WHERE collection_id = $1 AND work_id = $2", collectionID, workID).Scan(&existingItemID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Work is already in this collection"})
		return
	}

	// Add work to collection
	itemID := uuid.New()
	now := time.Now()
	isApproved := !collection.IsModerated || isMaintainer

	var approvedAt *time.Time
	if isApproved {
		approvedAt = &now
	}

	_, err = ws.db.Exec(`
		INSERT INTO collection_items (id, collection_id, work_id, added_by, is_approved, added_at, approved_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		itemID, collectionID, workID, userUUID, isApproved, now, approvedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add work to collection"})
		return
	}

	// Update collection work count if approved
	if isApproved {
		_, err = ws.db.Exec(`
			UPDATE collections SET 
				work_count = (SELECT COUNT(*) FROM collection_items WHERE collection_id = $1 AND is_approved = true),
				updated_at = $2
			WHERE id = $1`, collectionID, now)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update collection count"})
			return
		}
	}

	message := "Work added to collection"
	if !isApproved {
		message = "Work submitted to collection for approval"
	}

	c.JSON(http.StatusCreated, gin.H{"message": message})
}

func (ws *WorkService) RemoveWorkFromCollection(c *gin.Context) {
	collectionID, err := uuid.Parse(c.Param("collection_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, parseErr := uuid.Parse(userID.(string))
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if work is in collection and get details
	var item models.CollectionItem
	var collectionOwnerID uuid.UUID
	err = ws.db.QueryRow(`
		SELECT ci.id, ci.collection_id, ci.work_id, ci.added_by, ci.is_approved, ci.added_at, c.user_id
		FROM collection_items ci
		JOIN collections c ON ci.collection_id = c.id
		WHERE ci.collection_id = $1 AND ci.work_id = $2`, collectionID, workID).Scan(
		&item.ID, &item.CollectionID, &item.WorkID, &item.AddedBy, &item.IsApproved, &item.AddedAt, &collectionOwnerID)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found in collection"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch collection item"})
		return
	}

	// Check permissions - either collection maintainer, work author, or the person who added it
	var canRemove bool

	// Check if user is collection maintainer
	isMaintainer := collectionOwnerID == userUUID

	// Check if user is work author
	var isWorkAuthor bool
	err = ws.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM creatorships cr
			JOIN pseuds p ON cr.pseud_id = p.id
			WHERE cr.creation_id = $1 AND cr.creation_type = 'Work' 
			AND cr.approved = true AND p.user_id = $2
		)`, workID, userUUID).Scan(&isWorkAuthor)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check work authorship"})
		return
	}

	// Check if user is the one who added the work
	isAdder := item.AddedBy == userUUID

	canRemove = isMaintainer || isWorkAuthor || isAdder

	if !canRemove {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot remove work from this collection"})
		return
	}

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Remove work from collection
	_, err = tx.Exec("DELETE FROM collection_items WHERE id = $1", item.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove work from collection"})
		return
	}

	// Update collection work count
	_, err = tx.Exec(`
		UPDATE collections SET 
			work_count = (SELECT COUNT(*) FROM collection_items WHERE collection_id = $1 AND is_approved = true),
			updated_at = $2
		WHERE id = $1`, collectionID, time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update collection count"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Work removed from collection successfully",
		"was_approved": item.IsApproved,
	})
}

func (ws *WorkService) GetUserWorks(c *gin.Context) {
	userIDParam := c.Param("user_id")
	targetUserID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get authenticated user (may be nil for guest viewing)
	var viewerID *uuid.UUID
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			if uid, parseErr := uuid.Parse(userIDStr); parseErr == nil {
				viewerID = &uid
			}
		}
	}

	// Check if viewer can see target user's works
	isOwnProfile := viewerID != nil && *viewerID == targetUserID

	// Build pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Get works created by the user
	baseQuery := `
		SELECT w.id, w.title, w.summary, w.language, w.rating,
			w.category, w.warnings, w.fandoms, w.characters, w.relationships, w.freeform_tags,
			w.word_count, w.chapter_count, w.max_chapters, w.is_complete, w.status,
			w.published_at, w.updated_at, w.created_at,
			COALESCE(w.hit_count, 0) as hits, COALESCE(w.kudos_count, 0) as kudos,
			COALESCE(w.comment_count, 0) as comments, COALESCE(w.bookmark_count, 0) as bookmarks
		FROM works w
		JOIN creatorships cr ON w.id = cr.creation_id AND cr.creation_type = 'Work'
		JOIN pseuds p ON cr.pseud_id = p.id
		WHERE p.user_id = $1 AND cr.approved = true`

	// If not viewing own profile, only show published, non-restricted works
	if !isOwnProfile {
		baseQuery += " AND w.status = 'posted' AND w.restricted = false"
	}

	baseQuery += " ORDER BY w.updated_at DESC LIMIT $2 OFFSET $3"

	rows, err := ws.db.Query(baseQuery, targetUserID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user works"})
		return
	}
	defer rows.Close()

	works := []models.Work{}
	for rows.Next() {
		var work models.Work
		var categoryStr, warningsStr sql.NullString
		var fandoms, characters, relationships, freeformTags pq.StringArray
		var summary sql.NullString
		var publishedAt sql.NullTime
		var status sql.NullString
		var maxChapters sql.NullInt64

		err := rows.Scan(
			&work.ID, &work.Title, &summary,
			&work.Language, &work.Rating, &categoryStr, &warningsStr,
			&fandoms, &characters, &relationships, &freeformTags,
			&work.WordCount, &work.ChapterCount, &maxChapters,
			&work.IsComplete, &status, &publishedAt, &work.UpdatedAt, &work.CreatedAt,
			&work.Hits, &work.Kudos, &work.Comments, &work.Bookmarks)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan work"})
			return
		}

		// Handle nullable fields
		if summary.Valid {
			work.Summary = summary.String
		}
		if publishedAt.Valid {
			work.PublishedAt = &publishedAt.Time
		}
		if maxChapters.Valid {
			maxChapInt := int(maxChapters.Int64)
			work.MaxChapters = &maxChapInt
		}
		if status.Valid {
			work.Status = status.String
		}

		// Convert string fields to arrays
		if categoryStr.Valid && categoryStr.String != "" {
			work.Category = []string{categoryStr.String}
		}
		if warningsStr.Valid && warningsStr.String != "" {
			work.Warnings = []string{warningsStr.String}
		}
		work.Fandoms = []string(fandoms)
		work.Characters = []string(characters)
		work.Relationships = []string(relationships)
		work.FreeformTags = []string(freeformTags)

		works = append(works, work)
	}

	// Get total count for pagination
	countQuery := `
		SELECT COUNT(*) 
		FROM works w
		JOIN creatorships cr ON w.id = cr.creation_id AND cr.creation_type = 'Work'
		JOIN pseuds p ON cr.pseud_id = p.id
		WHERE p.user_id = $1 AND cr.approved = true`

	args := []interface{}{targetUserID}
	if !isOwnProfile {
		countQuery += " AND w.status = 'posted' AND w.restricted = false"
	}

	var total int
	err = ws.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		total = len(works) // Fallback
	}

	// Get user info
	var username string
	err = ws.db.QueryRow("SELECT username FROM users WHERE id = $1", targetUserID).Scan(&username)
	if err != nil {
		username = "Unknown User"
	}

	c.JSON(http.StatusOK, gin.H{
		"works":    works,
		"username": username,
		"user_id":  targetUserID,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

func (ws *WorkService) GetUserSeries(c *gin.Context) {
	userIDParam := c.Param("user_id")
	targetUserID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Build query with pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	query := `
		SELECT s.id, s.title, s.description, s.notes, s.user_id, s.is_complete,
			s.work_count, s.created_at, s.updated_at, u.username
		FROM series s
		JOIN users u ON s.user_id = u.id
		WHERE s.user_id = $1
		ORDER BY s.updated_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := ws.db.Query(query, targetUserID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user series"})
		return
	}
	defer rows.Close()

	var series []models.Series
	for rows.Next() {
		var s models.Series
		err := rows.Scan(
			&s.ID, &s.Title, &s.Summary, &s.Notes, &s.UserID, &s.IsComplete,
			&s.WorkCount, &s.CreatedAt, &s.UpdatedAt, &s.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan series"})
			return
		}

		// Calculate word count for each series
		ws.db.QueryRow(`
			SELECT COALESCE(SUM(w.word_count), 0) 
			FROM works w 
			JOIN series_works sw ON w.id = sw.work_id
			WHERE sw.series_id = $1 AND w.status != 'draft'`, s.ID).Scan(&s.WordCount)

		series = append(series, s)
	}

	// Get total count for pagination
	var total int
	err = ws.db.QueryRow("SELECT COUNT(*) FROM series WHERE user_id = $1", targetUserID).Scan(&total)
	if err != nil {
		total = len(series) // Fallback
	}

	c.JSON(http.StatusOK, gin.H{
		"series": series,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

func (ws *WorkService) GetUserBookmarks(c *gin.Context) {
	userIDParam := c.Param("user_id")
	targetUserID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get authenticated user (may be nil for guest viewing)
	var viewerID *uuid.UUID
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			if uid, parseErr := uuid.Parse(userIDStr); parseErr == nil {
				viewerID = &uid
			}
		}
	}

	// Build query to get user's bookmarks
	query := `
		SELECT b.id, b.work_id, b.notes, b.tags, b.is_private, b.created_at, b.updated_at,
			   w.title, w.summary, w.rating, w.fandoms, w.characters, w.relationships, 
			   w.freeform_tags, w.word_count, w.chapter_count, w.is_complete, w.status,
			   w.published_at, w.updated_at as work_updated_at,
			   COALESCE(ws.hits, 0) as hits, COALESCE(ws.kudos, 0) as kudos,
			   COALESCE(ws.comments, 0) as comments, COALESCE(ws.bookmarks, 0) as bookmarks
		FROM bookmarks b
		JOIN works w ON b.work_id = w.id
		LEFT JOIN work_statistics ws ON w.id = ws.work_id
		WHERE b.user_id = $1`

	args := []interface{}{targetUserID}

	// If viewer is not the bookmark owner, only show public bookmarks
	if viewerID == nil || *viewerID != targetUserID {
		query += " AND b.is_private = false"
	}

	// Only show works the viewer can access
	if viewerID != nil {
		query += " AND can_user_view_work(w.id, $2)"
		args = append(args, *viewerID)
	} else {
		query += " AND w.restricted = false AND w.status = 'posted'"
	}

	query += " ORDER BY b.created_at DESC"

	rows, err := ws.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookmarks"})
		return
	}
	defer rows.Close()

	var bookmarks []gin.H
	for rows.Next() {
		var b models.Bookmark
		var w models.Work
		var hits, kudos, comments, bookmarkCount int

		err := rows.Scan(
			&b.ID, &b.WorkID, &b.Notes, pq.Array(&b.Tags), &b.IsPrivate, &b.CreatedAt, &b.UpdatedAt,
			&w.Title, &w.Summary, &w.Rating, pq.Array(&w.Fandoms), pq.Array(&w.Characters),
			pq.Array(&w.Relationships), pq.Array(&w.FreeformTags), &w.WordCount, &w.ChapterCount,
			&w.IsComplete, &w.Status, &w.PublishedAt, &w.UpdatedAt,
			&hits, &kudos, &comments, &bookmarkCount)

		if err != nil {
			continue
		}

		w.ID = b.WorkID
		w.Hits = hits
		w.Kudos = kudos
		w.Comments = comments
		w.Bookmarks = bookmarkCount

		// Get work authors using database function
		var authors []models.WorkAuthor
		authorRows, err := ws.db.Query("SELECT * FROM get_work_authors($1, $2)", w.ID, viewerID)
		if err == nil {
			defer authorRows.Close()
			for authorRows.Next() {
				var author models.WorkAuthor
				if err := authorRows.Scan(&author.PseudID, &author.PseudName, &author.UserID, &author.Username, &author.IsAnonymous); err == nil {
					authors = append(authors, author)
				}
			}
		}

		bookmarks = append(bookmarks, gin.H{
			"id":         b.ID,
			"work_id":    b.WorkID,
			"notes":      b.Notes,
			"tags":       b.Tags,
			"is_private": b.IsPrivate,
			"created_at": b.CreatedAt,
			"updated_at": b.UpdatedAt,
			"work": gin.H{
				"id":            w.ID,
				"title":         w.Title,
				"summary":       w.Summary,
				"rating":        w.Rating,
				"fandoms":       w.Fandoms,
				"characters":    w.Characters,
				"relationships": w.Relationships,
				"freeform_tags": w.FreeformTags,
				"word_count":    w.WordCount,
				"chapter_count": w.ChapterCount,
				"is_complete":   w.IsComplete,
				"status":        w.Status,
				"published_at":  w.PublishedAt,
				"updated_at":    w.UpdatedAt,
				"hits":          w.Hits,
				"kudos":         w.Kudos,
				"comments":      w.Comments,
				"bookmarks":     w.Bookmarks,
				"authors":       authors,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"bookmarks": bookmarks})
}

func (ws *WorkService) CreateBookmark(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Notes     string   `json:"notes"`
		Tags      []string `json:"tags"`
		IsPrivate bool     `json:"is_private"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	userIDStr := userID.(string)
	userUUID, parseErr := uuid.Parse(userIDStr)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if user can view this work
	var canView bool
	err = ws.db.QueryRow("SELECT can_user_view_work($1, $2)", workID, userUUID).Scan(&canView)
	if err != nil || !canView {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot bookmark this work"})
		return
	}

	// Check if bookmark already exists
	var existingID uuid.UUID
	err = ws.db.QueryRow("SELECT id FROM bookmarks WHERE work_id = $1 AND user_id = $2", workID, userUUID).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "You have already bookmarked this work"})
		return
	}

	// Create bookmark
	bookmarkID := uuid.New()
	now := time.Now()

	bookmark := &models.Bookmark{
		ID:        bookmarkID,
		WorkID:    workID,
		UserID:    userUUID,
		Notes:     req.Notes,
		Tags:      req.Tags,
		IsPrivate: req.IsPrivate,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = ws.db.Exec(`
		INSERT INTO bookmarks (id, work_id, user_id, notes, tags, is_private, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		bookmark.ID, bookmark.WorkID, bookmark.UserID, bookmark.Notes,
		pq.Array(bookmark.Tags), bookmark.IsPrivate, bookmark.CreatedAt, bookmark.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bookmark"})
		return
	}

	// Update work bookmark count
	_, err = ws.db.Exec(`
		UPDATE works SET 
			bookmark_count = (SELECT COUNT(*) FROM bookmarks WHERE work_id = $1),
			updated_at = $2
		WHERE id = $1`, workID, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bookmark count"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"bookmark": bookmark})
}

func (ws *WorkService) GetBookmarkStatus(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{"is_bookmarked": false})
		return
	}

	userIDStr := userID.(string)
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if bookmark exists
	var bookmarkID uuid.UUID
	err = ws.db.QueryRow("SELECT id FROM bookmarks WHERE work_id = $1 AND user_id = $2", workID, userUUID).Scan(&bookmarkID)

	isBookmarked := err == nil
	c.JSON(http.StatusOK, gin.H{"is_bookmarked": isBookmarked})
}

func (ws *WorkService) UpdateBookmark(c *gin.Context) {
	bookmarkID, err := uuid.Parse(c.Param("bookmark_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bookmark ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Notes     *string  `json:"notes"`
		Tags      []string `json:"tags"`
		IsPrivate *bool    `json:"is_private"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	userIDStr := userID.(string)
	userUUID, parseErr := uuid.Parse(userIDStr)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if bookmark exists and belongs to user
	var existingBookmark models.Bookmark
	err = ws.db.QueryRow(`
		SELECT id, work_id, user_id, notes, tags, is_private, created_at, updated_at
		FROM bookmarks WHERE id = $1 AND user_id = $2`,
		bookmarkID, userUUID).Scan(
		&existingBookmark.ID, &existingBookmark.WorkID, &existingBookmark.UserID,
		&existingBookmark.Notes, pq.Array(&existingBookmark.Tags), &existingBookmark.IsPrivate,
		&existingBookmark.CreatedAt, &existingBookmark.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bookmark not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookmark"})
		}
		return
	}

	// Update fields if provided
	if req.Notes != nil {
		existingBookmark.Notes = *req.Notes
	}
	if req.Tags != nil {
		existingBookmark.Tags = req.Tags
	}
	if req.IsPrivate != nil {
		existingBookmark.IsPrivate = *req.IsPrivate
	}
	existingBookmark.UpdatedAt = time.Now()

	// Update bookmark in database
	_, err = ws.db.Exec(`
		UPDATE bookmarks SET notes = $1, tags = $2, is_private = $3, updated_at = $4
		WHERE id = $5`,
		existingBookmark.Notes, pq.Array(existingBookmark.Tags),
		existingBookmark.IsPrivate, existingBookmark.UpdatedAt, bookmarkID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bookmark"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bookmark": existingBookmark})
}

func (ws *WorkService) DeleteBookmark(c *gin.Context) {
	bookmarkID, err := uuid.Parse(c.Param("bookmark_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bookmark ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr := userID.(string)
	userUUID, parseErr := uuid.Parse(userIDStr)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get bookmark to check ownership and get work_id for count update
	var workID uuid.UUID
	err = ws.db.QueryRow(`
		SELECT work_id FROM bookmarks WHERE id = $1 AND user_id = $2`,
		bookmarkID, userUUID).Scan(&workID)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bookmark not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookmark"})
		}
		return
	}

	// Delete bookmark
	_, err = ws.db.Exec("DELETE FROM bookmarks WHERE id = $1", bookmarkID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bookmark"})
		return
	}

	// Update work bookmark count
	_, err = ws.db.Exec(`
		UPDATE works SET 
			bookmark_count = (SELECT COUNT(*) FROM bookmarks WHERE work_id = $1),
			updated_at = $2
		WHERE id = $1`, workID, time.Now())

	if err != nil {
		// Log error but don't fail the request since bookmark was deleted
		log.Printf("Failed to update bookmark count for work %s: %v", workID, err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bookmark deleted successfully"})
}

func (ws *WorkService) GetMyBookmarks(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr := userID.(string)
	userUUID, parseErr := uuid.Parse(userIDStr)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse query parameters for filtering and pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	tag := c.Query("tag")
	search := c.Query("q")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Build query with optional filters
	baseQuery := `
		SELECT b.id, b.work_id, b.notes, b.tags, b.is_private, b.created_at, b.updated_at,
			   w.title, w.summary, w.rating, w.fandoms, w.characters, w.relationships, 
			   w.freeform_tags, w.word_count, w.chapter_count, w.is_complete, w.status,
			   w.published_at, w.updated_at as work_updated_at,
			   COALESCE(ws.hits, 0) as hits, COALESCE(ws.kudos, 0) as kudos,
			   COALESCE(ws.comments, 0) as comments, COALESCE(ws.bookmarks, 0) as bookmarks
		FROM bookmarks b
		JOIN works w ON b.work_id = w.id
		LEFT JOIN work_statistics ws ON w.id = ws.work_id
		WHERE b.user_id = $1`

	args := []interface{}{userUUID}
	argCount := 1

	// Add tag filter
	if tag != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND $%d = ANY(b.tags)", argCount)
		args = append(args, tag)
	}

	// Add search filter
	if search != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND (w.title ILIKE $%d OR w.summary ILIKE $%d OR b.notes ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+search+"%")
	}

	// Count total bookmarks for pagination
	countQuery := strings.Replace(baseQuery,
		"SELECT b.id, b.work_id, b.notes, b.tags, b.is_private, b.created_at, b.updated_at, w.title, w.summary, w.rating, w.fandoms, w.characters, w.relationships, w.freeform_tags, w.word_count, w.chapter_count, w.is_complete, w.status, w.published_at, w.updated_at as work_updated_at, COALESCE(ws.hits, 0) as hits, COALESCE(ws.kudos, 0) as kudos, COALESCE(ws.comments, 0) as comments, COALESCE(ws.bookmarks, 0) as bookmarks",
		"SELECT COUNT(*)", 1)

	var total int
	err := ws.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count bookmarks"})
		return
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY b.created_at DESC"
	argCount++
	baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, limit)
	argCount++
	baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)

	rows, err := ws.db.Query(baseQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookmarks"})
		return
	}
	defer rows.Close()

	var bookmarks []gin.H
	for rows.Next() {
		var b models.Bookmark
		var w models.Work
		var hits, kudos, comments, bookmarkCount int

		err := rows.Scan(
			&b.ID, &b.WorkID, &b.Notes, pq.Array(&b.Tags), &b.IsPrivate, &b.CreatedAt, &b.UpdatedAt,
			&w.Title, &w.Summary, &w.Rating, pq.Array(&w.Fandoms), pq.Array(&w.Characters),
			pq.Array(&w.Relationships), pq.Array(&w.FreeformTags), &w.WordCount, &w.ChapterCount,
			&w.IsComplete, &w.Status, &w.PublishedAt, &w.UpdatedAt,
			&hits, &kudos, &comments, &bookmarkCount)

		if err != nil {
			continue
		}

		w.ID = b.WorkID
		w.Hits = hits
		w.Kudos = kudos
		w.Comments = comments
		w.Bookmarks = bookmarkCount

		// Get work authors
		var authors []models.WorkAuthor
		authorRows, err := ws.db.Query("SELECT * FROM get_work_authors($1, $2)", w.ID, userUUID)
		if err == nil {
			defer authorRows.Close()
			for authorRows.Next() {
				var author models.WorkAuthor
				if err := authorRows.Scan(&author.PseudID, &author.PseudName, &author.UserID, &author.Username, &author.IsAnonymous); err == nil {
					authors = append(authors, author)
				}
			}
		}

		bookmarks = append(bookmarks, gin.H{
			"id":         b.ID,
			"work_id":    b.WorkID,
			"notes":      b.Notes,
			"tags":       b.Tags,
			"is_private": b.IsPrivate,
			"created_at": b.CreatedAt,
			"updated_at": b.UpdatedAt,
			"work": gin.H{
				"id":            w.ID,
				"title":         w.Title,
				"summary":       w.Summary,
				"rating":        w.Rating,
				"fandoms":       w.Fandoms,
				"characters":    w.Characters,
				"relationships": w.Relationships,
				"freeform_tags": w.FreeformTags,
				"word_count":    w.WordCount,
				"chapter_count": w.ChapterCount,
				"is_complete":   w.IsComplete,
				"status":        w.Status,
				"published_at":  w.PublishedAt,
				"updated_at":    w.UpdatedAt,
				"hits":          w.Hits,
				"kudos":         w.Kudos,
				"comments":      w.Comments,
				"bookmarks":     w.Bookmarks,
				"authors":       authors,
			},
		})
	}

	totalPages := (total + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
		"bookmarks": bookmarks,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (ws *WorkService) GetMyWorks(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Build query with pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Query to get user's works through creatorships
	query := `
		SELECT DISTINCT w.id, w.title, w.summary, w.language, w.rating, 
			w.category, w.warnings, w.fandoms, w.characters, w.relationships, w.freeform_tags,
			w.word_count, w.chapter_count, w.max_chapters, w.is_complete, w.status,
			w.published_at, w.updated_at, w.hit_count, w.kudos_count, w.comment_count, w.bookmark_count
		FROM works w
		JOIN creatorships c ON w.id = c.creation_id
		JOIN pseuds p ON c.pseud_id = p.id
		WHERE c.creation_type = 'Work' 
		AND c.approved = true
		AND p.user_id = $1
		ORDER BY w.updated_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := ws.db.Query(query, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch your works"})
		return
	}
	defer rows.Close()

	var works []map[string]interface{}
	for rows.Next() {
		var work map[string]interface{} = make(map[string]interface{})
		var id, title, language, rating, status string
		var summary sql.NullString
		var category, warnings sql.NullString
		var fandoms, characters, relationships, freeformTags pq.StringArray
		var wordCount, chapterCount int
		var maxChapters sql.NullInt64
		var isComplete bool
		var publishedAt sql.NullTime
		var updatedAt time.Time
		var hits, kudos, comments, bookmarks int

		err := rows.Scan(
			&id, &title, &summary, &language, &rating,
			&category, &warnings, &fandoms, &characters, &relationships, &freeformTags,
			&wordCount, &chapterCount, &maxChapters, &isComplete, &status,
			&publishedAt, &updatedAt, &hits, &kudos, &comments, &bookmarks)
		if err != nil {
			continue
		}

		work["id"] = id
		work["title"] = title
		work["summary"] = summary.String
		work["language"] = language
		work["rating"] = rating
		work["fandoms"] = []string(fandoms)
		work["characters"] = []string(characters)
		work["relationships"] = []string(relationships)
		work["freeform_tags"] = []string(freeformTags)
		work["word_count"] = wordCount
		work["chapter_count"] = chapterCount
		if maxChapters.Valid {
			work["max_chapters"] = maxChapters.Int64
		}
		work["is_complete"] = isComplete
		work["status"] = status
		if publishedAt.Valid {
			work["published_at"] = publishedAt.Time
		} else {
			work["published_at"] = nil
		}
		work["updated_at"] = updatedAt
		work["hits"] = hits
		work["kudos"] = kudos
		work["comments"] = comments
		work["bookmarks"] = bookmarks

		works = append(works, work)
	}

	c.JSON(http.StatusOK, gin.H{"works": works})
}

func (ws *WorkService) GetMySeries(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Build query with pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	query := `
		SELECT s.id, s.title, s.description, s.notes, s.user_id, s.is_complete,
			s.work_count, s.created_at, s.updated_at, u.username
		FROM series s
		JOIN users u ON s.user_id = u.id
		WHERE s.user_id = $1
		ORDER BY s.updated_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := ws.db.Query(query, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch your series"})
		return
	}
	defer rows.Close()

	var series []models.Series
	for rows.Next() {
		var s models.Series
		err := rows.Scan(
			&s.ID, &s.Title, &s.Summary, &s.Notes, &s.UserID, &s.IsComplete,
			&s.WorkCount, &s.CreatedAt, &s.UpdatedAt, &s.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan series"})
			return
		}

		// Calculate word count for each series
		ws.db.QueryRow(`
			SELECT COALESCE(SUM(w.word_count), 0) 
			FROM works w 
			JOIN series_works sw ON w.id = sw.work_id
			WHERE sw.series_id = $1`, s.ID).Scan(&s.WordCount)

		series = append(series, s)
	}

	// Get total count for pagination
	var total int
	err = ws.db.QueryRow("SELECT COUNT(*) FROM series WHERE user_id = $1", userID).Scan(&total)
	if err != nil {
		total = len(series) // Fallback
	}

	c.JSON(http.StatusOK, gin.H{
		"series": series,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

func (ws *WorkService) GetMyCollections(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Build pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Get user's collections (ones they maintain)
	query := `
		SELECT c.id, c.name, c.title, c.description, c.user_id, c.is_open,
			c.is_moderated, c.is_anonymous, c.work_count, c.created_at, c.updated_at,
			u.username
		FROM collections c
		JOIN users u ON c.user_id = u.id
		WHERE c.user_id = $1
		ORDER BY c.updated_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := ws.db.Query(query, userUUID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch collections"})
		return
	}
	defer rows.Close()

	var collections []gin.H
	for rows.Next() {
		var collection models.Collection
		var username string
		err := rows.Scan(
			&collection.ID, &collection.Name, &collection.Title, &collection.Description,
			&collection.UserID, &collection.IsOpen, &collection.IsModerated,
			&collection.IsAnonymous, &collection.WorkCount, &collection.CreatedAt,
			&collection.UpdatedAt, &username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan collection"})
			return
		}

		// Get pending approval count for moderated collections
		var pendingCount int
		if collection.IsModerated {
			err = ws.db.QueryRow(`
				SELECT COUNT(*) FROM collection_items 
				WHERE collection_id = $1 AND is_approved = false`, collection.ID).Scan(&pendingCount)
			if err != nil {
				pendingCount = 0
			}
		}

		collections = append(collections, gin.H{
			"collection":    collection,
			"maintainer":    username,
			"pending_count": pendingCount,
		})
	}

	// Get total count for pagination
	var total int
	err = ws.db.QueryRow("SELECT COUNT(*) FROM collections WHERE user_id = $1", userUUID).Scan(&total)
	if err != nil {
		total = len(collections) // Fallback
	}

	c.JSON(http.StatusOK, gin.H{
		"collections": collections,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

func (ws *WorkService) GetMyComments(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Build pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Filter by status (optional)
	status := c.Query("status") // e.g., "published", "pending_moderation", etc.

	// Get user's comments with work details
	baseQuery := `
		SELECT c.id, c.work_id, c.chapter_id, c.user_id, c.parent_comment_id, c.content,
			c.status, c.is_anonymous, c.created_at, c.updated_at,
			w.title as work_title, w.id as work_id,
			CASE WHEN c.parent_comment_id IS NOT NULL THEN true ELSE false END as is_reply
		FROM comments c
		JOIN works w ON c.work_id = w.id
		WHERE c.user_id = $1`

	args := []interface{}{userUUID}
	argIndex := 1

	if status != "" {
		argIndex++
		baseQuery += fmt.Sprintf(" AND c.status = $%d", argIndex)
		args = append(args, status)
	}

	baseQuery += " ORDER BY c.created_at DESC LIMIT $%d OFFSET $%d"
	args = append(args, limit, offset)

	rows, err := ws.db.Query(baseQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
		return
	}
	defer rows.Close()

	comments := []gin.H{}
	for rows.Next() {
		var comment models.WorkComment
		var workTitle string
		var workID uuid.UUID
		var isReply bool
		var chapterID sql.NullString
		var parentID sql.NullString

		err := rows.Scan(
			&comment.ID, &comment.WorkID, &chapterID, &comment.UserID, &parentID,
			&comment.Content, &comment.Status, &comment.IsAnonymous,
			&comment.CreatedAt, &comment.UpdatedAt, &workTitle, &workID, &isReply)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan comment"})
			return
		}

		// Handle nullable fields
		if chapterID.Valid {
			if chapterUUID, parseErr := uuid.Parse(chapterID.String); parseErr == nil {
				comment.ChapterID = &chapterUUID
			}
		}
		if parentID.Valid {
			if parentUUID, parseErr := uuid.Parse(parentID.String); parseErr == nil {
				comment.ParentID = &parentUUID
			}
		}

		// Get parent comment content if this is a reply (for context)
		var parentContent string
		if isReply && comment.ParentID != nil {
			err = ws.db.QueryRow(`
				SELECT SUBSTRING(content FROM 1 FOR 100) || CASE WHEN LENGTH(content) > 100 THEN '...' ELSE '' END
				FROM comments WHERE id = $1`, *comment.ParentID).Scan(&parentContent)
			if err != nil {
				parentContent = ""
			}
		}

		commentData := gin.H{
			"comment":        comment,
			"work_title":     workTitle,
			"work_id":        workID,
			"is_reply":       isReply,
			"parent_content": parentContent,
		}

		comments = append(comments, commentData)
	}

	// Get total count for pagination
	countQuery := `
		SELECT COUNT(*) 
		FROM comments c
		WHERE c.user_id = $1`

	countArgs := []interface{}{userUUID}
	if status != "" {
		countQuery += " AND c.status = $2"
		countArgs = append(countArgs, status)
	}

	var total int
	err = ws.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		total = len(comments) // Fallback
	}

	// Get status counts for filtering
	var statusCounts gin.H
	statusRows, err := ws.db.Query(`
		SELECT status, COUNT(*) 
		FROM comments 
		WHERE user_id = $1 
		GROUP BY status`, userUUID)

	if err == nil {
		defer statusRows.Close()
		statusCounts = gin.H{}
		for statusRows.Next() {
			var status string
			var count int
			if err := statusRows.Scan(&status, &count); err == nil {
				statusCounts[status] = count
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
		"status_counts": statusCounts,
	})
}

func (ws *WorkService) GetMyStats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get comprehensive user statistics
	var stats struct {
		UserID uuid.UUID `json:"user_id"`

		// Work counts
		TotalWorks     int `json:"total_works"`
		PublishedWorks int `json:"published_works"`
		DraftWorks     int `json:"draft_works"`
		CompleteWorks  int `json:"complete_works"`

		// Content statistics
		TotalWordCount   int `json:"total_word_count"`
		TotalChapters    int `json:"total_chapters"`
		AverageWordCount int `json:"average_word_count"`

		// Engagement statistics
		TotalHits          int `json:"total_hits"`
		TotalKudos         int `json:"total_kudos"`
		TotalComments      int `json:"total_comments"`
		TotalBookmarks     int `json:"total_bookmarks"`
		TotalSubscriptions int `json:"total_subscriptions"`

		// Series and collection statistics
		TotalSeries      int `json:"total_series"`
		TotalCollections int `json:"total_collections"`

		// Activity metrics
		FirstPublished *time.Time `json:"first_published"`
		LastPublished  *time.Time `json:"last_published"`
		LastUpdated    *time.Time `json:"last_updated"`

		// Top performing works
		TopWorks []struct {
			ID        uuid.UUID `json:"id"`
			Title     string    `json:"title"`
			Hits      int       `json:"hits"`
			Kudos     int       `json:"kudos"`
			Comments  int       `json:"comments"`
			Bookmarks int       `json:"bookmarks"`
		} `json:"top_works"`
	}

	stats.UserID = userUUID

	// Get work counts and content statistics
	err = ws.db.QueryRow(`
		SELECT 
			COUNT(*) as total_works,
			COUNT(CASE WHEN w.status = 'posted' THEN 1 END) as published_works,
			COUNT(CASE WHEN w.status = 'draft' THEN 1 END) as draft_works,
			COUNT(CASE WHEN w.is_complete = true THEN 1 END) as complete_works,
			COALESCE(SUM(w.word_count), 0) as total_word_count,
			COALESCE(SUM(w.chapter_count), 0) as total_chapters,
			COALESCE(AVG(w.word_count)::int, 0) as average_word_count
		FROM works w
		JOIN creatorships cr ON w.id = cr.creation_id AND cr.creation_type = 'Work'
		JOIN pseuds p ON cr.pseud_id = p.id
		WHERE p.user_id = $1 AND cr.approved = true`, userUUID).Scan(
		&stats.TotalWorks, &stats.PublishedWorks, &stats.DraftWorks,
		&stats.CompleteWorks, &stats.TotalWordCount, &stats.TotalChapters,
		&stats.AverageWordCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch work statistics"})
		return
	}

	// Get engagement statistics
	err = ws.db.QueryRow(`
		SELECT 
			COALESCE(SUM(w.hit_count), 0) as total_hits,
			COALESCE(SUM(w.kudos_count), 0) as total_kudos,
			COALESCE(SUM(w.comment_count), 0) as total_comments,
			COALESCE(SUM(w.bookmark_count), 0) as total_bookmarks
		FROM works w
		JOIN creatorships cr ON w.id = cr.creation_id AND cr.creation_type = 'Work'
		JOIN pseuds p ON cr.pseud_id = p.id
		WHERE p.user_id = $1 AND cr.approved = true`, userUUID).Scan(
		&stats.TotalHits, &stats.TotalKudos, &stats.TotalComments, &stats.TotalBookmarks)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch engagement statistics"})
		return
	}

	// Get subscription count
	err = ws.db.QueryRow(`
		SELECT COUNT(*)
		FROM subscriptions s
		JOIN works w ON s.subscribable_id = w.id AND s.subscribable_type = 'Work'
		JOIN creatorships cr ON w.id = cr.creation_id AND cr.creation_type = 'Work'
		JOIN pseuds p ON cr.pseud_id = p.id
		WHERE p.user_id = $1 AND cr.approved = true`, userUUID).Scan(&stats.TotalSubscriptions)

	if err != nil {
		stats.TotalSubscriptions = 0
	}

	// Get series count
	err = ws.db.QueryRow(`
		SELECT COUNT(*) FROM series WHERE user_id = $1`, userUUID).Scan(&stats.TotalSeries)

	if err != nil {
		stats.TotalSeries = 0
	}

	// Get collection count (collections created by user)
	err = ws.db.QueryRow(`
		SELECT COUNT(*) FROM collections WHERE user_id = $1`, userUUID).Scan(&stats.TotalCollections)

	if err != nil {
		stats.TotalCollections = 0
	}

	// Get activity metrics
	var firstPublished, lastPublished, lastUpdated sql.NullTime
	err = ws.db.QueryRow(`
		SELECT 
			MIN(w.published_at) as first_published,
			MAX(w.published_at) as last_published,
			MAX(w.updated_at) as last_updated
		FROM works w
		JOIN creatorships cr ON w.id = cr.creation_id AND cr.creation_type = 'Work'
		JOIN pseuds p ON cr.pseud_id = p.id
		WHERE p.user_id = $1 AND cr.approved = true AND w.status = 'posted'`, userUUID).Scan(
		&firstPublished, &lastPublished, &lastUpdated)

	if err == nil {
		if firstPublished.Valid {
			stats.FirstPublished = &firstPublished.Time
		}
		if lastPublished.Valid {
			stats.LastPublished = &lastPublished.Time
		}
		if lastUpdated.Valid {
			stats.LastUpdated = &lastUpdated.Time
		}
	}

	// Get top 5 performing works by total engagement (hits + kudos + comments + bookmarks)
	topWorksQuery := `
		SELECT w.id, w.title, 
			COALESCE(w.hit_count, 0) as hits,
			COALESCE(w.kudos_count, 0) as kudos,
			COALESCE(w.comment_count, 0) as comments,
			COALESCE(w.bookmark_count, 0) as bookmarks
		FROM works w
		JOIN creatorships cr ON w.id = cr.creation_id AND cr.creation_type = 'Work'
		JOIN pseuds p ON cr.pseud_id = p.id
		WHERE p.user_id = $1 AND cr.approved = true AND w.status = 'posted'
		ORDER BY (COALESCE(w.hit_count, 0) + COALESCE(w.kudos_count, 0) + 
		         COALESCE(w.comment_count, 0) + COALESCE(w.bookmark_count, 0)) DESC
		LIMIT 5`

	rows, err := ws.db.Query(topWorksQuery, userUUID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var topWork struct {
				ID        uuid.UUID `json:"id"`
				Title     string    `json:"title"`
				Hits      int       `json:"hits"`
				Kudos     int       `json:"kudos"`
				Comments  int       `json:"comments"`
				Bookmarks int       `json:"bookmarks"`
			}
			err := rows.Scan(&topWork.ID, &topWork.Title, &topWork.Hits,
				&topWork.Kudos, &topWork.Comments, &topWork.Bookmarks)
			if err == nil {
				stats.TopWorks = append(stats.TopWorks, topWork)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// Privacy and moderation handlers

func (ws *WorkService) ModerateComment(c *gin.Context) {
	commentID, err := uuid.Parse(c.Param("comment_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Status string `json:"status" validate:"required,oneof=published hidden deleted spam"`
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Check if user is the author of the work this comment belongs to
	var authorID uuid.UUID
	err = ws.db.QueryRow(`
		SELECT w.user_id 
		FROM works w
		JOIN comments c ON w.id = c.work_id
		WHERE c.id = $1`, commentID).Scan(&authorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	userIDStr := userID.(string)
	userUUID, parseErr := uuid.Parse(userIDStr)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if authorID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only work authors can moderate comments"})
		return
	}

	// Update comment status
	now := time.Now()
	_, err = ws.db.Exec(`
		UPDATE comments 
		SET status = $1, moderation_reason = $2, moderated_by = $3, moderated_at = $4, updated_at = $5
		WHERE id = $6`,
		req.Status, req.Reason, userID, now, now, commentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to moderate comment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment moderation updated"})
}

func (ws *WorkService) BlockUser(c *gin.Context) {
	blockedUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	blockerID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		BlockType string `json:"block_type" validate:"required,oneof=full comments works"`
		Reason    string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Can't block yourself
	if blockerID.(uuid.UUID) == blockedUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot block yourself"})
		return
	}

	// Create or update block
	blockID := uuid.New()
	now := time.Now()

	_, err = ws.db.Exec(`
		INSERT INTO user_blocks (id, blocker_id, blocked_id, block_type, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (blocker_id, blocked_id)
		DO UPDATE SET block_type = EXCLUDED.block_type, reason = EXCLUDED.reason`,
		blockID, blockerID, blockedUserID, req.BlockType, req.Reason, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User blocked successfully"})
}

func (ws *WorkService) UnblockUser(c *gin.Context) {
	blockedUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	blockerID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	_, err = ws.db.Exec(`
		DELETE FROM user_blocks 
		WHERE blocker_id = $1 AND blocked_id = $2`,
		blockerID, blockedUserID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unblock user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unblocked successfully"})
}

func (ws *WorkService) ReportComment(c *gin.Context) {
	commentID, err := uuid.Parse(c.Param("comment_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	var req struct {
		Reason      string `json:"reason" validate:"required,oneof=spam harassment off_topic inappropriate hate_speech doxxing other"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Get reporter info
	reporterID, hasUser := c.Get("user_id")
	var reporterUUID *uuid.UUID
	if hasUser {
		reporterVal := reporterID.(uuid.UUID)
		reporterUUID = &reporterVal
	}

	clientIP := c.ClientIP()

	// Create report
	reportID := uuid.New()
	now := time.Now()

	_, err = ws.db.Exec(`
		INSERT INTO comment_reports (id, comment_id, reporter_id, reporter_ip, reason, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		reportID, commentID, reporterUUID, clientIP, req.Reason, req.Description, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}

func (ws *WorkService) ReportWork(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	var req struct {
		Reason      string `json:"reason" validate:"required,oneof=copyright plagiarism harassment inappropriate_content wrong_rating missing_warnings spam other"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Get reporter info
	reporterID, hasUser := c.Get("user_id")
	var reporterUUID *uuid.UUID
	if hasUser {
		reporterVal := reporterID.(uuid.UUID)
		reporterUUID = &reporterVal
	}

	clientIP := c.ClientIP()

	// Create report
	reportID := uuid.New()
	now := time.Now()

	_, err = ws.db.Exec(`
		INSERT INTO work_reports (id, work_id, reporter_id, reporter_ip, reason, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		reportID, workID, reporterUUID, clientIP, req.Reason, req.Description, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}

// User muting handlers (matching AO3's implementation)

func (ws *WorkService) MuteUser(c *gin.Context) {
	mutedUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	muterID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Can't mute yourself
	if muterID.(uuid.UUID) == mutedUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot mute yourself"})
		return
	}

	// Check mute limit (similar to AO3's limits)
	var muteCount int
	err = ws.db.QueryRow("SELECT count_user_mutes($1)", muterID).Scan(&muteCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check mute limit"})
		return
	}

	maxMutes := 100 // Similar to AO3's limits
	if muteCount >= maxMutes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum muted users limit reached"})
		return
	}

	// Create or update mute
	muteID := uuid.New()
	now := time.Now()

	_, err = ws.db.Exec(`
		INSERT INTO user_mutes (id, muter_id, muted_id, reason, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (muter_id, muted_id)
		DO UPDATE SET reason = EXCLUDED.reason, updated_at = EXCLUDED.updated_at`,
		muteID, muterID, mutedUserID, req.Reason, now, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mute user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User muted successfully"})
}

func (ws *WorkService) UnmuteUser(c *gin.Context) {
	mutedUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	muterID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	_, err = ws.db.Exec(`
		DELETE FROM user_mutes 
		WHERE muter_id = $1 AND muted_id = $2`,
		muterID, mutedUserID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmute user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unmuted successfully"})
}

func (ws *WorkService) GetMutedUsers(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get user's muted list using the helper function
	rows, err := ws.db.Query("SELECT * FROM get_user_muted_list($1)", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch muted users"})
		return
	}
	defer rows.Close()

	type MutedUser struct {
		UserID   uuid.UUID `json:"user_id"`
		Username string    `json:"username"`
		Reason   string    `json:"reason"`
		MutedAt  time.Time `json:"muted_at"`
	}

	mutedUsers := []MutedUser{}
	for rows.Next() {
		var mutedUser MutedUser
		err := rows.Scan(&mutedUser.UserID, &mutedUser.Username, &mutedUser.Reason, &mutedUser.MutedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan muted user"})
			return
		}
		mutedUsers = append(mutedUsers, mutedUser)
	}

	c.JSON(http.StatusOK, gin.H{"muted_users": mutedUsers})
}

func (ws *WorkService) GetMuteStatus(c *gin.Context) {
	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user has muted the target user
	var isMuted bool
	err = ws.db.QueryRow("SELECT is_user_muted($1, $2)", userID, targetUserID).Scan(&isMuted)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check mute status"})
		return
	}

	var muteInfo struct {
		IsMuted bool      `json:"is_muted"`
		Reason  string    `json:"reason,omitempty"`
		MutedAt time.Time `json:"muted_at,omitempty"`
	}

	muteInfo.IsMuted = isMuted

	// If muted, get additional details
	if isMuted {
		err = ws.db.QueryRow(`
			SELECT reason, created_at 
			FROM user_mutes 
			WHERE muter_id = $1 AND muted_id = $2`,
			userID, targetUserID).Scan(&muteInfo.Reason, &muteInfo.MutedAt)
		if err != nil {
			// Still return the mute status even if we can't get details
			c.JSON(http.StatusOK, gin.H{"is_muted": true})
			return
		}
	}

	c.JSON(http.StatusOK, muteInfo)
}

// AO3 Core Features Handlers

// Pseud management
func (ws *WorkService) CreatePseud(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Name        string `json:"name" validate:"required,min=1,max=40"`
		Description string `json:"description"`
		IsDefault   bool   `json:"is_default"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Create pseud using database function
	var pseudID uuid.UUID
	err := ws.db.QueryRow("SELECT create_pseud($1, $2, $3)", userID, req.Name, req.IsDefault).Scan(&pseudID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pseud", "details": err.Error()})
		return
	}

	// Get the created pseud
	var pseud models.Pseud
	err = ws.db.QueryRow(`
		SELECT id, user_id, name, description, is_default, created_at, updated_at
		FROM pseuds WHERE id = $1`, pseudID).Scan(
		&pseud.ID, &pseud.UserID, &pseud.Name, &pseud.Description,
		&pseud.IsDefault, &pseud.CreatedAt, &pseud.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch created pseud"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"pseud": pseud})
}

func (ws *WorkService) GetUserPseuds(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	rows, err := ws.db.Query(`
		SELECT id, user_id, name, description, is_default, created_at, updated_at
		FROM pseuds WHERE user_id = $1 ORDER BY is_default DESC, name`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pseuds"})
		return
	}
	defer rows.Close()

	var pseuds []models.Pseud
	for rows.Next() {
		var pseud models.Pseud
		err := rows.Scan(&pseud.ID, &pseud.UserID, &pseud.Name, &pseud.Description,
			&pseud.IsDefault, &pseud.CreatedAt, &pseud.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan pseud"})
			return
		}
		pseuds = append(pseuds, pseud)
	}

	c.JSON(http.StatusOK, gin.H{"pseuds": pseuds})
}

// Work gifting
func (ws *WorkService) GiftWork(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		PseudID       *uuid.UUID `json:"pseud_id"`
		RecipientName string     `json:"recipient_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate that user is an author of the work
	var isAuthor bool
	err = ws.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM creatorships c
			JOIN pseuds p ON c.pseud_id = p.id
			WHERE c.creation_id = $1 AND c.creation_type = 'Work' 
			AND c.approved = true AND p.user_id = $2
		)`, workID, userID).Scan(&isAuthor)

	if err != nil || !isAuthor {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only gift works you authored"})
		return
	}

	// Create gift
	giftID := uuid.New()
	now := time.Now()

	gift := &models.Gift{
		ID:            giftID,
		WorkID:        workID,
		PseudID:       req.PseudID,
		RecipientName: req.RecipientName,
		Rejected:      false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	_, err = ws.db.Exec(`
		INSERT INTO gifts (id, work_id, pseud_id, recipient_name, rejected, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		gift.ID, gift.WorkID, gift.PseudID, gift.RecipientName,
		gift.Rejected, gift.CreatedAt, gift.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create gift"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"gift": gift})
}

func (ws *WorkService) GetWorkGifts(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	rows, err := ws.db.Query(`
		SELECT g.id, g.work_id, g.pseud_id, g.recipient_name, g.rejected, 
			g.created_at, g.updated_at, p.name as pseud_name, u.username
		FROM gifts g
		LEFT JOIN pseuds p ON g.pseud_id = p.id
		LEFT JOIN users u ON p.user_id = u.id
		WHERE g.work_id = $1 AND g.rejected = false
		ORDER BY g.created_at`, workID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch gifts"})
		return
	}
	defer rows.Close()

	var gifts []interface{}
	for rows.Next() {
		var gift models.Gift
		var pseudName, username sql.NullString
		err := rows.Scan(&gift.ID, &gift.WorkID, &gift.PseudID, &gift.RecipientName,
			&gift.Rejected, &gift.CreatedAt, &gift.UpdatedAt, &pseudName, &username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan gift"})
			return
		}

		giftData := gin.H{
			"id":         gift.ID,
			"work_id":    gift.WorkID,
			"rejected":   gift.Rejected,
			"created_at": gift.CreatedAt,
		}

		if gift.PseudID != nil {
			giftData["recipient"] = gin.H{
				"pseud_id":   gift.PseudID,
				"pseud_name": pseudName.String,
				"username":   username.String,
			}
		} else {
			giftData["recipient_name"] = gift.RecipientName
		}

		gifts = append(gifts, giftData)
	}

	c.JSON(http.StatusOK, gin.H{"gifts": gifts})
}

// Work orphaning
func (ws *WorkService) OrphanWork(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Use database function to orphan the work
	var success bool
	err = ws.db.QueryRow("SELECT orphan_work($1, $2)", workID, userID).Scan(&success)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to orphan work"})
		return
	}

	if !success {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not an author of this work"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Work orphaned successfully"})
}

// Get work authors (respecting anonymity)
func (ws *WorkService) GetWorkAuthors(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, _ := c.Get("user_id")

	// Use database function to get authors respecting anonymity
	rows, err := ws.db.Query("SELECT * FROM get_work_authors($1, $2)", workID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch authors"})
		return
	}
	defer rows.Close()

	var authors []models.WorkAuthor
	for rows.Next() {
		var author models.WorkAuthor
		err := rows.Scan(&author.PseudID, &author.PseudName, &author.UserID, &author.Username, &author.IsAnonymous)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan author"})
			return
		}
		authors = append(authors, author)
	}

	c.JSON(http.StatusOK, gin.H{"authors": authors})
}

// Add co-author to work
func (ws *WorkService) AddCoAuthor(c *gin.Context) {
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		PseudID uuid.UUID `json:"pseud_id" validate:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Verify user is current author
	var isAuthor bool
	err = ws.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM creatorships c
			JOIN pseuds p ON c.pseud_id = p.id
			WHERE c.creation_id = $1 AND c.creation_type = 'Work' 
			AND c.approved = true AND p.user_id = $2
		)`, workID, userID).Scan(&isAuthor)

	if err != nil || !isAuthor {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only existing authors can add co-authors"})
		return
	}

	// Add new creatorship
	creatorshipID := uuid.New()
	now := time.Now()

	_, err = ws.db.Exec(`
		INSERT INTO creatorships (id, creation_id, creation_type, pseud_id, approved, created_at, updated_at)
		VALUES ($1, $2, 'Work', $3, false, $4, $5)
		ON CONFLICT (creation_id, creation_type, pseud_id) DO NOTHING`,
		creatorshipID, workID, req.PseudID, now, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add co-author"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Co-author invitation sent"})
}

func (ws *WorkService) AdminListWorks(c *gin.Context) {
	// TODO: Implement admin work listing
	c.JSON(http.StatusOK, gin.H{"works": []gin.H{}})
}

func (ws *WorkService) AdminUpdateWorkStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user has moderator or admin privileges
	var role string
	err := ws.db.QueryRow(`
		SELECT COALESCE(role, 'user') FROM users WHERE id = $1`, userID).Scan(&role)

	if err != nil || (role != "moderator" && role != "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderator or admin access required"})
		return
	}

	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	var req struct {
		Status string `json:"status" validate:"required,oneof=draft published complete abandoned hiatus"`
		Reason string `json:"reason"` // Moderation reason
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Get current work status and author
	var currentStatus string
	var authorID uuid.UUID
	var workTitle string
	err = ws.db.QueryRow(`
		SELECT status, user_id, title FROM works WHERE id = $1`, workID).Scan(
		&currentStatus, &authorID, &workTitle)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch work"})
		return
	}

	// Only allow certain status changes for moderation (using database-allowed statuses)
	validTransitions := map[string][]string{
		"published": {"abandoned", "hiatus"},    // Published works can be marked abandoned or on hiatus
		"complete":  {"abandoned"},              // Complete works can only be abandoned
		"draft":     {"published", "abandoned"}, // Drafts can be published or abandoned
		"abandoned": {"published"},              // Abandoned works can be restored
		"hiatus":    {"published", "abandoned"}, // Works on hiatus can be resumed or abandoned
	}

	allowedStatuses, exists := validTransitions[currentStatus]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid current work status"})
		return
	}

	isValidTransition := false
	for _, allowedStatus := range allowedStatuses {
		if req.Status == allowedStatus {
			isValidTransition = true
			break
		}
	}

	if !isValidTransition {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Cannot change status from %s to %s", currentStatus, req.Status),
		})
		return
	}

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Update work status
	now := time.Now()
	_, err = tx.Exec(`
		UPDATE works 
		SET status = $1, updated_at = $2
		WHERE id = $3`, req.Status, now, workID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update work status"})
		return
	}

	// Log moderation action
	moderationLogID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO moderation_logs (id, moderator_id, target_type, target_id, action, reason, created_at)
		VALUES ($1, $2, 'work', $3, $4, $5, $6)`,
		moderationLogID, userID, workID,
		fmt.Sprintf("status_change_%s_to_%s", currentStatus, req.Status),
		req.Reason, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log moderation action"})
		return
	}

	// Send notification to work author (if not deleted)
	if req.Status != "deleted" {
		notificationID := uuid.New()
		_, err = tx.Exec(`
			INSERT INTO notifications (id, user_id, type, title, message, data, created_at)
			VALUES ($1, $2, 'moderator_action', $3, $4, $5, $6)`,
			notificationID, authorID,
			fmt.Sprintf("Work Status Changed: %s", workTitle),
			fmt.Sprintf("Your work '%s' status has been changed to '%s' by a moderator. Reason: %s",
				workTitle, req.Status, req.Reason),
			fmt.Sprintf(`{"work_id": "%s", "old_status": "%s", "new_status": "%s"}`,
				workID, currentStatus, req.Status),
			now)

		if err != nil {
			// Don't fail the transaction for notification errors
			log.Printf("Failed to create notification: %v", err)
		}
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Work status updated successfully",
		"work_id":      workID,
		"old_status":   currentStatus,
		"new_status":   req.Status,
		"moderator_id": userID,
	})
}

func (ws *WorkService) AdminDeleteWork(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user has admin privileges (only admins can permanently delete)
	var role string
	err := ws.db.QueryRow(`
		SELECT COALESCE(role, 'user') FROM users WHERE id = $1`, userID).Scan(&role)

	if err != nil || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required for permanent deletion"})
		return
	}

	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	var req struct {
		Reason  string `json:"reason" validate:"required"`  // Deletion reason is required
		Confirm bool   `json:"confirm" validate:"required"` // Explicit confirmation required
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	if !req.Confirm {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Explicit confirmation required for deletion"})
		return
	}

	// Get work details before deletion
	var workTitle string
	var authorID uuid.UUID
	var workStatus string
	err = ws.db.QueryRow(`
		SELECT title, user_id, status FROM works WHERE id = $1`, workID).Scan(
		&workTitle, &authorID, &workStatus)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch work"})
		return
	}

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Log deletion action before deleting
	moderationLogID := uuid.New()
	now := time.Now()
	_, err = tx.Exec(`
		INSERT INTO moderation_logs (id, moderator_id, target_type, target_id, action, reason, metadata, created_at)
		VALUES ($1, $2, 'work', $3, 'permanent_deletion', $4, $5, $6)`,
		moderationLogID, userID, workID, req.Reason,
		fmt.Sprintf(`{"work_title": "%s", "author_id": "%s", "original_status": "%s"}`,
			workTitle, authorID, workStatus),
		now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log deletion action"})
		return
	}

	// Delete related data in correct order (respecting foreign key constraints)
	// Only include tables that actually exist in the current schema
	deletionOrder := []string{
		"collection_works", // Remove from collections
		"kudos",            // Remove kudos
		"comments",         // Remove comments
		"bookmarks",        // Remove bookmarks
		"subscriptions",    // Remove subscriptions (WHERE subscribable_type = 'Work')
		"work_statistics",  // Remove statistics
		"chapters",         // Remove chapters
		"gifts",            // Remove gift associations
		"creatorships",     // Remove authorship (WHERE creation_type = 'Work')
		"works",            // Finally remove the work itself
	}

	for _, table := range deletionOrder {
		var query string
		if table == "creatorships" {
			query = fmt.Sprintf("DELETE FROM %s WHERE creation_id = $1 AND creation_type = 'Work'", table)
		} else if table == "subscriptions" {
			query = fmt.Sprintf("DELETE FROM %s WHERE target_id = $1 AND type = 'work'", table)
		} else if table == "works" {
			query = fmt.Sprintf("DELETE FROM %s WHERE id = $1", table)
		} else {
			query = fmt.Sprintf("DELETE FROM %s WHERE work_id = $1", table)
		}

		_, err = tx.Exec(query, workID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   fmt.Sprintf("Failed to delete from %s", table),
				"details": err.Error(),
			})
			return
		}
	}

	// Send notification to author
	notificationID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO notifications (id, user_id, type, title, message, data, created_at)
		VALUES ($1, $2, 'moderator_action', $3, $4, $5, $6)`,
		notificationID, authorID,
		fmt.Sprintf("Work Permanently Deleted: %s", workTitle),
		fmt.Sprintf("Your work '%s' has been permanently deleted by an administrator. Reason: %s",
			workTitle, req.Reason),
		fmt.Sprintf(`{"work_id": "%s", "deletion_reason": "%s"}`, workID, req.Reason),
		now)

	if err != nil {
		// Don't fail the transaction for notification errors
		log.Printf("Failed to create deletion notification: %v", err)
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit deletion"})
		return
	}

	// Clear any cached data (if Redis is available)
	if ws.redis != nil {
		cacheKey := fmt.Sprintf("work:%s", workID)
		ws.redis.Del(c.Request.Context(), cacheKey)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Work permanently deleted",
		"work_id":         workID,
		"work_title":      workTitle,
		"deletion_reason": req.Reason,
		"deleted_by":      userID,
		"deleted_at":      now,
	})
}

func (ws *WorkService) AdminListComments(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user has moderator or admin privileges
	var role string
	err := ws.db.QueryRow(`
		SELECT COALESCE(role, 'user') FROM users WHERE id = $1`, userID).Scan(&role)

	if err != nil || (role != "moderator" && role != "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderator or admin access required"})
		return
	}

	// Parse query parameters
	status := c.Query("status") // e.g., "pending_moderation", "flagged", "published", "hidden"
	workID := c.Query("work_id")
	userFilter := c.Query("user_id")

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 50 // Higher limit for admin interface
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Build query for admin comment listing
	baseQuery := `
		SELECT c.id, c.work_id, c.chapter_id, c.user_id, c.parent_comment_id, c.content,
			c.status, c.is_anonymous, c.created_at, c.updated_at,
			w.title as work_title, u.username, u.email,
			COALESCE(reporter_count.count, 0) as report_count
		FROM comments c
		JOIN works w ON c.work_id = w.id
		LEFT JOIN users u ON c.user_id = u.id
		LEFT JOIN (
			SELECT target_id, COUNT(*) as count 
			FROM reports 
			WHERE target_type = 'comment' AND status = 'pending'
			GROUP BY target_id
		) reporter_count ON c.id = reporter_count.target_id
		WHERE 1=1`

	args := []interface{}{}
	argIndex := 0

	// Add filters
	if status != "" {
		argIndex++
		baseQuery += fmt.Sprintf(" AND c.status = $%d", argIndex)
		args = append(args, status)
	}

	if workID != "" {
		if workUUID, parseErr := uuid.Parse(workID); parseErr == nil {
			argIndex++
			baseQuery += fmt.Sprintf(" AND c.work_id = $%d", argIndex)
			args = append(args, workUUID)
		}
	}

	if userFilter != "" {
		if userUUID, parseErr := uuid.Parse(userFilter); parseErr == nil {
			argIndex++
			baseQuery += fmt.Sprintf(" AND c.user_id = $%d", argIndex)
			args = append(args, userUUID)
		}
	}

	// Order by most recent first, prioritizing reported comments
	baseQuery += " ORDER BY report_count DESC, c.created_at DESC"

	// Add pagination
	argIndex++
	baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, limit)

	argIndex++
	baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
	args = append(args, offset)

	rows, err := ws.db.Query(baseQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments", "details": err.Error()})
		return
	}
	defer rows.Close()

	comments := []gin.H{}
	for rows.Next() {
		var comment models.WorkComment
		var workTitle, username, email string
		var reportCount int
		var chapterID, parentID sql.NullString

		err := rows.Scan(
			&comment.ID, &comment.WorkID, &chapterID, &comment.UserID, &parentID,
			&comment.Content, &comment.Status, &comment.IsAnonymous,
			&comment.CreatedAt, &comment.UpdatedAt, &workTitle, &username, &email, &reportCount)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan comment"})
			return
		}

		// Handle nullable fields
		if chapterID.Valid {
			if chapterUUID, parseErr := uuid.Parse(chapterID.String); parseErr == nil {
				comment.ChapterID = &chapterUUID
			}
		}
		if parentID.Valid {
			if parentUUID, parseErr := uuid.Parse(parentID.String); parseErr == nil {
				comment.ParentID = &parentUUID
			}
		}

		// Get recent reports for this comment
		var reports []gin.H
		if reportCount > 0 {
			reportRows, reportErr := ws.db.Query(`
				SELECT r.id, r.reporter_id, r.reason, r.description, r.created_at, u.username as reporter_username
				FROM reports r
				LEFT JOIN users u ON r.reporter_id = u.id
				WHERE r.target_type = 'comment' AND r.target_id = $1 AND r.status = 'pending'
				ORDER BY r.created_at DESC
				LIMIT 5`, comment.ID)

			if reportErr == nil {
				defer reportRows.Close()
				for reportRows.Next() {
					var report gin.H = gin.H{}
					var reportID, reporterID uuid.UUID
					var reason, description, reporterUsername string
					var reportCreatedAt time.Time

					reportRows.Scan(&reportID, &reporterID, &reason, &description, &reportCreatedAt, &reporterUsername)
					report["id"] = reportID
					report["reporter_id"] = reporterID
					report["reporter_username"] = reporterUsername
					report["reason"] = reason
					report["description"] = description
					report["created_at"] = reportCreatedAt

					reports = append(reports, report)
				}
			}
		}

		commentData := gin.H{
			"comment":      comment,
			"work_title":   workTitle,
			"username":     username,
			"user_email":   email,
			"report_count": reportCount,
			"reports":      reports,
		}

		comments = append(comments, commentData)
	}

	// Get total count for pagination
	countQuery := `
		SELECT COUNT(*) 
		FROM comments c
		JOIN works w ON c.work_id = w.id
		WHERE 1=1`

	countArgs := []interface{}{}
	countArgIndex := 0

	// Apply same filters to count query
	if status != "" {
		countArgIndex++
		countQuery += fmt.Sprintf(" AND c.status = $%d", countArgIndex)
		countArgs = append(countArgs, status)
	}

	if workID != "" {
		if workUUID, parseErr := uuid.Parse(workID); parseErr == nil {
			countArgIndex++
			countQuery += fmt.Sprintf(" AND c.work_id = $%d", countArgIndex)
			countArgs = append(countArgs, workUUID)
		}
	}

	if userFilter != "" {
		if userUUID, parseErr := uuid.Parse(userFilter); parseErr == nil {
			countArgIndex++
			countQuery += fmt.Sprintf(" AND c.user_id = $%d", countArgIndex)
			countArgs = append(countArgs, userUUID)
		}
	}

	var total int
	err = ws.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		total = len(comments) // Fallback
	}

	// Get status counts for admin dashboard
	statusCounts := gin.H{}
	statusRows, err := ws.db.Query(`
		SELECT status, COUNT(*) 
		FROM comments 
		GROUP BY status`)

	if err == nil {
		defer statusRows.Close()
		for statusRows.Next() {
			var status string
			var count int
			if err := statusRows.Scan(&status, &count); err == nil {
				statusCounts[status] = count
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
		"status_counts": statusCounts,
		"filters_applied": gin.H{
			"status":  status,
			"work_id": workID,
			"user_id": userFilter,
		},
	})
}

func (ws *WorkService) AdminUpdateCommentStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user has moderator or admin privileges
	var role string
	err := ws.db.QueryRow(`
		SELECT COALESCE(role, 'user') FROM users WHERE id = $1`, userID).Scan(&role)

	if err != nil || (role != "moderator" && role != "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderator or admin access required"})
		return
	}

	commentID, err := uuid.Parse(c.Param("comment_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	var req struct {
		Status         string `json:"status" validate:"required,oneof=published hidden flagged pending_moderation"`
		Reason         string `json:"reason"`          // Moderation reason
		ResolveReports bool   `json:"resolve_reports"` // Whether to resolve pending reports
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Get current comment details
	var currentStatus string
	var authorID uuid.UUID
	var workID uuid.UUID
	var content string
	err = ws.db.QueryRow(`
		SELECT status, user_id, work_id, SUBSTRING(content FROM 1 FOR 100) || 
		CASE WHEN LENGTH(content) > 100 THEN '...' ELSE '' END
		FROM comments WHERE id = $1`, commentID).Scan(
		&currentStatus, &authorID, &workID, &content)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comment"})
		return
	}

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Update comment status
	now := time.Now()
	_, err = tx.Exec(`
		UPDATE comments 
		SET status = $1, updated_at = $2
		WHERE id = $3`, req.Status, now, commentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update comment status"})
		return
	}

	// Log moderation action
	moderationLogID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO moderation_logs (id, moderator_id, target_type, target_id, action, reason, metadata, created_at)
		VALUES ($1, $2, 'comment', $3, $4, $5, $6, $7)`,
		moderationLogID, userID, commentID,
		fmt.Sprintf("status_change_%s_to_%s", currentStatus, req.Status),
		req.Reason,
		fmt.Sprintf(`{"work_id": "%s", "comment_preview": "%s"}`, workID, content),
		now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log moderation action"})
		return
	}

	// Resolve pending reports if requested
	var resolvedReports int
	if req.ResolveReports {
		result, err := tx.Exec(`
			UPDATE reports 
			SET status = 'resolved', resolved_by = $1, resolved_at = $2, resolution = $3
			WHERE target_type = 'comment' AND target_id = $4 AND status = 'pending'`,
			userID, now, fmt.Sprintf("Comment status changed to %s", req.Status), commentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve reports"})
			return
		}

		if rowsAffected, err := result.RowsAffected(); err == nil {
			resolvedReports = int(rowsAffected)
		}
	}

	// Send notification to comment author (if not the same as moderator)
	if authorID.String() != userID.(string) {
		var notificationTitle, notificationMessage string

		switch req.Status {
		case "hidden":
			notificationTitle = "Comment Hidden"
			notificationMessage = fmt.Sprintf("Your comment has been hidden by a moderator. Reason: %s", req.Reason)
		case "flagged":
			notificationTitle = "Comment Flagged"
			notificationMessage = fmt.Sprintf("Your comment has been flagged for review. Reason: %s", req.Reason)
		case "published":
			notificationTitle = "Comment Approved"
			notificationMessage = "Your comment has been approved and is now visible."
		case "pending_moderation":
			notificationTitle = "Comment Under Review"
			notificationMessage = "Your comment is under moderation review."
		}

		if notificationTitle != "" {
			notificationID := uuid.New()
			_, err = tx.Exec(`
				INSERT INTO notifications (id, user_id, type, title, message, data, created_at)
				VALUES ($1, $2, 'moderator_action', $3, $4, $5, $6)`,
				notificationID, authorID, notificationTitle, notificationMessage,
				fmt.Sprintf(`{"comment_id": "%s", "work_id": "%s", "new_status": "%s"}`,
					commentID, workID, req.Status),
				now)

			if err != nil {
				// Don't fail the transaction for notification errors
				log.Printf("Failed to create notification: %v", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Comment status updated successfully",
		"comment_id":       commentID,
		"old_status":       currentStatus,
		"new_status":       req.Status,
		"moderator_id":     userID,
		"resolved_reports": resolvedReports,
	})
}

func (ws *WorkService) AdminDeleteComment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user has moderator or admin privileges
	var role string
	err := ws.db.QueryRow(`
		SELECT COALESCE(role, 'user') FROM users WHERE id = $1`, userID).Scan(&role)

	if err != nil || (role != "moderator" && role != "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderator or admin access required"})
		return
	}

	commentID, err := uuid.Parse(c.Param("comment_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	var req struct {
		Reason        string `json:"reason" validate:"required"` // Deletion reason is required
		DeleteReplies bool   `json:"delete_replies"`             // Whether to delete child comments too
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Get comment details before deletion
	var authorID uuid.UUID
	var workID uuid.UUID
	var content string
	var parentID sql.NullString
	err = ws.db.QueryRow(`
		SELECT user_id, work_id, parent_comment_id,
		SUBSTRING(content FROM 1 FOR 100) || CASE WHEN LENGTH(content) > 100 THEN '...' ELSE '' END
		FROM comments WHERE id = $1`, commentID).Scan(
		&authorID, &workID, &parentID, &content)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comment"})
		return
	}

	// Check for child comments
	var childCount int
	err = ws.db.QueryRow(`
		SELECT COUNT(*) FROM comments WHERE parent_comment_id = $1`, commentID).Scan(&childCount)
	if err != nil {
		childCount = 0
	}

	if childCount > 0 && !req.DeleteReplies {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "Comment has replies. Set delete_replies=true to delete all replies as well.",
			"child_count": childCount,
		})
		return
	}

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	now := time.Now()
	var deletedCommentIDs []uuid.UUID

	// If deleting replies, collect all child comment IDs recursively
	if req.DeleteReplies && childCount > 0 {
		// Get all descendant comments recursively
		rows, err := tx.Query(`
			WITH RECURSIVE comment_tree AS (
				SELECT id, parent_comment_id, user_id
				FROM comments 
				WHERE parent_comment_id = $1
				UNION ALL
				SELECT c.id, c.parent_comment_id, c.user_id
				FROM comments c
				JOIN comment_tree ct ON c.parent_comment_id = ct.id
			)
			SELECT id, user_id FROM comment_tree`, commentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find child comments"})
			return
		}
		defer rows.Close()

		for rows.Next() {
			var childID, childAuthorID uuid.UUID
			if err := rows.Scan(&childID, &childAuthorID); err == nil {
				deletedCommentIDs = append(deletedCommentIDs, childID)
			}
		}
	}

	// Add the main comment to deletion list
	deletedCommentIDs = append(deletedCommentIDs, commentID)

	// Log deletion action for each comment
	for _, delCommentID := range deletedCommentIDs {
		moderationLogID := uuid.New()
		_, err = tx.Exec(`
			INSERT INTO moderation_logs (id, moderator_id, target_type, target_id, action, reason, metadata, created_at)
			VALUES ($1, $2, 'comment', $3, 'permanent_deletion', $4, $5, $6)`,
			moderationLogID, userID, delCommentID, req.Reason,
			fmt.Sprintf(`{"work_id": "%s", "is_child_comment": %t}`,
				workID, delCommentID != commentID),
			now)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log deletion action"})
			return
		}
	}

	// Delete all comments (children first, then parent)
	for i := len(deletedCommentIDs) - 1; i >= 0; i-- {
		delCommentID := deletedCommentIDs[i]

		// Delete comment (this will also cascade to any notifications, etc.)
		_, err = tx.Exec("DELETE FROM comments WHERE id = $1", delCommentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":      "Failed to delete comment",
				"comment_id": delCommentID,
				"details":    err.Error(),
			})
			return
		}
	}

	// Resolve any pending reports for all deleted comments
	var resolvedReports int
	for _, delCommentID := range deletedCommentIDs {
		result, err := tx.Exec(`
			UPDATE reports 
			SET status = 'resolved', resolved_by = $1, resolved_at = $2, resolution = 'Comment deleted by moderator'
			WHERE target_type = 'comment' AND target_id = $3 AND status = 'pending'`,
			userID, now, delCommentID)

		if err == nil {
			if rowsAffected, err := result.RowsAffected(); err == nil {
				resolvedReports += int(rowsAffected)
			}
		}
	}

	// Update work comment count
	_, err = tx.Exec(`
		UPDATE works 
		SET comment_count = (SELECT COUNT(*) FROM comments WHERE work_id = $1 AND status = 'published'),
			updated_at = $2
		WHERE id = $1`, workID, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update work comment count"})
		return
	}

	// Send notification to comment author (not for child comments to avoid spam)
	notificationID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO notifications (id, user_id, type, title, message, data, created_at)
		VALUES ($1, $2, 'moderator_action', $3, $4, $5, $6)`,
		notificationID, authorID,
		"Comment Deleted",
		fmt.Sprintf("Your comment has been deleted by a moderator. Reason: %s", req.Reason),
		fmt.Sprintf(`{"comment_id": "%s", "work_id": "%s", "deleted_replies": %d}`,
			commentID, workID, len(deletedCommentIDs)-1),
		now)

	if err != nil {
		// Don't fail the transaction for notification errors
		log.Printf("Failed to create deletion notification: %v", err)
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit deletion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             "Comment(s) deleted successfully",
		"deleted_comment_id":  commentID,
		"deleted_reply_count": len(deletedCommentIDs) - 1,
		"total_deleted":       len(deletedCommentIDs),
		"resolved_reports":    resolvedReports,
		"deletion_reason":     req.Reason,
		"deleted_by":          userID,
		"deleted_at":          now,
	})
}

func (ws *WorkService) AdminGetReports(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user has moderator or admin privileges
	var role string
	err := ws.db.QueryRow(`
		SELECT COALESCE(role, 'user') FROM users WHERE id = $1`, userID).Scan(&role)

	if err != nil || (role != "moderator" && role != "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Moderator or admin access required"})
		return
	}

	// Parse query parameters
	status := c.DefaultQuery("status", "pending") // pending, in_review, resolved, dismissed
	targetType := c.Query("target_type")          // work, comment, user
	reason := c.Query("reason")

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 25
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Build query for admin reports listing
	baseQuery := `
		SELECT r.id, r.target_type, r.target_id, r.reporter_id, r.reason, r.description,
			r.status, r.created_at, r.resolved_at, r.resolved_by, r.resolution,
			reporter.username as reporter_username, reporter.email as reporter_email,
			resolver.username as resolver_username
		FROM reports r
		LEFT JOIN users reporter ON r.reporter_id = reporter.id
		LEFT JOIN users resolver ON r.resolved_by = resolver.id
		WHERE 1=1`

	args := []interface{}{}
	argIndex := 0

	// Add filters
	if status != "" {
		argIndex++
		baseQuery += fmt.Sprintf(" AND r.status = $%d", argIndex)
		args = append(args, status)
	}

	if targetType != "" {
		argIndex++
		baseQuery += fmt.Sprintf(" AND r.target_type = $%d", argIndex)
		args = append(args, targetType)
	}

	if reason != "" {
		argIndex++
		baseQuery += fmt.Sprintf(" AND r.reason = $%d", argIndex)
		args = append(args, reason)
	}

	// Order by creation date, most recent first
	baseQuery += " ORDER BY r.created_at DESC"

	// Add pagination
	argIndex++
	baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, limit)

	argIndex++
	baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
	args = append(args, offset)

	rows, err := ws.db.Query(baseQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports", "details": err.Error()})
		return
	}
	defer rows.Close()

	reports := []gin.H{}
	for rows.Next() {
		var report gin.H = gin.H{}
		var reportID, targetID, reporterID uuid.UUID
		var targetType, reason, description, reportStatus string
		var createdAt time.Time
		var resolvedAt sql.NullTime
		var resolvedBy sql.NullString
		var resolution sql.NullString
		var reporterUsername, reporterEmail, resolverUsername sql.NullString

		err := rows.Scan(
			&reportID, &targetType, &targetID, &reporterID, &reason, &description,
			&reportStatus, &createdAt, &resolvedAt, &resolvedBy, &resolution,
			&reporterUsername, &reporterEmail, &resolverUsername)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan report"})
			return
		}

		report["id"] = reportID
		report["target_type"] = targetType
		report["target_id"] = targetID
		report["reporter_id"] = reporterID
		report["reason"] = reason
		report["description"] = description
		report["status"] = reportStatus
		report["created_at"] = createdAt

		if reporterUsername.Valid {
			report["reporter_username"] = reporterUsername.String
		}
		if reporterEmail.Valid {
			report["reporter_email"] = reporterEmail.String
		}
		if resolvedAt.Valid {
			report["resolved_at"] = resolvedAt.Time
		}
		if resolvedBy.Valid {
			if resolvedByID, parseErr := uuid.Parse(resolvedBy.String); parseErr == nil {
				report["resolved_by"] = resolvedByID
			}
		}
		if resolution.Valid {
			report["resolution"] = resolution.String
		}
		if resolverUsername.Valid {
			report["resolver_username"] = resolverUsername.String
		}

		// Get target content details based on type
		var targetDetails gin.H = gin.H{}
		switch targetType {
		case "work":
			var workTitle, workStatus string
			var authorID uuid.UUID
			err = ws.db.QueryRow(`
				SELECT title, status, user_id FROM works WHERE id = $1`, targetID).Scan(
				&workTitle, &workStatus, &authorID)
			if err == nil {
				targetDetails["title"] = workTitle
				targetDetails["status"] = workStatus
				targetDetails["author_id"] = authorID
			}

		case "comment":
			var commentContent, commentStatus string
			var commentAuthorID, workID uuid.UUID
			err = ws.db.QueryRow(`
				SELECT SUBSTRING(content FROM 1 FOR 100) || CASE WHEN LENGTH(content) > 100 THEN '...' ELSE '' END,
				status, user_id, work_id FROM comments WHERE id = $1`, targetID).Scan(
				&commentContent, &commentStatus, &commentAuthorID, &workID)
			if err == nil {
				targetDetails["content_preview"] = commentContent
				targetDetails["status"] = commentStatus
				targetDetails["author_id"] = commentAuthorID
				targetDetails["work_id"] = workID
			}

		case "user":
			var username, userStatus string
			err = ws.db.QueryRow(`
				SELECT username, COALESCE(status, 'active') FROM users WHERE id = $1`, targetID).Scan(
				&username, &userStatus)
			if err == nil {
				targetDetails["username"] = username
				targetDetails["status"] = userStatus
			}
		}

		report["target_details"] = targetDetails
		reports = append(reports, report)
	}

	// Get total count for pagination
	countQuery := `SELECT COUNT(*) FROM reports r WHERE 1=1`
	countArgs := []interface{}{}
	countArgIndex := 0

	// Apply same filters to count query
	if status != "" {
		countArgIndex++
		countQuery += fmt.Sprintf(" AND r.status = $%d", countArgIndex)
		countArgs = append(countArgs, status)
	}

	if targetType != "" {
		countArgIndex++
		countQuery += fmt.Sprintf(" AND r.target_type = $%d", countArgIndex)
		countArgs = append(countArgs, targetType)
	}

	if reason != "" {
		countArgIndex++
		countQuery += fmt.Sprintf(" AND r.reason = $%d", countArgIndex)
		countArgs = append(countArgs, reason)
	}

	var total int
	err = ws.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		total = len(reports) // Fallback
	}

	// Get status counts for admin dashboard
	statusCounts := gin.H{}
	statusRows, err := ws.db.Query(`
		SELECT status, COUNT(*) 
		FROM reports 
		GROUP BY status`)

	if err == nil {
		defer statusRows.Close()
		for statusRows.Next() {
			var status string
			var count int
			if err := statusRows.Scan(&status, &count); err == nil {
				statusCounts[status] = count
			}
		}
	}

	// Get reason counts for filtering
	reasonCounts := gin.H{}
	reasonRows, err := ws.db.Query(`
		SELECT reason, COUNT(*) 
		FROM reports 
		WHERE status = 'pending'
		GROUP BY reason
		ORDER BY COUNT(*) DESC`)

	if err == nil {
		defer reasonRows.Close()
		for reasonRows.Next() {
			var reason string
			var count int
			if err := reasonRows.Scan(&reason, &count); err == nil {
				reasonCounts[reason] = count
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"reports": reports,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
		"status_counts": statusCounts,
		"reason_counts": reasonCounts,
		"filters_applied": gin.H{
			"status":      status,
			"target_type": targetType,
			"reason":      reason,
		},
	})
}

func (ws *WorkService) AdminGetStatistics(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user has admin privileges
	var isAdmin bool
	err := ws.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM users 
			WHERE id = $1 AND (role = 'admin' OR role = 'superadmin')
		)`, userID).Scan(&isAdmin)

	if err != nil || !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	// Get comprehensive admin statistics
	var stats struct {
		// Work statistics
		TotalWorks     int `json:"total_works"`
		PublishedWorks int `json:"published_works"`
		DraftWorks     int `json:"draft_works"`
		CompleteWorks  int `json:"complete_works"`
		WorksThisMonth int `json:"works_this_month"`
		WorksToday     int `json:"works_today"`

		// User statistics
		TotalUsers        int `json:"total_users"`
		ActiveUsers       int `json:"active_users"`
		NewUsersThisMonth int `json:"new_users_this_month"`
		NewUsersToday     int `json:"new_users_today"`

		// Content statistics
		TotalChapters    int `json:"total_chapters"`
		TotalWordCount   int `json:"total_word_count"`
		AverageWordCount int `json:"average_word_count"`

		// Engagement statistics
		TotalHits          int `json:"total_hits"`
		TotalKudos         int `json:"total_kudos"`
		TotalComments      int `json:"total_comments"`
		TotalBookmarks     int `json:"total_bookmarks"`
		TotalSubscriptions int `json:"total_subscriptions"`

		// Collections and series
		TotalSeries      int `json:"total_series"`
		TotalCollections int `json:"total_collections"`

		// Moderation statistics
		ReportsToReview      int `json:"reports_to_review"`
		CommentsInModeration int `json:"comments_in_moderation"`

		// System health
		DatabaseConnections int     `json:"database_connections"`
		CacheHitRate        float64 `json:"cache_hit_rate,omitempty"`
	}

	// Get work statistics
	err = ws.db.QueryRow(`
		SELECT 
			COUNT(*) as total_works,
			COUNT(CASE WHEN status = 'posted' THEN 1 END) as published_works,
			COUNT(CASE WHEN status = 'draft' THEN 1 END) as draft_works,
			COUNT(CASE WHEN is_complete = true THEN 1 END) as complete_works,
			COUNT(CASE WHEN created_at >= date_trunc('month', CURRENT_DATE) THEN 1 END) as works_this_month,
			COUNT(CASE WHEN created_at >= CURRENT_DATE THEN 1 END) as works_today
		FROM works`).Scan(
		&stats.TotalWorks, &stats.PublishedWorks, &stats.DraftWorks,
		&stats.CompleteWorks, &stats.WorksThisMonth, &stats.WorksToday)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch work statistics"})
		return
	}

	// Get user statistics
	err = ws.db.QueryRow(`
		SELECT 
			COUNT(*) as total_users,
			COUNT(CASE WHEN last_seen_at >= CURRENT_DATE - INTERVAL '30 days' THEN 1 END) as active_users,
			COUNT(CASE WHEN created_at >= date_trunc('month', CURRENT_DATE) THEN 1 END) as new_users_this_month,
			COUNT(CASE WHEN created_at >= CURRENT_DATE THEN 1 END) as new_users_today
		FROM users`).Scan(
		&stats.TotalUsers, &stats.ActiveUsers, &stats.NewUsersThisMonth, &stats.NewUsersToday)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user statistics"})
		return
	}

	// Get content statistics
	err = ws.db.QueryRow(`
		SELECT 
			COALESCE(SUM(chapter_count), 0) as total_chapters,
			COALESCE(SUM(word_count), 0) as total_word_count,
			COALESCE(AVG(word_count)::int, 0) as average_word_count
		FROM works WHERE status = 'posted'`).Scan(
		&stats.TotalChapters, &stats.TotalWordCount, &stats.AverageWordCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch content statistics"})
		return
	}

	// Get engagement statistics
	err = ws.db.QueryRow(`
		SELECT 
			COALESCE(SUM(hit_count), 0) as total_hits,
			COALESCE(SUM(kudos_count), 0) as total_kudos,
			COALESCE(SUM(comment_count), 0) as total_comments,
			COALESCE(SUM(bookmark_count), 0) as total_bookmarks
		FROM works WHERE status = 'posted'`).Scan(
		&stats.TotalHits, &stats.TotalKudos, &stats.TotalComments, &stats.TotalBookmarks)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch engagement statistics"})
		return
	}

	// Get subscription count
	err = ws.db.QueryRow(`SELECT COUNT(*) FROM subscriptions`).Scan(&stats.TotalSubscriptions)
	if err != nil {
		stats.TotalSubscriptions = 0
	}

	// Get series and collection counts
	err = ws.db.QueryRow(`SELECT COUNT(*) FROM series`).Scan(&stats.TotalSeries)
	if err != nil {
		stats.TotalSeries = 0
	}

	err = ws.db.QueryRow(`SELECT COUNT(*) FROM collections`).Scan(&stats.TotalCollections)
	if err != nil {
		stats.TotalCollections = 0
	}

	// Get moderation statistics
	err = ws.db.QueryRow(`
		SELECT COUNT(*) FROM reports WHERE status = 'pending'`).Scan(&stats.ReportsToReview)
	if err != nil {
		stats.ReportsToReview = 0
	}

	err = ws.db.QueryRow(`
		SELECT COUNT(*) FROM comments WHERE status = 'pending_moderation'`).Scan(&stats.CommentsInModeration)
	if err != nil {
		stats.CommentsInModeration = 0
	}

	// Get database connection count
	err = ws.db.QueryRow(`
		SELECT count(*) FROM pg_stat_activity WHERE state = 'active'`).Scan(&stats.DatabaseConnections)
	if err != nil {
		stats.DatabaseConnections = 0
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// Subscription handlers

// CreateSubscription creates a new subscription for a user
func (ws *WorkService) CreateSubscription(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Type         string   `json:"type" binding:"required"`
		TargetID     string   `json:"target_id" binding:"required"`
		TargetName   string   `json:"target_name"`
		Events       []string `json:"events"`
		Frequency    string   `json:"frequency"`
		FilterTags   []string `json:"filter_tags"`
		FilterRating []string `json:"filter_rating"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Validate target ID
	targetUUID, err := uuid.Parse(req.TargetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target ID format"})
		return
	}

	// Validate subscription type
	validTypes := []string{"work", "author", "series", "tag", "collection"}
	isValidType := false
	for _, validType := range validTypes {
		if req.Type == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription type"})
		return
	}

	// Set default values
	if req.Frequency == "" {
		req.Frequency = "immediate"
	}
	if len(req.Events) == 0 {
		switch req.Type {
		case "work":
			req.Events = []string{"work_updated", "new_work"}
		case "author":
			req.Events = []string{"new_work", "work_updated"}
		case "series":
			req.Events = []string{"series_updated", "work_updated"}
		default:
			req.Events = []string{"new_work"}
		}
	}

	// Get target name if not provided
	if req.TargetName == "" {
		switch req.Type {
		case "work":
			ws.db.QueryRow("SELECT title FROM works WHERE id = $1", targetUUID).Scan(&req.TargetName)
		case "author":
			ws.db.QueryRow("SELECT username FROM users WHERE id = $1", targetUUID).Scan(&req.TargetName)
		case "series":
			ws.db.QueryRow("SELECT title FROM series WHERE id = $1", targetUUID).Scan(&req.TargetName)
		}
	}

	// Create subscription
	subscriptionID := uuid.New()
	_, err = ws.db.Exec(`
		INSERT INTO subscriptions (
			id, user_id, type, target_id, target_name, events, frequency, 
			filter_tags, filter_rating, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, NOW(), NOW())
		ON CONFLICT (user_id, type, target_id) 
		DO UPDATE SET 
			events = $6, frequency = $7, filter_tags = $8, filter_rating = $9,
			is_active = true, updated_at = NOW()`,
		subscriptionID, userID, req.Type, targetUUID, req.TargetName,
		pq.Array(req.Events), req.Frequency, pq.Array(req.FilterTags), pq.Array(req.FilterRating))

	if err != nil {
		log.Printf("Error creating subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Subscription created successfully",
		"subscription": gin.H{
			"id":          subscriptionID,
			"type":        req.Type,
			"target_id":   req.TargetID,
			"target_name": req.TargetName,
			"events":      req.Events,
			"frequency":   req.Frequency,
		},
	})
}

// GetUserSubscriptions retrieves all subscriptions for a user
func (ws *WorkService) GetUserSubscriptions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	rows, err := ws.db.Query(`
		SELECT id, type, target_id, target_name, events, frequency, 
			   filter_tags, filter_rating, is_active, created_at, updated_at
		FROM subscriptions 
		WHERE user_id = $1 AND is_active = true
		ORDER BY created_at DESC`, userID)

	if err != nil {
		log.Printf("Error fetching subscriptions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subscriptions"})
		return
	}
	defer rows.Close()

	var subscriptions []map[string]interface{}
	for rows.Next() {
		var sub struct {
			ID           uuid.UUID      `db:"id"`
			Type         string         `db:"type"`
			TargetID     uuid.UUID      `db:"target_id"`
			TargetName   string         `db:"target_name"`
			Events       pq.StringArray `db:"events"`
			Frequency    string         `db:"frequency"`
			FilterTags   pq.StringArray `db:"filter_tags"`
			FilterRating pq.StringArray `db:"filter_rating"`
			IsActive     bool           `db:"is_active"`
			CreatedAt    time.Time      `db:"created_at"`
			UpdatedAt    time.Time      `db:"updated_at"`
		}

		err := rows.Scan(&sub.ID, &sub.Type, &sub.TargetID, &sub.TargetName,
			&sub.Events, &sub.Frequency, &sub.FilterTags, &sub.FilterRating,
			&sub.IsActive, &sub.CreatedAt, &sub.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning subscription: %v", err)
			continue
		}

		subscriptions = append(subscriptions, map[string]interface{}{
			"id":            sub.ID,
			"type":          sub.Type,
			"target_id":     sub.TargetID,
			"target_name":   sub.TargetName,
			"events":        []string(sub.Events),
			"frequency":     sub.Frequency,
			"filter_tags":   []string(sub.FilterTags),
			"filter_rating": []string(sub.FilterRating),
			"is_active":     sub.IsActive,
			"created_at":    sub.CreatedAt,
			"updated_at":    sub.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"subscriptions": subscriptions})
}

// CheckSubscriptionStatus checks if user is subscribed to a specific target
func (ws *WorkService) CheckSubscriptionStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	targetType := c.Query("type")
	targetID := c.Query("target_id")

	if targetType == "" || targetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type and target_id parameters required"})
		return
	}

	targetUUID, err := uuid.Parse(targetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target ID format"})
		return
	}

	var subscriptionID uuid.UUID
	var isActive bool
	err = ws.db.QueryRow(`
		SELECT id, is_active FROM subscriptions 
		WHERE user_id = $1 AND type = $2 AND target_id = $3`,
		userID, targetType, targetUUID).Scan(&subscriptionID, &isActive)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{
				"subscribed":      false,
				"subscription_id": nil,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check subscription status"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscribed":      isActive,
		"subscription_id": subscriptionID,
	})
}

// UpdateSubscription updates an existing subscription
func (ws *WorkService) UpdateSubscription(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	subscriptionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}

	var req struct {
		Events       []string `json:"events"`
		Frequency    string   `json:"frequency"`
		FilterTags   []string `json:"filter_tags"`
		FilterRating []string `json:"filter_rating"`
		IsActive     *bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Verify subscription belongs to user
	var ownerID uuid.UUID
	err = ws.db.QueryRow("SELECT user_id FROM subscriptions WHERE id = $1", subscriptionID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify subscription ownership"})
		}
		return
	}

	if ownerID.String() != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Build update query dynamically
	setParts := []string{}
	args := []interface{}{}
	argCount := 1

	if len(req.Events) > 0 {
		setParts = append(setParts, fmt.Sprintf("events = $%d", argCount))
		args = append(args, pq.Array(req.Events))
		argCount++
	}

	if req.Frequency != "" {
		setParts = append(setParts, fmt.Sprintf("frequency = $%d", argCount))
		args = append(args, req.Frequency)
		argCount++
	}

	if req.FilterTags != nil {
		setParts = append(setParts, fmt.Sprintf("filter_tags = $%d", argCount))
		args = append(args, pq.Array(req.FilterTags))
		argCount++
	}

	if req.FilterRating != nil {
		setParts = append(setParts, fmt.Sprintf("filter_rating = $%d", argCount))
		args = append(args, pq.Array(req.FilterRating))
		argCount++
	}

	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argCount))
		args = append(args, *req.IsActive)
		argCount++
	}

	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	setParts = append(setParts, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE subscriptions SET %s WHERE id = $%d", strings.Join(setParts, ", "), argCount)
	args = append(args, subscriptionID)

	_, err = ws.db.Exec(query, args...)
	if err != nil {
		log.Printf("Error updating subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription updated successfully"})
}

// DeleteSubscription removes a subscription
func (ws *WorkService) DeleteSubscription(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	subscriptionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}

	// Verify subscription belongs to user and delete
	result, err := ws.db.Exec("DELETE FROM subscriptions WHERE id = $1 AND user_id = $2", subscriptionID, userID)
	if err != nil {
		log.Printf("Error deleting subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete subscription"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found or access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription deleted successfully"})
}

// triggerWorkNotification sends a notification when a work is updated
func (ws *WorkService) triggerWorkNotification(ctx context.Context, workID uuid.UUID, eventType models.NotificationEvent, title, description string) {
	if ws.notificationService == nil {
		log.Printf("Notification service not initialized, skipping notification for work %s", workID)
		return
	}

	event := &notifications.EventData{
		Type:        eventType,
		SourceID:    workID,
		SourceType:  "work",
		Title:       title,
		Description: description,
		ActionURL:   fmt.Sprintf("/works/%s", workID),
		ActorID:     nil, // TODO: Get user ID from context
		ActorName:   "",  // TODO: Get username from context
		ExtraData:   make(map[string]interface{}),
	}

	if err := ws.notificationService.ProcessEvent(ctx, event); err != nil {
		log.Printf("Failed to process notification event for work %s: %v", workID, err)
	} else {
		log.Printf("Successfully triggered notification for work %s", workID)
	}
}
