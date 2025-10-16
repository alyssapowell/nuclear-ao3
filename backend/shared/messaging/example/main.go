package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"nuclear-ao3/shared/messaging"
	"nuclear-ao3/shared/messaging/email"
	"nuclear-ao3/shared/messaging/errors"
	"nuclear-ao3/shared/messaging/telemetry"
	"nuclear-ao3/shared/messaging/templates"
	"nuclear-ao3/shared/models"
)

// Example showing how to set up and use the Universal Messaging Service
func main() {
	log.Println("Setting up Universal Messaging Service...")

	// Initialize telemetry collector
	telemetryCollector := telemetry.NewInMemoryTelemetryCollector()

	// Initialize file-based template renderer with hot reload enabled for development
	templatesDir := "../templates/files"
	templateRenderer, err := templates.NewFileBasedTemplateRenderer(templatesDir, true)
	if err != nil {
		log.Fatalf("Failed to initialize template renderer: %v", err)
	}

	// Initialize error classifier
	errorClassifier := errors.NewSMTPErrorClassifier()

	// Create email channel provider with AO3-compatible config
	emailConfig := email.AO3CompatibleSMTPConfig()
	emailConfig.FromEmail = "noreply@nuclear-ao3.local"
	emailConfig.FromName = "Nuclear AO3 Demo"

	emailProvider := email.NewEmailChannelProvider(
		emailConfig,
		telemetryCollector,
		templateRenderer,
		errorClassifier,
	)

	// Initialize message service
	messageService := messaging.NewUniversalMessageService(
		telemetryCollector,
		&messaging.SimpleMessageValidator{},
		messaging.NewSimpleRateLimiter(),
		&InMemoryMessageRepo{},
		&InMemoryAttemptRepo{},
		&InMemoryPreferenceService{},
	)

	// Register email channel
	if err := messageService.RegisterChannelProvider(emailProvider); err != nil {
		log.Fatalf("Failed to register email provider: %v", err)
	}

	// Create a test user
	userID := uuid.New()
	userEmail := "test@example.com"

	// Create a subscription update message
	message := &models.Message{
		Type: models.MessageSubscriptionUpdate,
		Content: models.MessageContent{
			Subject:   "New Chapter Available!",
			PlainText: "A new chapter has been posted for 'My Favorite Fic'.",
			HTML:      "<p>A new chapter has been posted for <strong>My Favorite Fic</strong>.</p>",
			Variables: map[string]interface{}{
				"work_title":  "My Favorite Fic",
				"author_name": "TestAuthor",
				"action_url":  "https://nuclear-ao3.local/works/123",
			},
		},
		Recipients: []models.Recipient{
			{
				UserID:      userID,
				Preferences: models.DefaultUserNotificationSettings(userID, userEmail),
			},
		},
	}

	// Send the message
	log.Println("Sending subscription update message...")
	ctx := context.Background()
	if err := messageService.SendMessage(ctx, message); err != nil {
		log.Printf("Failed to send message: %v", err)
	} else {
		log.Printf("Message sent successfully! Message ID: %s", message.ID)
	}

	// Check message status
	time.Sleep(100 * time.Millisecond) // Give it a moment to process
	status, err := messageService.GetMessageStatus(ctx, message.ID.String())
	if err != nil {
		log.Printf("Failed to get message status: %v", err)
	} else {
		log.Printf("Message status: %s", status.Message.Status)
		log.Printf("Delivery attempts: %d", len(status.DeliveryAttempts))
		for _, attempt := range status.DeliveryAttempts {
			log.Printf("  - Channel %s: %s", attempt.Channel, attempt.Status)
		}
	}

	// Get available channels
	channels := messageService.GetAvailableChannels(ctx)
	log.Printf("Available channels: %v", channels)

	// Get metrics
	metrics, err := messageService.GetMetrics(ctx, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		log.Printf("Failed to get metrics: %v", err)
	} else {
		log.Printf("Total sent: %d, delivered: %d, failed: %d",
			metrics.TotalSent, metrics.TotalDelivered, metrics.TotalFailed)
	}

	// Example of a comment notification
	log.Println("\nSending comment notification...")
	commentMessage := &models.Message{
		Type: models.MessageCommentNotify,
		Content: models.MessageContent{
			Subject:   "New comment on your work",
			PlainText: "Someone left a comment on your work!",
			Variables: map[string]interface{}{
				"work_title":     "My Favorite Fic",
				"commenter_name": "Reader123",
				"action_url":     "https://nuclear-ao3.local/works/123#comments",
			},
		},
		Recipients: []models.Recipient{
			{
				UserID:      userID,
				Preferences: models.DefaultUserNotificationSettings(userID, userEmail),
			},
		},
	}

	if err := messageService.SendMessage(ctx, commentMessage); err != nil {
		log.Printf("Failed to send comment notification: %v", err)
	} else {
		log.Printf("Comment notification sent! Message ID: %s", commentMessage.ID)
	}

	log.Println("\nUniversal Messaging Service demo completed!")
}

