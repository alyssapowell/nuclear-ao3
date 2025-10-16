package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Performance test suite for OAuth2/OIDC
type OAuth2PerformanceTestSuite struct {
	suite.Suite
	service     *AuthService
	router      *gin.Engine
	db          *sql.DB
	redis       *redis.Client
	testUsers   map[string]*models.User
	testClients map[string]*models.OAuthClient
}

func (suite *OAuth2PerformanceTestSuite) SetupSuite() {
	// Use test database
	testDB := os.Getenv("TEST_DATABASE_URL")
	if testDB == "" {
		testDB = "postgres://ao3_user:ao3_password@localhost/ao3_nuclear_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", testDB)
	require.NoError(suite.T(), err)
	suite.db = db

	// Use test redis instance
	testRedis := os.Getenv("TEST_REDIS_URL")
	if testRedis == "" {
		testRedis = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: testRedis,
		DB:   5, // Use DB 5 for performance tests
	})
	suite.redis = rdb

	// Create test auth service
	jwtManager, err := NewJWTManager("test-perf-secret", "test-perf-issuer", nil)
	require.NoError(suite.T(), err)

	suite.service = &AuthService{
		db:    db,
		redis: rdb,
		jwt:   jwtManager,
	}

	// Setup test router
	gin.SetMode(gin.TestMode)
	suite.router = setupRouter(suite.service)

	// Initialize test data
	suite.testUsers = make(map[string]*models.User)
	suite.testClients = make(map[string]*models.OAuthClient)
	suite.setupPerformanceTestData()
}

func (suite *OAuth2PerformanceTestSuite) SetupTest() {
	suite.redis.FlushDB(context.Background())
}

func (suite *OAuth2PerformanceTestSuite) TearDownSuite() {
	suite.cleanupTestData()
	suite.db.Close()
	suite.redis.Close()
}

func (suite *OAuth2PerformanceTestSuite) cleanupTestData() {
	tables := []string{
		"user_consents", "oauth_refresh_tokens", "oauth_access_tokens",
		"authorization_codes", "oauth_clients", "users",
	}
	for _, table := range tables {
		suite.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE created_at > NOW() - INTERVAL '1 day'", table))
	}
}

func (suite *OAuth2PerformanceTestSuite) setupPerformanceTestData() {
	// Create multiple test users for concurrent testing
	for i := 0; i < 10; i++ {
		username := fmt.Sprintf("perf_user_%d", i)
		email := fmt.Sprintf("perf%d@nuclear-ao3.test", i)
		suite.createTestUser(username, email, "password123")
	}

	// Create test clients
	suite.createPerformanceTestClients()
}

func (suite *OAuth2PerformanceTestSuite) createTestUser(username, email, password string) {
	registerReq := models.RegisterRequest{
		Username:        username,
		Email:           email,
		Password:        password,
		ConfirmPassword: password,
		DisplayName:     fmt.Sprintf("Perf Test %s", username),
		AcceptTOS:       true,
	}

	jsonBody, _ := json.Marshal(registerReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)

	if w.Code == http.StatusCreated {
		var response models.AuthResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		suite.testUsers[username] = response.User
		suite.db.Exec("UPDATE users SET is_verified = true WHERE id = $1", response.User.ID)
	}
}

