package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"nuclear-ao3/internal/testdb"
	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type OAuth2TestSuite struct {
	suite.Suite
	service     *AuthService
	router      *gin.Engine
	db          *sql.DB
	testUsers   map[string]*models.User
	testClients map[string]*models.OAuthClient
}

func (suite *OAuth2TestSuite) SetupSuite() {
	// A fresh, fully migrated database for this suite
	db := testdb.New(suite.T())
	suite.db = db

	// Create test auth service
	service, err := newTestAuthService(db)
	require.NoError(suite.T(), err)
	suite.service = service

	// Setup test router
	gin.SetMode(gin.TestMode)
	suite.router = setupRouter(suite.service)

	// Initialize test data containers
	suite.testUsers = make(map[string]*models.User)
	suite.testClients = make(map[string]*models.OAuthClient)
}

func (suite *OAuth2TestSuite) SetupTest() {
	// Clear test data
	suite.cleanupOAuthTestData()

	// Create fresh test data for each test
	suite.createTestUsers()
	suite.createTestClients()
	suite.consentToEverything()
}

// consentToEverything records that every test user already allowed every test
// client its full scopes, so the flows below reach the authorization code. The
// consent screen itself is tested in TestAuthorize_AsksForConsentFirst.
func (suite *OAuth2TestSuite) consentToEverything() {
	for _, user := range suite.testUsers {
		for _, client := range suite.testClients {
			suite.Require().NoError(suite.service.storeUserConsent(user.ID, client.ID, client.Scopes))
		}
	}
}

func (suite *OAuth2TestSuite) TearDownTest() {
	suite.cleanupOAuthTestData()
}

func (suite *OAuth2TestSuite) TearDownSuite() {
	suite.cleanupOAuthTestData()
	suite.db.Close()
}

func (suite *OAuth2TestSuite) cleanupOAuthTestData() {
	oauthTables := []string{
		"user_consents",
		"oauth_refresh_tokens",
		"oauth_access_tokens",
		"authorization_codes",
		"oauth_clients",
		"refresh_tokens",
		"user_sessions",
		"password_reset_tokens",
		"email_verification_tokens",
		"security_events",
		"user_roles",
		"users",
	}

	for _, table := range oauthTables {
		suite.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE created_at > NOW() - INTERVAL '1 day'", table))
	}
}

func (suite *OAuth2TestSuite) createTestUsers() {
	users := []struct {
		username string
		email    string
		password string
		roles    []string
		verified bool
	}{
		{"oauth_user", "oauth@nuclear-ao3.test", "password123", []string{"user"}, true},
		{"oauth_admin", "oauth-admin@nuclear-ao3.test", "admin123", []string{"user", "admin"}, true},
		{"app_developer", "developer@nuclear-ao3.test", "dev123", []string{"user"}, true},
	}

	for _, u := range users {
		user, err := seedUser(suite.db, u.username, u.email, u.password, u.roles[1:]...)
		suite.Require().NoError(err)
		suite.testUsers[u.username] = user
	}
}

