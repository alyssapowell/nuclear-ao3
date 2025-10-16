package notifications

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/google/uuid"
	"nuclear-ao3/shared/models"
)

// BatchProcessor handles batching and digest creation
type BatchProcessor struct {
	service         *NotificationService
	intervalMinutes int
	maxBatchSize    int
	ticker          *time.Ticker
	stopChan        chan bool
	pendingBatches  map[string][]*models.NotificationItem // userID -> notifications
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(service *NotificationService, intervalMinutes, maxBatchSize int) *BatchProcessor {
	bp := &BatchProcessor{
		service:         service,
		intervalMinutes: intervalMinutes,
		maxBatchSize:    maxBatchSize,
		stopChan:        make(chan bool),
		pendingBatches:  make(map[string][]*models.NotificationItem),
	}

	// Start the batch processing ticker
	bp.start()

	return bp
}

// start begins the batch processing routine
func (bp *BatchProcessor) start() {
	bp.ticker = time.NewTicker(time.Duration(bp.intervalMinutes) * time.Minute)

	go func() {
		for {
			select {
			case <-bp.ticker.C:
				bp.processPendingBatches()
			case <-bp.stopChan:
				bp.ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the batch processor
func (bp *BatchProcessor) Stop() {
	close(bp.stopChan)
}

// AddToBatch adds a notification to the pending batch for a user
func (bp *BatchProcessor) AddToBatch(ctx context.Context, notification *models.NotificationItem) error {
	userID := notification.UserID.String()

	// Add to pending batch
	bp.pendingBatches[userID] = append(bp.pendingBatches[userID], notification)

	// Check if batch is full and should be sent immediately
	if len(bp.pendingBatches[userID]) >= bp.maxBatchSize {
		return bp.processBatchForUser(ctx, userID)
	}

	return nil
}

// processPendingBatches processes all pending batches
func (bp *BatchProcessor) processPendingBatches() {
	ctx := context.Background()

	for userID := range bp.pendingBatches {
		if len(bp.pendingBatches[userID]) > 0 {
			if err := bp.processBatchForUser(ctx, userID); err != nil {
				log.Printf("Failed to process batch for user %s: %v", userID, err)
			}
		}
	}
}

// processBatchForUser processes the pending batch for a specific user
func (bp *BatchProcessor) processBatchForUser(ctx context.Context, userID string) error {
	notifications := bp.pendingBatches[userID]
	if len(notifications) == 0 {
		return nil
	}

	// Clear the pending batch
	bp.pendingBatches[userID] = nil

	// Get user preferences to determine digest type
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	prefs, err := bp.service.preferenceRepo.GetPreferences(ctx, uid)
	if err != nil {
		log.Printf("Failed to get preferences for user %s, using defaults: %v", userID, err)
		defaultPrefs := models.DefaultNotificationPreferences(uid)
		prefs = &defaultPrefs
	}

	// Create digest based on user's batch frequency
	digestType := string(prefs.BatchFrequency)

	// Group notifications by type and importance
	groupedNotifications := bp.groupNotifications(notifications)

	// Convert notification pointers to values for digest
	notificationValues := make([]models.NotificationItem, len(notifications))
	for i, notification := range notifications {
		notificationValues[i] = *notification
	}

	// Create digest
	digest := &models.NotificationDigest{
		ID:            uuid.New(),
		UserID:        uid,
		DigestType:    digestType,
		Notifications: notificationValues,
		CreatedAt:     time.Now(),
		Status:        models.DigestPending,
	}

	// Save digest
	if err := bp.service.digestRepo.CreateDigest(ctx, digest); err != nil {
		return fmt.Errorf("failed to create digest: %w", err)
	}

	// Send digest email
	return bp.sendDigestEmail(ctx, digest, groupedNotifications, prefs)
}

// groupNotifications groups notifications by type and priority for better digest formatting
func (bp *BatchProcessor) groupNotifications(notifications []*models.NotificationItem) map[string][]*models.NotificationItem {
	groups := make(map[string][]*models.NotificationItem)

	// Sort notifications by priority and time
	sort.Slice(notifications, func(i, j int) bool {
		// Sort by priority first (high > medium > low)
		if notifications[i].Priority != notifications[j].Priority {
			priorityOrder := map[models.NotificationPriority]int{
				models.PriorityHigh:   3,
				models.PriorityMedium: 2,
				models.PriorityLow:    1,
			}
			return priorityOrder[notifications[i].Priority] > priorityOrder[notifications[j].Priority]
		}
		// Then by time (newest first)
		return notifications[i].CreatedAt.After(notifications[j].CreatedAt)
	})

	// Group by event type
	for _, notification := range notifications {
		eventType := string(notification.Event)
		groups[eventType] = append(groups[eventType], notification)
	}

	return groups
}

// sendDigestEmail sends a digest email to the user
func (bp *BatchProcessor) sendDigestEmail(ctx context.Context, digest *models.NotificationDigest, groupedNotifications map[string][]*models.NotificationItem, prefs *models.NotificationPreferences) error {
	// Generate digest content
	subject := bp.generateDigestSubject(digest, groupedNotifications)
	plainText := bp.generateDigestPlainText(digest, groupedNotifications)
	html := bp.generateDigestHTML(digest, groupedNotifications)

	// Create message content
	content := &models.MessageContent{
		Subject:   subject,
		PlainText: plainText,
		HTML:      html,
		Variables: map[string]interface{}{
			"digest_type":        digest.DigestType,
			"notification_count": len(digest.Notifications),
			"user_id":            digest.UserID.String(),
			"digest_id":          digest.ID.String(),
		},
	}

	// Determine which channels to use for digest
	digestChannels := []models.DeliveryChannel{models.ChannelEmail}
	if prefs.WebEnabled {
		digestChannels = append(digestChannels, models.ChannelInApp)
	}

	// Create channel configs for digest channels
	channelConfigs := make(map[models.DeliveryChannel]models.ChannelConfig)
	for _, channel := range digestChannels {
		channelConfigs[channel] = models.ChannelConfig{
			Enabled: true,
		}
	}

	// Create message
	message := &models.Message{
		Type:    models.MessageSystemAlert, // Use system alert for digests
		Content: *content,
		Recipients: []models.Recipient{
			{
				UserID:   digest.UserID,
				Channels: digestChannels,
				Preferences: models.UserNotificationSettings{
					UserID:        digest.UserID,
					GlobalEnabled: true,
					Channels:      channelConfigs,
					UpdatedAt:     time.Now(),
				},
			},
		},
	}

	// Send message
	if err := bp.service.messageService.SendMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to send digest message: %w", err)
	}

	// Update digest as sent
	digest.Status = models.DigestSent
	now := time.Now()
	digest.SentAt = &now

	if err := bp.service.digestRepo.UpdateDigest(ctx, digest); err != nil {
		log.Printf("Failed to update digest status: %v", err)
	}

	// Mark all notifications in digest as delivered
	for i := range digest.Notifications {
		digest.Notifications[i].IsDelivered = true
		digest.Notifications[i].DeliveredAt = &now
		digest.Notifications[i].DigestID = &digest.ID

		if err := bp.service.notificationRepo.UpdateNotification(ctx, &digest.Notifications[i]); err != nil {
			log.Printf("Failed to update notification %s: %v", digest.Notifications[i].ID, err)
		}
	}

	return nil
}

// generateDigestSubject creates a subject line for the digest
func (bp *BatchProcessor) generateDigestSubject(digest *models.NotificationDigest, groups map[string][]*models.NotificationItem) string {
	count := len(digest.Notifications)

	if count == 1 {
		return fmt.Sprintf("[Nuclear AO3] 1 new notification")
	}

	return fmt.Sprintf("[Nuclear AO3] %d new notifications", count)
}

// generateDigestPlainText creates plain text content for the digest
func (bp *BatchProcessor) generateDigestPlainText(digest *models.NotificationDigest, groups map[string][]*models.NotificationItem) string {
	var content string

	content += fmt.Sprintf("You have %d new notifications:\n\n", len(digest.Notifications))

	// Add content for each group
	for eventType, notifications := range groups {
		content += fmt.Sprintf("%s (%d):\n", bp.getEventDisplayName(eventType), len(notifications))

		for _, notification := range notifications {
			content += fmt.Sprintf("  • %s\n", notification.Title)
			if notification.ActionURL != "" {
				content += fmt.Sprintf("    %s\n", notification.ActionURL)
			}
		}
		content += "\n"
	}

	content += "---\n"
	content += "To manage your notification preferences, visit your account settings.\n"
	content += "To unsubscribe from digest emails, change your batch frequency to 'never'.\n"

	return content
}

// generateDigestHTML creates HTML content for the digest
func (bp *BatchProcessor) generateDigestHTML(digest *models.NotificationDigest, groups map[string][]*models.NotificationItem) string {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Nuclear AO3 Notifications</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #990000; color: white; padding: 20px; border-radius: 5px 5px 0 0; }
        .content { background: #f9f9f9; padding: 20px; border-radius: 0 0 5px 5px; }
        .notification-group { margin-bottom: 25px; }
        .group-title { font-size: 18px; font-weight: bold; color: #990000; margin-bottom: 10px; border-bottom: 2px solid #990000; padding-bottom: 5px; }
        .notification-item { background: white; padding: 15px; margin-bottom: 10px; border-radius: 3px; border-left: 4px solid #990000; }
        .notification-title { font-weight: bold; margin-bottom: 5px; }
        .notification-desc { color: #666; margin-bottom: 8px; }
        .notification-action { margin-top: 10px; }
        .action-button { background: #990000; color: white; padding: 8px 15px; text-decoration: none; border-radius: 3px; display: inline-block; }
        .footer { text-align: center; margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 12px; color: #666; }
    </style>
</head>
<body>`

	html += fmt.Sprintf(`
    <div class="header">
        <h1>Nuclear AO3 Notifications</h1>
        <p>You have %d new notifications</p>
    </div>
    <div class="content">`, len(digest.Notifications))

	// Add content for each group
	for eventType, notifications := range groups {
		html += fmt.Sprintf(`
        <div class="notification-group">
            <div class="group-title">%s (%d)</div>`, bp.getEventDisplayName(eventType), len(notifications))

		for _, notification := range notifications {
			html += fmt.Sprintf(`
            <div class="notification-item">
                <div class="notification-title">%s</div>`, notification.Title)

			if notification.Description != "" {
				html += fmt.Sprintf(`
                <div class="notification-desc">%s</div>`, notification.Description)
			}

			if notification.ActionURL != "" {
				html += fmt.Sprintf(`
                <div class="notification-action">
                    <a href="%s" class="action-button">View</a>
                </div>`, notification.ActionURL)
			}

			html += `
            </div>`
		}

		html += `
        </div>`
	}

	html += `
    </div>
    <div class="footer">
        <p>To manage your notification preferences, visit your account settings.<br>
        To unsubscribe from digest emails, change your batch frequency to 'never'.</p>
    </div>
</body>
</html>`

	return html
}

// getEventDisplayName returns a user-friendly name for an event type
func (bp *BatchProcessor) getEventDisplayName(eventType string) string {
	switch eventType {
	case string(models.EventWorkUpdated):
		return "📖 Work Updates"
	case string(models.EventCommentReceived):
		return "💬 New Comments"
	case string(models.EventKudosReceived):
		return "❤️ Kudos"
	case string(models.EventNewWork):
		return "✨ New Works"
	case string(models.EventSeriesUpdated):
		return "📚 Series Updates"
	case string(models.EventCollectionInvite):
		return "📥 Collection Invites"
	case string(models.EventSystemAlert):
		return "⚠️ System Alerts"
	default:
		return "📢 Notifications"
	}
}