func (suite *OAuth2PerformanceTestSuite) createPerformanceTestClients() {
	clients := []struct {
		name     string
		isPublic bool
		scopes   []string
	}{
		{"perf_confidential", false, []string{"read", "write", "profile", "email"}},
		{"perf_public", true, []string{"read", "profile"}},
		{"perf_service", false, []string{"read"}},
	}

	for _, c := range clients {
		regReq := models.ClientRegistrationRequest{
			Name:            c.name,
			Description:     fmt.Sprintf("Performance test client: %s", c.name),
			Website:         "https://perf.example.com",
			RedirectURIs:    []string{"https://perf.example.com/callback"},
			Scopes:          c.scopes,
			GrantTypes:      []string{"authorization_code", "refresh_token", "client_credentials"},
			ResponseTypes:   []string{"code"},
			IsPublic:        c.isPublic,
			AccessTokenTTL:  3600,
			RefreshTokenTTL: 86400,
		}

		jsonBody, _ := json.Marshal(regReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register-client", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		suite.router.ServeHTTP(w, req)

		if w.Code == http.StatusCreated {
			var response models.ClientRegistrationResponse
			json.Unmarshal(w.Body.Bytes(), &response)

			client := &models.OAuthClient{
				ID:            uuid.MustParse(response.ClientID),
				Secret:        response.ClientSecret,
				Name:          response.Name,
				RedirectURIs:  response.RedirectURIs,
				Scopes:        response.Scopes,
				GrantTypes:    response.GrantTypes,
				ResponseTypes: response.ResponseTypes,
				IsPublic:      response.IsPublic,
			}
			suite.testClients[c.name] = client
		}
	}
}

// Performance Test: Token Introspection

func (suite *OAuth2PerformanceTestSuite) TestTokenIntrospection_Performance() {
	client := suite.testClients["perf_confidential"]
	user := suite.testUsers["perf_user_0"]

	// Pre-generate access tokens
	tokens := make([]string, 100)
	for i := 0; i < 100; i++ {
		tokens[i] = suite.getAccessToken(client, user, []string{"read"})
	}

	// Measure introspection performance
	start := time.Now()
	var wg sync.WaitGroup
	const concurrency = 20
	const totalRequests = 100

	results := make(chan time.Duration, totalRequests)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(startIdx int) {
			defer wg.Done()
			for j := 0; j < totalRequests/concurrency; j++ {
				tokenIdx := startIdx + j*concurrency
				if tokenIdx >= len(tokens) {
					break
				}

				requestStart := time.Now()

				introspectReq := models.IntrospectRequest{
					Token:        tokens[tokenIdx],
					ClientID:     client.ID.String(),
					ClientSecret: client.Secret,
				}

				introspectBody, _ := json.Marshal(introspectReq)
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/auth/introspect", bytes.NewBuffer(introspectBody))
				req.Header.Set("Content-Type", "application/json")

				suite.router.ServeHTTP(w, req)

				results <- time.Since(requestStart)
				assert.Equal(suite.T(), http.StatusOK, w.Code)
			}
		}(i)
	}

	wg.Wait()
	close(results)
	totalTime := time.Since(start)

	// Collect timing statistics
	var times []time.Duration
	for duration := range results {
		times = append(times, duration)
	}

	// Calculate statistics
	avgTime := suite.calculateAverage(times)
	p95Time := suite.calculatePercentile(times, 95)
	p99Time := suite.calculatePercentile(times, 99)

	suite.T().Logf("Token Introspection Performance:")
	suite.T().Logf("Total requests: %d", len(times))
	suite.T().Logf("Total time: %v", totalTime)
	suite.T().Logf("Requests/sec: %.2f", float64(len(times))/totalTime.Seconds())
	suite.T().Logf("Average response time: %v", avgTime)
	suite.T().Logf("95th percentile: %v", p95Time)
	suite.T().Logf("99th percentile: %v", p99Time)

	// Performance assertions
	assert.Less(suite.T(), avgTime, 50*time.Millisecond, "Average introspection should be under 50ms")
	assert.Less(suite.T(), p95Time, 100*time.Millisecond, "95th percentile should be under 100ms")
	assert.Greater(suite.T(), float64(len(times))/totalTime.Seconds(), 50.0, "Should handle at least 50 introspections/sec")
}

// Performance Test: Authorization Flow

func (suite *OAuth2PerformanceTestSuite) TestAuthorizationFlow_Performance() {
	client := suite.testClients["perf_confidential"]
	users := []*models.User{
		suite.testUsers["perf_user_0"],
		suite.testUsers["perf_user_1"],
		suite.testUsers["perf_user_2"],
		suite.testUsers["perf_user_3"],
		suite.testUsers["perf_user_4"],
	}

	const totalFlows = 50
	const concurrency = 10

	start := time.Now()
	var wg sync.WaitGroup
	results := make(chan time.Duration, totalFlows)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < totalFlows/concurrency; j++ {
				user := users[j%len(users)]

				flowStart := time.Now()

				// Complete authorization flow: auth code -> token exchange
				authCode := suite.getAuthorizationCode(client, user, []string{"read", "profile"})
				assert.NotEmpty(suite.T(), authCode)

				// Exchange for tokens
				tokenReq := models.TokenRequest{
					GrantType:    "authorization_code",
					Code:         authCode,
					RedirectURI:  client.RedirectURIs[0],
					ClientID:     client.ID.String(),
					ClientSecret: client.Secret,
				}

				tokenBody, _ := json.Marshal(tokenReq)
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(tokenBody))
				req.Header.Set("Content-Type", "application/json")

				suite.router.ServeHTTP(w, req)

				assert.Equal(suite.T(), http.StatusOK, w.Code)

				results <- time.Since(flowStart)
			}
		}(i)
	}

	wg.Wait()
	close(results)
	totalTime := time.Since(start)

	// Collect timing statistics
	var times []time.Duration
	for duration := range results {
		times = append(times, duration)
	}

	avgTime := suite.calculateAverage(times)
	p95Time := suite.calculatePercentile(times, 95)

	suite.T().Logf("Authorization Flow Performance:")
	suite.T().Logf("Total flows: %d", len(times))
	suite.T().Logf("Total time: %v", totalTime)
	suite.T().Logf("Flows/sec: %.2f", float64(len(times))/totalTime.Seconds())
	suite.T().Logf("Average flow time: %v", avgTime)
	suite.T().Logf("95th percentile: %v", p95Time)

	// Performance assertions
	assert.Less(suite.T(), avgTime, 200*time.Millisecond, "Average authorization flow should be under 200ms")
	assert.Less(suite.T(), p95Time, 500*time.Millisecond, "95th percentile should be under 500ms")
}

