package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationPriority defines the priority level of notifications
type NotificationPriority string

const (
	PriorityHigh   NotificationPriority = "high"
	PriorityMedium NotificationPriority = "medium"
	PriorityLow    NotificationPriority = "low"
)

// SubscriptionType defines what type of content subscription this is
type SubscriptionType string

const (
	SubscriptionWork         SubscriptionType = "work"
	SubscriptionSeries       SubscriptionType = "series"
	SubscriptionAuthor       SubscriptionType = "author"
	SubscriptionTag          SubscriptionType = "tag"
	SubscriptionCollection   SubscriptionType = "collection"
	SubscriptionUser         SubscriptionType = "user"
	SubscriptionGiftExchange SubscriptionType = "gift_exchange"
)

// NotificationEvent represents different types of events that can trigger notifications
type NotificationEvent string

const (
	EventWorkUpdated      NotificationEvent = "work_updated"
	EventWorkCompleted    NotificationEvent = "work_completed"
	EventSeriesUpdated    NotificationEvent = "series_updated"
	EventNewWork          NotificationEvent = "new_work"
	EventCommentReceived  NotificationEvent = "comment_received"
	EventCommentReplied   NotificationEvent = "comment_replied"
	EventKudosReceived    NotificationEvent = "kudos_received"
	EventBookmarkAdded    NotificationEvent = "bookmark_added"
	EventGiftReceived     NotificationEvent = "gift_received"
	EventCollectionInvite NotificationEvent = "collection_invite"
	EventModeratorAction  NotificationEvent = "moderator_action"
	EventSystemAlert      NotificationEvent = "system_alert"
	EventAccountSecurity  NotificationEvent = "account_security"
	EventPasswordReset    NotificationEvent = "password_reset"
)

// Subscription represents a user's subscription to content
type Subscription struct {
	ID           uuid.UUID             `json:"id" db:"id"`
	UserID       uuid.UUID             `json:"user_id" db:"user_id"`
	Type         SubscriptionType      `json:"type" db:"type"`
	TargetID     uuid.UUID             `json:"target_id" db:"target_id"` // Work, Series, Author, etc.
	TargetName   string                `json:"target_name" db:"target_name"`
	Events       []NotificationEvent   `json:"events" db:"events"`
	Frequency    NotificationFrequency `json:"frequency" db:"frequency"`
	LastNotified *time.Time            `json:"last_notified,omitempty" db:"last_notified"`
	CreatedAt    time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at" db:"updated_at"`
	IsActive     bool                  `json:"is_active" db:"is_active"`

	// Advanced filtering options
	FilterCompleted *bool    `json:"filter_completed,omitempty" db:"filter_completed"`
	FilterRating    []string `json:"filter_rating,omitempty" db:"filter_rating"`
	FilterWarnings  []string `json:"filter_warnings,omitempty" db:"filter_warnings"`
	FilterTags      []string `json:"filter_tags,omitempty" db:"filter_tags"`
	MinWordCount    *int     `json:"min_word_count,omitempty" db:"min_word_count"`
	MaxWordCount    *int     `json:"max_word_count,omitempty" db:"max_word_count"`
}

