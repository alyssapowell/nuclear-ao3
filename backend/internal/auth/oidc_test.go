package auth

import (
	"bytes"
	"database/sql"
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
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
	testUsers   map[string]*models.User
	testClients map[string]*models.OAuthClient
}

func (suite *OIDCTestSuite) SetupSuite() {
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

	// Initialize test data
	suite.testUsers = make(map[string]*models.User)
	suite.testClients = make(map[string]*models.OAuthClient)
	suite.setupTestData()
}

// Each test starts with the person having already allowed the app, so the
// flows reach the authorization code. The consent tests start by undoing it.
func (suite *OIDCTestSuite) SetupTest() {
	suite.db.Exec(`DELETE FROM oauth_refresh_tokens`)
	suite.db.Exec(`DELETE FROM oauth_access_tokens`)
	suite.db.Exec(`DELETE FROM authorization_codes`)
	suite.grantConsent(suite.testUsers["oidc_user"].ID, suite.testClients["oidc_client"].ID, []string{"openid", "profile", "email"})
}

func (suite *OIDCTestSuite) TearDownSuite() {
	suite.cleanupTestData()
	suite.db.Close()
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
	user, err := seedUser(suite.db, username, email, password)
	suite.Require().NoError(err)
	suite.testUsers[username] = user
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
	assert.Contains(suite.T(), w.Header().Get("Content-Type"), "application/json")

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
	_, err := suite.db.Exec(`DELETE FROM user_consents`)
	suite.Require().NoError(err)

	// First authorization asks the person instead of issuing a code
	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid profile email&state=consent_test",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", authURL, nil)
	suite.signInAs(req, user.ID)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Empty(suite.T(), w.Header().Get("Location"))
	var prompt struct {
		Required  bool   `json:"consent_required"`
		ConsentID string `json:"consent_id"`
	}
	suite.Require().NoError(json.Unmarshal(w.Body.Bytes(), &prompt))
	suite.Require().True(prompt.Required)

	// Someone else cannot answer for them
	other, err := seedUser(suite.db, "oidc_other", "oidc-other@nuclear-ao3.test", "password123")
	suite.Require().NoError(err)
	ow := httptest.NewRecorder()
	oreq, _ := http.NewRequest("POST", "/auth/consent/"+prompt.ConsentID, strings.NewReader("approved=true"))
	oreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	suite.signInAs(oreq, other.ID)
	suite.router.ServeHTTP(ow, oreq)
	assert.Equal(suite.T(), http.StatusForbidden, ow.Code)

	// Agreeing sends them back to the app with a code, and is remembered
	aw := httptest.NewRecorder()
	areq, _ := http.NewRequest("POST", "/auth/consent/"+prompt.ConsentID, strings.NewReader("approved=true"))
	areq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	suite.signInAs(areq, user.ID)
	suite.router.ServeHTTP(aw, areq)
	assert.Equal(suite.T(), http.StatusFound, aw.Code)
	assert.Contains(suite.T(), aw.Header().Get("Location"), "code=")
	assert.Contains(suite.T(), aw.Header().Get("Location"), "state=consent_test")
	assert.True(suite.T(), suite.service.hasValidConsent(user.ID, client.ID, []string{"openid", "profile", "email"}))
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
	suite.signInAs(req, user.ID)

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

	accessToken := suite.getAccessToken(client, user, []string{"openid", "profile"})
	suite.Require().NotEmpty(accessToken)

	// The person ends the app's access from their own login
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/auth/authorized-applications/%s", client.ID), nil)
	suite.signInAs(req, user.ID)
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// The app's token stops working
	uw := httptest.NewRecorder()
	ureq, _ := http.NewRequest("GET", "/auth/userinfo", nil)
	ureq.Header.Set("Authorization", "Bearer "+accessToken)
	suite.router.ServeHTTP(uw, ureq)
	assert.Equal(suite.T(), http.StatusUnauthorized, uw.Code)

	// And the next authorization asks again instead of issuing a code
	authURL := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid profile",
		client.ID.String(),
		url.QueryEscape(client.RedirectURIs[0]))
	authW := httptest.NewRecorder()
	authReq, _ := http.NewRequest("GET", authURL, nil)
	suite.signInAs(authReq, user.ID)
	suite.router.ServeHTTP(authW, authReq)
	assert.Equal(suite.T(), http.StatusOK, authW.Code)
	assert.Empty(suite.T(), authW.Header().Get("Location"))
	assert.Contains(suite.T(), authW.Body.String(), "consent_required")
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
	suite.signInAs(req, user.ID)

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

func (suite *OIDCTestSuite) signInAs(req *http.Request, userID uuid.UUID) {
	suite.Require().NoError(signIn(suite.service, req, userID))
}

func (suite *OIDCTestSuite) grantConsent(userID, clientID uuid.UUID, scopes []string) {
	suite.Require().NoError(suite.service.storeUserConsent(userID, clientID, scopes))
}

func TestOIDCTestSuite(t *testing.T) {
	suite.Run(t, new(OIDCTestSuite))
}
