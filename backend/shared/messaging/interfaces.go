package messaging

import (
	"context"
	"time"

	"nuclear-ao3/shared/models"
)

// ChannelProvider defines the interface for all message delivery channels
type ChannelProvider interface {
	// GetChannelType returns the channel type this provider handles
	GetChannelType() models.DeliveryChannel

	// DeliverMessage delivers a message to a recipient through this channel
	DeliverMessage(ctx context.Context, msg *models.Message, recipient *models.Recipient) (*models.DeliveryAttempt, error)

	// ValidateAddress validates if an address is valid for this channel
	ValidateAddress(address string) error

	// SendVerification sends a verification message to the address
	SendVerification(ctx context.Context, address string, token string) error

	// GetDeliveryStatus retrieves the current delivery status for a message
	GetDeliveryStatus(ctx context.Context, messageID string) (*models.DeliveryAttempt, error)

	// GetMetrics returns channel-specific metrics for a time period
	GetMetrics(ctx context.Context, start, end time.Time) (*models.ChannelMetrics, error)

	// IsAvailable checks if the channel is currently available for delivery
	IsAvailable(ctx context.Context) bool
}

// MessageService defines the interface for the universal messaging service
type MessageService interface {
	// SendMessage sends a message to recipients based on their preferences
	SendMessage(ctx context.Context, msg *models.Message) error

	// ScheduleMessage schedules a message for future delivery
	ScheduleMessage(ctx context.Context, msg *models.Message, deliverAt time.Time) error

	// GetMessageStatus retrieves the status of a message and all its delivery attempts
	GetMessageStatus(ctx context.Context, messageID string) (*MessageStatus, error)

	// RetryFailedDeliveries retries failed delivery attempts for a message
	RetryFailedDeliveries(ctx context.Context, messageID string) error

	// GetMetrics returns aggregate metrics for message delivery
	GetMetrics(ctx context.Context, start, end time.Time) (*models.MessageMetrics, error)

	// RegisterChannelProvider registers a new channel provider
	RegisterChannelProvider(provider ChannelProvider) error

	// GetAvailableChannels returns a list of available delivery channels
	GetAvailableChannels(ctx context.Context) []models.DeliveryChannel
}

// PreferenceService defines the interface for managing user notification preferences
type PreferenceService interface {
	// GetUserPreferences retrieves notification preferences for a user
	GetUserPreferences(ctx context.Context, userID string) (*models.UserNotificationSettings, error)

	// UpdateUserPreferences updates notification preferences for a user
	UpdateUserPreferences(ctx context.Context, userID string, preferences *models.UserNotificationSettings) error

	// UpdateChannelSettings updates settings for a specific channel
	UpdateChannelSettings(ctx context.Context, userID string, channel models.DeliveryChannel, settings models.ChannelConfig) error

	// VerifyChannel verifies a channel address for a user
	VerifyChannel(ctx context.Context, userID string, channel models.DeliveryChannel, token string) error

	// SendChannelVerification sends verification for a channel
	SendChannelVerification(ctx context.Context, userID string, channel models.DeliveryChannel, address string) error

	// GetChannelVerificationStatus checks if a channel is verified
	GetChannelVerificationStatus(ctx context.Context, userID string, channel models.DeliveryChannel) (bool, error)

	// DisableNotifications temporarily disables all notifications for a user
	DisableNotifications(ctx context.Context, userID string, duration time.Duration) error

	// GetNotificationHistory returns recent notification history for a user
	GetNotificationHistory(ctx context.Context, userID string, limit int) ([]*models.DeliveryAttempt, error)
}

// TemplateService defines the interface for message template management
type TemplateService interface {
	// RenderTemplate renders a template for a specific channel
	RenderTemplate(ctx context.Context, templateName string, channel models.DeliveryChannel, variables map[string]interface{}) (*RenderedTemplate, error)

	// RegisterTemplate registers a new template
	RegisterTemplate(template *MessageTemplate) error

	// GetTemplate retrieves a template by name and channel
	GetTemplate(templateName string, channel models.DeliveryChannel) (*MessageTemplate, error)

	// ValidateTemplate validates template syntax
	ValidateTemplate(template *MessageTemplate) error
}

