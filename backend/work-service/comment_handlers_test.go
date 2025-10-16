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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"nuclear-ao3/shared/models"
)

type CommentHandlersTestSuite struct {
	suite.Suite
	db          *sql.DB
	workService *WorkService
	router      *gin.Engine
	testUserID  uuid.UUID
	testWorkID  uuid.UUID
	testPseudID uuid.UUID
}

func (suite *CommentHandlersTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup test database connection
	dbURL := "postgres://ao3_user:ao3_password@localhost/ao3_nuclear?sslmode=disable"
	db, err := sql.Open("postgres", dbURL)
	suite.Require().NoError(err)

	err = db.Ping()
	suite.Require().NoError(err)

	suite.db = db

	// Create a minimal WorkService for testing
	suite.workService = &WorkService{
		db:    db,
		redis: nil, // Redis not needed for basic tests
	}

	// Setup router with comment routes
	suite.router = gin.New()
	suite.router.Use(func(c *gin.Context) {
		// Mock authentication middleware
		c.Set("user_id", suite.testUserID.String())
		c.Next()
	})

	api := suite.router.Group("/api/v1")
	{
		api.GET("/works/:id/comments", suite.workService.GetWorkComments)
		api.POST("/works/:id/comments", suite.workService.CreateComment)
		api.PUT("/comments/:commentId", suite.workService.UpdateComment)
		api.DELETE("/comments/:commentId", suite.workService.DeleteComment)
		api.POST("/comments/:commentId/kudos", suite.workService.GiveCommentKudos)
	}
}

