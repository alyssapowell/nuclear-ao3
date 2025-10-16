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

type CollectionsTestSuite struct {
	suite.Suite
	config      *TestDBConfig
	db          *sql.DB
	workService *WorkService
	router      *gin.Engine

	userID       uuid.UUID
	testWorkID   uuid.UUID
	collectionID uuid.UUID
}

func (suite *CollectionsTestSuite) SetupSuite() {
	// Setup database using test utilities
	suite.config = SetupTestDB(suite.T())
	suite.db = suite.config.DB

	// Clean up any existing test data
	suite.config.CleanupTestData()

	suite.workService = &WorkService{db: suite.db, redis: nil}

	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Create test user and work
	suite.createTestData()

	// Setup routes
	api := suite.router.Group("/api/v1")
	{
		collections := api.Group("/collections")
		{
			collections.GET("", suite.workService.SearchCollections)
			collections.GET("/:collection_id", suite.workService.GetCollection)
			collections.GET("/:collection_id/works", suite.workService.GetCollectionWorks)
			collections.POST("", suite.withAuth(), suite.workService.CreateCollection)
			collections.PUT("/:collection_id", suite.withAuth(), suite.workService.UpdateCollection)
			collections.DELETE("/:collection_id", suite.withAuth(), suite.workService.DeleteCollection)
			collections.POST("/:collection_id/works/:work_id", suite.withAuth(), suite.workService.AddWorkToCollection)
			collections.DELETE("/:collection_id/works/:work_id", suite.withAuth(), suite.workService.RemoveWorkFromCollection)
		}

		my := api.Group("/my")
		{
			my.GET("/collections", suite.withAuth(), suite.workService.GetMyCollections)
		}
	}
}

func (suite *CollectionsTestSuite) TearDownSuite() {
	if suite.config != nil {
		suite.config.CleanupTestData()
	}
}

func (suite *CollectionsTestSuite) createTestData() {
	// Create test user using test utilities with unique name
	var err error
	uniqueName := fmt.Sprintf("colltest_%d", time.Now().UnixNano())
	suite.userID, _, err = suite.config.CreateTestUser(uniqueName, uniqueName+"@test.com")
	suite.Require().NoError(err)

	// Create test work using test utilities
	suite.testWorkID, err = suite.config.CreateTestWork(suite.userID, "Test Work for Collection", "published")
	suite.Require().NoError(err)
}

func (suite *CollectionsTestSuite) withAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", suite.userID.String())
		c.Next()
	}
}

