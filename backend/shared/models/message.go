package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MessageType represents the type of message being sent
type MessageType string

const (
	MessageSubscriptionUpdate MessageType = "subscription_update"
	MessageCommentNotify      MessageType = "comment_notification"
	MessageKudosNotify        MessageType = "kudos_notification"
	MessageSystemAlert        MessageType = "system_alert"
	MessagePasswordReset      MessageType = "password_reset"
	MessageAccountSecurity    MessageType = "account_security"
	MessageCollectionUpdate   MessageType = "collection_update"
	MessageSeriesUpdate       MessageType = "series_update"
	MessageInvitation         MessageType = "invitation"
)

// MessageStatus represents the current status of a message
type MessageStatus string

const (
	MessageStatusPending    MessageStatus = "pending"
	MessageStatusProcessing MessageStatus = "processing"
	MessageStatusCompleted  MessageStatus = "completed"
	MessageStatusFailed     MessageStatus = "failed"
	MessageStatusPartial    MessageStatus = "partial"
)

// DeliveryChannel represents the channel through which a message can be delivered
type DeliveryChannel string

const (
	ChannelEmail   DeliveryChannel = "email"
	ChannelPush    DeliveryChannel = "push"
	ChannelSMS     DeliveryChannel = "sms"
	ChannelWebhook DeliveryChannel = "webhook"
	ChannelInApp   DeliveryChannel = "in_app"
)

// DeliveryStatus represents the status of a delivery attempt
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusSent      DeliveryStatus = "sent"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusBounced   DeliveryStatus = "bounced"
	DeliveryStatusRetrying  DeliveryStatus = "retrying"
)

// NotificationFrequency represents how often notifications should be sent
type NotificationFrequency string

const (
	FrequencyImmediate NotificationFrequency = "immediate"
	FrequencyBatched   NotificationFrequency = "batched"
	FrequencyDaily     NotificationFrequency = "daily"
	FrequencyWeekly    NotificationFrequency = "weekly"
	FrequencyNever     NotificationFrequency = "never"
)

// Message represents a notification message that can be delivered through multiple channels
type Message struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	Type       MessageType            `json:"type" db:"type"`
	Content    MessageContent         `json:"content" db:"content"`
	Metadata   map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
	Status     MessageStatus          `json:"status" db:"status"`
	Recipients []Recipient            `json:"recipients,omitempty" db:"-"`
}

// MessageContent represents the content of a message across different channels
type MessageContent struct {
	Subject   string                 `json:"subject"`
	PlainText string                 `json:"plain_text"`
	HTML      string                 `json:"html,omitempty"`
	Templates map[string]string      `json:"templates,omitempty"` // channel-specific templates
	Variables map[string]interface{} `json:"variables,omitempty"`
	ActionURL string                 `json:"action_url,omitempty"`
}

// Recipient represents a message recipient with their delivery preferences
type Recipient struct {
	UserID      uuid.UUID                `json:"user_id"`
	Channels    []DeliveryChannel        `json:"channels"`
	Preferences UserNotificationSettings `json:"preferences"`
	Context     map[string]interface{}   `json:"context,omitempty"`
}

// DeliveryAttempt represents an attempt to deliver a message through a specific channel
type DeliveryAttempt struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	MessageID   uuid.UUID              `json:"message_id" db:"message_id"`
	UserID      uuid.UUID              `json:"user_id" db:"user_id"`
	Channel     DeliveryChannel        `json:"channel" db:"channel"`
	Status      DeliveryStatus         `json:"status" db:"status"`
	AttemptedAt time.Time              `json:"attempted_at" db:"attempted_at"`
	DeliveredAt *time.Time             `json:"delivered_at,omitempty" db:"delivered_at"`
	Error       *DeliveryError         `json:"error,omitempty" db:"error"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	RetryCount  int                    `json:"retry_count" db:"retry_count"`
	NextRetryAt *time.Time             `json:"next_retry_at,omitempty" db:"next_retry_at"`
}

// DeliveryError represents an error that occurred during message delivery
type DeliveryError struct {
	Type      string                 `json:"type"`
	Code      string                 `json:"code,omitempty"`
	Message   string                 `json:"message"`
	Retryable bool                   `json:"retryable"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// UserNotificationSettings represents a user's notification preferences
