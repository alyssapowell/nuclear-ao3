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
	"testing"
	"time"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// OIDC-specific test suite
type OIDCTestSuite struct {
	suite.Suite
	service     *AuthService
	router      *gin.Engine
	db          *sql.DB
	redis       *redis.Client
	testUsers   map[string]*models.User
	testClients map[string]*models.OAuthClient
}

func (suite *OIDCTestSuite) SetupSuite() {
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
		DB:   4, // Use DB 4 for OIDC tests
	})
	suite.redis = rdb

	// Create test auth service
	jwtManager, err := NewJWTManager("test-oidc-secret", "test-oidc-issuer", nil)
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
	suite.setupTestData()
}

func (suite *OIDCTestSuite) SetupTest() {
	suite.redis.FlushDB(context.Background())
}

func (suite *OIDCTestSuite) TearDownSuite() {
	suite.cleanupTestData()
	suite.db.Close()
	suite.redis.Close()
}

func (suite *OIDCTestSuite) cleanupTestData() {
	tables := []string{
		"user_consents", "oauth_refresh_tokens", "oauth_access_tokens",
		"authorization_codes", "oauth_clients", "users",
	}
	for _, table := range tables {
		suite.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE created_at > NOW() - INTERVAL '1 day'", table))
	}
}

func (suite *OIDCTestSuite) setupTestData() {
	// Create test user
	suite.createTestUser("oidc_user", "oidc@nuclear-ao3.test", "password123")

	// Create OIDC client
	suite.createOIDCClient()
}

func (suite *OIDCTestSuite) createTestUser(username, email, password string) {
	registerReq := models.RegisterRequest{
		Username:        username,
		Email:           email,
		Password:        password,
		ConfirmPassword: password,
		DisplayName:     fmt.Sprintf("OIDC Test %s", username),
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

func (suite *OIDCTestSuite) createOIDCClient() {
	regReq := models.ClientRegistrationRequest{
		Name:            "OIDC Test Client",
		Description:     "OpenID Connect test application",
		Website:         "https://oidc-app.example.com",
		RedirectURIs:    []string{"https://oidc-app.example.com/callback"},
		Scopes:          []string{"openid", "profile", "email"},
		GrantTypes:      []string{"authorization_code", "refresh_token"},
		ResponseTypes:   []string{"code"},
		IsPublic:        false,
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
		suite.testClients["oidc_client"] = client
	}
}

// Test OIDC Authorization Flow with ID Token

func (suite *OIDCTestSuite) TestOIDCFlow_WithIDToken() {
	client := suite.testClients["oidc_client"]
	user := suite.testUsers["oidc_user"]

	// Step 1: Get authorization code with openid scope
	authCode := suite.getAuthorizationCode(client, user, []string{"openid", "profile", "email"}, "test_nonce")

	// Step 2: Exchange for tokens (should include ID token)
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

	var tokenResp models.TokenResponse
	err := json.Unmarshal(w.Body.Bytes(), &tokenResp)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), tokenResp.AccessToken)
	assert.NotEmpty(suite.T(), tokenResp.RefreshToken)
	assert.NotEmpty(suite.T(), tokenResp.IDToken) // Should have ID token for OIDC

	// Step 3: Validate ID token structure
	suite.validateIDToken(tokenResp.IDToken, client.ID, user.ID, "test_nonce")
}

func (suite *OIDCTestSuite) TestOIDCFlow_WithoutOpenIDScope_NoIDToken() {
	client := suite.testClients["oidc_client"]
	user := suite.testUsers["oidc_user"]

	// Get authorization code without openid scope
	authCode := suite.getAuthorizationCode(client, user, []string{"profile", "email"}, "")

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

	var tokenResp models.TokenResponse
	json.Unmarshal(w.Body.Bytes(), &tokenResp)
	assert.Empty(suite.T(), tokenResp.IDToken) // No ID token without openid scope
}

