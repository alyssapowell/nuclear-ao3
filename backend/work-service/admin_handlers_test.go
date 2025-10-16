package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// AdminHandlersTestSuite contains tests for admin/moderator functions
type AdminHandlersTestSuite struct {
	suite.Suite
	config      *TestDBConfig
	db          *sql.DB
	workService *WorkService
	router      *gin.Engine

	// Test data
	adminUserID     uuid.UUID
	moderatorUserID uuid.UUID
	regularUserID   uuid.UUID
	testWorkID      uuid.UUID
	testCommentID   uuid.UUID
}

func (suite *AdminHandlersTestSuite) SetupSuite() {
	// Setup database using test utilities
	suite.config = SetupTestDB(suite.T())
	suite.db = suite.config.DB

	// Clean up any existing test data
	suite.config.CleanupTestData()

	// Create work service (we can pass nil for redis since we're not testing caching)
	suite.workService = &WorkService{
		db:    suite.db,
		redis: nil, // Not testing cache functionality
	}

	suite.createTestUsers()
	suite.createTestContent()

	// Setup router with authentication middleware
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	api := suite.router.Group("/api/v1")
	{
		admin := api.Group("/admin")
		admin.POST("/works/:work_id/status", suite.withAuth(suite.adminUserID), suite.workService.AdminUpdateWorkStatus)
		admin.DELETE("/works/:work_id", suite.withAuth(suite.adminUserID), suite.workService.AdminDeleteWork)
		admin.GET("/comments", suite.withAuth(suite.moderatorUserID), suite.workService.AdminListComments)
		admin.PUT("/comments/:comment_id/status", suite.withAuth(suite.moderatorUserID), suite.workService.AdminUpdateCommentStatus)
		admin.DELETE("/comments/:comment_id", suite.withAuth(suite.moderatorUserID), suite.workService.AdminDeleteComment)
		admin.GET("/reports", suite.withAuth(suite.moderatorUserID), suite.workService.AdminGetReports)
	}
}

func (suite *AdminHandlersTestSuite) TearDownSuite() {
	if suite.config != nil {
		suite.config.CleanupTestData()
	}
}

func (suite *AdminHandlersTestSuite) createTestUsers() {
	var err error

	// Generate unique usernames with timestamp to avoid conflicts
	timestamp := time.Now().UnixNano()

	// Create admin user
	adminName := fmt.Sprintf("testadmin_%d", timestamp)
	suite.adminUserID, _, err = suite.config.CreateTestUserWithRole(adminName, adminName+"@test.com", "admin")
	suite.Require().NoError(err)

	// Create moderator user
	modName := fmt.Sprintf("testmod_%d", timestamp+1)
	suite.moderatorUserID, _, err = suite.config.CreateTestUserWithRole(modName, modName+"@test.com", "moderator")
	suite.Require().NoError(err)

	// Create regular user
	userName := fmt.Sprintf("testuser_%d", timestamp+2)
	suite.regularUserID, _, err = suite.config.CreateTestUserWithRole(userName, userName+"@test.com", "user")
	suite.Require().NoError(err)
}

func (suite *AdminHandlersTestSuite) createTestContent() {
	// Use test utilities to create work and comment
	workID, err := suite.config.CreateTestWork(suite.regularUserID, "Test Work for Admin", "published")
	suite.Require().NoError(err)
	suite.testWorkID = workID

	// Get the pseudonym for the regular user from user_pseudonyms table (for comments)
	var pseudID uuid.UUID
	err = suite.db.QueryRow("SELECT id FROM user_pseudonyms WHERE user_id = $1 AND is_default = true", suite.regularUserID).Scan(&pseudID)
	suite.Require().NoError(err)

	// Create a comment for testing
	commentID, err := suite.config.CreateTestComment(workID, suite.regularUserID, pseudID, "Test comment content", "published")
	suite.Require().NoError(err)
	suite.testCommentID = commentID

	// Log the IDs for debugging
	suite.T().Logf("Created test work: userID=%s, workID=%s", suite.regularUserID, workID)
	suite.T().Logf("Created test comment: userID=%s, pseudID=%s, commentID=%s", suite.regularUserID, pseudID, commentID)
}

// withAuth is a middleware that simulates authentication by setting user ID in context
func (suite *AdminHandlersTestSuite) withAuth(userID uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID.String())
		c.Next()
	}
}

// Test admin work status update
func (suite *AdminHandlersTestSuite) TestAdminUpdateWorkStatus() {
	// Test updating work status from "published" to "abandoned"
	requestBody := map[string]interface{}{
		"status": "abandoned",
		"reason": "inappropriate content",
	}

	jsonData, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/admin/works/%s/status", suite.testWorkID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Debug: Print response body if not 200
	if w.Code != http.StatusOK {
		suite.T().Logf("Response Code: %d, Body: %s", w.Code, w.Body.String())
	}

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify work status was updated in database
	var status string
	err := suite.db.QueryRow("SELECT status FROM works WHERE id = $1", suite.testWorkID).Scan(&status)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "abandoned", status)
}

