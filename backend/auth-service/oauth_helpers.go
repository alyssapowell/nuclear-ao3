package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Client authentication

func (as *AuthService) authenticateClient(clientID, clientSecret string, r *http.Request) (*models.OAuthClient, error) {
	client, err := as.getClientByID(clientID)
	if err != nil {
		return nil, fmt.Errorf("client not found")
	}

	// Public clients don't need authentication
	if client.IsPublic {
		if clientSecret != "" {
			return nil, fmt.Errorf("public clients must not send secret")
		}
		return client, nil
	}

	// Confidential clients must authenticate
	if clientSecret == "" {
		// Try HTTP Basic Auth
		basicClientID, basicSecret, ok := r.BasicAuth()
		if !ok {
			return nil, fmt.Errorf("client authentication required")
		}
		if basicClientID != clientID {
			return nil, fmt.Errorf("client ID mismatch")
		}
		clientSecret = basicSecret
	}

	// Verify client secret
	if err := bcrypt.CompareHashAndPassword([]byte(client.Secret), []byte(clientSecret)); err != nil {
		return nil, fmt.Errorf("invalid client secret")
	}

	return client, nil
}

// Authorization request handling

func (as *AuthService) storeAuthorizationRequest(req models.AuthorizeRequest) string {
	requestID := uuid.New().String()

	// Store in Redis with 10 minute expiry
	reqJSON, _ := json.Marshal(req)
	as.redis.Set(context.Background(), fmt.Sprintf("auth_req:%s", requestID), reqJSON, time.Minute*10)

	return requestID
}

func (as *AuthService) getAuthorizationRequest(requestID string) (*models.AuthorizeRequest, error) {
	reqJSON, err := as.redis.Get(context.Background(), fmt.Sprintf("auth_req:%s", requestID)).Result()
	if err != nil {
		return nil, err
	}

	var req models.AuthorizeRequest
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		return nil, err
	}

	return &req, nil
}

func (as *AuthService) getAuthenticatedUser(c *gin.Context) *uuid.UUID {
	// Check if user_id is already set by middleware
	if userID, exists := c.Get("user_id"); exists {
		uid := userID.(uuid.UUID)
		return &uid
	}

	// Check for test user header (for testing OAuth flows)
	if testUserID := c.GetHeader("X-Test-User-ID"); testUserID != "" && gin.Mode() == gin.TestMode {
		if userID, err := uuid.Parse(testUserID); err == nil {
			c.Set("user_id", userID)
			return &userID
		}
	}

	// Check Authorization header for JWT token
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		tokenString := extractBearerToken(authHeader)
		if tokenString != "" {
			if claims, err := as.jwt.ValidateToken(tokenString); err == nil {
				if userID, err := uuid.Parse(claims.Subject); err == nil {
					c.Set("user_id", userID)
					return &userID
				}
			}
		}
	}

	// Check session cookie
	if sessionID, err := c.Cookie("session_id"); err == nil {
		if userID := as.getUserFromSession(sessionID); userID != nil {
			return userID
		}
	}

	return nil
}

func (as *AuthService) getUserFromSession(sessionID string) *uuid.UUID {
	// Get user ID from Redis session
	userIDStr, err := as.redis.Get(context.Background(), fmt.Sprintf("session:%s", sessionID)).Result()
	if err != nil {
		return nil
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil
	}

	return &userID
}

// Consent management

func (as *AuthService) hasValidConsent(userID, clientID uuid.UUID, scopes []string) bool {
	// Check if user has already consented to these scopes for this client
	query := `
		SELECT scopes FROM user_consents 
		WHERE user_id = $1 AND client_id = $2 AND is_revoked = false 
		AND (expires_at IS NULL OR expires_at > NOW())`

	var consentedScopes []string
	err := as.db.QueryRow(query, userID, clientID).Scan(pq.Array(&consentedScopes))
	if err != nil {
		return false
	}

	// Check if all requested scopes are covered by consent
	return as.isScopeSubset(scopes, consentedScopes)
}

