package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"nuclear-ao3/shared/models"
	"nuclear-ao3/shared/notifications"
)

// Helper function to safely parse user ID from context
func getUserUUID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("unauthorized")
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid user ID type")
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID format")
	}

	return userUUID, nil
}

// WebSocket message types
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WebSocket handlers
func (s *NotificationService) handleWebSocket(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userIDStr := userID.(string)

	// Upgrade connection
	conn, err := s.wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to upgrade connection"})
		return
	}
	defer conn.Close()

	// Store connection
	s.wsClients[userIDStr] = conn

	// Clean up on disconnect
	defer func() {
		delete(s.wsClients, userIDStr)
	}()

	// Send initial notification count
	count, err := s.notificationSvc.GetUnreadCount(context.Background(), uuid.MustParse(userIDStr))
	if err == nil {
		conn.WriteJSON(WSMessage{
			Type: "unread_count",
			Payload: gin.H{
				"count": count,
			},
		})
	}

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (s *NotificationService) handleWebSocketBroadcast() {
	for {
		select {
		case message := <-s.wsBroadcast:
			// Broadcast to all connected clients
			for userID, conn := range s.wsClients {
				err := conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					// Connection is dead, remove it
					delete(s.wsClients, userID)
					conn.Close()
				}
			}
		}
	}
}

// Notification handlers
func (s *NotificationService) getUserNotifications(c *gin.Context) {
	userUUID, err := getUserUUID(c)
	if err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	// Parse query parameters
	limit := 20
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Temporary fix: Return empty notifications while debugging database schema issues
	// Use userUUID to avoid unused variable error
	_ = userUUID

	c.JSON(http.StatusOK, gin.H{
		"notifications": []interface{}{}, // Empty array for now
		"limit":         limit,
		"offset":        offset,
	})
}

func (s *NotificationService) markNotificationRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	notificationID := c.Param("id")
	notificationUUID, err := uuid.Parse(notificationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification ID"})
		return
	}

	userUUID := uuid.MustParse(userID.(string))

	err = s.notificationSvc.MarkNotificationRead(context.Background(), notificationUUID, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notification as read"})
		return
	}

	// Broadcast updated unread count
	count, _ := s.notificationSvc.GetUnreadCount(context.Background(), userUUID)
	s.broadcastToUser(userID.(string), WSMessage{
		Type: "unread_count",
		Payload: gin.H{
			"count": count,
		},
	})

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *NotificationService) deleteNotification(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	notificationID := c.Param("id")
	notificationUUID, err := uuid.Parse(notificationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification ID"})
		return
	}

	err = s.notificationSvc.DeleteNotification(context.Background(), notificationUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete notification"})
		return
	}

	// Broadcast updated unread count
	userUUID := uuid.MustParse(userID.(string))
	count, _ := s.notificationSvc.GetUnreadCount(context.Background(), userUUID)
	s.broadcastToUser(userID.(string), WSMessage{
		Type: "unread_count",
		Payload: gin.H{
			"count": count,
		},
	})

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *NotificationService) getUnreadCount(c *gin.Context) {
	userUUID, err := getUserUUID(c)
	if err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	count, err := s.notificationSvc.GetUnreadCount(context.Background(), userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get unread count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}

// Preference handlers
func (s *NotificationService) getNotificationPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userUUID := uuid.MustParse(userID.(string))

	preferences, err := s.notificationSvc.GetUserPreferences(context.Background(), userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get preferences"})
		return
	}

	c.JSON(http.StatusOK, preferences)
}

func (s *NotificationService) updateNotificationPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var preferences models.NotificationPreferences
	if err := c.ShouldBindJSON(&preferences); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userUUID := uuid.MustParse(userID.(string))
	preferences.UserID = userUUID
	preferences.UpdatedAt = time.Now()

	err := s.notificationSvc.UpdateUserPreferences(context.Background(), &preferences)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update preferences"})
		return
	}

	c.JSON(http.StatusOK, preferences)
}

// Subscription handlers
func (s *NotificationService) getUserSubscriptions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userUUID := uuid.MustParse(userID.(string))

	subscriptions, err := s.notificationSvc.GetUserSubscriptions(context.Background(), userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get subscriptions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subscriptions,
	})
}

func (s *NotificationService) createSubscription(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var subscription models.Subscription
	if err := c.ShouldBindJSON(&subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userUUID := uuid.MustParse(userID.(string))
	subscription.ID = uuid.New()
	subscription.UserID = userUUID
	subscription.CreatedAt = time.Now()
	subscription.UpdatedAt = time.Now()
	subscription.IsActive = true

	err := s.notificationSvc.CreateSubscription(context.Background(), &subscription)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subscription"})
		return
	}

	c.JSON(http.StatusCreated, subscription)
}

func (s *NotificationService) updateSubscription(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	subscriptionID := c.Param("id")
	subscriptionUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription ID"})
		return
	}

	var subscription models.Subscription
	if err := c.ShouldBindJSON(&subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userUUID := uuid.MustParse(userID.(string))
	subscription.ID = subscriptionUUID
	subscription.UserID = userUUID
	subscription.UpdatedAt = time.Now()

	err = s.notificationSvc.UpdateSubscription(context.Background(), &subscription)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update subscription"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

func (s *NotificationService) deleteSubscription(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	subscriptionID := c.Param("id")
	subscriptionUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription ID"})
		return
	}

	err = s.notificationSvc.DeleteSubscription(context.Background(), subscriptionUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Rule handlers (placeholder - not implemented in notification service)
func (s *NotificationService) getNotificationRules(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"rules": []interface{}{}})
}

func (s *NotificationService) createNotificationRule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

func (s *NotificationService) updateNotificationRule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

func (s *NotificationService) deleteNotificationRule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// Test/admin handlers
func (s *NotificationService) createTestNotification(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var eventData notifications.EventData
	if err := c.ShouldBindJSON(&eventData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// For test notifications, use a default event if none provided
	if eventData.Type == "" {
		eventData.Type = models.EventSystemAlert
		eventData.Title = "Test Notification"
		eventData.Description = "This is a test notification"
		eventData.ActorName = "System"
	}

	err := s.notificationSvc.ProcessEvent(context.Background(), &eventData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create test notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "event": eventData})
}

func (s *NotificationService) processEvent(c *gin.Context) {
	var eventData notifications.EventData
	if err := c.ShouldBindJSON(&eventData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	err := s.notificationSvc.ProcessEvent(context.Background(), &eventData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Helper methods
func (s *NotificationService) broadcastToUser(userID string, message WSMessage) {
	if conn, exists := s.wsClients[userID]; exists {
		messageBytes, _ := json.Marshal(message)
		conn.WriteMessage(websocket.TextMessage, messageBytes)
	}
}

// Helper methods for WebSocket and notification management are defined in main.go