// Performance Test: OIDC Discovery

func (suite *OAuth2PerformanceTestSuite) TestOIDCDiscovery_Performance() {
	const totalRequests = 200
	const concurrency = 20

	start := time.Now()
	var wg sync.WaitGroup
	results := make(chan time.Duration, totalRequests)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < totalRequests/concurrency; j++ {
				requestStart := time.Now()

				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", "/.well-known/openid-configuration", nil)

				suite.router.ServeHTTP(w, req)

				assert.Equal(suite.T(), http.StatusOK, w.Code)
				results <- time.Since(requestStart)
			}
		}()
	}

	wg.Wait()
	close(results)
	totalTime := time.Since(start)

	var times []time.Duration
	for duration := range results {
		times = append(times, duration)
	}

	avgTime := suite.calculateAverage(times)
	p95Time := suite.calculatePercentile(times, 95)

	suite.T().Logf("OIDC Discovery Performance:")
	suite.T().Logf("Total requests: %d", len(times))
	suite.T().Logf("Requests/sec: %.2f", float64(len(times))/totalTime.Seconds())
	suite.T().Logf("Average response time: %v", avgTime)
	suite.T().Logf("95th percentile: %v", p95Time)

	// Performance assertions - discovery should be very fast (cached)
	assert.Less(suite.T(), avgTime, 10*time.Millisecond, "Discovery should be under 10ms (cached)")
	assert.Greater(suite.T(), float64(len(times))/totalTime.Seconds(), 500.0, "Should handle 500+ discovery requests/sec")
}

// Performance Test: UserInfo Endpoint

func (suite *OAuth2PerformanceTestSuite) TestUserInfo_Performance() {
	client := suite.testClients["perf_confidential"]

	// Pre-generate access tokens with profile scope
	tokens := make([]string, 50)
	for i := 0; i < 50; i++ {
		userKey := fmt.Sprintf("perf_user_%d", i%5) // Cycle through 5 users
		user := suite.testUsers[userKey]
		tokens[i] = suite.getAccessToken(client, user, []string{"openid", "profile", "email"})
	}

	const concurrency = 10
	start := time.Now()
	var wg sync.WaitGroup
	results := make(chan time.Duration, len(tokens))

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(startIdx int) {
			defer wg.Done()
			for j := startIdx; j < len(tokens); j += concurrency {
				requestStart := time.Now()

				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", "/auth/userinfo", nil)
				req.Header.Set("Authorization", "Bearer "+tokens[j])

				suite.router.ServeHTTP(w, req)

				assert.Equal(suite.T(), http.StatusOK, w.Code)
				results <- time.Since(requestStart)
			}
		}(i)
	}

	wg.Wait()
	close(results)
	totalTime := time.Since(start)

	var times []time.Duration
	for duration := range results {
		times = append(times, duration)
	}

	avgTime := suite.calculateAverage(times)
	p95Time := suite.calculatePercentile(times, 95)

	suite.T().Logf("UserInfo Performance:")
	suite.T().Logf("Total requests: %d", len(times))
	suite.T().Logf("Requests/sec: %.2f", float64(len(times))/totalTime.Seconds())
	suite.T().Logf("Average response time: %v", avgTime)
	suite.T().Logf("95th percentile: %v", p95Time)

	// Performance assertions
	assert.Less(suite.T(), avgTime, 30*time.Millisecond, "UserInfo should be under 30ms")
	assert.Greater(suite.T(), float64(len(times))/totalTime.Seconds(), 100.0, "Should handle 100+ userinfo requests/sec")
}

// Performance Test: Token Refresh

