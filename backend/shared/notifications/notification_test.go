package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"nuclear-ao3/shared/messaging"
	"nuclear-ao3/shared/models"
)

// Mock repositories for testing
type mockSubscriptionRepo struct{}

func (m *mockSubscriptionRepo) CreateSubscription(ctx context.Context, subscription *models.Subscription) error {
	return nil
}

func (m *mockSubscriptionRepo) GetSubscription(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	return nil, nil
}

func (m *mockSubscriptionRepo) UpdateSubscription(ctx context.Context, subscription *models.Subscription) error {
	return nil
}

func (m *mockSubscriptionRepo) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockSubscriptionRepo) FindByUser(ctx context.Context, userID uuid.UUID) ([]*models.Subscription, error) {
	return []*models.Subscription{}, nil
}

func (m *mockSubscriptionRepo) FindByTarget(ctx context.Context, targetType models.SubscriptionType, targetID uuid.UUID) ([]*models.Subscription, error) {
	// Return a mock subscription for testing
	if targetType == models.SubscriptionWork {
		return []*models.Subscription{
			{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Type:     models.SubscriptionWork,
				TargetID: targetID,
				Events:   []models.NotificationEvent{models.EventWorkUpdated},
				IsActive: true,
			},
		}, nil
	}
	return []*models.Subscription{}, nil
}

func (m *mockSubscriptionRepo) FindByUserAndTarget(ctx context.Context, userID, targetID uuid.UUID, targetType models.SubscriptionType) (*models.Subscription, error) {
	return nil, nil
}

type mockNotificationRepo struct{}

func (m *mockNotificationRepo) CreateNotification(ctx context.Context, notification *models.NotificationItem) error {
	return nil
}

func (m *mockNotificationRepo) GetNotification(ctx context.Context, id uuid.UUID) (*models.NotificationItem, error) {
	return nil, nil
}

func (m *mockNotificationRepo) UpdateNotification(ctx context.Context, notification *models.NotificationItem) error {
	return nil
}

func (m *mockNotificationRepo) DeleteNotification(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockNotificationRepo) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.NotificationItem, error) {
	return []*models.NotificationItem{}, nil
}

func (m *mockNotificationRepo) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockNotificationRepo) GetNotificationsForBatch(ctx context.Context, userID uuid.UUID, frequency models.NotificationFrequency) ([]*models.NotificationItem, error) {
	return []*models.NotificationItem{}, nil
}

type mockDigestRepo struct{}

func (m *mockDigestRepo) CreateDigest(ctx context.Context, digest *models.NotificationDigest) error {
	return nil
}

func (m *mockDigestRepo) GetDigest(ctx context.Context, id uuid.UUID) (*models.NotificationDigest, error) {
	return nil, nil
}

func (m *mockDigestRepo) UpdateDigest(ctx context.Context, digest *models.NotificationDigest) error {
	return nil
}

func (m *mockDigestRepo) GetPendingDigests(ctx context.Context, digestType string) ([]*models.NotificationDigest, error) {
	return []*models.NotificationDigest{}, nil
}

type mockPreferenceRepo struct{}

func (m *mockPreferenceRepo) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
	prefs := models.DefaultNotificationPreferences(userID)
	return &prefs, nil
}

func (m *mockPreferenceRepo) UpdatePreferences(ctx context.Context, preferences *models.NotificationPreferences) error {
	return nil
}

func (m *mockPreferenceRepo) CreatePreferences(ctx context.Context, preferences *models.NotificationPreferences) error {
	return nil
}

// Mock message service
type mockMessageService struct{}

func (m *mockMessageService) SendMessage(ctx context.Context, message *models.Message) error {
	// Just return success for testing
	return nil
}

func (m *mockMessageService) ScheduleMessage(ctx context.Context, msg *models.Message, deliverAt time.Time) error {
	return nil
}

func (m *mockMessageService) GetMessageStatus(ctx context.Context, messageID string) (*messaging.MessageStatus, error) {
	return nil, nil
}

func (m *mockMessageService) RetryFailedDeliveries(ctx context.Context, messageID string) error {
	return nil
}

func (m *mockMessageService) GetMetrics(ctx context.Context, start, end time.Time) (*models.MessageMetrics, error) {
	return nil, nil
}

func (m *mockMessageService) RegisterChannelProvider(provider messaging.ChannelProvider) error {
	return nil
}

