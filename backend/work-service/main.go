package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"nuclear-ao3/shared/cache"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize services
	workService := NewWorkService()
	defer workService.Close()

	// Setup router
	router := setupRouter(workService)

	// Setup server
	srv := &http.Server{
		Addr:           ":" + getEnv("PORT", "8082"),
		Handler:        router,
		ReadTimeout:    time.Second * 15,
		WriteTimeout:   time.Second * 15,
		IdleTimeout:    time.Second * 60,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server in goroutine
	go func() {
		log.Printf("Work service starting on port %s", getEnv("PORT", "8082"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

func setupRouter(workService *WorkService) *gin.Engine {
	// Set Gin mode
	if getEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(CORSMiddleware())
	r.Use(LoggingMiddleware())
	r.Use(RateLimitMiddleware(workService.redis))
	r.Use(SecurityHeadersMiddleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":   "work-service",
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	// Metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API endpoints
	api := r.Group("/api/v1")
	{
		// Public endpoints with optional auth
		// Legacy routes (plural - supports both UUID and integer with redirect)
		legacy := api.Group("/works")
		legacy.Use(OptionalAuthMiddleware())
		{
			legacy.GET("", workService.SearchWorks)                              // GET /api/v1/works?q=search&fandom=HP (browse/search)
			legacy.GET("/:work_id", workService.CachedGetWork)                   // GET /api/v1/works/123 or /works/uuid (redirects legacy IDs)
			legacy.GET("/:work_id/chapters", workService.GetChapters)            // GET /api/v1/works/123/chapters
			legacy.GET("/:work_id/chapters/:chapter_id", workService.GetChapter) // GET /api/v1/works/123/chapters/1
			legacy.GET("/:work_id/comments", workService.GetComments)            // GET /api/v1/works/123/comments
			legacy.GET("/:work_id/kudos", workService.GetKudos)                  // GET /api/v1/works/123/kudos
			legacy.GET("/:work_id/stats", workService.CachedGetWorkStats)        // GET /api/v1/works/123/stats
			legacy.POST("/:work_id/comments", workService.CreateComment)         // POST /api/v1/works/123/comments (guest + auth comments)
		}

		// Modern routes (singular - UUID-based permanent URLs)
		modern := api.Group("/work")
		modern.Use(OptionalAuthMiddleware())
		{
			modern.GET("/:work_id", workService.CachedGetWork)                   // GET /api/v1/work/{uuid} (permanent)
			modern.GET("/:work_id/chapters", workService.GetChapters)            // GET /api/v1/work/{uuid}/chapters
			modern.GET("/:work_id/chapters/:chapter_id", workService.GetChapter) // GET /api/v1/work/{uuid}/chapters/{uuid}
			modern.GET("/:work_id/comments", workService.GetComments)            // GET /api/v1/work/{uuid}/comments
			modern.GET("/:work_id/kudos", workService.GetKudos)                  // GET /api/v1/work/{uuid}/kudos
			modern.GET("/:work_id/stats", workService.CachedGetWorkStats)        // GET /api/v1/work/{uuid}/stats
			modern.POST("/:work_id/comments", workService.CreateComment)         // POST /api/v1/work/{uuid}/comments (guest + auth comments)
		}

		// Series endpoints
		series := api.Group("/series")
		{
			series.GET("", workService.SearchSeries)                    // GET /api/v1/series?q=search
			series.GET("/:series_id", workService.GetSeries)            // GET /api/v1/series/123
			series.GET("/:series_id/works", workService.GetSeriesWorks) // GET /api/v1/series/123/works
		}

		// Collections endpoints
		collections := api.Group("/collections")
		{
			collections.GET("", workService.SearchCollections)                       // GET /api/v1/collections
			collections.GET("/:collection_id", workService.GetCollection)            // GET /api/v1/collections/123
			collections.GET("/:collection_id/works", workService.GetCollectionWorks) // GET /api/v1/collections/123/works
		}

		// Tag search endpoints (enhanced partial matching)
		tags := api.Group("/tags")
		{
			tags.GET("/search", workService.SearchTags) // GET /api/v1/tags/search?q=flu&limit=10
		}

		// User-specific endpoints
		users := api.Group("/users")
		{
			users.GET("/:user_id/works", workService.GetUserWorks)         // GET /api/v1/users/123/works
			users.GET("/:user_id/series", workService.GetUserSeries)       // GET /api/v1/users/123/series
			users.GET("/:user_id/bookmarks", workService.GetUserBookmarks) // GET /api/v1/users/123/bookmarks
		}

		// Authenticated endpoints
		protected := api.Group("")
		protected.Use(JWTAuthMiddleware())
		{
			// Work management
			protected.POST("/works", workService.CreateWorkEnhanced)                            // POST /api/v1/works
			protected.PUT("/works/:work_id", workService.UpdateWork)                            // PUT /api/v1/works/123
			protected.DELETE("/works/:work_id", workService.DeleteWork)                         // DELETE /api/v1/works/123
			protected.POST("/works/:work_id/chapters", workService.CreateChapter)               // POST /api/v1/works/123/chapters
			protected.PUT("/works/:work_id/chapters/:chapter_id", workService.UpdateChapter)    // PUT /api/v1/works/123/chapters/1
			protected.DELETE("/works/:work_id/chapters/:chapter_id", workService.DeleteChapter) // DELETE /api/v1/works/123/chapters/1

			// Engagement
			protected.POST("/works/:work_id/kudos", workService.GiveKudos)     // POST /api/v1/works/123/kudos
			protected.DELETE("/works/:work_id/kudos", workService.RemoveKudos) // DELETE /api/v1/works/123/kudos
			// Note: Comment creation moved to legacy/modern groups to support guest comments
			protected.PUT("/comments/:comment_id", workService.UpdateComment)    // PUT /api/v1/comments/123
			protected.DELETE("/comments/:comment_id", workService.DeleteComment) // DELETE /api/v1/comments/123

			// Bookmarks
			protected.POST("/works/:work_id/bookmark", workService.CreateBookmark)          // POST /api/v1/works/123/bookmark
			protected.GET("/works/:work_id/bookmark-status", workService.GetBookmarkStatus) // GET /api/v1/works/123/bookmark-status
			protected.PUT("/bookmarks/:bookmark_id", workService.UpdateBookmark)            // PUT /api/v1/bookmarks/123
			protected.DELETE("/bookmarks/:bookmark_id", workService.DeleteBookmark)         // DELETE /api/v1/bookmarks/123
			protected.GET("/bookmarks", workService.GetMyBookmarks)                         // GET /api/v1/bookmarks

			// Series management
			protected.POST("/series", workService.CreateSeries)                                     // POST /api/v1/series
			protected.PUT("/series/:series_id", workService.UpdateSeries)                           // PUT /api/v1/series/123
			protected.DELETE("/series/:series_id", workService.DeleteSeries)                        // DELETE /api/v1/series/123
			protected.POST("/series/:series_id/works/:work_id", workService.AddWorkToSeries)        // POST /api/v1/series/123/works/456
			protected.DELETE("/series/:series_id/works/:work_id", workService.RemoveWorkFromSeries) // DELETE /api/v1/series/123/works/456

			// Collections management
			protected.POST("/collections", workService.CreateCollection)                                         // POST /api/v1/collections
			protected.PUT("/collections/:collection_id", workService.UpdateCollection)                           // PUT /api/v1/collections/123
			protected.DELETE("/collections/:collection_id", workService.DeleteCollection)                        // DELETE /api/v1/collections/123
			protected.POST("/collections/:collection_id/works/:work_id", workService.AddWorkToCollection)        // POST /api/v1/collections/123/works/456
			protected.DELETE("/collections/:collection_id/works/:work_id", workService.RemoveWorkFromCollection) // DELETE /api/v1/collections/123/works/456

			// Comment moderation
			protected.PUT("/comments/:comment_id/moderate", workService.ModerateComment) // PUT /api/v1/comments/123/moderate

			// User blocking and reports
			protected.POST("/users/:user_id/block", workService.BlockUser)            // POST /api/v1/users/123/block
			protected.DELETE("/users/:user_id/block", workService.UnblockUser)        // DELETE /api/v1/users/123/block
			protected.POST("/comments/:comment_id/report", workService.ReportComment) // POST /api/v1/comments/123/report
			protected.POST("/works/:work_id/report", workService.ReportWork)          // POST /api/v1/works/123/report

			// User muting (matching AO3's implementation)
			protected.POST("/users/:user_id/mute", workService.MuteUser)            // POST /api/v1/users/123/mute
			protected.DELETE("/users/:user_id/mute", workService.UnmuteUser)        // DELETE /api/v1/users/123/mute
			protected.GET("/users/:user_id/mute-status", workService.GetMuteStatus) // GET /api/v1/users/123/mute-status
			protected.GET("/my/muted-users", workService.GetMutedUsers)             // GET /api/v1/my/muted-users

			// Core AO3 Features: Pseuds, Gifting, Orphaning, Co-authors
			protected.POST("/pseuds", workService.CreatePseud)                    // POST /api/v1/pseuds
			protected.GET("/my/pseuds", workService.GetUserPseuds)                // GET /api/v1/my/pseuds
			protected.POST("/works/:work_id/gift", workService.GiftWork)          // POST /api/v1/works/123/gift
			protected.GET("/works/:work_id/gifts", workService.GetWorkGifts)      // GET /api/v1/works/123/gifts
			protected.POST("/works/:work_id/orphan", workService.OrphanWork)      // POST /api/v1/works/123/orphan
			protected.GET("/works/:work_id/authors", workService.GetWorkAuthors)  // GET /api/v1/works/123/authors
			protected.POST("/works/:work_id/co-authors", workService.AddCoAuthor) // POST /api/v1/works/123/co-authors

			// User dashboard
			protected.GET("/my/works", workService.GetMyWorks)             // GET /api/v1/my/works
			protected.GET("/my/series", workService.GetMySeries)           // GET /api/v1/my/series
			protected.GET("/my/collections", workService.GetMyCollections) // GET /api/v1/my/collections
			protected.GET("/my/comments", workService.GetMyComments)       // GET /api/v1/my/comments
			protected.GET("/my/stats", workService.GetMyStats)             // GET /api/v1/my/stats
		}

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(JWTAuthMiddleware())
		admin.Use(RequireRoleMiddleware("admin"))
		{
			admin.GET("/works", workService.AdminListWorks)                                 // GET /api/v1/admin/works
			admin.PUT("/works/:work_id/status", workService.AdminUpdateWorkStatus)          // PUT /api/v1/admin/works/123/status
			admin.DELETE("/works/:work_id", workService.AdminDeleteWork)                    // DELETE /api/v1/admin/works/123
			admin.GET("/comments", workService.AdminListComments)                           // GET /api/v1/admin/comments
			admin.PUT("/comments/:comment_id/status", workService.AdminUpdateCommentStatus) // PUT /api/v1/admin/comments/123/status
			admin.DELETE("/comments/:comment_id", workService.AdminDeleteComment)           // DELETE /api/v1/admin/comments/123
			admin.GET("/reports", workService.AdminGetReports)                              // GET /api/v1/admin/reports
			admin.GET("/statistics", workService.AdminGetStatistics)                        // GET /api/v1/admin/statistics
		}
	}

	return r
}

// WorkService holds all dependencies for work management
type WorkService struct {
	db    *sql.DB
	redis *redis.Client
	cache *cache.Cache
}

func NewWorkService() *WorkService {
	// Database connection
	dbURL := getEnv("DATABASE_URL", "postgres://ao3_user:ao3_password@localhost/ao3_nuclear?sslmode=disable")
	log.Printf("DATABASE_URL: %s", dbURL)
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Set optimized connection pool settings for budget hosting
	db.SetMaxOpenConns(10) // Reduced from 25 to 10
	db.SetMaxIdleConns(3)  // Reduced from 5 to 3
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(15 * time.Minute) // Add idle timeout

	// Redis connection
	redisURL := getEnv("REDIS_URL", "localhost:6379")
	rdb := redis.NewClient(&redis.Options{
		Addr:         redisURL,
		Password:     getEnv("REDIS_PASSWORD", ""),
		DB:           1, // Use DB 1 for work service
		PoolSize:     10,
		MinIdleConns: 2,
		MaxRetries:   3,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	// Initialize cache
	workCache := cache.NewCache(rdb, "work-service")

	// Validate database schema at startup
	validator := NewSchemaValidator(db)
	if err := validator.ValidateAllSchemas(); err != nil {
		log.Fatal("❌ Schema validation failed:", err)
	}

	log.Println("Work service initialized successfully")

	return &WorkService{
		db:    db,
		redis: rdb,
		cache: workCache,
	}
}

func (ws *WorkService) Close() {
	if ws.db != nil {
		ws.db.Close()
	}
	if ws.redis != nil {
		ws.redis.Close()
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Middleware functions (simplified versions - would normally be in shared package)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:3002",
			"https://nuclear-ao3.com",
			"https://www.nuclear-ao3.com",
		}

		isAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				isAllowed = true
				break
			}
		}

		if isAllowed || getEnv("GIN_MODE", "debug") == "debug" {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Next()
	}
}

func LoggingMiddleware() gin.HandlerFunc {
	return gin.Logger()
}

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if this route should be exempt from JWT auth for guest comments
		path := c.Request.URL.Path
		method := c.Request.Method

		// Allow guest comment creation without auth
		if method == "POST" && (strings.HasSuffix(path, "/comments") || strings.Contains(path, "/guest-comments/")) {
			c.Next()
			return
		}

		// Extract and validate JWT token to get real user ID
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization header"})
			c.Abort()
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// For now, make a request to auth service to validate token and get user ID
		// In production, this would use shared JWT validation
		userID, err := validateTokenWithAuthService(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

func validateTokenWithAuthService(tokenString string) (string, error) {
	// Make request to auth service to validate token and get user info
	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://ao3_auth_service:8081")
	log.Printf("DEBUG: Using auth service URL: %s", authServiceURL)

	req, err := http.NewRequest("GET", authServiceURL+"/api/v1/auth/me", nil)
	if err != nil {
		log.Printf("DEBUG: Failed to create request: %v", err)
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+tokenString)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("DEBUG: Failed to make request to auth service: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	log.Printf("DEBUG: Auth service response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		log.Printf("DEBUG: Auth service returned non-200 status: %d", resp.StatusCode)
		return "", fmt.Errorf("auth service returned status %d", resp.StatusCode)
	}

	var result struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("DEBUG: Failed to decode response: %v", err)
		return "", err
	}

	log.Printf("DEBUG: Successfully got user ID: %s", result.UserID)
	return result.UserID, nil
}

func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract and validate JWT token if present, but don't require it
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			// No token provided - continue without user context
			c.Next()
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Try to validate token and get user ID
		userID, err := validateTokenWithAuthService(tokenString)
		if err != nil {
			// Invalid token - continue without user context (don't block access)
			c.Next()
			return
		}

		// Valid token - set user context
		c.Set("user_id", userID)
		c.Next()
	}
}

func RequireRoleMiddleware(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Role validation - would integrate with auth service
		c.Next()
	}
}
