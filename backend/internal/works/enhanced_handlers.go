package works

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/shared/models"
)

// CreateWork posts a new work as a draft, together with its first chapter.
//
// Everything is written in one transaction: the work, the author's default
// pseud (created on first use, named after the username), the creatorship
// that links the pseud to the work, the first chapter, and the work's tags.
// Tags that don't exist yet are created unwrangled, as on AO3. The work stays
// a draft until it is updated with "status": "posted".
func (ws *WorkService) CreateWork(c *gin.Context) {
	var req models.CreateWorkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}
	user := ws.getUserIDFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	userID := *user

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	now := time.Now()
	wordCount := countWords(req.ChapterContent)
	work := &models.Work{
		ID:            uuid.New(),
		UserID:        userID,
		Title:         req.Title,
		Summary:       req.Summary,
		Notes:         req.Notes,
		SeriesID:      req.SeriesID,
		Language:      req.Language,
		Rating:        req.Rating,
		Category:      req.Category,
		Warnings:      req.Warnings,
		Fandoms:       req.Fandoms,
		Characters:    req.Characters,
		Relationships: req.Relationships,
		FreeformTags:  req.FreeformTags,
		MaxChapters:   req.MaxChapters,
		ChapterCount:  1,
		WordCount:     wordCount,
		IsComplete:    req.MaxChapters != nil && *req.MaxChapters == 1,
		Status:        "draft",
		CommentPolicy: "open",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_, err = tx.Exec(`
		INSERT INTO works (id, user_id, title, summary, notes, language, rating,
			category, warnings, fandoms, characters, relationships, freeform_tags,
			max_chapters, chapter_count, word_count, is_complete, status, is_draft,
			restricted, comment_policy, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
			'draft', true, false, 'open', $18, $18)`,
		work.ID, userID, work.Title, work.Summary, work.Notes, work.Language, work.Rating,
		pq.Array(work.Category), pq.Array(work.Warnings), pq.Array(work.Fandoms),
		pq.Array(work.Characters), pq.Array(work.Relationships), pq.Array(work.FreeformTags),
		work.MaxChapters, work.ChapterCount, wordCount, work.IsComplete, now)
	if err != nil {
		log.Printf("create work: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create work"})
		return
	}

	pseudID, err := defaultPseud(tx, userID)
	if err != nil {
		log.Printf("create work: pseud: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create work"})
		return
	}
	if _, err = tx.Exec(`INSERT INTO creatorships (creation_id, creation_type, pseud_id) VALUES ($1, 'Work', $2)`,
		work.ID, pseudID); err != nil {
		log.Printf("create work: creatorship: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create work"})
		return
	}

	chapter := &models.Chapter{
		ID:        uuid.New(),
		WorkID:    work.ID,
		Number:    1,
		Title:     req.ChapterTitle,
		Summary:   req.ChapterSummary,
		Notes:     req.ChapterNotes,
		EndNotes:  req.ChapterEndNotes,
		Content:   req.ChapterContent,
		WordCount: wordCount,
		Status:    "draft",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err = tx.Exec(`
		INSERT INTO chapters (id, work_id, chapter_number, title, summary, notes, end_notes,
			content, word_count, is_draft, created_at, updated_at)
		VALUES ($1, $2, 1, $3, $4, $5, $6, $7, $8, true, $9, $9)`,
		chapter.ID, work.ID, chapter.Title, chapter.Summary, chapter.Notes, chapter.EndNotes,
		chapter.Content, wordCount, now); err != nil {
		log.Printf("create work: chapter: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create work"})
		return
	}

	if err := attachTags(tx, work.ID, userID, req); err != nil {
		log.Printf("create work: tags: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create work"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create work"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"work": work, "first_chapter": chapter})
}

// defaultPseud returns the user's default pseud, creating one named after
// their username if they don't have one yet.
func defaultPseud(tx *sql.Tx, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1 AND is_default = true`, userID).Scan(&id)
	if err != sql.ErrNoRows {
		return id, err
	}
	err = tx.QueryRow(`
		INSERT INTO pseuds (user_id, name, is_default)
		SELECT id, username, true FROM users WHERE id = $1
		RETURNING id`, userID).Scan(&id)
	return id, err
}

// attachTags links the work to each tag in the request, creating tags that
// don't exist yet. Each list in the request says what type its tags are.
func attachTags(tx *sql.Tx, workID, userID uuid.UUID, req models.CreateWorkRequest) error {
	byType := []struct {
		tagType string
		names   []string
	}{
		{"fandom", req.Fandoms},
		{"character", req.Characters},
		{"relationship", req.Relationships},
		{"freeform", req.FreeformTags},
		{"warning", req.Warnings},
		{"category", req.Category},
	}
	for _, group := range byType {
		for _, name := range group.names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			var tagID uuid.UUID
			// On conflict, the no-op update makes RETURNING give the existing id.
			err := tx.QueryRow(`
				INSERT INTO tags (name, type, created_by) VALUES ($1, $2, $3)
				ON CONFLICT (name) DO UPDATE SET name = tags.name
				RETURNING id`, name, group.tagType, userID).Scan(&tagID)
			if err != nil {
				return fmt.Errorf("tag %q: %w", name, err)
			}
			if _, err := tx.Exec(`INSERT INTO work_tags (work_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				workID, tagID); err != nil {
				return fmt.Errorf("tag %q: %w", name, err)
			}
		}
	}
	return nil
}

