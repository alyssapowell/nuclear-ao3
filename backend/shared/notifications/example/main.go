package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"nuclear-ao3/shared/messaging"
	"nuclear-ao3/shared/models"
	"nuclear-ao3/shared/notifications"
)

// This example demonstrates how to integrate and use the Nuclear AO3 notification system
// The notification system provides:
// - Smart notification filtering with quiet hours and rate limiting
// - User-defined rules for custom notification handling
// - Intelligent batching with beautiful HTML digest emails
// - Multi-channel delivery with user preferences
// - Comprehensive subscription system for works, authors, series, tags

func main() {
	fmt.Println("Nuclear AO3 Advanced Notification System - Usage Example")
	fmt.Println(strings.Repeat("=", 60))

	// Step 1: Set up the notification system
	notificationService := setupNotificationSystem()

	// Step 2: Create some sample subscriptions
	userID := createSampleSubscriptions(notificationService)

	// Step 3: Process various notification events
	demonstrateEventProcessing(notificationService, userID)

	// Step 4: Show digest functionality
	demonstrateDigestSystem(notificationService, userID)

	fmt.Println("\nNotification system demonstration completed!")
}

func setupNotificationSystem() *notifications.NotificationService {
	fmt.Println("\n1. Setting up notification system components...")

	// In a real implementation, you would connect to actual databases
	// For this example, we'll use in-memory implementations

	// Create message service (you would initialize this properly with real providers)
	messageService := createMessageService()

	// Create repositories (you would implement these with your database)
	subscriptionRepo := &InMemorySubscriptionRepo{
		subscriptions: make(map[uuid.UUID]*models.Subscription),
	}
	notificationRepo := &InMemoryNotificationRepo{
		notifications: make(map[uuid.UUID]*models.NotificationItem),
	}
	digestRepo := &InMemoryDigestRepo{
		digests: make(map[uuid.UUID]*models.NotificationDigest),
	}
	preferenceRepo := &InMemoryPreferenceRepo{
		preferences: make(map[uuid.UUID]*models.NotificationPreferences),
	}

	// Configure the notification service
	config := notifications.NotificationServiceConfig{
		EnableBatching:       true,
		BatchIntervalMinutes: 30, // Batch notifications every 30 minutes
		MaxBatchSize:         50, // Max 50 notifications per batch
		EnableSmartFiltering: true,
		DefaultQuietHours:    []string{"22:00", "08:00"}, // 10 PM to 8 AM
	}

	// Create the notification service
	service := notifications.NewNotificationService(
		messageService,
		subscriptionRepo,
		notificationRepo,
		digestRepo,
		preferenceRepo,
		config,
	)

	fmt.Println("✓ Notification service initialized")
	return service
}

