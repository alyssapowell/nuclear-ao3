package auth

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"nuclear-ao3/internal/audit"
	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// OAuth2/OIDC Discovery endpoints

// discoveryUnavailable answers when the archive's public address (SITE_URL)
// is missing or still a placeholder. The provider must not name itself with
// a made-up address.
func discoveryUnavailable(c *gin.Context, err error) {
	log.Printf("oauth: discovery refused: %v", err)
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error":             "server_error",
		"error_description": "Sign-in with this archive is not set up yet",
	})
}

// userinfoClaims are the claims an app can receive, from the ID token or
// userinfo. "pseud" and "name" come with the profile scope (the one pseud the
// person chose), "email" and "email_verified" with the email scope. Nothing
// here names the account, and no claim is the same in two apps.
var userinfoClaims = []string{
	"sub", "iss", "aud", "azp", "exp", "iat", "auth_time", "nonce", "at_hash",
	"name", "pseud",
	"email", "email_verified",
}

func (as *AuthService) WellKnownOIDC(c *gin.Context) {
	baseURL, err := oauthIssuer()
	if err != nil {
		discoveryUnavailable(c, err)
		return
	}

	config := models.OIDCDiscoveryDocument{
		Issuer:                baseURL,
		AuthorizationEndpoint: baseURL + "/auth/authorize",
		TokenEndpoint:         baseURL + "/auth/token",
		UserinfoEndpoint:      baseURL + "/auth/userinfo",
		JWKSUri:               baseURL + "/auth/jwks",
		RevocationEndpoint:    baseURL + "/auth/revoke",
		IntrospectionEndpoint: baseURL + "/auth/introspect",

		ScopesSupported:                   models.GrantableScopes(),
		ResponseTypesSupported:            []string{"code"},
		ResponseModesSupported:            []string{"query"},
		GrantTypesSupported:               append([]string(nil), allowedGrantTypes...),
		SubjectTypesSupported:             []string{"pairwise"},
		IDTokenSigningAlgValuesSupported:  []string{"RS256"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post", "none"},
		// Introspection is for confidential apps only. A public app may
		// revoke its own tokens with just its client_id (RFC 7009).
		RevocationEndpointAuthMethodsSupported:    []string{"client_secret_basic", "client_secret_post", "none"},
		IntrospectionEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post"},
		CodeChallengeMethodsSupported:             []string{"S256"},
		ClaimsSupported:                           userinfoClaims,
		OpPolicyURI:                               baseURL + "/privacy",
		OpTosURI:                                  baseURL + "/terms",
	}

	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, config)
}

func (as *AuthService) WellKnownOAuth2(c *gin.Context) {
	baseURL, err := oauthIssuer()
	if err != nil {
		discoveryUnavailable(c, err)
		return
	}

	config := map[string]interface{}{
		"issuer":                                        baseURL,
		"authorization_endpoint":                        baseURL + "/auth/authorize",
		"token_endpoint":                                baseURL + "/auth/token",
		"jwks_uri":                                      baseURL + "/auth/jwks",
		"revocation_endpoint":                           baseURL + "/auth/revoke",
		"introspection_endpoint":                        baseURL + "/auth/introspect",
		"scopes_supported":                              models.GrantableScopes(),
		"response_types_supported":                      []string{"code"},
		"response_modes_supported":                      []string{"query"},
		"grant_types_supported":                         allowedGrantTypes,
		"token_endpoint_auth_methods_supported":         []string{"client_secret_basic", "client_secret_post", "none"},
		"revocation_endpoint_auth_methods_supported":    []string{"client_secret_basic", "client_secret_post", "none"},
		"introspection_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"code_challenge_methods_supported":              []string{"S256"},
	}

	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, config)
}

// Client Registration

// problemJSON is the body for a refused registration or admin change.
func problemJSON(code, desc string) gin.H {
	return gin.H{"error": code, "error_description": desc}
}

