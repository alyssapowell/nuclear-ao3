package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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
	// Initialize auth service for testing
	suite.authService = NewAuthService()
	suite.router = setupRouter(suite.authService)
}

func (suite *DebugTestSuite) TearDownSuite() {
	if suite.authService != nil {
		suite.authService.Close()
	}
}

func (suite *DebugTestSuite) TestDebugClientCredentials() {
	// First, register a client
	clientReq := models.ClientRegistrationRequest{
		Name:         "Test Client",
		Description:  "Test OAuth2 client",
		Website:      "https://example.com",
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"read", "write"},
		GrantTypes:   []string{"client_credentials"},
		ResponseTypes: []string{"code"}, // Add required response types
		IsPublic:     false,
	}

	reqBody, _ := json.Marshal(clientReq)
	req := httptest.NewRequest("POST", "/auth/register-client", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Debug: Print client registration response
	suite.T().Logf("Client Registration Status: %d", w.Code)
	suite.T().Logf("Client Registration Body: %s", w.Body.String())
	
	if w.Code != http.StatusCreated {
		suite.T().Fatalf("Client registration failed")
		return
	}

	var clientResp models.ClientRegistrationResponse
	err := json.Unmarshal(w.Body.Bytes(), &clientResp)
	suite.Require().NoError(err)

	// Debug: Print client details
	suite.T().Logf("Client ID: %s", clientResp.ClientID)
	suite.T().Logf("Client Secret: %s", clientResp.ClientSecret)
	
	// Now test client credentials flow
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

	// Debug: Print response
	suite.T().Logf("Token Response Status: %d", tokenW.Code)
	suite.T().Logf("Token Response Body: %s", tokenW.Body.String())
	
	if tokenW.Code != http.StatusOK {
		// Let's see what the error is
		var errorResp models.TokenErrorResponse
		if err := json.Unmarshal(tokenW.Body.Bytes(), &errorResp); err == nil {
			suite.T().Logf("Token Error: %s - %s", errorResp.Error, errorResp.ErrorDescription)
		}
	}
}

func TestDebugTestSuite(t *testing.T) {
	suite.Run(t, new(DebugTestSuite))
}