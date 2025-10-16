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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type StatisticsTestSuite struct {
	suite.Suite
	config      *TestDBConfig
	db          *sql.DB
	workService *WorkService
	router      *gin.Engine

	userID     uuid.UUID
	testWorkID uuid.UUID
}

func (suite *StatisticsTestSuite) SetupSuite() {
	// Setup database using test utilities
	suite.config = SetupTestDB(suite.T())
	suite.db = suite.config.DB

	// Clean up any existing test data
	suite.config.CleanupTestData()

	suite.workService = &WorkService{db: suite.db, redis: nil}

	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Create test data
	suite.createTestData()

	// Setup routes
	api := suite.router.Group("/api/v1")
	{
		// Work stats
		api.GET("/work/:work_id/stats", suite.workService.GetStats)

		// User stats
		my := api.Group("/my")
		{
			my.GET("/stats", suite.withAuth(), suite.workService.GetMyStats)
		}

		// Admin stats
		admin := api.Group("/admin")
		{
			admin.GET("/statistics", suite.withAdminAuth(), suite.workService.AdminGetStatistics)
		}
	}
}

func (suite *StatisticsTestSuite) TearDownSuite() {
	if suite.config != nil {
		suite.config.CleanupTestData()
	}
}

func (suite *StatisticsTestSuite) createTestData() {
	// Create test user using test utilities with unique name
	var err error
	uniqueName := fmt.Sprintf("statstest_%d", time.Now().UnixNano())
	suite.userID, _, err = suite.config.CreateTestUser(uniqueName, uniqueName+"@test.com")
	suite.Require().NoError(err)

	// Create test work using test utilities
	suite.testWorkID, err = suite.config.CreateTestWork(suite.userID, "Test Work for Statistics", "published")
	suite.Require().NoError(err)

	// Add some statistics (if the columns exist)
	_, err = suite.db.Exec(`
		UPDATE works SET hit_count = 50, kudos_count = 10, comment_count = 5, bookmark_count = 3
		WHERE id = $1`, suite.testWorkID)
	// Don't require this to pass as these columns may not exist in the current schema
	if err != nil {
		suite.T().Logf("Warning: Could not update work statistics: %v", err)
	}
}

func (suite *StatisticsTestSuite) withAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", suite.userID.String())
		c.Next()
	}
}

func (suite *StatisticsTestSuite) withAdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create admin user for testing
		adminID := uuid.New()
		_, err := suite.db.Exec(`
			INSERT INTO users (id, email, username, password_hash, role, email_verified, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())
			ON CONFLICT (email) DO NOTHING`,
			adminID, "admin@test.com", "admin", "$2a$10$hash", "admin")
		if err != nil {
			// If user already exists, get the ID
			suite.db.QueryRow("SELECT id FROM users WHERE email = 'admin@test.com'").Scan(&adminID)
		}

		c.Set("user_id", adminID.String())
		c.Next()
	}
}

func (suite *StatisticsTestSuite) TestGetWorkStats_Success() {
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/work/%s/stats", suite.testWorkID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	stats, ok := response["stats"].(map[string]interface{})
	assert.True(suite.T(), ok)

	// Check basic stats
	assert.Equal(suite.T(), "Test Work for Stats", stats["title"])
	assert.Equal(suite.T(), float64(1000), stats["word_count"])
	assert.Equal(suite.T(), float64(1), stats["chapter_count"])
	assert.Equal(suite.T(), float64(50), stats["hits"])
	assert.Equal(suite.T(), float64(10), stats["kudos"])
	assert.Equal(suite.T(), float64(5), stats["comments"])
	assert.Equal(suite.T(), float64(3), stats["bookmarks"])
}

func (suite *StatisticsTestSuite) TestGetWorkStats_WithOwnerAnalytics() {
	// Test with the work owner (should get detailed analytics)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/work/%s/stats", suite.testWorkID), nil)

	// Create router with owner auth
	router := gin.New()
	router.GET("/api/v1/work/:work_id/stats", suite.withAuth(), suite.workService.GetStats)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	stats, ok := response["stats"].(map[string]interface{})
	assert.True(suite.T(), ok)

	// Owner should have access to daily/monthly hits (even if empty)
	dailyHits, exists := stats["daily_hits"]
	assert.True(suite.T(), exists)
	if dailyHits != nil {
		hits := dailyHits.([]interface{})
		assert.IsType(suite.T(), []interface{}{}, hits)
	}
}

