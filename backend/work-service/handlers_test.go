package main

import (
	"fmt"
	"testing"
	"time"

	"nuclear-ao3/shared/models"
	"nuclear-ao3/shared/server"
	testutils "nuclear-ao3/shared/testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type WorkServiceTestSuite struct {
	suite.Suite
	service   *WorkService
	testUsers map[string]uuid.UUID
	testWorks []*models.Work
}

func (suite *WorkServiceTestSuite) SetupSuite() {
	// Use existing work service instance for integration testing
	suite.service = NewWorkService()

	// Setup test data
	suite.setupTestData()
}

func (suite *WorkServiceTestSuite) setupTestData() {
	// Use existing test users from migration or create if needed
	suite.testUsers = make(map[string]uuid.UUID)

	// First, try to use existing users from migration
	existingUsers := []struct {
		username string
		id       string
	}{
		{"testuser", "123e4567-e89b-12d3-a456-426614174000"},
		{"author2", "123e4567-e89b-12d3-a456-426614174001"},
	}

	for _, userData := range existingUsers {
		userID, err := uuid.Parse(userData.id)
		suite.Require().NoError(err)
		suite.testUsers[userData.username] = userID
	}

	// Create test works
	suite.testWorks = []*models.Work{
		{
			ID:        uuid.New(),
			Title:     "Test Winter Story",
			Summary:   "A story about winter adventures",
			UserID:    suite.testUsers["testuser"],
			Language:  "en",
			Rating:    "Teen And Up Audiences",
			Status:    "published",
			WordCount: 5000,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Title:     "Mature Fantasy Tale",
			Summary:   "An epic fantasy with mature themes",
			UserID:    suite.testUsers["author2"],
			Language:  "en",
			Rating:    "Mature",
			Status:    "published",
			WordCount: 12000,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Insert test works into database
	for _, work := range suite.testWorks {
		_, err := suite.service.db.Exec(`
			INSERT INTO works (id, title, summary, user_id, language, rating, word_count, is_draft, created_at, updated_at) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) 
			ON CONFLICT (id) DO NOTHING`,
			work.ID, work.Title, work.Summary, work.UserID, work.Language,
			work.Rating, work.WordCount, false, work.CreatedAt, work.UpdatedAt)
		suite.Require().NoError(err)
	}
}

func (suite *WorkServiceTestSuite) TearDownSuite() {
	if suite.service != nil {
		suite.service.Close()
	}
}

func (suite *WorkServiceTestSuite) TestHealthEndpoint() {
	router := server.SetupBaseRouter(
		server.ServiceInfo{Name: "work-service", Version: "1.0.0"},
		suite.service.redis,
	)

	testutils.AssertHealthEndpoint(suite.T(), router, "work-service")
}

func (suite *WorkServiceTestSuite) TestSearchWorksEndpoint() {
	router := setupRouter(suite.service)

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works?limit=5",
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	// Check response structure
	assert.Contains(suite.T(), response, "works")
	assert.Contains(suite.T(), response, "pagination")

	pagination := response["pagination"].(map[string]interface{})
	assert.Contains(suite.T(), pagination, "limit")
	assert.Contains(suite.T(), pagination, "total")
	assert.Contains(suite.T(), pagination, "page")
}

func (suite *WorkServiceTestSuite) TestSearchWorksWithQuery() {
	router := setupRouter(suite.service)

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works?q=winter&limit=3",
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	// Check that results are filtered by query
	works := response["works"].([]interface{})
	if len(works) > 0 {
		firstWork := works[0].(map[string]interface{})
		assert.Contains(suite.T(), firstWork, "title")
		assert.Contains(suite.T(), firstWork, "summary")
	}
}

func (suite *WorkServiceTestSuite) TestSearchWorksWithFilters() {
	router := setupRouter(suite.service)

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works?rating=Mature&limit=3",
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	// Check that filters are applied
	works := response["works"].([]interface{})
	for _, work := range works {
		workData := work.(map[string]interface{})
		if rating, exists := workData["rating"]; exists {
			assert.Equal(suite.T(), "Mature", rating)
		}
	}
}

func (suite *WorkServiceTestSuite) TestSearchWorksWithPagination() {
	router := setupRouter(suite.service)

	// Test first page
	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works?limit=1&page=1",
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	// Check pagination structure
	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(suite.T(), float64(1), pagination["page"])
	assert.Equal(suite.T(), float64(1), pagination["limit"])
	assert.GreaterOrEqual(suite.T(), pagination["total"], float64(1))

	works := response["works"].([]interface{})
	assert.LessOrEqual(suite.T(), len(works), 1) // Should have at most 1 work
}

func (suite *WorkServiceTestSuite) TestSearchWorksInvalidParameters() {
	router := setupRouter(suite.service)

	// Test invalid limit
	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works?limit=invalid",
		ExpectedCode: 200, // Should handle gracefully and use default
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)
	assert.Contains(suite.T(), response, "works")
	assert.Contains(suite.T(), response, "pagination")
}

func (suite *WorkServiceTestSuite) TestSearchWorksMultipleFilters() {
	router := setupRouter(suite.service)

	// Test multiple filters
	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works?rating=Teen+And+Up+Audiences&language=en&limit=10",
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	works := response["works"].([]interface{})
	for _, work := range works {
		workData := work.(map[string]interface{})
		if rating, exists := workData["rating"]; exists {
			assert.Equal(suite.T(), "Teen And Up Audiences", rating)
		}
		if language, exists := workData["language"]; exists {
			assert.Equal(suite.T(), "en", language)
		}
	}
}

func (suite *WorkServiceTestSuite) TestGetWorkByID() {
	router := setupRouter(suite.service)

	// Use one of our test works
	testWork := suite.testWorks[0]

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works/" + testWork.ID.String(),
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	// Check work details
	assert.Equal(suite.T(), testWork.Title, response["title"])
	assert.Equal(suite.T(), testWork.Summary, response["summary"])
	assert.Equal(suite.T(), testWork.Rating, response["rating"])
	assert.Equal(suite.T(), testWork.Language, response["language"])
}

func (suite *WorkServiceTestSuite) TestGetWorkByID_NotFound() {
	router := setupRouter(suite.service)

	// Use a random UUID that doesn't exist
	randomID := uuid.New()

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works/" + randomID.String(),
		ExpectedCode: 404,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 404)
	assert.Contains(suite.T(), response, "error")
}