func (suite *OAuth2TestSuite) createTestClients() {
	clients := []struct {
		name          string
		isPublic      bool
		redirectURIs  []string
		scopes        []string
		grantTypes    []string
		responseTypes []string
	}{
		{
			name:          "confidential_client",
			isPublic:      false,
			redirectURIs:  []string{"https://example.com/callback", "https://app.example.com/auth"},
			scopes:        []string{"read", "write", "profile", "email"},
			grantTypes:    []string{"authorization_code", "refresh_token"},
			responseTypes: []string{"code"},
		},
		{
			name:          "public_client",
			isPublic:      true,
			redirectURIs:  []string{"https://mobile-app.com/callback", "myapp://oauth/callback"},
			scopes:        []string{"read", "profile"},
			grantTypes:    []string{"authorization_code", "refresh_token"},
			responseTypes: []string{"code"},
		},
		{
			name:          "first_party_app",
			isPublic:      false,
			redirectURIs:  []string{"https://nuclear-ao3.com/auth/callback"},
			scopes:        []string{"read", "write", "works:manage", "bookmarks:manage", "profile", "email"},
			grantTypes:    []string{"authorization_code", "refresh_token", "client_credentials"},
			responseTypes: []string{"code"},
		},
		{
			name:          "service_client",
			isPublic:      false,
			redirectURIs:  []string{"https://api.external.com/callback"},
			scopes:        []string{"read"},
			grantTypes:    []string{"client_credentials"},
			responseTypes: []string{},
		},
	}

	for _, c := range clients {
		// Create client via registration endpoint
		regReq := models.ClientRegistrationRequest{
			Name:            c.name,
			Description:     fmt.Sprintf("Test OAuth2 client: %s", c.name),
			Website:         "https://example.com",
			RedirectURIs:    c.redirectURIs,
			Scopes:          c.scopes,
			GrantTypes:      c.grantTypes,
			ResponseTypes:   c.responseTypes,
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

// Test OAuth2 Client Registration

func (suite *OAuth2TestSuite) TestClientRegistration_ConfidentialClient_Success() {
	regReq := models.ClientRegistrationRequest{
		Name:            "Test App",
		Description:     "A test application",
		Website:         "https://testapp.example.com",
		RedirectURIs:    []string{"https://testapp.example.com/callback"},
		Scopes:          []string{"read", "profile"},
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

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response models.ClientRegistrationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), response.ClientID)
	assert.NotEmpty(suite.T(), response.ClientSecret)
	assert.Equal(suite.T(), "Test App", response.Name)
	assert.False(suite.T(), response.IsPublic)
}

func (suite *OAuth2TestSuite) TestClientRegistration_PublicClient_Success() {
	regReq := models.ClientRegistrationRequest{
		Name:            "Mobile App",
		Description:     "A mobile application",
		RedirectURIs:    []string{"myapp://oauth/callback"},
		Scopes:          []string{"read", "profile"},
		GrantTypes:      []string{"authorization_code", "refresh_token"},
		ResponseTypes:   []string{"code"},
		IsPublic:        true,
		AccessTokenTTL:  3600,
		RefreshTokenTTL: 86400,
	}

	jsonBody, _ := json.Marshal(regReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/register-client", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response models.ClientRegistrationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), response.ClientID)
	assert.Empty(suite.T(), response.ClientSecret) // Public clients don't get secrets
	assert.True(suite.T(), response.IsPublic)
}

func (suite *OAuth2TestSuite) TestClientRegistration_InvalidRedirectURI() {
	regReq := models.ClientRegistrationRequest{
		Name:          "Invalid App",
		RedirectURIs:  []string{"http://insecure.com/callback"}, // HTTP not allowed for confidential clients
		Scopes:        []string{"read"},
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		IsPublic:      false,
	}

	jsonBody, _ := json.Marshal(regReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/register-client", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *OAuth2TestSuite) TestClientRegistration_InvalidScope() {
	regReq := models.ClientRegistrationRequest{
		Name:          "Invalid App",
		RedirectURIs:  []string{"https://example.com/callback"},
		Scopes:        []string{"invalid_scope"},
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		IsPublic:      false,
	}

	jsonBody, _ := json.Marshal(regReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/register-client", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// Test OIDC Discovery Endpoints

func (suite *OAuth2TestSuite) TestOIDCDiscovery_WellKnownEndpoint() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/openid-configuration", nil)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Contains(suite.T(), w.Header().Get("Content-Type"), "application/json")

	var discovery models.OIDCDiscoveryDocument
	err := json.Unmarshal(w.Body.Bytes(), &discovery)
	assert.NoError(suite.T(), err)

	// Verify required OIDC discovery fields
	assert.NotEmpty(suite.T(), discovery.Issuer)
	assert.NotEmpty(suite.T(), discovery.AuthorizationEndpoint)
	assert.NotEmpty(suite.T(), discovery.TokenEndpoint)
	assert.NotEmpty(suite.T(), discovery.UserinfoEndpoint)
	assert.NotEmpty(suite.T(), discovery.JWKSUri)
	assert.Contains(suite.T(), discovery.ScopesSupported, "openid")
	assert.Contains(suite.T(), discovery.ResponseTypesSupported, "code")
	assert.Contains(suite.T(), discovery.GrantTypesSupported, "authorization_code")
	assert.Contains(suite.T(), discovery.SubjectTypesSupported, "public")
	assert.Contains(suite.T(), discovery.IDTokenSigningAlgValuesSupported, "RS256")
}

func (suite *OAuth2TestSuite) TestOAuth2Discovery_AuthorizationServer() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var discovery map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &discovery)
	assert.NoError(suite.T(), err)

	assert.NotEmpty(suite.T(), discovery["issuer"])
	assert.NotEmpty(suite.T(), discovery["authorization_endpoint"])
	assert.NotEmpty(suite.T(), discovery["token_endpoint"])
	assert.Contains(suite.T(), discovery["scopes_supported"], "read")
}

// Test OAuth2 Authorization Flow

func (suite *OAuth2TestSuite) TestAuthorizationFlow_ConfidentialClient_Success() {
	client := suite.testClients["confidential_client"]
	user := suite.testUsers["oauth_user"]

	// Step 1: Initiate authorization
	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=read profile&state=random_state",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)

	// Mock authenticated user session
	// In a real scenario, this would redirect to login first
	suite.signInAs(req, user.ID)

	suite.router.ServeHTTP(w, req)

	// Should redirect to callback with authorization code
	assert.Equal(suite.T(), http.StatusFound, w.Code)

	location := w.Header().Get("Location")
	assert.Contains(suite.T(), location, client.RedirectURIs[0])
	assert.Contains(suite.T(), location, "code=")
	assert.Contains(suite.T(), location, "state=random_state")

	// Extract authorization code from redirect
	authCode := suite.extractAuthCodeFromRedirect(location)
	assert.NotEmpty(suite.T(), authCode)

	// Step 2: Exchange authorization code for tokens
	tokenReq := models.TokenRequest{
		GrantType:    "authorization_code",
		Code:         authCode,
		RedirectURI:  client.RedirectURIs[0],
		ClientID:     client.ID.String(),
		ClientSecret: client.Secret,
	}

	tokenBody, _ := json.Marshal(tokenReq)
	tokenW := httptest.NewRecorder()
	tokenReq_, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(tokenBody))
	tokenReq_.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(tokenW, tokenReq_)

	assert.Equal(suite.T(), http.StatusOK, tokenW.Code)

	var tokenResp models.TokenResponse
	err := json.Unmarshal(tokenW.Body.Bytes(), &tokenResp)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), tokenResp.AccessToken)
	assert.NotEmpty(suite.T(), tokenResp.RefreshToken)
	assert.Equal(suite.T(), "Bearer", tokenResp.TokenType)
	assert.Greater(suite.T(), tokenResp.ExpiresIn, 0)
}

