package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"nuclear-ao3/shared/models"
)

// =============================================================================
// HECATE CORE TAG SERVICE - Advanced Tag Management System
// Ultra-fast tag operations with intelligent caching and AO3 compatibility
// =============================================================================

// CreateTag creates a new tag in the system
func (ts *TagService) CreateTag(c *gin.Context) {
	var req models.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Validate tag type
	validTypes := []string{"fandom", "character", "relationship", "freeform", "warning", "category", "rating", "additional"}
	if !contains(validTypes, req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag type"})
		return
	}

	tx, err := ts.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Check if tag already exists
	var existingID uuid.UUID
	err = tx.QueryRow(`
		SELECT id FROM tags WHERE LOWER(name) = LOWER($1)
	`, req.Name).Scan(&existingID)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Tag already exists", "existing_id": existingID})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Create new tag
	tagID := uuid.New()
	now := time.Now()

	_, err = tx.Exec(`
		INSERT INTO tags (id, name, canonical_name, type, description, is_canonical, is_filterable, use_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, tagID, req.Name, req.CanonicalName, req.Type, req.Description, req.IsCanonical, req.IsFilterable, 0, now, now)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Tag name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag", "details": err.Error()})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Return created tag
	tag := models.Tag{
		ID:            tagID,
		Name:          req.Name,
		CanonicalName: req.CanonicalName,
		Type:          req.Type,
		Description:   req.Description,
		IsCanonical:   req.IsCanonical,
		IsFilterable:  req.IsFilterable,
		UseCount:      0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Update cache
	ts.cacheTag(&tag)

	c.JSON(http.StatusCreated, gin.H{"tag": tag})
}

// GetTag retrieves a tag by ID
func (ts *TagService) GetTag(c *gin.Context) {
	tagIDStr := c.Param("tag_id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	// Try cache first
	if tag := ts.getCachedTag(tagID); tag != nil {
		c.JSON(http.StatusOK, gin.H{"tag": tag})
		return
	}

	// Query database
	var tag models.Tag
	err = ts.db.QueryRow(`
		SELECT id, name, canonical_name, type, description, is_canonical, is_filterable, use_count, created_at, updated_at
		FROM tags WHERE id = $1
	`, tagID).Scan(
		&tag.ID, &tag.Name, &tag.CanonicalName, &tag.Type, &tag.Description,
		&tag.IsCanonical, &tag.IsFilterable, &tag.UseCount, &tag.CreatedAt, &tag.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Cache the result
	ts.cacheTag(&tag)

	c.JSON(http.StatusOK, gin.H{"tag": tag})
}

// SearchTags provides advanced tag search with filtering
func (ts *TagService) SearchTags(c *gin.Context) {
	query := c.Query("q")
	tagType := c.Query("type")
	canonicalOnly := c.Query("canonical") == "true"
	filterableOnly := c.Query("filterable") == "true"
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	if limit > 100 {
		limit = 100
	}

	// Build query
	var conditions []string
	var args []interface{}
	argIndex := 1

	baseQuery := `
		SELECT id, name, canonical_name, type, description, is_canonical, is_filterable, use_count, created_at, updated_at
		FROM tags
	`

	if query != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(name) LIKE LOWER($%d)", argIndex))
		args = append(args, "%"+query+"%")
		argIndex++
	}

	if tagType != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, tagType)
		argIndex++
	}

	if canonicalOnly {
		conditions = append(conditions, "is_canonical = true")
	}

	if filterableOnly {
		conditions = append(conditions, "is_filterable = true")
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	baseQuery += " ORDER BY use_count DESC, name ASC"
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := ts.db.Query(baseQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
		return
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.CanonicalName, &tag.Type, &tag.Description,
			&tag.IsCanonical, &tag.IsFilterable, &tag.UseCount, &tag.CreatedAt, &tag.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan tag"})
			return
		}
		tags = append(tags, tag)
	}

	// Get total count for pagination
	var total int
	countQuery := "SELECT COUNT(*) FROM tags"
	if len(conditions) > 0 {
		countQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	ts.db.QueryRow(countQuery, args[:len(args)-2]...).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"tags":   tags,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// AutocompleteTags provides ultra-fast tag autocomplete
func (ts *TagService) AutocompleteTags(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	tagType := c.Query("type")
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)

	if limit > 50 {
		limit = 50
	}

	// Check cache first for popular queries
	cacheKey := fmt.Sprintf("autocomplete:%s:%s:%d", query, tagType, limit)
	if cached := ts.getCachedAutocomplete(cacheKey); cached != nil {
		c.JSON(http.StatusOK, gin.H{"suggestions": cached})
		return
	}

	// Build optimized autocomplete query
	var querySQL string
	var args []interface{}

	if tagType != "" {
		querySQL = `
			SELECT id, name, type, use_count, is_canonical
			FROM tags 
			WHERE name ILIKE $1 AND type = $2 AND is_filterable = true
			ORDER BY 
				CASE WHEN name ILIKE $3 THEN 1 ELSE 2 END,
				use_count DESC,
				name ASC
			LIMIT $4
		`
		args = []interface{}{query + "%", tagType, query, limit}
	} else {
		querySQL = `
			SELECT id, name, type, use_count, is_canonical
			FROM tags 
			WHERE name ILIKE $1 AND is_filterable = true
			ORDER BY 
				CASE WHEN name ILIKE $2 THEN 1 ELSE 2 END,
				use_count DESC,
				name ASC
			LIMIT $3
		`
		args = []interface{}{query + "%", query, limit}
	}

	rows, err := ts.db.Query(querySQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var suggestions []models.TagSuggestion
	for rows.Next() {
		var suggestion models.TagSuggestion
		err := rows.Scan(&suggestion.ID, &suggestion.Name, &suggestion.Type, &suggestion.UseCount, &suggestion.Canonical)
		if err != nil {
			continue
		}
		suggestions = append(suggestions, suggestion)
	}

	// Cache the results for fast subsequent requests
	ts.cacheAutocomplete(cacheKey, suggestions)

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// GetTagsByWork retrieves all tags for a specific work
func (ts *TagService) GetTagsByWork(c *gin.Context) {
	workIDStr := c.Param("work_id")
	workID, err := uuid.Parse(workIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	rows, err := ts.db.Query(`
		SELECT t.id, t.name, t.canonical_name, t.type, t.description, t.is_canonical, t.is_filterable, t.use_count, t.created_at, t.updated_at
		FROM tags t
		JOIN work_tags wt ON t.id = wt.tag_id
		WHERE wt.work_id = $1
		ORDER BY t.type, t.name
	`, workID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.CanonicalName, &tag.Type, &tag.Description,
			&tag.IsCanonical, &tag.IsFilterable, &tag.UseCount, &tag.CreatedAt, &tag.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan tag"})
			return
		}
		tags = append(tags, tag)
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// AddTagsToWork adds tags to a work (creates work-tag relationships)
func (ts *TagService) AddTagsToWork(c *gin.Context) {
	workIDStr := c.Param("work_id")
	workID, err := uuid.Parse(workIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	var req struct {
		TagIDs []uuid.UUID `json:"tag_ids" validate:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	tx, err := ts.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Add work-tag relationships
	for _, tagID := range req.TagIDs {
		_, err = tx.Exec(`
			INSERT INTO work_tags (work_id, tag_id, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (work_id, tag_id) DO NOTHING
		`, workID, tagID, time.Now())

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add tag relationship"})
			return
		}

		// Increment tag use count
		_, err = tx.Exec(`
			UPDATE tags SET use_count = use_count + 1, updated_at = $1 WHERE id = $2
		`, time.Now(), tagID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tag use count"})
			return
		}
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Clear relevant caches
	ts.clearWorkTagsCache(workID)

	c.JSON(http.StatusOK, gin.H{"message": "Tags added successfully"})
}

// RemoveTagFromWork removes a tag from a work
func (ts *TagService) RemoveTagFromWork(c *gin.Context) {
	workIDStr := c.Param("work_id")
	tagIDStr := c.Param("tag_id")

	workID, err := uuid.Parse(workIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}

	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	tx, err := ts.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Remove work-tag relationship
	result, err := tx.Exec(`
		DELETE FROM work_tags WHERE work_id = $1 AND tag_id = $2
	`, workID, tagID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove tag relationship"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag relationship not found"})
		return
	}

	// Decrement tag use count
	_, err = tx.Exec(`
		UPDATE tags SET use_count = GREATEST(use_count - 1, 0), updated_at = $1 WHERE id = $2
	`, time.Now(), tagID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tag use count"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Clear relevant caches
	ts.clearWorkTagsCache(workID)

	c.JSON(http.StatusOK, gin.H{"message": "Tag removed successfully"})
}

