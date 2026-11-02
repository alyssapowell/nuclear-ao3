package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nuclear-ao3/internal/audit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// JWKS endpoint for token verification
func (as *AuthService) GetJWKS(c *gin.Context) {
	// Return public keys in JWKS format for token verification
	jwks := as.jwt.GetJWKS()

	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, jwks)
}

// Consent handling
func (as *AuthService) ShowConsent(c *gin.Context) {
	consentID := c.Param("consent_id")

	// Get consent data
	consentJSON, err := as.getPending(c.Request.Context(), "consent:"+consentID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"Error": "Consent request not found or expired",
		})
		return
	}

	// Render consent screen (this would be a proper HTML template)
	c.HTML(http.StatusOK, "consent.html", gin.H{
		"ConsentID": consentID,
		"Data":      string(consentJSON),
	})
}

func (as *AuthService) ProcessConsent(c *gin.Context) {
	consentID := c.Param("consent_id")
	approved := c.PostForm("approved") == "true"

	as.processConsent(c, consentID, approved)
}

// User consent management
func (as *AuthService) GetUserConsents(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	query := `
		SELECT uc.id, uc.client_id, oc.client_name, oc.description, oc.website, oc.logo_url,
			uc.scopes, uc.granted_at, uc.expires_at
		FROM user_consents uc
		JOIN oauth_clients oc ON uc.client_id = oc.client_id
		WHERE uc.user_id = $1 AND uc.is_revoked = false
		ORDER BY uc.granted_at DESC`

	rows, err := as.db.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch consents"})
		return
	}
	defer rows.Close()

	var consents []gin.H
	for rows.Next() {
		var consent gin.H
		var id, clientID uuid.UUID
		var clientName, description, website, logoURL string
		var scopes []string
		var grantedAt time.Time
		var expiresAt *time.Time

		err := rows.Scan(&id, &clientID, &clientName, &description, &website, &logoURL,
			pq.Array(&scopes), &grantedAt, &expiresAt)
		if err != nil {
			continue
		}

		consent = gin.H{
			"id":          id,
			"client_id":   clientID,
			"client_name": clientName,
			"description": description,
			"website":     website,
			"logo_url":    logoURL,
			"scopes":      scopes,
			"granted_at":  grantedAt,
			"expires_at":  expiresAt,
		}

		consents = append(consents, consent)
	}

	c.JSON(http.StatusOK, gin.H{"consents": consents})
}

func (as *AuthService) RevokeConsent(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	uid, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	consentID := c.Param("consent_id")
	consentUUID, err := uuid.Parse(consentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consent ID"})
		return
	}

	// Find the app this approval is for; it has to be the caller's own.
	var clientID uuid.UUID
	err = as.db.QueryRowContext(c.Request.Context(),
		`SELECT client_id FROM user_consents WHERE id = $1 AND user_id = $2`, consentUUID, uid).Scan(&clientID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Consent not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke consent"})
		return
	}

	// Taking back an approval ends everything it allowed: access tokens,
	// refresh tokens and codes not yet exchanged, the same as removing the
	// app from "my apps".
	if err := as.revokeApplicationAccess(c.Request.Context(), uid, clientID); err != nil {
		log.Printf("revoke consent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke consent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Consent revoked successfully"})
}

// revokeApplicationAccess ends one person's access for one app, all or
// nothing.
func (as *AuthService) revokeApplicationAccess(ctx context.Context, userID, clientID uuid.UUID) error {
	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := revokeUserClientAccess(ctx, tx, userID, clientID); err != nil {
		return err
	}
	return tx.Commit()
}