// adminActor says who is making an admin request, for the audit log.
func (as *AuthService) adminActor(c *gin.Context) (audit.Actor, bool) {
	v, ok := c.Get("user_uuid")
	id, isID := v.(uuid.UUID)
	if !ok || !isID {
		return audit.Actor{}, false
	}
	var name string
	if err := as.db.QueryRowContext(c.Request.Context(), `SELECT email FROM users WHERE id = $1`, id).Scan(&name); err != nil {
		return audit.Actor{}, false
	}
	return audit.UserActor(id, name), true
}

func (as *AuthService) RegisterClient(c *gin.Context) {
	var req models.ClientRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_request", "Invalid client registration request"))
		return
	}
	actor, ok := as.adminActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}

	if n := len([]rune(strings.TrimSpace(req.Name))); n == 0 || n > maxClientNameLen {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_client_metadata", "client_name must be 1 to 100 characters"))
		return
	}
	if len([]rune(req.Description)) > maxDescriptionLen {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_client_metadata", "description is too long"))
		return
	}
	if !validOptionalURL(req.Website) || !validOptionalURL(req.LogoURL) {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_client_metadata", "website and logo_url must be https addresses"))
		return
	}

	redirectURIs, problem := cleanRedirectURIs(req.RedirectURIs, req.IsPublic)
	if problem != "" {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_redirect_uri", problem))
		return
	}
	scopes, problem := cleanScopes(req.Scopes)
	if problem != "" {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_scope", problem))
		return
	}

	// Defaults, then the allowlists. client_credentials is not on the list.
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code"}
	}
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}
	grantTypes, ok := cleanChoices(req.GrantTypes, allowedGrantTypes)
	if !ok || !contains(grantTypes, "authorization_code") {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_client_metadata",
			"grant_types must be authorization_code, optionally with refresh_token"))
		return
	}
	responseTypes, ok := cleanChoices(req.ResponseTypes, allowedResponseTypes)
	if !ok {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_client_metadata", "response_types must be code"))
		return
	}

	// Generate client credentials
	clientID := uuid.New()
	var clientSecret, secretHash string
	if !req.IsPublic {
		secret, err := generateClientSecret()
		if err != nil {
			c.JSON(http.StatusInternalServerError, problemJSON("server_error", "Failed to generate client secret"))
			return
		}
		clientSecret = secret
		hashed, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
			return
		}
		secretHash = string(hashed)
	}

	accessTTL := clampAccessTTL(req.AccessTokenTTL)
	refreshTTL := clampRefreshTTL(req.RefreshTokenTTL)

	tx, err := as.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		log.Printf("register client: %v", err)
		c.JSON(http.StatusInternalServerError, problemJSON("server_error", "Failed to register client"))
		return
	}
	defer tx.Rollback()

	// The route only lets admins through; the admin owns the client. The
	// admin is the reviewer, so the app is approved at once. It is not
	// trusted or first-party.
	var createdAt time.Time
	err = tx.QueryRowContext(c.Request.Context(), `
		INSERT INTO oauth_clients (
			client_id, client_secret, client_name, description, website, logo_url,
			redirect_uris, scopes, grant_types, response_types, is_public, is_confidential,
			is_trusted, is_first_party, owner_id, access_token_ttl, refresh_token_ttl, status, reviewed_by, reviewed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, false, false, $13, $14, $15, 'approved', $13, now())
		RETURNING created_at`,
		clientID, secretHash, strings.TrimSpace(req.Name), req.Description, req.Website, req.LogoURL,
		pq.Array(redirectURIs), pq.Array(scopes), pq.Array(grantTypes), pq.Array(responseTypes),
		req.IsPublic, !req.IsPublic, actor.UserID, accessTTL, refreshTTL).Scan(&createdAt)
	if err == nil {
		err = audit.Record(c.Request.Context(), tx, actor, "oauth_client.create", "oauth_client", clientID.String(), "",
			map[string]any{"client_name": strings.TrimSpace(req.Name), "is_public": req.IsPublic,
				"scopes": scopes, "grant_types": grantTypes, "redirect_uris": redirectURIs})
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("register client: %v", err)
		c.JSON(http.StatusInternalServerError, problemJSON("server_error", "Failed to register client"))
		return
	}

	response := models.ClientRegistrationResponse{
		ClientID:        clientID.String(),
		Name:            strings.TrimSpace(req.Name),
		Description:     req.Description,
		Website:         req.Website,
		LogoURL:         req.LogoURL,
		RedirectURIs:    redirectURIs,
		Scopes:          scopes,
		GrantTypes:      grantTypes,
		ResponseTypes:   responseTypes,
		IsPublic:        req.IsPublic,
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
	}
	response.CreatedAt = createdAt
	// Include client secret only for confidential clients
	if !req.IsPublic {
		response.ClientSecret = clientSecret
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, response)
}