// =============================================================================
// TAG WRANGLING OPERATIONS (Advanced AO3 Feature)
// =============================================================================

// CreateTagRelationship creates relationships between tags (parent/child, synonyms)
func (ts *TagService) CreateTagRelationship(c *gin.Context) {
	var req models.TagRelationship
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate relationship type
	validTypes := []string{"parent_child", "synonym", "related"}
	if !contains(validTypes, req.RelationshipType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid relationship type"})
		return
	}

	if req.ParentTagID == req.ChildTagID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot create relationship with self"})
		return
	}

	// Get user ID from JWT (for audit trail)
	userID, _ := c.Get("user_id")
	var createdBy *uuid.UUID
	if uid, ok := userID.(string); ok {
		if parsedUID, err := uuid.Parse(uid); err == nil {
			createdBy = &parsedUID
		}
	}

	_, err := ts.db.Exec(`
		INSERT INTO tag_relationships (parent_tag_id, child_tag_id, relationship_type, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (parent_tag_id, child_tag_id) DO UPDATE SET
			relationship_type = EXCLUDED.relationship_type,
			created_at = EXCLUDED.created_at,
			created_by = EXCLUDED.created_by
	`, req.ParentTagID, req.ChildTagID, req.RelationshipType, time.Now(), createdBy)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag relationship"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Tag relationship created successfully"})
}

