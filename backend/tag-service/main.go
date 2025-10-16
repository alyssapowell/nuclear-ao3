package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize services
	tagService := NewTagService()
	defer tagService.Close()

	// Setup router
	router := setupRouter(tagService)

	// Setup server
	srv := &http.Server{
		Addr:           ":" + getEnv("PORT", "8083"),
		Handler:        router,
		ReadTimeout:    time.Second * 15,
		WriteTimeout:   time.Second * 15,
		IdleTimeout:    time.Second * 60,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server in goroutine
	go func() {
		log.Printf("Tag service starting on port %s", getEnv("PORT", "8083"))
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

func setupRouter(tagService *TagService) *gin.Engine {
	// Set Gin mode
	if getEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(CORSMiddleware())
	r.Use(LoggingMiddleware())
	r.Use(RateLimitMiddleware(tagService.redis))
	r.Use(SecurityHeadersMiddleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":   "tag-service",
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	// Debug endpoint
	r.GET("/debug", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"debug":     "Tag service debug endpoint working",
			"timestamp": time.Now().Unix(),
		})
	})

	// Metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API endpoints
	api := r.Group("/api/v1")
	{
		// Public endpoints (no auth required)
		tags := api.Group("/tags")
		{
			tags.GET("", tagService.SearchTags)        // GET /api/v1/tags?q=search&type=fandom
			tags.GET("/search", tagService.SearchTags) // GET /api/v1/tags/search?q=search&type=fandom (frontend compatibility)
			tags.GET("/debug", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"debug": "Tags debug working", "timestamp": time.Now().Unix()})
			})
			tags.GET("/:tag_id", tagService.GetTag)                 // GET /api/v1/tags/123
			tags.GET("/:tag_id/related", tagService.GetRelatedTags) // GET /api/v1/tags/123/related
			tags.GET("/:tag_id/works", tagService.GetTagWorks)      // GET /api/v1/tags/123/works
			tags.GET("/autocomplete", tagService.AutocompleteTags)  // GET /api/v1/tags/autocomplete?q=harry
		}

		// Fandoms
		fandoms := api.Group("/fandoms")
		{
			fandoms.GET("", tagService.SearchFandoms)                                   // GET /api/v1/fandoms?q=search
			fandoms.GET("/:fandom_id", tagService.GetFandom)                            // GET /api/v1/fandoms/123
			fandoms.GET("/:fandom_id/tags", tagService.GetFandomTags)                   // GET /api/v1/fandoms/123/tags
			fandoms.GET("/:fandom_id/characters", tagService.GetFandomCharacters)       // GET /api/v1/fandoms/123/characters
			fandoms.GET("/:fandom_id/relationships", tagService.GetFandomRelationships) // GET /api/v1/fandoms/123/relationships
		}

		// Characters
		characters := api.Group("/characters")
		{
			characters.GET("", tagService.SearchCharacters)                                      // GET /api/v1/characters?q=search&fandom=123
			characters.GET("/:character_id", tagService.GetCharacter)                            // GET /api/v1/characters/123
			characters.GET("/:character_id/relationships", tagService.GetCharacterRelationships) // GET /api/v1/characters/123/relationships
		}

		// Relationships
		relationships := api.Group("/relationships")
		{
			relationships.GET("", tagService.SearchRelationships)              // GET /api/v1/relationships?q=search&fandom=123
			relationships.GET("/:relationship_id", tagService.GetRelationship) // GET /api/v1/relationships/123
		}

		// Tag hierarchies and navigation
		hierarchy := api.Group("/hierarchy")
		{
			hierarchy.GET("/fandoms", tagService.GetFandomHierarchy)             // GET /api/v1/hierarchy/fandoms
			hierarchy.GET("/characters", tagService.GetCharacterHierarchy)       // GET /api/v1/hierarchy/characters
			hierarchy.GET("/relationships", tagService.GetRelationshipHierarchy) // GET /api/v1/hierarchy/relationships
		}

		// Tag statistics
		stats := api.Group("/stats")
		{
			stats.GET("/popular", tagService.GetPopularTags)   // GET /api/v1/stats/popular?type=fandom&limit=50
			stats.GET("/trending", tagService.GetTrendingTags) // GET /api/v1/stats/trending?period=week
			stats.GET("/usage", tagService.GetTagUsageStats)   // GET /api/v1/stats/usage
		}

		// Authenticated endpoints
		protected := api.Group("")
		protected.Use(JWTAuthMiddleware())
		{
			// Tag management (user submissions)
			protected.POST("/tags", tagService.CreateTag)                     // POST /api/v1/tags
			protected.PUT("/tags/:tag_id", tagService.UpdateTag)              // PUT /api/v1/tags/123
			protected.POST("/tags/:tag_id/synonym", tagService.CreateSynonym) // POST /api/v1/tags/123/synonym
			protected.POST("/tags/merge", tagService.RequestTagMerge)         // POST /api/v1/tags/merge

			// User tag relationships
			protected.POST("/user/tags/follow", tagService.FollowTag)             // POST /api/v1/user/tags/follow
			protected.DELETE("/user/tags/follow/:tag_id", tagService.UnfollowTag) // DELETE /api/v1/user/tags/follow/123
			protected.GET("/user/tags/followed", tagService.GetFollowedTags)      // GET /api/v1/user/tags/followed

			// Tag reports
			protected.POST("/tags/:tag_id/report", tagService.ReportTag) // POST /api/v1/tags/123/report
		}

		// Wrangler endpoints (tag wranglers have special permissions)
		wrangler := api.Group("/wrangling")
		wrangler.Use(JWTAuthMiddleware())
		wrangler.Use(RequireRoleMiddleware("tag_wrangler", "admin"))
		{
			wrangler.GET("/queue", tagService.GetWranglingQueue)                           // GET /api/v1/wrangling/queue
			wrangler.GET("/tags/:tag_id", tagService.GetTagForWrangling)                   // GET /api/v1/wrangling/tags/123
			wrangler.POST("/tags/:tag_id/wrangle", tagService.WrangleTag)                  // POST /api/v1/wrangling/tags/123/wrangle
			wrangler.POST("/tags/:tag_id/canonical", tagService.MakeCanonical)             // POST /api/v1/wrangling/tags/123/canonical
			wrangler.POST("/tags/:tag_id/synonym", tagService.CreateCanonicalSynonym)      // POST /api/v1/wrangling/tags/123/synonym
			wrangler.POST("/tags/:tag_id/parent", tagService.AddParentTag)                 // POST /api/v1/wrangling/tags/123/parent
			wrangler.DELETE("/tags/:tag_id/parent/:parent_id", tagService.RemoveParentTag) // DELETE /api/v1/wrangling/tags/123/parent/456
			wrangler.PUT("/merge/:merge_id", tagService.ProcessTagMerge)                   // PUT /api/v1/wrangling/merge/123
			wrangler.GET("/reports", tagService.GetTagReports)                             // GET /api/v1/wrangling/reports
			wrangler.PUT("/reports/:report_id", tagService.ProcessTagReport)               // PUT /api/v1/wrangling/reports/123
		}

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(JWTAuthMiddleware())
		admin.Use(RequireRoleMiddleware("admin"))
		{
			admin.GET("/tags", tagService.AdminListTags)                        // GET /api/v1/admin/tags
			admin.DELETE("/tags/:tag_id", tagService.AdminDeleteTag)            // DELETE /api/v1/admin/tags/123
			admin.POST("/tags/:tag_id/ban", tagService.AdminBanTag)             // POST /api/v1/admin/tags/123/ban
			admin.DELETE("/tags/:tag_id/ban", tagService.AdminUnbanTag)         // DELETE /api/v1/admin/tags/123/ban
			admin.GET("/wranglers", tagService.AdminListWranglers)              // GET /api/v1/admin/wranglers
			admin.POST("/wranglers", tagService.AdminAddWrangler)               // POST /api/v1/admin/wranglers
			admin.DELETE("/wranglers/:user_id", tagService.AdminRemoveWrangler) // DELETE /api/v1/admin/wranglers/123
			admin.GET("/statistics", tagService.AdminGetTagStatistics)          // GET /api/v1/admin/statistics
		}
	}

	return r
}

// TagService holds all dependencies for tag management
type TagService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewTagService() *TagService {
	// Database connection
	dbURL := getEnv("DATABASE_URL", "postgres://ao3_user:ao3_password@localhost/ao3_nuclear?sslmode=disable")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Redis connection
	redisURL := getEnv("REDIS_URL", "localhost:6379")
	rdb := redis.NewClient(&redis.Options{
		Addr:         redisURL,
		Password:     getEnv("REDIS_PASSWORD", ""),
		DB:           2, // Use DB 2 for tag service
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

	log.Println("Tag service initialized successfully")

	return &TagService{
		db:    db,
		redis: rdb,
	}
}

func (ts *TagService) Close() {
	if ts.db != nil {
		ts.db.Close()
	}
	if ts.redis != nil {
		ts.redis.Close()
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Middleware functions (simplified versions)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:3001",
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
		// JWT validation - would integrate with auth service
		c.Next()
	}
}

func RequireRoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Role validation - would integrate with auth service
		c.Next()
	}
}