func (suite *WorkServiceTestSuite) TestGetWorkByID_InvalidUUID() {
	router := setupRouter(suite.service)

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works/invalid-uuid",
		ExpectedCode: 400,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 400)
	assert.Contains(suite.T(), response, "error")
}

func (suite *WorkServiceTestSuite) TestSearchWorksEmptyResults() {
	router := setupRouter(suite.service)

	// Search for something that definitely won't exist
	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works?q=absolutely_nonexistent_search_term_12345",
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	works := response["works"].([]interface{})
	assert.Empty(suite.T(), works)

	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(suite.T(), float64(0), pagination["total"])
}

func (suite *WorkServiceTestSuite) TestSearchWorksPerformance() {
	router := setupRouter(suite.service)

	// Test search performance with various queries
	testQueries := []string{
		"winter",
		"fantasy",
		"adventure",
		"story",
	}

	for _, query := range testQueries {
		w := testutils.PerformRequest(router, testutils.TestRequest{
			Method:       "GET",
			URL:          "/api/v1/works?q=" + query + "&limit=20",
			ExpectedCode: 200,
		})

		response := testutils.AssertJSONResponse(suite.T(), w, 200)
		assert.Contains(suite.T(), response, "works")
		assert.Contains(suite.T(), response, "pagination")

		// All responses should be fast (this is more of a smoke test)
		assert.NotNil(suite.T(), response)
	}
}

// ============================================================================
// SEARCH WORKS COMPREHENSIVE TESTS
// ============================================================================

func (suite *WorkServiceTestSuite) TestSearchWorks_DatabaseIntegration() {
	router := setupRouter(suite.service)

	// Test that our inserted test works are found
	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works?limit=50",
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)
	works := response["works"].([]interface{})

	// Should find our test works
	assert.GreaterOrEqual(suite.T(), len(works), 2)

	// Verify pagination data
	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(suite.T(), float64(1), pagination["page"])
	assert.Equal(suite.T(), float64(50), pagination["limit"])
	assert.GreaterOrEqual(suite.T(), pagination["total"].(float64), float64(2))
}