// Authorization endpoint

// authzProblem is a refused authorization request that is reported to the
// app by redirect.
type authzProblem struct{ code, desc string }

// checkAuthorizeRequest applies the rules that depend on the app: response
// type, grant, scopes and PKCE. It returns the scopes to grant.
func (as *AuthService) checkAuthorizeRequest(client *models.OAuthClient, req models.AuthorizeRequest) ([]string, *authzProblem) {
	if req.ResponseType != "code" || !contains(client.ResponseTypes, req.ResponseType) {
		return nil, &authzProblem{"unsupported_response_type", "Response type not supported"}
	}
	if !contains(client.GrantTypes, "authorization_code") {
		return nil, &authzProblem{"unauthorized_client", "This app may not use the authorization code grant"}
	}
	scopes := parseScopes(req.Scope)
	if !scopesAllowed(scopes, client.Scopes) {
		return nil, &authzProblem{"invalid_scope", "Invalid scope"}
	}
	if !validPKCEChallenge(req.CodeChallengeMethod, req.CodeChallenge) {
		return nil, &authzProblem{"invalid_request", "PKCE required: send a 43-character S256 code_challenge with code_challenge_method=S256"}
	}
	return scopes, nil
}

// jsonMode says the caller is the website, which can't follow a redirect to
// another site on its own: every answer is JSON, and where an HTTP redirect
// would have gone the answer is {"redirect": "<address>"}.
func jsonMode(c *gin.Context) bool { return c.GetBool("oauth_json") }

// sendRedirect sends the person to target: an HTTP redirect, or in JSON mode
// {"redirect": target} for the website to follow.
func sendRedirect(c *gin.Context, target string) {
	if jsonMode(c) {
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, gin.H{"redirect": target})
		return
	}
	c.Redirect(http.StatusFound, target)
}

// AuthorizeJSON is GET /api/v1/oauth/authorize, the website's way into the
// authorization flow. The browser can't send its login (a Bearer header) on
// a redirect, so the app sends the person to the website's /oauth/authorize
// page, and that page calls this with the same query and the header. It
// checks everything Authorize does and answers JSON, one of:
//
//	{"redirect": "<the app's redirect_uri with code or error>"}
//	{"consent": {...the consent screen...}}
//	{"error": "...", "error_description": "..."}   (400: unknown app, bad
//	    redirect_uri; shown on the page and never sent to the app)
//	{"error": "login_required"}                    (401)
func (as *AuthService) AuthorizeJSON(c *gin.Context) {
	c.Set("oauth_json", true)
	c.Header("Cache-Control", "no-store")
	as.Authorize(c)
}