// TelemetryCollector defines the interface for collecting messaging telemetry
type TelemetryCollector interface {
	// RecordDeliveryAttempt records a delivery attempt
	RecordDeliveryAttempt(attempt *models.DeliveryAttempt)

	// RecordLatency records delivery latency
	RecordLatency(channel models.DeliveryChannel, duration time.Duration)

	// RecordError records an error
	RecordError(channel models.DeliveryChannel, errorType string, err error)

	// IncrementCounter increments a named counter
	IncrementCounter(name string, tags map[string]string)

	// RecordGauge records a gauge value
	RecordGauge(name string, value float64, tags map[string]string)

	// GetMetrics returns collected metrics
	GetMetrics(start, end time.Time) (*models.MessageMetrics, error)
}

// RetryStrategy defines the interface for retry logic
type RetryStrategy interface {
	// ShouldRetry determines if a delivery attempt should be retried
	ShouldRetry(attempt *models.DeliveryAttempt) bool

	// GetNextRetryTime calculates when the next retry should occur
	GetNextRetryTime(attempt *models.DeliveryAttempt) time.Time

	// GetMaxRetries returns the maximum number of retries for a channel
	GetMaxRetries(channel models.DeliveryChannel) int

	// GetRetryDelay returns the delay for a specific retry attempt
	GetRetryDelay(attempt *models.DeliveryAttempt) time.Duration
}

// MessageStatus represents the complete status of a message
type MessageStatus struct {
	Message          *models.Message           `json:"message"`
	DeliveryAttempts []*models.DeliveryAttempt `json:"delivery_attempts"`
	Summary          *DeliveryStatusSummary    `json:"summary"`
}

// DeliveryStatusSummary provides a summary of delivery attempts
type DeliveryStatusSummary struct {
	TotalRecipients     int                                        `json:"total_recipients"`
	TotalAttempts       int                                        `json:"total_attempts"`
	SuccessfulSent      int                                        `json:"successful_sent"`
	SuccessfulDelivered int                                        `json:"successful_delivered"`
	Failed              int                                        `json:"failed"`
	Pending             int                                        `json:"pending"`
	ByChannel           map[models.DeliveryChannel]*ChannelSummary `json:"by_channel"`
}

// ChannelSummary provides a summary for a specific channel
type ChannelSummary struct {
	Attempts  int                           `json:"attempts"`
	Sent      int                           `json:"sent"`
	Delivered int                           `json:"delivered"`
	Failed    int                           `json:"failed"`
	Pending   int                           `json:"pending"`
	ByStatus  map[models.DeliveryStatus]int `json:"by_status"`
}

// RenderedTemplate represents a rendered message template
type RenderedTemplate struct {
	Subject   string            `json:"subject"`
	PlainText string            `json:"plain_text"`
	HTML      string            `json:"html,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// MessageTemplate represents a message template
type MessageTemplate struct {
	Name        string                 `json:"name"`
	Channel     models.DeliveryChannel `json:"channel"`
	MessageType models.MessageType     `json:"message_type"`
	Subject     string                 `json:"subject"`
	PlainText   string                 `json:"plain_text"`
	HTML        string                 `json:"html,omitempty"`
	Variables   []TemplateVariable     `json:"variables"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TemplateVariable represents a variable used in templates
type TemplateVariable struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
}

// ErrorClassifier defines the interface for classifying delivery errors
type ErrorClassifier interface {
	// ClassifyError classifies an error and determines if it's retryable
	ClassifyError(err error, context map[string]interface{}) *models.DeliveryError

	// IsRetryable determines if an error type is retryable
	IsRetryable(errorType string) bool

	// GetErrorCategory returns the category of an error
	GetErrorCategory(errorType string) string
}

// MessageValidator defines the interface for validating messages
type MessageValidator interface {
	// ValidateMessage validates a message before sending
	ValidateMessage(msg *models.Message) error

	// ValidateRecipient validates a recipient
	ValidateRecipient(recipient *models.Recipient) error

	// ValidateContent validates message content
	ValidateContent(content *models.MessageContent, channel models.DeliveryChannel) error
}

