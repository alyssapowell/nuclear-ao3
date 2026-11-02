package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

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
// that covers every one of the scopes. The profile scope also needs a chosen
// pseud that still exists, and the email scope needs an explicit yes to
// email; without them the app has to ask again.
func (as *AuthService) hasValidConsent(userID, clientID uuid.UUID, scopes []string) bool {
	query := `
		SELECT scopes, pseud_id IS NOT NULL, share_email FROM user_consents
		WHERE user_id = $1 AND client_id = $2 AND is_revoked = false
		AND (expires_at IS NULL OR expires_at > NOW())`

	var consentedScopes []string
	var hasPseud, shareEmail bool
	err := as.db.QueryRow(query, userID, clientID).Scan(pq.Array(&consentedScopes), &hasPseud, &shareEmail)
	if err != nil {
		return false
	}

	// Admin-only scopes never count, whatever an old row says.
	for _, s := range scopes {
		if !models.IsGrantableScope(s) {
			return false
		}
	}
	if contains(scopes, "profile") && !hasPseud {
		return false
	}
	if contains(scopes, "email") && !shareEmail {
		return false
	}
	return as.isScopeSubset(scopes, consentedScopes)
}

// pendingConsent is a consent request waiting for the person's answer. It
// names the person and the app by id only.
type pendingConsent struct {
	UserID           string                  `json:"user_id"`
	ClientID         string                  `json:"client_id"`
	Scopes           []string                `json:"scopes"`
	AuthorizeRequest models.AuthorizeRequest `json:"authorize_request"`
}

func (as *AuthService) readPendingConsent(ctx context.Context, consentID string) (*pendingConsent, error) {
	raw, err := as.getPending(ctx, "consent:"+consentID)
	if err != nil {
		return nil, err
	}
	var p pendingConsent
	if err := json.Unmarshal(raw, &p); err != nil || p.ClientID == "" {
		return nil, errors.New("invalid consent data")
	}
	return &p, nil
}

func (as *AuthService) showConsentScreen(c *gin.Context, client *models.OAuthClient, scopes []string, req models.AuthorizeRequest) {
	// Store the consent request, tied to the user who is looking at it, so a
	// consent ID can't be approved by anyone else. Only ids are kept, never
	// the app's record (which holds its secret hash).
	viewer := as.getAuthenticatedUser(c)
	if viewer == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	consentID := uuid.New().String()
	consentJSON, _ := json.Marshal(pendingConsent{
		UserID:           viewer.String(),
		ClientID:         client.ID.String(),
		Scopes:           scopes,
		AuthorizeRequest: req,
	})
	if err := as.putPending(c.Request.Context(), "consent:"+consentID, consentJSON); err != nil {
		log.Printf("oauth: store consent request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not start consent"})
		return
	}

	view, err := as.consentView(c.Request.Context(), consentID, client, scopes, *viewer)
	if err != nil {
		log.Printf("oauth: build consent screen: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not start consent"})
		return
	}
	// The website shows the consent screen from this answer
	view["consent_required"] = true
	view["consent_url"] = fmt.Sprintf("/auth/consent/%s", consentID)
	// Built with the URL parser, so a hostile state value can't add
	// parameters or change where it points.
	view["cancel_url"] = errorRedirectURL(req.RedirectURI, req.State, "access_denied", "")
	if jsonMode(c) {
		c.JSON(http.StatusOK, gin.H{"consent": view})
		return
	}
	c.JSON(http.StatusOK, view)
}

