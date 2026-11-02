package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Authorization request handling

func (as *AuthService) storeAuthorizationRequest(req models.AuthorizeRequest) string {
	requestID := uuid.New().String()

	// Kept for 10 minutes
	reqJSON, _ := json.Marshal(req)
	if err := as.putPending(context.Background(), "auth_req:"+requestID, reqJSON); err != nil {
		log.Printf("oauth: store authorization request: %v", err)
	}

	return requestID
}

func (as *AuthService) getAuthorizationRequest(requestID string) (*models.AuthorizeRequest, error) {
	reqJSON, err := as.getPending(context.Background(), "auth_req:"+requestID)
	if err != nil {
		return nil, err
	}

	var req models.AuthorizeRequest
	if err := json.Unmarshal(reqJSON, &req); err != nil {
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

	// Check Authorization header for JWT token
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		tokenString := extractBearerToken(authHeader)
		if tokenString != "" {
			if userID, _, err := as.validateSession(c, tokenString); err == nil {
				c.Set("user_id", userID)
				return &userID
			}
		}
	}

	return nil
}

// Consent management

// hasValidConsent says whether the user has a live approval for this app
// that covers every one of the scopes.
func (as *AuthService) hasValidConsent(userID, clientID uuid.UUID, scopes []string) bool {
	query := `
		SELECT scopes FROM user_consents 
		WHERE user_id = $1 AND client_id = $2 AND is_revoked = false 
		AND (expires_at IS NULL OR expires_at > NOW())`

	var consentedScopes []string
	err := as.db.QueryRow(query, userID, clientID).Scan(pq.Array(&consentedScopes))
	if err != nil {
		return false
	}

	// Admin-only scopes never count, whatever an old row says.
	for _, s := range scopes {
		if !models.IsGrantableScope(s) {
			return false
		}
	}
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

	// Store the consent request, tied to the user who is looking at it, so a
	// consent ID can't be approved by anyone else.
	viewer := as.getAuthenticatedUser(c)
	if viewer == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	consentID := uuid.New().String()
	consentData := map[string]interface{}{
		"user_id":            viewer.String(),
		"client":             client,
		"scopes":             scopes,
		"scope_descriptions": scopeDescriptions,
		"authorize_request":  req,
	}

	consentJSON, _ := json.Marshal(consentData)
	if err := as.putPending(context.Background(), "consent:"+consentID, consentJSON); err != nil {
		log.Printf("oauth: store consent request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not start consent"})
		return
	}

	// The website shows the consent screen from this answer
	c.JSON(http.StatusOK, gin.H{
		"consent_required":   true,
		"consent_id":         consentID,
		"client_name":        client.Name,
		"scopes":             scopes,
		"scope_descriptions": scopeDescriptions,
		"consent_url":        fmt.Sprintf("/auth/consent/%s", consentID),
		// Built with the URL parser, so a hostile state value can't add
		// parameters or change where it points.
		"cancel_url": errorRedirectURL(req.RedirectURI, req.State, "access_denied", ""),
	})
}

func (as *AuthService) processConsent(c *gin.Context, consentID string, approved bool) {
	// Get consent data
	consentJSON, err := as.getPending(context.Background(), "consent:"+consentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consent request"})
		return
	}

	var consentData struct {
		UserID           string                  `json:"user_id"`
		Client           *models.OAuthClient     `json:"client"`
		Scopes           []string                `json:"scopes"`
		AuthorizeRequest models.AuthorizeRequest `json:"authorize_request"`
	}
	if err := json.Unmarshal(consentJSON, &consentData); err != nil || consentData.Client == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consent data"})
		return
	}

	userID := as.getAuthenticatedUser(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	if consentData.UserID != userID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "This consent request belongs to another user"})
		return
	}
	// Each consent request can be answered once. If a second answer got here
	// first, there is nothing left to take.
	if _, err := as.takePending(context.Background(), "consent:"+consentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consent request"})
		return
	}

	// The app may have been switched off or changed since the screen was
	// shown, so check the request again against the app as it is now.
	req := consentData.AuthorizeRequest
	client, err := as.getClientByID(consentData.Client.ID.String())
	if err != nil || !contains(client.RedirectURIs, req.RedirectURI) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client", "error_description": "This app is no longer available"})
		return
	}
	scopes, problem := as.checkAuthorizeRequest(client, req)
	if problem != nil {
		as.redirectWithError(c, req.RedirectURI, req.State, problem.code, problem.desc)
		return
	}

	if !approved {
		// User denied consent
		as.redirectWithError(c, req.RedirectURI, req.State, "access_denied", "User denied access")
		return
	}

	// Store consent
	if err := as.storeUserConsent(*userID, client.ID, scopes); err != nil {
		log.Printf("oauth: store consent: %v", err)
		as.redirectWithError(c, req.RedirectURI, req.State, "server_error", "Failed to record consent")
		return
	}

	// Continue with authorization
	code, err := as.generateAuthorizationCode(*userID, client.ID, req)
	if err != nil {
		as.redirectWithError(c, req.RedirectURI, req.State, "server_error", "Failed to generate code")
		return
	}

	c.Redirect(http.StatusFound, as.buildCallbackURL(req.RedirectURI, code, req.State))
}

