package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// OAuth2/OIDC Discovery endpoints

func (as *AuthService) WellKnownOIDC(c *gin.Context) {
	baseURL := getEnv("BASE_URL", "https://ao3.example.com")

	config := models.OIDCDiscoveryDocument{
		Issuer:                baseURL,
		AuthorizationEndpoint: baseURL + "/auth/authorize",
		TokenEndpoint:         baseURL + "/auth/token",
		UserinfoEndpoint:      baseURL + "/auth/userinfo",
		JWKSUri:               baseURL + "/auth/jwks",
		RegistrationEndpoint:  baseURL + "/auth/register-client",
		RevocationEndpoint:    baseURL + "/auth/revoke",
		IntrospectionEndpoint: baseURL + "/auth/introspect",

		ScopesSupported: []string{
			"openid", "profile", "email", "read", "write", "works:manage",
			"comments:write", "bookmarks:manage", "collections:manage",
		},
		ResponseTypesSupported: []string{"code", "code id_token"},
		ResponseModesSupported: []string{"query", "fragment", "form_post"},
		GrantTypesSupported: []string{
			"authorization_code", "refresh_token", "client_credentials",
		},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256", "ES256"},
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_basic", "client_secret_post", "none",
		},
		CodeChallengeMethodsSupported: []string{"S256", "plain"},
		ClaimsSupported: []string{
			"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce",
			"name", "preferred_username", "email", "email_verified",
			"ao3_username", "ao3_display_name", "ao3_roles", "ao3_join_date",
		},
		ServiceDocumentation: baseURL + "/docs/oauth2",
		OpPolicyURI:          baseURL + "/terms",
		OpTosURI:             baseURL + "/privacy",
	}

	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, config)
}

func (as *AuthService) WellKnownOAuth2(c *gin.Context) {
	baseURL := getEnv("BASE_URL", "https://ao3.example.com")

	config := map[string]interface{}{
		"issuer":                                baseURL,
		"authorization_endpoint":                baseURL + "/auth/authorize",
		"token_endpoint":                        baseURL + "/auth/token",
		"jwks_uri":                              baseURL + "/auth/jwks",
		"registration_endpoint":                 baseURL + "/auth/register-client",
		"revocation_endpoint":                   baseURL + "/auth/revoke",
		"introspection_endpoint":                baseURL + "/auth/introspect",
		"scopes_supported":                      []string{"read", "write", "admin"},
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token", "client_credentials"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "none"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
	}

	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, config)
}

// Client Registration