func (as *AuthService) Authorize(c *gin.Context) {
	var req models.AuthorizeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "Invalid authorization request"})
		return
	}

	// Cap the size of what an app can make us hold on to.
	if len(req.State) > maxStateLen || len(req.Nonce) > maxNonceLen ||
		len(req.LoginHint) > maxLoginHintLen || len(req.RedirectURI) > maxRedirectURILen {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "A parameter is too long"})
		return
	}

	// Validate client. Nothing is stored for an unknown or switched-off app.
	client, err := as.getClientByID(req.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client", "error_description": "Invalid client"})
		return
	}

	// Validate redirect URI: it must be one the app registered, character for
	// character, and still acceptable today. Until it is verified, errors
	// are shown here and never redirected.
	if !contains(client.RedirectURIs, req.RedirectURI) || !validRedirectURI(req.RedirectURI, client.IsPublic) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "Invalid redirect_uri",
		})
		return
	}

	requestedScopes, problem := as.checkAuthorizeRequest(client, req)
	if problem != nil {
		as.redirectWithError(c, req.RedirectURI, req.State, problem.code, problem.desc)
		return
	}

	// Check if user is authenticated
	userID := as.getAuthenticatedUser(c)
	if userID == nil {
		if jsonMode(c) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "login_required"})
			return
		}
		// A top-level redirect can't carry the website's login, so this
		// entry point is always logged out. Send the person to the website's
		// own authorize page with the same query; it handles login and consent.
		c.Redirect(http.StatusFound, "/oauth/authorize?"+c.Request.URL.RawQuery)
		return
	}

	// Check consent. A trusted app may skip the screen only when it asks for
	// nothing but openid (a stable id that means nothing outside the app). A
	// request for profile, email or anything else needs the person's explicit
	// yes, trusted or not. prompt=consent always shows the screen, which is
	// how a person changes the pseud they share.
	openidOnly := len(requestedScopes) == 1 && requestedScopes[0] == "openid"
	if client.IsTrusted && openidOnly && req.Prompt != "consent" {
		// The approval is still recorded, so the person can see and take it back.
		if err := as.storeUserConsent(*userID, client.ID, requestedScopes); err != nil {
			log.Printf("oauth: record trusted consent: %v", err)
			as.redirectWithError(c, req.RedirectURI, req.State, "server_error", "Failed to record consent")
			return
		}
	} else if req.Prompt == "consent" || !as.hasValidConsent(*userID, client.ID, requestedScopes) {
		// Show consent screen
		as.showConsentScreen(c, client, requestedScopes, req)
		return
	}

	// Generate authorization code
	code, err := as.generateAuthorizationCode(*userID, client.ID, req)
	if err != nil {
		log.Printf("oauth: generate code: %v", err)
		as.redirectWithError(c, req.RedirectURI, req.State, "server_error", "Failed to generate authorization code")
		return
	}

	// Redirect back to client
	c.Header("Cache-Control", "no-store")
	sendRedirect(c, as.buildCallbackURL(req.RedirectURI, code, req.State))
}

// Token endpoint

func tokenError(c *gin.Context, status int, code, desc string) {
	c.JSON(status, models.TokenErrorResponse{Error: code, ErrorDescription: desc})
}

// noStore keeps proxies and browsers from keeping token responses.
func noStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
}

func (as *AuthService) Token(c *gin.Context) {
	noStore(c)
	if rejectSecretsInQuery(c) {
		return
	}

	grant := c.PostForm("grant_type")
	if grant != "authorization_code" && grant != "refresh_token" {
		tokenError(c, http.StatusBadRequest, "unsupported_grant_type", "Grant type not supported")
		return
	}

	// Public apps may skip the secret here, and only here.
	client, err := as.authenticateClient(c, authAllowPublic)
	if err != nil {
		clientAuthFailed(c, err)
		return
	}
	if !contains(client.GrantTypes, grant) {
		tokenError(c, http.StatusBadRequest, "unauthorized_client", "This app may not use that grant type")
		return
	}

	var result *tokenResult
	var gerr *grantError
	switch grant {
	case "authorization_code":
		result, gerr = as.exchangeCode(c.Request.Context(), client,
			c.PostForm("code"), c.PostForm("redirect_uri"), c.PostForm("code_verifier"),
			c.ClientIP(), c.GetHeader("User-Agent"))
	case "refresh_token":
		result, gerr = as.refreshTokens(c.Request.Context(), client,
			c.PostForm("refresh_token"), c.PostForm("scope"),
			c.ClientIP(), c.GetHeader("User-Agent"))
	}
	if gerr != nil {
		tokenError(c, gerr.status, gerr.code, gerr.desc)
		return
	}

	c.JSON(http.StatusOK, models.TokenResponse{
		AccessToken:  result.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    result.ExpiresIn,
		RefreshToken: result.RefreshToken,
		Scope:        strings.Join(result.Scopes, " "),
		IDToken:      result.IDToken,
	})
}

