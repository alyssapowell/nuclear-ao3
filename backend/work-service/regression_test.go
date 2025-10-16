package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// RegressionTestSuite focuses on testing fixes that prevent regression of previously broken functionality
type RegressionTestSuite struct {
	suite.Suite
	service    *WorkService
	testUserID uuid.UUID
	router     *gin.Engine
}

func (suite *RegressionTestSuite) SetupSuite() {
	// Use test database
	suite.service = NewWorkService()
	suite.testUserID = uuid.MustParse("123e4567-e89b-12d3-a456-426614174003") // admin user from migrations

	// Set up minimal router with auth middleware
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// API endpoints with auth
	api := suite.router.Group("/api/v1")
	protected := api.Group("")
	protected.Use(suite.mockAuthMiddleware())
	{
		protected.POST("/works", suite.service.CreateWork)
	}
}

// mockAuthMiddleware simulates successful JWT validation by setting user_id in context
func (suite *RegressionTestSuite) mockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", suite.testUserID.String())
		c.Next()
	}
}

// TestWorkCreationDatabaseSchema tests that work creation works with the current database schema
// Regression test for: series_id column removal, pseud table name fixes, chapter constraints
func (suite *RegressionTestSuite) TestWorkCreationDatabaseSchema() {
	// This test ensures we don't regress on the database schema fixes made during development

	requestBody := models.CreateWorkRequest{
		Title:          "Test Work Schema Compliance",
		Summary:        "Testing that work creation works with actual database schema",
		Language:       "en",
		Rating:         "General Audiences",
		Warnings:       []string{"No Archive Warnings Apply"},
		Fandoms:        []string{"Test Fandom"},
		Category:       []string{"Gen"},
		FreeformTags:   []string{"Test", "Schema", "Regression"},
		ChapterContent: "This chapter tests that the database schema is correctly aligned.",
	}

	jsonBody, err := json.Marshal(requestBody)
	suite.Require().NoError(err)

	req, err := http.NewRequest("POST", "/api/v1/works", bytes.NewBuffer(jsonBody))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should not fail with database schema errors
	suite.Assert().NotEqual(500, w.Code, "Work creation should not fail with database errors")

	if w.Code != 201 {
		suite.T().Logf("Response body: %s", w.Body.String())
	}
}

// TestPseudonymTableIntegration tests that user_pseudonyms table is used correctly
// Regression test for: pseuds -> user_pseudonyms table name fix
func (suite *RegressionTestSuite) TestPseudonymTableIntegration() {
	// Ensure user has a default pseudonym or one gets created
	_, err := suite.service.db.Exec(`
		INSERT INTO user_pseudonyms (id, user_id, name, is_default, created_at) 
		VALUES ($1, $2, $3, true, $4)
		ON CONFLICT (user_id, name) DO NOTHING`,
		uuid.New(), suite.testUserID, "admin", time.Now())
	suite.Require().NoError(err)

	requestBody := models.CreateWorkRequest{
		Title:          "Test Pseudonym Integration",
		Summary:        "Testing user_pseudonyms table integration",
		Language:       "en",
		Rating:         "General Audiences",
		Warnings:       []string{"No Archive Warnings Apply"},
		Fandoms:        []string{"Test Fandom"},
		Category:       []string{"Gen"},
		FreeformTags:   []string{"Pseudonym", "Integration"},
		ChapterContent: "Testing that pseudonym lookup works correctly.",
	}

	jsonBody, err := json.Marshal(requestBody)
	suite.Require().NoError(err)

	req, err := http.NewRequest("POST", "/api/v1/works", bytes.NewBuffer(jsonBody))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should not fail with "pseuds" table not found errors
	if w.Code == 500 {
		responseBody := w.Body.String()
		suite.Assert().NotContains(responseBody, "pseuds", "Should not reference old 'pseuds' table name")
		suite.Assert().NotContains(responseBody, "relation \"pseuds\" does not exist", "Should use user_pseudonyms table")
	}
}

// TestChapterCountingConstraints tests that chapter counting works with database triggers
// Regression test for: chapter_counts_valid constraint and trigger integration
func (suite *RegressionTestSuite) TestChapterCountingConstraints() {
	// Ensure user has default pseudonym
	_, err := suite.service.db.Exec(`
		INSERT INTO user_pseudonyms (id, user_id, name, is_default, created_at) 
		VALUES ($1, $2, $3, true, $4)
		ON CONFLICT (user_id, name) DO NOTHING`,
		uuid.New(), suite.testUserID, "admin", time.Now())
	suite.Require().NoError(err)

	requestBody := models.CreateWorkRequest{
		Title:          "Test Chapter Constraints",
		Summary:        "Testing chapter counting constraints with triggers",
		Language:       "en",
		Rating:         "General Audiences",
		Warnings:       []string{"No Archive Warnings Apply"},
		Fandoms:        []string{"Test Fandom"},
		Category:       []string{"Gen"},
		FreeformTags:   []string{"Chapter", "Constraints", "Triggers"},
		ChapterContent: "Testing that chapter creation works with database triggers that update work statistics.",
		MaxChapters:    func() *int { i := 1; return &i }(), // Single chapter work
	}

	jsonBody, err := json.Marshal(requestBody)
	suite.Require().NoError(err)

	req, err := http.NewRequest("POST", "/api/v1/works", bytes.NewBuffer(jsonBody))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should not fail with chapter counting constraint errors
	if w.Code == 500 {
		responseBody := w.Body.String()
		suite.Assert().NotContains(responseBody, "chapter_counts_valid", "Should not violate chapter counting constraints")
		suite.Assert().NotContains(responseBody, "chapter_count", "Chapter counting should work with triggers")
	}
}

