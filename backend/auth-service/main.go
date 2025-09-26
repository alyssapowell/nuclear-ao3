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
	authService := NewAuthService()
	defer authService.Close()

	// Setup router
	router := setupRouter(authService)

	// Setup server
	srv := &http.Server{
		Addr:           ":" + getEnv("PORT", "8081"),
		Handler:        router,
		ReadTimeout:    time.Second * 15,
		WriteTimeout:   time.Second * 15,
		IdleTimeout:    time.Second * 60,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server in goroutine
	go func() {
		log.Printf("Auth service starting on port %s", getEnv("PORT", "8081"))
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

func setupRouter(authService *AuthService) *gin.Engine {
	// Set Gin mode
	if getEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(CORSMiddleware())
	r.Use(LoggingMiddleware())
	r.Use(RateLimitMiddleware(authService.redis))
	r.Use(SecurityHeadersMiddleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":   "auth-service",
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	// Metrics endpoint for monitoring
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Auth endpoints
	api := r.Group("/api/v1/auth")
	{
		// Public endpoints (no authentication required)
		api.POST("/register", authService.Register)
		api.POST("/login", authService.Login)
		api.POST("/refresh", authService.RefreshToken)
		api.POST("/reset-password", authService.RequestPasswordReset)
		api.POST("/reset-password/confirm", authService.ConfirmPasswordReset)
		api.POST("/verify-email", authService.VerifyEmail)
		api.POST("/resend-verification", authService.ResendVerification)

		// Protected endpoints (require authentication)
		protected := api.Group("")
		protected.Use(JWTAuthMiddleware(authService))
		{
			protected.POST("/logout", authService.Logout)
			protected.GET("/me", authService.GetProfile)
			protected.PUT("/me", authService.UpdateProfile)
			protected.POST("/change-password", authService.ChangePassword)
			protected.GET("/sessions", authService.GetSessions)
			protected.DELETE("/sessions/:session_id", authService.RevokeSession)
			protected.GET("/security-events", authService.GetSecurityEvents)
		}

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(JWTAuthMiddleware(authService))
		admin.Use(RequireRoleMiddleware("admin"))
		{
			admin.GET("/users", authService.ListUsers)
			admin.GET("/users/:user_id", authService.GetUser)
			admin.PUT("/users/:user_id", authService.UpdateUser)
			admin.POST("/users/:user_id/roles", authService.GrantRole)
			admin.DELETE("/users/:user_id/roles/:role", authService.RevokeRole)
			admin.GET("/security-events", authService.GetAllSecurityEvents)
			admin.GET("/metrics", authService.GetAuthMetrics)
		}
	}

	return r
}

// AuthService holds all dependencies for authentication
type AuthService struct {
	db    *sql.DB
	redis *redis.Client
	jwt   *JWTManager
}

func NewAuthService() *AuthService {
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
		DB:           0,
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

	// JWT manager
	jwtManager, err := NewJWTManager(
		getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),
		getEnv("JWT_ISSUER", "nuclear-ao3"),
	)
	if err != nil {
		log.Fatal("Failed to create JWT manager:", err)
	}

	log.Println("Auth service initialized successfully")
	
	return &AuthService{
		db:    db,
		redis: rdb,
		jwt:   jwtManager,
	}
}

func (as *AuthService) Close() {
	if as.db != nil {
		as.db.Close()
	}
	if as.redis != nil {
		as.redis.Close()
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Allow specific origins in production
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
		c.Header("Access-Control-Max-Age", "86400") // 24 hours

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}

// LoggingMiddleware provides structured logging
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC3339),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}