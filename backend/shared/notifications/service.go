package notifications

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"nuclear-ao3/shared/messaging"
	"nuclear-ao3/shared/models"
)

// NotificationService provides comprehensive notification management
type NotificationService struct {
	messageService   messaging.MessageService
	subscriptionRepo SubscriptionRepository
	notificationRepo NotificationRepository
	digestRepo       DigestRepository
	preferenceRepo   PreferenceRepository
	ruleEngine       *RuleEngine
	batchProcessor   *BatchProcessor
	smartFilter      *SmartFilter
}

// NotificationServiceConfig configures the notification service
type NotificationServiceConfig struct {
	EnableBatching       bool
	BatchIntervalMinutes int
	MaxBatchSize         int
	EnableSmartFiltering bool
	DefaultQuietHours    []string // ["22:00", "08:00"] format
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	messageService messaging.MessageService,
	subscriptionRepo SubscriptionRepository,
	notificationRepo NotificationRepository,
	digestRepo DigestRepository,
	preferenceRepo PreferenceRepository,
	config NotificationServiceConfig,
) *NotificationService {
	ns := &NotificationService{
		messageService:   messageService,
		subscriptionRepo: subscriptionRepo,
		notificationRepo: notificationRepo,
		digestRepo:       digestRepo,
		preferenceRepo:   preferenceRepo,
		ruleEngine:       NewRuleEngine(),
		smartFilter:      NewSmartFilter(),
	}

	if config.EnableBatching {
		ns.batchProcessor = NewBatchProcessor(ns, config.BatchIntervalMinutes, config.MaxBatchSize)
	}

	return ns
}

// ProcessEvent processes an incoming event and creates appropriate notifications
func (ns *NotificationService) ProcessEvent(ctx context.Context, event *EventData) error {
	log.Printf("Processing event: %s for %s", event.Type, event.SourceID)

	// Find all subscriptions that match this event
	subscriptions, err := ns.findMatchingSubscriptions(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to find matching subscriptions: %w", err)
	}

	log.Printf("Found %d matching subscriptions", len(subscriptions))

	// Create notifications for each subscription
	for _, subscription := range subscriptions {
		if err := ns.createNotificationForSubscription(ctx, event, subscription); err != nil {
			log.Printf("Failed to create notification for subscription %s: %v", subscription.ID, err)
			continue
		}
	}

	return nil
}

// findMatchingSubscriptions finds all subscriptions that should be notified for an event
func (ns *NotificationService) findMatchingSubscriptions(ctx context.Context, event *EventData) ([]*models.Subscription, error) {
	var allSubscriptions []*models.Subscription

	// Find direct subscriptions based on target
	switch event.Type {
	case models.EventWorkUpdated, models.EventWorkCompleted:
		// Find work subscriptions
		workSubs, err := ns.subscriptionRepo.FindByTarget(ctx, models.SubscriptionWork, event.SourceID)
		if err != nil {
			return nil, err
		}
		allSubscriptions = append(allSubscriptions, workSubs...)

		// Find author subscriptions if this work belongs to authors
		if event.AuthorIDs != nil {
			for _, authorID := range event.AuthorIDs {
				authorSubs, err := ns.subscriptionRepo.FindByTarget(ctx, models.SubscriptionAuthor, authorID)
				if err != nil {
					continue
				}
				allSubscriptions = append(allSubscriptions, authorSubs...)
			}
		}

		// Find series subscriptions if this work is part of series
		if event.SeriesIDs != nil {
			for _, seriesID := range event.SeriesIDs {
				seriesSubs, err := ns.subscriptionRepo.FindByTarget(ctx, models.SubscriptionSeries, seriesID)
				if err != nil {
					continue
				}
				allSubscriptions = append(allSubscriptions, seriesSubs...)
			}
		}

	case models.EventCommentReceived, models.EventCommentReplied:
		// Find work subscriptions for comment notifications
		workSubs, err := ns.subscriptionRepo.FindByTarget(ctx, models.SubscriptionWork, event.SourceID)
		if err != nil {
			return nil, err
		}
		allSubscriptions = append(allSubscriptions, workSubs...)

	case models.EventNewWork:
		// Find author subscriptions for new works
		if event.AuthorIDs != nil {
			for _, authorID := range event.AuthorIDs {
				authorSubs, err := ns.subscriptionRepo.FindByTarget(ctx, models.SubscriptionAuthor, authorID)
				if err != nil {
					continue
				}
				allSubscriptions = append(allSubscriptions, authorSubs...)
			}
		}
	}

	// Filter subscriptions that have this event enabled
	var matchingSubscriptions []*models.Subscription
	for _, sub := range allSubscriptions {
		if ns.subscriptionMatchesEvent(sub, event) {
			matchingSubscriptions = append(matchingSubscriptions, sub)
		}
	}

	return matchingSubscriptions, nil
}