// GetTagRelationships retrieves all relationships for a tag
func (ts *TagService) GetTagRelationships(c *gin.Context) {
	tagIDStr := c.Param("tag_id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	// Get both parent and child relationships
	rows, err := ts.db.Query(`
		SELECT 
			tr.parent_tag_id,
			tr.child_tag_id,
			tr.relationship_type,
			tr.created_at,
			pt.name as parent_name,
			ct.name as child_name
		FROM tag_relationships tr
		JOIN tags pt ON tr.parent_tag_id = pt.id
		JOIN tags ct ON tr.child_tag_id = ct.id
		WHERE tr.parent_tag_id = $1 OR tr.child_tag_id = $1
		ORDER BY tr.relationship_type, pt.name, ct.name
	`, tagID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	type RelationshipInfo struct {
		ParentTagID      uuid.UUID `json:"parent_tag_id"`
		ChildTagID       uuid.UUID `json:"child_tag_id"`
		RelationshipType string    `json:"relationship_type"`
		CreatedAt        time.Time `json:"created_at"`
		ParentName       string    `json:"parent_name"`
		ChildName        string    `json:"child_name"`
	}

	var relationships []RelationshipInfo
	for rows.Next() {
		var rel RelationshipInfo
		err := rows.Scan(&rel.ParentTagID, &rel.ChildTagID, &rel.RelationshipType, &rel.CreatedAt, &rel.ParentName, &rel.ChildName)
		if err != nil {
			continue
		}
		relationships = append(relationships, rel)
	}

	c.JSON(http.StatusOK, gin.H{"relationships": relationships})
}

// GetPopularTags returns the most popular tags by use count
func (ts *TagService) GetPopularTags(c *gin.Context) {
	tagType := c.Query("type")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	if limit > 100 {
		limit = 100
	}

	var query string
	var args []interface{}

	if tagType != "" {
		query = `
			SELECT id, name, canonical_name, type, description, is_canonical, is_filterable, use_count, created_at, updated_at
			FROM tags 
			WHERE type = $1 AND is_filterable = true AND use_count > 0
			ORDER BY use_count DESC, name ASC
			LIMIT $2
		`
		args = []interface{}{tagType, limit}
	} else {
		query = `
			SELECT id, name, canonical_name, type, description, is_canonical, is_filterable, use_count, created_at, updated_at
			FROM tags 
			WHERE is_filterable = true AND use_count > 0
			ORDER BY use_count DESC, name ASC
			LIMIT $1
		`
		args = []interface{}{limit}
	}

	rows, err := ts.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.CanonicalName, &tag.Type, &tag.Description,
			&tag.IsCanonical, &tag.IsFilterable, &tag.UseCount, &tag.CreatedAt, &tag.UpdatedAt,
		)
		if err != nil {
			continue
		}
		tags = append(tags, tag)
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// =============================================================================
// CACHE MANAGEMENT METHODS
// =============================================================================

// cacheTag stores a tag in Redis cache
func (ts *TagService) cacheTag(tag *models.Tag) {
	if tag == nil {
		return
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("tag:%s", tag.ID)

	if data, err := json.Marshal(tag); err == nil {
		ts.redis.Set(ctx, cacheKey, data, time.Hour)
	}
}

// getCachedTag retrieves a tag from Redis cache
func (ts *TagService) getCachedTag(tagID uuid.UUID) *models.Tag {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("tag:%s", tagID)

	data, err := ts.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil
	}

	var tag models.Tag
	if err := json.Unmarshal([]byte(data), &tag); err != nil {
		return nil
	}

	return &tag
}

// cacheAutocomplete stores autocomplete results in Redis cache
func (ts *TagService) cacheAutocomplete(cacheKey string, suggestions []models.TagSuggestion) {
	ctx := context.Background()

	if data, err := json.Marshal(suggestions); err == nil {
		ts.redis.Set(ctx, cacheKey, data, 15*time.Minute)
	}
}

// getCachedAutocomplete retrieves autocomplete results from Redis cache
func (ts *TagService) getCachedAutocomplete(cacheKey string) []models.TagSuggestion {
	ctx := context.Background()

	data, err := ts.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil
	}

	var suggestions []models.TagSuggestion
	if err := json.Unmarshal([]byte(data), &suggestions); err != nil {
		return nil
	}

	return suggestions
}

// clearWorkTagsCache clears cache entries related to work tags
func (ts *TagService) clearWorkTagsCache(workID uuid.UUID) {
	ctx := context.Background()
	pattern := fmt.Sprintf("work_tags:%s*", workID)

	if keys, err := ts.redis.Keys(ctx, pattern).Result(); err == nil {
		if len(keys) > 0 {
			ts.redis.Del(ctx, keys...)
		}
	}
}

// clearTagCache clears cache entries related to a specific tag
func (ts *TagService) clearTagCache(tagID string) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("tag:%s", tagID)
	ts.redis.Del(ctx, cacheKey)

	// Also clear autocomplete caches that might contain this tag
	pattern := "autocomplete:*"
	if keys, err := ts.redis.Keys(ctx, pattern).Result(); err == nil {
		if len(keys) > 0 {
			ts.redis.Del(ctx, keys...)
		}
	}
}

// =============================================================================
// PLACEHOLDER METHODS FOR EXISTING ENDPOINTS
// These need to be implemented for full compatibility
// =============================================================================