func (as *AuthService) showConsentScreen(c *gin.Context, client *models.OAuthClient, scopes []string, req models.AuthorizeRequest) {
	// Build scope descriptions
	scopeDescriptions := make(map[string]string)
	for _, scope := range scopes {
		if scopeInfo, exists := models.AO3OAuthScopes[scope]; exists {
			scopeDescriptions[scope] = scopeInfo.Description
		}
	}

	// Store consent request
	consentID := uuid.New().String()
	consentData := map[string]interface{}{
		"client":             client,
		"scopes":             scopes,
		"scope_descriptions": scopeDescriptions,
		"authorize_request":  req,
	}

	consentJSON, _ := json.Marshal(consentData)
	as.redis.Set(context.Background(), fmt.Sprintf("consent:%s", consentID), consentJSON, time.Minute*10)

	// For test mode, automatically approve consent; in production, render HTML
	if gin.Mode() == gin.TestMode {
		// Auto-approve consent for testing
		as.processConsent(c, consentID, true)
		return
	}

	// In production, would render actual consent screen HTML
	// For now, return JSON response indicating consent is needed
	c.JSON(http.StatusOK, gin.H{
		"consent_required": true,
		"consent_id":       consentID,
		"client_name":      client.Name,
		"scopes":           scopes,
		"scope_descriptions": scopeDescriptions,
		"consent_url":      fmt.Sprintf("/auth/consent/%s", consentID),
		"cancel_url":       req.RedirectURI + "?error=access_denied&state=" + req.State,
	})
}

func (as *AuthService) processConsent(c *gin.Context, consentID string, approved bool) {
	// Get consent data
	consentJSON, err := as.redis.Get(context.Background(), fmt.Sprintf("consent:%s", consentID)).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consent request"})
		return
	}

	var consentData struct {
		Client           *models.OAuthClient     `json:"client"`
		Scopes           []string                `json:"scopes"`
		AuthorizeRequest models.AuthorizeRequest `json:"authorize_request"`
	}
	if err := json.Unmarshal([]byte(consentJSON), &consentData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consent data"})
		return
	}

	// Extract data
	clientID := consentData.Client.ID
	scopes := consentData.Scopes

	userID := as.getAuthenticatedUser(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	if !approved {
		// User denied consent
		req := consentData.AuthorizeRequest
		as.redirectWithError(c, req.RedirectURI, req.State, "access_denied", "User denied access")
		return
	}

	// Store consent
	as.storeUserConsent(*userID, clientID, scopes)

	// Continue with authorization
	req := consentData.AuthorizeRequest
	code, err := as.generateAuthorizationCode(*userID, clientID, req)
	if err != nil {
		as.redirectWithError(c, req.RedirectURI, req.State, "server_error", "Failed to generate code")
		return
	}

	callbackURL := as.buildCallbackURL(req.RedirectURI, code, req.State)
	c.Redirect(http.StatusFound, callbackURL)
}

func (as *AuthService) storeUserConsent(userID, clientID uuid.UUID, scopes []string) error {
	query := `
		INSERT INTO user_consents (id, user_id, client_id, scopes, granted_at, is_revoked)
		VALUES ($1, $2, $3, $4, $5, false)
		ON CONFLICT (user_id, client_id) 
		DO UPDATE SET scopes = $4, granted_at = $5, is_revoked = false`

	_, err := as.db.Exec(query, uuid.New(), userID, clientID, pq.Array(scopes), time.Now())
	return err
}

// Authorization code management

