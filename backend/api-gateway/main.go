package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// =============================================================================
// NUCLEAR AO3 API GATEWAY
// High-performance, bulletproof GraphQL API that unifies all microservices
// =============================================================================

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize API Gateway
	gateway := NewAPIGateway()
	defer gateway.Close()

	// Setup router with all middleware and endpoints
	router := setupRouter(gateway)

	// Configure server for production
	srv := &http.Server{
		Addr:           ":" + getEnv("GATEWAY_PORT", "8080"),
		Handler:        router,
		ReadTimeout:    time.Second * 30,  // Generous for large GraphQL queries
		WriteTimeout:   time.Second * 30,  // Allow complex query processing
		IdleTimeout:    time.Second * 120, // Keep connections alive
		MaxHeaderBytes: 1 << 21,           // 2MB headers for large GraphQL queries
	}

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Nuclear AO3 API Gateway starting on port %s", getEnv("GATEWAY_PORT", "8080"))
		log.Printf("📊 GraphQL Playground: http://localhost:%s/graphql", getEnv("GATEWAY_PORT", "8080"))
		log.Printf("📈 Metrics: http://localhost:%s/metrics", getEnv("GATEWAY_PORT", "8080"))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start API Gateway: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down API Gateway...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("❌ API Gateway forced to shutdown:", err)
	}

	log.Println("✅ API Gateway exited cleanly")
}

// =============================================================================
// API GATEWAY CORE STRUCTURE
// =============================================================================

// APIGateway manages all microservice connections and GraphQL resolution
type APIGateway struct {
	// Service clients
	authService   *ServiceClient
	workService   *ServiceClient
	tagService    *ServiceClient
	searchService *ServiceClient

	// Infrastructure
	redis *redis.Client

	// Performance & Monitoring
	metrics     *GatewayMetrics
	rateLimiter *RateLimiter
	cache       *CacheManager

	// GraphQL
	schema *GraphQLSchema
}

// ServiceClient represents a connection to a microservice
type ServiceClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Name       string
	Health     ServiceHealthStatus
}

// ServiceHealthStatus tracks service availability
type ServiceHealthStatus struct {
	IsHealthy    bool
	LastCheck    time.Time
	ResponseTime time.Duration
	ErrorCount   int
	LastError    error
}

// NewAPIGateway initializes the API Gateway with all dependencies
func NewAPIGateway() *APIGateway {
	log.Println("🔧 Initializing Nuclear AO3 API Gateway...")

	// Initialize Redis for caching and rate limiting
	redis := initializeRedis()

	// Initialize service clients
	authService := &ServiceClient{
		BaseURL:    getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		HTTPClient: createOptimizedHTTPClient(),
		Name:       "auth-service",
	}

	workService := &ServiceClient{
		BaseURL:    getEnv("WORK_SERVICE_URL", "http://localhost:8082"),
		HTTPClient: createOptimizedHTTPClient(),
		Name:       "work-service",
	}

	tagService := &ServiceClient{
		BaseURL:    getEnv("TAG_SERVICE_URL", "http://localhost:8083"),
		HTTPClient: createOptimizedHTTPClient(),
		Name:       "tag-service",
	}

	searchService := &ServiceClient{
		BaseURL:    getEnv("SEARCH_SERVICE_URL", "http://localhost:8084"),
		HTTPClient: createOptimizedHTTPClient(),
		Name:       "search-service",
	}

	// Initialize performance components
	metrics := initializeMetrics()
	rateLimiter := NewRateLimiter(redis)
	cache := NewCacheManager(redis)

	// Connect cache to metrics
	cache.SetMetrics(metrics)

	gateway := &APIGateway{
		authService:   authService,
		workService:   workService,
		tagService:    tagService,
		searchService: searchService,
		redis:         redis,
		metrics:       metrics,
		rateLimiter:   rateLimiter,
		cache:         cache,
	}

	// Health check all services
	gateway.checkServiceHealth()

	// Initialize GraphQL schema
	gateway.schema = NewGraphQLSchema(gateway)

	log.Println("✅ API Gateway initialized successfully")
	return gateway
}

// =============================================================================
// HTTP CLIENT OPTIMIZATION
// =============================================================================