func (ts *TagService) GetRelatedTags(c *gin.Context) {
	tagIDStr := c.Param("tag_id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	// Get related tags through relationships and tag co-occurrence
	rows, err := ts.db.Query(`
		SELECT DISTINCT 
			t.id, t.name, t.type, t.use_count, t.is_canonical,
			COUNT(*) as relation_strength
		FROM tags t
		LEFT JOIN tag_relationships tr1 ON (tr1.child_tag_id = t.id AND tr1.parent_tag_id = $1)
		LEFT JOIN tag_relationships tr2 ON (tr2.parent_tag_id = t.id AND tr2.child_tag_id = $1)
		LEFT JOIN work_tags wt1 ON wt1.tag_id = t.id
		LEFT JOIN work_tags wt2 ON wt2.work_id = wt1.work_id AND wt2.tag_id = $1
		WHERE t.id != $1 
		AND (tr1.parent_tag_id IS NOT NULL OR tr2.child_tag_id IS NOT NULL OR wt2.tag_id IS NOT NULL)
		GROUP BY t.id, t.name, t.type, t.use_count, t.is_canonical
		ORDER BY relation_strength DESC, t.use_count DESC
		LIMIT 20
	`, tagID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var relatedTags []models.TagSuggestion
	for rows.Next() {
		var tag models.TagSuggestion
		var relationStrength int
		err := rows.Scan(&tag.ID, &tag.Name, &tag.Type, &tag.UseCount, &tag.Canonical, &relationStrength)
		if err != nil {
			continue
		}
		relatedTags = append(relatedTags, tag)
	}

	c.JSON(http.StatusOK, gin.H{"related_tags": relatedTags})
}

func (ts *TagService) GetTagWorks(c *gin.Context) {
	tagIDStr := c.Param("tag_id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	sortBy := c.DefaultQuery("sort", "updated_at")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	if limit > 100 {
		limit = 100
	}

	// Validate sort field
	validSorts := []string{"created_at", "updated_at", "prominence_score", "title"}
	if !contains(validSorts, sortBy) {
		sortBy = "updated_at"
	}

	// Get works that use this tag with prominence information
	query := fmt.Sprintf(`
		SELECT 
			w.id, w.title, w.summary, w.word_count, w.chapter_count, w.kudos_count,
			w.created_at, w.updated_at, wt.prominence, wt.prominence_score
		FROM works w
		JOIN work_tags wt ON w.id = wt.work_id
		WHERE wt.tag_id = $1
		ORDER BY %s DESC
		LIMIT $2 OFFSET $3
	`, sortBy)

	rows, err := ts.db.Query(query, tagID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var works []gin.H
	for rows.Next() {
		var work gin.H = make(gin.H)
		var id, title, summary, prominence string
		var wordCount, chapterCount, kudosCount int
		var createdAt, updatedAt time.Time
		var prominenceScore float64

		err := rows.Scan(&id, &title, &summary, &wordCount, &chapterCount, &kudosCount,
			&createdAt, &updatedAt, &prominence, &prominenceScore)
		if err != nil {
			continue
		}

		work["id"] = id
		work["title"] = title
		work["summary"] = summary
		work["word_count"] = wordCount
		work["chapter_count"] = chapterCount
		work["kudos_count"] = kudosCount
		work["created_at"] = createdAt
		work["updated_at"] = updatedAt
		work["tag_prominence"] = prominence
		work["prominence_score"] = prominenceScore

		works = append(works, work)
	}

	// Get total count
	var total int
	err = ts.db.QueryRow(`SELECT COUNT(*) FROM work_tags WHERE tag_id = $1`, tagID).Scan(&total)
	if err != nil {
		total = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"works":  works,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (ts *TagService) SearchFandoms(c *gin.Context) {
	query := c.Query("q")
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	if limit > 100 {
		limit = 100
	}

	// Use the same proven logic as SearchTags but hardcoded for fandoms
	var conditions []string
	var args []interface{}
	argIndex := 1

	baseQuery := `
		SELECT id, name, canonical_name, type, description, is_canonical, is_filterable, use_count, created_at, updated_at
		FROM tags
	`

	// Always filter for fandoms
	conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
	args = append(args, "fandom")
	argIndex++

	// Always filter for filterable tags
	conditions = append(conditions, "is_filterable = true")

	if query != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(name) LIKE LOWER($%d)", argIndex))
		args = append(args, "%"+query+"%")
		argIndex++
	}

	baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	baseQuery += " ORDER BY use_count DESC, name ASC"
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := ts.db.Query(baseQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
		return
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.CanonicalName, &tag.Type, &tag.Description,
			&tag.IsCanonical, &tag.IsFilterable, &tag.UseCount, &tag.CreatedAt, &tag.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan tag"})
			return
		}
		tags = append(tags, tag)
	}

	// Get total count for pagination
	var total int
	countQuery := "SELECT COUNT(*) FROM tags"
	countQuery += " WHERE " + strings.Join(conditions, " AND ")
	ts.db.QueryRow(countQuery, args[:len(args)-2]...).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"fandoms":    tags,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
		"debug_rows": len(tags),
	})
}

func (ts *TagService) GetFandom(c *gin.Context) {
	fandomIDStr := c.Param("fandom_id")
	fandomID, err := uuid.Parse(fandomIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid fandom ID"})
		return
	}

	var tag models.Tag
	err = ts.db.QueryRow(`
		SELECT id, name, canonical_name, type, description, is_canonical, is_filterable, use_count, created_at, updated_at
		FROM tags 
		WHERE id = $1 AND type = 'fandom'
	`, fandomID).Scan(
		&tag.ID, &tag.Name, &tag.CanonicalName, &tag.Type, &tag.Description,
		&tag.IsCanonical, &tag.IsFilterable, &tag.UseCount, &tag.CreatedAt, &tag.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fandom not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get related statistics
	var stats gin.H = make(gin.H)

	// Count works in this fandom
	var workCount int
	ts.db.QueryRow(`SELECT COUNT(*) FROM work_tags WHERE tag_id = $1`, fandomID).Scan(&workCount)
	stats["work_count"] = workCount

	// Count characters in this fandom
	var characterCount int
	ts.db.QueryRow(`
		SELECT COUNT(DISTINCT c.id) 
		FROM tags c
		JOIN work_tags wt_char ON c.id = wt_char.tag_id
		JOIN work_tags wt_fandom ON wt_char.work_id = wt_fandom.work_id
		WHERE wt_fandom.tag_id = $1 AND c.type = 'character'
	`, fandomID).Scan(&characterCount)
	stats["character_count"] = characterCount

	// Count relationships in this fandom
	var relationshipCount int
	ts.db.QueryRow(`
		SELECT COUNT(DISTINCT r.id) 
		FROM tags r
		JOIN work_tags wt_rel ON r.id = wt_rel.tag_id
		JOIN work_tags wt_fandom ON wt_rel.work_id = wt_fandom.work_id
		WHERE wt_fandom.tag_id = $1 AND r.type = 'relationship'
	`, fandomID).Scan(&relationshipCount)
	stats["relationship_count"] = relationshipCount

	c.JSON(http.StatusOK, gin.H{
		"fandom":     tag,
		"statistics": stats,
	})
}

func (ts *TagService) GetFandomTags(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tags": []string{}})
}

func (ts *TagService) GetFandomCharacters(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"characters": []string{}})
}