// User Info endpoint (OIDC)

func (as *AuthService) UserInfo(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
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

	// Only the subject for this app and what the person chose to share.
	userInfo, err := as.identityFor(c.Request.Context(), *accessToken.UserID, accessToken.ClientID, accessToken.Scopes)
	if err != nil {
		log.Printf("oauth: build userinfo: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":             "server_error",
			"error_description": "Failed to retrieve user info",
		})
		return
	}

	// Update last used timestamp
	go as.updateTokenLastUsed(accessToken.ID)

	c.JSON(http.StatusOK, userInfo)
}

// Token introspection

func (as *AuthService) Introspect(c *gin.Context) {
	noStore(c)
	if rejectSecretsInQuery(c) {
		return
	}

	// Only apps that can keep a secret may ask, and only about their own tokens.
	client, err := as.authenticateClient(c, authConfidentialOnly)
	if err != nil {
		clientAuthFailed(c, err)
		return
	}

	token := c.PostForm("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "Missing token parameter",
		})
		return
	}

	// Validate token
	accessToken, err := as.validateAccessToken(token)
	if err != nil || accessToken.ClientID != client.ID {
		// Unknown, expired, revoked, or another app's token: all look the same.
		c.JSON(http.StatusOK, models.IntrospectResponse{Active: false})
		return
	}

	// The subject is the one this app sees, never the account id.
	sub, err := as.subjectFor(c.Request.Context(), accessToken.ClientID, *accessToken.UserID)
	if err != nil {
		log.Printf("oauth: introspect subject: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	response := models.IntrospectResponse{
		Active:    true,
		Scope:     strings.Join(accessToken.Scopes, " "),
		ClientID:  accessToken.ClientID.String(),
		TokenType: accessToken.TokenType,
		ExpiresAt: accessToken.ExpiresAt.Unix(),
		IssuedAt:  accessToken.CreatedAt.Unix(),
		JWTID:     accessToken.ID.String(),
		Subject:   sub,
	}

	c.JSON(http.StatusOK, response)
}

// Token revocation

func (as *AuthService) Revoke(c *gin.Context) {
	noStore(c)
	if rejectSecretsInQuery(c) {
		return
	}

	// A public app may revoke with just its client_id (RFC 7009), and can
	// only reach its own tokens. A confidential app proves its secret.
	client, err := as.authenticateClient(c, authAllowPublic)
	if err != nil {
		clientAuthFailed(c, err)
		return
	}

	token := c.PostForm("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "Missing token parameter",
		})
		return
	}

	// token_type_hint is only a hint (RFC 7009): look for the token as both
	// kinds. Only this app's own tokens can be found, and revoking one also
	// revokes the token it is linked to.
	as.revokeOwnToken(c.Request.Context(), client.ID, hashSecret(token))

	// Return 200 even if token not found (per RFC 7009)
	c.Status(http.StatusOK)
}