// createOptimizedHTTPClient creates a high-performance HTTP client for service communication
func createOptimizedHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,              // Pool connections
			MaxIdleConnsPerHost: 10,               // Per service
			IdleConnTimeout:     30 * time.Second, // Keep alive
			DisableCompression:  false,            // Enable compression
			ForceAttemptHTTP2:   true,             // Use HTTP/2 when possible
		},
	}
}

// =============================================================================
// REDIS INITIALIZATION
// =============================================================================

// initializeRedis sets up Redis connection with optimized settings
func initializeRedis() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         getEnv("REDIS_URL", "localhost:6379"),
		Password:     getEnv("REDIS_PASSWORD", ""),
		DB:           0, // API Gateway uses DB 0
		PoolSize:     20,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️ Redis connection failed (continuing without cache): %v", err)
		return nil
	}

	log.Println("✅ Redis connected successfully")
	return rdb
}

// =============================================================================
// ROUTER SETUP WITH MIDDLEWARE
// =============================================================================

// setupRouter configures the Gin router with all middleware and endpoints
func setupRouter(gateway *APIGateway) *gin.Engine {
	// Set Gin mode based on environment
	if getEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Core middleware stack
	r.Use(gin.Recovery())
	r.Use(CORSMiddleware())
	r.Use(LoggingMiddleware())
	r.Use(SecurityHeadersMiddleware())
	r.Use(MetricsMiddleware(gateway.metrics))

	// Health check endpoint
	r.GET("/health", gateway.HealthCheck)

	// Metrics endpoint for monitoring
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Service status endpoint (admin only)
	r.GET("/status", gateway.ServiceStatus)

	// GraphQL endpoints
	graphql := r.Group("/graphql")
	{
		// Main GraphQL endpoint
		graphql.POST("", gateway.RateLimitMiddleware(), gateway.GraphQLHandler)
		graphql.GET("", gateway.GraphQLPlaygroundHandler)

		// GraphQL subscriptions (WebSocket)
		graphql.GET("/ws", gateway.GraphQLSubscriptionHandler)
	}

	// REST API fallback endpoints (for compatibility)
	api := r.Group("/api/v1")
	api.Use(gateway.RateLimitMiddleware())
	api.Use(JWTAuthMiddleware()) // Add JWT authentication middleware
	{
		// Authentication - proxy everything under /auth
		auth := api.Group("/auth")
		{
			auth.Any("/*path", gateway.ProxyToAuth)
		}

		// Works - handle both exact match and sub-paths
		works := api.Group("/works")
		{
			works.Any("", gateway.ProxyToWork)       // Exact match /works
			works.Any("/*path", gateway.ProxyToWork) // Sub-paths /works/*
		}

		// Tags - proxy everything under /tags
		tags := api.Group("/tags")
		{
			tags.Any("/*path", gateway.ProxyToTag)
		}

		// Search - proxy everything under /search
		search := api.Group("/search")
		{
			search.Any("/*path", gateway.ProxyToSearch)
		}

		// My endpoints - proxy to work service (user-specific endpoints)
		my := api.Group("/my")
		{
			my.Any("/*path", gateway.ProxyToWork)
		}

		// User endpoints - proxy to work service
		users := api.Group("/users")
		{
			users.Any("/*path", gateway.ProxyToWork)
		}

		// Series endpoints - proxy to work service
		series := api.Group("/series")
		{
			series.Any("/*path", gateway.ProxyToWork)
		}

		// Collections endpoints - proxy to work service
		collections := api.Group("/collections")
		{
			collections.Any("/*path", gateway.ProxyToWork)
		}

		// Bookmarks endpoints - proxy to work service
		bookmarks := api.Group("/bookmarks")
		{
			bookmarks.Any("/*path", gateway.ProxyToWork)
		}

		// Comments endpoints - proxy to work service
		comments := api.Group("/comments")
		{
			comments.Any("/*path", gateway.ProxyToWork)
		}

		// Pseuds endpoints - proxy to work service
		pseuds := api.Group("/pseuds")
		{
			pseuds.Any("/*path", gateway.ProxyToWork)
		}
	}

	return r
}

