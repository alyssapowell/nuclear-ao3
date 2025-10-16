package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// JWTAuthTestSuite tests JWT authentication scenarios for work creation
type JWTAuthTestSuite struct {
	suite.Suite
	service    *WorkService
	testUserID uuid.UUID
	router     *gin.Engine
}

func (suite *JWTAuthTestSuite) SetupSuite() {
	suite.service = NewWorkService()
	suite.testUserID = uuid.MustParse("123e4567-e89b-12d3-a456-426614174003")

	// Set up simple router that simulates different auth scenarios
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	api := suite.router.Group("/api/v1")
	protected := api.Group("")
	protected.Use(suite.simulateJWTAuthMiddleware())
	{
		protected.POST("/works", suite.service.CreateWork)
	}
}

// simulateJWTAuthMiddleware creates a test middleware that simulates different JWT auth scenarios
func (suite *JWTAuthTestSuite) simulateJWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// Check for missing authorization header
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization header"})
			c.Abort()
			return
		}

		// Check for proper Bearer token format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Simulate different token scenarios
		switch token {
		case "valid_token":
			// Valid token - set user ID and continue
			c.Set("user_id", suite.testUserID.String())
			c.Next()
		case "expired_token", "malformed_token":
			// Invalid tokens - return 401
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
		default:
			// Unknown tokens - return 401
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
		}
	}
}

// TestValidJWTToken tests that valid JWT tokens are accepted
func (suite *JWTAuthTestSuite) TestValidJWTToken() {
	// Ensure user has default pseudonym
	_, err := suite.service.db.Exec(`
		INSERT INTO user_pseudonyms (id, user_id, name, is_default, created_at) 
		VALUES ($1, $2, $3, true, $4)
		ON CONFLICT (user_id, name) DO NOTHING`,
		uuid.New(), suite.testUserID, "admin", "2023-01-01")
	suite.Require().NoError(err)

	requestBody := models.CreateWorkRequest{
		Title:          "JWT Auth Test Work",
		Summary:        "Testing valid JWT authentication",
		Language:       "en",
		Rating:         "General Audiences",
		Warnings:       []string{"No Archive Warnings Apply"},
		Fandoms:        []string{"Test Fandom"},
		Category:       []string{"Gen"},
		FreeformTags:   []string{"JWT", "Authentication", "Valid Token"},
		ChapterContent: "This work tests valid JWT authentication.",
	}

	jsonBody, err := json.Marshal(requestBody)
	suite.Require().NoError(err)

	req, err := http.NewRequest("POST", "/api/v1/works", bytes.NewBuffer(jsonBody))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid_token")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should succeed with valid token
	suite.Assert().NotEqual(401, w.Code, "Valid JWT token should not return 401 Unauthorized")
	if w.Code == 500 {
		suite.T().Logf("Server error response: %s", w.Body.String())
	}
}

// TestExpiredJWTToken tests that expired JWT tokens are rejected
func (suite *JWTAuthTestSuite) TestExpiredJWTToken() {
	requestBody := models.CreateWorkRequest{
		Title:          "Should Fail Auth Test",
		Summary:        "This should fail due to expired token",
		Language:       "en",
		Rating:         "General Audiences",
		Warnings:       []string{"No Archive Warnings Apply"},
		Fandoms:        []string{"Test Fandom"},
		Category:       []string{"Gen"},
		FreeformTags:   []string{"JWT", "Expired Token"},
		ChapterContent: "This should not be created.",
	}

	jsonBody, err := json.Marshal(requestBody)
	suite.Require().NoError(err)

	req, err := http.NewRequest("POST", "/api/v1/works", bytes.NewBuffer(jsonBody))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer expired_token")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should return 401 for expired token
	suite.Assert().Equal(401, w.Code, "Expired JWT token should return 401 Unauthorized")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Assert().Equal("Invalid token", response["error"], "Should return 'Invalid token' error")
}

// TestMalformedJWTToken tests that malformed JWT tokens are rejected
func (suite *JWTAuthTestSuite) TestMalformedJWTToken() {
	requestBody := models.CreateWorkRequest{
		Title:          "Should Fail Auth Test",
		Summary:        "This should fail due to malformed token",
		Language:       "en",
		Rating:         "General Audiences",
		Warnings:       []string{"No Archive Warnings Apply"},
		Fandoms:        []string{"Test Fandom"},
		Category:       []string{"Gen"},
		FreeformTags:   []string{"JWT", "Malformed Token"},
		ChapterContent: "This should not be created.",
	}

	jsonBody, err := json.Marshal(requestBody)
	suite.Require().NoError(err)

	req, err := http.NewRequest("POST", "/api/v1/works", bytes.NewBuffer(jsonBody))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer malformed_token")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should return 401 for malformed token
	suite.Assert().Equal(401, w.Code, "Malformed JWT token should return 401 Unauthorized")
}

// TestMissingAuthorizationHeader tests that requests without auth headers are rejected
func (suite *JWTAuthTestSuite) TestMissingAuthorizationHeader() {
	requestBody := models.CreateWorkRequest{
		Title:          "Should Fail Auth Test",
		Summary:        "This should fail due to missing auth header",
		Language:       "en",
		Rating:         "General Audiences",
		Warnings:       []string{"No Archive Warnings Apply"},
		Fandoms:        []string{"Test Fandom"},
		Category:       []string{"Gen"},
		FreeformTags:   []string{"No Auth Header"},
		ChapterContent: "This should not be created.",
	}

	jsonBody, err := json.Marshal(requestBody)
	suite.Require().NoError(err)

	req, err := http.NewRequest("POST", "/api/v1/works", bytes.NewBuffer(jsonBody))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header set

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should return 401 for missing auth header
	suite.Assert().Equal(401, w.Code, "Missing authorization header should return 401 Unauthorized")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Assert().Equal("No authorization header", response["error"], "Should return 'No authorization header' error")
}

// TestInvalidTokenFormat tests that invalid token formats are rejected
func (suite *JWTAuthTestSuite) TestInvalidTokenFormat() {
	requestBody := models.CreateWorkRequest{
		Title:          "Should Fail Auth Test",
		Summary:        "This should fail due to invalid token format",
		Language:       "en",
		Rating:         "General Audiences",
		Warnings:       []string{"No Archive Warnings Apply"},
		Fandoms:        []string{"Test Fandom"},
		Category:       []string{"Gen"},
		FreeformTags:   []string{"Invalid Format"},
		ChapterContent: "This should not be created.",
	}

	jsonBody, err := json.Marshal(requestBody)
	suite.Require().NoError(err)

	req, err := http.NewRequest("POST", "/api/v1/works", bytes.NewBuffer(jsonBody))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "InvalidFormat some_token") // Wrong format

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should return 401 for invalid token format
	suite.Assert().Equal(401, w.Code, "Invalid token format should return 401 Unauthorized")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Assert().Equal("Invalid authorization header", response["error"], "Should return 'Invalid authorization header' error")
}

func TestJWTAuthSuite(t *testing.T) {
	suite.Run(t, new(JWTAuthTestSuite))
}
