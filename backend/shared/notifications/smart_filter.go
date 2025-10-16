package notifications

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"nuclear-ao3/shared/models"
)

// SmartFilter provides intelligent notification filtering
type SmartFilter struct {
	recentWindow time.Duration
}

// NewSmartFilter creates a new smart filter
func NewSmartFilter() *SmartFilter {
	return &SmartFilter{
		recentWindow: 24 * time.Hour, // Look at last 24 hours for patterns
	}
}

// ShouldNotify determines if a notification should be sent based on smart filtering rules
func (sf *SmartFilter) ShouldNotify(ctx context.Context, prefs *models.NotificationPreferences, notification *models.NotificationItem) (bool, *models.NotificationItem) {
	// Check quiet hours
	if sf.isInQuietHours(prefs, time.Now()) {
		log.Printf("Notification blocked: quiet hours for user %s", notification.UserID)
		return false, nil
	}

	// Check rate limiting
	if sf.isRateLimited(prefs, notification) {
		log.Printf("Notification blocked: rate limited for user %s", notification.UserID)
		return false, nil
	}

	// Apply smart duplicate detection
	if sf.isDuplicate(notification) {
		log.Printf("Notification blocked: duplicate detected for user %s", notification.UserID)
		return false, nil
	}

	// Enhance notification with smart content
	enhanced := sf.enhanceNotification(notification)

	return true, enhanced
}

// isInQuietHours checks if the current time is within user's quiet hours
func (sf *SmartFilter) isInQuietHours(prefs *models.NotificationPreferences, now time.Time) bool {
	if prefs.QuietHoursStart == nil || prefs.QuietHoursEnd == nil {
		return false
	}

	// Convert to user's timezone
	location, err := time.LoadLocation(prefs.Timezone)
	if err != nil {
		location = time.UTC
	}

	userTime := now.In(location)
	startTime := prefs.QuietHoursStart.In(location)
	endTime := prefs.QuietHoursEnd.In(location)

	// Handle overnight quiet hours (e.g., 22:00 to 08:00)
	if startTime.After(endTime) {
		return userTime.After(startTime) || userTime.Before(endTime)
	}

	// Normal quiet hours (e.g., 01:00 to 06:00)
	return userTime.After(startTime) && userTime.Before(endTime)
}

// isRateLimited checks if the user has exceeded their notification rate limits
func (sf *SmartFilter) isRateLimited(prefs *models.NotificationPreferences, notification *models.NotificationItem) bool {
	// This would typically check against a cache or database of recent notifications
	// For now, we'll implement a simple check

	if prefs.MaxNotificationsPerHour <= 0 {
		return false // No rate limiting
	}

	// In a real implementation, this would query recent notifications from the database
	// and count how many were sent in the last hour

	return false // Placeholder - implement with actual notification counting
}

// isDuplicate checks if this notification is a duplicate of recent ones
func (sf *SmartFilter) isDuplicate(notification *models.NotificationItem) bool {
	// This would typically check recent notifications for similar content
	// For now, we'll implement a simple check based on source and event type

	// In a real implementation, this would:
	// 1. Query recent notifications for the same user
	// 2. Compare source ID, event type, and content similarity
	// 3. Apply deduplication rules based on time window

	return false // Placeholder - implement with actual duplicate detection
}

// enhanceNotification adds smart content enhancements to notifications
func (sf *SmartFilter) enhanceNotification(notification *models.NotificationItem) *models.NotificationItem {
	enhanced := *notification // Copy the notification

	// Add smart subject line enhancements
	enhanced.Title = sf.enhanceTitle(notification)

	// Add contextual information
	enhanced.Description = sf.enhanceDescription(notification)

	return &enhanced
}

// enhanceTitle improves notification titles with smart content
func (sf *SmartFilter) enhanceTitle(notification *models.NotificationItem) string {
	switch notification.Event {
	case models.EventWorkUpdated:
		if workTitle, ok := notification.ExtraData["work_title"].(string); ok {
			return fmt.Sprintf("📖 New chapter: %s", workTitle)
		}
	case models.EventCommentReceived:
		if workTitle, ok := notification.ExtraData["work_title"].(string); ok {
			return fmt.Sprintf("💬 New comment on %s", workTitle)
		}
	case models.EventKudosReceived:
		if workTitle, ok := notification.ExtraData["work_title"].(string); ok {
			return fmt.Sprintf("❤️ Kudos on %s", workTitle)
		}
	}

	return notification.Title
}

// enhanceDescription improves notification descriptions with smart content
func (sf *SmartFilter) enhanceDescription(notification *models.NotificationItem) string {
	switch notification.Event {
	case models.EventWorkUpdated:
		// Add chapter information if available
		if chapterTitle, ok := notification.ExtraData["chapter_title"].(string); ok {
			if authorName, ok := notification.ExtraData["author_name"].(string); ok {
				return fmt.Sprintf("%s posted a new chapter: \"%s\"", authorName, chapterTitle)
			}
		}
	case models.EventCommentReceived:
		// Add comment preview if available
		if commentPreview, ok := notification.ExtraData["comment_preview"].(string); ok {
			if len(commentPreview) > 100 {
				commentPreview = commentPreview[:97] + "..."
			}
			return fmt.Sprintf("%s: \"%s\"", notification.ActorName, commentPreview)
		}
	}

	return notification.Description
}