func (as *AuthService) RegisterClient(c *gin.Context) {
	var req models.ClientRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "Invalid client registration request",
		})
		return
	}

	// Validate redirect URIs
	for _, uri := range req.RedirectURIs {
		if !isValidRedirectURI(uri, req.IsPublic) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_redirect_uri",
				"error_description": fmt.Sprintf("Invalid redirect URI: %s", uri),
			})
			return
		}
	}

	// Validate scopes
	for _, scope := range req.Scopes {
		if _, exists := models.AO3OAuthScopes[scope]; !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_scope",
				"error_description": fmt.Sprintf("Invalid scope: %s", scope),
			})
			return
		}
	}

	// Generate client credentials
	clientID := uuid.New()
	var clientSecret string
	if !req.IsPublic {
		secret, err := generateClientSecret()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":             "server_error",
				"error_description": "Failed to generate client secret",
			})
			return
		}
		clientSecret = secret
	}

	// Get owner (if authenticated)
	var ownerID *uuid.UUID
	if userID, exists := c.Get("user_id"); exists {
		uid := userID.(uuid.UUID)
		ownerID = &uid
	}

	// Set defaults
	if req.AccessTokenTTL == 0 {
		req.AccessTokenTTL = 86400 // 24 hours (dev-friendly)
	}
	if req.RefreshTokenTTL == 0 {
		req.RefreshTokenTTL = 2592000 // 30 days
	}

	// Create client
	client := &models.OAuthClient{
		ID:              clientID,
		Secret:          clientSecret,
		Name:            req.Name,
		Description:     req.Description,
		Website:         req.Website,
		LogoURL:         req.LogoURL,
		RedirectURIs:    req.RedirectURIs,
		Scopes:          req.Scopes,
		GrantTypes:      req.GrantTypes,
		ResponseTypes:   req.ResponseTypes,
		IsPublic:        req.IsPublic,
		IsConfidential:  !req.IsPublic,
		IsTrusted:       false, // Admin-only
		IsFirstParty:    false, // Admin-only
		OwnerID:         ownerID,
		AccessTokenTTL:  req.AccessTokenTTL,
		RefreshTokenTTL: req.RefreshTokenTTL,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Hash client secret if present
	if clientSecret != "" {
		hashedSecret, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
			return
		}
		client.Secret = string(hashedSecret)
	}

	// Store in database
	query := `
		INSERT INTO oauth_clients (
			client_id, client_secret, client_name, description, website, logo_url,
			redirect_uris, scopes, grant_types, response_types, is_public, is_confidential,
			is_trusted, is_first_party, owner_id, access_token_ttl, refresh_token_ttl,
			is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`

	_, err := as.db.Exec(query,
		client.ID, client.Secret, client.Name, client.Description, client.Website, client.LogoURL,
		pq.Array(client.RedirectURIs), pq.Array(client.Scopes), pq.Array(client.GrantTypes),
		pq.Array(client.ResponseTypes), client.IsPublic, client.IsConfidential,
		client.IsTrusted, client.IsFirstParty, client.OwnerID, client.AccessTokenTTL,
		client.RefreshTokenTTL, client.IsActive, client.CreatedAt, client.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":             "server_error",
			"error_description": "Failed to register client",
		})
		return
	}

	// Return registration response
	response := models.ClientRegistrationResponse{
		ClientID:        clientID.String(),
		Name:            client.Name,
		Description:     client.Description,
		Website:         client.Website,
		LogoURL:         client.LogoURL,
		RedirectURIs:    client.RedirectURIs,
		Scopes:          client.Scopes,
		GrantTypes:      client.GrantTypes,
		ResponseTypes:   client.ResponseTypes,
		IsPublic:        client.IsPublic,
		AccessTokenTTL:  client.AccessTokenTTL,
		RefreshTokenTTL: client.RefreshTokenTTL,
		CreatedAt:       client.CreatedAt,
	}

	// Include client secret only for confidential clients
	if !client.IsPublic {
		response.ClientSecret = clientSecret
	}

	c.JSON(http.StatusCreated, response)
}

// Authorization endpoint

func (as *AuthService) Authorize(c *gin.Context) {
	var req models.AuthorizeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		as.redirectWithError(c, req.RedirectURI, req.State, "invalid_request", "Invalid authorization request")
		return
	}

	// Validate client
	client, err := as.getClientByID(req.ClientID)
	if err != nil {
		as.redirectWithError(c, req.RedirectURI, req.State, "invalid_client", "Invalid client")
		return
	}

	if !client.IsActive {
		as.redirectWithError(c, req.RedirectURI, req.State, "invalid_client", "Client is disabled")
		return
	}

	// Validate redirect URI
	if !contains(client.RedirectURIs, req.RedirectURI) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "Invalid redirect_uri",
		})
		return
	}

	// Validate response type
	if !contains(client.ResponseTypes, req.ResponseType) {
		as.redirectWithError(c, req.RedirectURI, req.State, "unsupported_response_type", "Response type not supported")
		return
	}

	// Validate scopes
	requestedScopes := strings.Fields(req.Scope)
	if !as.validateScopes(requestedScopes, client.Scopes) {
		as.redirectWithError(c, req.RedirectURI, req.State, "invalid_scope", "Invalid scope")
		return
	}

	// Validate PKCE for public clients
	if client.IsPublic && req.CodeChallenge == "" {
		as.redirectWithError(c, req.RedirectURI, req.State, "invalid_request", "PKCE required for public clients")
		return
	}

	// Check if user is authenticated
	userID := as.getAuthenticatedUser(c)
	if userID == nil {
		// Store authorization request and redirect to login
		authReqID := as.storeAuthorizationRequest(req)
		loginURL := fmt.Sprintf("/login?auth_request=%s", authReqID)
		c.Redirect(http.StatusFound, loginURL)
		return
	}

	// Check consent (skip for trusted clients)
	if !client.IsTrusted && !as.hasValidConsent(*userID, client.ID, requestedScopes) {
		// Show consent screen
		as.showConsentScreen(c, client, requestedScopes, req)
		return
	}

	// Generate authorization code
	code, err := as.generateAuthorizationCode(*userID, client.ID, req)
	if err != nil {
		as.redirectWithError(c, req.RedirectURI, req.State, "server_error", "Failed to generate authorization code")
		return
	}

	// Redirect back to client
	callbackURL := as.buildCallbackURL(req.RedirectURI, code, req.State)
	c.Redirect(http.StatusFound, callbackURL)
}