// NotificationPreferences represents a user's notification preferences
type NotificationPreferences struct {
	UserID uuid.UUID `json:"user_id" db:"user_id"`

	// Global settings
	EmailEnabled bool `json:"email_enabled" db:"email_enabled"`
	WebEnabled   bool `json:"web_enabled" db:"web_enabled"`
	PushEnabled  bool `json:"push_enabled" db:"push_enabled"`

	// Event-specific settings
	EventPreferences map[NotificationEvent]EventPreference `json:"event_preferences" db:"event_preferences"`

	// Batching settings
	EnableBatching  bool                  `json:"enable_batching" db:"enable_batching"`
	BatchFrequency  NotificationFrequency `json:"batch_frequency" db:"batch_frequency"`
	QuietHoursStart *time.Time            `json:"quiet_hours_start,omitempty" db:"quiet_hours_start"`
	QuietHoursEnd   *time.Time            `json:"quiet_hours_end,omitempty" db:"quiet_hours_end"`
	Timezone        string                `json:"timezone" db:"timezone"`

	// Anti-spam settings
	MaxNotificationsPerHour int           `json:"max_notifications_per_hour" db:"max_notifications_per_hour"`
	MinTimeBetweenSimilar   time.Duration `json:"min_time_between_similar" db:"min_time_between_similar"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// EventPreference defines preferences for a specific event type
type EventPreference struct {
	Enabled   bool                  `json:"enabled"`
	Channels  []DeliveryChannel     `json:"channels"`
	Frequency NotificationFrequency `json:"frequency"`
	Priority  NotificationPriority  `json:"priority"`
	Template  string                `json:"template,omitempty"`
}

// NotificationDigest represents a batched collection of notifications
type NotificationDigest struct {
	ID            uuid.UUID          `json:"id" db:"id"`
	UserID        uuid.UUID          `json:"user_id" db:"user_id"`
	DigestType    string             `json:"digest_type" db:"digest_type"` // daily, weekly
	Notifications []NotificationItem `json:"notifications" db:"notifications"`
	SentAt        *time.Time         `json:"sent_at,omitempty" db:"sent_at"`
	CreatedAt     time.Time          `json:"created_at" db:"created_at"`
	Status        DigestStatus       `json:"status" db:"status"`
}

type DigestStatus string

const (
	DigestPending DigestStatus = "pending"
	DigestSent    DigestStatus = "sent"
	DigestFailed  DigestStatus = "failed"
)

// NotificationItem represents a single notification within a digest or standalone
type NotificationItem struct {
	ID       uuid.UUID            `json:"id" db:"id"`
	UserID   uuid.UUID            `json:"user_id" db:"user_id"`
	Event    NotificationEvent    `json:"event" db:"event"`
	Priority NotificationPriority `json:"priority" db:"priority"`

	// Content details
	SourceID    uuid.UUID `json:"source_id" db:"source_id"` // Work, Comment, etc.
	SourceType  string    `json:"source_type" db:"source_type"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	ActionURL   string    `json:"action_url" db:"action_url"`

	// Metadata
	ActorID   *uuid.UUID             `json:"actor_id,omitempty" db:"actor_id"` // User who triggered the event
	ActorName string                 `json:"actor_name" db:"actor_name"`
	ExtraData map[string]interface{} `json:"extra_data,omitempty" db:"extra_data"`

	// Delivery tracking
	IsRead      bool       `json:"is_read" db:"is_read"`
	ReadAt      *time.Time `json:"read_at,omitempty" db:"read_at"`
	IsDelivered bool       `json:"is_delivered" db:"is_delivered"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty" db:"delivered_at"`
	DigestID    *uuid.UUID `json:"digest_id,omitempty" db:"digest_id"`

	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}

// NotificationRule defines smart filtering rules for notifications
type NotificationRule struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`

	// Conditions
	Events          []NotificationEvent    `json:"events" db:"events"`
	SourceTypes     []string               `json:"source_types,omitempty" db:"source_types"`
	ActorConditions map[string]interface{} `json:"actor_conditions,omitempty" db:"actor_conditions"`
	ContentFilters  map[string]interface{} `json:"content_filters,omitempty" db:"content_filters"`
	TimeConditions  map[string]interface{} `json:"time_conditions,omitempty" db:"time_conditions"`

	// Actions
	Action       RuleAction            `json:"action" db:"action"`
	Priority     *NotificationPriority `json:"priority,omitempty" db:"priority"`
	ForceChannel *DeliveryChannel      `json:"force_channel,omitempty" db:"force_channel"`
	DelayMinutes *int                  `json:"delay_minutes,omitempty" db:"delay_minutes"`

	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type RuleAction string

const (
	ActionAllow    RuleAction = "allow"
	ActionBlock    RuleAction = "block"
	ActionModify   RuleAction = "modify"
	ActionBatch    RuleAction = "batch"
	ActionEscalate RuleAction = "escalate"
)

// SmartNotificationFilter provides intelligent filtering for notifications
type SmartNotificationFilter struct {
	UserID              uuid.UUID               `json:"user_id"`
	RecentNotifications []NotificationItem      `json:"recent_notifications"`
	UserPreferences     NotificationPreferences `json:"user_preferences"`
	ActiveRules         []NotificationRule      `json:"active_rules"`
}

// DefaultNotificationPreferences returns default notification preferences for a new user
func DefaultNotificationPreferences(userID uuid.UUID) NotificationPreferences {
	return NotificationPreferences{
		UserID:       userID,
		EmailEnabled: true,
		WebEnabled:   true,
		PushEnabled:  false,
		EventPreferences: map[NotificationEvent]EventPreference{
			EventWorkUpdated: {
				Enabled:   true,
				Channels:  []DeliveryChannel{ChannelEmail, ChannelInApp},
				Frequency: FrequencyImmediate,
				Priority:  PriorityMedium,
			},
			EventCommentReceived: {
				Enabled:   true,
				Channels:  []DeliveryChannel{ChannelEmail, ChannelInApp},
				Frequency: FrequencyImmediate,
				Priority:  PriorityHigh,
			},
			EventKudosReceived: {
				Enabled:   true,
				Channels:  []DeliveryChannel{ChannelInApp},
				Frequency: FrequencyDaily,
				Priority:  PriorityLow,
			},
			EventSystemAlert: {
				Enabled:   true,
				Channels:  []DeliveryChannel{ChannelEmail, ChannelInApp},
				Frequency: FrequencyImmediate,
				Priority:  PriorityHigh,
			},
		},
		EnableBatching:          true,
		BatchFrequency:          FrequencyDaily,
		MaxNotificationsPerHour: 10,
		MinTimeBetweenSimilar:   time.Hour,
		Timezone:                "UTC",
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}
}