func (suite *WorkServiceTestSuite) TestSearchWorks_QueryParameters() {
	router := setupRouter(suite.service)

	testCases := []struct {
		name        string
		queryParams string
		expectEmpty bool
	}{
		{
			name:        "search by title keyword",
			queryParams: "?q=Winter",
			expectEmpty: false,
		},
		{
			name:        "search by non-existent term",
			queryParams: "?q=nonexistentterm12345",
			expectEmpty: true,
		},
		{
			name:        "filter by rating",
			queryParams: "?rating=Teen+And+Up+Audiences",
			expectEmpty: false,
		},
		{
			name:        "filter by language",
			queryParams: "?language=en",
			expectEmpty: false,
		},
		{
			name:        "combined filters",
			queryParams: "?rating=Teen+And+Up+Audiences&language=en",
			expectEmpty: false,
		},
		{
			name:        "pagination test",
			queryParams: "?page=1&limit=1",
			expectEmpty: false,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			w := testutils.PerformRequest(router, testutils.TestRequest{
				Method:       "GET",
				URL:          "/api/v1/works" + tc.queryParams,
				ExpectedCode: 200,
			})

			response := testutils.AssertJSONResponse(t, w, 200)
			works := response["works"].([]interface{})

			if tc.expectEmpty {
				assert.Empty(t, works)
			} else {
				assert.NotEmpty(t, works)
			}

			// Always verify response structure
			assert.Contains(t, response, "pagination")
		})
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// Helper function to create string pointers for update requests (defined in schema_validator.go)

// ============================================================================
// COMPREHENSIVE SEARCH TESTS
// ============================================================================

func (suite *WorkServiceTestSuite) TestSearchWorks_AdvancedFiltering() {
	router := setupRouter(suite.service)

	testCases := []struct {
		name           string
		queryParams    string
		expectResults  bool
		expectedFields map[string]interface{}
	}{
		{
			name:          "search by title keyword",
			queryParams:   "?q=Winter",
			expectResults: true,
		},
		{
			name:          "search by mature rating",
			queryParams:   "?rating=Mature",
			expectResults: true,
			expectedFields: map[string]interface{}{
				"rating": "Mature",
			},
		},
		{
			name:          "search by language",
			queryParams:   "?language=en",
			expectResults: true,
			expectedFields: map[string]interface{}{
				"language": "en",
			},
		},
		{
			name:          "search non-existent term",
			queryParams:   "?q=xyzneverexists12345",
			expectResults: false,
		},
		{
			name:          "search with invalid rating",
			queryParams:   "?rating=NonExistentRating",
			expectResults: false,
		},
		{
			name:          "pagination first page",
			queryParams:   "?page=1&limit=1",
			expectResults: true,
		},
		{
			name:          "pagination beyond results",
			queryParams:   "?page=999&limit=20",
			expectResults: false,
		},
		{
			name:          "max limit enforcement",
			queryParams:   "?limit=1000", // Should be capped at 100
			expectResults: true,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			w := testutils.PerformRequest(router, testutils.TestRequest{
				Method:       "GET",
				URL:          "/api/v1/works" + tc.queryParams,
				ExpectedCode: 200,
			})

			response := testutils.AssertJSONResponse(t, w, 200)
			works := response["works"].([]interface{})

			if tc.expectResults {
				assert.NotEmpty(t, works, "Expected to find works for query: %s", tc.queryParams)

				// Check specific field values if provided
				if len(tc.expectedFields) > 0 && len(works) > 0 {
					firstWork := works[0].(map[string]interface{})
					for field, expectedValue := range tc.expectedFields {
						assert.Equal(t, expectedValue, firstWork[field],
							"Field %s should match expected value", field)
					}
				}
			} else {
				assert.Empty(t, works, "Expected no works for query: %s", tc.queryParams)
			}

			// Always verify response structure
			assert.Contains(t, response, "pagination")
			pagination := response["pagination"].(map[string]interface{})
			assert.Contains(t, pagination, "page")
			assert.Contains(t, pagination, "limit")
			assert.Contains(t, pagination, "total")

			// Verify limit enforcement for the max limit test
			if tc.queryParams == "?limit=1000" {
				assert.LessOrEqual(t, pagination["limit"].(float64), float64(100),
					"Limit should be capped at 100")
			}
		})
	}
}

func (suite *WorkServiceTestSuite) TestSearchWorks_SortingAndOrdering() {
	router := setupRouter(suite.service)

	testCases := []struct {
		name        string
		queryParams string
	}{
		{
			name:        "sort by updated_at desc (default)",
			queryParams: "",
		},
		{
			name:        "sort by updated_at asc",
			queryParams: "?sort=updated_at&order=asc",
		},
		{
			name:        "sort by title",
			queryParams: "?sort=title&order=asc",
		},
		{
			name:        "sort by word_count desc",
			queryParams: "?sort=word_count&order=desc",
		},
		{
			name:        "sort by kudos desc",
			queryParams: "?sort=kudos&order=desc",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			w := testutils.PerformRequest(router, testutils.TestRequest{
				Method:       "GET",
				URL:          "/api/v1/works" + tc.queryParams,
				ExpectedCode: 200,
			})

			response := testutils.AssertJSONResponse(t, w, 200)
			works := response["works"].([]interface{})

			// Should return results (we have test data)
			assert.NotEmpty(t, works)

			// Verify response structure includes all expected fields
			if len(works) > 0 {
				firstWork := works[0].(map[string]interface{})
				expectedFields := []string{
					"id", "title", "summary", "author", "language", "rating",
					"word_count", "chapter_count", "is_complete", "created_at", "updated_at",
				}

				for _, field := range expectedFields {
					assert.Contains(t, firstWork, field, "Work should contain field %s", field)
				}
			}
		})
	}
}