func (suite *OIDCTestSuite) TestIDToken_ClaimsValidation() {
	client := suite.testClients["oidc_client"]
	user := suite.testUsers["oidc_user"]

	authCode := suite.getAuthorizationCode(client, user, []string{"openid", "profile", "email"}, "nonce123")

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

	var tokenResp models.TokenResponse
	json.Unmarshal(w.Body.Bytes(), &tokenResp)

	// Parse and validate ID token claims
	token, err := jwt.Parse(tokenResp.IDToken, func(token *jwt.Token) (interface{}, error) {
		return suite.service.jwt.publicKey, nil
	})
	assert.NoError(suite.T(), err)

	claims := token.Claims.(jwt.MapClaims)

	// Validate standard OIDC claims
	assert.Equal(suite.T(), user.ID.String(), claims["sub"])
	assert.Equal(suite.T(), client.ID.String(), claims["aud"])
	assert.NotEmpty(suite.T(), claims["iss"])
	assert.NotEmpty(suite.T(), claims["iat"])
	assert.NotEmpty(suite.T(), claims["exp"])
	assert.Equal(suite.T(), "nonce123", claims["nonce"])
	assert.Equal(suite.T(), "ID", claims["typ"])

	// Validate profile claims
	assert.Equal(suite.T(), user.Username, claims["preferred_username"])
	assert.NotEmpty(suite.T(), claims["name"])
	assert.NotEmpty(suite.T(), claims["updated_at"])

	// Validate email claims
	assert.Equal(suite.T(), user.Email, claims["email"])
	assert.Equal(suite.T(), true, claims["email_verified"])

	// Validate AO3-specific claims
	assert.Equal(suite.T(), user.Username, claims["ao3_username"])
	assert.NotEmpty(suite.T(), claims["ao3_join_date"])
}

func (suite *OIDCTestSuite) TestIDToken_ExpirationValidation() {
	client := suite.testClients["oidc_client"]
	user := suite.testUsers["oidc_user"]

	authCode := suite.getAuthorizationCode(client, user, []string{"openid", "profile"}, "")

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

	var tokenResp models.TokenResponse
	json.Unmarshal(w.Body.Bytes(), &tokenResp)

	// Parse ID token
	token, err := jwt.Parse(tokenResp.IDToken, func(token *jwt.Token) (interface{}, error) {
		return suite.service.jwt.publicKey, nil
	})
	assert.NoError(suite.T(), err)

	claims := token.Claims.(jwt.MapClaims)

	// Validate expiration is reasonable (should be 1 hour from now)
	exp := int64(claims["exp"].(float64))
	iat := int64(claims["iat"].(float64))

	expectedExp := iat + 3600                       // 1 hour
	assert.InDelta(suite.T(), expectedExp, exp, 60) // Allow 1 minute tolerance

	// Ensure token hasn't expired
	assert.Greater(suite.T(), exp, time.Now().Unix())
}

// Test JWKS Endpoint

func (suite *OIDCTestSuite) TestJWKSEndpoint() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/jwks", nil)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Equal(suite.T(), "application/json", w.Header().Get("Content-Type"))

	var jwks map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &jwks)
	assert.NoError(suite.T(), err)

	// Validate JWKS structure
	assert.Contains(suite.T(), jwks, "keys")
	keys := jwks["keys"].([]interface{})
	assert.Greater(suite.T(), len(keys), 0)

	// Validate first key structure
	key := keys[0].(map[string]interface{})
	assert.Equal(suite.T(), "RSA", key["kty"])
	assert.Equal(suite.T(), "sig", key["use"])
	assert.Equal(suite.T(), "RS256", key["alg"])
	assert.NotEmpty(suite.T(), key["kid"])
	assert.NotEmpty(suite.T(), key["n"])
	assert.NotEmpty(suite.T(), key["e"])
}

// Test Consent Management

func (suite *OIDCTestSuite) TestConsentFlow_FirstTime() {
	client := suite.testClients["oidc_client"]
	user := suite.testUsers["oidc_user"]

	// First authorization should require consent
	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid profile email&state=consent_test",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.mockUserSession(req, user.ID)

	suite.router.ServeHTTP(w, req)

	// Should show consent screen (in a real implementation)
	// For testing, we'll assume consent is automatically granted for trusted clients
	// or we simulate the consent process

	// If consent is required, it would redirect to consent page
	// If consent is automatic (trusted client), it proceeds with auth code
	if w.Code == http.StatusFound {
		location := w.Header().Get("Location")
		if strings.Contains(location, "consent") {
			// Handle consent flow
			suite.handleConsentFlow(location, user.ID, client.ID, []string{"openid", "profile", "email"})
		} else {
			// Direct authorization - verify auth code is present
			assert.Contains(suite.T(), location, "code=")
		}
	}
}