func (ts *TagService) GetFandomRelationships(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"relationships": []string{}})
}

func (ts *TagService) SearchCharacters(c *gin.Context) {
	query := c.Query("q")
	fandomQuery := c.Query("fandom")
	limitStr := c.DefaultQuery("limit", "20")

	limit, _ := strconv.Atoi(limitStr)
	if limit > 100 {
		limit = 100
	}

	searchPattern := "%" + strings.ToLower(query) + "%"
	var rows *sql.Rows
	var err error

	if fandomQuery != "" {
		// Search characters within a specific fandom
		fandomPattern := "%" + strings.ToLower(fandomQuery) + "%"
		rows, err = ts.db.Query(`
			SELECT DISTINCT c.id, c.name, c.type, c.use_count, c.is_canonical
			FROM tags c
			JOIN work_tags wt_char ON c.id = wt_char.tag_id
			JOIN work_tags wt_fandom ON wt_char.work_id = wt_fandom.work_id
			JOIN tags f ON wt_fandom.tag_id = f.id
			WHERE c.type = 'character' 
			AND f.type = 'fandom'
			AND LOWER(f.name) LIKE $1
			AND ($2 = '' OR LOWER(c.name) LIKE $3)
			AND c.is_filterable = true
			ORDER BY c.use_count DESC, LENGTH(c.name)
			LIMIT $4
		`, fandomPattern, query, searchPattern, limit)
	} else {
		// General character search
		if query == "" {
			rows, err = ts.db.Query(`
				SELECT id, name, type, use_count, is_canonical
				FROM tags 
				WHERE type = 'character' AND is_filterable = true
				ORDER BY use_count DESC 
				LIMIT $1
			`, limit)
		} else {
			rows, err = ts.db.Query(`
				SELECT id, name, type, use_count, is_canonical
				FROM tags 
				WHERE type = 'character' 
				AND LOWER(name) LIKE $1
				AND is_filterable = true
				ORDER BY 
					CASE WHEN LOWER(name) = LOWER($2) THEN 0 ELSE 1 END,
					use_count DESC,
					LENGTH(name)
				LIMIT $3
			`, searchPattern, query, limit)
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var characters []models.TagSuggestion
	for rows.Next() {
		var character models.TagSuggestion
		err := rows.Scan(&character.ID, &character.Name, &character.Type, &character.UseCount, &character.Canonical)
		if err != nil {
			continue
		}
		characters = append(characters, character)
	}

	c.JSON(http.StatusOK, gin.H{"characters": characters})
}

func (ts *TagService) GetCharacter(c *gin.Context) {
	characterIDStr := c.Param("character_id")
	characterID, err := uuid.Parse(characterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid character ID"})
		return
	}

	var tag models.Tag
	err = ts.db.QueryRow(`
		SELECT id, name, canonical_name, type, description, is_canonical, is_filterable, use_count, created_at, updated_at
		FROM tags 
		WHERE id = $1 AND type = 'character'
	`, characterID).Scan(
		&tag.ID, &tag.Name, &tag.CanonicalName, &tag.Type, &tag.Description,
		&tag.IsCanonical, &tag.IsFilterable, &tag.UseCount, &tag.CreatedAt, &tag.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Character not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get top fandoms for this character
	fandomRows, err := ts.db.Query(`
		SELECT f.id, f.name, COUNT(*) as work_count
		FROM tags f
		JOIN work_tags wt_fandom ON f.id = wt_fandom.tag_id
		JOIN work_tags wt_char ON wt_fandom.work_id = wt_char.work_id
		WHERE wt_char.tag_id = $1 AND f.type = 'fandom'
		GROUP BY f.id, f.name
		ORDER BY work_count DESC
		LIMIT 10
	`, characterID)

	var fandoms []gin.H
	if err == nil {
		defer fandomRows.Close()
		for fandomRows.Next() {
			var fandomID, fandomName string
			var workCount int
			if err := fandomRows.Scan(&fandomID, &fandomName, &workCount); err == nil {
				fandoms = append(fandoms, gin.H{
					"id":         fandomID,
					"name":       fandomName,
					"work_count": workCount,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"character": tag,
		"fandoms":   fandoms,
	})
}

func (ts *TagService) GetCharacterRelationships(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"relationships": []string{}})
}

func (ts *TagService) SearchRelationships(c *gin.Context) {
	query := c.Query("q")
	fandomQuery := c.Query("fandom")
	limitStr := c.DefaultQuery("limit", "20")

	limit, _ := strconv.Atoi(limitStr)
	if limit > 100 {
		limit = 100
	}

	var sqlQuery string
	var args []interface{}

	if fandomQuery != "" {
		// Search relationships within a specific fandom
		fandomPattern := "%" + strings.ToLower(fandomQuery) + "%"
		searchPattern := "%" + strings.ToLower(query) + "%"
		sqlQuery = `
			SELECT DISTINCT r.id, r.name, r.type, r.use_count, r.is_canonical
			FROM tags r
			JOIN work_tags wt_rel ON r.id = wt_rel.tag_id
			JOIN work_tags wt_fandom ON wt_rel.work_id = wt_fandom.work_id
			JOIN tags f ON wt_fandom.tag_id = f.id
			WHERE r.type = 'relationship' 
			AND f.type = 'fandom'
			AND LOWER(f.name) LIKE $1
			AND ($2 = '' OR LOWER(r.name) LIKE $3)
			AND r.is_filterable = true
			ORDER BY r.use_count DESC, LENGTH(r.name)
			LIMIT $4`
		args = []interface{}{fandomPattern, query, searchPattern, limit}
	} else if query == "" {
		// Popular relationships
		sqlQuery = `
			SELECT id, name, type, use_count, is_canonical
			FROM tags 
			WHERE type = 'relationship' AND is_filterable = true
			ORDER BY use_count DESC 
			LIMIT $1`
		args = []interface{}{limit}
	} else {
		// General relationship search
		searchPattern := "%" + strings.ToLower(query) + "%"
		sqlQuery = `
			SELECT id, name, type, use_count, is_canonical
			FROM tags 
			WHERE type = 'relationship' 
			AND LOWER(name) LIKE $1
			AND is_filterable = true
			ORDER BY 
				CASE WHEN LOWER(name) = LOWER($2) THEN 0 ELSE 1 END,
				use_count DESC,
				LENGTH(name)
			LIMIT $3`
		args = []interface{}{searchPattern, query, limit}
	}

	rows, err := ts.db.Query(sqlQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
		return
	}
	defer rows.Close()

	var relationships []gin.H
	for rows.Next() {
		var id, name, tagType string
		var useCount int
		var isCanonical bool

		err := rows.Scan(&id, &name, &tagType, &useCount, &isCanonical)
		if err != nil {
			continue
		}

		relationships = append(relationships, gin.H{
			"id":        id,
			"name":      name,
			"type":      tagType,
			"use_count": useCount,
			"canonical": isCanonical,
		})
	}

	c.JSON(http.StatusOK, gin.H{"relationships": relationships})
}

func (ts *TagService) GetRelationship(c *gin.Context) {
	relationshipIDStr := c.Param("relationship_id")
	relationshipID, err := uuid.Parse(relationshipIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid relationship ID"})
		return
	}

	var tag models.Tag
	err = ts.db.QueryRow(`
		SELECT id, name, canonical_name, type, description, is_canonical, is_filterable, use_count, created_at, updated_at
		FROM tags 
		WHERE id = $1 AND type = 'relationship'
	`, relationshipID).Scan(
		&tag.ID, &tag.Name, &tag.CanonicalName, &tag.Type, &tag.Description,
		&tag.IsCanonical, &tag.IsFilterable, &tag.UseCount, &tag.CreatedAt, &tag.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Relationship not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get top fandoms for this relationship
	fandomRows, err := ts.db.Query(`
		SELECT f.id, f.name, COUNT(*) as work_count
		FROM tags f
		JOIN work_tags wt_fandom ON f.id = wt_fandom.tag_id
		JOIN work_tags wt_rel ON wt_fandom.work_id = wt_rel.work_id
		WHERE wt_rel.tag_id = $1 AND f.type = 'fandom'
		GROUP BY f.id, f.name
		ORDER BY work_count DESC
		LIMIT 10
	`, relationshipID)

	var fandoms []gin.H
	if err == nil {
		defer fandomRows.Close()
		for fandomRows.Next() {
			var fandomID, fandomName string
			var workCount int
			if err := fandomRows.Scan(&fandomID, &fandomName, &workCount); err == nil {
				fandoms = append(fandoms, gin.H{
					"id":         fandomID,
					"name":       fandomName,
					"work_count": workCount,
				})
			}
		}
	}

	// Get prominence statistics for this relationship
	var primaryCount, secondaryCount, microCount int
	var avgScore sql.NullFloat64
	ts.db.QueryRow(`
		SELECT 
			COUNT(CASE WHEN prominence = 'primary' THEN 1 END) as primary_count,
			COUNT(CASE WHEN prominence = 'secondary' THEN 1 END) as secondary_count,
			COUNT(CASE WHEN prominence = 'micro' THEN 1 END) as micro_count,
			AVG(prominence_score) as avg_score
		FROM work_tags WHERE tag_id = $1
	`, relationshipID).Scan(&primaryCount, &secondaryCount, &microCount, &avgScore)

	prominenceStats := gin.H{
		"primary_count":   primaryCount,
		"secondary_count": secondaryCount,
		"micro_count":     microCount,
		"avg_score":       0.0,
	}
	if avgScore.Valid {
		prominenceStats["avg_score"] = avgScore.Float64
	}

	c.JSON(http.StatusOK, gin.H{
		"relationship":     tag,
		"fandoms":          fandoms,
		"prominence_stats": prominenceStats,
	})
}

// Additional stub methods for routes defined in main.go
func (ts *TagService) GetFandomHierarchy(c *gin.Context) {
	fandomIDStr := c.Param("fandom_id")
	fandomID, err := uuid.Parse(fandomIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid fandom ID"})
		return
	}

	// Get hierarchical relationships for this fandom
	rows, err := ts.db.Query(`
		SELECT 
			tr.relationship_type,
			CASE 
				WHEN tr.parent_tag_id = $1 THEN 'child'
				ELSE 'parent'
			END as direction,
			CASE 
				WHEN tr.parent_tag_id = $1 THEN ct.id
				ELSE pt.id
			END as related_id,
			CASE 
				WHEN tr.parent_tag_id = $1 THEN ct.name
				ELSE pt.name
			END as related_name,
			CASE 
				WHEN tr.parent_tag_id = $1 THEN ct.type
				ELSE pt.type
			END as related_type
		FROM tag_relationships tr
		JOIN tags pt ON tr.parent_tag_id = pt.id
		JOIN tags ct ON tr.child_tag_id = ct.id
		WHERE tr.parent_tag_id = $1 OR tr.child_tag_id = $1
		ORDER BY tr.relationship_type, related_name
	`, fandomID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	hierarchy := gin.H{
		"parents":  []gin.H{},
		"children": []gin.H{},
		"synonyms": []gin.H{},
		"related":  []gin.H{},
	}

	for rows.Next() {
		var relationshipType, direction, relatedID, relatedName, relatedType string
		err := rows.Scan(&relationshipType, &direction, &relatedID, &relatedName, &relatedType)
		if err != nil {
			continue
		}

		tag := gin.H{
			"id":   relatedID,
			"name": relatedName,
			"type": relatedType,
		}

		switch relationshipType {
		case "parent_child":
			if direction == "parent" {
				hierarchy["parents"] = append(hierarchy["parents"].([]gin.H), tag)
			} else {
				hierarchy["children"] = append(hierarchy["children"].([]gin.H), tag)
			}
		case "synonym":
			hierarchy["synonyms"] = append(hierarchy["synonyms"].([]gin.H), tag)
		case "related":
			hierarchy["related"] = append(hierarchy["related"].([]gin.H), tag)
		}
	}

	c.JSON(http.StatusOK, gin.H{"hierarchy": hierarchy})
}

func (ts *TagService) GetCharacterHierarchy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"hierarchy": []string{}})
}

func (ts *TagService) GetRelationshipHierarchy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"hierarchy": []string{}})
}