func (as *AuthService) generateAuthorizationCode(userID, clientID uuid.UUID, req models.AuthorizeRequest) (string, error) {
	// Generate secure code
	codeBytes := make([]byte, 32)
	if _, err := rand.Read(codeBytes); err != nil {
		return "", err
	}
	code := base64.URLEncoding.EncodeToString(codeBytes)

	// Store authorization code
	authCode := &models.AuthorizationCode{
		Code:                code,
		ClientID:            clientID,
		UserID:              userID,
		RedirectURI:         req.RedirectURI,
		Scopes:              strings.Fields(req.Scope),
		State:               req.State,
		Nonce:               req.Nonce,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(time.Minute * 10), // 10 minute expiry
		CreatedAt:           time.Now(),
	}

	query := `
		INSERT INTO authorization_codes (
			code, client_id, user_id, redirect_uri, scopes, state, nonce,
			code_challenge, code_challenge_method, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := as.db.Exec(query,
		authCode.Code, authCode.ClientID, authCode.UserID, authCode.RedirectURI,
		pq.Array(authCode.Scopes), authCode.State, authCode.Nonce,
		authCode.CodeChallenge, authCode.CodeChallengeMethod,
		authCode.ExpiresAt, authCode.CreatedAt)

	return code, err
}

func (as *AuthService) validateAuthorizationCode(code string, clientID uuid.UUID, redirectURI, codeVerifier string) (*models.AuthorizationCode, error) {
	authCode := &models.AuthorizationCode{}

	query := `
		SELECT code, client_id, user_id, redirect_uri, scopes, state, nonce,
			code_challenge, code_challenge_method, expires_at, used_at, created_at
		FROM authorization_codes 
		WHERE code = $1 AND client_id = $2 AND used_at IS NULL`

	err := as.db.QueryRow(query, code, clientID).Scan(
		&authCode.Code, &authCode.ClientID, &authCode.UserID, &authCode.RedirectURI,
		pq.Array(&authCode.Scopes), &authCode.State, &authCode.Nonce,
		&authCode.CodeChallenge, &authCode.CodeChallengeMethod,
		&authCode.ExpiresAt, &authCode.UsedAt, &authCode.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("invalid authorization code")
	}

	// Check expiry
	if time.Now().After(authCode.ExpiresAt) {
		return nil, fmt.Errorf("authorization code expired")
	}

	// Check redirect URI
	if authCode.RedirectURI != redirectURI {
		return nil, fmt.Errorf("redirect URI mismatch")
	}

	// Validate PKCE if present
	if authCode.CodeChallenge != "" {
		if codeVerifier == "" {
			return nil, fmt.Errorf("code verifier required")
		}

		if !as.validatePKCE(authCode.CodeChallenge, authCode.CodeChallengeMethod, codeVerifier) {
			return nil, fmt.Errorf("invalid code verifier")
		}
	}

	return authCode, nil
}

func (as *AuthService) validatePKCE(codeChallenge, codeChallengeMethod, codeVerifier string) bool {
	switch codeChallengeMethod {
	case "S256":
		hash := sha256.Sum256([]byte(codeVerifier))
		computed := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
		return computed == codeChallenge
	case "plain":
		return codeVerifier == codeChallenge
	default:
		return false
	}
}

func (as *AuthService) markCodeAsUsed(code string) {
	query := `UPDATE authorization_codes SET used_at = NOW() WHERE code = $1`
	as.db.Exec(query, code)
}

// Token management

func (as *AuthService) generateTokens(userID, clientID uuid.UUID, scopes []string, ipAddress, userAgent string) (*models.OAuthAccessToken, *models.OAuthRefreshToken, error) {
	// Generate access token
	accessTokenStr, err := generateSecureToken()
	if err != nil {
		return nil, nil, err
	}

	refreshTokenStr, err := generateSecureToken()
	if err != nil {
		return nil, nil, err
	}

	// Get client to determine TTL
	client, err := as.getClientByID(clientID.String())
	if err != nil {
		return nil, nil, err
	}

	accessToken := &models.OAuthAccessToken{
		ID:        uuid.New(),
		Token:     accessTokenStr,
		UserID:    &userID,
		ClientID:  clientID,
		Scopes:    scopes,
		TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Duration(client.AccessTokenTTL) * time.Second),
		IPAddress: ipAddress,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
	}

	refreshToken := &models.OAuthRefreshToken{
		ID:            uuid.New(),
		Token:         refreshTokenStr,
		AccessTokenID: accessToken.ID,
		UserID:        userID,
		ClientID:      clientID,
		Scopes:        scopes,
		ExpiresAt:     time.Now().Add(time.Duration(client.RefreshTokenTTL) * time.Second),
		CreatedAt:     time.Now(),
	}

	// Store tokens in database
	if err := as.storeAccessToken(accessToken); err != nil {
		return nil, nil, err
	}

	if err := as.storeRefreshToken(refreshToken); err != nil {
		return nil, nil, err
	}

	return accessToken, refreshToken, nil
}

func (as *AuthService) storeAccessToken(token *models.OAuthAccessToken) error {
	query := `
		INSERT INTO oauth_access_tokens (
			id, token, user_id, client_id, scopes, token_type, expires_at,
			is_revoked, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, false, $8, $9, $10)`

	_, err := as.db.Exec(query,
		token.ID, token.Token, token.UserID, token.ClientID, pq.Array(token.Scopes),
		token.TokenType, token.ExpiresAt, token.IPAddress, token.UserAgent, token.CreatedAt)

	return err
}

func (as *AuthService) storeRefreshToken(token *models.OAuthRefreshToken) error {
	query := `
		INSERT INTO oauth_refresh_tokens (
			id, token, access_token_id, user_id, client_id, scopes, expires_at,
			is_revoked, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, false, $8)`

	_, err := as.db.Exec(query,
		token.ID, token.Token, token.AccessTokenID, token.UserID, token.ClientID,
		pq.Array(token.Scopes), token.ExpiresAt, token.CreatedAt)

	return err
}

func (as *AuthService) validateAccessToken(token string) (*models.OAuthAccessToken, error) {
	accessToken := &models.OAuthAccessToken{}

	query := `
		SELECT id, token, user_id, client_id, scopes, token_type, expires_at,
			is_revoked, last_used, ip_address, user_agent, created_at
		FROM oauth_access_tokens 
		WHERE token = $1 AND is_revoked = false`

	err := as.db.QueryRow(query, token).Scan(
		&accessToken.ID, &accessToken.Token, &accessToken.UserID, &accessToken.ClientID,
		pq.Array(&accessToken.Scopes), &accessToken.TokenType, &accessToken.ExpiresAt,
		&accessToken.IsRevoked, &accessToken.LastUsed, &accessToken.IPAddress,
		&accessToken.UserAgent, &accessToken.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("invalid access token")
	}

	// Check expiry
	if time.Now().After(accessToken.ExpiresAt) {
		return nil, fmt.Errorf("access token expired")
	}

	return accessToken, nil
}

func (as *AuthService) validateRefreshToken(token string, clientID uuid.UUID) (*models.OAuthRefreshToken, error) {
	refreshToken := &models.OAuthRefreshToken{}

	query := `
		SELECT id, token, access_token_id, user_id, client_id, scopes, expires_at,
			is_revoked, last_used, created_at
		FROM oauth_refresh_tokens 
		WHERE token = $1 AND client_id = $2 AND is_revoked = false`

	err := as.db.QueryRow(query, token, clientID).Scan(
		&refreshToken.ID, &refreshToken.Token, &refreshToken.AccessTokenID,
		&refreshToken.UserID, &refreshToken.ClientID, pq.Array(&refreshToken.Scopes),
		&refreshToken.ExpiresAt, &refreshToken.IsRevoked, &refreshToken.LastUsed,
		&refreshToken.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Check expiry
	if time.Now().After(refreshToken.ExpiresAt) {
		return nil, fmt.Errorf("refresh token expired")
	}

	return refreshToken, nil
}

// OIDC ID Token generation

func (as *AuthService) generateIDToken(userID, clientID uuid.UUID, nonce string, scopes []string) (string, error) {
	user, err := as.getUserByID(userID)
	if err != nil {
		return "", err
	}

	now := time.Now()
	baseURL := getEnv("BASE_URL", "https://ao3.example.com")

	// Set auth_time to last login or current time if never logged in
	authTime := now.Unix()
	if user.LastLoginAt != nil {
		authTime = user.LastLoginAt.Unix()
	}

	claims := models.OIDCClaims{
		Issuer:    baseURL,
		Subject:   userID.String(),
		Audience:  clientID.String(),
		ExpiresAt: now.Add(time.Hour).Unix(),
		IssuedAt:  now.Unix(),
		AuthTime:  authTime,
		Nonce:     nonce,
	}

	// Add profile claims if scope is present
	if contains(scopes, "profile") {
		claims.Name = user.DisplayName
		claims.PreferredUsername = user.Username
		claims.Profile = fmt.Sprintf("%s/users/%s", baseURL, user.Username)
		claims.UpdatedAt = user.UpdatedAt.Unix()
		claims.AO3Username = user.Username
		claims.AO3DisplayName = user.DisplayName
		claims.AO3JoinDate = user.CreatedAt.Unix()

		// Get user roles
		if roles, err := as.getUserRoles(user.ID); err == nil {
			claims.AO3Roles = roles
		}

		// Get user statistics
		if stats, err := as.getUserStats(user.ID); err == nil {
			claims.AO3WorkCount = stats.WorkCount
			claims.AO3BookmarkCount = stats.BookmarkCount
		}
	}

	// Add email claims if scope is present
	if contains(scopes, "email") {
		claims.Email = user.Email
		claims.EmailVerified = user.IsVerified
	}

	// Create and sign JWT
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":                claims.Issuer,
		"sub":                claims.Subject,
		"aud":                claims.Audience,
		"exp":                claims.ExpiresAt,
		"iat":                claims.IssuedAt,
		"auth_time":          claims.AuthTime,
		"nonce":              claims.Nonce,
		"name":               claims.Name,
		"preferred_username": claims.PreferredUsername,
		"profile":            claims.Profile,
		"email":              claims.Email,
		"email_verified":     claims.EmailVerified,
		"updated_at":         claims.UpdatedAt,
		"ao3_username":       claims.AO3Username,
		"ao3_display_name":   claims.AO3DisplayName,
		"ao3_roles":          claims.AO3Roles,
		"ao3_join_date":      claims.AO3JoinDate,
		"ao3_work_count":     claims.AO3WorkCount,
		"ao3_bookmark_count": claims.AO3BookmarkCount,
	})

	return token.SignedString(as.jwt.privateKey)
}

// Token revocation

func (as *AuthService) revokeRefreshToken(tokenID uuid.UUID) {
	query := `UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = NOW() WHERE id = $1`
	as.db.Exec(query, tokenID)
}

func (as *AuthService) revokeRefreshTokenByValue(token string) bool {
	query := `UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = NOW() WHERE token = $1`
	result, err := as.db.Exec(query, token)
	if err != nil {
		return false
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0
}

func (as *AuthService) revokeAccessTokenByValue(token string) bool {
	query := `UPDATE oauth_access_tokens SET is_revoked = true, revoked_at = NOW() WHERE token = $1`
	result, err := as.db.Exec(query, token)
	if err != nil {
		return false
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0
}

// Utility functions

func (as *AuthService) updateTokenLastUsed(tokenID uuid.UUID) {
	query := `UPDATE oauth_access_tokens SET last_used = NOW() WHERE id = $1`
	as.db.Exec(query, tokenID)
}

func (as *AuthService) getUserRoles(userID uuid.UUID) ([]string, error) {
	query := `SELECT role FROM user_roles WHERE user_id = $1`
	rows, err := as.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			continue
		}
		roles = append(roles, role)
	}

	return roles, nil
}

func (as *AuthService) getUserStats(userID uuid.UUID) (*UserStats, error) {
	stats := &UserStats{}

	// Get work count
	workQuery := `SELECT COUNT(*) FROM works WHERE user_id = $1 AND status = 'posted'`
	as.db.QueryRow(workQuery, userID).Scan(&stats.WorkCount)

	// Get bookmark count
	bookmarkQuery := `SELECT COUNT(*) FROM bookmarks WHERE user_id = $1`
	as.db.QueryRow(bookmarkQuery, userID).Scan(&stats.BookmarkCount)

	return stats, nil
}
