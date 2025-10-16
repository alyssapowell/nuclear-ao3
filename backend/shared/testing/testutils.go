package testing

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestRequest represents a test HTTP request
type TestRequest struct {
	Method       string
	URL          string
	Body         interface{}
	Headers      map[string]string
	ExpectedCode int
}

// PerformRequest executes a test request against a Gin router
func PerformRequest(router *gin.Engine, req TestRequest) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader

	if req.Body != nil {
		bodyBytes, _ := json.Marshal(req.Body)
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	httpReq, _ := http.NewRequest(req.Method, req.URL, bodyReader)

	// Add headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set content type for POST/PUT requests
	if req.Method == "POST" || req.Method == "PUT" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	return w
}

// AssertJSONResponse checks that the response is valid JSON with expected status
func AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedCode int) map[string]interface{} {
	assert.Equal(t, expectedCode, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	return response
}

// AssertHealthEndpoint tests the standard health endpoint
func AssertHealthEndpoint(t *testing.T, router *gin.Engine, serviceName string) {
	w := PerformRequest(router, TestRequest{
		Method:       "GET",
		URL:          "/health",
		ExpectedCode: 200,
	})

	response := AssertJSONResponse(t, w, 200)
	assert.Equal(t, serviceName, response["service"])
	assert.Equal(t, "healthy", response["status"])
	assert.Contains(t, response, "timestamp")
	assert.Contains(t, response, "version")
}