// subscriptionMatchesEvent checks if a subscription should be notified for an event
func (ns *NotificationService) subscriptionMatchesEvent(sub *models.Subscription, event *EventData) bool {
	if !sub.IsActive {
		return false
	}

	// Check if this event type is enabled for this subscription
	eventEnabled := false
	for _, enabledEvent := range sub.Events {
		if enabledEvent == event.Type {
			eventEnabled = true
			break
		}
	}
	if !eventEnabled {
		return false
	}

	// Apply content filters
	if sub.FilterCompleted != nil && *sub.FilterCompleted != event.IsCompleted {
		return false
	}

	if len(sub.FilterRating) > 0 && event.Rating != "" {
		ratingAllowed := false
		for _, allowedRating := range sub.FilterRating {
			if allowedRating == event.Rating {
				ratingAllowed = true
				break
			}
		}
		if !ratingAllowed {
			return false
		}
	}

	if len(sub.FilterTags) > 0 && len(event.Tags) > 0 {
		hasRequiredTag := false
		for _, requiredTag := range sub.FilterTags {
			for _, eventTag := range event.Tags {
				if strings.EqualFold(requiredTag, eventTag) {
					hasRequiredTag = true
					break
				}
			}
			if hasRequiredTag {
				break
			}
		}
		if !hasRequiredTag {
			return false
		}
	}

	if sub.MinWordCount != nil && event.WordCount < *sub.MinWordCount {
		return false
	}

	if sub.MaxWordCount != nil && event.WordCount > *sub.MaxWordCount {
		return false
	}

	return true
}

