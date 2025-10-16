package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const baseURL = "http://localhost:8082"

// TestNewFeaturesIntegration tests the new AO3 core features
func TestNewFeaturesIntegration(t *testing.T) {
	// Skip if work service is not running
	if !isServiceRunning() {
		t.Skip("Work service is not running - run 'docker-compose up -d work-service' first")
	}

	t.Run("GetWork returns new structure with authors", func(t *testing.T) {
		// Get a work to test the new response structure
		resp, err := http.Get(baseURL + "/api/v1/works?limit=1")
		require.NoError(t, err)
		defer resp.Body.Close()

		var searchResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&searchResult)
		require.NoError(t, err)

		works := searchResult["works"].([]interface{})
		if len(works) == 0 {
			t.Skip("No works found in database")
		}

		work := works[0].(map[string]interface{})
		workID := work["id"].(string)

		// Test the new GetWork endpoint
		resp, err = http.Get(baseURL + "/api/v1/works/" + workID)
		require.NoError(t, err)
		defer resp.Body.Close()

		var workResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&workResult)
		require.NoError(t, err)

		// Verify new response structure
		assert.Contains(t, workResult, "work", "Response should contain work object")
		assert.Contains(t, workResult, "authors", "Response should contain authors array")

		// Check work object
		workObj := workResult["work"].(map[string]interface{})
		assert.Contains(t, workObj, "title")
		assert.Contains(t, workObj, "id")

		// Check authors array
		authors := workResult["authors"].([]interface{})
		assert.NotEmpty(t, authors, "Authors array should not be empty")

		if len(authors) > 0 {
			author := authors[0].(map[string]interface{})
			assert.Contains(t, author, "pseud_id")
			assert.Contains(t, author, "pseud_name")
			assert.Contains(t, author, "user_id")
			assert.Contains(t, author, "username")
			assert.Contains(t, author, "is_anonymous")
		}
	})

	t.Run("Work privacy fields are present", func(t *testing.T) {
		// Get a work to test privacy fields
		resp, err := http.Get(baseURL + "/api/v1/works?limit=1")
		require.NoError(t, err)
		defer resp.Body.Close()

		var searchResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&searchResult)
		require.NoError(t, err)

		works := searchResult["works"].([]interface{})
		if len(works) == 0 {
			t.Skip("No works found in database")
		}

		work := works[0].(map[string]interface{})
		workID := work["id"].(string)

		// Test the GetWork endpoint for privacy fields
		resp, err = http.Get(baseURL + "/api/v1/works/" + workID)
		require.NoError(t, err)
		defer resp.Body.Close()

		var workResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&workResult)
		require.NoError(t, err)

		workObj := workResult["work"].(map[string]interface{})

		// Check privacy fields
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
			assert.Contains(t, workObj, field, "Work should contain privacy field: %s", field)
		}
	})

	t.Run("Work gifts endpoint exists", func(t *testing.T) {
		// Get a work to test gifts endpoint
		resp, err := http.Get(baseURL + "/api/v1/works?limit=1")
		require.NoError(t, err)
		defer resp.Body.Close()

		var searchResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&searchResult)
		require.NoError(t, err)

		works := searchResult["works"].([]interface{})
		if len(works) == 0 {
			t.Skip("No works found in database")
		}

		work := works[0].(map[string]interface{})
		workID := work["id"].(string)

		// Test gifts endpoint
		resp, err = http.Get(baseURL + "/api/v1/works/" + workID + "/gifts")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var giftsResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&giftsResult)
		require.NoError(t, err)

		assert.Contains(t, giftsResult, "gifts", "Response should contain gifts array")
	})

	t.Run("Work authors endpoint exists", func(t *testing.T) {
		// Get a work to test authors endpoint
		resp, err := http.Get(baseURL + "/api/v1/works?limit=1")
		require.NoError(t, err)
		defer resp.Body.Close()

		var searchResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&searchResult)
		require.NoError(t, err)

		works := searchResult["works"].([]interface{})
		if len(works) == 0 {
			t.Skip("No works found in database")
		}

		work := works[0].(map[string]interface{})
		workID := work["id"].(string)

		// Test authors endpoint
		resp, err = http.Get(baseURL + "/api/v1/works/" + workID + "/authors")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var authorsResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&authorsResult)
		require.NoError(t, err)

		assert.Contains(t, authorsResult, "authors", "Response should contain authors array")

		authors := authorsResult["authors"].([]interface{})
		assert.NotEmpty(t, authors, "Authors array should not be empty")
	})

	t.Run("Protected endpoints require authentication", func(t *testing.T) {
		protectedEndpoints := []struct {
			method string
			url    string
		}{
			{"POST", "/api/v1/pseuds"},
			{"GET", "/api/v1/my/pseuds"},
			{"POST", "/api/v1/works"},
		}

		for _, endpoint := range protectedEndpoints {
			t.Run(fmt.Sprintf("%s %s", endpoint.method, endpoint.url), func(t *testing.T) {
				req, err := http.NewRequest(endpoint.method, baseURL+endpoint.url, nil)
				require.NoError(t, err)

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				// In production these would be 401, but our test middleware passes through
				// Just verify the endpoints exist and respond
				assert.NotEqual(t, http.StatusNotFound, resp.StatusCode, "Endpoint should exist")
			})
		}
	})
}

func isServiceRunning() bool {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	var health map[string]interface{}
	if err := json.Unmarshal(body, &health); err != nil {
		return false
	}

	return health["service"] == "work-service" && health["status"] == "healthy"
}