// Simple in-memory implementations for demo purposes
// In production, these would be proper database-backed implementations

type InMemoryMessageRepo struct {
	messages map[string]*models.Message
}

func (r *InMemoryMessageRepo) CreateMessage(ctx context.Context, msg *models.Message) error {
	if r.messages == nil {
		r.messages = make(map[string]*models.Message)
	}
	r.messages[msg.ID.String()] = msg
	return nil
}

func (r *InMemoryMessageRepo) GetMessage(ctx context.Context, messageID string) (*models.Message, error) {
	if r.messages == nil {
		return nil, fmt.Errorf("message not found")
	}
	msg, exists := r.messages[messageID]
	if !exists {
		return nil, fmt.Errorf("message not found")
	}
	return msg, nil
}

func (r *InMemoryMessageRepo) UpdateMessage(ctx context.Context, msg *models.Message) error {
	if r.messages == nil {
		r.messages = make(map[string]*models.Message)
	}
	r.messages[msg.ID.String()] = msg
	return nil
}

func (r *InMemoryMessageRepo) DeleteMessage(ctx context.Context, messageID string) error {
	if r.messages != nil {
		delete(r.messages, messageID)
	}
	return nil
}

func (r *InMemoryMessageRepo) ListMessages(ctx context.Context, filter messaging.MessageFilter, limit, offset int) ([]*models.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *InMemoryMessageRepo) GetMessageCount(ctx context.Context, filter messaging.MessageFilter) (int, error) {
	return 0, nil
}

type InMemoryAttemptRepo struct {
	attempts map[string][]*models.DeliveryAttempt
}

func (r *InMemoryAttemptRepo) CreateDeliveryAttempt(ctx context.Context, attempt *models.DeliveryAttempt) error {
	if r.attempts == nil {
		r.attempts = make(map[string][]*models.DeliveryAttempt)
	}
	messageID := attempt.MessageID.String()
	r.attempts[messageID] = append(r.attempts[messageID], attempt)
	return nil
}

func (r *InMemoryAttemptRepo) GetDeliveryAttempt(ctx context.Context, attemptID string) (*models.DeliveryAttempt, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *InMemoryAttemptRepo) UpdateDeliveryAttempt(ctx context.Context, attempt *models.DeliveryAttempt) error {
	return nil // For demo, we don't update
}

func (r *InMemoryAttemptRepo) ListDeliveryAttempts(ctx context.Context, messageID string) ([]*models.DeliveryAttempt, error) {
	if r.attempts == nil {
		return []*models.DeliveryAttempt{}, nil
	}
	attempts, exists := r.attempts[messageID]
	if !exists {
		return []*models.DeliveryAttempt{}, nil
	}
	return attempts, nil
}

func (r *InMemoryAttemptRepo) ListFailedAttempts(ctx context.Context, channel models.DeliveryChannel, before time.Time) ([]*models.DeliveryAttempt, error) {
	return []*models.DeliveryAttempt{}, nil
}

func (r *InMemoryAttemptRepo) GetAttemptMetrics(ctx context.Context, start, end time.Time) (*models.MessageMetrics, error) {
	return &models.MessageMetrics{}, nil
}

type InMemoryPreferenceService struct{}

func (s *InMemoryPreferenceService) GetUserPreferences(ctx context.Context, userID string) (*models.UserNotificationSettings, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	prefs := models.DefaultUserNotificationSettings(uid, "test@example.com")
	return &prefs, nil
}

func (s *InMemoryPreferenceService) UpdateUserPreferences(ctx context.Context, userID string, preferences *models.UserNotificationSettings) error {
	return nil
}

func (s *InMemoryPreferenceService) UpdateChannelSettings(ctx context.Context, userID string, channel models.DeliveryChannel, settings models.ChannelConfig) error {
	return nil
}

func (s *InMemoryPreferenceService) VerifyChannel(ctx context.Context, userID string, channel models.DeliveryChannel, token string) error {
	return nil
}

func (s *InMemoryPreferenceService) SendChannelVerification(ctx context.Context, userID string, channel models.DeliveryChannel, address string) error {
	return nil
}

func (s *InMemoryPreferenceService) GetChannelVerificationStatus(ctx context.Context, userID string, channel models.DeliveryChannel) (bool, error) {
	return true, nil
}

func (s *InMemoryPreferenceService) DisableNotifications(ctx context.Context, userID string, duration time.Duration) error {
	return nil
}

func (s *InMemoryPreferenceService) GetNotificationHistory(ctx context.Context, userID string, limit int) ([]*models.DeliveryAttempt, error) {
	return []*models.DeliveryAttempt{}, nil
}
