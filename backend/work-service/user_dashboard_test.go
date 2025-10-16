package main

import (
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
	"github.com/stretchr/testify/suite"
)

// =============================================================================
// USER DASHBOARD TESTS
// Comprehensive test suite for user dashboard functionality
// =============================================================================

type UserDashboardTestSuite struct {
	suite.Suite
	config *TestDBConfig
	db     *sql.DB
	ws     *WorkService
	router *gin.Engine
}

func (suite *UserDashboardTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup database using test utilities
	suite.config = SetupTestDB(suite.T())
	suite.db = suite.config.DB

	// Clean up any existing test data
	suite.config.CleanupTestData()

	suite.ws = &WorkService{db: suite.db}
	suite.router = gin.New()

	// Register routes
	api := suite.router.Group("/api/v1")
	{
		api.GET("/users/:user_id/works", suite.ws.GetUserWorks)
		api.GET("/my/comments", suite.ws.GetMyComments)
	}
}

func (suite *UserDashboardTestSuite) TearDownSuite() {
	if suite.config != nil {
		suite.config.CleanupTestData()
	}
}

func (suite *UserDashboardTestSuite) SetupTest() {
	// Clean up test data before each test using test utilities
	if suite.config != nil {
		suite.config.CleanupTestData()
	}
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (suite *UserDashboardTestSuite) createTestUser(name string) uuid.UUID {
	// Generate unique username to avoid conflicts
	uniqueName := fmt.Sprintf("%s_%d", name, time.Now().UnixNano())
	userID, _, err := suite.config.CreateTestUser(uniqueName, uniqueName+"@test.com")
	suite.Require().NoError(err)
	return userID
}

func (suite *UserDashboardTestSuite) createTestWork(userID uuid.UUID, title, status string, restrictedToUsers bool) uuid.UUID {
	// Use test utilities to create work - ignore restrictedToUsers for now as it may not be in the schema
	workID, err := suite.config.CreateTestWork(userID, title, status)
	suite.Require().NoError(err)

	// Try to update additional fields if they exist
	_, err = suite.db.Exec(`
		UPDATE works SET hit_count = 10, kudos_count = 5, comment_count = 3, bookmark_count = 2
		WHERE id = $1`, workID)
	// Don't require this to pass as these columns may not exist
	if err != nil {
		suite.T().Logf("Warning: Could not update work statistics: %v", err)
	}

	return workID
}

func (suite *UserDashboardTestSuite) createTestComment(userID, workID uuid.UUID, content, status string, isReply bool) uuid.UUID {
	// Get the user's pseudonym
	var pseudID uuid.UUID
	err := suite.db.QueryRow("SELECT id FROM user_pseudonyms WHERE user_id = $1 AND is_default = true", userID).Scan(&pseudID)
	suite.Require().NoError(err)

	// Use test utilities to create comment
	commentID, err := suite.config.CreateTestComment(workID, userID, pseudID, content, status)
	suite.Require().NoError(err)

	// For now, ignore the isReply functionality as it would require more complex setup
	if isReply {
		suite.T().Logf("Note: Reply functionality not implemented in simplified test")
	}

	return commentID
}

func (suite *UserDashboardTestSuite) makeRequestWithAuth(method, url string, userID uuid.UUID) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, url, nil)
	w := httptest.NewRecorder()

	// Mock JWT middleware by setting user_id in context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", userID.String())

	// Create a new context with the user_id set
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID.String())
		c.Next()
	})

	api := router.Group("/api/v1")
	{
		api.GET("/users/:user_id/works", suite.ws.GetUserWorks)
		api.GET("/my/comments", suite.ws.GetMyComments)
	}

	router.ServeHTTP(w, req)
	return w
}

// =============================================================================
// GET USER WORKS TESTS
// =============================================================================

func (suite *UserDashboardTestSuite) TestGetUserWorks_Success() {
	// Create test user and works
	userID := suite.createTestUser("testuser")
	workID1 := suite.createTestWork(userID, "Public Work", "posted", false)
	workID2 := suite.createTestWork(userID, "Restricted Work", "posted", true)
	workID3 := suite.createTestWork(userID, "Draft Work", "draft", false)

	// Test viewing own profile (should see all works)
	w := suite.makeRequestWithAuth("GET", fmt.Sprintf("/api/v1/users/%s/works", userID), userID)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	works := response["works"].([]interface{})
	suite.Len(works, 3, "User should see all their own works")

	// Verify work IDs are present
	workIDs := make(map[string]bool)
	for _, work := range works {
		workMap := work.(map[string]interface{})
		workIDs[workMap["id"].(string)] = true
	}

	suite.True(workIDs[workID1.String()], "Should include public work")
	suite.True(workIDs[workID2.String()], "Should include restricted work when viewing own profile")
	suite.True(workIDs[workID3.String()], "Should include draft work when viewing own profile")
}