func createSampleSubscriptions(service *notifications.NotificationService) uuid.UUID {
	fmt.Println("\n2. Creating sample user subscriptions...")

	userID := uuid.New()
	workID := uuid.New()
	authorID := uuid.New()
	seriesID := uuid.New()

	// Create user preferences (this would typically be done when user signs up)
	prefs := models.DefaultNotificationPreferences(userID)
	// Customize preferences for demo
	prefs.EnableBatching = true
	prefs.BatchFrequency = models.FrequencyDaily
	prefs.QuietHoursStart = parseTime("22:00")
	prefs.QuietHoursEnd = parseTime("08:00")
	prefs.MaxNotificationsPerHour = 5

	// Subscribe to a specific work
	workSubscription := &models.Subscription{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       models.SubscriptionWork,
		TargetID:   workID,
		TargetName: "The Adventures of Nuclear AO3",
		Events:     []models.NotificationEvent{models.EventWorkUpdated, models.EventWorkCompleted},
		Frequency:  models.FrequencyImmediate,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Subscribe to an author
	authorSubscription := &models.Subscription{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       models.SubscriptionAuthor,
		TargetID:   authorID,
		TargetName: "FavoriteAuthor123",
		Events:     []models.NotificationEvent{models.EventNewWork, models.EventWorkUpdated},
		Frequency:  models.FrequencyDaily, // Batch daily for author updates
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		// Content filters
		FilterRating: []string{"General Audiences", "Teen And Up Audiences"},
		MinWordCount: intPtr(1000), // Only notify for works >= 1000 words
	}

	// Subscribe to a series
	seriesSubscription := &models.Subscription{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       models.SubscriptionSeries,
		TargetID:   seriesID,
		TargetName: "Epic Fantasy Series",
		Events:     []models.NotificationEvent{models.EventSeriesUpdated},
		Frequency:  models.FrequencyImmediate,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	fmt.Printf("✓ Created subscriptions for user %s\n", userID)
	fmt.Printf("  - Work subscription: %s\n", workSubscription.TargetName)
	fmt.Printf("  - Author subscription: %s\n", authorSubscription.TargetName)
	fmt.Printf("  - Series subscription: %s\n", seriesSubscription.TargetName)

	return userID
}

func demonstrateEventProcessing(service *notifications.NotificationService, userID uuid.UUID) {
	fmt.Println("\n3. Processing notification events...")

	ctx := context.Background()

	// Simulate a work update event
	workUpdateEvent := &notifications.EventData{
		Type:        models.EventWorkUpdated,
		SourceID:    uuid.New(),
		SourceType:  "work",
		Title:       "Chapter 15: The Final Battle",
		Description: "The epic conclusion to our story has arrived! Thank you for following along.",
		ActionURL:   "https://nuclear-ao3.org/works/123/chapters/15",
		ActorID:     &userID,
		ActorName:   "AuthorName",
		AuthorIDs:   []uuid.UUID{userID},
		Tags:        []string{"Fantasy", "Adventure", "Completed"},
		Rating:      "Teen And Up Audiences",
		WordCount:   8500,
		IsCompleted: true,
	}

	// Process the event
	err := service.ProcessEvent(ctx, workUpdateEvent)
	if err != nil {
		log.Printf("Error processing work update: %v", err)
	} else {
		fmt.Printf("✓ Processed work update event: %s\n", workUpdateEvent.Title)
	}

	// Simulate a comment notification
	commentEvent := &notifications.EventData{
		Type:        models.EventCommentReceived,
		SourceID:    uuid.New(),
		SourceType:  "comment",
		Title:       "New comment on your work",
		Description: "Reader123 left a comment: 'This was absolutely amazing! Can't wait for the sequel!'",
		ActionURL:   "https://nuclear-ao3.org/works/123#comment_456",
		ActorID:     &userID,
		ActorName:   "Reader123",
	}

	err = service.ProcessEvent(ctx, commentEvent)
	if err != nil {
		log.Printf("Error processing comment: %v", err)
	} else {
		fmt.Printf("✓ Processed comment notification: %s\n", commentEvent.Title)
	}

	// Simulate kudos notification
	kudosEvent := &notifications.EventData{
		Type:        models.EventKudosReceived,
		SourceID:    uuid.New(),
		SourceType:  "kudos",
		Title:       "Your work received kudos",
		Description: "5 new readers left kudos on 'The Adventures of Nuclear AO3'",
		ActionURL:   "https://nuclear-ao3.org/works/123",
		ActorName:   "Multiple readers",
	}

	err = service.ProcessEvent(ctx, kudosEvent)
	if err != nil {
		log.Printf("Error processing kudos: %v", err)
	} else {
		fmt.Printf("✓ Processed kudos notification: %s\n", kudosEvent.Title)
	}
}

func demonstrateDigestSystem(service *notifications.NotificationService, userID uuid.UUID) {
	fmt.Println("\n4. Demonstrating digest/batching system...")

	ctx := context.Background()

	// The batch processor would normally run on a timer, but for demonstration
	// we can show how multiple notifications would be batched together

	// Get user notifications (in a real app, this would show recent notifications)
	notifications, err := service.GetUserNotifications(ctx, userID, 10, 0)
	if err != nil {
		log.Printf("Error getting notifications: %v", err)
		return
	}

	fmt.Printf("✓ User has %d recent notifications\n", len(notifications))

	// The batch processor automatically groups notifications by type and sends
	// beautiful HTML digest emails. Here's what the system provides:

	fmt.Println("\nNotification System Features:")
	fmt.Println("• Smart filtering with quiet hours and rate limiting")
	fmt.Println("• User-defined rules for custom notification handling")
	fmt.Println("• Intelligent batching with HTML digest emails")
	fmt.Println("• Multi-channel delivery (email, in-app, push)")
	fmt.Println("• Comprehensive subscription system")
	fmt.Println("• Content filtering (rating, tags, word count)")
	fmt.Println("• Integration with file-based email templates")
}

// Helper functions and mock implementations

func parseTime(timeStr string) *time.Time {
	// Simple time parsing for HH:MM format
	hour := int(timeStr[0]-'0')*10 + int(timeStr[1]-'0')
	min := int(timeStr[3]-'0')*10 + int(timeStr[4]-'0')

	now := time.Now()
	t := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, now.Location())
	return &t
}

func intPtr(i int) *int {
	return &i
}

func createMessageService() messaging.MessageService {
	// In a real implementation, you would create the actual UniversalMessageService
	// with proper email providers, database connections, etc.
	// For this example, we'll use a simple mock
	return &MockMessageService{}
}

// Mock implementations for the example (in a real app, these would be database-backed)

type MockMessageService struct{}

func (m *MockMessageService) SendMessage(ctx context.Context, msg *models.Message) error {
	fmt.Printf("📧 Mock: Sending message '%s' to %d recipients\n", msg.Content.Subject, len(msg.Recipients))
	return nil
}

func (m *MockMessageService) ScheduleMessage(ctx context.Context, msg *models.Message, deliverAt time.Time) error {
	return nil
}

func (m *MockMessageService) GetMessageStatus(ctx context.Context, messageID string) (*messaging.MessageStatus, error) {
	return nil, nil
}

func (m *MockMessageService) RetryFailedDeliveries(ctx context.Context, messageID string) error {
	return nil
}

func (m *MockMessageService) GetMetrics(ctx context.Context, start, end time.Time) (*models.MessageMetrics, error) {
	return nil, nil
}

func (m *MockMessageService) RegisterChannelProvider(provider messaging.ChannelProvider) error {
	return nil
}

func (m *MockMessageService) GetAvailableChannels(ctx context.Context) []models.DeliveryChannel {
	return []models.DeliveryChannel{models.ChannelEmail, models.ChannelInApp}
}

// In-memory repository implementations (replace with your database implementations)

type InMemorySubscriptionRepo struct {
	subscriptions map[uuid.UUID]*models.Subscription
}

func (r *InMemorySubscriptionRepo) CreateSubscription(ctx context.Context, subscription *models.Subscription) error {
	r.subscriptions[subscription.ID] = subscription
	return nil
}

func (r *InMemorySubscriptionRepo) GetSubscription(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	sub, exists := r.subscriptions[id]
	if !exists {
		return nil, fmt.Errorf("subscription not found")
	}
	return sub, nil
}

func (r *InMemorySubscriptionRepo) UpdateSubscription(ctx context.Context, subscription *models.Subscription) error {
	r.subscriptions[subscription.ID] = subscription
	return nil
}

func (r *InMemorySubscriptionRepo) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	delete(r.subscriptions, id)
	return nil
}