// Token endpoint

func (as *AuthService) Token(c *gin.Context) {
	var req models.TokenRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.TokenErrorResponse{
			Error:            "invalid_request",
			ErrorDescription: "Invalid token request",
		})
		return
	}

	switch req.GrantType {
	case "authorization_code":
		as.handleAuthorizationCodeGrant(c, req)
	case "refresh_token":
		as.handleRefreshTokenGrant(c, req)
	case "client_credentials":
		as.handleClientCredentialsGrant(c, req)
	default:
		c.JSON(http.StatusBadRequest, models.TokenErrorResponse{
			Error:            "unsupported_grant_type",
			ErrorDescription: "Grant type not supported",
		})
	}
}

func (as *AuthService) handleAuthorizationCodeGrant(c *gin.Context, req models.TokenRequest) {
	// Authenticate client
	client, err := as.authenticateClient(req.ClientID, req.ClientSecret, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.TokenErrorResponse{
			Error:            "invalid_client",
			ErrorDescription: "Client authentication failed",
		})
		return
	}

	// Validate authorization code
	authCode, err := as.validateAuthorizationCode(req.Code, client.ID, req.RedirectURI, req.CodeVerifier)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.TokenErrorResponse{
			Error:            "invalid_grant",
			ErrorDescription: "Invalid authorization code",
		})
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := as.generateTokens(authCode.UserID, client.ID, authCode.Scopes, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.TokenErrorResponse{
			Error:            "server_error",
			ErrorDescription: "Failed to generate tokens",
		})
		return
	}

	// Generate ID token for OIDC
	var idToken string
	if contains(authCode.Scopes, "openid") {
		idToken, err = as.generateIDToken(authCode.UserID, client.ID, authCode.Nonce, authCode.Scopes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.TokenErrorResponse{
				Error:            "server_error",
				ErrorDescription: "Failed to generate ID token",
			})
			return
		}
	}

	// Mark code as used
	as.markCodeAsUsed(authCode.Code)

	// Build response
	response := models.TokenResponse{
		AccessToken:  accessToken.Token,
		TokenType:    "Bearer",
		ExpiresIn:    int(time.Until(accessToken.ExpiresAt).Seconds()),
		RefreshToken: refreshToken.Token,
		Scope:        strings.Join(authCode.Scopes, " "),
	}

	if idToken != "" {
		response.IDToken = idToken
	}

	c.JSON(http.StatusOK, response)
}

