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

	"github.com/elastic/go-elasticsearch/v8"
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
	searchService := NewSearchService()
	defer searchService.Close()

	// Setup router
	router := setupRouter(searchService)

	// Setup server
	srv := &http.Server{
		Addr:           ":" + getEnv("PORT", "8084"),
		Handler:        router,
		ReadTimeout:    time.Second * 15,
		WriteTimeout:   time.Second * 15,
		IdleTimeout:    time.Second * 60,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server in goroutine
	go func() {
		log.Printf("Search service starting on port %s", getEnv("PORT", "8084"))
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

func setupRouter(searchService *SearchService) *gin.Engine {
	// Set Gin mode
	if getEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(CORSMiddleware())
	r.Use(LoggingMiddleware())
	r.Use(RateLimitMiddleware(searchService.redis))
	r.Use(SecurityHeadersMiddleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		// Check Elasticsearch health
		esStatus := "healthy"
		if _, err := searchService.es.Ping(); err != nil {
			esStatus = "unhealthy"
		}

		c.JSON(http.StatusOK, gin.H{
			"service":       "search-service",
			"status":        "healthy",
			"elasticsearch": esStatus,
			"timestamp":     time.Now().Unix(),
			"version":       "1.0.0",
		})
	})

	// Metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API endpoints
	api := r.Group("/api/v1")
	{
		// Search endpoints
		search := api.Group("/search")
		{
			// General search
			search.GET("/works", searchService.SearchWorks)             // GET /api/v1/search/works?q=harry+potter
			search.GET("/tags", searchService.SearchTags)               // GET /api/v1/search/tags?q=angst
			search.GET("/users", searchService.SearchUsers)             // GET /api/v1/search/users?q=author_name
			search.GET("/collections", searchService.SearchCollections) // GET /api/v1/search/collections?q=prompt_fest
			search.GET("/series", searchService.SearchSeries)           // GET /api/v1/search/series?q=trilogy

			// Advanced/filtered search
			search.POST("/works/advanced", searchService.AdvancedWorkSearch) // POST /api/v1/search/works/advanced
			search.POST("/tags/advanced", searchService.AdvancedTagSearch)   // POST /api/v1/search/tags/advanced

			// Enhanced smart filtering (Task 3)
			search.POST("/works/smart", searchService.SmartFilteredSearch)   // POST /api/v1/search/works/smart
			search.POST("/facets/smart", searchService.GetSmartFacets)       // POST /api/v1/search/facets/smart
			search.POST("/quality/analyze", searchService.AnalyzeTagQuality) // POST /api/v1/search/quality/analyze

			// Autocomplete/suggestions
			search.GET("/suggestions", searchService.GetSuggestions)   // GET /api/v1/search/suggestions?q=har
			search.GET("/popular", searchService.GetPopularSearches)   // GET /api/v1/search/popular
			search.GET("/trending", searchService.GetTrendingSearches) // GET /api/v1/search/trending
		}

		// Indexing operations (internal/admin only)
		index := api.Group("/index")
		index.Use(JWTAuthMiddleware())
		index.Use(RequireRoleMiddleware("admin", "indexer"))
		{
			// Enhanced work indexing operations
			index.POST("/works", searchService.EnhancedIndexWork)           // POST /api/v1/index/works
			index.PUT("/works/:id", searchService.EnhancedIndexWork)        // PUT /api/v1/index/works/123
			index.DELETE("/works/:id", searchService.DeleteWorkFromIndex)   // DELETE /api/v1/index/works/123
			index.POST("/works/bulk", searchService.EnhancedBulkIndexWorks) // POST /api/v1/index/works/bulk

			// Legacy tag indexing (to be enhanced)
			index.POST("/tags", searchService.IndexTag)                 // POST /api/v1/index/tags
			index.PUT("/tags/:tag_id", searchService.UpdateTagIndex)    // PUT /api/v1/index/tags/123
			index.DELETE("/tags/:tag_id", searchService.DeleteTagIndex) // DELETE /api/v1/index/tags/123
			index.POST("/tags/bulk", searchService.BulkIndexTags)       // POST /api/v1/index/tags/bulk

			// Legacy user indexing (to be enhanced)
			index.POST("/users", searchService.IndexUser)                  // POST /api/v1/index/users
			index.PUT("/users/:user_id", searchService.UpdateUserIndex)    // PUT /api/v1/index/users/123
			index.DELETE("/users/:user_id", searchService.DeleteUserIndex) // DELETE /api/v1/index/users/123

			// Enhanced index management
			index.POST("/rebuild", searchService.EnhancedRebuildIndex) // POST /api/v1/index/rebuild
			index.GET("/status", searchService.GetIndexingStatus)      // GET /api/v1/index/status
			index.POST("/optimize", searchService.OptimizeIndex)       // POST /api/v1/index/optimize

			// Tag enhancement suggestions
			index.GET("/works/:id/suggest-tags", searchService.SuggestTagEnhancements) // GET /api/v1/index/works/123/suggest-tags
		}

		// Analytics and insights
		analytics := api.Group("/analytics")
		analytics.Use(JWTAuthMiddleware())
		analytics.Use(RequireRoleMiddleware("admin"))
		{
			// Legacy analytics endpoints
			analytics.GET("/search-stats", searchService.GetSearchStats)      // GET /api/v1/analytics/search-stats
			analytics.GET("/popular-terms", searchService.GetPopularTerms)    // GET /api/v1/analytics/popular-terms
			analytics.GET("/zero-results", searchService.GetZeroResultTerms)  // GET /api/v1/analytics/zero-results
			analytics.GET("/performance", searchService.GetSearchPerformance) // GET /api/v1/analytics/performance

			// Enhanced analytics dashboard (Task 5)
			analytics.GET("/dashboard", searchService.GetAnalyticsDashboard)             // GET /api/v1/analytics/dashboard
			analytics.GET("/metrics/performance", searchService.GetPerformanceMetrics)   // GET /api/v1/analytics/metrics/performance
			analytics.GET("/trends", searchService.GetSearchTrends)                      // GET /api/v1/analytics/trends
			analytics.GET("/tag-quality", searchService.GetTagQualityInsights)           // GET /api/v1/analytics/tag-quality
			analytics.GET("/realtime", searchService.GetRealtimeMetrics)                 // GET /api/v1/analytics/realtime
			analytics.GET("/recommendations", searchService.GetAnalyticsRecommendations) // GET /api/v1/analytics/recommendations
		}

		// Search history and saved searches (authenticated users)
		protected := api.Group("")
		protected.Use(JWTAuthMiddleware())
		{
			protected.GET("/history", searchService.GetSearchHistory)                           // GET /api/v1/history
			protected.DELETE("/history", searchService.ClearSearchHistory)                      // DELETE /api/v1/history
			protected.POST("/saved-searches", searchService.SaveSearch)                         // POST /api/v1/saved-searches
			protected.GET("/saved-searches", searchService.GetSavedSearches)                    // GET /api/v1/saved-searches
			protected.DELETE("/saved-searches/:search_id", searchService.DeleteSavedSearch)     // DELETE /api/v1/saved-searches/123
			protected.POST("/saved-searches/:search_id/alert", searchService.CreateSearchAlert) // POST /api/v1/saved-searches/123/alert
		}

		// Search filters and facets
		filters := api.Group("/filters")
		{
			filters.GET("/fandoms", searchService.GetFandomFilters)             // GET /api/v1/filters/fandoms
			filters.GET("/characters", searchService.GetCharacterFilters)       // GET /api/v1/filters/characters
			filters.GET("/relationships", searchService.GetRelationshipFilters) // GET /api/v1/filters/relationships
			filters.GET("/tags", searchService.GetTagFilters)                   // GET /api/v1/filters/tags
			filters.GET("/stats", searchService.GetStatFilters)                 // GET /api/v1/filters/stats (word count ranges, etc)
		}
	}

	return r
}

// SearchService holds all dependencies for search functionality
type SearchService struct {
	db    *sql.DB
	redis *redis.Client
	es    *elasticsearch.Client
}

func NewSearchService() *SearchService {
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
		DB:           3, // Use DB 3 for search service
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

	// Elasticsearch connection
	esConfig := elasticsearch.Config{
		Addresses: []string{
			getEnv("ELASTICSEARCH_URL", "http://localhost:9200"),
		},
		Username: getEnv("ELASTICSEARCH_USERNAME", ""),
		Password: getEnv("ELASTICSEARCH_PASSWORD", ""),
		CloudID:  getEnv("ELASTICSEARCH_CLOUD_ID", ""),
		APIKey:   getEnv("ELASTICSEARCH_API_KEY", ""),
	}

	es, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		log.Fatal("Failed to create Elasticsearch client:", err)
	}

	// Test Elasticsearch connection
	if _, err := es.Ping(); err != nil {
		log.Fatal("Failed to connect to Elasticsearch:", err)
	}

	log.Println("Search service initialized successfully")

	return &SearchService{
		db:    db,
		redis: rdb,
		es:    es,
	}
}

func (ss *SearchService) Close() {
	if ss.db != nil {
		ss.db.Close()
	}
	if ss.redis != nil {
		ss.redis.Close()
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