func (suite *CommentHandlersTestSuite) SetupTest() {
	// Create test user with unique username
	suite.testUserID = uuid.New()
	username := fmt.Sprintf("testuser_%s", suite.testUserID.String()[:8])
	email := fmt.Sprintf("test_%s@example.com", suite.testUserID.String()[:8])
	_, err := suite.db.Exec(`
		INSERT INTO users (id, username, email, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, suite.testUserID, username, email, "hashed_password")
	suite.Require().NoError(err)

	// Create test pseudonym
	suite.testPseudID = uuid.New()
	_, err = suite.db.Exec(`
		INSERT INTO user_pseudonyms (id, user_id, name, is_default, created_at)
		VALUES ($1, $2, $3, true, NOW())
		ON CONFLICT (id) DO NOTHING
	`, suite.testPseudID, suite.testUserID, "TestPseud")
	suite.Require().NoError(err)

	// Create test work
	suite.testWorkID = uuid.New()
	_, err = suite.db.Exec(`
		INSERT INTO works (id, title, summary, user_id, language, rating, status, word_count, chapter_count, is_complete, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'en', 'General Audiences', 'published', 1000, 1, true, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, suite.testWorkID, "Test Work", "Test Summary", suite.testUserID)
	suite.Require().NoError(err)
}

func (suite *CommentHandlersTestSuite) TearDownTest() {
	// Clean up test data
	suite.db.Exec("DELETE FROM comment_kudos WHERE comment_id IN (SELECT id FROM comments WHERE work_id = $1)", suite.testWorkID)
	suite.db.Exec("DELETE FROM comments WHERE work_id = $1", suite.testWorkID)
	suite.db.Exec("DELETE FROM works WHERE id = $1", suite.testWorkID)
	suite.db.Exec("DELETE FROM user_pseudonyms WHERE user_id = $1", suite.testUserID)
	suite.db.Exec("DELETE FROM users WHERE id = $1", suite.testUserID)
}

func (suite *CommentHandlersTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *CommentHandlersTestSuite) TestCreateComment_Success() {
	requestBody := models.CommentCreateRequest{
		WorkID:      &suite.testWorkID,
		Content:     "This is a test comment",
		PseudonymID: &suite.testPseudID,
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/works/%s/comments", suite.testWorkID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		suite.T().Logf("Response body: %s", w.Body.String())
	}
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	// Verify comment was created in database
	var commentCount int
	suite.db.QueryRow("SELECT COUNT(*) FROM comments WHERE work_id = $1", suite.testWorkID).Scan(&commentCount)
	assert.Equal(suite.T(), 1, commentCount)
}

func (suite *CommentHandlersTestSuite) TestCreateComment_GuestComment() {
	// Remove authentication for guest comment test
	router := gin.New()
	api := router.Group("/api/v1")
	api.POST("/works/:id/comments", suite.workService.CreateComment)

	requestBody := models.CommentCreateRequest{
		WorkID:    &suite.testWorkID,
		Content:   "This is a guest comment",
		GuestName: commentStringPtr("Guest User"),
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/works/%s/comments", suite.testWorkID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Verify guest comment was created
	var guestName sql.NullString
	suite.db.QueryRow("SELECT guest_name FROM comments WHERE work_id = $1", suite.testWorkID).Scan(&guestName)
	assert.True(suite.T(), guestName.Valid)
	assert.Equal(suite.T(), "Guest User", guestName.String)
}

func (suite *CommentHandlersTestSuite) TestCreateComment_ThreadedReply() {
	// Create parent comment first
	parentComment := suite.createTestComment("Parent comment", nil)

	// Create reply
	requestBody := models.CommentCreateRequest{
		WorkID:          &suite.testWorkID,
		Content:         "This is a reply",
		PseudonymID:     &suite.testPseudID,
		ParentCommentID: &parentComment.ID,
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/works/%s/comments", suite.testWorkID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Verify reply was created with correct parent
	var parentID sql.NullString
	suite.db.QueryRow("SELECT parent_comment_id FROM comments WHERE content = 'This is a reply'").Scan(&parentID)
	assert.True(suite.T(), parentID.Valid)
	assert.Equal(suite.T(), parentComment.ID.String(), parentID.String)
}

func (suite *CommentHandlersTestSuite) TestGetWorkComments_Success() {
	// Create test comments
	comment1 := suite.createTestComment("First comment", nil)
	_ = suite.createTestComment("Second comment", nil)
	_ = suite.createTestComment("Reply to first", &comment1.ID)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/works/%s/comments", suite.testWorkID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	comments := response["comments"].([]interface{})
	assert.Len(suite.T(), comments, 3)

	// Check pagination info
	assert.Equal(suite.T(), float64(3), response["total_count"])
	assert.Equal(suite.T(), float64(1), response["page"])
}

func (suite *CommentHandlersTestSuite) TestGetWorkComments_ThreadedView() {
	// Create threaded comments
	parent := suite.createTestComment("Parent comment", nil)
	reply1 := suite.createTestComment("Reply 1", &parent.ID)
	_ = suite.createTestComment("Reply 2", &parent.ID)
	_ = suite.createTestComment("Nested reply", &reply1.ID)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/works/%s/comments?threaded=true", suite.testWorkID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), true, response["threaded"])
	comments := response["comments"].([]interface{})

	// Should have one root comment with replies nested
	assert.Len(suite.T(), comments, 1)
	rootComment := comments[0].(map[string]interface{})
	replies := rootComment["replies"].([]interface{})
	assert.Len(suite.T(), replies, 2)
}

func (suite *CommentHandlersTestSuite) TestUpdateComment_Success() {
	comment := suite.createTestComment("Original content", nil)

	requestBody := models.CommentUpdateRequest{
		Content: "Updated content",
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/comments/%s", comment.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify content was updated
	var updatedContent string
	suite.db.QueryRow("SELECT content FROM comments WHERE id = $1", comment.ID).Scan(&updatedContent)
	assert.Equal(suite.T(), "Updated content", updatedContent)
}

func (suite *CommentHandlersTestSuite) TestUpdateComment_Unauthorized() {
	// Create comment by different user
	otherUserID := uuid.New()
	_, err := suite.db.Exec(`
		INSERT INTO users (id, username, email, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, 'hashed', true, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, otherUserID, fmt.Sprintf("otheruser_%s", otherUserID.String()[:8]), fmt.Sprintf("other_%s@example.com", otherUserID.String()[:8]))
	suite.Require().NoError(err)

	// Create pseudonym for other user
	otherPseudID := uuid.New()
	_, err = suite.db.Exec(`
		INSERT INTO user_pseudonyms (id, user_id, name, is_default, created_at)
		VALUES ($1, $2, $3, true, NOW())
		ON CONFLICT (id) DO NOTHING
	`, otherPseudID, otherUserID, fmt.Sprintf("OtherPseud_%s", otherUserID.String()[:8]))
	suite.Require().NoError(err)

	comment := suite.createTestCommentByUserAndPseud("Comment by other user", nil, otherUserID, otherPseudID)

	requestBody := models.CommentUpdateRequest{
		Content: "Trying to update",
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/comments/%s", comment.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)

	// Cleanup
	suite.db.Exec("DELETE FROM comments WHERE user_id = $1", otherUserID)
	suite.db.Exec("DELETE FROM user_pseudonyms WHERE user_id = $1", otherUserID)
	suite.db.Exec("DELETE FROM users WHERE id = $1", otherUserID)
}

func (suite *CommentHandlersTestSuite) TestDeleteComment_Success() {
	comment := suite.createTestComment("Comment to delete", nil)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/comments/%s", comment.ID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify comment was soft deleted
	var isDeleted bool
	suite.db.QueryRow("SELECT is_deleted FROM comments WHERE id = $1", comment.ID).Scan(&isDeleted)
	assert.True(suite.T(), isDeleted)
}

func (suite *CommentHandlersTestSuite) TestGiveCommentKudos_Success() {
	comment := suite.createTestComment("Comment to give kudos", nil)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/comments/%s/kudos", comment.ID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Verify kudos was created
	var kudosCount int
	suite.db.QueryRow("SELECT COUNT(*) FROM comment_kudos WHERE comment_id = $1", comment.ID).Scan(&kudosCount)
	assert.Equal(suite.T(), 1, kudosCount)
}

func (suite *CommentHandlersTestSuite) TestGiveCommentKudos_Duplicate() {
	comment := suite.createTestComment("Comment to give kudos", nil)

	// Give kudos first time
	req1, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/comments/%s/kudos", comment.ID), nil)
	w1 := httptest.NewRecorder()
	suite.router.ServeHTTP(w1, req1)
	assert.Equal(suite.T(), http.StatusCreated, w1.Code)

	// Try to give kudos again
	req2, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/comments/%s/kudos", comment.ID), nil)
	w2 := httptest.NewRecorder()
	suite.router.ServeHTTP(w2, req2)
	assert.Equal(suite.T(), http.StatusConflict, w2.Code)

	// Verify only one kudos exists
	var kudosCount int
	suite.db.QueryRow("SELECT COUNT(*) FROM comment_kudos WHERE comment_id = $1", comment.ID).Scan(&kudosCount)
	assert.Equal(suite.T(), 1, kudosCount)
}

func (suite *CommentHandlersTestSuite) TestGiveCommentKudos_GuestUser() {
	comment := suite.createTestComment("Comment for guest kudos", nil)

	// Remove authentication for guest test
	router := gin.New()
	api := router.Group("/api/v1")
	api.POST("/comments/:commentId/kudos", suite.workService.GiveCommentKudos)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/comments/%s/kudos", comment.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Verify guest kudos was created
	var guestSession sql.NullString
	suite.db.QueryRow("SELECT guest_session FROM comment_kudos WHERE comment_id = $1", comment.ID).Scan(&guestSession)
	assert.True(suite.T(), guestSession.Valid)
}

func (suite *CommentHandlersTestSuite) TestCreateComment_ValidationErrors() {
	testCases := []struct {
		name           string
		requestBody    models.CommentCreateRequest
		expectedStatus int
	}{
		{
			name: "Missing content",
			requestBody: models.CommentCreateRequest{
				WorkID:      &suite.testWorkID,
				PseudonymID: &suite.testPseudID,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing work and chapter ID",
			requestBody: models.CommentCreateRequest{
				Content:     "Valid content",
				PseudonymID: &suite.testPseudID,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing pseudonym and guest name",
			requestBody: models.CommentCreateRequest{
				WorkID:  &suite.testWorkID,
				Content: "Valid content",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.requestBody)
			req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/works/%s/comments", suite.testWorkID), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// Helper functions

func (suite *CommentHandlersTestSuite) createTestComment(content string, parentID *uuid.UUID) *models.Comment {
	return suite.createTestCommentByUser(content, parentID, suite.testUserID)
}

func (suite *CommentHandlersTestSuite) createTestCommentByUser(content string, parentID *uuid.UUID, userID uuid.UUID) *models.Comment {
	commentID := uuid.New()

	_, err := suite.db.Exec(`
		INSERT INTO comments (id, work_id, user_id, pseudonym_id, parent_comment_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, commentID, suite.testWorkID, userID, suite.testPseudID, parentID, content)
	suite.Require().NoError(err)

	return &models.Comment{
		ID:              commentID,
		WorkID:          &suite.testWorkID,
		UserID:          &userID,
		PseudonymID:     &suite.testPseudID,
		ParentCommentID: parentID,
		Content:         content,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func (suite *CommentHandlersTestSuite) createTestCommentByUserAndPseud(content string, parentID *uuid.UUID, userID uuid.UUID, pseudID uuid.UUID) *models.Comment {
	commentID := uuid.New()

	_, err := suite.db.Exec(`
		INSERT INTO comments (id, work_id, user_id, pseudonym_id, parent_comment_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, commentID, suite.testWorkID, userID, pseudID, parentID, content)
	suite.Require().NoError(err)

	return &models.Comment{
		ID:              commentID,
		WorkID:          &suite.testWorkID,
		UserID:          &userID,
		PseudonymID:     &pseudID,
		ParentCommentID: parentID,
		Content:         content,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func commentStringPtr(s string) *string {
	return &s
}

func TestCommentHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(CommentHandlersTestSuite))
}