func (as *AuthService) handleRefreshTokenGrant(c *gin.Context, req models.TokenRequest) {
	// Authenticate client
	client, err := as.authenticateClient(req.ClientID, req.ClientSecret, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.TokenErrorResponse{
			Error:            "invalid_client",
			ErrorDescription: "Client authentication failed",
		})
		return
	}

	// Validate refresh token
	refreshToken, err := as.validateRefreshToken(req.RefreshToken, client.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.TokenErrorResponse{
			Error:            "invalid_grant",
			ErrorDescription: "Invalid refresh token",
		})
		return
	}

	// Determine scopes (use original scopes or subset)
	scopes := refreshToken.Scopes
	if req.Scope != "" {
		requestedScopes := strings.Fields(req.Scope)
		if !as.isScopeSubset(requestedScopes, refreshToken.Scopes) {
			c.JSON(http.StatusBadRequest, models.TokenErrorResponse{
				Error:            "invalid_scope",
				ErrorDescription: "Requested scope exceeds original grant",
			})
			return
		}
		scopes = requestedScopes
	}

	// Generate new tokens
	newAccessToken, newRefreshToken, err := as.generateTokens(refreshToken.UserID, client.ID, scopes, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.TokenErrorResponse{
			Error:            "server_error",
			ErrorDescription: "Failed to generate tokens",
		})
		return
	}

	// Revoke old refresh token
	as.revokeRefreshToken(refreshToken.ID)

	// Generate new ID token for OIDC
	var idToken string
	if contains(scopes, "openid") {
		idToken, err = as.generateIDToken(refreshToken.UserID, client.ID, "", scopes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.TokenErrorResponse{
				Error:            "server_error",
				ErrorDescription: "Failed to generate ID token",
			})
			return
		}
	}

	// Build response
	response := models.TokenResponse{
		AccessToken:  newAccessToken.Token,
		TokenType:    "Bearer",
		ExpiresIn:    int(time.Until(newAccessToken.ExpiresAt).Seconds()),
		RefreshToken: newRefreshToken.Token,
		Scope:        strings.Join(scopes, " "),
	}

	if idToken != "" {
		response.IDToken = idToken
	}

	c.JSON(http.StatusOK, response)
}

func (as *AuthService) handleClientCredentialsGrant(c *gin.Context, req models.TokenRequest) {
	// Authenticate client
	client, err := as.authenticateClient(req.ClientID, req.ClientSecret, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.TokenErrorResponse{
			Error:            "invalid_client",
			ErrorDescription: "Client authentication failed",
		})
		return
	}

	// Client credentials flow doesn't involve a user
	if !contains(client.GrantTypes, "client_credentials") {
		c.JSON(http.StatusBadRequest, models.TokenErrorResponse{
			Error:            "unauthorized_client",
			ErrorDescription: "Client not authorized for client credentials grant",
		})
		return
	}

	// Determine scopes
	scopes := client.Scopes
	if req.Scope != "" {
		requestedScopes := strings.Fields(req.Scope)
		if !as.validateScopes(requestedScopes, client.Scopes) {
			c.JSON(http.StatusBadRequest, models.TokenErrorResponse{
				Error:            "invalid_scope",
				ErrorDescription: "Invalid scope",
			})
			return
		}
		scopes = requestedScopes
	}

	// Generate access token (no refresh token for client credentials)
	tokenID := uuid.New()
	tokenStr, err := generateSecureToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.TokenErrorResponse{
			Error:            "server_error",
			ErrorDescription: "Failed to generate token",
		})
		return
	}

	expiresAt := time.Now().Add(time.Duration(client.AccessTokenTTL) * time.Second)

	accessToken := &models.OAuthAccessToken{
		ID:        tokenID,
		Token:     tokenStr,
		UserID:    nil, // No user for client credentials
		ClientID:  client.ID,
		Scopes:    scopes,
		TokenType: "Bearer",
		ExpiresAt: expiresAt,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		CreatedAt: time.Now(),
	}

	// Store access token
	err = as.storeAccessToken(accessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.TokenErrorResponse{
			Error:            "server_error",
			ErrorDescription: "Failed to store token",
		})
		return
	}

	// Build response
	response := models.TokenResponse{
		AccessToken: accessToken.Token,
		TokenType:   "Bearer",
		ExpiresIn:   int(time.Until(accessToken.ExpiresAt).Seconds()),
		Scope:       strings.Join(scopes, " "),
	}

	c.JSON(http.StatusOK, response)
}