// processConsent answers a consent screen. pseudRaw is the pseud to share
// (empty: the default). shareEmail is the person's own choice about their
// address: if the app asked for email and it is false, the app is allowed
// without the email scope rather than refused.
func (as *AuthService) processConsent(c *gin.Context, consentID string, approved bool, pseudRaw string, shareEmail bool) {
	ctx := c.Request.Context()
	consentData, err := as.readPendingConsent(ctx, consentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consent request"})
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

	// The pseud the person chose to share. It has to be one of their own;
	// this is checked before the request is used up, so a mistake can be
	// fixed. Left out, it is their default pseud.
	var chosen *uuid.UUID
	if raw := strings.TrimSpace(pseudRaw); raw != "" && approved {
		id, perr := uuid.Parse(raw)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pseud_id"})
			return
		}
		own, perr := as.userOwnsPseud(ctx, *userID, id)
		if perr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not check the pseud"})
			return
		}
		if !own {
			c.JSON(http.StatusForbidden, gin.H{"error": "That pseud is not yours"})
			return
		}
		chosen = &id
	}

	// Each consent request can be answered once. If a second answer got here
	// first, there is nothing left to take.
	if _, err := as.takePending(ctx, "consent:"+consentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consent request"})
		return
	}

	// The app may have been switched off or changed since the screen was
	// shown, so check the request again against the app as it is now.
	req := consentData.AuthorizeRequest
	client, err := as.getClientByID(consentData.ClientID)
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
	// The person said no to sharing their email: the app gets everything
	// else it asked for, and the code and the record carry no email scope.
	if !shareEmail && contains(scopes, "email") {
		kept := make([]string, 0, len(scopes))
		for _, sc := range scopes {
			if sc != "email" {
				kept = append(kept, sc)
			}
		}
		if len(kept) == 0 {
			as.redirectWithError(c, req.RedirectURI, req.State, "access_denied", "User denied access")
			return
		}
		scopes = kept
		req.Scope = strings.Join(kept, " ")
	}

	// Store consent
	if err := as.storeUserConsentWithPseud(ctx, *userID, client.ID, scopes, chosen); err != nil {
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

	sendRedirect(c, as.buildCallbackURL(req.RedirectURI, code, req.State))
}

// storeUserConsent records that the user allows an app these scopes, sharing
// their default pseud if the scopes include profile.
func (as *AuthService) storeUserConsent(userID, clientID uuid.UUID, scopes []string) error {
	return as.storeUserConsentWithPseud(context.Background(), userID, clientID, scopes, nil)
}

// storeUserConsentWithPseud records that the user allows an app these scopes.
// A new approval adds to what was approved before; after a revocation it
// starts again from only what was just approved. When the scopes include
// profile, pseudID (which the caller has checked belongs to the user; nil
// means their default pseud) becomes the one pseud the app sees, replacing
// an earlier choice. Email is shared only if this approval includes the
// email scope.
func (as *AuthService) storeUserConsentWithPseud(ctx context.Context, userID, clientID uuid.UUID, scopes []string, pseudID *uuid.UUID) error {
	var pseud interface{} // NULL unless the app is to see a pseud
	if contains(scopes, "profile") {
		if pseudID == nil {
			def, err := as.defaultPseudID(ctx, userID)
			if err != nil {
				return fmt.Errorf("find default pseud: %w", err)
			}
			pseudID = &def
		}
		pseud = *pseudID
	}
	query := `
		INSERT INTO user_consents (id, user_id, client_id, scopes, granted_at, is_revoked, pseud_id, share_email)
		VALUES ($1, $2, $3, $4, now(), false, $5, $6)
		ON CONFLICT (user_id, client_id)
		DO UPDATE SET
			scopes = CASE WHEN user_consents.is_revoked THEN EXCLUDED.scopes
				ELSE ARRAY(SELECT DISTINCT s FROM unnest(user_consents.scopes || EXCLUDED.scopes) s) END,
			pseud_id = CASE WHEN user_consents.is_revoked THEN EXCLUDED.pseud_id
				ELSE COALESCE(EXCLUDED.pseud_id, user_consents.pseud_id) END,
			share_email = EXCLUDED.share_email OR (NOT user_consents.is_revoked AND user_consents.share_email),
			granted_at = now(), is_revoked = false, revoked_at = NULL`

	_, err := as.db.ExecContext(ctx, query, uuid.New(), userID, clientID, pq.Array(scopes), pseud, contains(scopes, "email"))
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
		`UPDATE user_consents        SET is_revoked = true, revoked_at = now(), pseud_id = NULL, share_email = false WHERE user_id = $1 AND client_id = $2 AND NOT is_revoked`,
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
		qs = append(qs, `UPDATE user_consents SET is_revoked = true, revoked_at = now(), pseud_id = NULL, share_email = false WHERE client_id = $1 AND NOT is_revoked`)
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
