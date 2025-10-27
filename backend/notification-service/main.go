package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
	"nuclear-ao3/shared/messaging"
	"nuclear-ao3/shared/models"
	"nuclear-ao3/shared/notifications"
)

type NotificationService struct {
	db               *sql.DB
	notificationSvc  *NotificationServiceExtended
	messagingService messaging.MessageService
	wsUpgrader       websocket.Upgrader
	wsClients        map[string]*websocket.Conn // userID -> connection
	wsBroadcast      chan []byte
}

// NotificationServiceExtended adds additional methods to the notification service
type NotificationServiceExtended struct {
	*notifications.NotificationService
	subscriptionRepo notifications.SubscriptionRepository
	notificationRepo notifications.NotificationRepository
	preferenceRepo   notifications.PreferenceRepository
}

func (ns *NotificationServiceExtended) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return ns.notificationRepo.GetUnreadCount(ctx, userID)
}

func (ns *NotificationServiceExtended) DeleteNotification(ctx context.Context, notificationID uuid.UUID) error {
	return ns.notificationRepo.DeleteNotification(ctx, notificationID)
}

func (ns *NotificationServiceExtended) GetUserPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
	return ns.preferenceRepo.GetPreferences(ctx, userID)
}

func (ns *NotificationServiceExtended) UpdateUserPreferences(ctx context.Context, preferences *models.NotificationPreferences) error {
	return ns.preferenceRepo.UpdatePreferences(ctx, preferences)
}

func (ns *NotificationServiceExtended) GetUserSubscriptions(ctx context.Context, userID uuid.UUID) ([]*models.Subscription, error) {
	return ns.subscriptionRepo.FindByUser(ctx, userID)
}

func (ns *NotificationServiceExtended) CreateSubscription(ctx context.Context, subscription *models.Subscription) error {
	return ns.subscriptionRepo.CreateSubscription(ctx, subscription)
}

func (ns *NotificationServiceExtended) UpdateSubscription(ctx context.Context, subscription *models.Subscription) error {
	return ns.subscriptionRepo.UpdateSubscription(ctx, subscription)
}

func (ns *NotificationServiceExtended) DeleteSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	return ns.subscriptionRepo.DeleteSubscription(ctx, subscriptionID)
}

func main() {
	// Initialize database connection
	dbURL := getEnv("DATABASE_URL", "postgres://ao3_user:ao3_password@localhost/ao3_nuclear?sslmode=disable")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Initialize messaging service
	messagingService := messaging.NewUniversalMessageService(
		nil, // telemetry
		&messaging.SimpleMessageValidator{},
		messaging.NewSimpleRateLimiter(),
		nil, // messageRepo - can be nil for basic functionality
		nil, // attemptRepo - can be nil for basic functionality
		nil, // preferenceService - can be nil for basic functionality
	)

	// Initialize repositories
	subscriptionRepo := NewSubscriptionRepository(db)
	notificationRepo := NewNotificationRepository(db)
	digestRepo := NewDigestRepository(db)
	preferenceRepo := NewPreferenceRepository(db)

	// Initialize notification service
	coreNotificationSvc := notifications.NewNotificationService(
		messagingService,
		subscriptionRepo,
		notificationRepo,
		digestRepo,
		preferenceRepo,
		notifications.NotificationServiceConfig{
			EnableBatching:       getEnvBool("ENABLE_BATCHING", true),
			BatchIntervalMinutes: getEnvInt("BATCH_INTERVAL_MINUTES", 60),
			MaxBatchSize:         getEnvInt("MAX_BATCH_SIZE", 50),
			EnableSmartFiltering: getEnvBool("ENABLE_SMART_FILTERING", true),
		},
	)

	// Create extended notification service with repository access
	extendedNotificationSvc := &NotificationServiceExtended{
		NotificationService: coreNotificationSvc,
		subscriptionRepo:    subscriptionRepo,
		notificationRepo:    notificationRepo,
		preferenceRepo:      preferenceRepo,
	}

	// Initialize WebSocket upgrader
	wsUpgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Configure CORS for WebSocket connections
			origin := r.Header.Get("Origin")
			allowedOrigins := []string{
				"http://localhost:3000",
				"https://localhost:3000",
				"http://localhost:3001",
				"http://127.0.0.1:3001",
				"https://127.0.0.1:3001",
				getEnv("FRONTEND_URL", "http://localhost:3000"),
			}
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		},
	}

	// Initialize service
	service := &NotificationService{
		db:               db,
		notificationSvc:  extendedNotificationSvc,
		messagingService: messagingService,
		wsUpgrader:       wsUpgrader,
		wsClients:        make(map[string]*websocket.Conn),
		wsBroadcast:      make(chan []byte),
	}

	// Setup HTTP server
	router := gin.Default()

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://localhost:3000", "http://localhost:3001", "http://127.0.0.1:3001", "https://127.0.0.1:3001", getEnv("FRONTEND_URL", "http://localhost:3000")},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Temporary simple auth middleware - accepts any Bearer token with valid X-User-ID
	authMiddleware := func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		userIDHeader := c.GetHeader("X-User-ID")

		if authHeader == "" || userIDHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header and X-User-ID required"})
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		c.Set("user_id", userIDHeader)
		c.Next()
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "notification-service"})
	})

	// WebSocket endpoint for real-time notifications - use query param auth
	router.GET("/ws", func(c *gin.Context) {
		token := c.Query("token")
		userID := c.Query("user_id")

		if token == "" || userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token and user_id query parameters required"})
			return
		}

		// Basic token validation - just check that token is not empty
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token cannot be empty"})
			return
		}

		c.Set("user_id", userID)
		service.handleWebSocket(c)
	})

	// API routes
	api := router.Group("/api/v1")
	api.Use(authMiddleware)
	{
		// Notifications
		api.GET("/notifications", service.getUserNotifications)
		api.PUT("/notifications/:id/read", service.markNotificationRead)
		api.DELETE("/notifications/:id", service.deleteNotification)
		api.GET("/notifications/unread-count", service.getUnreadCount)

		// Preferences
		api.GET("/preferences", service.getNotificationPreferences)
		api.PUT("/preferences", service.updateNotificationPreferences)

		// Subscriptions
		api.GET("/subscriptions", service.getUserSubscriptions)
		api.POST("/subscriptions", service.createSubscription)
		api.PUT("/subscriptions/:id", service.updateSubscription)
		api.DELETE("/subscriptions/:id", service.deleteSubscription)

		// Rules
		api.GET("/rules", service.getNotificationRules)
		api.POST("/rules", service.createNotificationRule)
		api.PUT("/rules/:id", service.updateNotificationRule)
		api.DELETE("/rules/:id", service.deleteNotificationRule)

		// Admin/testing endpoints
		api.POST("/test-notification", service.createTestNotification)
		api.POST("/process-event", service.processEvent)
	}

	// Start WebSocket broadcaster
	go service.handleWebSocketBroadcast()

	// Start HTTP server
	port := getEnv("PORT", "8004")
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Notification service started on port %s", port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down notification service...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Notification service shutdown complete")
}

// Environment helpers
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