func (suite *CollectionsTestSuite) TestCreateCollection_Success() {
	requestBody := map[string]interface{}{
		"name":         "test-collection",
		"title":        "Test Collection",
		"description":  "A test collection",
		"is_open":      true,
		"is_moderated": false,
		"is_anonymous": false,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/collections", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	collection, ok := response["collection"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "test-collection", collection["name"])
	assert.Equal(suite.T(), "Test Collection", collection["title"])

	// Store collection ID for cleanup
	collectionIDStr, ok := collection["id"].(string)
	assert.True(suite.T(), ok)
	suite.collectionID, err = uuid.Parse(collectionIDStr)
	assert.NoError(suite.T(), err)
}

func (suite *CollectionsTestSuite) TestCreateCollection_DuplicateName() {
	// First collection
	requestBody := map[string]interface{}{
		"name":  "duplicate-test",
		"title": "First Collection",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/collections", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Get collection ID for cleanup
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	collection := response["collection"].(map[string]interface{})
	firstCollectionID, _ := uuid.Parse(collection["id"].(string))

	// Try to create duplicate
	req = httptest.NewRequest("POST", "/api/v1/collections", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusConflict, w.Code)

	// Cleanup
	suite.db.Exec("DELETE FROM collections WHERE id = $1", firstCollectionID)
}

func (suite *CollectionsTestSuite) TestSearchCollections() {
	// Create a test collection
	collectionID := uuid.New()
	_, err := suite.db.Exec(`
		INSERT INTO collections (id, name, title, description, user_id, is_open, is_moderated, is_anonymous, work_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())`,
		collectionID, "searchable-collection", "Searchable Collection", "A collection for testing search",
		suite.userID, true, false, false, 0)
	suite.Require().NoError(err)

	req := httptest.NewRequest("GET", "/api/v1/collections?q=searchable", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	collections, ok := response["collections"].([]interface{})
	assert.True(suite.T(), ok)
	assert.GreaterOrEqual(suite.T(), len(collections), 1)

	// Check pagination
	pagination, ok := response["pagination"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), float64(1), pagination["page"])

	// Cleanup
	suite.db.Exec("DELETE FROM collections WHERE id = $1", collectionID)
}

func (suite *CollectionsTestSuite) TestGetCollection() {
	// Create a test collection
	collectionID := uuid.New()
	_, err := suite.db.Exec(`
		INSERT INTO collections (id, name, title, description, user_id, is_open, is_moderated, is_anonymous, work_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())`,
		collectionID, "get-test-collection", "Get Test Collection", "Description",
		suite.userID, true, false, false, 0)
	suite.Require().NoError(err)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/collections/%s", collectionID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	collection, ok := response["collection"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "get-test-collection", collection["name"])

	maintainer, ok := response["maintainer"].(string)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "colltest", maintainer)

	// Cleanup
	suite.db.Exec("DELETE FROM collections WHERE id = $1", collectionID)
}

func (suite *CollectionsTestSuite) TestUpdateCollection() {
	// Create a test collection first
	collectionID := uuid.New()
	_, err := suite.db.Exec(`
		INSERT INTO collections (id, name, title, description, user_id, is_open, is_moderated, is_anonymous, work_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())`,
		collectionID, "update-collection", "Original Title", "Original description",
		suite.userID, true, false, false, 0)
	suite.Require().NoError(err)

	requestBody := map[string]interface{}{
		"title":       "Updated Title",
		"description": "Updated description",
		"is_open":     false,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/collections/%s", collectionID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	collection, ok := response["collection"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "Updated Title", collection["title"])
	assert.Equal(suite.T(), "Updated description", collection["description"])
	assert.Equal(suite.T(), false, collection["is_open"])

	// Cleanup
	suite.db.Exec("DELETE FROM collections WHERE id = $1", collectionID)
}

func (suite *CollectionsTestSuite) TestAddWorkToCollection() {
	// Create a test collection
	collectionID := uuid.New()
	_, err := suite.db.Exec(`
		INSERT INTO collections (id, name, title, description, user_id, is_open, is_moderated, is_anonymous, work_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())`,
		collectionID, "work-collection", "Work Collection", "For testing work addition",
		suite.userID, true, false, false, 0)
	suite.Require().NoError(err)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/collections/%s/works/%s", collectionID, suite.testWorkID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Work added to collection", response["message"])

	// Verify work was added
	var exists bool
	err = suite.db.QueryRow("SELECT EXISTS(SELECT 1 FROM collection_items WHERE collection_id = $1 AND work_id = $2)",
		collectionID, suite.testWorkID).Scan(&exists)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// Cleanup
	suite.db.Exec("DELETE FROM collection_items WHERE collection_id = $1", collectionID)
	suite.db.Exec("DELETE FROM collections WHERE id = $1", collectionID)
}

func (suite *CollectionsTestSuite) TestGetCollectionWorks() {
	// Create a test collection
	collectionID := uuid.New()
	_, err := suite.db.Exec(`
		INSERT INTO collections (id, name, title, description, user_id, is_open, is_moderated, is_anonymous, work_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())`,
		collectionID, "works-collection", "Works Collection", "Collection with works",
		suite.userID, true, false, false, 1)
	suite.Require().NoError(err)

	// Add work to collection
	_, err = suite.db.Exec(`
		INSERT INTO collection_items (id, collection_id, work_id, added_by, is_approved, added_at)
		VALUES ($1, $2, $3, $4, $5, NOW())`,
		uuid.New(), collectionID, suite.testWorkID, suite.userID, true)
	suite.Require().NoError(err)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/collections/%s/works", collectionID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	works, ok := response["works"].([]interface{})
	assert.True(suite.T(), ok)
	assert.GreaterOrEqual(suite.T(), len(works), 1)

	// Check first work
	firstWork := works[0].(map[string]interface{})
	work := firstWork["work"].(map[string]interface{})
	assert.Equal(suite.T(), "Test Work for Collection", work["title"])
	assert.Equal(suite.T(), true, firstWork["is_approved"])

	// Cleanup
	suite.db.Exec("DELETE FROM collection_items WHERE collection_id = $1", collectionID)
	suite.db.Exec("DELETE FROM collections WHERE id = $1", collectionID)
}

func (suite *CollectionsTestSuite) TestRemoveWorkFromCollection() {
	// Create collection and add work
	collectionID := uuid.New()
	_, err := suite.db.Exec(`
		INSERT INTO collections (id, name, title, description, user_id, is_open, is_moderated, is_anonymous, work_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())`,
		collectionID, "remove-collection", "Remove Collection", "For testing removal",
		suite.userID, true, false, false, 1)
	suite.Require().NoError(err)

	// Add work to collection
	itemID := uuid.New()
	_, err = suite.db.Exec(`
		INSERT INTO collection_items (id, collection_id, work_id, added_by, is_approved, added_at)
		VALUES ($1, $2, $3, $4, $5, NOW())`,
		itemID, collectionID, suite.testWorkID, suite.userID, true)
	suite.Require().NoError(err)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/collections/%s/works/%s", collectionID, suite.testWorkID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Work removed from collection successfully", response["message"])

	// Verify work was removed
	var exists bool
	err = suite.db.QueryRow("SELECT EXISTS(SELECT 1 FROM collection_items WHERE id = $1)", itemID).Scan(&exists)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)

	// Cleanup
	suite.db.Exec("DELETE FROM collections WHERE id = $1", collectionID)
}