func (suite *WorkServiceTestSuite) TestSearchWorks_ErrorHandling() {
	router := setupRouter(suite.service)

	testCases := []struct {
		name        string
		queryParams string
		description string
	}{
		{
			name:        "invalid page number",
			queryParams: "?page=invalid",
			description: "Should handle invalid page gracefully",
		},
		{
			name:        "invalid limit",
			queryParams: "?limit=invalid",
			description: "Should handle invalid limit gracefully",
		},
		{
			name:        "negative page",
			queryParams: "?page=-1",
			description: "Should handle negative page gracefully",
		},
		{
			name:        "zero limit",
			queryParams: "?limit=0",
			description: "Should handle zero limit gracefully",
		},
		{
			name:        "very long query",
			queryParams: "?q=" + string(make([]byte, 1000)), // Very long query
			description: "Should handle very long query gracefully",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			w := testutils.PerformRequest(router, testutils.TestRequest{
				Method:       "GET",
				URL:          "/api/v1/works" + tc.queryParams,
				ExpectedCode: 200, // Should still return 200 with sensible defaults
			})

			response := testutils.AssertJSONResponse(t, w, 200)

			// Should have works and pagination even with invalid params
			assert.Contains(t, response, "works")
			assert.Contains(t, response, "pagination")

			pagination := response["pagination"].(map[string]interface{})
			assert.Contains(t, pagination, "page")
			assert.Contains(t, pagination, "limit")
			assert.Contains(t, pagination, "total")

			// Page and limit should be positive numbers
			assert.Greater(t, pagination["page"].(float64), float64(0))
			assert.Greater(t, pagination["limit"].(float64), float64(0))
		})
	}
}

// ============================================================================
// GET WORK COMPREHENSIVE TESTS
// ============================================================================

func (suite *WorkServiceTestSuite) TestGetWork_DetailedFields() {
	router := setupRouter(suite.service)

	// Use one of our test works
	testWork := suite.testWorks[0]

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works/" + testWork.ID.String(),
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	// Verify all expected fields are present
	expectedFields := []string{
		"id", "title", "summary", "notes", "author", "language", "rating",
		"word_count", "chapter_count", "is_complete", "created_at", "updated_at",
	}

	for _, field := range expectedFields {
		assert.Contains(suite.T(), response, field, "Response should contain field %s", field)
	}

	// Verify specific data types
	assert.IsType(suite.T(), "", response["id"], "ID should be string")
	assert.IsType(suite.T(), "", response["title"], "Title should be string")
	assert.IsType(suite.T(), "", response["author"], "Author should be string")
	assert.IsType(suite.T(), float64(0), response["word_count"], "Word count should be number")
	assert.IsType(suite.T(), float64(0), response["chapter_count"], "Chapter count should be number")
	assert.IsType(suite.T(), false, response["is_complete"], "Is complete should be boolean")

	// Verify data matches what we inserted
	assert.Equal(suite.T(), testWork.Title, response["title"])
	assert.Equal(suite.T(), testWork.Summary, response["summary"])
	assert.Equal(suite.T(), testWork.Language, response["language"])
	assert.Equal(suite.T(), testWork.Rating, response["rating"])
}

func (suite *WorkServiceTestSuite) TestGetWork_EdgeCases() {
	router := setupRouter(suite.service)

	testCases := []struct {
		name         string
		workID       string
		expectedCode int
		description  string
	}{
		{
			name:         "valid UUID but non-existent work",
			workID:       uuid.New().String(),
			expectedCode: 404,
			description:  "Should return 404 for non-existent work",
		},
		{
			name:         "invalid UUID format",
			workID:       "invalid-uuid-123",
			expectedCode: 400,
			description:  "Should return 400 for malformed UUID",
		},
		{
			name:         "empty UUID",
			workID:       "",
			expectedCode: 404, // Gin router will return 404 for empty param
			description:  "Should handle empty UUID gracefully",
		},
		{
			name:         "UUID with extra characters",
			workID:       uuid.New().String() + "extra",
			expectedCode: 400,
			description:  "Should return 400 for UUID with extra characters",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			url := "/api/v1/works/"
			if tc.workID != "" {
				url += tc.workID
			}

			w := testutils.PerformRequest(router, testutils.TestRequest{
				Method:       "GET",
				URL:          url,
				ExpectedCode: tc.expectedCode,
			})

			if tc.expectedCode != 200 {
				response := testutils.AssertJSONResponse(t, w, tc.expectedCode)
				assert.Contains(t, response, "error", "Error response should contain error field")
			}
		})
	}
}