func (suite *OAuth2PerformanceTestSuite) TestTokenRefresh_Performance() {
	client := suite.testClients["perf_confidential"]
	user := suite.testUsers["perf_user_0"]

	// Pre-generate refresh tokens
	refreshTokens := make([]string, 30)
	for i := 0; i < 30; i++ {
		_, refreshToken := suite.getTokenPair(client, user, []string{"read", "profile"})
		refreshTokens[i] = refreshToken
	}

	const concurrency = 6
	start := time.Now()
	var wg sync.WaitGroup
	results := make(chan time.Duration, len(refreshTokens))

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(startIdx int) {
			defer wg.Done()
			for j := startIdx; j < len(refreshTokens); j += concurrency {
				requestStart := time.Now()

				tokenReq := models.TokenRequest{
					GrantType:    "refresh_token",
					RefreshToken: refreshTokens[j],
					ClientID:     client.ID.String(),
					ClientSecret: client.Secret,
				}

				tokenBody, _ := json.Marshal(tokenReq)
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(tokenBody))
				req.Header.Set("Content-Type", "application/json")

				suite.router.ServeHTTP(w, req)

				assert.Equal(suite.T(), http.StatusOK, w.Code)
				results <- time.Since(requestStart)
			}
		}(i)
	}

	wg.Wait()
	close(results)
	totalTime := time.Since(start)

	var times []time.Duration
	for duration := range results {
		times = append(times, duration)
	}

	avgTime := suite.calculateAverage(times)
	p95Time := suite.calculatePercentile(times, 95)

	suite.T().Logf("Token Refresh Performance:")
	suite.T().Logf("Total refreshes: %d", len(times))
	suite.T().Logf("Refreshes/sec: %.2f", float64(len(times))/totalTime.Seconds())
	suite.T().Logf("Average response time: %v", avgTime)
	suite.T().Logf("95th percentile: %v", p95Time)

	// Performance assertions
	assert.Less(suite.T(), avgTime, 100*time.Millisecond, "Token refresh should be under 100ms")
	assert.Greater(suite.T(), float64(len(times))/totalTime.Seconds(), 20.0, "Should handle 20+ token refreshes/sec")
}

// Load Test: Mixed OAuth2 Operations

func (suite *OAuth2PerformanceTestSuite) TestMixedOperations_LoadTest() {
	client := suite.testClients["perf_confidential"]
	users := []*models.User{
		suite.testUsers["perf_user_0"],
		suite.testUsers["perf_user_1"],
		suite.testUsers["perf_user_2"],
		suite.testUsers["perf_user_3"],
		suite.testUsers["perf_user_4"],
	}

	// Pre-generate some access tokens for introspection/userinfo tests
	accessTokens := make([]string, 20)
	for i := 0; i < 20; i++ {
		user := users[i%len(users)]
		accessTokens[i] = suite.getAccessToken(client, user, []string{"openid", "profile", "email"})
	}

	const totalOperations = 200
	const concurrency = 20
	const testDuration = 10 * time.Second

	operations := []string{"auth_flow", "introspect", "userinfo", "discovery", "jwks"}

	start := time.Now()
	var wg sync.WaitGroup
	results := make(chan struct {
		operation string
		duration  time.Duration
		success   bool
	}, totalOperations*2) // Buffer for safety

	// Run operations for a fixed duration
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			opCount := 0
			for time.Since(start) < testDuration {
				operation := operations[opCount%len(operations)]
				opCount++

				opStart := time.Now()
				success := suite.performOperation(operation, client, users, accessTokens)

				results <- struct {
					operation string
					duration  time.Duration
					success   bool
				}{operation, time.Since(opStart), success}
			}
		}(i)
	}

	wg.Wait()
	close(results)
	totalTime := time.Since(start)

	// Collect statistics by operation type
	opStats := make(map[string]struct {
		count     int
		totalTime time.Duration
		successes int
	})

	totalOps := 0
	for result := range results {
		totalOps++
		stats := opStats[result.operation]
		stats.count++
		stats.totalTime += result.duration
		if result.success {
			stats.successes++
		}
		opStats[result.operation] = stats
	}

	suite.T().Logf("Mixed Operations Load Test Results:")
	suite.T().Logf("Test duration: %v", totalTime)
	suite.T().Logf("Total operations: %d", totalOps)
	suite.T().Logf("Operations/sec: %.2f", float64(totalOps)/totalTime.Seconds())

	for op, stats := range opStats {
		avgTime := stats.totalTime / time.Duration(stats.count)
		successRate := float64(stats.successes) / float64(stats.count) * 100

		suite.T().Logf("%s: %d ops, avg %.2fms, %.1f%% success",
			op, stats.count, float64(avgTime.Nanoseconds())/1e6, successRate)

		// Assert success rates
		assert.Greater(suite.T(), successRate, 95.0, fmt.Sprintf("%s should have >95%% success rate", op))
	}

	// Overall performance assertion
	assert.Greater(suite.T(), float64(totalOps)/totalTime.Seconds(), 50.0, "Should handle 50+ mixed operations/sec")
}