// revokeOwnToken revokes the access or refresh token with this hash if it
// belongs to the app, along with its partner: a refresh token's access token,
// or an access token's refresh token.
func (as *AuthService) revokeOwnToken(ctx context.Context, clientID uuid.UUID, tokenHash string) {
	var accessID uuid.UUID
	err := as.db.QueryRowContext(ctx, `
		UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = now()
		WHERE token_hash = $1 AND client_id = $2 AND NOT is_revoked
		RETURNING access_token_id`, tokenHash, clientID).Scan(&accessID)
	if err == nil {
		as.db.ExecContext(ctx,
			`UPDATE oauth_access_tokens SET is_revoked = true, revoked_at = now() WHERE id = $1 AND client_id = $2 AND NOT is_revoked`,
			accessID, clientID)
	}

	err = as.db.QueryRowContext(ctx, `
		UPDATE oauth_access_tokens SET is_revoked = true, revoked_at = now()
		WHERE token_hash = $1 AND client_id = $2 AND NOT is_revoked
		RETURNING id`, tokenHash, clientID).Scan(&accessID)
	if err == nil {
		as.db.ExecContext(ctx,
			`UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = now() WHERE access_token_id = $1 AND client_id = $2 AND NOT is_revoked`,
			accessID, clientID)
	}
}

// Helper functions

const clientColumns = `client_id, COALESCE(client_secret, ''), client_name, COALESCE(description, ''),
	COALESCE(website, ''), COALESCE(logo_url, ''), redirect_uris, scopes, grant_types, response_types,
	is_public, is_confidential, is_trusted, is_first_party, owner_id, access_token_ttl, refresh_token_ttl,
	status, created_at, updated_at, revision`

type rowScanner interface{ Scan(dest ...any) error }

func scanClient(row rowScanner) (*models.OAuthClient, error) {
	client := &models.OAuthClient{}
	err := row.Scan(
		&client.ID, &client.Secret, &client.Name, &client.Description, &client.Website, &client.LogoURL,
		pq.Array(&client.RedirectURIs), pq.Array(&client.Scopes), pq.Array(&client.GrantTypes),
		pq.Array(&client.ResponseTypes), &client.IsPublic, &client.IsConfidential,
		&client.IsTrusted, &client.IsFirstParty, &client.OwnerID, &client.AccessTokenTTL,
		&client.RefreshTokenTTL, &client.Status, &client.CreatedAt, &client.UpdatedAt, &client.Revision)
	if err != nil {
		return nil, err
	}
	// is_active is a copy of the status (migration 047); read it from there.
	client.IsActive = client.Status == "approved"
	return client, nil
}

// getClientByID finds an app that is approved. It is what every user-facing
// and app-facing path uses: an app that is pending, rejected or suspended
// does not exist to them.
func (as *AuthService) getClientByID(clientID string) (*models.OAuthClient, error) {
	id, err := uuid.Parse(clientID)
	if err != nil {
		return nil, err
	}
	return scanClient(as.db.QueryRow(
		`SELECT `+clientColumns+` FROM oauth_clients WHERE client_id = $1 AND status = 'approved'`, id))
}

// getClientForAdmin finds an app in any status.
func (as *AuthService) getClientForAdmin(id uuid.UUID) (*models.OAuthClient, error) {
	return scanClient(as.db.QueryRow(
		`SELECT `+clientColumns+` FROM oauth_clients WHERE client_id = $1`, id))
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (as *AuthService) isScopeSubset(subset, superset []string) bool {
	return subsetOf(subset, superset)
}

// errorRedirectURL builds the address that sends an error back to an app.
// The pieces are set with the URL parser, so nothing the app or the person
// typed can add parameters or change the destination. It returns "" if the
// address does not parse.
func errorRedirectURL(redirectURI, state, errorCode, errorDescription string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("error", errorCode)
	if errorDescription != "" {
		q.Set("error_description", errorDescription)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (as *AuthService) redirectWithError(c *gin.Context, redirectURI, state, errorCode, errorDescription string) {
	target := ""
	if redirectURI != "" {
		target = errorRedirectURL(redirectURI, state, errorCode, errorDescription)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             errorCode,
			"error_description": errorDescription,
		})
		return
	}
	sendRedirect(c, target)
}

func (as *AuthService) buildCallbackURL(redirectURI, code, state string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
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
