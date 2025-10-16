package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
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

// =============================================================================
// ENHANCED WORK SERVICE - Tag Integration & Advanced Features
// Integrates with Hecate Core Tag Service and Search Service
// =============================================================================

// TagService client for making requests to tag service
type TagServiceClient struct {
	baseURL string
	client  *http.Client
}

func NewTagServiceClient(baseURL string) *TagServiceClient {
	return &TagServiceClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// SearchServiceClient for making requests to search service
type SearchServiceClient struct {
	baseURL string
	client  *http.Client
}

func NewSearchServiceClient(baseURL string) *SearchServiceClient {
	return &SearchServiceClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// Enhanced CreateWork with tag integration
func (ws *WorkService) CreateWorkEnhanced(c *gin.Context) {
	log.Printf("DEBUG: Using ENHANCED CreateWork handler with automatic indexing")
	var req models.CreateWorkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Get user ID from JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse user ID to UUID
	userUUID, parseErr := uuid.Parse(userID.(string))
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	tx, err := ws.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Create work
	workID := uuid.New()
	now := time.Now()

	work := &models.Work{
		ID:                     workID,
		UserID:                 userUUID,
		Title:                  req.Title,
		Summary:                req.Summary,
		Notes:                  req.Notes,
		SeriesID:               req.SeriesID,
		Language:               req.Language,
		Rating:                 req.Rating,
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
		RestrictedToUsers:      false,
		CommentPolicy:          "open",
		ModerateComments:       false,
		DisableComments:        false,
		IsAnonymous:            false,
		InAnonCollection:       false,
		InUnrevealedCollection: false,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	// Insert work (series_id removed - handled via series_works table)
	query := `
		INSERT INTO works (id, user_id, title, summary, notes, language, rating, 
			category, warnings, fandoms, characters, relationships, freeform_tags, 
			max_chapters, chapter_count, is_complete, status, 
			restricted, comment_policy, moderate_comments, disable_comments,
			is_anonymous, in_anon_collection, in_unrevealed_collection,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)`

	_, err = tx.Exec(query,
		work.ID, work.UserID, work.Title, work.Summary, work.Notes,
		work.Language, work.Rating, pq.Array(work.Category), pq.Array(work.Warnings),
		pq.Array(work.Fandoms), pq.Array(work.Characters), pq.Array(work.Relationships),
		pq.Array(work.FreeformTags), work.MaxChapters, work.ChapterCount,
		work.IsComplete, work.Status, work.RestrictedToUsers, work.CommentPolicy,
		work.ModerateComments, work.DisableComments, work.IsAnonymous,
		work.InAnonCollection, work.InUnrevealedCollection, work.CreatedAt, work.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create work", "details": err.Error()})
		return
	}

	// Create creatorship
	var defaultPseudID uuid.UUID
	err = tx.QueryRow(`
		SELECT id FROM pseuds WHERE user_id = $1 AND is_default = true
	`, userUUID).Scan(&defaultPseudID)

	if err != nil {
		// Create default pseud if it doesn't exist
		defaultPseudID = uuid.New()
		_, err = tx.Exec(`
			INSERT INTO pseuds (id, user_id, name, is_default, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, defaultPseudID, userUUID, "DefaultPseud", true, now, now)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pseud"})
			return
		}
	}

	// Create creatorship
	log.Printf("DEBUG: Creating creatorship with workID=%s, defaultPseudID=%s", workID, defaultPseudID)
	_, err = tx.Exec(`
		INSERT INTO creatorships (id, creation_id, creation_type, pseud_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.New(), workID, "Work", defaultPseudID, now)

	if err != nil {
		log.Printf("ERROR: Failed to create creatorship: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create creatorship", "details": err.Error()})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Process tags asynchronously
	go ws.processWorkTags(workID, req)

	// Index in search service asynchronously
	go ws.indexWorkInSearch(workID, work)

	c.JSON(http.StatusCreated, gin.H{"work": work})
}

// processWorkTags processes and creates tag relationships for a work
func (ws *WorkService) processWorkTags(workID uuid.UUID, req models.CreateWorkRequest) {
	ctx := context.Background()

	// Collect all tag names from different categories
	var allTagNames []string
	allTagNames = append(allTagNames, req.Fandoms...)
	allTagNames = append(allTagNames, req.Characters...)
	allTagNames = append(allTagNames, req.Relationships...)
	allTagNames = append(allTagNames, req.FreeformTags...)
	allTagNames = append(allTagNames, req.Category...)
	allTagNames = append(allTagNames, req.Warnings...)
	allTagNames = append(allTagNames, req.Rating)

	// Get or create tags
	var tagIDs []uuid.UUID
	for _, tagName := range allTagNames {
		if tagName == "" {
			continue
		}

		tagID, err := ws.getOrCreateTag(ctx, tagName)
		if err == nil && tagID != uuid.Nil {
			tagIDs = append(tagIDs, tagID)
		}
	}

	// Create work-tag relationships via tag service
	if len(tagIDs) > 0 {
		ws.addTagsToWork(ctx, workID, tagIDs)
	}
}

// getOrCreateTag gets an existing tag or creates a new one
func (ws *WorkService) getOrCreateTag(ctx context.Context, tagName string) (uuid.UUID, error) {
	// First try to find existing tag
	tagClient := NewTagServiceClient("http://tag-service:8083")

	// Search for existing tag
	searchURL := fmt.Sprintf("%s/api/v1/tags?q=%s&limit=1", tagClient.baseURL, tagName)
	resp, err := tagClient.client.Get(searchURL)
	if err != nil {
		return uuid.Nil, err
	}
	defer resp.Body.Close()

	var searchResult struct {
		Tags []models.Tag `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err == nil && len(searchResult.Tags) > 0 {
		// Found existing tag
		return searchResult.Tags[0].ID, nil
	}

	// Create new tag
	tagType := ws.inferTagType(tagName)
	createReq := models.CreateTagRequest{
		Name:         tagName,
		Type:         tagType,
		IsFilterable: true,
	}

	reqBody, _ := json.Marshal(createReq)
	createURL := fmt.Sprintf("%s/api/v1/tags", tagClient.baseURL)

	resp, err = tagClient.client.Post(createURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return uuid.Nil, err
	}
	defer resp.Body.Close()

	var createResult struct {
		Tag models.Tag `json:"tag"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&createResult); err == nil {
		return createResult.Tag.ID, nil
	}

	return uuid.Nil, fmt.Errorf("failed to create tag")
}

// inferTagType attempts to determine tag type from name
func (ws *WorkService) inferTagType(tagName string) string {
	lower := strings.ToLower(tagName)

	// Rating tags
	ratings := []string{"general audiences", "teen and up audiences", "mature", "explicit"}
	for _, rating := range ratings {
		if strings.Contains(lower, rating) {
			return "rating"
		}
	}

	// Warning tags
	warnings := []string{"violence", "death", "underage", "rape", "graphic"}
	for _, warning := range warnings {
		if strings.Contains(lower, warning) {
			return "warning"
		}
	}

	// Relationship indicators
	if strings.Contains(lower, "/") || strings.Contains(lower, "&") {
		return "relationship"
	}

	// Character indicators (usually proper nouns with multiple words or obvious names)
	words := strings.Fields(tagName)
	if len(words) >= 2 && len(words) <= 3 && strings.Title(tagName) == tagName {
		// Common freeform tags that might be title-cased
		commonFreeform := []string{"fluff", "angst", "hurt", "comfort", "smut", "crack", "fix", "au", "pwp"}
		for _, freeform := range commonFreeform {
			if strings.Contains(lower, freeform) {
				return "freeform"
			}
		}
		return "character"
	}

	// Default to freeform
	return "freeform"
}

// addTagsToWork adds tags to a work via tag service
func (ws *WorkService) addTagsToWork(ctx context.Context, workID uuid.UUID, tagIDs []uuid.UUID) {
	tagClient := NewTagServiceClient("http://tag-service:8083")

	reqBody := map[string][]uuid.UUID{
		"tag_ids": tagIDs,
	}

	body, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s/api/v1/works/%s/tags", tagClient.baseURL, workID)

	resp, err := tagClient.client.Post(url, "application/json", bytes.NewBuffer(body))
	if err == nil {
		resp.Body.Close()
	}
}

// indexWorkInSearch indexes a work in the search service
func (ws *WorkService) indexWorkInSearch(workID uuid.UUID, work *models.Work) {
	log.Printf("DEBUG: Starting indexing for work %s", workID)
	searchClient := NewSearchServiceClient("http://localhost:8084")

	// Prepare work data for search indexing in the format expected by search service
	searchDoc := map[string]interface{}{
		"work_id":           workID.String(),
		"title":             work.Title,
		"summary":           work.Summary,
		"rating":            work.Rating,
		"language":          work.Language,
		"fandoms":           work.Fandoms,
		"characters":        work.Characters,
		"relationships":     work.Relationships,
		"additional_tags":   work.FreeformTags,
		"warnings":          work.Warnings,
		"categories":        []string{}, // TODO: extract from work.Category if needed
		"word_count":        work.WordCount,
		"chapter_count":     work.ChapterCount,
		"completion_status": work.Status,
		"published_date":    work.CreatedAt,
		"updated_date":      work.UpdatedAt,
		"author_ids":        []string{work.UserID.String()},
		"author_names":      []string{}, // TODO: fetch author name if needed
		"hits":              work.Hits,
		"kudos":             work.Kudos,
		"comments":          work.Comments,
		"bookmarks":         work.Bookmarks,
		"collections":       []string{},
		"series":            []string{},
		"is_restricted":     work.RestrictedToUsers,
		"is_anonymous":      work.IsAnonymous,
	}

	body, _ := json.Marshal(searchDoc)
	url := fmt.Sprintf("%s/api/v1/index/works/%s", searchClient.baseURL, workID.String())

	log.Printf("DEBUG: Indexing work at URL: %s", url)

	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := searchClient.client.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to index work %s: %v", workID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("ERROR: Indexing failed for work %s, status: %d, response: %s", workID, resp.StatusCode, string(bodyBytes))
	} else {
		log.Printf("DEBUG: Successfully indexed work %s", workID)
	}
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

// getWorkTags retrieves tags for a work from tag service
func (ws *WorkService) getWorkTags(workID uuid.UUID) ([]models.Tag, error) {
	tagClient := NewTagServiceClient("http://tag-service:8083")
	url := fmt.Sprintf("%s/api/v1/works/%s/tags", tagClient.baseURL, workID)

	resp, err := tagClient.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Tags []models.Tag `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Tags, nil
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

	// Execute search via search service
	results, err := ws.searchWorksViaService(searchParams)
	if err != nil {
		// Fallback to database search if search service unavailable
		results, err = ws.searchWorksDatabase(searchParams)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
			return
		}
	}

	c.JSON(http.StatusOK, results)
}

// searchWorksViaService searches works via search service
func (ws *WorkService) searchWorksViaService(params map[string]interface{}) (gin.H, error) {
	searchClient := NewSearchServiceClient("http://localhost:8084")

	body, _ := json.Marshal(params)
	url := fmt.Sprintf("%s/api/v1/search/works", searchClient.baseURL)

	resp, err := searchClient.client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results gin.H
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	return results, nil
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