func (suite *StatisticsTestSuite) TestGetWorkStats_NonExistentWork() {
	fakeWorkID := uuid.New()
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/work/%s/stats", fakeWorkID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *StatisticsTestSuite) TestGetMyStats_Success() {
	req := httptest.NewRequest("GET", "/api/v1/my/stats", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	stats, ok := response["stats"].(map[string]interface{})
	assert.True(suite.T(), ok)

	// Check user stats
	userIDStr := stats["user_id"].(string)
	assert.Equal(suite.T(), suite.userID.String(), userIDStr)

	// Should have at least 1 work
	assert.GreaterOrEqual(suite.T(), int(stats["total_works"].(float64)), 1)
	assert.GreaterOrEqual(suite.T(), int(stats["published_works"].(float64)), 1)

	// Check engagement stats from our test work
	assert.GreaterOrEqual(suite.T(), int(stats["total_hits"].(float64)), 50)
	assert.GreaterOrEqual(suite.T(), int(stats["total_kudos"].(float64)), 10)
	assert.GreaterOrEqual(suite.T(), int(stats["total_comments"].(float64)), 5)
	assert.GreaterOrEqual(suite.T(), int(stats["total_bookmarks"].(float64)), 3)

	// Check top works
	topWorks, ok := stats["top_works"].([]interface{})
	assert.True(suite.T(), ok)
	if len(topWorks) > 0 {
		firstWork := topWorks[0].(map[string]interface{})
		assert.Equal(suite.T(), "Test Work for Stats", firstWork["title"])
	}
}

func (suite *StatisticsTestSuite) TestGetMyStats_Unauthorized() {
	req := httptest.NewRequest("GET", "/api/v1/my/stats", nil)

	// Router without auth
	router := gin.New()
	router.GET("/api/v1/my/stats", suite.workService.GetMyStats)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *StatisticsTestSuite) TestAdminGetStatistics_Success() {
	req := httptest.NewRequest("GET", "/api/v1/admin/statistics", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	stats, ok := response["stats"].(map[string]interface{})
	assert.True(suite.T(), ok)

	// Check system-wide stats
	assert.GreaterOrEqual(suite.T(), int(stats["total_works"].(float64)), 1)
	assert.GreaterOrEqual(suite.T(), int(stats["published_works"].(float64)), 1)
	assert.GreaterOrEqual(suite.T(), int(stats["total_users"].(float64)), 1)

	// Check content stats
	assert.GreaterOrEqual(suite.T(), int(stats["total_chapters"].(float64)), 1)
	assert.GreaterOrEqual(suite.T(), int(stats["total_word_count"].(float64)), 1000)

	// Check engagement stats
	assert.GreaterOrEqual(suite.T(), int(stats["total_hits"].(float64)), 50)
	assert.GreaterOrEqual(suite.T(), int(stats["total_kudos"].(float64)), 10)

	// Check moderation stats (should exist even if 0)
	assert.Contains(suite.T(), stats, "reports_to_review")
	assert.Contains(suite.T(), stats, "comments_in_moderation")

	// Check system health
	assert.Contains(suite.T(), stats, "database_connections")
}

func (suite *StatisticsTestSuite) TestAdminGetStatistics_Unauthorized() {
	req := httptest.NewRequest("GET", "/api/v1/admin/statistics", nil)

	// Router with regular user auth
	router := gin.New()
	router.GET("/api/v1/admin/statistics", suite.withAuth(), suite.workService.AdminGetStatistics)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

func (suite *StatisticsTestSuite) TestGetWorkStats_LegacyID() {
	// First set a legacy ID for our test work
	legacyID := 12345
	_, err := suite.db.Exec("UPDATE works SET legacy_id = $1 WHERE id = $2", legacyID, suite.testWorkID)
	suite.Require().NoError(err)

	// Test accessing stats via legacy ID
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/work/%d/stats", legacyID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	stats, ok := response["stats"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "Test Work for Stats", stats["title"])
}

func (suite *StatisticsTestSuite) TestGetWorkStats_PrivacyCheck() {
	// Create a restricted work
	restrictedWorkID := uuid.New()
	_, err := suite.db.Exec(`
		INSERT INTO works (id, title, summary, user_id, language, rating, status, restricted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`,
		restrictedWorkID, "Restricted Work", "Secret content", suite.userID,
		"en", "General Audiences", "posted", true)
	suite.Require().NoError(err)

	// Try to access stats without authentication
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/work/%s/stats", restrictedWorkID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)

	// Cleanup
	suite.db.Exec("DELETE FROM works WHERE id = $1", restrictedWorkID)
}

func TestStatisticsTestSuite(t *testing.T) {
	suite.Run(t, new(StatisticsTestSuite))
}