func (ts *TagService) GetTrendingTags(c *gin.Context) {
	tagType := c.Query("type")
	limitStr := c.DefaultQuery("limit", "20")
	daysStr := c.DefaultQuery("days", "7")

	limit, _ := strconv.Atoi(limitStr)
	days, _ := strconv.Atoi(daysStr)

	if limit > 100 {
		limit = 100
	}
	if days > 365 {
		days = 365
	}

	// Calculate trending based on recent work activity
	query := `
		SELECT 
			t.id, t.name, t.type, t.use_count, t.is_canonical,
			COUNT(wt.work_id) as recent_uses,
			(COUNT(wt.work_id)::float / GREATEST(t.use_count, 1)) as trend_ratio
		FROM tags t
		JOIN work_tags wt ON t.id = wt.tag_id
		JOIN works w ON wt.work_id = w.id
		WHERE w.created_at >= $1
		AND t.is_filterable = true
	`

	args := []interface{}{time.Now().AddDate(0, 0, -days)}
	argIndex := 2

	if tagType != "" {
		query += fmt.Sprintf(" AND t.type = $%d", argIndex)
		args = append(args, tagType)
		argIndex++
	}

	query += `
		GROUP BY t.id, t.name, t.type, t.use_count, t.is_canonical
		HAVING COUNT(wt.work_id) >= 3
		ORDER BY trend_ratio DESC, recent_uses DESC
	`

	query += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := ts.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var trending []gin.H
	for rows.Next() {
		var tag models.TagSuggestion
		var recentUses int
		var trendRatio float64

		err := rows.Scan(&tag.ID, &tag.Name, &tag.Type, &tag.UseCount, &tag.Canonical, &recentUses, &trendRatio)
		if err != nil {
			continue
		}

		trending = append(trending, gin.H{
			"id":          tag.ID,
			"name":        tag.Name,
			"type":        tag.Type,
			"use_count":   tag.UseCount,
			"canonical":   tag.Canonical,
			"recent_uses": recentUses,
			"trend_ratio": trendRatio,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"trending":    trending,
		"period_days": days,
		"tag_type":    tagType,
	})
}

func (ts *TagService) GetTagUsageStats(c *gin.Context) {
	tagIDStr := c.Param("tag_id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	// Get basic usage statistics
	var stats gin.H = make(gin.H)

	// Total works using this tag
	var totalWorks int
	ts.db.QueryRow(`SELECT COUNT(*) FROM work_tags WHERE tag_id = $1`, tagID).Scan(&totalWorks)

	// Usage by prominence
	var primaryCount, secondaryCount, microCount, unassignedCount int
	ts.db.QueryRow(`
		SELECT 
			COUNT(CASE WHEN prominence = 'primary' THEN 1 END),
			COUNT(CASE WHEN prominence = 'secondary' THEN 1 END),
			COUNT(CASE WHEN prominence = 'micro' THEN 1 END),
			COUNT(CASE WHEN prominence = 'unassigned' THEN 1 END)
		FROM work_tags WHERE tag_id = $1
	`, tagID).Scan(&primaryCount, &secondaryCount, &microCount, &unassignedCount)

	// Recent usage (last 30 days)
	var recentWorks int
	ts.db.QueryRow(`
		SELECT COUNT(DISTINCT wt.work_id)
		FROM work_tags wt
		JOIN works w ON wt.work_id = w.id
		WHERE wt.tag_id = $1 AND w.created_at >= $2
	`, tagID, time.Now().AddDate(0, 0, -30)).Scan(&recentWorks)

	// Co-occurrence with other tags (top 10)
	cotagRows, err := ts.db.Query(`
		SELECT t.name, t.type, COUNT(*) as cooccurrence_count
		FROM work_tags wt1
		JOIN work_tags wt2 ON wt1.work_id = wt2.work_id
		JOIN tags t ON wt2.tag_id = t.id
		WHERE wt1.tag_id = $1 AND wt2.tag_id != $1
		GROUP BY t.id, t.name, t.type
		ORDER BY cooccurrence_count DESC
		LIMIT 10
	`, tagID)

	var coTags []gin.H
	if err == nil {
		defer cotagRows.Close()
		for cotagRows.Next() {
			var name, tagType string
			var count int
			if err := cotagRows.Scan(&name, &tagType, &count); err == nil {
				coTags = append(coTags, gin.H{
					"name":               name,
					"type":               tagType,
					"cooccurrence_count": count,
				})
			}
		}
	}

	stats = gin.H{
		"total_works":      totalWorks,
		"recent_works_30d": recentWorks,
		"prominence_breakdown": gin.H{
			"primary":    primaryCount,
			"secondary":  secondaryCount,
			"micro":      microCount,
			"unassigned": unassignedCount,
		},
		"frequently_paired_with": coTags,
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (ts *TagService) UpdateTag(c *gin.Context) {
	tagIDStr := c.Param("tag_id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	var req gin.H
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Build dynamic update query
	var setParts []string
	var args []interface{}
	argIndex := 1

	if name, exists := req["name"]; exists {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, name)
		argIndex++
	}

	if canonicalName, exists := req["canonical_name"]; exists {
		setParts = append(setParts, fmt.Sprintf("canonical_name = $%d", argIndex))
		args = append(args, canonicalName)
		argIndex++
	}

	if description, exists := req["description"]; exists {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, description)
		argIndex++
	}

	if isCanonical, exists := req["is_canonical"]; exists {
		setParts = append(setParts, fmt.Sprintf("is_canonical = $%d", argIndex))
		args = append(args, isCanonical)
		argIndex++
	}

	if isFilterable, exists := req["is_filterable"]; exists {
		setParts = append(setParts, fmt.Sprintf("is_filterable = $%d", argIndex))
		args = append(args, isFilterable)
		argIndex++
	}

	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Add updated_at and tag ID
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, tagID)

	query := fmt.Sprintf(`
		UPDATE tags 
		SET %s 
		WHERE id = $%d
	`, strings.Join(setParts, ", "), argIndex)

	result, err := ts.db.Exec(query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Tag name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tag"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
		return
	}

	// Clear cache
	ts.clearTagCache(tagID.String())

	c.JSON(http.StatusOK, gin.H{"message": "Tag updated successfully"})
}

func (ts *TagService) CreateSynonym(c *gin.Context) {
	var req gin.H
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	canonicalIDStr, ok := req["canonical_id"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "canonical_id is required"})
		return
	}

	synonymName, ok := req["synonym_name"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "synonym_name is required"})
		return
	}

	canonicalID, err := uuid.Parse(canonicalIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid canonical_id"})
		return
	}

	// Get user ID from JWT
	userID, _ := c.Get("user_id")
	var createdBy uuid.UUID
	if uid, ok := userID.(string); ok {
		if parsedUID, err := uuid.Parse(uid); err == nil {
			createdBy = parsedUID
		}
	}

	// Check if canonical tag exists
	var canonicalExists bool
	err = ts.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM tags WHERE id = $1)`, canonicalID).Scan(&canonicalExists)
	if err != nil || !canonicalExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Canonical tag not found"})
		return
	}

	// Create the synonym relationship
	_, err = ts.db.Exec(`
		INSERT INTO tag_relationships (parent_tag_id, child_tag_id, relationship_type, created_at, created_by)
		VALUES ($1, (SELECT id FROM tags WHERE name = $2), 'synonym', $3, $4)
		ON CONFLICT (parent_tag_id, child_tag_id) DO UPDATE SET
			relationship_type = 'synonym',
			created_at = EXCLUDED.created_at
	`, canonicalID, synonymName, time.Now(), createdBy)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create synonym"})
		return
	}

	// Clear relevant caches
	ts.clearTagCache(canonicalID.String())

	c.JSON(http.StatusCreated, gin.H{"message": "Synonym created successfully"})
}

func (ts *TagService) RequestTagMerge(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Merge requested"})
}

func (ts *TagService) FollowTag(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tag followed"})
}

func (ts *TagService) UnfollowTag(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tag unfollowed"})
}

func (ts *TagService) GetFollowedTags(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"followed": []string{}})
}

func (ts *TagService) ReportTag(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tag reported"})
}

func (ts *TagService) GetWranglingQueue(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"queue": []string{}})
}

func (ts *TagService) GetTagForWrangling(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tag": nil})
}

func (ts *TagService) WrangleTag(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tag wrangled"})
}

func (ts *TagService) MakeCanonical(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tag made canonical"})
}

func (ts *TagService) CreateCanonicalSynonym(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Canonical synonym created"})
}

func (ts *TagService) AddParentTag(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Parent tag added"})
}

func (ts *TagService) RemoveParentTag(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Parent tag removed"})
}

func (ts *TagService) ProcessTagMerge(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tag merge processed"})
}

func (ts *TagService) GetTagReports(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"reports": []string{}})
}

func (ts *TagService) ProcessTagReport(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Report processed"})
}

func (ts *TagService) AdminListTags(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tags": []string{}})
}

func (ts *TagService) AdminDeleteTag(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tag deleted"})
}

func (ts *TagService) AdminBanTag(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tag banned"})
}

func (ts *TagService) AdminUnbanTag(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tag unbanned"})
}

func (ts *TagService) AdminListWranglers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"wranglers": []string{}})
}

func (ts *TagService) AdminAddWrangler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Wrangler added"})
}

func (ts *TagService) AdminRemoveWrangler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Wrangler removed"})
}

func (ts *TagService) AdminGetTagStatistics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"statistics": gin.H{}})
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