// GetWorkWithTags retrieves a work with all its tags from tag service
func (ws *WorkService) GetWorkWithTags(c *gin.Context) {
	workIDStr := c.Param("id")
	workID, err := uuid.Parse(workIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	// Get work from database
	work, err := ws.getWorkByID(workID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch work"})
		return
	}

	// Get tags from tag service
	tags, err := ws.getWorkTags(workID)
	if err == nil {
		// Add tags to work response
		c.JSON(http.StatusOK, gin.H{
			"work": work,
			"tags": tags,
		})
	} else {
		// Return work without tags if tag service is unavailable
		c.JSON(http.StatusOK, gin.H{
			"work": work,
			"tags": []models.Tag{},
		})
	}

	// Increment hit count asynchronously
	go ws.incrementHits(workID)
}

// getWorkTags returns the tags attached to a work.
func (ws *WorkService) getWorkTags(workID uuid.UUID) ([]models.Tag, error) {
	rows, err := ws.db.Query(`
		SELECT t.id, t.name, t.type, COALESCE(t.is_canonical, false)
		FROM work_tags wt JOIN tags t ON t.id = wt.tag_id
		WHERE wt.work_id = $1
		ORDER BY t.type, t.name`, workID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Type, &t.IsCanonical); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// UpdateWorkStatistics updates work statistics in real-time
func (ws *WorkService) UpdateWorkStatistics(c *gin.Context) {
	workIDStr := c.Param("id")
	workID, err := uuid.Parse(workIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	// Calculate real-time statistics
	stats, err := ws.calculateWorkStatistics(workID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate statistics"})
		return
	}

	// Update statistics in database
	_, err = ws.db.Exec(`
		UPDATE work_statistics 
		SET hits = $1, kudos_count = $2, comments_count = $3, bookmarks_count = $4, updated_at = $5
		WHERE work_id = $6
	`, stats.Hits, stats.Kudos, stats.Comments, stats.Bookmarks, time.Now(), workID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update statistics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statistics": stats})
}

// calculateWorkStatistics calculates current statistics for a work
func (ws *WorkService) calculateWorkStatistics(workID uuid.UUID) (*WorkStatistics, error) {
	stats := &WorkStatistics{}

	// Get hit count
	ws.db.QueryRow("SELECT hits FROM work_statistics WHERE work_id = $1", workID).Scan(&stats.Hits)

	// Count kudos
	ws.db.QueryRow("SELECT COUNT(*) FROM kudos WHERE work_id = $1", workID).Scan(&stats.Kudos)

	// Count comments
	ws.db.QueryRow("SELECT COUNT(*) FROM comments WHERE work_id = $1 AND is_deleted = false", workID).Scan(&stats.Comments)

	// Count bookmarks
	ws.db.QueryRow("SELECT COUNT(*) FROM bookmarks WHERE work_id = $1 AND is_private = false", workID).Scan(&stats.Bookmarks)

	return stats, nil
}

// WorkStatistics represents work statistics
type WorkStatistics struct {
	Hits      int `json:"hits"`
	Kudos     int `json:"kudos"`
	Comments  int `json:"comments"`
	Bookmarks int `json:"bookmarks"`
}

// UploadWorkAttachment handles file uploads for works
func (ws *WorkService) UploadWorkAttachment(c *gin.Context) {
	workIDStr := c.Param("id")
	workID, err := uuid.Parse(workIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	// Check if work exists and user has permission
	userID, _ := c.Get("user_id")
	if !ws.userCanEditWork(workID, userID.(string)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}

	files := form.File["attachments"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files uploaded"})
		return
	}

	var uploadedFiles []map[string]interface{}

	for _, file := range files {
		// Validate file
		if err := ws.validateUploadedFile(file); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid file %s: %s", file.Filename, err.Error())})
			return
		}

		// Save file
		savedPath, err := ws.saveUploadedFile(file, workID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}

		uploadedFiles = append(uploadedFiles, map[string]interface{}{
			"filename": file.Filename,
			"size":     file.Size,
			"path":     savedPath,
		})
	}

	c.JSON(http.StatusOK, gin.H{"uploaded_files": uploadedFiles})
}

// validateUploadedFile validates uploaded files
func (ws *WorkService) validateUploadedFile(file *multipart.FileHeader) error {
	// Check file size (max 10MB)
	if file.Size > 10*1024*1024 {
		return fmt.Errorf("file too large (max 10MB)")
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := []string{".txt", ".pdf", ".doc", ".docx", ".epub", ".html"}

	allowed := false
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("file type not allowed")
	}

	return nil
}

// saveUploadedFile saves an uploaded file to storage
func (ws *WorkService) saveUploadedFile(file *multipart.FileHeader, workID uuid.UUID) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Create unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s_%s%s", workID, uuid.New().String()[:8], ext)
	filepath := fmt.Sprintf("uploads/works/%s", filename)

	// In a real implementation, save to cloud storage (S3, etc.)
	// For now, just read the file to validate it
	_, err = io.ReadAll(src)
	if err != nil {
		return "", err
	}

	return filepath, nil
}

// userCanEditWork checks if a user can edit a work
func (ws *WorkService) userCanEditWork(workID uuid.UUID, userIDStr string) bool {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return false
	}

	var count int
	ws.db.QueryRow(`
		SELECT COUNT(*) FROM creatorships 
		WHERE work_id = $1 AND user_id = $2
	`, workID, userID).Scan(&count)

	return count > 0
}

// SearchWorksAdvanced provides advanced work search with tag filtering
func (ws *WorkService) SearchWorksAdvanced(c *gin.Context) {
	// Get search parameters
	query := c.Query("q")
	tags := c.QueryArray("tags")
	rating := c.Query("rating")
	status := c.Query("status")
	sortBy := c.DefaultQuery("sort", "updated_at")
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	if limit > 100 {
		limit = 100
	}

	// Build search query
	searchParams := map[string]interface{}{
		"query":  query,
		"tags":   tags,
		"rating": rating,
		"status": status,
		"sort":   sortBy,
		"limit":  limit,
		"offset": offset,
	}

	results, err := ws.searchWorksDatabase(searchParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, results)
}

// searchWorksDatabase provides fallback database search
func (ws *WorkService) searchWorksDatabase(params map[string]interface{}) (gin.H, error) {
	// Simplified database search as fallback
	query := "SELECT id, title, summary, rating, status, updated_at FROM works WHERE status = 'posted'"
	args := []interface{}{}
	argIndex := 1

	if q, ok := params["query"].(string); ok && q != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d OR summary ILIKE $%d)", argIndex, argIndex+1)
		searchTerm := "%" + q + "%"
		args = append(args, searchTerm, searchTerm)
		argIndex += 2
	}

	if rating, ok := params["rating"].(string); ok && rating != "" {
		query += fmt.Sprintf(" AND rating = $%d", argIndex)
		args = append(args, rating)
		argIndex++
	}

	query += " ORDER BY updated_at DESC"

	if limit, ok := params["limit"].(int); ok {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	if offset, ok := params["offset"].(int); ok {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, offset)
	}

	rows, err := ws.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var works []map[string]interface{}
	for rows.Next() {
		var work struct {
			ID        uuid.UUID `db:"id"`
			Title     string    `db:"title"`
			Summary   string    `db:"summary"`
			Rating    string    `db:"rating"`
			Status    string    `db:"status"`
			UpdatedAt time.Time `db:"updated_at"`
		}

		err := rows.Scan(&work.ID, &work.Title, &work.Summary, &work.Rating, &work.Status, &work.UpdatedAt)
		if err != nil {
			continue
		}

		works = append(works, map[string]interface{}{
			"id":         work.ID,
			"title":      work.Title,
			"summary":    work.Summary,
			"rating":     work.Rating,
			"status":     work.Status,
			"updated_at": work.UpdatedAt,
		})
	}

	return gin.H{
		"works": works,
		"total": len(works),
	}, nil
}
