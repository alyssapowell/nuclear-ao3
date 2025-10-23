package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

// TTL Configuration - Conservative Security Model
const (
	DEFAULT_EXPORT_TTL = 24 * time.Hour   // 24 hours for all exports
	MAX_EXPORT_TTL     = 24 * time.Hour   // Maximum 24 hours (no longer TTLs)
	MIN_EXPORT_TTL     = 1 * time.Hour    // 1 hour minimum TTL
	CLEANUP_INTERVAL   = 15 * time.Minute // Check every 15 minutes for rapid response
	DMCA_RESPONSE_TIME = 5 * time.Minute  // Target DMCA response time
)

type ExportService struct {
	db          *sql.DB
	redisClient *redis.Client
}

type ExportRequest struct {
	WorkID      string        `json:"work_id" binding:"required"`
	Format      string        `json:"format" binding:"required,oneof=epub mobi pdf"`
	Options     ExportOptions `json:"options"`
	UserID      string        `json:"user_id"`
	RequestedAt time.Time     `json:"requested_at"`
	TTL         time.Duration `json:"ttl,omitempty"` // Optional custom TTL
}

type ExportOptions struct {
	IncludeImages   bool   `json:"include_images"`
	CustomStyling   string `json:"custom_styling,omitempty"`
	FontFamily      string `json:"font_family,omitempty"`
	FontSize        string `json:"font_size,omitempty"`
	ChapterBreaks   bool   `json:"chapter_breaks"`
	IncludeMetadata bool   `json:"include_metadata"`
	IncludeComments bool   `json:"include_comments"`
	IncludeTags     bool   `json:"include_tags"`
}

type ExportStatus struct {
	ID          string     `json:"id"`
	WorkID      string     `json:"work_id"`
	UserID      string     `json:"user_id,omitempty"`
	Format      string     `json:"format"`
	Status      string     `json:"status"`   // pending, processing, completed, failed, expired
	Progress    int        `json:"progress"` // 0-100
	DownloadURL string     `json:"download_url,omitempty"`
	Error       string     `json:"error,omitempty"`
	Options     string     `json:"options"` // JSON serialized ExportOptions
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ExpiresAt   time.Time  `json:"expires_at"`
	TTL         int64      `json:"ttl_seconds"`           // TTL in seconds for client display
	RefreshURL  string     `json:"refresh_url,omitempty"` // URL to refresh/extend TTL
}

func main() {
	// Database connection
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "password"),
		getEnv("DB_NAME", "ao3_development"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_SSLMODE", "disable"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_URL", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	// Create export table if it doesn't exist
	createExportTable(db)

	service := &ExportService{
		db:          db,
		redisClient: redisClient,
	}

	// Start cleanup routine
	go service.startCleanupRoutine()

	// Set up Gin router
	r := gin.Default()

	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:3000",
		"http://localhost:3001",
		"https://nuclear-ao3.org",
	}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "export-service"})
	})

	// Export endpoints
	v1 := r.Group("/api/v1")
	{
		v1.POST("/export", service.CreateExport)
		v1.GET("/export/:id", service.GetExportStatus)
		v1.GET("/export/:id/download", service.DownloadExport)
		v1.POST("/export/:id/refresh", service.RefreshExport) // TTL refresh endpoint
		v1.DELETE("/export/:id", service.CancelExport)
		v1.GET("/exports/user/:user_id", service.GetUserExports)
		v1.POST("/exports/cleanup", service.ManualCleanup) // Manual cleanup endpoint
	}

	port := getEnv("PORT", "8085")
	log.Printf("Export service starting on port %s", port)
	log.Printf("Export TTL settings: Default=%v, Max=%v, Min=%v", DEFAULT_EXPORT_TTL, MAX_EXPORT_TTL, MIN_EXPORT_TTL)
	log.Fatal(r.Run(":" + port))
}

func createExportTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS export_status (
		id VARCHAR(255) PRIMARY KEY,
		work_id VARCHAR(255) NOT NULL,
		user_id VARCHAR(255),
		format VARCHAR(10) NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'pending',
		progress INTEGER DEFAULT 0,
		download_url TEXT,
		error_message TEXT,
		options TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		expires_at TIMESTAMP NOT NULL,
		ttl_seconds BIGINT NOT NULL
	);
	
	CREATE INDEX IF NOT EXISTS idx_export_status_expires_at ON export_status(expires_at);
	CREATE INDEX IF NOT EXISTS idx_export_status_user_id ON export_status(user_id);
	CREATE INDEX IF NOT EXISTS idx_export_status_work_id ON export_status(work_id);
	`

	if _, err := db.Exec(query); err != nil {
		log.Fatal("Failed to create export table:", err)
	}
}

func (s *ExportService) CreateExport(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate work exists and user has access
	if !s.validateWorkAccess(req.WorkID, req.UserID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this work"})
		return
	}

	// Validate and set TTL
	ttl := req.TTL
	if ttl == 0 {
		ttl = DEFAULT_EXPORT_TTL
	}

	// Enforce TTL limits
	if ttl < MIN_EXPORT_TTL {
		ttl = MIN_EXPORT_TTL
	}
	if ttl > MAX_EXPORT_TTL {
		ttl = MAX_EXPORT_TTL
	}

	// Check for existing recent export
	existingID, err := s.checkExistingExport(req.WorkID, req.UserID, req.Format)
	if err == nil && existingID != "" {
		c.JSON(http.StatusConflict, gin.H{
			"error":              "Recent export already exists",
			"existing_export_id": existingID,
			"message":            "Please wait for the existing export to complete or expire",
		})
		return
	}

	// Serialize options
	optionsJSON, _ := json.Marshal(req.Options)

	// Create export status record
	exportID := generateExportID()
	expiresAt := time.Now().Add(ttl)

	query := `
		INSERT INTO export_status (id, work_id, user_id, format, status, progress, options, expires_at, ttl_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = s.db.Exec(query, exportID, req.WorkID, req.UserID, req.Format, "pending", 0,
		string(optionsJSON), expiresAt, int64(ttl.Seconds()))

	if err != nil {
		log.Printf("Failed to create export: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create export"})
		return
	}

	// Queue export job
	go s.processExport(exportID)

	c.JSON(http.StatusCreated, gin.H{
		"export_id":      exportID,
		"status":         "pending",
		"estimated_time": s.estimateProcessingTime(req.Format),
		"expires_at":     expiresAt,
		"ttl_seconds":    int64(ttl.Seconds()),
		"refresh_url":    fmt.Sprintf("/api/v1/export/%s/refresh", exportID),
	})
}