// User Info endpoint (OIDC)

func (as *AuthService) UserInfo(c *gin.Context) {
	// Extract access token
	token := as.extractBearerToken(c.GetHeader("Authorization"))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":             "invalid_token",
			"error_description": "Missing or invalid access token",
		})
		return
	}

	// Validate access token
	accessToken, err := as.validateAccessToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":             "invalid_token",
			"error_description": "Invalid access token",
		})
		return
	}

	// Check if profile scope is present
	if !contains(accessToken.Scopes, "profile") && !contains(accessToken.Scopes, "openid") {
		c.JSON(http.StatusForbidden, gin.H{
			"error":             "insufficient_scope",
			"error_description": "Profile scope required",
		})
		return
	}

	// Get user info (only for user-based tokens)
	if accessToken.UserID == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_token",
			"error_description": "User info not available for client credentials tokens",
		})
		return
	}

	user, err := as.getUserByID(*accessToken.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":             "server_error",
			"error_description": "Failed to retrieve user info",
		})
		return
	}

	// Build user info response based on scopes
	userInfo := models.UserInfoResponse{
		Subject: accessToken.UserID.String(),
	}

	if contains(accessToken.Scopes, "profile") {
		userInfo.Name = user.DisplayName
		userInfo.PreferredUsername = user.Username
		userInfo.Profile = fmt.Sprintf("https://ao3.example.com/users/%s", user.Username)
		userInfo.UpdatedAt = user.UpdatedAt.Unix()
		userInfo.AO3Username = user.Username
		userInfo.AO3DisplayName = user.DisplayName
		userInfo.AO3JoinDate = user.CreatedAt.Unix()

		// Get user roles
		if roles, err := as.getUserRoles(user.ID); err == nil {
			userInfo.AO3Roles = roles
		}

		// Get user statistics
		if stats, err := as.getUserStats(user.ID); err == nil {
			userInfo.AO3WorkCount = stats.WorkCount
			userInfo.AO3BookmarkCount = stats.BookmarkCount
		}
	}

	if contains(accessToken.Scopes, "email") {
		userInfo.Email = user.Email
		userInfo.EmailVerified = user.IsVerified
	}

	// Update last used timestamp
	go as.updateTokenLastUsed(accessToken.ID)

	c.JSON(http.StatusOK, userInfo)
}

// Token introspection

func (as *AuthService) Introspect(c *gin.Context) {
	var req models.IntrospectRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "Invalid introspection request",
		})
		return
	}

	// Authenticate client
	_, err := as.authenticateClient(req.ClientID, req.ClientSecret, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":             "invalid_client",
			"error_description": "Client authentication failed",
		})
		return
	}

	// Validate token
	accessToken, err := as.validateAccessToken(req.Token)
	if err != nil {
		// Return inactive for invalid tokens
		c.JSON(http.StatusOK, models.IntrospectResponse{Active: false})
		return
	}

	// Build introspection response
	response := models.IntrospectResponse{
		Active:   true,
		Scope:    strings.Join(accessToken.Scopes, " "),
		ClientID: accessToken.ClientID.String(),
	}

	// Add user info if available (not for client credentials)
	if accessToken.UserID != nil {
		user, err := as.getUserByID(*accessToken.UserID)
		if err != nil {
			c.JSON(http.StatusOK, models.IntrospectResponse{Active: false})
			return
		}
		response.Username = user.Username
		response.Subject = accessToken.UserID.String()
	}

	response.TokenType = accessToken.TokenType
	response.ExpiresAt = accessToken.ExpiresAt.Unix()
	response.IssuedAt = accessToken.CreatedAt.Unix()
	response.JWTID = accessToken.ID.String()

	c.JSON(http.StatusOK, response)
}

// Token revocation

