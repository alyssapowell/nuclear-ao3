package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"nuclear-ao3/shared/models"
	"nuclear-ao3/shared/notifications"
)

// SubscriptionRepositoryImpl implements the SubscriptionRepository interface
type SubscriptionRepositoryImpl struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) notifications.SubscriptionRepository {
	return &SubscriptionRepositoryImpl{db: db}
}

func (r *SubscriptionRepositoryImpl) CreateSubscription(ctx context.Context, subscription *models.Subscription) error {
	eventsJSON, _ := json.Marshal(subscription.Events)
	filterRatingJSON, _ := json.Marshal(subscription.FilterRating)
	filterTagsJSON, _ := json.Marshal(subscription.FilterTags)

	query := `
		INSERT INTO content_subscriptions 
		(id, user_id, target_type, target_id, events, is_active, created_at, updated_at,
		 filter_completed, filter_rating, filter_tags, min_word_count, max_word_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.ExecContext(ctx, query,
		subscription.ID, subscription.UserID, subscription.Type, subscription.TargetID,
		eventsJSON, subscription.IsActive, subscription.CreatedAt, subscription.UpdatedAt,
		subscription.FilterCompleted, filterRatingJSON, filterTagsJSON,
		subscription.MinWordCount, subscription.MaxWordCount,
	)
	return err
}

func (r *SubscriptionRepositoryImpl) GetSubscription(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	query := `
		SELECT id, user_id, target_type, target_id, events, is_active, created_at, updated_at,
		       filter_completed, filter_rating, filter_tags, min_word_count, max_word_count
		FROM content_subscriptions WHERE id = $1
	`
	var subscription models.Subscription
	var eventsJSON, filterRatingJSON, filterTagsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&subscription.ID, &subscription.UserID, &subscription.Type, &subscription.TargetID,
		&eventsJSON, &subscription.IsActive, &subscription.CreatedAt, &subscription.UpdatedAt,
		&subscription.FilterCompleted, &filterRatingJSON, &filterTagsJSON,
		&subscription.MinWordCount, &subscription.MaxWordCount,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(eventsJSON, &subscription.Events)
	json.Unmarshal(filterRatingJSON, &subscription.FilterRating)
	json.Unmarshal(filterTagsJSON, &subscription.FilterTags)

	return &subscription, nil
}

func (r *SubscriptionRepositoryImpl) UpdateSubscription(ctx context.Context, subscription *models.Subscription) error {
	eventsJSON, _ := json.Marshal(subscription.Events)
	filterRatingJSON, _ := json.Marshal(subscription.FilterRating)
	filterTagsJSON, _ := json.Marshal(subscription.FilterTags)

	query := `
		UPDATE content_subscriptions 
		SET events = $1, is_active = $2, updated_at = $3, filter_completed = $4,
		    filter_rating = $5, filter_tags = $6, min_word_count = $7, max_word_count = $8
		WHERE id = $9
	`
	_, err := r.db.ExecContext(ctx, query,
		eventsJSON, subscription.IsActive, time.Now(), subscription.FilterCompleted,
		filterRatingJSON, filterTagsJSON, subscription.MinWordCount, subscription.MaxWordCount,
		subscription.ID,
	)
	return err
}

func (r *SubscriptionRepositoryImpl) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM content_subscriptions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *SubscriptionRepositoryImpl) FindByUser(ctx context.Context, userID uuid.UUID) ([]*models.Subscription, error) {
	query := `
		SELECT id, user_id, target_type, target_id, events, is_active, created_at, updated_at,
		       filter_completed, filter_rating, filter_tags, min_word_count, max_word_count
		FROM content_subscriptions WHERE user_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []*models.Subscription
	for rows.Next() {
		var subscription models.Subscription
		var eventsJSON, filterRatingJSON, filterTagsJSON []byte

		err := rows.Scan(
			&subscription.ID, &subscription.UserID, &subscription.Type, &subscription.TargetID,
			&eventsJSON, &subscription.IsActive, &subscription.CreatedAt, &subscription.UpdatedAt,
			&subscription.FilterCompleted, &filterRatingJSON, &filterTagsJSON,
			&subscription.MinWordCount, &subscription.MaxWordCount,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(eventsJSON, &subscription.Events)
		json.Unmarshal(filterRatingJSON, &subscription.FilterRating)
		json.Unmarshal(filterTagsJSON, &subscription.FilterTags)

		subscriptions = append(subscriptions, &subscription)
	}

	return subscriptions, nil
}

func (r *SubscriptionRepositoryImpl) FindByTarget(ctx context.Context, targetType models.SubscriptionType, targetID uuid.UUID) ([]*models.Subscription, error) {
	query := `
		SELECT id, user_id, target_type, target_id, events, is_active, created_at, updated_at,
		       filter_completed, filter_rating, filter_tags, min_word_count, max_word_count
		FROM content_subscriptions WHERE target_type = $1 AND target_id = $2 AND is_active = true
	`
	rows, err := r.db.QueryContext(ctx, query, targetType, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []*models.Subscription
	for rows.Next() {
		var subscription models.Subscription
		var eventsJSON, filterRatingJSON, filterTagsJSON []byte

		err := rows.Scan(
			&subscription.ID, &subscription.UserID, &subscription.Type, &subscription.TargetID,
			&eventsJSON, &subscription.IsActive, &subscription.CreatedAt, &subscription.UpdatedAt,
			&subscription.FilterCompleted, &filterRatingJSON, &filterTagsJSON,
			&subscription.MinWordCount, &subscription.MaxWordCount,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(eventsJSON, &subscription.Events)
		json.Unmarshal(filterRatingJSON, &subscription.FilterRating)
		json.Unmarshal(filterTagsJSON, &subscription.FilterTags)

		subscriptions = append(subscriptions, &subscription)
	}

	return subscriptions, nil
}

func (r *SubscriptionRepositoryImpl) FindByUserAndTarget(ctx context.Context, userID, targetID uuid.UUID, targetType models.SubscriptionType) (*models.Subscription, error) {
	query := `
		SELECT id, user_id, target_type, target_id, events, is_active, created_at, updated_at,
		       filter_completed, filter_rating, filter_tags, min_word_count, max_word_count
		FROM content_subscriptions WHERE user_id = $1 AND target_id = $2 AND target_type = $3
	`
	var subscription models.Subscription
	var eventsJSON, filterRatingJSON, filterTagsJSON []byte

	err := r.db.QueryRowContext(ctx, query, userID, targetID, targetType).Scan(
		&subscription.ID, &subscription.UserID, &subscription.Type, &subscription.TargetID,
		&eventsJSON, &subscription.IsActive, &subscription.CreatedAt, &subscription.UpdatedAt,
		&subscription.FilterCompleted, &filterRatingJSON, &filterTagsJSON,
		&subscription.MinWordCount, &subscription.MaxWordCount,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(eventsJSON, &subscription.Events)
	json.Unmarshal(filterRatingJSON, &subscription.FilterRating)
	json.Unmarshal(filterTagsJSON, &subscription.FilterTags)

	return &subscription, nil
}

// NotificationRepositoryImpl implements the NotificationRepository interface
type NotificationRepositoryImpl struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) notifications.NotificationRepository {
	return &NotificationRepositoryImpl{db: db}
}

func (r *NotificationRepositoryImpl) CreateNotification(ctx context.Context, notification *models.NotificationItem) error {
	extraDataJSON, _ := json.Marshal(notification.ExtraData)

	query := `
		INSERT INTO notification_items 
		(id, user_id, event, priority, source_id, source_type, title, description, action_url,
		 actor_id, actor_name, extra_data, is_read, is_delivered, created_at, read_at, delivered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	_, err := r.db.ExecContext(ctx, query,
		notification.ID, notification.UserID, notification.Event, notification.Priority,
		notification.SourceID, notification.SourceType, notification.Title, notification.Description,
		notification.ActionURL, notification.ActorID, notification.ActorName, extraDataJSON,
		notification.IsRead, notification.IsDelivered, notification.CreatedAt,
		notification.ReadAt, notification.DeliveredAt,
	)
	return err
}

func (r *NotificationRepositoryImpl) GetNotification(ctx context.Context, id uuid.UUID) (*models.NotificationItem, error) {
	query := `
		SELECT id, user_id, event, priority, source_id, source_type, title, description, action_url,
		       actor_id, actor_name, extra_data, is_read, is_delivered, created_at, read_at, delivered_at
		FROM notification_items WHERE id = $1
	`
	var notification models.NotificationItem
	var extraDataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&notification.ID, &notification.UserID, &notification.Event, &notification.Priority,
		&notification.SourceID, &notification.SourceType, &notification.Title, &notification.Description,
		&notification.ActionURL, &notification.ActorID, &notification.ActorName, &extraDataJSON,
		&notification.IsRead, &notification.IsDelivered, &notification.CreatedAt,
		&notification.ReadAt, &notification.DeliveredAt,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(extraDataJSON, &notification.ExtraData)

	return &notification, nil
}

func (r *NotificationRepositoryImpl) UpdateNotification(ctx context.Context, notification *models.NotificationItem) error {
	extraDataJSON, _ := json.Marshal(notification.ExtraData)

	query := `
		UPDATE notification_items 
		SET is_read = $1, is_delivered = $2, read_at = $3, delivered_at = $4, extra_data = $5
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query,
		notification.IsRead, notification.IsDelivered, notification.ReadAt, notification.DeliveredAt,
		extraDataJSON, notification.ID,
	)
	return err
}

func (r *NotificationRepositoryImpl) DeleteNotification(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notification_items WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *NotificationRepositoryImpl) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.NotificationItem, error) {
	query := `
		SELECT id, user_id, event, priority, source_id, source_type, title, message, action_url,
		       actor_id, actor_name, extra_data, is_read, is_delivered, created_at, read_at, delivered_at
		FROM notification_items WHERE user_id = $1 
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*models.NotificationItem
	for rows.Next() {
		var notification models.NotificationItem
		var extraDataJSON []byte
		var message string // Temporary variable for the message column

		err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.Event, &notification.Priority,
			&notification.SourceID, &notification.SourceType, &notification.Title, &message,
			&notification.ActionURL, &notification.ActorID, &notification.ActorName, &extraDataJSON,
			&notification.IsRead, &notification.IsDelivered, &notification.CreatedAt,
			&notification.ReadAt, &notification.DeliveredAt,
		)
		if err != nil {
			return nil, err
		}

		notification.Description = message // Map message to Description
		json.Unmarshal(extraDataJSON, &notification.ExtraData)
		notifications = append(notifications, &notification)
	}

	return notifications, nil
}

func (r *NotificationRepositoryImpl) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM notification_items WHERE user_id = $1 AND is_read = false`
	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

func (r *NotificationRepositoryImpl) GetNotificationsForBatch(ctx context.Context, userID uuid.UUID, frequency models.NotificationFrequency) ([]*models.NotificationItem, error) {
	// Get undelivered notifications for batching
	query := `
		SELECT id, user_id, event, priority, source_id, source_type, title, description, action_url,
		       actor_id, actor_name, extra_data, is_read, is_delivered, created_at, read_at, delivered_at
		FROM notification_items 
		WHERE user_id = $1 AND is_delivered = false
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*models.NotificationItem
	for rows.Next() {
		var notification models.NotificationItem
		var extraDataJSON []byte

		err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.Event, &notification.Priority,
			&notification.SourceID, &notification.SourceType, &notification.Title, &notification.Description,
			&notification.ActionURL, &notification.ActorID, &notification.ActorName, &extraDataJSON,
			&notification.IsRead, &notification.IsDelivered, &notification.CreatedAt,
			&notification.ReadAt, &notification.DeliveredAt,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(extraDataJSON, &notification.ExtraData)
		notifications = append(notifications, &notification)
	}

	return notifications, nil
}

// DigestRepositoryImpl implements the DigestRepository interface
type DigestRepositoryImpl struct {
	db *sql.DB
}

func NewDigestRepository(db *sql.DB) notifications.DigestRepository {
	return &DigestRepositoryImpl{db: db}
}

func (r *DigestRepositoryImpl) CreateDigest(ctx context.Context, digest *models.NotificationDigest) error {
	notificationsJSON, _ := json.Marshal(digest.Notifications)

	query := `
		INSERT INTO notification_digests 
		(id, user_id, digest_type, notifications, created_at, sent_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		digest.ID, digest.UserID, digest.DigestType, notificationsJSON,
		digest.CreatedAt, digest.SentAt, digest.Status,
	)
	return err
}

func (r *DigestRepositoryImpl) GetDigest(ctx context.Context, id uuid.UUID) (*models.NotificationDigest, error) {
	query := `
		SELECT id, user_id, digest_type, notifications, created_at, sent_at, status
		FROM notification_digests WHERE id = $1
	`
	var digest models.NotificationDigest
	var notificationsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&digest.ID, &digest.UserID, &digest.DigestType, &notificationsJSON,
		&digest.CreatedAt, &digest.SentAt, &digest.Status,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(notificationsJSON, &digest.Notifications)

	return &digest, nil
}

func (r *DigestRepositoryImpl) UpdateDigest(ctx context.Context, digest *models.NotificationDigest) error {
	notificationsJSON, _ := json.Marshal(digest.Notifications)

	query := `
		UPDATE notification_digests 
		SET sent_at = $1, status = $2, notifications = $3
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query,
		digest.SentAt, digest.Status, notificationsJSON, digest.ID,
	)
	return err
}

func (r *DigestRepositoryImpl) GetPendingDigests(ctx context.Context, digestType string) ([]*models.NotificationDigest, error) {
	query := `
		SELECT id, user_id, digest_type, notifications, created_at, sent_at, status
		FROM notification_digests 
		WHERE digest_type = $1 AND status = 'pending'
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, digestType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var digests []*models.NotificationDigest
	for rows.Next() {
		var digest models.NotificationDigest
		var notificationsJSON []byte

		err := rows.Scan(
			&digest.ID, &digest.UserID, &digest.DigestType, &notificationsJSON,
			&digest.CreatedAt, &digest.SentAt, &digest.Status,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(notificationsJSON, &digest.Notifications)
		digests = append(digests, &digest)
	}

	return digests, nil
}

// PreferenceRepositoryImpl implements the PreferenceRepository interface
type PreferenceRepositoryImpl struct {
	db *sql.DB
}

func NewPreferenceRepository(db *sql.DB) notifications.PreferenceRepository {
	return &PreferenceRepositoryImpl{db: db}
}

func (r *PreferenceRepositoryImpl) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
	query := `
		SELECT user_id, email_enabled, web_enabled, push_enabled, quiet_hours_start, quiet_hours_end, timezone,
		       event_preferences, enable_batching, batch_frequency, max_notifications_per_hour, 
		       min_time_between_similar, created_at, updated_at
		FROM user_notification_preferences WHERE user_id = $1
	`
	var preferences models.NotificationPreferences
	var eventPreferencesJSON []byte
	var minTimeBetweenSimilarNs int64

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&preferences.UserID, &preferences.EmailEnabled, &preferences.WebEnabled, &preferences.PushEnabled,
		&preferences.QuietHoursStart, &preferences.QuietHoursEnd, &preferences.Timezone, &eventPreferencesJSON,
		&preferences.EnableBatching, &preferences.BatchFrequency, &preferences.MaxNotificationsPerHour,
		&minTimeBetweenSimilarNs, &preferences.CreatedAt, &preferences.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default preferences
			defaultPrefs := models.DefaultNotificationPreferences(userID)
			return &defaultPrefs, nil
		}
		return nil, err
	}

	preferences.MinTimeBetweenSimilar = time.Duration(minTimeBetweenSimilarNs)
	json.Unmarshal(eventPreferencesJSON, &preferences.EventPreferences)

	return &preferences, nil
}