func (s *ExportService) GetExportStatus(c *gin.Context) {
	exportID := c.Param("id")

	query := `
		SELECT id, work_id, user_id, format, status, progress, download_url, error_message, 
		       options, created_at, completed_at, expires_at, ttl_seconds
		FROM export_status WHERE id = $1
	`

	var export ExportStatus
	var completedAt sql.NullTime
	var downloadURL, errorMsg sql.NullString

	err := s.db.QueryRow(query, exportID).Scan(
		&export.ID, &export.WorkID, &export.UserID, &export.Format, &export.Status,
		&export.Progress, &downloadURL, &errorMsg, &export.Options,
		&export.CreatedAt, &completedAt, &export.ExpiresAt, &export.TTL,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Export not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Check if export has expired
	if time.Now().After(export.ExpiresAt) {
		// Mark as expired and clean up
		s.markExportExpired(exportID)
		c.JSON(http.StatusGone, gin.H{
			"error":      "Export has expired",
			"expired_at": export.ExpiresAt,
			"message":    "Please create a new export request",
		})
		return
	}

	// Populate optional fields
	if downloadURL.Valid {
		export.DownloadURL = downloadURL.String
	}
	if errorMsg.Valid {
		export.Error = errorMsg.String
	}
	if completedAt.Valid {
		export.CompletedAt = &completedAt.Time
	}

	// Calculate time remaining
	timeRemaining := export.ExpiresAt.Sub(time.Now())

	response := gin.H{
		"id":                     export.ID,
		"work_id":                export.WorkID,
		"format":                 export.Format,
		"status":                 export.Status,
		"progress":               export.Progress,
		"created_at":             export.CreatedAt,
		"expires_at":             export.ExpiresAt,
		"ttl_seconds":            export.TTL,
		"time_remaining_seconds": int64(timeRemaining.Seconds()),
		"refresh_url":            fmt.Sprintf("/api/v1/export/%s/refresh", export.ID),
	}

	if export.Status == "completed" && export.DownloadURL != "" {
		response["download_url"] = fmt.Sprintf("/api/v1/export/%s/download", export.ID)
		response["completed_at"] = export.CompletedAt
	}

	if export.Status == "failed" && export.Error != "" {
		response["error"] = export.Error
	}

	c.JSON(http.StatusOK, response)
}

func (s *ExportService) RefreshExport(c *gin.Context) {
	exportID := c.Param("id")
	userID := c.Query("user_id")

	// Only allow users to refresh their own exports
	query := `
		SELECT status, expires_at, user_id FROM export_status 
		WHERE id = $1 AND (user_id = $2 OR $2 = '')
	`

	var status string
	var expiresAt time.Time
	var dbUserID string

	err := s.db.QueryRow(query, exportID, userID).Scan(&status, &expiresAt, &dbUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Export not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Check if export is in a refreshable state
	if status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Can only refresh completed exports",
			"current_status": status,
		})
		return
	}

	// Extend TTL by default amount (but not beyond max)
	newExpiresAt := time.Now().Add(DEFAULT_EXPORT_TTL)
	maxExpiresAt := time.Now().Add(MAX_EXPORT_TTL)
	if newExpiresAt.After(maxExpiresAt) {
		newExpiresAt = maxExpiresAt
	}

	// Update expiration time
	updateQuery := `
		UPDATE export_status 
		SET expires_at = $1, ttl_seconds = $2 
		WHERE id = $3
	`

	newTTL := newExpiresAt.Sub(time.Now())
	_, err = s.db.Exec(updateQuery, newExpiresAt, int64(newTTL.Seconds()), exportID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh export"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Export TTL refreshed successfully",
		"new_expires_at": newExpiresAt,
		"ttl_seconds":    int64(newTTL.Seconds()),
	})
}

func (s *ExportService) DownloadExport(c *gin.Context) {
	exportID := c.Param("id")

	query := `
		SELECT status, expires_at, format, work_id FROM export_status 
		WHERE id = $1 AND status = 'completed'
	`

	var status, format, workID string
	var expiresAt time.Time

	err := s.db.QueryRow(query, exportID).Scan(&status, &expiresAt, &format, &workID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Export not found or not ready"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Check if export has expired
	if time.Now().After(expiresAt) {
		s.markExportExpired(exportID)
		c.JSON(http.StatusGone, gin.H{
			"error":      "Export has expired",
			"expired_at": expiresAt,
			"message":    "Please create a new export request",
		})
		return
	}

	// Check if file exists
	filePath := fmt.Sprintf("./exports/%s.%s", exportID, format)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Export file not found"})
		return
	}

	// Get work title for filename
	workTitle := s.getWorkTitle(workID)
	filename := fmt.Sprintf("%s.%s", sanitizeFilename(workTitle), format)

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", s.getMimeType(format))
	c.File(filePath)
}

// Additional methods for TTL management and cleanup...

func (s *ExportService) startCleanupRoutine() {
	ticker := time.NewTicker(CLEANUP_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupExpiredExports()
		}
	}
}

func (s *ExportService) cleanupExpiredExports() {
	log.Println("Running scheduled cleanup of expired exports...")

	// Find expired exports
	query := `
		SELECT id, format FROM export_status 
		WHERE expires_at < CURRENT_TIMESTAMP AND status != 'expired'
	`

	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("Error finding expired exports: %v", err)
		return
	}
	defer rows.Close()

	var expiredCount int
	for rows.Next() {
		var id, format string
		if err := rows.Scan(&id, &format); err == nil {
			// Mark as expired
			s.markExportExpired(id)

			// Delete file
			filePath := fmt.Sprintf("./exports/%s.%s", id, format)
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				log.Printf("Error removing file %s: %v", filePath, err)
			}

			expiredCount++
		}
	}

	if expiredCount > 0 {
		log.Printf("Cleaned up %d expired exports", expiredCount)
	}
}

