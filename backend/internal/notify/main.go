package notify

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"strconv"
	"strings"

	"nuclear-ao3/internal/platform"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"

	"nuclear-ao3/shared/email"
	"nuclear-ao3/shared/models"
	"nuclear-ao3/shared/notifications"
)

type NotificationService struct {
	db              *sql.DB
	notificationSvc *NotificationServiceExtended
	wsUpgrader      websocket.Upgrader
	wsClients       map[string]*websocket.Conn // userID -> connection
	wsBroadcast     chan []byte
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

func (ns *NotificationServiceExtended) DeleteNotification(ctx context.Context, notificationID, userID uuid.UUID) error {
	return ns.notificationRepo.DeleteNotification(ctx, notificationID, userID)
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

// New builds the notification service on the shared database. Notification
// emails are sent through mail; nil turns them off.
func New(db *sql.DB, mail *email.Config) *NotificationService {

	subscriptionRepo := NewSubscriptionRepository(db)
	notificationRepo := NewNotificationRepository(db)
	digestRepo := NewDigestRepository(db)
	preferenceRepo := NewPreferenceRepository(db)

	coreNotificationSvc := notifications.NewNotificationService(
		&mailSender{db: db, cfg: mail},
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

	allowedOrigins := map[string]bool{}
	for _, o := range strings.Split(platform.Env("CORS_ORIGINS", "http://localhost:3000,http://localhost:3001"), ",") {
		allowedOrigins[strings.TrimSpace(o)] = true
	}

	coreNotificationSvc.SetBlockChecker(func(ctx context.Context, recipient, actor uuid.UUID) bool {
		var blocked bool
		err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM user_blocks
			WHERE blocker_id = $1 AND blocked_id = $2 AND block_type IN ('full', 'comments'))`,
			recipient, actor).Scan(&blocked)
		return err != nil || blocked // if we can't tell, don't notify
	})

	return &NotificationService{
		db: db,
		notificationSvc: &NotificationServiceExtended{
			NotificationService: coreNotificationSvc,
			subscriptionRepo:    subscriptionRepo,
			notificationRepo:    notificationRepo,
			preferenceRepo:      preferenceRepo,
		},
		wsUpgrader: websocket.Upgrader{
			// Same-origin connections have an Origin matching the Host.
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || allowedOrigins[origin] ||
					origin == "https://"+r.Host || origin == "http://"+r.Host
			},
		},
		wsClients:   make(map[string]*websocket.Conn),
		wsBroadcast: make(chan []byte),
	}
}

// Mount registers the notification routes.
//
// Subscriptions are served by the works package; the copies that used to
// live here are gone.
func Mount(r *gin.Engine, service *NotificationService, d *platform.Deps) {
	// Browsers can't send headers on a WebSocket, so the token comes in ?token=.
	r.GET("/ws", d.RequireUserFromQuery, service.handleWebSocket)

	api := r.Group("/api/v1")
	api.Use(d.RequireUser)
	{
		api.GET("/notifications", service.getUserNotifications)
		api.PUT("/notifications/:id/read", service.markNotificationRead)
		api.DELETE("/notifications/:id", service.deleteNotification)
		api.GET("/notifications/unread-count", service.getUnreadCount)

		api.GET("/notification-preferences", service.getNotificationPreferences)
		api.PUT("/notification-preferences", service.updateNotificationPreferences)

		api.GET("/notification-rules", service.getNotificationRules)
		api.POST("/notification-rules", service.createNotificationRule)
		api.PUT("/notification-rules/:id", service.updateNotificationRule)
		api.DELETE("/notification-rules/:id", service.deleteNotificationRule)
	}

	admin := r.Group("/api/v1/admin/notifications", d.RequireUser, d.RequireRole("admin"))
	{
		admin.POST("/test", service.createTestNotification)
		admin.POST("/process-event", service.processEvent)
	}
}

// Start runs the live-update broadcaster in the background.
func (s *NotificationService) Start() {
	go s.handleWebSocketBroadcast()
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

// Notifier returns the notification engine other packages send events to.
func (s *NotificationService) Notifier() *notifications.NotificationService {
	return s.notificationSvc.NotificationService
}
