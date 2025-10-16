package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"nuclear-ao3/shared/middleware"
)

// ServiceInfo holds basic service information
type ServiceInfo struct {
	Name    string
	Version string
}

// SetupBaseRouter creates a Gin router with common middleware and health check
func SetupBaseRouter(serviceInfo ServiceInfo, redisClient *redis.Client) *gin.Engine {
	// Set Gin mode based on environment
	if gin.Mode() == "" {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()

	// Common middleware
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware())

	if redisClient != nil {
		r.Use(RateLimitMiddleware(redisClient))
	}

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":   serviceInfo.Name,
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   serviceInfo.Version,
		})
	})

	return r
}

// RateLimitMiddleware provides basic rate limiting (simplified version)
func RateLimitMiddleware(redis *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simple rate limiting - would be more sophisticated in production
		// For now, just pass through
		c.Next()
	}
}