// =============================================================================
// HEALTH CHECKING
// =============================================================================

// HealthCheck returns the overall health of the API Gateway and services
func (gw *APIGateway) HealthCheck(c *gin.Context) {
	// Count unhealthy services
	unhealthyCount := 0
	totalServices := 0
	for _, service := range []*ServiceClient{gw.authService, gw.workService, gw.tagService, gw.searchService} {
		totalServices++
		if !service.Health.IsHealthy {
			unhealthyCount++
		}
	}

	// Determine overall system status
	var overallStatus string
	if unhealthyCount == 0 {
		overallStatus = "healthy"
	} else if unhealthyCount < totalServices {
		overallStatus = "degraded" // Some services down
	} else {
		overallStatus = "outage" // All services down
	}

	healthStatus := map[string]interface{}{
		"status":    overallStatus,
		"gateway":   "healthy", // API Gateway itself is responding
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
		"uptime":    time.Since(startTime).Seconds(),
		"services":  gw.getServiceHealthSummary(), // Flat structure for UI compatibility
	}

	// Always return 200 OK since the API Gateway itself is healthy and responding
	c.JSON(http.StatusOK, healthStatus)
}

// ServiceStatus returns detailed status of all services (admin endpoint)
func (gw *APIGateway) ServiceStatus(c *gin.Context) {
	// In production, this would require admin authentication
	services := map[string]interface{}{
		"auth-service":   gw.authService.Health,
		"work-service":   gw.workService.Health,
		"tag-service":    gw.tagService.Health,
		"search-service": gw.searchService.Health,
	}

	c.JSON(http.StatusOK, gin.H{
		"services": services,
		"redis": gin.H{
			"connected": gw.redis != nil,
		},
		"timestamp": time.Now(),
	})
}

// checkServiceHealth performs health checks on all services
func (gw *APIGateway) checkServiceHealth() {
	services := []*ServiceClient{gw.authService, gw.workService, gw.tagService, gw.searchService}

	for _, service := range services {
		go gw.healthCheckService(service)
	}
}

// healthCheckService performs a health check on a single service
func (gw *APIGateway) healthCheckService(service *ServiceClient) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", service.BaseURL+"/health", nil)
	if err != nil {
		service.Health.IsHealthy = false
		service.Health.LastError = err
		service.Health.ErrorCount++
		return
	}

	resp, err := service.HTTPClient.Do(req)
	if err != nil {
		service.Health.IsHealthy = false
		service.Health.LastError = err
		service.Health.ErrorCount++
		log.Printf("⚠️ Health check failed for %s: %v", service.Name, err)
		return
	}
	defer resp.Body.Close()

	service.Health.ResponseTime = time.Since(start)
	service.Health.LastCheck = time.Now()

	if resp.StatusCode == http.StatusOK {
		service.Health.IsHealthy = true
		service.Health.LastError = nil
		log.Printf("✅ %s is healthy (response time: %v)", service.Name, service.Health.ResponseTime)
	} else {
		service.Health.IsHealthy = false
		service.Health.ErrorCount++
		log.Printf("⚠️ %s returned status %d", service.Name, resp.StatusCode)
	}
}

// getServiceHealthSummary returns a summary of all service health statuses
func (gw *APIGateway) getServiceHealthSummary() map[string]string {
	return map[string]string{
		"auth-service":   gw.getServiceStatusString(gw.authService),
		"work-service":   gw.getServiceStatusString(gw.workService),
		"tag-service":    gw.getServiceStatusString(gw.tagService),
		"search-service": gw.getServiceStatusString(gw.searchService),
	}
}

// getServiceStatusString returns a human-readable status string for a service
func (gw *APIGateway) getServiceStatusString(service *ServiceClient) string {
	if service.Health.IsHealthy {
		return fmt.Sprintf("healthy (%v)", service.Health.ResponseTime)
	}
	return "unhealthy"
}

// =============================================================================
// CLEANUP
// =============================================================================

// Close gracefully shuts down the API Gateway
func (gw *APIGateway) Close() {
	if gw.redis != nil {
		gw.redis.Close()
	}
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

var startTime = time.Now()

// getEnv gets environment variable with fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
