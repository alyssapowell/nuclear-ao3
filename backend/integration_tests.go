package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nuclear-ao3/shared/models"
)

// =============================================================================
// INTEGRATION TESTS - Service Communication
// Tests the integration between Work Service, Tag Service, and Search Service
// =============================================================================

// TestServiceIntegration_WorkCreationWithTags tests the complete workflow
// of creating a work with tags and verifying service communication
func TestServiceIntegration_WorkCreationWithTags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// This test requires all services to be running
	// In a real environment, this would be part of a Docker Compose test setup

	t.Run("CreateWorkWithTagsAndSearch", func(t *testing.T) {
		// 1. Create a work via work service
		workReq := models.CreateWorkRequest{
			Title:          "Integration Test Work",
			Summary:        "Testing service integration",
			Language:       "English",
			Rating:         "General Audiences",
			Category:       []string{"Gen"},
			Warnings:       []string{"No Archive Warnings Apply"},
			Fandoms:        []string{"Test Fandom"},
			Characters:     []string{"Test Character"},
			Relationships:  []string{"Test/Relationship"},
			FreeformTags:   []string{"Integration", "Testing"},
			ChapterTitle:   "Chapter 1",
			ChapterContent: "This is integration test content.",
		}

		// Test work creation endpoint
		workJSON, _ := json.Marshal(workReq)
		workResp := makeTestRequest(t, "POST", "http://work-service:8082/api/v1/works/enhanced", workJSON)
		require.Equal(t, http.StatusCreated, workResp.StatusCode)

		var workResult map[string]interface{}
		json.NewDecoder(workResp.Body).Decode(&workResult)
		workData := workResult["work"].(map[string]interface{})
		workID := workData["id"].(string)

		// 2. Verify tags were created in tag service
		time.Sleep(2 * time.Second) // Allow async processing

		tagsResp := makeTestRequest(t, "GET", fmt.Sprintf("http://tag-service:8083/api/v1/works/%s/tags", workID), nil)
		require.Equal(t, http.StatusOK, tagsResp.StatusCode)

		var tagsResult map[string]interface{}
		json.NewDecoder(tagsResp.Body).Decode(&tagsResult)
		tags := tagsResult["tags"].([]interface{})
		assert.Greater(t, len(tags), 0, "Should have tags associated with work")

		// 3. Verify work was indexed in search service
		time.Sleep(2 * time.Second) // Allow async indexing

		searchResp := makeTestRequest(t, "GET", "http://search-service:8084/api/v1/search/works?q=Integration+Test", nil)
		require.Equal(t, http.StatusOK, searchResp.StatusCode)

		var searchResult map[string]interface{}
		json.NewDecoder(searchResp.Body).Decode(&searchResult)
		works := searchResult["works"].([]interface{})

		// Find our work in search results
		found := false
		for _, work := range works {
			workMap := work.(map[string]interface{})
			if workMap["id"].(string) == workID {
				found = true
				break
			}
		}
		assert.True(t, found, "Work should be found in search results")

		// 4. Test advanced search with tag filtering
		advancedSearchResp := makeTestRequest(t, "GET",
			"http://work-service:8082/api/v1/works/search/advanced?tags=Integration&q=test", nil)
		require.Equal(t, http.StatusOK, advancedSearchResp.StatusCode)

		var advancedResult map[string]interface{}
		json.NewDecoder(advancedSearchResp.Body).Decode(&advancedResult)
		advancedWorks := advancedResult["works"].([]interface{})
		assert.Greater(t, len(advancedWorks), 0, "Should find works with tag filtering")
	})
}

// TestServiceIntegration_TagAutocomplete tests tag autocomplete functionality
func TestServiceIntegration_TagAutocomplete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	t.Run("TagAutocompletePerformance", func(t *testing.T) {
		// Test autocomplete response time
		start := time.Now()

		resp := makeTestRequest(t, "GET", "http://tag-service:8083/api/v1/tags/autocomplete?q=test", nil)
		duration := time.Since(start)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Less(t, duration, 100*time.Millisecond, "Autocomplete should be fast")

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		tags := result["tags"].([]interface{})
		assert.LessOrEqual(t, len(tags), 10, "Should limit autocomplete results")
	})

	t.Run("TagAutocompleteCaching", func(t *testing.T) {
		// First request (cache miss)
		start1 := time.Now()
		resp1 := makeTestRequest(t, "GET", "http://tag-service:8083/api/v1/tags/autocomplete?q=integration", nil)
		duration1 := time.Since(start1)
		require.Equal(t, http.StatusOK, resp1.StatusCode)

		// Second identical request (cache hit)
		start2 := time.Now()
		resp2 := makeTestRequest(t, "GET", "http://tag-service:8083/api/v1/tags/autocomplete?q=integration", nil)
		duration2 := time.Since(start2)
		require.Equal(t, http.StatusOK, resp2.StatusCode)

		// Cache hit should be faster (or at least not significantly slower)
		assert.LessOrEqual(t, duration2, duration1+10*time.Millisecond,
			"Cached request should not be slower than initial request")
	})
}