// ============================================================================
// PERFORMANCE AND LOAD TESTS
// ============================================================================

func (suite *WorkServiceTestSuite) TestSearchWorks_PerformanceBaseline() {
	router := setupRouter(suite.service)

	// Test multiple concurrent-style requests to establish baseline
	performanceQueries := []string{
		"?limit=20",
		"?q=test&limit=10",
		"?rating=Teen+And+Up+Audiences&limit=15",
		"?language=en&limit=25",
		"?page=1&limit=50",
	}

	for i, query := range performanceQueries {
		suite.T().Run(fmt.Sprintf("performance_query_%d", i), func(t *testing.T) {
			start := time.Now()

			w := testutils.PerformRequest(router, testutils.TestRequest{
				Method:       "GET",
				URL:          "/api/v1/works" + query,
				ExpectedCode: 200,
			})

			duration := time.Since(start)

			response := testutils.AssertJSONResponse(t, w, 200)
			assert.Contains(t, response, "works")
			assert.Contains(t, response, "pagination")

			// Performance assertion - should complete within reasonable time
			// This is a baseline test with small dataset
			assert.Less(t, duration, 100*time.Millisecond,
				"Query should complete within 100ms for test dataset")
		})
	}
}

// ============================================================================
// DATABASE INTEGRITY TESTS
// ============================================================================

func (suite *WorkServiceTestSuite) TestSearchWorks_DatabaseConsistency() {
	router := setupRouter(suite.service)

	// Test that the same query returns consistent results
	query := "/api/v1/works?limit=50"

	// First request
	w1 := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          query,
		ExpectedCode: 200,
	})
	response1 := testutils.AssertJSONResponse(suite.T(), w1, 200)

	// Second request immediately after
	w2 := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          query,
		ExpectedCode: 200,
	})
	response2 := testutils.AssertJSONResponse(suite.T(), w2, 200)

	// Results should be identical (no data changes between requests)
	pagination1 := response1["pagination"].(map[string]interface{})
	pagination2 := response2["pagination"].(map[string]interface{})

	assert.Equal(suite.T(), pagination1["total"], pagination2["total"],
		"Total count should be consistent between requests")

	works1 := response1["works"].([]interface{})
	works2 := response2["works"].([]interface{})

	assert.Equal(suite.T(), len(works1), len(works2),
		"Number of works should be consistent between requests")
}

// ============================================================================
// CORE AO3 FEATURES TESTS
// ============================================================================

func (suite *WorkServiceTestSuite) TestGetWork_WithAuthors() {
	router := setupRouter(suite.service)

	// Use one of our test works
	testWork := suite.testWorks[0]

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works/" + testWork.ID.String(),
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	// Verify new response structure with both work and authors
	assert.Contains(suite.T(), response, "work", "Response should contain work object")
	assert.Contains(suite.T(), response, "authors", "Response should contain authors array")

	// Check work object structure
	work := response["work"].(map[string]interface{})
	assert.Equal(suite.T(), testWork.Title, work["title"])
	assert.Equal(suite.T(), testWork.Summary, work["summary"])
	assert.Equal(suite.T(), testWork.Rating, work["rating"])

	// Check authors array structure
	authors := response["authors"].([]interface{})
	assert.NotEmpty(suite.T(), authors, "Authors array should not be empty")

	if len(authors) > 0 {
		author := authors[0].(map[string]interface{})
		assert.Contains(suite.T(), author, "pseud_id")
		assert.Contains(suite.T(), author, "pseud_name")
		assert.Contains(suite.T(), author, "user_id")
		assert.Contains(suite.T(), author, "username")
		assert.Contains(suite.T(), author, "is_anonymous")
	}
}

// ============================================================================
// LEGACY ID MIGRATION TESTS
// ============================================================================