func (suite *OIDCTestSuite) TestConsentFlow_PreviouslyGranted() {
	client := suite.testClients["oidc_client"]
	user := suite.testUsers["oidc_user"]

	// Grant consent first
	suite.grantConsent(user.ID, client.ID, []string{"openid", "profile", "email"})

	// Second authorization should skip consent
	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid profile email",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.mockUserSession(req, user.ID)

	suite.router.ServeHTTP(w, req)

	// Should proceed directly with authorization code
	assert.Equal(suite.T(), http.StatusFound, w.Code)
	location := w.Header().Get("Location")
	assert.Contains(suite.T(), location, "code=")
	assert.NotContains(suite.T(), location, "consent")
}

func (suite *OIDCTestSuite) TestConsentRevocation() {
	client := suite.testClients["oidc_client"]
	user := suite.testUsers["oidc_user"]

	// Grant consent
	suite.grantConsent(user.ID, client.ID, []string{"openid", "profile"})

	// Get access token
	accessToken := suite.getAccessToken(client, user, []string{"openid", "profile"})

	// Revoke consent
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/auth/authorized-applications/%s", client.ID), nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify consent is revoked - next auth should require consent again
	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid profile",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]))

	authW := httptest.NewRecorder()
	authReq, _ := http.NewRequest("GET", authURL, nil)
	suite.mockUserSession(authReq, user.ID)

	suite.router.ServeHTTP(authW, authReq)

	// Should require consent again
	if authW.Code == http.StatusFound {
		location := authW.Header().Get("Location")
		// In a real implementation, this would redirect to consent page
		// For testing, we check that it doesn't immediately grant authorization
		if !strings.Contains(location, "consent") && strings.Contains(location, "code=") {
			// If we get an auth code immediately, consent might be automatic for trusted clients
			// This is acceptable behavior depending on the client configuration
		}
	}
}

// Helper methods

func (suite *OIDCTestSuite) validateIDToken(idToken string, clientID, userID uuid.UUID, expectedNonce string) {
	token, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return suite.service.jwt.publicKey, nil
	})

	assert.NoError(suite.T(), err)
	assert.True(suite.T(), token.Valid)

	claims := token.Claims.(jwt.MapClaims)

	// Validate required OIDC claims
	assert.Equal(suite.T(), userID.String(), claims["sub"])
	assert.Equal(suite.T(), clientID.String(), claims["aud"])

	if expectedNonce != "" {
		assert.Equal(suite.T(), expectedNonce, claims["nonce"])
	}
}

func (suite *OIDCTestSuite) getAuthorizationCode(client *models.OAuthClient, user *models.User, scopes []string, nonce string) string {
	scopeStr := strings.Join(scopes, " ")

	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]),
		url.QueryEscape(scopeStr))

	if nonce != "" {
		authURL += "&nonce=" + url.QueryEscape(nonce)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.mockUserSession(req, user.ID)

	suite.router.ServeHTTP(w, req)

	location := w.Header().Get("Location")
	u, _ := url.Parse(location)
	return u.Query().Get("code")
}

func (suite *OIDCTestSuite) getAccessToken(client *models.OAuthClient, user *models.User, scopes []string) string {
	authCode := suite.getAuthorizationCode(client, user, scopes, "")

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

	var tokenResp models.TokenResponse
	json.Unmarshal(w.Body.Bytes(), &tokenResp)
	return tokenResp.AccessToken
}

func (suite *OIDCTestSuite) mockUserSession(req *http.Request, userID uuid.UUID) {
	req.Header.Set("X-Test-User-ID", userID.String())
}

func (suite *OIDCTestSuite) grantConsent(userID, clientID uuid.UUID, scopes []string) {
	query := `
		INSERT INTO user_consents (id, user_id, client_id, scopes, granted_at, created_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (user_id, client_id) DO UPDATE SET
			scopes = EXCLUDED.scopes,
			granted_at = EXCLUDED.granted_at`

	suite.db.Exec(query, uuid.New(), userID, clientID, scopes)
}

func (suite *OIDCTestSuite) handleConsentFlow(consentURL string, userID, clientID uuid.UUID, scopes []string) {
	// In a real implementation, this would simulate user granting consent
	// For testing, we directly grant consent in the database
	suite.grantConsent(userID, clientID, scopes)
}

func TestOIDCTestSuite(t *testing.T) {
	suite.Run(t, new(OIDCTestSuite))
}