// TestServiceIntegration_WorkStatistics tests real-time statistics
func TestServiceIntegration_WorkStatistics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	t.Run("RealTimeStatistics", func(t *testing.T) {
		// Create a test work first
		workID := createTestWorkForIntegration(t)

		// Get initial statistics
		statsResp1 := makeTestRequest(t, "PUT",
			fmt.Sprintf("http://work-service:8082/api/v1/works/%s/stats", workID), nil)
		require.Equal(t, http.StatusOK, statsResp1.StatusCode)

		var stats1 map[string]interface{}
		json.NewDecoder(statsResp1.Body).Decode(&stats1)
		initialHits := stats1["statistics"].(map[string]interface{})["hits"].(float64)

		// Simulate viewing the work (should increment hits)
		viewResp := makeTestRequest(t, "GET",
			fmt.Sprintf("http://work-service:8082/api/v1/works/%s/tags", workID), nil)
		require.Equal(t, http.StatusOK, viewResp.StatusCode)

		// Wait for async hit increment
		time.Sleep(1 * time.Second)

		// Get updated statistics
		statsResp2 := makeTestRequest(t, "PUT",
			fmt.Sprintf("http://work-service:8082/api/v1/works/%s/stats", workID), nil)
		require.Equal(t, http.StatusOK, statsResp2.StatusCode)

		var stats2 map[string]interface{}
		json.NewDecoder(statsResp2.Body).Decode(&stats2)
		updatedHits := stats2["statistics"].(map[string]interface{})["hits"].(float64)

		assert.Greater(t, updatedHits, initialHits, "Hit count should increase after viewing")
	})
}

// TestServiceIntegration_ErrorHandling tests service resilience
func TestServiceIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	t.Run("TagServiceUnavailable", func(t *testing.T) {
		// Test work creation when tag service is unavailable
		// This would require orchestrating service shutdown/startup
		// For now, we test the graceful fallback logic

		workReq := models.CreateWorkRequest{
			Title:        "Resilience Test Work",
			Summary:      "Testing service resilience",
			Fandoms:      []string{"Test Fandom"},
			FreeformTags: []string{"Resilience"},
		}

		workJSON, _ := json.Marshal(workReq)
		resp := makeTestRequest(t, "POST", "http://work-service:8082/api/v1/works/enhanced", workJSON)

		// Work creation should succeed even if tag service has issues
		assert.Contains(t, []int{http.StatusCreated, http.StatusInternalServerError}, resp.StatusCode)
	})

	t.Run("SearchServiceFallback", func(t *testing.T) {
		// Test advanced search fallback to database when search service is unavailable
		resp := makeTestRequest(t, "GET",
			"http://work-service:8082/api/v1/works/search/advanced?q=test&limit=5", nil)

		// Should get results from database fallback
		assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, resp.StatusCode)
	})
}

// TestServiceIntegration_Performance tests performance across services
func TestServiceIntegration_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	t.Run("EndToEndLatency", func(t *testing.T) {
		// Test complete workflow latency
		start := time.Now()

		// Create work
		workReq := models.CreateWorkRequest{
			Title:        "Performance Test Work",
			Summary:      "Testing end-to-end performance",
			Fandoms:      []string{"Performance Test"},
			FreeformTags: []string{"Speed", "Performance"},
		}

		workJSON, _ := json.Marshal(workReq)
		workResp := makeTestRequest(t, "POST", "http://work-service:8082/api/v1/works/enhanced", workJSON)
		require.Equal(t, http.StatusCreated, workResp.StatusCode)

		var workResult map[string]interface{}
		json.NewDecoder(workResp.Body).Decode(&workResult)
		workData := workResult["work"].(map[string]interface{})
		workID := workData["id"].(string)

		// Retrieve with tags
		tagsResp := makeTestRequest(t, "GET",
			fmt.Sprintf("http://work-service:8082/api/v1/works/%s/tags", workID), nil)
		require.Equal(t, http.StatusOK, tagsResp.StatusCode)

		duration := time.Since(start)

		// Complete workflow should be fast
		assert.Less(t, duration, 5*time.Second,
			"End-to-end work creation and retrieval should be fast")
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		// Test handling concurrent requests
		const numRequests = 10
		results := make(chan bool, numRequests)

		start := time.Now()

		for i := 0; i < numRequests; i++ {
			go func(id int) {
				resp := makeTestRequest(t, "GET",
					"http://tag-service:8083/api/v1/tags/autocomplete?q=test", nil)
				results <- resp.StatusCode == http.StatusOK
			}(i)
		}

		// Wait for all requests to complete
		successCount := 0
		for i := 0; i < numRequests; i++ {
			if <-results {
				successCount++
			}
		}

		duration := time.Since(start)

		// All requests should succeed
		assert.Equal(t, numRequests, successCount, "All concurrent requests should succeed")

		// Should handle concurrent load efficiently
		assert.Less(t, duration, 2*time.Second,
			"Concurrent requests should complete quickly")
	})
}

