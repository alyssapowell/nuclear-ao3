package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

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

	// Get consent data from Redis
	consentJSON, err := as.redis.Get(c.Request.Context(), "consent:"+consentID).Result()
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"Error": "Consent request not found or expired",
		})
		return
	}

	// Render consent screen (this would be a proper HTML template)
	c.HTML(http.StatusOK, "consent.html", gin.H{
		"ConsentID": consentID,
		"Data":      consentJSON,
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

	consentID := c.Param("consent_id")
	consentUUID, err := uuid.Parse(consentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consent ID"})
		return
	}

	// Revoke consent
	query := `
		UPDATE user_consents 
		SET is_revoked = true, revoked_at = NOW() 
		WHERE id = $1 AND user_id = $2`

	result, err := as.db.Exec(query, consentUUID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke consent"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Consent not found"})
		return
	}

	// Also revoke all active tokens for this client
	var clientID uuid.UUID
	as.db.QueryRow("SELECT client_id FROM user_consents WHERE id = $1", consentUUID).Scan(&clientID)

	revokeQuery := `
		UPDATE oauth_access_tokens 
		SET is_revoked = true, revoked_at = NOW() 
		WHERE user_id = $1 AND client_id = $2 AND is_revoked = false`
	as.db.Exec(revokeQuery, userID, clientID)

	c.JSON(http.StatusOK, gin.H{"message": "Consent revoked successfully"})
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

	clientID := c.Param("client_id")
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	// Revoke all tokens for this client and user
	tokenQuery := `
		UPDATE oauth_access_tokens 
		SET is_revoked = true, revoked_at = NOW() 
		WHERE user_id = $1 AND client_id = $2 AND is_revoked = false`

	refreshQuery := `
		UPDATE oauth_refresh_tokens 
		SET is_revoked = true, revoked_at = NOW() 
		WHERE user_id = $1 AND client_id = $2 AND is_revoked = false`

	// Revoke consent
	consentQuery := `
		UPDATE user_consents 
		SET is_revoked = true, revoked_at = NOW() 
		WHERE user_id = $1 AND client_id = $2 AND is_revoked = false`

	_, err1 := as.db.Exec(tokenQuery, userID, clientUUID)
	_, err2 := as.db.Exec(refreshQuery, userID, clientUUID)
	_, err3 := as.db.Exec(consentQuery, userID, clientUUID)

	if err1 != nil || err2 != nil || err3 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke application access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Application access revoked successfully"})
}

// Admin OAuth2 management endpoints

func (as *AuthService) AdminListClients(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset := (page - 1) * limit

	query := `
		SELECT oc.client_id, oc.client_name, oc.description, oc.is_public, oc.is_first_party,
			oc.is_active, oc.created_at, oc.updated_at,
			COUNT(DISTINCT at.user_id) as unique_users,
			COUNT(at.id) as total_tokens
		FROM oauth_clients oc
		LEFT JOIN oauth_access_tokens at ON oc.client_id = at.client_id
		GROUP BY oc.client_id, oc.client_name, oc.description, oc.is_public, oc.is_first_party,
			oc.is_active, oc.created_at, oc.updated_at
		ORDER BY oc.created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := as.db.Query(query, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch clients"})
		return
	}
	defer rows.Close()

	var clients []gin.H
	for rows.Next() {
		var clientID uuid.UUID
		var clientName, description string
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
			"description":    description,
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
	clientID := c.Param("client_id")
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	client, err := as.getClientByID(clientUUID.String())
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

func (as *AuthService) AdminUpdateClient(c *gin.Context) {
	clientID := c.Param("client_id")
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Build dynamic update query (simplified - would need proper validation)
	query := `UPDATE oauth_clients SET updated_at = NOW()`
	args := []interface{}{}
	argIndex := 1

	if name, exists := updates["client_name"]; exists {
		query += fmt.Sprintf(", client_name = $%d", argIndex)
		args = append(args, name)
		argIndex++
	}

	if description, exists := updates["description"]; exists {
		query += fmt.Sprintf(", description = $%d", argIndex)
		args = append(args, description)
		argIndex++
	}

	if isActive, exists := updates["is_active"]; exists {
		query += fmt.Sprintf(", is_active = $%d", argIndex)
		args = append(args, isActive)
		argIndex++
	}

	if isTrusted, exists := updates["is_trusted"]; exists {
		query += fmt.Sprintf(", is_trusted = $%d", argIndex)
		args = append(args, isTrusted)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE client_id = $%d", argIndex)
	args = append(args, clientUUID)

	_, err = as.db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update client"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client updated successfully"})
}

func (as *AuthService) AdminDeleteClient(c *gin.Context) {
	clientID := c.Param("client_id")
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	// Check if client is first-party (can't delete)
	var isFirstParty bool
	err = as.db.QueryRow("SELECT is_first_party FROM oauth_clients WHERE client_id = $1", clientUUID).Scan(&isFirstParty)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	if isFirstParty {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete first-party clients"})
		return
	}

	// Soft delete (deactivate) instead of hard delete to preserve audit trail
	query := `UPDATE oauth_clients SET is_active = false, updated_at = NOW() WHERE client_id = $1`
	_, err = as.db.Exec(query, clientUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete client"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client deleted successfully"})
}

func (as *AuthService) AdminResetClientSecret(c *gin.Context) {
	clientID := c.Param("client_id")
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	// Check if client is confidential
	var isPublic bool
	err = as.db.QueryRow("SELECT is_public FROM oauth_clients WHERE client_id = $1", clientUUID).Scan(&isPublic)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	if isPublic {
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

	query := `UPDATE oauth_clients SET client_secret = $1, updated_at = NOW() WHERE client_id = $2`
	_, err = as.db.Exec(query, string(hashedSecret), clientUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update secret"})
		return
	}

	// Revoke all existing tokens for this client
	revokeQuery := `
		UPDATE oauth_access_tokens 
		SET is_revoked = true, revoked_at = NOW() 
		WHERE client_id = $1 AND is_revoked = false`
	as.db.Exec(revokeQuery, clientUUID)

	c.JSON(http.StatusOK, gin.H{
		"message":       "Client secret reset successfully",
		"client_secret": newSecret, // Return new secret only once
	})
}

func (as *AuthService) AdminListTokens(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset := (page - 1) * limit

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
		query += fmt.Sprintf(" AND at.client_id = $%d", argIndex)
		args = append(args, clientID)
		argIndex++
	}

	if userID != "" {
		query += fmt.Sprintf(" AND at.user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY at.created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := as.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tokens"})
		return
	}
	defer rows.Close()

	var tokens []gin.H
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
	tokenID := c.Param("token_id")
	tokenUUID, err := uuid.Parse(tokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	query := `
		UPDATE oauth_access_tokens 
		SET is_revoked = true, revoked_at = NOW() 
		WHERE id = $1`

	result, err := as.db.Exec(query, tokenUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token revoked successfully"})
}