func (s *ExportService) markExportExpired(exportID string) {
	query := `UPDATE export_status SET status = 'expired' WHERE id = $1`
	s.db.Exec(query, exportID)
}

func (s *ExportService) checkExistingExport(workID, userID, format string) (string, error) {
	query := `
		SELECT id FROM export_status 
		WHERE work_id = $1 AND user_id = $2 AND format = $3 
		AND status IN ('pending', 'processing', 'completed') 
		AND expires_at > CURRENT_TIMESTAMP
		ORDER BY created_at DESC LIMIT 1
	`

	var existingID string
	err := s.db.QueryRow(query, workID, userID, format).Scan(&existingID)
	return existingID, err
}

// Implement remaining helper methods...
func (s *ExportService) processExport(exportID string) {
	// TODO: Implement actual export processing
	// For now, simulate processing
	time.Sleep(2 * time.Second)

	query := `UPDATE export_status SET status = 'completed', progress = 100, completed_at = CURRENT_TIMESTAMP WHERE id = $1`
	s.db.Exec(query, exportID)
}

func (s *ExportService) validateWorkAccess(workID, userID string) bool {
	// TODO: Implement proper work access validation
	return true
}

func (s *ExportService) estimateProcessingTime(format string) string {
	switch format {
	case "epub":
		return "2-5 minutes"
	case "mobi":
		return "3-7 minutes"
	case "pdf":
		return "1-3 minutes"
	default:
		return "2-5 minutes"
	}
}

func (s *ExportService) getWorkTitle(workID string) string {
	// TODO: Fetch actual work title from database
	return "Untitled Work"
}

func (s *ExportService) getMimeType(format string) string {
	switch format {
	case "epub":
		return "application/epub+zip"
	case "mobi":
		return "application/x-mobipocket-ebook"
	case "pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

func generateExportID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "export_" + hex.EncodeToString(bytes)
}

func sanitizeFilename(filename string) string {
	reg := regexp.MustCompile(`[<>:"/\\|?*]`)
	cleaned := reg.ReplaceAllString(filename, "_")
	return strings.TrimSpace(cleaned)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Additional endpoint implementations for completeness...

func (s *ExportService) CancelExport(c *gin.Context) {
	exportID := c.Param("id")
	userID := c.Query("user_id")

	query := `
		UPDATE export_status SET status = 'cancelled' 
		WHERE id = $1 AND user_id = $2 AND status IN ('pending', 'processing')
	`

	result, err := s.db.Exec(query, exportID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Export not found or cannot be cancelled"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Export cancelled"})
}

func (s *ExportService) GetUserExports(c *gin.Context) {
	userID := c.Param("user_id")

	query := `
		SELECT id, work_id, format, status, progress, created_at, expires_at, ttl_seconds
		FROM export_status WHERE user_id = $1 
		ORDER BY created_at DESC LIMIT 50
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var exports []gin.H
	for rows.Next() {
		var id, workID, format, status string
		var progress int
		var createdAt, expiresAt time.Time
		var ttlSeconds int64

		if err := rows.Scan(&id, &workID, &format, &status, &progress, &createdAt, &expiresAt, &ttlSeconds); err == nil {
			exports = append(exports, gin.H{
				"id":                     id,
				"work_id":                workID,
				"format":                 format,
				"status":                 status,
				"progress":               progress,
				"created_at":             createdAt,
				"expires_at":             expiresAt,
				"ttl_seconds":            ttlSeconds,
				"time_remaining_seconds": int64(expiresAt.Sub(time.Now()).Seconds()),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"exports": exports})
}

func (s *ExportService) ManualCleanup(c *gin.Context) {
	go s.cleanupExpiredExports()
	c.JSON(http.StatusOK, gin.H{"message": "Cleanup initiated"})
}