func (as *AuthService) Revoke(c *gin.Context) {
	token := c.PostForm("token")
	tokenTypeHint := c.PostForm("token_type_hint")
	clientID := c.PostForm("client_id")
	clientSecret := c.PostForm("client_secret")

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "Missing token parameter",
		})
		return
	}

	// Authenticate client
	_, err := as.authenticateClient(clientID, clientSecret, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":             "invalid_client",
			"error_description": "Client authentication failed",
		})
		return
	}

	// Try to revoke as refresh token first, then access token
	if tokenTypeHint == "refresh_token" || tokenTypeHint == "" {
		if as.revokeRefreshTokenByValue(token) {
			c.Status(http.StatusOK)
			return
		}
	}

	if tokenTypeHint == "access_token" || tokenTypeHint == "" {
		if as.revokeAccessTokenByValue(token) {
			c.Status(http.StatusOK)
			return
		}
	}

	// Return 200 even if token not found (per RFC 7009)
	c.Status(http.StatusOK)
}

// Helper functions

func (as *AuthService) getClientByID(clientID string) (*models.OAuthClient, error) {
	client := &models.OAuthClient{}
	query := `
		SELECT client_id, client_secret, client_name, description, website, logo_url,
			redirect_uris, scopes, grant_types, response_types, is_public, is_confidential,
			is_trusted, is_first_party, owner_id, access_token_ttl, refresh_token_ttl,
			is_active, created_at, updated_at
		FROM oauth_clients 
		WHERE client_id = $1 AND is_active = true`

	err := as.db.QueryRow(query, clientID).Scan(
		&client.ID, &client.Secret, &client.Name, &client.Description, &client.Website, &client.LogoURL,
		pq.Array(&client.RedirectURIs), pq.Array(&client.Scopes), pq.Array(&client.GrantTypes),
		pq.Array(&client.ResponseTypes), &client.IsPublic, &client.IsConfidential,
		&client.IsTrusted, &client.IsFirstParty, &client.OwnerID, &client.AccessTokenTTL,
		&client.RefreshTokenTTL, &client.IsActive, &client.CreatedAt, &client.UpdatedAt)

	return client, err
}

func generateClientSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func isValidRedirectURI(uri string, isPublic bool) bool {
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}

	// For public clients, allow localhost and custom schemes
	if isPublic {
		if parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1") {
			return true
		}
		if parsed.Scheme != "https" && parsed.Scheme != "http" {
			return true // Custom schemes for mobile apps
		}
	}

	// For confidential clients, require HTTPS (except localhost)
	if parsed.Scheme != "https" {
		return parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1")
	}

	return true
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (as *AuthService) validateScopes(requested, allowed []string) bool {
	for _, scope := range requested {
		if !contains(allowed, scope) {
			return false
		}
	}
	return true
}

func (as *AuthService) isScopeSubset(subset, superset []string) bool {
	for _, scope := range subset {
		if !contains(superset, scope) {
			return false
		}
	}
	return true
}

func (as *AuthService) redirectWithError(c *gin.Context, redirectURI, state, errorCode, errorDescription string) {
	if redirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             errorCode,
			"error_description": errorDescription,
		})
		return
	}

	errorURL := fmt.Sprintf("%s?error=%s&error_description=%s",
		redirectURI, url.QueryEscape(errorCode), url.QueryEscape(errorDescription))

	if state != "" {
		errorURL += "&state=" + url.QueryEscape(state)
	}

	c.Redirect(http.StatusFound, errorURL)
}

func (as *AuthService) buildCallbackURL(redirectURI, code, state string) string {
	callbackURL := fmt.Sprintf("%s?code=%s", redirectURI, url.QueryEscape(code))
	if state != "" {
		callbackURL += "&state=" + url.QueryEscape(state)
	}
	return callbackURL
}

func (as *AuthService) extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// Additional helper functions would be implemented here:
// - storeAuthorizationRequest, showConsentScreen, generateAuthorizationCode
// - validateAuthorizationCode, generateTokens, generateIDToken
// - validateAccessToken, validateRefreshToken, etc.
//
// These would handle the database operations and business logic
// for the complete OAuth2/OIDC flow.