func (r *InMemorySubscriptionRepo) FindByUser(ctx context.Context, userID uuid.UUID) ([]*models.Subscription, error) {
	var result []*models.Subscription
	for _, sub := range r.subscriptions {
		if sub.UserID == userID {
			result = append(result, sub)
		}
	}
	return result, nil
}

func (r *InMemorySubscriptionRepo) FindByTarget(ctx context.Context, targetType models.SubscriptionType, targetID uuid.UUID) ([]*models.Subscription, error) {
	var result []*models.Subscription
	for _, sub := range r.subscriptions {
		if sub.Type == targetType && sub.TargetID == targetID {
			result = append(result, sub)
		}
	}
	return result, nil
}

func (r *InMemorySubscriptionRepo) FindByUserAndTarget(ctx context.Context, userID, targetID uuid.UUID, targetType models.SubscriptionType) (*models.Subscription, error) {
	for _, sub := range r.subscriptions {
		if sub.UserID == userID && sub.TargetID == targetID && sub.Type == targetType {
			return sub, nil
		}
	}
	return nil, fmt.Errorf("subscription not found")
}

type InMemoryNotificationRepo struct {
	notifications map[uuid.UUID]*models.NotificationItem
}

func (r *InMemoryNotificationRepo) CreateNotification(ctx context.Context, notification *models.NotificationItem) error {
	r.notifications[notification.ID] = notification
	return nil
}

