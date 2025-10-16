package messaging

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"nuclear-ao3/shared/models"
)

// UniversalMessageService implements the MessageService interface
type UniversalMessageService struct {
	mu                sync.RWMutex
	channelProviders  map[models.DeliveryChannel]ChannelProvider
	telemetry         TelemetryCollector
	validator         MessageValidator
	rateLimiter       RateLimiter
	messageRepo       MessageRepository
	attemptRepo       DeliveryAttemptRepository
	preferenceService PreferenceService
}

// NewUniversalMessageService creates a new universal message service
func NewUniversalMessageService(
	telemetry TelemetryCollector,
	validator MessageValidator,
	rateLimiter RateLimiter,
	messageRepo MessageRepository,
	attemptRepo DeliveryAttemptRepository,
	preferenceService PreferenceService,
) *UniversalMessageService {
	return &UniversalMessageService{
		channelProviders:  make(map[models.DeliveryChannel]ChannelProvider),
		telemetry:         telemetry,
		validator:         validator,
		rateLimiter:       rateLimiter,
		messageRepo:       messageRepo,
		attemptRepo:       attemptRepo,
		preferenceService: preferenceService,
	}
}

// RegisterChannelProvider registers a new channel provider
func (s *UniversalMessageService) RegisterChannelProvider(provider ChannelProvider) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	channelType := provider.GetChannelType()
	s.channelProviders[channelType] = provider

	log.Printf("Registered channel provider for %s", channelType)
	return nil
}

// SendMessage sends a message to recipients based on their preferences
func (s *UniversalMessageService) SendMessage(ctx context.Context, msg *models.Message) error {
	// Validate message
	if err := s.validator.ValidateMessage(msg); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	// Generate message ID if not provided
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}

	// Set initial status and timestamps
	msg.Status = models.MessageStatusPending
	now := time.Now()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	msg.UpdatedAt = now

	// Store message
	if err := s.messageRepo.CreateMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	// Process each recipient
	var allErrors []error
	successCount := 0

	for _, recipient := range msg.Recipients {
		if err := s.processRecipient(ctx, msg, &recipient); err != nil {
			log.Printf("Failed to process recipient %s: %v", recipient.UserID, err)
			allErrors = append(allErrors, err)
		} else {
			successCount++
		}
	}

	// Update message status based on results
	if successCount == 0 {
		msg.Status = models.MessageStatusFailed
	} else if len(allErrors) > 0 {
		msg.Status = models.MessageStatusPartial
	} else {
		msg.Status = models.MessageStatusCompleted
	}

	msg.UpdatedAt = time.Now()
	s.messageRepo.UpdateMessage(ctx, msg)

	// Return error if no recipients were processed successfully
	if successCount == 0 {
		return fmt.Errorf("failed to deliver to any recipients: %d errors", len(allErrors))
	}

	log.Printf("Message %s sent to %d/%d recipients", msg.ID, successCount, len(msg.Recipients))
	return nil
}

// processRecipient processes a single recipient
func (s *UniversalMessageService) processRecipient(ctx context.Context, msg *models.Message, recipient *models.Recipient) error {
	// Validate recipient
	if err := s.validator.ValidateRecipient(recipient); err != nil {
		return fmt.Errorf("recipient validation failed: %w", err)
	}

	// Get user preferences if not already provided
	if recipient.Preferences.UserID == uuid.Nil {
		userPrefs, err := s.preferenceService.GetUserPreferences(ctx, recipient.UserID.String())
		if err != nil {
			log.Printf("Failed to get user preferences for %s, using defaults: %v", recipient.UserID, err)
			// Use default preferences
			recipient.Preferences = models.DefaultUserNotificationSettings(recipient.UserID, "")
		} else {
			recipient.Preferences = *userPrefs
		}
	}

	// Check if user has notifications globally disabled
	if !recipient.Preferences.GlobalEnabled {
		log.Printf("Notifications globally disabled for user %s", recipient.UserID)
		return nil
	}

	// Check quiet hours
	if recipient.Preferences.QuietHours != nil && recipient.Preferences.QuietHours.IsInQuietHours(time.Now()) {
		log.Printf("User %s is in quiet hours, skipping immediate delivery", recipient.UserID)
		// In a full implementation, we would schedule for later delivery
		return nil
	}

	// Determine which channels to use
	channels := s.determineChannelsForRecipient(msg, &recipient.Preferences)
	if len(channels) == 0 {
		log.Printf("No enabled channels for user %s and message type %s", recipient.UserID, msg.Type)
		return nil
	}

	// Send through each enabled channel
	var channelErrors []error
	for _, channel := range channels {
		if err := s.sendThroughChannel(ctx, msg, recipient, channel); err != nil {
			channelErrors = append(channelErrors, fmt.Errorf("channel %s: %w", channel, err))
		}
	}

	if len(channelErrors) == len(channels) {
		return fmt.Errorf("all channels failed: %v", channelErrors)
	}

	return nil
}