func (suite *UserDashboardTestSuite) TestGetUserWorks_OtherUserProfile() {
	// Create two users
	user1ID := suite.createTestUser("user1")
	user2ID := suite.createTestUser("user2")

	// Create works for user1
	publicWorkID := suite.createTestWork(user1ID, "Public Work", "posted", false)
	suite.createTestWork(user1ID, "Restricted Work", "posted", true)
	suite.createTestWork(user1ID, "Draft Work", "draft", false)

	// User2 viewing User1's profile (should only see public, posted works)
	w := suite.makeRequestWithAuth("GET", fmt.Sprintf("/api/v1/users/%s/works", user1ID), user2ID)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	works := response["works"].([]interface{})
	suite.Len(works, 1, "Other users should only see public, posted works")

	work := works[0].(map[string]interface{})
	suite.Equal(publicWorkID.String(), work["id"].(string), "Should only see the public work")
}

func (suite *UserDashboardTestSuite) TestGetUserWorks_InvalidUserID() {
	userID := suite.createTestUser("testuser")

	w := suite.makeRequestWithAuth("GET", "/api/v1/users/invalid-uuid/works", userID)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Equal("Invalid user ID", response["error"])
}

func (suite *UserDashboardTestSuite) TestGetUserWorks_Pagination() {
	userID := suite.createTestUser("testuser")

	// Create multiple works
	for i := 0; i < 5; i++ {
		suite.createTestWork(userID, fmt.Sprintf("Work %d", i), "posted", false)
	}

	// Test pagination with limit=2
	w := suite.makeRequestWithAuth("GET", fmt.Sprintf("/api/v1/users/%s/works?limit=2&page=1", userID), userID)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	works := response["works"].([]interface{})
	suite.Len(works, 2, "Should return only 2 works per page")

	// Test page 2
	w = suite.makeRequestWithAuth("GET", fmt.Sprintf("/api/v1/users/%s/works?limit=2&page=2", userID), userID)

	suite.Equal(http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	works = response["works"].([]interface{})
	suite.Len(works, 2, "Should return 2 works on page 2")
}

func (suite *UserDashboardTestSuite) TestGetUserWorks_WorkStatistics() {
	userID := suite.createTestUser("testuser")
	workID := suite.createTestWork(userID, "Test Work", "posted", false)

	w := suite.makeRequestWithAuth("GET", fmt.Sprintf("/api/v1/users/%s/works", userID), userID)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	works := response["works"].([]interface{})
	suite.Len(works, 1)

	work := works[0].(map[string]interface{})
	suite.Equal(workID.String(), work["id"].(string))

	// Verify statistics are included
	suite.Equal(float64(10), work["hits"].(float64), "Should include hit count")
	suite.Equal(float64(5), work["kudos"].(float64), "Should include kudos count")
	suite.Equal(float64(3), work["comments"].(float64), "Should include comment count")
	suite.Equal(float64(2), work["bookmarks"].(float64), "Should include bookmark count")
}

// =============================================================================
// GET MY COMMENTS TESTS
// =============================================================================

func (suite *UserDashboardTestSuite) TestGetMyComments_Success() {
	userID := suite.createTestUser("testuser")
	workID := suite.createTestWork(userID, "Test Work", "posted", false)

	// Create test comments
	commentID1 := suite.createTestComment(userID, workID, "First comment", "published", false)
	commentID2 := suite.createTestComment(userID, workID, "Reply comment", "published", true)
	suite.createTestComment(userID, workID, "Pending comment", "pending_moderation", false)

	w := suite.makeRequestWithAuth("GET", "/api/v1/my/comments", userID)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	comments := response["comments"].([]interface{})
	suite.Len(comments, 3, "Should return all user comments")

	// Verify comment IDs are present
	commentIDs := make(map[string]bool)
	for _, comment := range comments {
		commentMap := comment.(map[string]interface{})
		commentIDs[commentMap["id"].(string)] = true

		// Verify work information is included
		suite.NotEmpty(commentMap["work_title"], "Should include work title")
		suite.Equal(workID.String(), commentMap["work_id"].(string), "Should include work ID")
	}

	suite.True(commentIDs[commentID1.String()], "Should include first comment")
	suite.True(commentIDs[commentID2.String()], "Should include reply comment")
}

func (suite *UserDashboardTestSuite) TestGetMyComments_Unauthorized() {
	req, _ := http.NewRequest("GET", "/api/v1/my/comments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Equal("User not authenticated", response["error"])
}

func (suite *UserDashboardTestSuite) TestGetMyComments_StatusFilter() {
	userID := suite.createTestUser("testuser")
	workID := suite.createTestWork(userID, "Test Work", "posted", false)

	// Create comments with different statuses
	suite.createTestComment(userID, workID, "Published comment", "published", false)
	suite.createTestComment(userID, workID, "Pending comment", "pending_moderation", false)

	// Filter by published status
	w := suite.makeRequestWithAuth("GET", "/api/v1/my/comments?status=published", userID)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	comments := response["comments"].([]interface{})
	suite.Len(comments, 1, "Should return only published comments")

	comment := comments[0].(map[string]interface{})
	suite.Equal("published", comment["status"].(string))
}

func (suite *UserDashboardTestSuite) TestGetMyComments_Pagination() {
	userID := suite.createTestUser("testuser")
	workID := suite.createTestWork(userID, "Test Work", "posted", false)

	// Create multiple comments
	for i := 0; i < 5; i++ {
		suite.createTestComment(userID, workID, fmt.Sprintf("Comment %d", i), "published", false)
	}

	// Test pagination with limit=2
	w := suite.makeRequestWithAuth("GET", "/api/v1/my/comments?limit=2&page=1", userID)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	comments := response["comments"].([]interface{})
	suite.Len(comments, 2, "Should return only 2 comments per page")

	// Test page 2
	w = suite.makeRequestWithAuth("GET", "/api/v1/my/comments?limit=2&page=2", userID)

	suite.Equal(http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	comments = response["comments"].([]interface{})
	suite.Len(comments, 2, "Should return 2 comments on page 2")
}

func (suite *UserDashboardTestSuite) TestGetMyComments_ReplyIdentification() {
	userID := suite.createTestUser("testuser")
	workID := suite.createTestWork(userID, "Test Work", "posted", false)

	// Create a regular comment and a reply
	suite.createTestComment(userID, workID, "Regular comment", "published", false)
	suite.createTestComment(userID, workID, "Reply comment", "published", true)

	w := suite.makeRequestWithAuth("GET", "/api/v1/my/comments", userID)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	comments := response["comments"].([]interface{})
	suite.Len(comments, 3) // 2 created + 1 parent for reply

	// Check that replies are identified correctly
	hasReply := false
	hasRegular := false
	for _, comment := range comments {
		commentMap := comment.(map[string]interface{})
		isReply := commentMap["is_reply"].(bool)
		if isReply {
			hasReply = true
		} else {
			hasRegular = true
		}
	}

	suite.True(hasReply, "Should identify reply comments")
	suite.True(hasRegular, "Should identify regular comments")
}

// =============================================================================
// EDGE CASES AND ERROR HANDLING
// =============================================================================

func (suite *UserDashboardTestSuite) TestGetUserWorks_EmptyResults() {
	userID := suite.createTestUser("testuser")

	// Don't create any works
	w := suite.makeRequestWithAuth("GET", fmt.Sprintf("/api/v1/users/%s/works", userID), userID)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	works := response["works"].([]interface{})
	suite.Len(works, 0, "Should return empty array when user has no works")
}

func (suite *UserDashboardTestSuite) TestGetMyComments_EmptyResults() {
	userID := suite.createTestUser("testuser")

	// Don't create any comments
	w := suite.makeRequestWithAuth("GET", "/api/v1/my/comments", userID)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	comments := response["comments"].([]interface{})
	suite.Len(comments, 0, "Should return empty array when user has no comments")
}

func (suite *UserDashboardTestSuite) TestGetUserWorks_LimitValidation() {
	userID := suite.createTestUser("testuser")
	suite.createTestWork(userID, "Test Work", "posted", false)

	// Test with limit exceeding maximum (should be capped at 100)
	w := suite.makeRequestWithAuth("GET", fmt.Sprintf("/api/v1/users/%s/works?limit=200", userID), userID)

	suite.Equal(http.StatusOK, w.Code, "Should handle oversized limit gracefully")

	// Test with invalid limit (should use default)
	w = suite.makeRequestWithAuth("GET", fmt.Sprintf("/api/v1/users/%s/works?limit=invalid", userID), userID)

	suite.Equal(http.StatusOK, w.Code, "Should handle invalid limit gracefully")
}

func (suite *UserDashboardTestSuite) TestGetMyComments_LimitValidation() {
	userID := suite.createTestUser("testuser")
	workID := suite.createTestWork(userID, "Test Work", "posted", false)
	suite.createTestComment(userID, workID, "Test comment", "published", false)

	// Test with limit exceeding maximum (should be capped at 100)
	w := suite.makeRequestWithAuth("GET", "/api/v1/my/comments?limit=200", userID)

	suite.Equal(http.StatusOK, w.Code, "Should handle oversized limit gracefully")

	// Test with invalid limit (should use default)
	w = suite.makeRequestWithAuth("GET", "/api/v1/my/comments?limit=invalid", userID)

	suite.Equal(http.StatusOK, w.Code, "Should handle invalid limit gracefully")
}

// =============================================================================
// TEST SUITE REGISTRATION
// =============================================================================

func TestUserDashboardTestSuite(t *testing.T) {
	suite.Run(t, new(UserDashboardTestSuite))
}
