package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"nuclear-ao3/shared/models"
)

type UserProfileHandlersTestSuite struct {
	suite.Suite
	db           *sql.DB
	authService  *AuthService
	router       *gin.Engine
	testUserID   uuid.UUID
	testUsername string
}

func (suite *UserProfileHandlersTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup test database connection
	dbURL := "postgres://ao3_user:ao3_password@localhost/ao3_nuclear?sslmode=disable"
	db, err := sql.Open("postgres", dbURL)
	suite.Require().NoError(err)

	err = db.Ping()
	suite.Require().NoError(err)

	suite.db = db

	// Create a minimal AuthService for testing
	suite.authService = &AuthService{
		db: db,
	}

	// Setup router with user profile routes
	suite.router = gin.New()
	suite.router.Use(func(c *gin.Context) {
		// Mock authentication middleware
		c.Set("user_id", suite.testUserID.String())
		c.Next()
	})

	api := suite.router.Group("/api/v1")
	{
		api.GET("/users/:username", suite.authService.GetUserProfile)
		api.PUT("/profile", suite.authService.UpdateUserProfile)
		api.POST("/pseudonyms", suite.authService.CreateUserPseudonym)
		api.GET("/pseudonyms", suite.authService.GetUserPseudonyms)
		api.POST("/users/:username/friend-request", suite.authService.SendFriendRequest)
		api.PUT("/friend-requests/:relationshipId", suite.authService.RespondToFriendRequest)
		api.POST("/users/:username/block", suite.authService.BlockUser)
		api.DELETE("/users/:username/block", suite.authService.UnblockUser)
		api.GET("/dashboard", suite.authService.GetUserDashboard)
	}
}

