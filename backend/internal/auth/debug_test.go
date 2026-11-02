package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"nuclear-ao3/internal/testdb"
	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type DebugTestSuite struct {
	suite.Suite
	authService *AuthService
	router      *gin.Engine
}

func (suite *DebugTestSuite) SetupSuite() {
	// Initialize auth service on a fresh test database
	as, err := newTestAuthService(testdb.New(suite.T()))
	suite.Require().NoError(err)
	suite.authService = as
	suite.router = setupRouter(suite.authService)
}

func (suite *DebugTestSuite) TearDownSuite() {
	if suite.authService != nil {
		suite.authService.db.Close()
	}
}

// The client-credentials grant is gone: an app cannot be registered for it,
// and asking for it at the token endpoint is refused even by an app that
// authenticates correctly.
func (suite *DebugTestSuite) TestClientCredentialsGrantIsGone() {
	register := func(grants []string) *httptest.ResponseRecorder {
		clientReq := models.ClientRegistrationRequest{
			Name:          "Test Client",
			Description:   "Test OAuth2 client",
			Website:       "https://example.com",
			RedirectURIs:  []string{"https://example.com/callback"},
			Scopes:        []string{"read", "write"},
			GrantTypes:    grants,
			ResponseTypes: []string{"code"},
			IsPublic:      false,
		}
		reqBody, _ := json.Marshal(clientReq)
		req := httptest.NewRequest("POST", "/auth/register-client", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		suite.Require().NoError(signInAsAdmin(suite.authService, req))
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		return w
	}

	suite.Equal(http.StatusBadRequest, register([]string{"client_credentials"}).Code)
	suite.Equal(http.StatusBadRequest, register([]string{"authorization_code", "client_credentials"}).Code)

	w := register([]string{"authorization_code"})
	suite.Require().Equal(http.StatusCreated, w.Code)
	var clientResp models.ClientRegistrationResponse
	suite.Require().NoError(json.Unmarshal(w.Body.Bytes(), &clientResp))

	tokenData := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientResp.ClientID},
		"client_secret": {clientResp.ClientSecret},
		"scope":         {"read"},
	}
	tokenReq := httptest.NewRequest("POST", "/auth/token", strings.NewReader(tokenData.Encode()))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenW := httptest.NewRecorder()
	suite.router.ServeHTTP(tokenW, tokenReq)

	suite.Equal(http.StatusBadRequest, tokenW.Code)
	var errorResp models.TokenErrorResponse
	suite.Require().NoError(json.Unmarshal(tokenW.Body.Bytes(), &errorResp))
	suite.Equal("unsupported_grant_type", errorResp.Error)
}

func TestDebugTestSuite(t *testing.T) {
	suite.Run(t, new(DebugTestSuite))
}