type UserNotificationSettings struct {
	UserID        uuid.UUID                         `json:"user_id" db:"user_id"`
	GlobalEnabled bool                              `json:"global_enabled" db:"global_enabled"`
	Channels      map[DeliveryChannel]ChannelConfig `json:"channels" db:"channels"`
	MessageTypes  map[MessageType]MessageTypeConfig `json:"message_types" db:"message_types"`
	QuietHours    *QuietHoursConfig                 `json:"quiet_hours,omitempty" db:"quiet_hours"`
	UpdatedAt     time.Time                         `json:"updated_at" db:"updated_at"`
}

// ChannelConfig represents configuration for a specific delivery channel
type ChannelConfig struct {
	Enabled    bool                   `json:"enabled"`
	Address    string                 `json:"address"` // email, phone, webhook URL
	Settings   map[string]interface{} `json:"settings,omitempty"`
	VerifiedAt *time.Time             `json:"verified_at,omitempty"`
}

// MessageTypeConfig represents configuration for a specific message type
type MessageTypeConfig struct {
	Enabled     bool                  `json:"enabled"`
	Channels    []DeliveryChannel     `json:"channels"`
	Frequency   NotificationFrequency `json:"frequency"`
	BatchWindow *time.Duration        `json:"batch_window,omitempty"`
}

// QuietHoursConfig represents user's quiet hours preferences
type QuietHoursConfig struct {
	Enabled   bool   `json:"enabled"`
	StartTime string `json:"start_time"` // HH:MM format
	EndTime   string `json:"end_time"`   // HH:MM format
	Timezone  string `json:"timezone"`   // IANA timezone
}