func (suite *OAuth2TestSuite) TestAuthorizationFlow_PKCERequired_PublicClient() {
	client := suite.testClients["public_client"]

	// Public client attempting authorization without PKCE should fail
	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=read",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.signInAs(req, suite.testUsers["oauth_user"].ID)

	suite.router.ServeHTTP(w, req)

	// Should redirect with error
	assert.Equal(suite.T(), http.StatusFound, w.Code)
	location := w.Header().Get("Location")
	assert.Contains(suite.T(), location, "error=invalid_request")
	assert.Contains(suite.T(), location, "PKCE+required")
}

func (suite *OAuth2TestSuite) TestAuthorizationFlow_PKCE_Success() {
	client := suite.testClients["public_client"]
	user := suite.testUsers["oauth_user"]

	// Generate PKCE challenge
	codeVerifier := suite.generateCodeVerifier()
	codeChallenge := suite.generateCodeChallenge(codeVerifier)

	// Step 1: Authorization with PKCE
	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=read&code_challenge=%s&code_challenge_method=S256&state=pkce_test",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]),
		codeChallenge)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.signInAs(req, user.ID)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusFound, w.Code)

	location := w.Header().Get("Location")
	authCode := suite.extractAuthCodeFromRedirect(location)
	assert.NotEmpty(suite.T(), authCode)

	// Step 2: Token exchange with PKCE verifier
	tokenReq := models.TokenRequest{
		GrantType:    "authorization_code",
		Code:         authCode,
		RedirectURI:  client.RedirectURIs[0],
		ClientID:     client.ID.String(),
		CodeVerifier: codeVerifier,
	}

	tokenBody, _ := json.Marshal(tokenReq)
	tokenW := httptest.NewRecorder()
	tokenReq_, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(tokenBody))
	tokenReq_.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(tokenW, tokenReq_)

	assert.Equal(suite.T(), http.StatusOK, tokenW.Code)

	var tokenResp models.TokenResponse
	err := json.Unmarshal(tokenW.Body.Bytes(), &tokenResp)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), tokenResp.AccessToken)
}