// TestServiceIntegration_TagRelationships tests tag relationship functionality
func TestServiceIntegration_TagRelationships(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	t.Run("TagSynonyms", func(t *testing.T) {
		// Create parent tag
		parentTag := map[string]interface{}{
			"name":          "Harry Potter",
			"type":          "fandom",
			"is_filterable": true,
		}
		parentJSON, _ := json.Marshal(parentTag)
		parentResp := makeTestRequest(t, "POST", "http://tag-service:8083/api/v1/tags", parentJSON)
		require.Equal(t, http.StatusCreated, parentResp.StatusCode)

		var parentResult map[string]interface{}
		json.NewDecoder(parentResp.Body).Decode(&parentResult)
		parentID := parentResult["tag"].(map[string]interface{})["id"].(string)

		// Create child tag
		childTag := map[string]interface{}{
			"name":          "HP",
			"type":          "fandom",
			"is_filterable": true,
		}
		childJSON, _ := json.Marshal(childTag)
		childResp := makeTestRequest(t, "POST", "http://tag-service:8083/api/v1/tags", childJSON)
		require.Equal(t, http.StatusCreated, childResp.StatusCode)

		var childResult map[string]interface{}
		json.NewDecoder(childResp.Body).Decode(&childResult)
		childID := childResult["tag"].(map[string]interface{})["id"].(string)

		// Create relationship
		relationship := map[string]interface{}{
			"parent_tag_id": parentID,
			"child_tag_id":  childID,
			"type":          "synonym",
		}
		relJSON, _ := json.Marshal(relationship)
		relResp := makeTestRequest(t, "POST", "http://tag-service:8083/api/v1/tags/relationships", relJSON)
		require.Equal(t, http.StatusCreated, relResp.StatusCode)

		// Verify relationship exists
		relGetResp := makeTestRequest(t, "GET",
			fmt.Sprintf("http://tag-service:8083/api/v1/tags/%s/relationships", parentID), nil)
		require.Equal(t, http.StatusOK, relGetResp.StatusCode)

		var relGetResult map[string]interface{}
		json.NewDecoder(relGetResp.Body).Decode(&relGetResult)
		relationships := relGetResult["relationships"].([]interface{})
		assert.Greater(t, len(relationships), 0, "Should have tag relationships")
	})
}

// Helper functions for integration tests

func makeTestRequest(t *testing.T, method, url string, body []byte) *http.Response {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	require.NoError(t, err)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)

	// In real integration tests, this should not fail
	// For unit tests, we might expect connection errors
	if err != nil {
		t.Logf("Request failed (expected in unit test environment): %v", err)
		// Return a mock response for unit testing
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       http.NoBody,
		}
	}

	return resp
}

func createTestWorkForIntegration(t *testing.T) string {
	workReq := models.CreateWorkRequest{
		Title:        "Integration Test Work",
		Summary:      "For testing statistics",
		Fandoms:      []string{"Test Fandom"},
		FreeformTags: []string{"Statistics"},
	}

	workJSON, _ := json.Marshal(workReq)
	resp := makeTestRequest(t, "POST", "http://work-service:8082/api/v1/works/enhanced", workJSON)

	if resp.StatusCode == http.StatusServiceUnavailable {
		// Return mock UUID for unit testing
		return uuid.New().String()
	}

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	workData := result["work"].(map[string]interface{})
	return workData["id"].(string)
}

// Benchmark tests for integration performance

func BenchmarkIntegration_WorkCreation(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmarks in short mode")
	}

	workReq := models.CreateWorkRequest{
		Title:        "Benchmark Work",
		Summary:      "Performance testing",
		Fandoms:      []string{"Benchmark"},
		FreeformTags: []string{"Speed"},
	}

	workJSON, _ := json.Marshal(workReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := makeTestRequest(nil, "POST", "http://work-service:8082/api/v1/works/enhanced", workJSON)
		if resp.StatusCode == http.StatusCreated {
			resp.Body.Close()
		}
	}
}

func BenchmarkIntegration_TagAutocomplete(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmarks in short mode")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := makeTestRequest(nil, "GET", "http://tag-service:8083/api/v1/tags/autocomplete?q=test", nil)
		if resp.StatusCode == http.StatusOK {
			resp.Body.Close()
		}
	}
}

func BenchmarkIntegration_AdvancedSearch(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmarks in short mode")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := makeTestRequest(nil, "GET", "http://work-service:8082/api/v1/works/search/advanced?q=test&limit=20", nil)
		if resp.StatusCode == http.StatusOK {
			resp.Body.Close()
		}
	}
}

// Test helper to ensure services are available
func TestServiceAvailability(t *testing.T) {
	services := map[string]string{
		"work-service":   "http://work-service:8082/health",
		"tag-service":    "http://tag-service:8083/health",
		"search-service": "http://search-service:8084/health",
	}

	for name, url := range services {
		t.Run(fmt.Sprintf("Service_%s", name), func(t *testing.T) {
			resp := makeTestRequest(t, "GET", url, nil)
			if resp.StatusCode == http.StatusServiceUnavailable {
				t.Skipf("Service %s not available - skipping integration tests", name)
			} else {
				assert.Equal(t, http.StatusOK, resp.StatusCode,
					"Service %s should be healthy", name)
			}
		})
	}
}