func (as *AuthService) GetAuthorizedApplications(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	query := `
		SELECT DISTINCT oc.client_id, oc.client_name, oc.description, oc.website, oc.logo_url,
			COUNT(at.id) as active_tokens, MAX(at.last_used) as last_used
		FROM oauth_clients oc
		JOIN oauth_access_tokens at ON oc.client_id = at.client_id
		WHERE at.user_id = $1 AND at.is_revoked = false AND at.expires_at > NOW()
		GROUP BY oc.client_id, oc.client_name, oc.description, oc.website, oc.logo_url
		ORDER BY last_used DESC NULLS LAST`

	rows, err := as.db.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch applications"})
		return
	}
	defer rows.Close()

	var applications []gin.H
	for rows.Next() {
		var clientID uuid.UUID
		var clientName, description, website, logoURL string
		var activeTokens int
		var lastUsed *time.Time

		err := rows.Scan(&clientID, &clientName, &description, &website, &logoURL,
			&activeTokens, &lastUsed)
		if err != nil {
			continue
		}

		applications = append(applications, gin.H{
			"client_id":     clientID,
			"client_name":   clientName,
			"description":   description,
			"website":       website,
			"logo_url":      logoURL,
			"active_tokens": activeTokens,
			"last_used":     lastUsed,
		})
	}

	c.JSON(http.StatusOK, gin.H{"applications": applications})
}

func (as *AuthService) RevokeApplication(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	uid, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	clientUUID, err := uuid.Parse(c.Param("client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	// Revoke all tokens and codes for this client and user, and the consent
	if err := as.revokeApplicationAccess(c.Request.Context(), uid, clientUUID); err != nil {
		log.Printf("revoke application: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke application access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Application access revoked successfully"})
}

// Admin OAuth2 management endpoints

// pageParams reads ?page and ?limit and keeps them in range: page 1 or more
// (and not absurdly large), limit 1 to 100, 50 if missing or wrong.
func pageParams(c *gin.Context) (page, limit, offset int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	limit, err = strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit, (page - 1) * limit
}

func (as *AuthService) AdminListClients(c *gin.Context) {
	page, limit, offset := pageParams(c)

	// Every app is listed, switched off or not.
	query := `
		SELECT oc.client_id, oc.client_name, oc.description, oc.is_public, oc.is_first_party,
			oc.is_active, oc.created_at, oc.updated_at,
			COUNT(DISTINCT at.user_id) as unique_users,
			COUNT(at.id) as total_tokens
		FROM oauth_clients oc
		LEFT JOIN oauth_access_tokens at ON oc.client_id = at.client_id
		GROUP BY oc.client_id, oc.client_name, oc.description, oc.is_public, oc.is_first_party,
			oc.is_active, oc.created_at, oc.updated_at
		ORDER BY oc.created_at DESC, oc.client_id
		LIMIT $1 OFFSET $2`

	rows, err := as.db.Query(query, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch clients"})
		return
	}
	defer rows.Close()

	clients := []gin.H{}
	for rows.Next() {
		var clientID uuid.UUID
		var clientName string
		var description sql.NullString
		var isPublic, isFirstParty, isActive bool
		var createdAt, updatedAt time.Time
		var uniqueUsers, totalTokens int

		err := rows.Scan(&clientID, &clientName, &description, &isPublic, &isFirstParty,
			&isActive, &createdAt, &updatedAt, &uniqueUsers, &totalTokens)
		if err != nil {
			continue
		}

		clients = append(clients, gin.H{
			"client_id":      clientID,
			"client_name":    clientName,
			"description":    description.String,
			"is_public":      isPublic,
			"is_first_party": isFirstParty,
			"is_active":      isActive,
			"created_at":     createdAt,
			"updated_at":     updatedAt,
			"unique_users":   uniqueUsers,
			"total_tokens":   totalTokens,
		})
	}

	// Get total count
	var total int
	as.db.QueryRow("SELECT COUNT(*) FROM oauth_clients").Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"clients": clients,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + limit - 1) / limit,
		},
	})
}