// ChannelVerification represents a pending verification for a delivery channel
type ChannelVerification struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	UserID     uuid.UUID       `json:"user_id" db:"user_id"`
	Channel    DeliveryChannel `json:"channel" db:"channel"`
	Address    string          `json:"address" db:"address"`
	Token      string          `json:"token" db:"token"`
	VerifiedAt *time.Time      `json:"verified_at,omitempty" db:"verified_at"`
	ExpiresAt  time.Time       `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}

// MessageMetrics represents aggregated metrics for message delivery
type MessageMetrics struct {
	TotalSent      int64                              `json:"total_sent"`
	TotalDelivered int64                              `json:"total_delivered"`
	TotalFailed    int64                              `json:"total_failed"`
	DeliveryRate   float64                            `json:"delivery_rate"`
	AverageLatency int64                              `json:"average_latency_ms"`
	ByChannel      map[DeliveryChannel]ChannelMetrics `json:"by_channel"`
}

// ChannelMetrics represents metrics for a specific delivery channel
type ChannelMetrics struct {
	Sent         int64   `json:"sent"`
	Delivered    int64   `json:"delivered"`
	Failed       int64   `json:"failed"`
	DeliveryRate float64 `json:"delivery_rate"`
	AvgLatency   int64   `json:"avg_latency_ms"`
}

// DefaultUserNotificationSettings returns the default notification settings for a new user
func DefaultUserNotificationSettings(userID uuid.UUID, email string) UserNotificationSettings {
	return UserNotificationSettings{
		UserID:        userID,
		GlobalEnabled: true,
		Channels: map[DeliveryChannel]ChannelConfig{
			ChannelEmail: {
				Enabled: true,
				Address: email,
				Settings: map[string]interface{}{
					"format": "html",
				},
			},
			ChannelPush: {
				Enabled: false,
				Settings: map[string]interface{}{
					"platforms": []string{},
				},
			},
			ChannelSMS: {
				Enabled: false,
			},
		},
		MessageTypes: map[MessageType]MessageTypeConfig{
			MessageSubscriptionUpdate: {
				Enabled:   true,
				Channels:  []DeliveryChannel{ChannelEmail},
				Frequency: FrequencyImmediate,
			},
			MessageCommentNotify: {
				Enabled:   true,
				Channels:  []DeliveryChannel{ChannelEmail},
				Frequency: FrequencyImmediate,
			},
			MessageKudosNotify: {
				Enabled:   true,
				Channels:  []DeliveryChannel{ChannelEmail},
				Frequency: FrequencyImmediate,
			},
			MessageSystemAlert: {
				Enabled:   true,
				Channels:  []DeliveryChannel{ChannelEmail},
				Frequency: FrequencyImmediate,
			},
			MessagePasswordReset: {
				Enabled:   true,
				Channels:  []DeliveryChannel{ChannelEmail},
				Frequency: FrequencyImmediate,
			},
			MessageAccountSecurity: {
				Enabled:   true,
				Channels:  []DeliveryChannel{ChannelEmail},
				Frequency: FrequencyImmediate,
			},
		},
		UpdatedAt: time.Now(),
	}
}

// ShouldSendToChannel determines if a message should be sent through a specific channel
func (m *Message) ShouldSendToChannel(userSettings UserNotificationSettings, channel DeliveryChannel) bool {
	// Check global enabled
	if !userSettings.GlobalEnabled {
		return false
	}

	// Check channel enabled
	channelConfig, exists := userSettings.Channels[channel]
	if !exists || !channelConfig.Enabled {
		return false
	}

	// Check message type settings
	messageTypeConfig, exists := userSettings.MessageTypes[m.Type]
	if !exists || !messageTypeConfig.Enabled {
		return false
	}

	// Check if channel is in allowed channels for this message type
	for _, allowedChannel := range messageTypeConfig.Channels {
		if allowedChannel == channel {
			return true
		}
	}

	return false
}

// IsInQuietHours checks if the current time is within the user's quiet hours
func (q *QuietHoursConfig) IsInQuietHours(now time.Time) bool {
	if !q.Enabled {
		return false
	}

	// Load timezone
	loc, err := time.LoadLocation(q.Timezone)
	if err != nil {
		// Default to UTC if timezone is invalid
		loc = time.UTC
	}

	// Convert to user's timezone
	userTime := now.In(loc)

	// Parse start and end times
	startHour, startMin := parseTime(q.StartTime)
	endHour, endMin := parseTime(q.EndTime)

	startTime := time.Date(userTime.Year(), userTime.Month(), userTime.Day(), startHour, startMin, 0, 0, loc)
	endTime := time.Date(userTime.Year(), userTime.Month(), userTime.Day(), endHour, endMin, 0, 0, loc)

	// Handle overnight quiet hours (e.g., 22:00 to 08:00)
	if endTime.Before(startTime) {
		endTime = endTime.Add(24 * time.Hour)
		if userTime.Before(startTime) {
			startTime = startTime.Add(-24 * time.Hour)
		}
	}

	return userTime.After(startTime) && userTime.Before(endTime)
}

// parseTime parses a time string in HH:MM format
func parseTime(timeStr string) (hour, min int) {
	// Simple parsing for HH:MM format
	if len(timeStr) != 5 || timeStr[2] != ':' {
		return 0, 0
	}

	hour = int(timeStr[0]-'0')*10 + int(timeStr[1]-'0')
	min = int(timeStr[3]-'0')*10 + int(timeStr[4]-'0')

	if hour < 0 || hour > 23 || min < 0 || min > 59 {
		return 0, 0
	}

	return hour, min
}

// JSON marshaling helpers for JSONB storage in PostgreSQL

func (mc MessageContent) Value() (interface{}, error) {
	return json.Marshal(mc)
}

func (mc *MessageContent) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, mc)
}

func (de DeliveryError) Value() (interface{}, error) {
	return json.Marshal(de)
}

func (de *DeliveryError) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, de)
}