// determineChannelsForRecipient determines which channels to use for a recipient
func (s *UniversalMessageService) determineChannelsForRecipient(msg *models.Message, prefs *models.UserNotificationSettings) []models.DeliveryChannel {
	var channels []models.DeliveryChannel

	// Check message type configuration
	msgTypeConfig, exists := prefs.MessageTypes[msg.Type]
	if !exists || !msgTypeConfig.Enabled {
		return channels
	}

	// Check each channel from message type config
	for _, channel := range msgTypeConfig.Channels {
		// Check if channel is enabled for user
		channelConfig, exists := prefs.Channels[channel]
		if !exists || !channelConfig.Enabled {
			continue
		}

		// Check if we have a provider for this channel
		s.mu.RLock()
		provider, exists := s.channelProviders[channel]
		s.mu.RUnlock()
		if !exists {
			continue
		}

		// Check if channel is available
		if !provider.IsAvailable(context.Background()) {
			continue
		}

		channels = append(channels, channel)
	}

	return channels
}

// sendThroughChannel sends a message through a specific channel
func (s *UniversalMessageService) sendThroughChannel(ctx context.Context, msg *models.Message, recipient *models.Recipient, channel models.DeliveryChannel) error {
	s.mu.RLock()
	provider, exists := s.channelProviders[channel]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no provider for channel %s", channel)
	}

	// Check rate limiting
	if !s.rateLimiter.Allow(ctx, channel, recipient.Preferences.Channels[channel].Address) {
		return fmt.Errorf("rate limited for channel %s", channel)
	}

	// Validate content for this channel
	if err := s.validator.ValidateContent(&msg.Content, channel); err != nil {
		return fmt.Errorf("content validation failed for channel %s: %w", channel, err)
	}

	// Deliver message
	attempt, err := provider.DeliverMessage(ctx, msg, recipient)
	if err != nil {
		s.telemetry.RecordError(channel, "delivery_error", err)
	}

	// Store delivery attempt
	if attempt != nil {
		s.attemptRepo.CreateDeliveryAttempt(ctx, attempt)
		s.telemetry.RecordDeliveryAttempt(attempt)
	}

	return err
}

// ScheduleMessage schedules a message for future delivery
func (s *UniversalMessageService) ScheduleMessage(ctx context.Context, msg *models.Message, deliverAt time.Time) error {
	// Validate message
	if err := s.validator.ValidateMessage(msg); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	// Generate ID and set status
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	msg.Status = models.MessageStatusPending
	now := time.Now()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	msg.UpdatedAt = now

	// Store message with scheduled delivery time in metadata
	msg.Metadata = map[string]interface{}{
		"scheduled_delivery": deliverAt.Format(time.RFC3339),
		"scheduled":          true,
	}

	if err := s.messageRepo.CreateMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to store scheduled message: %w", err)
	}

	log.Printf("Message %s scheduled for delivery at %s", msg.ID, deliverAt.Format(time.RFC3339))
	return nil
}

// GetMessageStatus retrieves the status of a message and all its delivery attempts
func (s *UniversalMessageService) GetMessageStatus(ctx context.Context, messageID string) (*MessageStatus, error) {
	_, err := uuid.Parse(messageID)
	if err != nil {
		return nil, fmt.Errorf("invalid message ID: %w", err)
	}

	// Get message
	msg, err := s.messageRepo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Get delivery attempts
	attempts, err := s.attemptRepo.ListDeliveryAttempts(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery attempts: %w", err)
	}

	// Build summary
	summary := &DeliveryStatusSummary{
		TotalRecipients: len(msg.Recipients),
		TotalAttempts:   len(attempts),
		ByChannel:       make(map[models.DeliveryChannel]*ChannelSummary),
	}

	for _, attempt := range attempts {
		// Update totals
		switch attempt.Status {
		case models.DeliveryStatusSent:
			summary.SuccessfulSent++
		case models.DeliveryStatusDelivered:
			summary.SuccessfulDelivered++
		case models.DeliveryStatusFailed:
			summary.Failed++
		case models.DeliveryStatusPending:
			summary.Pending++
		}

		// Update channel summary
		if _, exists := summary.ByChannel[attempt.Channel]; !exists {
			summary.ByChannel[attempt.Channel] = &ChannelSummary{
				ByStatus: make(map[models.DeliveryStatus]int),
			}
		}
		channelSummary := summary.ByChannel[attempt.Channel]
		channelSummary.Attempts++
		channelSummary.ByStatus[attempt.Status]++

		switch attempt.Status {
		case models.DeliveryStatusSent:
			channelSummary.Sent++
		case models.DeliveryStatusDelivered:
			channelSummary.Delivered++
		case models.DeliveryStatusFailed:
			channelSummary.Failed++
		case models.DeliveryStatusPending:
			channelSummary.Pending++
		}
	}

	status := &MessageStatus{
		Message:          msg,
		DeliveryAttempts: attempts,
		Summary:          summary,
	}

	return status, nil
}