func (suite *CollectionsTestSuite) TestDeleteCollection() {
	// Create a test collection
	collectionID := uuid.New()
	_, err := suite.db.Exec(`
		INSERT INTO collections (id, name, title, description, user_id, is_open, is_moderated, is_anonymous, work_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())`,
		collectionID, "delete-collection", "Delete Collection", "To be deleted",
		suite.userID, true, false, false, 0)
	suite.Require().NoError(err)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/collections/%s", collectionID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Collection deleted successfully", response["message"])

	// Verify collection was deleted
	var exists bool
	err = suite.db.QueryRow("SELECT EXISTS(SELECT 1 FROM collections WHERE id = $1)", collectionID).Scan(&exists)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

func (suite *CollectionsTestSuite) TestGetMyCollections() {
	// Create a test collection
	collectionID := uuid.New()
	_, err := suite.db.Exec(`
		INSERT INTO collections (id, name, title, description, user_id, is_open, is_moderated, is_anonymous, work_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())`,
		collectionID, "my-collection", "My Collection", "My personal collection",
		suite.userID, true, false, false, 0)
	suite.Require().NoError(err)

	req := httptest.NewRequest("GET", "/api/v1/my/collections", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	collections, ok := response["collections"].([]interface{})
	assert.True(suite.T(), ok)
	assert.GreaterOrEqual(suite.T(), len(collections), 1)

	// Check first collection
	firstCollection := collections[0].(map[string]interface{})
	collection := firstCollection["collection"].(map[string]interface{})
	assert.Equal(suite.T(), "my-collection", collection["name"])
	assert.Equal(suite.T(), float64(0), firstCollection["pending_count"]) // No pending items

	// Cleanup
	suite.db.Exec("DELETE FROM collections WHERE id = $1", collectionID)
}

func TestCollectionsTestSuite(t *testing.T) {
	suite.Run(t, new(CollectionsTestSuite))
}