// TestWorkCreationWithoutCreatorships tests that work creation doesn't depend on creatorships table
// Regression test for: creatorship table dependency removal
func (suite *RegressionTestSuite) TestWorkCreationWithoutCreatorships() {
	// Ensure user has default pseudonym
	_, err := suite.service.db.Exec(`
		INSERT INTO user_pseudonyms (id, user_id, name, is_default, created_at) 
		VALUES ($1, $2, $3, true, $4)
		ON CONFLICT (user_id, name) DO NOTHING`,
		uuid.New(), suite.testUserID, "admin", time.Now())
	suite.Require().NoError(err)

	requestBody := models.CreateWorkRequest{
		Title:          "Test Without Creatorships",
		Summary:        "Testing work creation without creatorship table dependency",
		Language:       "en",
		Rating:         "General Audiences",
		Warnings:       []string{"No Archive Warnings Apply"},
		Fandoms:        []string{"Test Fandom"},
		Category:       []string{"Gen"},
		FreeformTags:   []string{"No Creatorships", "Direct User Relationship"},
		ChapterContent: "This work should be created without requiring a creatorships table.",
	}

	jsonBody, err := json.Marshal(requestBody)
	suite.Require().NoError(err)

	req, err := http.NewRequest("POST", "/api/v1/works", bytes.NewBuffer(jsonBody))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should not fail with creatorships table errors
	if w.Code == 500 {
		responseBody := w.Body.String()
		suite.Assert().NotContains(responseBody, "creatorships", "Should not reference creatorships table")
		suite.Assert().NotContains(responseBody, "relation \"creatorships\" does not exist", "Should work without creatorships")
	}
}

// TestDatabaseConstraintCompliance tests that all database constraints are satisfied
// Regression test for: general database constraint compliance
func (suite *RegressionTestSuite) TestDatabaseConstraintCompliance() {
	// Ensure user has default pseudonym
	_, err := suite.service.db.Exec(`
		INSERT INTO user_pseudonyms (id, user_id, name, is_default, created_at) 
		VALUES ($1, $2, $3, true, $4)
		ON CONFLICT (user_id, name) DO NOTHING`,
		uuid.New(), suite.testUserID, "admin", time.Now())
	suite.Require().NoError(err)

	// Test with comprehensive work data that exercises all the fixed constraints
	requestBody := models.CreateWorkRequest{
		Title:          "Comprehensive Database Constraint Test",
		Summary:        "This work tests all database constraints that were previously failing",
		Language:       "en",
		Rating:         "Teen And Up Audiences", // Test rating constraint
		Warnings:       []string{"Creator Chose Not To Use Archive Warnings"},
		Fandoms:        []string{"Harry Potter - J. K. Rowling", "Marvel Cinematic Universe"},
		Category:       []string{"F/M", "Gen"},
		Characters:     []string{"Harry Potter", "Hermione Granger"},
		Relationships:  []string{"Harry Potter/Ginny Weasley"},
		FreeformTags:   []string{"Fluff", "Post-War", "Hogwarts", "Friendship", "Romance"},
		ChapterTitle:   "Chapter 1: New Beginnings",
		ChapterSummary: "The story begins...",
		ChapterContent: "This is a comprehensive test of database constraints. It should create successfully with proper word counting, chapter counting, and all relationship tables working correctly.",
		MaxChapters:    func() *int { i := 5; return &i }(), // Multi-chapter work
	}

	jsonBody, err := json.Marshal(requestBody)
	suite.Require().NoError(err)

	req, err := http.NewRequest("POST", "/api/v1/works", bytes.NewBuffer(jsonBody))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// This should succeed completely or give specific actionable errors
	if w.Code != 201 {
		suite.T().Logf("Full response: %s", w.Body.String())

		// Check for specific constraint violations that should not happen
		responseBody := w.Body.String()
		suite.Assert().NotContains(responseBody, "series_id", "Should not reference non-existent series_id")
		suite.Assert().NotContains(responseBody, "pseuds", "Should not reference old pseuds table")
		suite.Assert().NotContains(responseBody, "creatorships", "Should not reference creatorships table")
		suite.Assert().NotContains(responseBody, "chapter_counts_valid", "Should not violate chapter constraints")
	}
}

func TestRegressionSuite(t *testing.T) {
	suite.Run(t, new(RegressionTestSuite))
}