// RetryFailedDeliveries retries failed delivery attempts for a message
func (s *UniversalMessageService) RetryFailedDeliveries(ctx context.Context, messageID string) error {
	// Get failed attempts
	attempts, err := s.attemptRepo.ListFailedAttempts(ctx, models.ChannelEmail, time.Now())
	if err != nil {
		return fmt.Errorf("failed to get failed attempts: %w", err)
	}

	retryCount := 0
	for _, attempt := range attempts {
		if attempt.MessageID.String() == messageID {
			// Check if retry is appropriate
			if attempt.Error != nil && !attempt.Error.Retryable {
				continue
			}

			// Get message and recipient info
			msg, err := s.messageRepo.GetMessage(ctx, messageID)
			if err != nil {
				continue
			}

			// Find the recipient for this attempt
			var recipient *models.Recipient
			for _, r := range msg.Recipients {
				if r.UserID == attempt.UserID {
					recipient = &r
					break
				}
			}

			if recipient == nil {
				continue
			}

			// Retry delivery
			if err := s.sendThroughChannel(ctx, msg, recipient, attempt.Channel); err != nil {
				log.Printf("Retry failed for attempt %s: %v", attempt.ID, err)
			} else {
				retryCount++
			}
		}
	}

	log.Printf("Retried %d failed deliveries for message %s", retryCount, messageID)
	return nil
}

// GetMetrics returns aggregate metrics for message delivery
func (s *UniversalMessageService) GetMetrics(ctx context.Context, start, end time.Time) (*models.MessageMetrics, error) {
	return s.telemetry.GetMetrics(start, end)
}

// GetAvailableChannels returns a list of available delivery channels
func (s *UniversalMessageService) GetAvailableChannels(ctx context.Context) []models.DeliveryChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var channels []models.DeliveryChannel
	for channel, provider := range s.channelProviders {
		if provider.IsAvailable(ctx) {
			channels = append(channels, channel)
		}
	}

	return channels
}

// Simple implementations for missing interfaces (these would normally be separate packages)

// SimpleMessageValidator provides basic message validation
type SimpleMessageValidator struct{}

func (v *SimpleMessageValidator) ValidateMessage(msg *models.Message) error {
	if msg.Type == "" {
		return fmt.Errorf("message type cannot be empty")
	}
	if msg.Content.Subject == "" {
		return fmt.Errorf("message subject cannot be empty")
	}
	if msg.Content.PlainText == "" {
		return fmt.Errorf("message content cannot be empty")
	}
	if len(msg.Recipients) == 0 {
		return fmt.Errorf("message must have at least one recipient")
	}
	return nil
}

func (v *SimpleMessageValidator) ValidateRecipient(recipient *models.Recipient) error {
	if recipient.UserID == uuid.Nil {
		return fmt.Errorf("recipient user ID cannot be empty")
	}
	return nil
}

func (v *SimpleMessageValidator) ValidateContent(content *models.MessageContent, channel models.DeliveryChannel) error {
	switch channel {
	case models.ChannelEmail:
		if content.Subject == "" {
			return fmt.Errorf("email subject cannot be empty")
		}
		if content.PlainText == "" {
			return fmt.Errorf("email content cannot be empty")
		}
	case models.ChannelSMS:
		if len(content.PlainText) > 160 {
			return fmt.Errorf("SMS content too long (max 160 characters)")
		}
	}
	return nil
}

// SimpleRateLimiter provides basic rate limiting
type SimpleRateLimiter struct {
	limits map[models.DeliveryChannel]int
}

func NewSimpleRateLimiter() *SimpleRateLimiter {
	return &SimpleRateLimiter{
		limits: map[models.DeliveryChannel]int{
			models.ChannelEmail: 100, // 100 emails per window
			models.ChannelSMS:   10,  // 10 SMS per window
		},
	}
}

func (r *SimpleRateLimiter) Allow(ctx context.Context, channel models.DeliveryChannel, recipient string) bool {
	// Simple implementation - always allow for now
	// In production, this would track usage per channel/recipient
	return true
}

func (r *SimpleRateLimiter) GetLimit(channel models.DeliveryChannel) (int, time.Duration) {
	limit, exists := r.limits[channel]
	if !exists {
		limit = 50 // Default limit
	}
	return limit, time.Hour // Default window
}

func (r *SimpleRateLimiter) SetLimit(channel models.DeliveryChannel, requests int, window time.Duration) error {
	r.limits[channel] = requests
	return nil
}

func (r *SimpleRateLimiter) GetUsage(ctx context.Context, channel models.DeliveryChannel) (int, error) {
	return 0, nil // Always return 0 usage for this simple implementation
}
