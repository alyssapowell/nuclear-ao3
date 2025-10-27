package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"nuclear-ao3/shared/models"
)

type NotificationServiceTestSuite struct {
	suite.Suite
	service    *NotificationService
	router     *gin.Engine
	testUserID uuid.UUID
	testWorkID uuid.UUID
}

func (suite *NotificationServiceTestSuite) SetupSuite() {
	// Initialize test user and work IDs
	suite.testUserID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	suite.testWorkID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
}

func (suite *NotificationServiceTestSuite) SetupTest() {
	// Create a mock notification service for testing
	suite.service = &NotificationService{
		notificationSvc: &NotificationServiceExtended{
			subscriptionRepo: &MockSubscriptionRepository{},
			notificationRepo: &MockNotificationRepository{},
			preferenceRepo:   &MockPreferenceRepository{},
		},
		wsClients:   make(map[string]*websocket.Conn),
		wsBroadcast: make(chan []byte, 10),
	}

	// Setup router
	suite.router = gin.New()
	suite.setupRoutes()
}

func (suite *NotificationServiceTestSuite) setupRoutes() {
	// Mock auth middleware
	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", suite.testUserID.String())
		c.Next()
	}

	api := suite.router.Group("/api/v1")
	api.Use(authMiddleware)
	{
		api.GET("/notifications", suite.service.getUserNotifications)
		api.PUT("/notifications/:id/read", suite.service.markNotificationRead)
		api.DELETE("/notifications/:id", suite.service.deleteNotification)
		api.GET("/notifications/unread-count", suite.service.getUnreadCount)
		api.GET("/preferences", suite.service.getNotificationPreferences)
		api.PUT("/preferences", suite.service.updateNotificationPreferences)
		api.GET("/subscriptions", suite.service.getUserSubscriptions)
		api.POST("/subscriptions", suite.service.createSubscription)
		api.POST("/test-notification", suite.service.createTestNotification)
		api.POST("/process-event", suite.service.processEvent)
	}
}

func (suite *NotificationServiceTestSuite) TestGetUserNotifications_Success() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response, "notifications")
}

func (suite *NotificationServiceTestSuite) TestGetUnreadCount_Success() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/unread-count", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response, "count")
}

func (suite *NotificationServiceTestSuite) TestMarkNotificationRead_Success() {
	notificationID := uuid.New()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/notifications/"+notificationID.String()+"/read", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), true, response["success"])
}

func (suite *NotificationServiceTestSuite) TestCreateSubscription_Success() {
	subscription := map[string]interface{}{
		"type":      "work",
		"target_id": suite.testWorkID.String(),
		"events":    []string{"comment_received", "work_updated"},
	}

	jsonData, _ := json.Marshal(subscription)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/subscriptions", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response models.Subscription
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testUserID, response.UserID)
	assert.Equal(suite.T(), suite.testWorkID, response.TargetID)
}

func (suite *NotificationServiceTestSuite) TestProcessEvent_Success() {
	eventData := map[string]interface{}{
		"type":        "comment_received",
		"source_id":   suite.testWorkID.String(),
		"source_type": "work",
		"title":       "Test Comment Notification",
		"description": "Test user commented on your work",
		"actor_id":    suite.testUserID.String(),
		"actor_name":  "Test User",
	}

	jsonData, _ := json.Marshal(eventData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/process-event", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), true, response["success"])
}

func (suite *NotificationServiceTestSuite) TestCreateTestNotification_Success() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/test-notification", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), true, response["success"])
}

func (suite *NotificationServiceTestSuite) TestGetNotificationPreferences_Success() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/preferences", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.NotificationPreferences
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testUserID, response.UserID)
}

func (suite *NotificationServiceTestSuite) TestUpdateNotificationPreferences_Success() {
	preferences := models.DefaultNotificationPreferences(suite.testUserID)
	preferences.EmailEnabled = false

	jsonData, _ := json.Marshal(preferences)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/preferences", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.NotificationPreferences
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), false, response.EmailEnabled)
}

// Mock repositories for testing

type MockSubscriptionRepository struct{}

func (m *MockSubscriptionRepository) CreateSubscription(ctx context.Context, subscription *models.Subscription) error {
	return nil
}

func (m *MockSubscriptionRepository) GetSubscription(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	return &models.Subscription{
		ID:       id,
		UserID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Type:     models.SubscriptionWork,
		TargetID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Events:   []models.NotificationEvent{models.EventCommentReceived},
		IsActive: true,
	}, nil
}

func (m *MockSubscriptionRepository) UpdateSubscription(ctx context.Context, subscription *models.Subscription) error {
	return nil
}

func (m *MockSubscriptionRepository) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockSubscriptionRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]*models.Subscription, error) {
	return []*models.Subscription{
		{
			ID:       uuid.New(),
			UserID:   userID,
			Type:     models.SubscriptionWork,
			TargetID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			Events:   []models.NotificationEvent{models.EventCommentReceived},
			IsActive: true,
		},
	}, nil
}

func (m *MockSubscriptionRepository) FindByTarget(ctx context.Context, targetType models.SubscriptionType, targetID uuid.UUID) ([]*models.Subscription, error) {
	return []*models.Subscription{}, nil
}

func (m *MockSubscriptionRepository) FindByUserAndTarget(ctx context.Context, userID, targetID uuid.UUID, targetType models.SubscriptionType) (*models.Subscription, error) {
	return nil, sql.ErrNoRows
}

type MockNotificationRepository struct{}

func (m *MockNotificationRepository) CreateNotification(ctx context.Context, notification *models.NotificationItem) error {
	return nil
}

func (m *MockNotificationRepository) GetNotification(ctx context.Context, id uuid.UUID) (*models.NotificationItem, error) {
	return &models.NotificationItem{
		ID:       id,
		UserID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Event:    models.EventCommentReceived,
		Priority: models.PriorityMedium,
		Title:    "Test Notification",
		IsRead:   false,
	}, nil
}

func (m *MockNotificationRepository) UpdateNotification(ctx context.Context, notification *models.NotificationItem) error {
	return nil
}

func (m *MockNotificationRepository) DeleteNotification(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockNotificationRepository) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.NotificationItem, error) {
	return []*models.NotificationItem{
		{
			ID:          uuid.New(),
			UserID:      userID,
			Event:       models.EventCommentReceived,
			Priority:    models.PriorityMedium,
			Title:       "Test Notification",
			Description: "This is a test notification",
			IsRead:      false,
			CreatedAt:   time.Now(),
		},
	}, nil
}

func (m *MockNotificationRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return 3, nil
}

func (m *MockNotificationRepository) GetNotificationsForBatch(ctx context.Context, userID uuid.UUID, frequency models.NotificationFrequency) ([]*models.NotificationItem, error) {
	return []*models.NotificationItem{}, nil
}

type MockPreferenceRepository struct{}

func (m *MockPreferenceRepository) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
	defaultPrefs := models.DefaultNotificationPreferences(userID)
	return &defaultPrefs, nil
}

func (m *MockPreferenceRepository) UpdatePreferences(ctx context.Context, preferences *models.NotificationPreferences) error {
	return nil
}

func (m *MockPreferenceRepository) CreatePreferences(ctx context.Context, preferences *models.NotificationPreferences) error {
	return nil
}

func TestNotificationServiceTestSuite(t *testing.T) {
	suite.Run(t, new(NotificationServiceTestSuite))
}