func (r *InMemoryNotificationRepo) GetNotification(ctx context.Context, id uuid.UUID) (*models.NotificationItem, error) {
	notif, exists := r.notifications[id]
	if !exists {
		return nil, fmt.Errorf("notification not found")
	}
	return notif, nil
}

func (r *InMemoryNotificationRepo) UpdateNotification(ctx context.Context, notification *models.NotificationItem) error {
	r.notifications[notification.ID] = notification
	return nil
}

func (r *InMemoryNotificationRepo) DeleteNotification(ctx context.Context, id uuid.UUID) error {
	delete(r.notifications, id)
	return nil
}

func (r *InMemoryNotificationRepo) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.NotificationItem, error) {
	var result []*models.NotificationItem
	for _, notif := range r.notifications {
		if notif.UserID == userID {
			result = append(result, notif)
		}
	}
	return result, nil
}

func (r *InMemoryNotificationRepo) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	count := 0
	for _, notif := range r.notifications {
		if notif.UserID == userID && !notif.IsRead {
			count++
		}
	}
	return count, nil
}

func (r *InMemoryNotificationRepo) GetNotificationsForBatch(ctx context.Context, userID uuid.UUID, frequency models.NotificationFrequency) ([]*models.NotificationItem, error) {
	var result []*models.NotificationItem
	for _, notif := range r.notifications {
		if notif.UserID == userID && !notif.IsDelivered {
			result = append(result, notif)
		}
	}
	return result, nil
}

type InMemoryDigestRepo struct {
	digests map[uuid.UUID]*models.NotificationDigest
}

func (r *InMemoryDigestRepo) CreateDigest(ctx context.Context, digest *models.NotificationDigest) error {
	r.digests[digest.ID] = digest
	return nil
}

func (r *InMemoryDigestRepo) GetDigest(ctx context.Context, id uuid.UUID) (*models.NotificationDigest, error) {
	digest, exists := r.digests[id]
	if !exists {
		return nil, fmt.Errorf("digest not found")
	}
	return digest, nil
}

func (r *InMemoryDigestRepo) UpdateDigest(ctx context.Context, digest *models.NotificationDigest) error {
	r.digests[digest.ID] = digest
	return nil
}

func (r *InMemoryDigestRepo) GetPendingDigests(ctx context.Context, digestType string) ([]*models.NotificationDigest, error) {
	var result []*models.NotificationDigest
	for _, digest := range r.digests {
		if digest.Status == models.DigestPending && digest.DigestType == digestType {
			result = append(result, digest)
		}
	}
	return result, nil
}

type InMemoryPreferenceRepo struct {
	preferences map[uuid.UUID]*models.NotificationPreferences
}

func (r *InMemoryPreferenceRepo) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
	prefs, exists := r.preferences[userID]
	if !exists {
		// Return default preferences
		defaultPrefs := models.DefaultNotificationPreferences(userID)
		return &defaultPrefs, nil
	}
	return prefs, nil
}

func (r *InMemoryPreferenceRepo) UpdatePreferences(ctx context.Context, preferences *models.NotificationPreferences) error {
	r.preferences[preferences.UserID] = preferences
	return nil
}

func (r *InMemoryPreferenceRepo) CreatePreferences(ctx context.Context, preferences *models.NotificationPreferences) error {
	r.preferences[preferences.UserID] = preferences
	return nil
}