// createNotificationForSubscription creates a notification for a specific subscription
func (ns *NotificationService) createNotificationForSubscription(ctx context.Context, event *EventData, subscription *models.Subscription) error {
	// Get user preferences
	prefs, err := ns.preferenceRepo.GetPreferences(ctx, subscription.UserID)
	if err != nil {
		log.Printf("Failed to get preferences for user %s, using defaults: %v", subscription.UserID, err)
		defaultPrefs := models.DefaultNotificationPreferences(subscription.UserID)
		prefs = &defaultPrefs
	}

	// Check if user wants notifications for this event
	eventPref, exists := prefs.EventPreferences[event.Type]
	if !exists || !eventPref.Enabled {
		return nil // User has disabled this event type
	}

	// Create notification item
	notification := &models.NotificationItem{
		ID:          uuid.New(),
		UserID:      subscription.UserID,
		Event:       event.Type,
		Priority:    eventPref.Priority,
		SourceID:    event.SourceID,
		SourceType:  event.SourceType,
		Title:       event.Title,
		Description: event.Description,
		ActionURL:   event.ActionURL,
		ActorID:     event.ActorID,
		ActorName:   event.ActorName,
		ExtraData:   event.ExtraData,
		CreatedAt:   time.Now(),
	}

	// Apply smart filtering
	if ns.smartFilter != nil {
		shouldNotify, modifiedNotification := ns.smartFilter.ShouldNotify(ctx, prefs, notification)
		if !shouldNotify {
			log.Printf("Smart filter blocked notification for user %s", subscription.UserID)
			return nil
		}
		if modifiedNotification != nil {
			notification = modifiedNotification
		}
	}

	// Apply user rules
	if ns.ruleEngine != nil {
		action := ns.ruleEngine.EvaluateNotification(ctx, prefs, notification)
		switch action.Action {
		case models.ActionBlock:
			log.Printf("User rule blocked notification for user %s", subscription.UserID)
			return nil
		case models.ActionModify:
			if action.ModifiedNotification != nil {
				notification = action.ModifiedNotification
			}
		}
	}

	// Save notification
	if err := ns.notificationRepo.CreateNotification(ctx, notification); err != nil {
		return fmt.Errorf("failed to save notification: %w", err)
	}

	// Handle delivery based on frequency preference
	switch eventPref.Frequency {
	case models.FrequencyImmediate:
		return ns.deliverNotificationImmediate(ctx, notification, eventPref.Channels)
	case models.FrequencyBatched, models.FrequencyDaily, models.FrequencyWeekly:
		if ns.batchProcessor != nil {
			return ns.batchProcessor.AddToBatch(ctx, notification)
		}
		return ns.deliverNotificationImmediate(ctx, notification, eventPref.Channels)
	case models.FrequencyNever:
		return nil // Just save, don't deliver
	default:
		return ns.deliverNotificationImmediate(ctx, notification, eventPref.Channels)
	}
}

// deliverNotificationImmediate delivers a notification immediately
func (ns *NotificationService) deliverNotificationImmediate(ctx context.Context, notification *models.NotificationItem, channels []models.DeliveryChannel) error {
	// Create message content
	content := &models.MessageContent{
		Subject:   notification.Title,
		PlainText: notification.Description,
		HTML:      notification.Description,
		ActionURL: notification.ActionURL,
		Variables: map[string]interface{}{
			"title":       notification.Title,
			"description": notification.Description,
			"action_url":  notification.ActionURL,
			"actor_name":  notification.ActorName,
		},
	}

	// Merge extra data into variables
	for k, v := range notification.ExtraData {
		content.Variables[k] = v
	}

	// Map notification event to message type
	messageType := ns.mapEventToMessageType(notification.Event)

	// Create message
	// Create channel configs for enabled channels
	channelConfigs := make(map[models.DeliveryChannel]models.ChannelConfig)
	for _, channel := range channels {
		channelConfigs[channel] = models.ChannelConfig{
			Enabled: true,
		}
	}

	message := &models.Message{
		Type:    messageType,
		Content: *content,
		Recipients: []models.Recipient{
			{
				UserID:   notification.UserID,
				Channels: channels,
				Preferences: models.UserNotificationSettings{
					UserID:        notification.UserID,
					GlobalEnabled: true,
					Channels:      channelConfigs,
					UpdatedAt:     time.Now(),
				},
			},
		},
	}

	// Send message
	if err := ns.messageService.SendMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to send notification message: %w", err)
	}

	// Update notification as delivered
	notification.IsDelivered = true
	now := time.Now()
	notification.DeliveredAt = &now

	return ns.notificationRepo.UpdateNotification(ctx, notification)
}

// mapEventToMessageType maps notification events to message types
func (ns *NotificationService) mapEventToMessageType(event models.NotificationEvent) models.MessageType {
	switch event {
	case models.EventWorkUpdated:
		return models.MessageSubscriptionUpdate
	case models.EventCommentReceived, models.EventCommentReplied:
		return models.MessageCommentNotify
	case models.EventKudosReceived:
		return models.MessageKudosNotify
	case models.EventSystemAlert:
		return models.MessageSystemAlert
	case models.EventPasswordReset:
		return models.MessagePasswordReset
	case models.EventAccountSecurity:
		return models.MessageAccountSecurity
	case models.EventCollectionInvite:
		return models.MessageInvitation
	case models.EventSeriesUpdated:
		return models.MessageSeriesUpdate
	default:
		return models.MessageSystemAlert // Default fallback
	}
}