// storeUserConsent records that the user allows an app these scopes. A new
// approval adds to what was approved before; after a revocation it starts
// again from only what was just approved.
func (as *AuthService) storeUserConsent(userID, clientID uuid.UUID, scopes []string) error {
	query := `
		INSERT INTO user_consents (id, user_id, client_id, scopes, granted_at, is_revoked)
		VALUES ($1, $2, $3, $4, now(), false)
		ON CONFLICT (user_id, client_id) 
		DO UPDATE SET
			scopes = CASE WHEN user_consents.is_revoked THEN EXCLUDED.scopes
				ELSE ARRAY(SELECT DISTINCT s FROM unnest(user_consents.scopes || EXCLUDED.scopes) s) END,
			granted_at = now(), is_revoked = false, revoked_at = NULL`

	_, err := as.db.Exec(query, uuid.New(), userID, clientID, pq.Array(scopes))
	return err
}

// Authorization code management

// generateAuthorizationCode stores a code that is good for 60 seconds and
// tied to the redirect address it was asked for, and returns the raw code.
// Only the code's hash is kept.
func (as *AuthService) generateAuthorizationCode(userID, clientID uuid.UUID, req models.AuthorizeRequest) (string, error) {
	code, err := randomToken()
	if err != nil {
		return "", err
	}
	_, err = as.db.Exec(`
		INSERT INTO authorization_codes (
			code_hash, client_id, user_id, redirect_uri, scopes, state, nonce,
			code_challenge, code_challenge_method, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now() + $10 * interval '1 second')`,
		hashSecret(code), clientID, userID, req.RedirectURI, pq.Array(parseScopes(req.Scope)),
		req.State, req.Nonce, req.CodeChallenge, req.CodeChallengeMethod, int(authCodeTTL.Seconds()))
	if err != nil {
		return "", err
	}
	return code, nil
}

// OIDC ID Token generation

func (as *AuthService) generateIDToken(userID, clientID uuid.UUID, nonce string, scopes []string) (string, error) {
	issuer, err := oauthIssuer()
	if err != nil {
		return "", err
	}
	user, err := as.getUserByID(userID)
	if err != nil {
		return "", err
	}

	now := time.Now()

	// Set auth_time to last login or current time if never logged in
	authTime := now.Unix()
	if user.LastLoginAt != nil {
		authTime = user.LastLoginAt.Unix()
	}

	claims := models.OIDCClaims{
		Issuer:    issuer,
		Subject:   userID.String(), // TODO(phase 2): replace with a pairwise subject
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
		claims.Profile = fmt.Sprintf("%s/users/%s", issuer, url.PathEscape(user.Username))
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
	// Name the signing key, so apps can pick the right key from the JWKS.
	token.Header["kid"] = as.jwt.keyID

	return token.SignedString(as.jwt.privateKey)
}

// Revoking access

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// revokeUserClientAccess ends one person's access for one app: every access
// and refresh token, every code not yet exchanged, and the approval itself,
// so the app has to ask again. Removing an app from "my apps" and revoking a
// consent both use it.
func revokeUserClientAccess(ctx context.Context, tx execer, userID, clientID uuid.UUID) error {
	for _, q := range []string{
		`UPDATE oauth_access_tokens  SET is_revoked = true, revoked_at = now() WHERE user_id = $1 AND client_id = $2 AND NOT is_revoked`,
		`UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = now() WHERE user_id = $1 AND client_id = $2 AND NOT is_revoked`,
		`UPDATE authorization_codes  SET used_at = now() WHERE user_id = $1 AND client_id = $2 AND used_at IS NULL`,
		`UPDATE user_consents        SET is_revoked = true, revoked_at = now() WHERE user_id = $1 AND client_id = $2 AND NOT is_revoked`,
	} {
		if _, err := tx.ExecContext(ctx, q, userID, clientID); err != nil {
			return err
		}
	}
	return nil
}

// revokeClientAccess ends access for everyone who uses an app. Consents are
// forgotten too when forgetConsents is true (the app is switched off or
// removed); a secret reset leaves them, since the people did not change
// their minds.
func revokeClientAccess(ctx context.Context, tx execer, clientID uuid.UUID, forgetConsents bool) error {
	qs := []string{
		`UPDATE oauth_access_tokens  SET is_revoked = true, revoked_at = now() WHERE client_id = $1 AND NOT is_revoked`,
		`UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = now() WHERE client_id = $1 AND NOT is_revoked`,
		`UPDATE authorization_codes  SET used_at = now() WHERE client_id = $1 AND used_at IS NULL`,
	}
	if forgetConsents {
		qs = append(qs, `UPDATE user_consents SET is_revoked = true, revoked_at = now() WHERE client_id = $1 AND NOT is_revoked`)
	}
	for _, q := range qs {
		if _, err := tx.ExecContext(ctx, q, clientID); err != nil {
			return err
		}
	}
	return nil
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
	workQuery := `SELECT COUNT(*) FROM works WHERE user_id = $1 AND status = 'published'`
	as.db.QueryRow(workQuery, userID).Scan(&stats.WorkCount)

	// Get bookmark count
	bookmarkQuery := `SELECT COUNT(*) FROM bookmarks WHERE user_id = $1`
	as.db.QueryRow(bookmarkQuery, userID).Scan(&stats.BookmarkCount)

	return stats, nil
}