func (suite *OAuth2TestSuite) TestAuthorizationFlow_InvalidPKCE() {
	client := suite.testClients["public_client"]
	user := suite.testUsers["oauth_user"]

	codeVerifier := suite.generateCodeVerifier()
	codeChallenge := suite.generateCodeChallenge(codeVerifier)

	// Step 1: Authorization with PKCE
	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=read&code_challenge=%s&code_challenge_method=S256",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]),
		codeChallenge)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.signInAs(req, user.ID)

	suite.router.ServeHTTP(w, req)

	authCode := suite.extractAuthCodeFromRedirect(w.Header().Get("Location"))

	// Step 2: Token exchange with wrong PKCE verifier
	tokenReq := models.TokenRequest{
		GrantType:    "authorization_code",
		Code:         authCode,
		RedirectURI:  client.RedirectURIs[0],
		ClientID:     client.ID.String(),
		CodeVerifier: "wrong_verifier",
	}

	tokenBody, _ := json.Marshal(tokenReq)
	tokenW := httptest.NewRecorder()
	tokenReq_, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(tokenBody))
	tokenReq_.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(tokenW, tokenReq_)

	assert.Equal(suite.T(), http.StatusBadRequest, tokenW.Code)

	var errorResp models.TokenErrorResponse
	json.Unmarshal(tokenW.Body.Bytes(), &errorResp)
	assert.Equal(suite.T(), "invalid_grant", errorResp.Error)
}

// Helper methods

func (suite *OAuth2TestSuite) signInAs(req *http.Request, userID uuid.UUID) {
	suite.Require().NoError(signIn(suite.service, req, userID))
}

func (suite *OAuth2TestSuite) extractAuthCodeFromRedirect(location string) string {
	u, _ := url.Parse(location)
	return u.Query().Get("code")
}

func (suite *OAuth2TestSuite) generateCodeVerifier() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
}

func (suite *OAuth2TestSuite) generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
}

// Test Token Introspection

func (suite *OAuth2TestSuite) TestTokenIntrospection_ValidToken() {
	// First get a valid access token
	client := suite.testClients["confidential_client"]
	accessToken := suite.getValidAccessToken(client, suite.testUsers["oauth_user"], []string{"read", "profile"})

	// Introspect the token
	introspectReq := models.IntrospectRequest{
		Token:        accessToken,
		ClientID:     client.ID.String(),
		ClientSecret: client.Secret,
	}

	introspectBody, _ := json.Marshal(introspectReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/introspect", bytes.NewBuffer(introspectBody))
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.IntrospectResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Active)
	assert.Equal(suite.T(), "read profile", response.Scope)
	assert.Equal(suite.T(), client.ID.String(), response.ClientID)
	assert.Equal(suite.T(), "oauth_user", response.Username)
	assert.Equal(suite.T(), "Bearer", response.TokenType)
	assert.Greater(suite.T(), response.ExpiresAt, time.Now().Unix())
}