func (r *PreferenceRepositoryImpl) UpdatePreferences(ctx context.Context, preferences *models.NotificationPreferences) error {
	eventPreferencesJSON, _ := json.Marshal(preferences.EventPreferences)
	minTimeBetweenSimilarNs := int64(preferences.MinTimeBetweenSimilar)

	query := `
		UPDATE user_notification_preferences 
		SET email_enabled = $1, web_enabled = $2, push_enabled = $3, quiet_hours_start = $4, 
		    quiet_hours_end = $5, timezone = $6, event_preferences = $7, enable_batching = $8,
		    batch_frequency = $9, max_notifications_per_hour = $10, min_time_between_similar = $11,
		    updated_at = $12
		WHERE user_id = $13
	`
	_, err := r.db.ExecContext(ctx, query,
		preferences.EmailEnabled, preferences.WebEnabled, preferences.PushEnabled,
		preferences.QuietHoursStart, preferences.QuietHoursEnd, preferences.Timezone,
		eventPreferencesJSON, preferences.EnableBatching, preferences.BatchFrequency,
		preferences.MaxNotificationsPerHour, minTimeBetweenSimilarNs, time.Now(), preferences.UserID,
	)
	return err
}

func (r *PreferenceRepositoryImpl) CreatePreferences(ctx context.Context, preferences *models.NotificationPreferences) error {
	eventPreferencesJSON, _ := json.Marshal(preferences.EventPreferences)
	minTimeBetweenSimilarNs := int64(preferences.MinTimeBetweenSimilar)

	query := `
		INSERT INTO user_notification_preferences 
		(user_id, email_enabled, web_enabled, push_enabled, quiet_hours_start, quiet_hours_end, 
		 timezone, event_preferences, enable_batching, batch_frequency, max_notifications_per_hour,
		 min_time_between_similar, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (user_id) DO UPDATE SET
		email_enabled = EXCLUDED.email_enabled,
		web_enabled = EXCLUDED.web_enabled,
		push_enabled = EXCLUDED.push_enabled,
		quiet_hours_start = EXCLUDED.quiet_hours_start,
		quiet_hours_end = EXCLUDED.quiet_hours_end,
		timezone = EXCLUDED.timezone,
		event_preferences = EXCLUDED.event_preferences,
		enable_batching = EXCLUDED.enable_batching,
		batch_frequency = EXCLUDED.batch_frequency,
		max_notifications_per_hour = EXCLUDED.max_notifications_per_hour,
		min_time_between_similar = EXCLUDED.min_time_between_similar,
		updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		preferences.UserID, preferences.EmailEnabled, preferences.WebEnabled, preferences.PushEnabled,
		preferences.QuietHoursStart, preferences.QuietHoursEnd, preferences.Timezone, eventPreferencesJSON,
		preferences.EnableBatching, preferences.BatchFrequency, preferences.MaxNotificationsPerHour,
		minTimeBetweenSimilarNs, preferences.CreatedAt, preferences.UpdatedAt,
	)
	return err
}