// RateLimiter defines the interface for rate limiting message delivery
type RateLimiter interface {
	// Allow determines if a delivery attempt should be allowed
	Allow(ctx context.Context, channel models.DeliveryChannel, recipient string) bool

	// GetLimit returns the current rate limit for a channel
	GetLimit(channel models.DeliveryChannel) (int, time.Duration)

	// SetLimit sets the rate limit for a channel
	SetLimit(channel models.DeliveryChannel, requests int, window time.Duration) error

	// GetUsage returns current usage for a channel
	GetUsage(ctx context.Context, channel models.DeliveryChannel) (int, error)
}

// DeliveryQueue defines the interface for queueing message deliveries
type DeliveryQueue interface {
	// Enqueue adds a delivery attempt to the queue
	Enqueue(ctx context.Context, attempt *models.DeliveryAttempt) error

	// Dequeue removes and returns the next delivery attempt from the queue
	Dequeue(ctx context.Context, channel models.DeliveryChannel) (*models.DeliveryAttempt, error)

	// EnqueueDelayed adds a delivery attempt to be processed at a specific time
	EnqueueDelayed(ctx context.Context, attempt *models.DeliveryAttempt, deliverAt time.Time) error

	// GetQueueDepth returns the number of items in the queue for a channel
	GetQueueDepth(ctx context.Context, channel models.DeliveryChannel) (int, error)

	// GetQueueStats returns statistics about the queue
	GetQueueStats(ctx context.Context) (*QueueStats, error)
}

// QueueStats represents statistics about the delivery queue
type QueueStats struct {
	TotalPending int                            `json:"total_pending"`
	ByChannel    map[models.DeliveryChannel]int `json:"by_channel"`
	OldestItem   *time.Time                     `json:"oldest_item,omitempty"`
	AverageAge   time.Duration                  `json:"average_age"`
}

// MessageRepository defines the interface for persisting messages
type MessageRepository interface {
	// CreateMessage creates a new message
	CreateMessage(ctx context.Context, msg *models.Message) error

	// GetMessage retrieves a message by ID
	GetMessage(ctx context.Context, messageID string) (*models.Message, error)

	// UpdateMessage updates a message
	UpdateMessage(ctx context.Context, msg *models.Message) error

	// DeleteMessage deletes a message
	DeleteMessage(ctx context.Context, messageID string) error

	// ListMessages lists messages with pagination
	ListMessages(ctx context.Context, filter MessageFilter, limit, offset int) ([]*models.Message, error)

	// GetMessageCount returns the total count of messages matching the filter
	GetMessageCount(ctx context.Context, filter MessageFilter) (int, error)
}

// DeliveryAttemptRepository defines the interface for persisting delivery attempts
type DeliveryAttemptRepository interface {
	// CreateDeliveryAttempt creates a new delivery attempt
	CreateDeliveryAttempt(ctx context.Context, attempt *models.DeliveryAttempt) error

	// GetDeliveryAttempt retrieves a delivery attempt by ID
	GetDeliveryAttempt(ctx context.Context, attemptID string) (*models.DeliveryAttempt, error)

	// UpdateDeliveryAttempt updates a delivery attempt
	UpdateDeliveryAttempt(ctx context.Context, attempt *models.DeliveryAttempt) error

	// ListDeliveryAttempts lists delivery attempts for a message
	ListDeliveryAttempts(ctx context.Context, messageID string) ([]*models.DeliveryAttempt, error)

	// ListFailedAttempts lists failed delivery attempts that should be retried
	ListFailedAttempts(ctx context.Context, channel models.DeliveryChannel, before time.Time) ([]*models.DeliveryAttempt, error)

	// GetAttemptMetrics returns metrics for delivery attempts
	GetAttemptMetrics(ctx context.Context, start, end time.Time) (*models.MessageMetrics, error)
}

// MessageFilter defines filters for querying messages
type MessageFilter struct {
	MessageType *models.MessageType   `json:"message_type,omitempty"`
	Status      *models.MessageStatus `json:"status,omitempty"`
	StartTime   *time.Time            `json:"start_time,omitempty"`
	EndTime     *time.Time            `json:"end_time,omitempty"`
	UserID      *string               `json:"user_id,omitempty"`
}