func (suite *OAuth2TestSuite) TestTokenIntrospection_InvalidToken() {
	client := suite.testClients["confidential_client"]

	introspectReq := models.IntrospectRequest{
		Token:        "invalid.token.here",
		ClientID:     client.ID.String(),
		ClientSecret: client.Secret,
	}

	introspectBody, _ := json.Marshal(introspectReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/introspect", bytes.NewBuffer(introspectBody))
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.IntrospectResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Active)
}

func (suite *OAuth2TestSuite) TestTokenIntrospection_InvalidClient() {
	accessToken := suite.getValidAccessToken(suite.testClients["confidential_client"], suite.testUsers["oauth_user"], []string{"read"})

	introspectReq := models.IntrospectRequest{
		Token:        accessToken,
		ClientID:     "invalid-client-id",
		ClientSecret: "invalid-secret",
	}

	introspectBody, _ := json.Marshal(introspectReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/introspect", bytes.NewBuffer(introspectBody))
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// Test OIDC UserInfo Endpoint

func (suite *OAuth2TestSuite) TestUserInfo_ValidToken_ProfileScope() {
	// Get access token with profile scope
	client := suite.testClients["confidential_client"]
	accessToken := suite.getValidAccessToken(client, suite.testUsers["oauth_user"], []string{"profile"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var userInfo models.UserInfoResponse
	err := json.Unmarshal(w.Body.Bytes(), &userInfo)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testUsers["oauth_user"].ID.String(), userInfo.Subject)
	assert.Equal(suite.T(), "oauth_user", userInfo.PreferredUsername)
	assert.NotEmpty(suite.T(), userInfo.Name)
	assert.Greater(suite.T(), userInfo.UpdatedAt, int64(0))
	assert.Empty(suite.T(), userInfo.Email) // Email scope not requested
}

func (suite *OAuth2TestSuite) TestUserInfo_ValidToken_EmailScope() {
	client := suite.testClients["confidential_client"]
	accessToken := suite.getValidAccessToken(client, suite.testUsers["oauth_user"], []string{"profile", "email"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var userInfo models.UserInfoResponse
	err := json.Unmarshal(w.Body.Bytes(), &userInfo)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "oauth@nuclear-ao3.test", userInfo.Email)
	assert.True(suite.T(), userInfo.EmailVerified)
}

func (suite *OAuth2TestSuite) TestUserInfo_InsufficientScope() {
	client := suite.testClients["confidential_client"]
	accessToken := suite.getValidAccessToken(client, suite.testUsers["oauth_user"], []string{"read"}) // No profile scope

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

func (suite *OAuth2TestSuite) TestUserInfo_InvalidToken() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer invalid.token")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// Test Token Revocation

func (suite *OAuth2TestSuite) TestTokenRevocation_AccessToken() {
	client := suite.testClients["confidential_client"]
	accessToken := suite.getValidAccessToken(client, suite.testUsers["oauth_user"], []string{"read"})

	// Revoke the token
	data := url.Values{}
	data.Set("token", accessToken)
	data.Set("token_type_hint", "access_token")
	data.Set("client_id", client.ID.String())
	data.Set("client_secret", client.Secret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/revoke", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify token is now invalid
	introspectReq := models.IntrospectRequest{
		Token:        accessToken,
		ClientID:     client.ID.String(),
		ClientSecret: client.Secret,
	}

	introspectBody, _ := json.Marshal(introspectReq)
	introspectW := httptest.NewRecorder()
	introspectReq_, _ := http.NewRequest("POST", "/auth/introspect", bytes.NewBuffer(introspectBody))
	introspectReq_.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(introspectW, introspectReq_)

	var introspectResp models.IntrospectResponse
	json.Unmarshal(introspectW.Body.Bytes(), &introspectResp)
	assert.False(suite.T(), introspectResp.Active)
}

func (suite *OAuth2TestSuite) TestTokenRevocation_RefreshToken() {
	client := suite.testClients["confidential_client"]
	_, refreshToken := suite.getValidTokenPair(client, suite.testUsers["oauth_user"], []string{"read"})

	// Revoke refresh token
	data := url.Values{}
	data.Set("token", refreshToken)
	data.Set("token_type_hint", "refresh_token")
	data.Set("client_id", client.ID.String())
	data.Set("client_secret", client.Secret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/revoke", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Try to use refresh token - should fail
	tokenReq := models.TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     client.ID.String(),
		ClientSecret: client.Secret,
	}

	tokenBody, _ := json.Marshal(tokenReq)
	tokenW := httptest.NewRecorder()
	tokenReq_, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(tokenBody))
	tokenReq_.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(tokenW, tokenReq_)

	assert.Equal(suite.T(), http.StatusBadRequest, tokenW.Code)
}

// Test Scope Validation

func (suite *OAuth2TestSuite) TestScopeValidation_InvalidScope() {
	client := suite.testClients["confidential_client"]

	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=admin",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.signInAs(req, suite.testUsers["oauth_user"].ID)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusFound, w.Code)
	location := w.Header().Get("Location")
	assert.Contains(suite.T(), location, "error=invalid_scope")
}

func (suite *OAuth2TestSuite) TestScopeValidation_ExceedsClientScopes() {
	client := suite.testClients["public_client"] // Only has read, profile scopes

	codeVerifier := suite.generateCodeVerifier()
	codeChallenge := suite.generateCodeChallenge(codeVerifier)

	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=read write&code_challenge=%s&code_challenge_method=S256",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]),
		codeChallenge)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.signInAs(req, suite.testUsers["oauth_user"].ID)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusFound, w.Code)
	location := w.Header().Get("Location")
	assert.Contains(suite.T(), location, "error=invalid_scope")
}

// Test Client Credentials Grant

func (suite *OAuth2TestSuite) TestClientCredentials_Success() {
	client := suite.testClients["service_client"]

	tokenReq := models.TokenRequest{
		GrantType:    "client_credentials",
		Scope:        "read",
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
	assert.Empty(suite.T(), tokenResp.RefreshToken) // No refresh token for client credentials
	assert.Equal(suite.T(), "Bearer", tokenResp.TokenType)
}

func (suite *OAuth2TestSuite) TestClientCredentials_UnauthorizedClient() {
	client := suite.testClients["public_client"] // Doesn't support client_credentials

	tokenReq := models.TokenRequest{
		GrantType: "client_credentials",
		Scope:     "read",
		ClientID:  client.ID.String(),
	}

	tokenBody, _ := json.Marshal(tokenReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(tokenBody))
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// Test Refresh Token Flow

func (suite *OAuth2TestSuite) TestRefreshToken_Success() {
	client := suite.testClients["confidential_client"]
	_, refreshToken := suite.getValidTokenPair(client, suite.testUsers["oauth_user"], []string{"read", "profile"})

	tokenReq := models.TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
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
	assert.NotEqual(suite.T(), refreshToken, tokenResp.RefreshToken) // Should be new refresh token
}

func (suite *OAuth2TestSuite) TestRefreshToken_ScopeReduction() {
	client := suite.testClients["confidential_client"]
	_, refreshToken := suite.getValidTokenPair(client, suite.testUsers["oauth_user"], []string{"read", "profile"})

	// Request subset of original scopes
	tokenReq := models.TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		Scope:        "read", // Subset of original scopes
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
	assert.Equal(suite.T(), "read", tokenResp.Scope)
}

func (suite *OAuth2TestSuite) TestRefreshToken_ScopeExpansion_ShouldFail() {
	client := suite.testClients["confidential_client"]
	_, refreshToken := suite.getValidTokenPair(client, suite.testUsers["oauth_user"], []string{"read"})

	// Try to request more scopes than originally granted
	tokenReq := models.TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		Scope:        "read write", // More than original
		ClientID:     client.ID.String(),
		ClientSecret: client.Secret,
	}

	tokenBody, _ := json.Marshal(tokenReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(tokenBody))
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// Test Security and Edge Cases

func (suite *OAuth2TestSuite) TestAuthorizationCode_DoubleUse_ShouldFail() {
	client := suite.testClients["confidential_client"]
	user := suite.testUsers["oauth_user"]

	// Get authorization code
	authCode := suite.getAuthorizationCode(client, user, []string{"read"})

	// Use it once
	tokenReq := models.TokenRequest{
		GrantType:    "authorization_code",
		Code:         authCode,
		RedirectURI:  client.RedirectURIs[0],
		ClientID:     client.ID.String(),
		ClientSecret: client.Secret,
	}

	tokenBody, _ := json.Marshal(tokenReq)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(tokenBody))
	req1.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w1, req1)
	assert.Equal(suite.T(), http.StatusOK, w1.Code)

	// Try to use it again - should fail
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(tokenBody))
	req2.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w2, req2)
	assert.Equal(suite.T(), http.StatusBadRequest, w2.Code)
}

func (suite *OAuth2TestSuite) TestInvalidRedirectURI() {
	client := suite.testClients["confidential_client"]

	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=read",
		client.ID.String(),
		url.QueryEscape("https://evil.com/callback"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.signInAs(req, suite.testUsers["oauth_user"].ID)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *OAuth2TestSuite) TestMissingClientID() {
	authURL := "/auth/authorize?response_type=code&redirect_uri=https://example.com/callback&scope=read"

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// Helper methods for token generation

func (suite *OAuth2TestSuite) getValidAccessToken(client *models.OAuthClient, user *models.User, scopes []string) string {
	accessToken, _ := suite.getValidTokenPair(client, user, scopes)
	return accessToken
}

func (suite *OAuth2TestSuite) getValidTokenPair(client *models.OAuthClient, user *models.User, scopes []string) (string, string) {
	authCode := suite.getAuthorizationCode(client, user, scopes)

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
	return tokenResp.AccessToken, tokenResp.RefreshToken
}

func (suite *OAuth2TestSuite) getAuthorizationCode(client *models.OAuthClient, user *models.User, scopes []string) string {
	scopeStr := strings.Join(scopes, " ")

	var authURL string
	if client.IsPublic {
		codeVerifier := suite.generateCodeVerifier()
		codeChallenge := suite.generateCodeChallenge(codeVerifier)
		authURL = fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&code_challenge=%s&code_challenge_method=S256",
			client.ID.String(),
			url.QueryEscape(client.RedirectURIs[0]),
			url.QueryEscape(scopeStr),
			codeChallenge)
	} else {
		authURL = fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s",
			client.ID.String(),
			url.QueryEscape(client.RedirectURIs[0]),
			url.QueryEscape(scopeStr))
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.signInAs(req, user.ID)

	suite.router.ServeHTTP(w, req)

	location := w.Header().Get("Location")
	return suite.extractAuthCodeFromRedirect(location)
}

// Without a recorded consent, an app gets no code: the person is asked first.
func (suite *OAuth2TestSuite) TestAuthorize_AsksForConsentFirst() {
	client := suite.testClients["confidential_client"]
	user := suite.testUsers["oauth_user"]
	_, err := suite.db.Exec(`DELETE FROM user_consents WHERE user_id = $1`, user.ID)
	suite.Require().NoError(err)

	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=read&state=s1",
		client.ID, url.QueryEscape(client.RedirectURIs[0]))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.signInAs(req, user.ID)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Empty(suite.T(), w.Header().Get("Location"))
	var body map[string]any
	assert.NoError(suite.T(), json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(suite.T(), true, body["consent_required"])
	var codes int
	suite.db.QueryRow(`SELECT count(*) FROM authorization_codes`).Scan(&codes)
	assert.Equal(suite.T(), 0, codes, "no code before the person agrees")
}

func TestOAuth2TestSuite(t *testing.T) {
	suite.Run(t, new(OAuth2TestSuite))
}