func (m *mockMessageService) GetAvailableChannels(ctx context.Context) []models.DeliveryChannel {
	return []models.DeliveryChannel{models.ChannelEmail, models.ChannelInApp}
}

func TestNotificationServiceCreation(t *testing.T) {
	// Create mock message service
	messageService := &mockMessageService{}

	// Create mock repositories
	subscriptionRepo := &mockSubscriptionRepo{}
	notificationRepo := &mockNotificationRepo{}
	digestRepo := &mockDigestRepo{}
	preferenceRepo := &mockPreferenceRepo{}

	// Create service config
	config := NotificationServiceConfig{
		EnableBatching:       true,
		BatchIntervalMinutes: 30,
		MaxBatchSize:         50,
		EnableSmartFiltering: true,
		DefaultQuietHours:    []string{"22:00", "08:00"},
	}

	// Create notification service
	service := NewNotificationService(
		messageService,
		subscriptionRepo,
		notificationRepo,
		digestRepo,
		preferenceRepo,
		config,
	)

	if service == nil {
		t.Fatal("Failed to create notification service")
	}

	if service.smartFilter == nil {
		t.Error("Smart filter not initialized")
	}

	if service.ruleEngine == nil {
		t.Error("Rule engine not initialized")
	}

	if service.batchProcessor == nil {
		t.Error("Batch processor not initialized when batching enabled")
	}
}

func TestEventProcessing(t *testing.T) {
	// Create the service same as above
	messageService := &mockMessageService{}
	subscriptionRepo := &mockSubscriptionRepo{}
	notificationRepo := &mockNotificationRepo{}
	digestRepo := &mockDigestRepo{}
	preferenceRepo := &mockPreferenceRepo{}

	config := NotificationServiceConfig{
		EnableBatching:       false, // Disable batching for immediate testing
		EnableSmartFiltering: true,
	}

	service := NewNotificationService(
		messageService,
		subscriptionRepo,
		notificationRepo,
		digestRepo,
		preferenceRepo,
		config,
	)

	// Create a test event
	workID := uuid.New()
	authorID := uuid.New()

	event := &EventData{
		Type:        models.EventWorkUpdated,
		SourceID:    workID,
		SourceType:  "work",
		Title:       "Test Work Updated",
		Description: "Chapter 5 has been published",
		ActionURL:   "https://example.com/works/123",
		ActorID:     &authorID,
		ActorName:   "Test Author",
		AuthorIDs:   []uuid.UUID{authorID},
		Tags:        []string{"Romance", "Fluff"},
		Rating:      "Teen And Up Audiences",
		WordCount:   5000,
		IsCompleted: false,
	}

	// Process the event
	ctx := context.Background()
	err := service.ProcessEvent(ctx, event)

	if err != nil {
		t.Errorf("Failed to process event: %v", err)
	}

	// If we get here without error, the basic flow works
	t.Log("Event processing completed successfully")
}

func TestSmartFilterCreation(t *testing.T) {
	filter := NewSmartFilter()
	if filter == nil {
		t.Fatal("Failed to create smart filter")
	}

	// Test basic functionality doesn't panic
	userID := uuid.New()
	prefs := models.DefaultNotificationPreferences(userID)

	notification := &models.NotificationItem{
		ID:        uuid.New(),
		UserID:    userID,
		Event:     models.EventWorkUpdated,
		Priority:  models.PriorityMedium,
		Title:     "Test notification",
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	shouldNotify, enhanced := filter.ShouldNotify(ctx, &prefs, notification)

	// Should return true for basic case
	if !shouldNotify {
		t.Error("Smart filter unexpectedly blocked notification")
	}

	if enhanced == nil {
		t.Error("Smart filter should return enhanced notification")
	}
}

func TestRuleEngineCreation(t *testing.T) {
	engine := NewRuleEngine()
	if engine == nil {
		t.Fatal("Failed to create rule engine")
	}

	// Test basic rule evaluation doesn't panic
	userID := uuid.New()
	prefs := models.DefaultNotificationPreferences(userID)

	notification := &models.NotificationItem{
		ID:        uuid.New(),
		UserID:    userID,
		Event:     models.EventWorkUpdated,
		Priority:  models.PriorityMedium,
		Title:     "Test notification",
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	action := engine.EvaluateNotification(ctx, &prefs, notification)

	// Should return allow action for basic case
	if action.Action != models.ActionAllow {
		t.Errorf("Expected ActionAllow, got %v", action.Action)
	}
}