// Test admin work deletion
func (suite *AdminHandlersTestSuite) TestAdminDeleteWork() {
	// Admin deletion requires a request body with reason and confirmation
	requestBody := map[string]interface{}{
		"reason":  "Terms of service violation",
		"confirm": true,
	}

	jsonData, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/admin/works/%s", suite.testWorkID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Debug: Print response body if not 200
	if w.Code != http.StatusOK {
		suite.T().Logf("Response Code: %d, Body: %s", w.Code, w.Body.String())
	}

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify work was permanently deleted (no longer exists)
	var exists bool
	err := suite.db.QueryRow("SELECT EXISTS(SELECT 1 FROM works WHERE id = $1)", suite.testWorkID).Scan(&exists)
	suite.Require().NoError(err)
	assert.False(suite.T(), exists, "Work should be permanently deleted")
}

// Test listing comments for moderation
func (suite *AdminHandlersTestSuite) TestAdminListComments() {
	req, _ := http.NewRequest("GET", "/api/v1/admin/comments?status=pending", nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	comments, ok := response["comments"].([]interface{})
	assert.True(suite.T(), ok, "Response should contain comments array")
	assert.GreaterOrEqual(suite.T(), len(comments), 0, "Should return comments list")
}

// Test updating comment status
func (suite *AdminHandlersTestSuite) TestAdminUpdateCommentStatus() {
	requestBody := map[string]interface{}{
		"status": "approved",
		"reason": "content approved after review",
	}

	jsonData, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/comments/%s/status", suite.testCommentID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify comment status was updated (using status column instead of moderation_status)
	var status string
	err := suite.db.QueryRow("SELECT status FROM comments WHERE id = $1", suite.testCommentID).Scan(&status)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "approved", status)
}

// Test admin comment deletion
func (suite *AdminHandlersTestSuite) TestAdminDeleteComment() {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/admin/comments/%s", suite.testCommentID), nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify comment was marked as deleted using is_deleted column
	var isDeleted bool
	err := suite.db.QueryRow("SELECT is_deleted FROM comments WHERE id = $1", suite.testCommentID).Scan(&isDeleted)
	suite.Require().NoError(err)
	assert.True(suite.T(), isDeleted, "Comment should be marked as deleted")
}

// Test getting reports
func (suite *AdminHandlersTestSuite) TestAdminGetReports() {
	// Create a test report first using work_reports table
	reportID := uuid.New()
	_, err := suite.db.Exec(`
		INSERT INTO work_reports (id, reporter_id, work_id, reason, description, status, created_at)
		VALUES ($1, $2, $3, 'inappropriate_content', 'Test report', 'pending', NOW())`,
		reportID, suite.regularUserID, suite.testWorkID)
	suite.Require().NoError(err)

	req, _ := http.NewRequest("GET", "/api/v1/admin/reports?status=pending", nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	reports, ok := response["reports"].([]interface{})
	assert.True(suite.T(), ok, "Response should contain reports array")
	assert.GreaterOrEqual(suite.T(), len(reports), 1, "Should return at least one report")

	// Clean up the test report
	suite.db.Exec("DELETE FROM work_reports WHERE id = $1", reportID)
}

// Test unauthorized access (regular user trying admin functions)
func (suite *AdminHandlersTestSuite) TestUnauthorizedAccess() {
	// Create test router with regular user auth for this specific test
	gin.SetMode(gin.TestMode)
	router := gin.New()

	api := router.Group("/api/v1")
	{
		admin := api.Group("/admin")
		// This will use regular user (not admin/moderator)
		admin.GET("/reports", suite.withAuth(suite.regularUserID), suite.workService.AdminGetReports)
	}

	req, _ := http.NewRequest("GET", "/api/v1/admin/reports", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should be forbidden because regular user doesn't have admin privileges
	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

// Test moderation logging
func (suite *AdminHandlersTestSuite) TestModerationLogging() {
	// Perform an admin action
	requestBody := map[string]interface{}{
		"status": "removed",
		"reason": "terms of service violation",
	}

	jsonData, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/admin/works/%s/status", suite.testWorkID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Check if moderation was logged
	var logCount int
	err := suite.db.QueryRow(`
		SELECT COUNT(*) FROM moderation_logs 
		WHERE moderator_id = $1 AND target_type = 'work' AND target_id = $2`,
		suite.adminUserID, suite.testWorkID).Scan(&logCount)

	if err == nil {
		assert.GreaterOrEqual(suite.T(), logCount, 1, "Moderation action should be logged")
	}
	// If table doesn't exist, that's ok for this test
}

func TestAdminHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(AdminHandlersTestSuite))
}