func (as *AuthService) AdminGetClient(c *gin.Context) {
	clientUUID, err := uuid.Parse(c.Param("client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	// Admins can look at switched-off apps too.
	client, err := as.getClientForAdmin(clientUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	// Don't return the secret in admin view
	clientData := gin.H{
		"client_id":         client.ID,
		"client_name":       client.Name,
		"description":       client.Description,
		"website":           client.Website,
		"logo_url":          client.LogoURL,
		"redirect_uris":     client.RedirectURIs,
		"scopes":            client.Scopes,
		"grant_types":       client.GrantTypes,
		"response_types":    client.ResponseTypes,
		"is_public":         client.IsPublic,
		"is_confidential":   client.IsConfidential,
		"is_trusted":        client.IsTrusted,
		"is_first_party":    client.IsFirstParty,
		"owner_id":          client.OwnerID,
		"access_token_ttl":  client.AccessTokenTTL,
		"refresh_token_ttl": client.RefreshTokenTTL,
		"is_active":         client.IsActive,
		"created_at":        client.CreatedAt,
		"updated_at":        client.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{"client": clientData})
}

// clientUpdate is what an admin may change on an app. A field left out is
// left alone. Fields not listed here (the secret, is_public, is_first_party)
// cannot be changed, and a request that names one is refused.
type clientUpdate struct {
	Name            *string   `json:"client_name"`
	Description     *string   `json:"description"`
	Website         *string   `json:"website"`
	LogoURL         *string   `json:"logo_url"`
	RedirectURIs    *[]string `json:"redirect_uris"`
	Scopes          *[]string `json:"scopes"`
	GrantTypes      *[]string `json:"grant_types"`
	ResponseTypes   *[]string `json:"response_types"`
	AccessTokenTTL  *int      `json:"access_token_ttl"`
	RefreshTokenTTL *int      `json:"refresh_token_ttl"`
	IsActive        *bool     `json:"is_active"`
	IsTrusted       *bool     `json:"is_trusted"`
}

func (as *AuthService) AdminUpdateClient(c *gin.Context) {
	clientUUID, err := uuid.Parse(c.Param("client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}
	actor, ok := as.adminActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}

	var up clientUpdate
	dec := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&up); err != nil {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_request", "Invalid request data"))
		return
	}

	client, err := as.getClientForAdmin(clientUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	bad := func(code, desc string) { c.JSON(http.StatusBadRequest, problemJSON(code, desc)) }
	var sets []string
	var args []interface{}
	changed := map[string]any{}
	set := func(column string, value interface{}, field string, shown any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
		changed[field] = shown
	}

	if up.Name != nil {
		name := strings.TrimSpace(*up.Name)
		if n := len([]rune(name)); n == 0 || n > maxClientNameLen {
			bad("invalid_client_metadata", "client_name must be 1 to 100 characters")
			return
		}
		set("client_name", name, "client_name", name)
	}
	if up.Description != nil {
		if len([]rune(*up.Description)) > maxDescriptionLen {
			bad("invalid_client_metadata", "description is too long")
			return
		}
		set("description", *up.Description, "description", "changed")
	}
	if up.Website != nil {
		if !validOptionalURL(*up.Website) {
			bad("invalid_client_metadata", "website must be an https address")
			return
		}
		set("website", *up.Website, "website", *up.Website)
	}
	if up.LogoURL != nil {
		if !validOptionalURL(*up.LogoURL) {
			bad("invalid_client_metadata", "logo_url must be an https address")
			return
		}
		set("logo_url", *up.LogoURL, "logo_url", *up.LogoURL)
	}
	if up.RedirectURIs != nil {
		uris, problem := cleanRedirectURIs(*up.RedirectURIs, client.IsPublic)
		if problem != "" {
			bad("invalid_redirect_uri", problem)
			return
		}
		set("redirect_uris", pq.Array(uris), "redirect_uris", uris)
	}
	scopesNow := client.Scopes
	scopesChanged := false
	if up.Scopes != nil {
		scopes, problem := cleanScopes(*up.Scopes)
		if problem != "" {
			bad("invalid_scope", problem)
			return
		}
		scopesNow, scopesChanged = scopes, true
		set("scopes", pq.Array(scopes), "scopes", scopes)
	}
	if up.GrantTypes != nil {
		grants, ok := cleanChoices(*up.GrantTypes, allowedGrantTypes)
		if !ok || !contains(grants, "authorization_code") {
			bad("invalid_client_metadata", "grant_types must be authorization_code, optionally with refresh_token")
			return
		}
		set("grant_types", pq.Array(grants), "grant_types", grants)
	}
	if up.ResponseTypes != nil {
		responses, ok := cleanChoices(*up.ResponseTypes, allowedResponseTypes)
		if !ok {
			bad("invalid_client_metadata", "response_types must be code")
			return
		}
		set("response_types", pq.Array(responses), "response_types", responses)
	}
	if up.AccessTokenTTL != nil {
		v := clampAccessTTL(*up.AccessTokenTTL)
		set("access_token_ttl", v, "access_token_ttl", v)
	}
	if up.RefreshTokenTTL != nil {
		v := clampRefreshTTL(*up.RefreshTokenTTL)
		set("refresh_token_ttl", v, "refresh_token_ttl", v)
	}
	if up.IsActive != nil {
		set("is_active", *up.IsActive, "is_active", *up.IsActive)
	}
	if up.IsTrusted != nil {
		set("is_trusted", *up.IsTrusted, "is_trusted", *up.IsTrusted)
	}
	if len(sets) == 0 {
		bad("invalid_request", "Nothing to change")
		return
	}

	ctx := c.Request.Context()
	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update client"})
		return
	}
	defer tx.Rollback()

	args = append(args, clientUUID)
	_, err = tx.ExecContext(ctx, "UPDATE oauth_clients SET "+strings.Join(sets, ", ")+
		fmt.Sprintf(", updated_at = now() WHERE client_id = $%d", len(args)), args...)

	// Switching an app off ends everything it was given.
	if err == nil && up.IsActive != nil && !*up.IsActive {
		err = revokeClientAccess(ctx, tx, clientUUID, true)
	}
	// Taking a scope away from an app takes it from tokens already issued.
	if err == nil && scopesChanged {
		for _, q := range []string{
			`UPDATE oauth_access_tokens  SET is_revoked = true, revoked_at = now() WHERE client_id = $1 AND NOT is_revoked AND NOT (scopes <@ $2::text[])`,
			`UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = now() WHERE client_id = $1 AND NOT is_revoked AND NOT (scopes <@ $2::text[])`,
		} {
			if _, err = tx.ExecContext(ctx, q, clientUUID, pq.Array(scopesNow)); err != nil {
				break
			}
		}
	}
	if err == nil {
		err = audit.Record(ctx, tx, actor, "oauth_client.update", "oauth_client", clientUUID.String(), "",
			map[string]any{"client_name": client.Name, "changed": changed})
	}
	// A change of trust gets a row of its own, so it is easy to find.
	if err == nil && up.IsTrusted != nil && *up.IsTrusted != client.IsTrusted {
		action := "oauth_client.untrust"
		if *up.IsTrusted {
			action = "oauth_client.trust"
		}
		err = audit.Record(ctx, tx, actor, action, "oauth_client", clientUUID.String(), "",
			map[string]any{"client_name": client.Name})
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("update client: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update client"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client updated successfully"})
}

func (as *AuthService) AdminDeleteClient(c *gin.Context) {
	clientUUID, err := uuid.Parse(c.Param("client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}
	actor, ok := as.adminActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}

	client, err := as.getClientForAdmin(clientUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}
	if client.IsFirstParty {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete first-party clients"})
		return
	}

	ctx := c.Request.Context()
	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete client"})
		return
	}
	defer tx.Rollback()

	// Soft delete (deactivate) instead of hard delete to preserve audit trail,
	// and take back everything the app was given.
	_, err = tx.ExecContext(ctx, `UPDATE oauth_clients SET is_active = false, updated_at = NOW() WHERE client_id = $1`, clientUUID)
	if err == nil {
		err = revokeClientAccess(ctx, tx, clientUUID, true)
	}
	if err == nil {
		err = audit.Record(ctx, tx, actor, "oauth_client.delete", "oauth_client", clientUUID.String(), "",
			map[string]any{"client_name": client.Name})
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("delete client: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete client"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client deleted successfully"})
}

func (as *AuthService) AdminResetClientSecret(c *gin.Context) {
	clientUUID, err := uuid.Parse(c.Param("client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}
	actor, ok := as.adminActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}

	client, err := as.getClientForAdmin(clientUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}
	if client.IsPublic {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Public clients don't have secrets"})
		return
	}

	// Generate new secret
	newSecret, err := generateClientSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate secret"})
		return
	}

	// Hash and store new secret
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(newSecret), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash secret"})
		return
	}

	ctx := c.Request.Context()
	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update secret"})
		return
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `UPDATE oauth_clients SET client_secret = $1, updated_at = NOW() WHERE client_id = $2`,
		string(hashedSecret), clientUUID)
	// Revoke all existing tokens for this client: access and refresh, and any
	// code not yet exchanged. People's approvals stay.
	if err == nil {
		err = revokeClientAccess(ctx, tx, clientUUID, false)
	}
	if err == nil {
		err = audit.Record(ctx, tx, actor, "oauth_client.reset_secret", "oauth_client", clientUUID.String(), "",
			map[string]any{"client_name": client.Name})
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("reset client secret: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update secret"})
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"message":       "Client secret reset successfully",
		"client_secret": newSecret, // Return new secret only once
	})
}

func (as *AuthService) AdminListTokens(c *gin.Context) {
	_, limit, offset := pageParams(c)

	clientID := c.Query("client_id")
	userID := c.Query("user_id")

	query := `
		SELECT at.id, at.user_id, u.username, at.client_id, oc.client_name,
			at.scopes, at.expires_at, at.is_revoked, at.last_used, at.created_at
		FROM oauth_access_tokens at
		JOIN users u ON at.user_id = u.id
		JOIN oauth_clients oc ON at.client_id = oc.client_id
		WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	if clientID != "" {
		id, err := uuid.Parse(clientID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
			return
		}
		query += fmt.Sprintf(" AND at.client_id = $%d", argIndex)
		args = append(args, id)
		argIndex++
	}

	if userID != "" {
		id, err := uuid.Parse(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		query += fmt.Sprintf(" AND at.user_id = $%d", argIndex)
		args = append(args, id)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY at.created_at DESC, at.id LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := as.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tokens"})
		return
	}
	defer rows.Close()

	tokens := []gin.H{}
	for rows.Next() {
		var tokenID, tokenUserID, tokenClientID uuid.UUID
		var username, clientName string
		var scopes []string
		var expiresAt time.Time
		var isRevoked bool
		var lastUsed *time.Time
		var createdAt time.Time

		err := rows.Scan(&tokenID, &tokenUserID, &username, &tokenClientID, &clientName,
			pq.Array(&scopes), &expiresAt, &isRevoked, &lastUsed, &createdAt)
		if err != nil {
			continue
		}

		tokens = append(tokens, gin.H{
			"id":          tokenID,
			"user_id":     tokenUserID,
			"username":    username,
			"client_id":   tokenClientID,
			"client_name": clientName,
			"scopes":      scopes,
			"expires_at":  expiresAt,
			"is_revoked":  isRevoked,
			"last_used":   lastUsed,
			"created_at":  createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

func (as *AuthService) AdminRevokeToken(c *gin.Context) {
	tokenUUID, err := uuid.Parse(c.Param("token_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}
	actor, ok := as.adminActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}

	ctx := c.Request.Context()
	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
		return
	}
	defer tx.Rollback()

	var clientID, ownerID uuid.UUID
	err = tx.QueryRowContext(ctx, `
		UPDATE oauth_access_tokens SET is_revoked = true, revoked_at = NOW()
		WHERE id = $1 RETURNING client_id, user_id`, tokenUUID).Scan(&clientID, &ownerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}
	// The refresh token that could mint a new one goes too.
	if err == nil {
		_, err = tx.ExecContext(ctx, `
			UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = NOW()
			WHERE access_token_id = $1 AND NOT is_revoked`, tokenUUID)
	}
	if err == nil {
		err = audit.Record(ctx, tx, actor, "oauth_token.revoke", "oauth_token", tokenUUID.String(), "",
			map[string]any{"client_id": clientID.String(), "user_id": ownerID.String()})
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("admin revoke token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token revoked successfully"})
}