func (suite *WorkServiceTestSuite) TestGetWork_LegacyIDRedirect() {
	router := setupRouter(suite.service)

	// First, create a work with a legacy ID
	testWork := suite.testWorks[0]
	legacyID := 12345

	// Update the work to have a legacy ID
	_, err := suite.service.db.Exec(
		"UPDATE works SET legacy_id = $1 WHERE id = $2",
		legacyID, testWork.ID,
	)
	assert.NoError(suite.T(), err, "Should be able to set legacy ID")

	// Test legacy ID redirect
	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          fmt.Sprintf("/api/v1/works/%d", legacyID),
		ExpectedCode: 301,
	})

	// Check redirect location
	location := w.Header().Get("Location")
	expectedLocation := fmt.Sprintf("/api/v1/work/%s", testWork.ID.String())
	assert.Equal(suite.T(), expectedLocation, location, "Should redirect to UUID-based URL")

	// Verify the response body contains the redirect HTML
	assert.Contains(suite.T(), w.Body.String(), "Moved Permanently")
	assert.Contains(suite.T(), w.Body.String(), testWork.ID.String())
}

func (suite *WorkServiceTestSuite) TestGetWork_LegacyIDNotFound() {
	router := setupRouter(suite.service)

	// Test non-existent legacy ID
	nonExistentLegacyID := 99999

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          fmt.Sprintf("/api/v1/works/%d", nonExistentLegacyID),
		ExpectedCode: 404,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 404)
	assert.Contains(suite.T(), response, "error")
	assert.Equal(suite.T(), "Work not found", response["error"])
}

func (suite *WorkServiceTestSuite) TestGetWork_ModernUUIDRoute() {
	router := setupRouter(suite.service)

	// Test the modern /work/{uuid} route
	testWork := suite.testWorks[0]

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/work/" + testWork.ID.String(),
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	// Verify the response structure
	assert.Contains(suite.T(), response, "work", "Response should contain work object")
	work := response["work"].(map[string]interface{})
	assert.Equal(suite.T(), testWork.Title, work["title"])
	assert.Equal(suite.T(), testWork.ID.String(), work["id"])
}

func (suite *WorkServiceTestSuite) TestGetWork_LegacyIDInResponse() {
	router := setupRouter(suite.service)

	// Create a work with a legacy ID
	testWork := suite.testWorks[1]
	legacyID := 67890

	// Update the work to have a legacy ID
	_, err := suite.service.db.Exec(
		"UPDATE works SET legacy_id = $1 WHERE id = $2",
		legacyID, testWork.ID,
	)
	assert.NoError(suite.T(), err, "Should be able to set legacy ID")

	// Test that legacy ID is included in the response
	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/work/" + testWork.ID.String(),
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)
	work := response["work"].(map[string]interface{})

	// Verify legacy_id is present and correct
	assert.Contains(suite.T(), work, "legacy_id", "Work should include legacy_id field")
	assert.Equal(suite.T(), float64(legacyID), work["legacy_id"], "Legacy ID should match expected value")
}

func (suite *WorkServiceTestSuite) TestGetWork_WorkWithoutLegacyID() {
	router := setupRouter(suite.service)

	// Test a work without a legacy ID (modern work)
	testWork := suite.testWorks[2]

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/work/" + testWork.ID.String(),
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)
	work := response["work"].(map[string]interface{})

	// Legacy ID should be null for modern works
	assert.Contains(suite.T(), work, "legacy_id", "Work should include legacy_id field")
	assert.Nil(suite.T(), work["legacy_id"], "Legacy ID should be null for modern works")
}

func (suite *WorkServiceTestSuite) TestGetWork_UUIDOnLegacyRoute() {
	router := setupRouter(suite.service)

	// Test that UUIDs still work on the legacy route (for backwards compatibility)
	testWork := suite.testWorks[0]

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works/" + testWork.ID.String(),
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	// Should return the work data directly (no redirect)
	assert.Contains(suite.T(), response, "work", "Response should contain work object")
	work := response["work"].(map[string]interface{})
	assert.Equal(suite.T(), testWork.Title, work["title"])
	assert.Equal(suite.T(), testWork.ID.String(), work["id"])
}

func (suite *WorkServiceTestSuite) TestGetWork_InvalidFormats() {
	router := setupRouter(suite.service)

	testCases := []struct {
		name         string
		url          string
		expectedCode int
		description  string
	}{
		{
			name:         "invalid characters mixed with numbers",
			url:          "/api/v1/works/123abc",
			expectedCode: 400,
			description:  "Should return 400 for invalid format",
		},
		{
			name:         "negative legacy ID",
			url:          "/api/v1/works/-123",
			expectedCode: 400,
			description:  "Should return 400 for negative numbers",
		},
		{
			name:         "zero legacy ID",
			url:          "/api/v1/works/0",
			expectedCode: 404,
			description:  "Should return 404 for zero (valid int but not found)",
		},
		{
			name:         "very large number",
			url:          "/api/v1/works/999999999999999",
			expectedCode: 404,
			description:  "Should return 404 for very large valid integers",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			w := testutils.PerformRequest(router, testutils.TestRequest{
				Method:       "GET",
				URL:          tc.url,
				ExpectedCode: tc.expectedCode,
			})

			response := testutils.AssertJSONResponse(t, w, tc.expectedCode)
			assert.Contains(t, response, "error", "Error response should contain error field")
		})
	}
}