func (suite *UserProfileHandlersTestSuite) SetupTest() {
	// Create test user with unique username
	suite.testUserID = uuid.New()
	suite.testUsername = fmt.Sprintf("testuser_%s", suite.testUserID.String()[:8])
	email := fmt.Sprintf("test_%s@example.com", suite.testUserID.String()[:8])

	_, err := suite.db.Exec(`
		INSERT INTO users (id, username, email, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, suite.testUserID, suite.testUsername, email, "hashed_password")
	suite.Require().NoError(err)
}

func (suite *UserProfileHandlersTestSuite) TearDownTest() {
	// Clean up test data
	suite.db.Exec("DELETE FROM user_relationships WHERE requester_id = $1 OR addressee_id = $1", suite.testUserID)
	suite.db.Exec("DELETE FROM user_blocks WHERE blocker_id = $1 OR blocked_id = $1", suite.testUserID)
	suite.db.Exec("DELETE FROM user_pseudonyms WHERE user_id = $1", suite.testUserID)
	suite.db.Exec("DELETE FROM users WHERE id = $1", suite.testUserID)
}

func (suite *UserProfileHandlersTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *UserProfileHandlersTestSuite) TestGetUserProfile_Success() {
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/users/%s", suite.testUsername), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var profile models.UserProfile
	err := json.Unmarshal(w.Body.Bytes(), &profile)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testUsername, profile.Username)
	assert.Equal(suite.T(), suite.testUserID, profile.ID)
}

func (suite *UserProfileHandlersTestSuite) TestGetUserProfile_NotFound() {
	req, _ := http.NewRequest("GET", "/api/v1/users/nonexistentuser", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *UserProfileHandlersTestSuite) TestUpdateUserProfile_Success() {
	requestBody := models.UserProfileUpdateRequest{
		DisplayName: stringPtr("Updated Display Name"),
		Bio:         stringPtr("This is my updated bio"),
		Location:    stringPtr("New York"),
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("PUT", "/api/v1/profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify profile was updated
	var displayName, bio, location sql.NullString
	suite.db.QueryRow("SELECT display_name, bio, location FROM users WHERE id = $1", suite.testUserID).Scan(&displayName, &bio, &location)

	assert.True(suite.T(), displayName.Valid)
	assert.Equal(suite.T(), "Updated Display Name", displayName.String)
	assert.True(suite.T(), bio.Valid)
	assert.Equal(suite.T(), "This is my updated bio", bio.String)
	assert.True(suite.T(), location.Valid)
	assert.Equal(suite.T(), "New York", location.String)
}

func (suite *UserProfileHandlersTestSuite) TestCreateUserPseudonym_Success() {
	requestBody := models.UserPseudonymRequest{
		Name:        "TestPseudonym",
		IsDefault:   true,
		Description: stringPtr("My test pseudonym"),
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/v1/pseudonyms", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response models.UserPseudonym
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "TestPseudonym", response.Name)
	assert.True(suite.T(), response.IsDefault)
	assert.Equal(suite.T(), suite.testUserID, response.UserID)

	// Verify pseudonym was created in database
	var count int
	suite.db.QueryRow("SELECT COUNT(*) FROM user_pseudonyms WHERE user_id = $1 AND name = $2", suite.testUserID, "TestPseudonym").Scan(&count)
	assert.Equal(suite.T(), 1, count)
}

func (suite *UserProfileHandlersTestSuite) TestCreateUserPseudonym_DuplicateName() {
	// Create first pseudonym
	suite.createTestPseudonym("TestPseud", true)

	// Try to create another with same name
	requestBody := models.UserPseudonymRequest{
		Name:      "TestPseud",
		IsDefault: false,
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/v1/pseudonyms", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusConflict, w.Code)
}

func (suite *UserProfileHandlersTestSuite) TestGetUserPseudonyms_Success() {
	// Create test pseudonyms
	suite.createTestPseudonym("MainPseud", true)
	suite.createTestPseudonym("AltPseud", false)

	req, _ := http.NewRequest("GET", "/api/v1/pseudonyms", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	pseudonyms := response["pseudonyms"].([]interface{})
	assert.Len(suite.T(), pseudonyms, 2)

	// Default pseudonym should be first
	firstPseud := pseudonyms[0].(map[string]interface{})
	assert.Equal(suite.T(), "MainPseud", firstPseud["name"])
	assert.Equal(suite.T(), true, firstPseud["is_default"])
}

func (suite *UserProfileHandlersTestSuite) TestSendFriendRequest_Success() {
	// Create target user
	targetUserID := uuid.New()
	targetUsername := fmt.Sprintf("target_%s", targetUserID.String()[:8])
	suite.db.Exec(`
		INSERT INTO users (id, username, email, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, 'hashed', true, NOW(), NOW())
	`, targetUserID, targetUsername, fmt.Sprintf("%s@example.com", targetUsername))
	defer suite.db.Exec("DELETE FROM users WHERE id = $1", targetUserID)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/users/%s/friend-request", targetUsername), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Verify friend request was created
	var count int
	suite.db.QueryRow(`
		SELECT COUNT(*) FROM user_relationships 
		WHERE requester_id = $1 AND addressee_id = $2 AND status = 'pending'
	`, suite.testUserID, targetUserID).Scan(&count)
	assert.Equal(suite.T(), 1, count)
}

func (suite *UserProfileHandlersTestSuite) TestSendFriendRequest_ToSelf() {
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/users/%s/friend-request", suite.testUsername), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *UserProfileHandlersTestSuite) TestBlockUser_Success() {
	// Create target user
	targetUserID := uuid.New()
	targetUsername := fmt.Sprintf("target_%s", targetUserID.String()[:8])
	suite.db.Exec(`
		INSERT INTO users (id, username, email, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, 'hashed', true, NOW(), NOW())
	`, targetUserID, targetUsername, fmt.Sprintf("%s@example.com", targetUsername))
	defer suite.db.Exec("DELETE FROM users WHERE id = $1", targetUserID)

	requestBody := map[string]interface{}{
		"block_type": "full",
		"reason":     "Testing block functionality",
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/users/%s/block", targetUsername), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Verify block was created
	var count int
	suite.db.QueryRow(`
		SELECT COUNT(*) FROM user_blocks 
		WHERE blocker_id = $1 AND blocked_id = $2 AND block_type = 'full'
	`, suite.testUserID, targetUserID).Scan(&count)
	assert.Equal(suite.T(), 1, count)
}

func (suite *UserProfileHandlersTestSuite) TestUnblockUser_Success() {
	// Create target user and block them first
	targetUserID := uuid.New()
	targetUsername := fmt.Sprintf("target_%s", targetUserID.String()[:8])
	suite.db.Exec(`
		INSERT INTO users (id, username, email, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, 'hashed', true, NOW(), NOW())
	`, targetUserID, targetUsername, fmt.Sprintf("%s@example.com", targetUsername))
	defer suite.db.Exec("DELETE FROM users WHERE id = $1", targetUserID)

	// Create block
	suite.db.Exec(`
		INSERT INTO user_blocks (id, blocker_id, blocked_id, block_type, reason, created_at)
		VALUES (uuid_generate_v4(), $1, $2, 'full', 'test', NOW())
	`, suite.testUserID, targetUserID)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%s/block", targetUsername), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify block was removed
	var count int
	suite.db.QueryRow(`
		SELECT COUNT(*) FROM user_blocks 
		WHERE blocker_id = $1 AND blocked_id = $2
	`, suite.testUserID, targetUserID).Scan(&count)
	assert.Equal(suite.T(), 0, count)
}

func (suite *UserProfileHandlersTestSuite) TestGetUserDashboard_Success() {
	req, _ := http.NewRequest("GET", "/api/v1/dashboard", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var dashboard models.UserDashboard
	err := json.Unmarshal(w.Body.Bytes(), &dashboard)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testUserID, dashboard.ID)
	assert.Equal(suite.T(), suite.testUsername, dashboard.Username)
}

// Helper functions

func (suite *UserProfileHandlersTestSuite) createTestPseudonym(name string, isDefault bool) {
	_, err := suite.db.Exec(`
		INSERT INTO user_pseudonyms (id, user_id, name, is_default, created_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, NOW())
	`, suite.testUserID, name, isDefault)
	suite.Require().NoError(err)
}

func stringPtr(s string) *string {
	return &s
}

func TestUserProfileHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(UserProfileHandlersTestSuite))
}