// GetUserNotifications retrieves notifications for a user
func (ns *NotificationService) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.NotificationItem, error) {
	return ns.notificationRepo.GetUserNotifications(ctx, userID, limit, offset)
}

// MarkNotificationRead marks a notification as read
func (ns *NotificationService) MarkNotificationRead(ctx context.Context, notificationID uuid.UUID, userID uuid.UUID) error {
	notification, err := ns.notificationRepo.GetNotification(ctx, notificationID)
	if err != nil {
		return err
	}

	if notification.UserID != userID {
		return fmt.Errorf("notification does not belong to user")
	}

	notification.IsRead = true
	now := time.Now()
	notification.ReadAt = &now

	return ns.notificationRepo.UpdateNotification(ctx, notification)
}

// EventData represents an event that can trigger notifications
type EventData struct {
	Type        models.NotificationEvent `json:"type"`
	SourceID    uuid.UUID                `json:"source_id"`
	SourceType  string                   `json:"source_type"`
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	ActionURL   string                   `json:"action_url"`
	ActorID     *uuid.UUID               `json:"actor_id,omitempty"`
	ActorName   string                   `json:"actor_name"`
	ExtraData   map[string]interface{}   `json:"extra_data,omitempty"`

	// Content metadata for filtering
	AuthorIDs   []uuid.UUID `json:"author_ids,omitempty"`
	SeriesIDs   []uuid.UUID `json:"series_ids,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Rating      string      `json:"rating,omitempty"`
	WordCount   int         `json:"word_count,omitempty"`
	IsCompleted bool        `json:"is_completed,omitempty"`
}

// Repository interfaces for data access
type SubscriptionRepository interface {
	CreateSubscription(ctx context.Context, subscription *models.Subscription) error
	GetSubscription(ctx context.Context, id uuid.UUID) (*models.Subscription, error)
	UpdateSubscription(ctx context.Context, subscription *models.Subscription) error
	DeleteSubscription(ctx context.Context, id uuid.UUID) error
	FindByUser(ctx context.Context, userID uuid.UUID) ([]*models.Subscription, error)
	FindByTarget(ctx context.Context, targetType models.SubscriptionType, targetID uuid.UUID) ([]*models.Subscription, error)
	FindByUserAndTarget(ctx context.Context, userID, targetID uuid.UUID, targetType models.SubscriptionType) (*models.Subscription, error)
}

type NotificationRepository interface {
	CreateNotification(ctx context.Context, notification *models.NotificationItem) error
	GetNotification(ctx context.Context, id uuid.UUID) (*models.NotificationItem, error)
	UpdateNotification(ctx context.Context, notification *models.NotificationItem) error
	DeleteNotification(ctx context.Context, id uuid.UUID) error
	GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.NotificationItem, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	GetNotificationsForBatch(ctx context.Context, userID uuid.UUID, frequency models.NotificationFrequency) ([]*models.NotificationItem, error)
}

type DigestRepository interface {
	CreateDigest(ctx context.Context, digest *models.NotificationDigest) error
	GetDigest(ctx context.Context, id uuid.UUID) (*models.NotificationDigest, error)
	UpdateDigest(ctx context.Context, digest *models.NotificationDigest) error
	GetPendingDigests(ctx context.Context, digestType string) ([]*models.NotificationDigest, error)
}

type PreferenceRepository interface {
	GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error)
	UpdatePreferences(ctx context.Context, preferences *models.NotificationPreferences) error
	CreatePreferences(ctx context.Context, preferences *models.NotificationPreferences) error
}