// RuleAction represents the result of rule evaluation
type RuleAction struct {
	Action               models.RuleAction
	ModifiedNotification *models.NotificationItem
	DelayMinutes         int
	Reason               string
}

// RuleEngine evaluates user-defined notification rules
type RuleEngine struct {
	cache map[string][]*models.NotificationRule // Simple cache for rules
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		cache: make(map[string][]*models.NotificationRule),
	}
}

// EvaluateNotification evaluates a notification against user rules
func (re *RuleEngine) EvaluateNotification(ctx context.Context, prefs *models.NotificationPreferences, notification *models.NotificationItem) RuleAction {
	// Get user rules (in a real implementation, this would fetch from database)
	rules := re.getUserRules(notification.UserID.String())

	// Evaluate rules in priority order
	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		if re.ruleMatches(rule, notification) {
			return RuleAction{
				Action:               rule.Action,
				ModifiedNotification: re.applyRuleModifications(rule, notification),
				DelayMinutes:         *rule.DelayMinutes,
				Reason:               fmt.Sprintf("Matched rule: %s", rule.Name),
			}
		}
	}

	// No rules matched, allow notification
	return RuleAction{
		Action: models.ActionAllow,
		Reason: "No matching rules",
	}
}

// getUserRules retrieves rules for a user (placeholder implementation)
func (re *RuleEngine) getUserRules(userID string) []*models.NotificationRule {
	// In a real implementation, this would fetch from database
	// For now, return empty slice
	return []*models.NotificationRule{}
}

// ruleMatches checks if a rule matches a notification
func (re *RuleEngine) ruleMatches(rule *models.NotificationRule, notification *models.NotificationItem) bool {
	// Check event type
	eventMatches := false
	for _, event := range rule.Events {
		if event == notification.Event {
			eventMatches = true
			break
		}
	}
	if !eventMatches {
		return false
	}

	// Check source type if specified
	if len(rule.SourceTypes) > 0 {
		sourceMatches := false
		for _, sourceType := range rule.SourceTypes {
			if sourceType == notification.SourceType {
				sourceMatches = true
				break
			}
		}
		if !sourceMatches {
			return false
		}
	}

	// Check actor conditions if specified
	if rule.ActorConditions != nil {
		if !re.checkActorConditions(rule.ActorConditions, notification) {
			return false
		}
	}

	// Check content filters if specified
	if rule.ContentFilters != nil {
		if !re.checkContentFilters(rule.ContentFilters, notification) {
			return false
		}
	}

	// Check time conditions if specified
	if rule.TimeConditions != nil {
		if !re.checkTimeConditions(rule.TimeConditions, notification) {
			return false
		}
	}

	return true
}

// checkActorConditions evaluates actor-based rule conditions
func (re *RuleEngine) checkActorConditions(conditions map[string]interface{}, notification *models.NotificationItem) bool {
	// Example conditions:
	// - "blocked_users": ["user1", "user2"]
	// - "only_followed": true
	// - "min_reputation": 100

	if blockedUsers, ok := conditions["blocked_users"].([]string); ok {
		for _, blockedUser := range blockedUsers {
			if notification.ActorName == blockedUser {
				return false
			}
		}
	}

	// Add more condition checks as needed
	return true
}

// checkContentFilters evaluates content-based rule conditions
func (re *RuleEngine) checkContentFilters(filters map[string]interface{}, notification *models.NotificationItem) bool {
	// Example filters:
	// - "required_tags": ["tag1", "tag2"]
	// - "excluded_tags": ["tag3"]
	// - "min_word_count": 1000
	// - "max_word_count": 50000

	if requiredTags, ok := filters["required_tags"].([]string); ok {
		// Check if notification contains required tags
		if tags, exists := notification.ExtraData["tags"].([]string); exists {
			for _, requiredTag := range requiredTags {
				found := false
				for _, tag := range tags {
					if tag == requiredTag {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		} else {
			return false // Required tags specified but no tags in notification
		}
	}

	// Add more filter checks as needed
	return true
}

// checkTimeConditions evaluates time-based rule conditions
func (re *RuleEngine) checkTimeConditions(conditions map[string]interface{}, notification *models.NotificationItem) bool {
	// Example conditions:
	// - "days_of_week": ["monday", "tuesday"]
	// - "hours_of_day": [9, 10, 11, 12, 13, 14, 15, 16, 17]
	// - "timezone": "America/New_York"

	if daysOfWeek, ok := conditions["days_of_week"].([]string); ok {
		currentDay := time.Now().Weekday().String()
		dayAllowed := false
		for _, allowedDay := range daysOfWeek {
			if strings.EqualFold(allowedDay, currentDay) {
				dayAllowed = true
				break
			}
		}
		if !dayAllowed {
			return false
		}
	}

	// Add more time condition checks as needed
	return true
}

// applyRuleModifications applies rule modifications to a notification
func (re *RuleEngine) applyRuleModifications(rule *models.NotificationRule, notification *models.NotificationItem) *models.NotificationItem {
	modified := *notification // Copy notification

	// Apply priority changes
	if rule.Priority != nil {
		modified.Priority = *rule.Priority
	}

	// Apply other modifications as needed
	return &modified
}