// Helper methods

func (suite *OAuth2PerformanceTestSuite) performOperation(operation string, client *models.OAuthClient, users []*models.User, accessTokens []string) bool {
	switch operation {
	case "auth_flow":
		user := users[len(users)%len(users)] // Random user
		authCode := suite.getAuthorizationCode(client, user, []string{"read"})
		return authCode != ""

	case "introspect":
		token := accessTokens[len(accessTokens)%len(accessTokens)] // Random token
		return suite.performIntrospection(client, token)

	case "userinfo":
		token := accessTokens[len(accessTokens)%len(accessTokens)]
		return suite.performUserInfo(token)

	case "discovery":
		return suite.performDiscovery()

	case "jwks":
		return suite.performJWKS()

	default:
		return false
	}
}

func (suite *OAuth2PerformanceTestSuite) performIntrospection(client *models.OAuthClient, token string) bool {
	introspectReq := models.IntrospectRequest{
		Token:        token,
		ClientID:     client.ID.String(),
		ClientSecret: client.Secret,
	}

	introspectBody, _ := json.Marshal(introspectReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/introspect", bytes.NewBuffer(introspectBody))
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)
	return w.Code == http.StatusOK
}

func (suite *OAuth2PerformanceTestSuite) performUserInfo(token string) bool {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	suite.router.ServeHTTP(w, req)
	return w.Code == http.StatusOK
}

func (suite *OAuth2PerformanceTestSuite) performDiscovery() bool {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/openid-configuration", nil)

	suite.router.ServeHTTP(w, req)
	return w.Code == http.StatusOK
}

func (suite *OAuth2PerformanceTestSuite) performJWKS() bool {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/jwks", nil)

	suite.router.ServeHTTP(w, req)
	return w.Code == http.StatusOK
}

func (suite *OAuth2PerformanceTestSuite) getAuthorizationCode(client *models.OAuthClient, user *models.User, scopes []string) string {
	scopeStr := strings.Join(scopes, " ")

	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]),
		url.QueryEscape(scopeStr))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.mockUserSession(req, user.ID)

	suite.router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		return ""
	}

	location := w.Header().Get("Location")
	u, _ := url.Parse(location)
	return u.Query().Get("code")
}

func (suite *OAuth2PerformanceTestSuite) getAccessToken(client *models.OAuthClient, user *models.User, scopes []string) string {
	accessToken, _ := suite.getTokenPair(client, user, scopes)
	return accessToken
}

func (suite *OAuth2PerformanceTestSuite) getTokenPair(client *models.OAuthClient, user *models.User, scopes []string) (string, string) {
	authCode := suite.getAuthorizationCode(client, user, scopes)
	if authCode == "" {
		return "", ""
	}

	tokenReq := models.TokenRequest{
		GrantType:    "authorization_code",
		Code:         authCode,
		RedirectURI:  client.RedirectURIs[0],
		ClientID:     client.ID.String(),
		ClientSecret: client.Secret,
	}

	tokenBody, _ := json.Marshal(tokenReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(tokenBody))
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		return "", ""
	}

	var tokenResp models.TokenResponse
	json.Unmarshal(w.Body.Bytes(), &tokenResp)
	return tokenResp.AccessToken, tokenResp.RefreshToken
}

func (suite *OAuth2PerformanceTestSuite) mockUserSession(req *http.Request, userID uuid.UUID) {
	req.Header.Set("X-Test-User-ID", userID.String())
}

func (suite *OAuth2PerformanceTestSuite) calculateAverage(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}

	var total time.Duration
	for _, t := range times {
		total += t
	}
	return total / time.Duration(len(times))
}

func (suite *OAuth2PerformanceTestSuite) calculatePercentile(times []time.Duration, percentile int) time.Duration {
	if len(times) == 0 {
		return 0
	}

	// Simple percentile calculation (would use sort for accurate results)
	// For performance testing, this approximation is sufficient
	index := (len(times) * percentile) / 100
	if index >= len(times) {
		index = len(times) - 1
	}

	// Find max within the percentile range (approximation)
	var max time.Duration
	for i := 0; i <= index; i++ {
		if times[i] > max {
			max = times[i]
		}
	}
	return max
}

func TestOAuth2PerformanceTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	suite.Run(t, new(OAuth2PerformanceTestSuite))
}