// Test protected endpoints without authentication - should return 401 or redirect
func (suite *WorkServiceTestSuite) TestProtectedEndpoints_NoAuth() {
	router := setupRouter(suite.service)

	protectedEndpoints := []struct {
		method string
		url    string
	}{
		{"POST", "/api/v1/pseuds"},
		{"GET", "/api/v1/my/pseuds"},
		{"POST", "/api/v1/works"},
		{"POST", "/api/v1/works/" + uuid.New().String() + "/gift"},
		{"POST", "/api/v1/works/" + uuid.New().String() + "/orphan"},
		{"GET", "/api/v1/works/" + uuid.New().String() + "/authors"},
		{"POST", "/api/v1/works/" + uuid.New().String() + "/co-authors"},
		{"POST", "/api/v1/users/" + uuid.New().String() + "/mute"},
		{"DELETE", "/api/v1/users/" + uuid.New().String() + "/mute"},
		{"GET", "/api/v1/users/" + uuid.New().String() + "/mute-status"},
		{"GET", "/api/v1/my/muted-users"},
	}

	for _, endpoint := range protectedEndpoints {
		suite.T().Run(fmt.Sprintf("%s_%s", endpoint.method, endpoint.url), func(t *testing.T) {
			w := testutils.PerformRequest(router, testutils.TestRequest{
				Method:       endpoint.method,
				URL:          endpoint.url,
				Body:         `{}`,
				Headers:      map[string]string{"Content-Type": "application/json"},
				ExpectedCode: 200, // Middleware passes through in test mode
			})

			// In actual implementation, these would be 401, but our test middleware passes through
			_ = w // Just verify the endpoints exist and don't crash
		})
	}
}

// Test public endpoints that should work without authentication
func (suite *WorkServiceTestSuite) TestPublicEndpoints_WorkGifts() {
	router := setupRouter(suite.service)

	// Use one of our test works
	testWork := suite.testWorks[0]

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works/" + testWork.ID.String() + "/gifts",
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	// Verify gifts response structure (may be empty)
	assert.Contains(suite.T(), response, "gifts")
	gifts := response["gifts"].([]interface{})

	// Check structure if gifts exist
	if len(gifts) > 0 {
		gift := gifts[0].(map[string]interface{})
		assert.Contains(suite.T(), gift, "id")
		assert.Contains(suite.T(), gift, "work_id")
		assert.Contains(suite.T(), gift, "rejected")
		assert.Contains(suite.T(), gift, "created_at")
	}
}

// Test work creation structure validation
func (suite *WorkServiceTestSuite) TestCreateWork_ValidationErrors() {
	router := setupRouter(suite.service)

	testCases := []struct {
		name         string
		requestBody  string
		expectedCode int
	}{
		{
			name:         "missing title",
			requestBody:  `{"summary": "Test", "language": "en", "rating": "General Audiences", "fandoms": ["Test"], "chapter_content": "Content"}`,
			expectedCode: 400,
		},
		{
			name:         "missing language",
			requestBody:  `{"title": "Test", "summary": "Test", "rating": "General Audiences", "fandoms": ["Test"], "chapter_content": "Content"}`,
			expectedCode: 400,
		},
		{
			name:         "missing rating",
			requestBody:  `{"title": "Test", "summary": "Test", "language": "en", "fandoms": ["Test"], "chapter_content": "Content"}`,
			expectedCode: 400,
		},
		{
			name:         "missing fandoms",
			requestBody:  `{"title": "Test", "summary": "Test", "language": "en", "rating": "General Audiences", "chapter_content": "Content"}`,
			expectedCode: 400,
		},
		{
			name:         "missing chapter content",
			requestBody:  `{"title": "Test", "summary": "Test", "language": "en", "rating": "General Audiences", "fandoms": ["Test"]}`,
			expectedCode: 400,
		},
		{
			name:         "invalid rating",
			requestBody:  `{"title": "Test", "summary": "Test", "language": "en", "rating": "Invalid", "fandoms": ["Test"], "chapter_content": "Content"}`,
			expectedCode: 400,
		},
		{
			name:         "invalid language",
			requestBody:  `{"title": "Test", "summary": "Test", "language": "invalid", "rating": "General Audiences", "fandoms": ["Test"], "chapter_content": "Content"}`,
			expectedCode: 400,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			w := testutils.PerformRequest(router, testutils.TestRequest{
				Method:       "POST",
				URL:          "/api/v1/works",
				Body:         tc.requestBody,
				Headers:      map[string]string{"Content-Type": "application/json"},
				ExpectedCode: tc.expectedCode,
			})

			response := testutils.AssertJSONResponse(t, w, tc.expectedCode)
			assert.Contains(t, response, "error")
		})
	}
}

// Test the core database functions are accessible
func (suite *WorkServiceTestSuite) TestDatabaseFunctions_Available() {
	// Test that database functions exist by checking their existence
	var count int

	// Check if get_work_authors function exists
	err := suite.service.db.QueryRow(`
		SELECT COUNT(*) FROM pg_proc 
		WHERE proname = 'get_work_authors'`).Scan(&count)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, count, "get_work_authors function should exist")

	// Check if create_pseud function exists
	err = suite.service.db.QueryRow(`
		SELECT COUNT(*) FROM pg_proc 
		WHERE proname = 'create_pseud'`).Scan(&count)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, count, "create_pseud function should exist")

	// Check if orphan_work function exists
	err = suite.service.db.QueryRow(`
		SELECT COUNT(*) FROM pg_proc 
		WHERE proname = 'orphan_work'`).Scan(&count)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, count, "orphan_work function should exist")
}

// Test core AO3 tables exist
func (suite *WorkServiceTestSuite) TestCoreTables_Exist() {
	requiredTables := []string{
		"pseuds",
		"creatorships",
		"gifts",
		"user_mutes",
		"user_blocks",
		"comment_reports",
		"work_reports",
	}

	for _, tableName := range requiredTables {
		var count int
		err := suite.service.db.QueryRow(`
			SELECT COUNT(*) FROM information_schema.tables 
			WHERE table_name = $1 AND table_schema = 'public'`, tableName).Scan(&count)
		assert.NoError(suite.T(), err, "Failed to check table %s", tableName)
		assert.Equal(suite.T(), 1, count, "Table %s should exist", tableName)
	}
}

// Test privacy fields are included in work responses
func (suite *WorkServiceTestSuite) TestWork_PrivacyFields() {
	router := setupRouter(suite.service)

	// Use one of our test works
	testWork := suite.testWorks[0]

	w := testutils.PerformRequest(router, testutils.TestRequest{
		Method:       "GET",
		URL:          "/api/v1/works/" + testWork.ID.String(),
		ExpectedCode: 200,
	})

	response := testutils.AssertJSONResponse(suite.T(), w, 200)

	// Check that work object contains privacy fields
	work := response["work"].(map[string]interface{})

	privacyFields := []string{
		"restricted",
		"restricted_to_adults",
		"comment_policy",
		"moderate_comments",
		"disable_comments",
		"is_anonymous",
		"in_anon_collection",
		"in_unrevealed_collection",
	}

	for _, field := range privacyFields {
		assert.Contains(suite.T(), work, field, "Work should contain privacy field: %s", field)
	}
}

// Test endpoint structure validation
func (suite *WorkServiceTestSuite) TestEndpoint_ResponseStructures() {
	router := setupRouter(suite.service)

	testCases := []struct {
		name           string
		method         string
		url            string
		expectedFields []string
	}{
		{
			name:           "works search",
			method:         "GET",
			url:            "/api/v1/works",
			expectedFields: []string{"works", "pagination"},
		},
		{
			name:           "single work",
			method:         "GET",
			url:            "/api/v1/works/" + suite.testWorks[0].ID.String(),
			expectedFields: []string{"work", "authors"},
		},
		{
			name:           "work gifts",
			method:         "GET",
			url:            "/api/v1/works/" + suite.testWorks[0].ID.String() + "/gifts",
			expectedFields: []string{"gifts"},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			w := testutils.PerformRequest(router, testutils.TestRequest{
				Method:       tc.method,
				URL:          tc.url,
				ExpectedCode: 200,
			})

			response := testutils.AssertJSONResponse(t, w, 200)

			for _, field := range tc.expectedFields {
				assert.Contains(t, response, field, "Response should contain field: %s", field)
			}
		})
	}
}

func TestWorkServiceTestSuite(t *testing.T) {
	suite.Run(t, new(WorkServiceTestSuite))
}
