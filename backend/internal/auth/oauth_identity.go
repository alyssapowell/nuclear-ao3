package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// What an app is told about a person (security audit, phase 2).
//
// Two rules decide everything in this file:
//  1. No linking across apps. Every app gets its own random subject for the
//     same person, and nothing else an app receives is the same for two apps.
//  2. Pseud rules. An app sees only the one pseud the person chose to share,
//     as {id, name}. Never the account id, username, other pseuds, roles or
//     counts. Email only with the email scope and explicit consent.

// subjectFor returns the subject this app sees for this person, creating it
// the first time. It is 32 random bytes, so it can't be worked out from the
// account id and can't be matched with any other app's subject.
func (as *AuthService) subjectFor(ctx context.Context, clientID, userID uuid.UUID) (string, error) {
	var sub string
	err := as.db.QueryRowContext(ctx,
		`SELECT sub FROM oauth_subjects WHERE client_id = $1 AND user_id = $2`, clientID, userID).Scan(&sub)
	if err == nil {
		return sub, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	fresh, err := randomToken()
	if err != nil {
		return "", err
	}
	// If two requests race, the first one wins and both return its value.
	err = as.db.QueryRowContext(ctx, `
		INSERT INTO oauth_subjects (client_id, user_id, sub) VALUES ($1, $2, $3)
		ON CONFLICT (client_id, user_id) DO UPDATE SET sub = oauth_subjects.sub
		RETURNING sub`, clientID, userID, fresh).Scan(&sub)
	return sub, err
}

// identityFor builds what the app may know about the person, for the scopes
// the token has. It reads the person's live approval, so a revoked approval,
// a deleted pseud or an email that was not explicitly approved gives nothing
// beyond the subject.
func (as *AuthService) identityFor(ctx context.Context, userID, clientID uuid.UUID, scopes []string) (models.UserInfoResponse, error) {
	var out models.UserInfoResponse
	sub, err := as.subjectFor(ctx, clientID, userID)
	if err != nil {
		return out, err
	}
	out.Subject = sub
	wantProfile, wantEmail := contains(scopes, "profile"), contains(scopes, "email")
	if !wantProfile && !wantEmail {
		return out, nil
	}

	var (
		pseudID    uuid.NullUUID
		pseudName  sql.NullString
		shareEmail bool
		mail       string
		verified   bool
	)
	// The pseud must still exist and belong to this person. If it is gone
	// the join finds nothing and the app gets no pseud; it is never swapped
	// for another one.
	err = as.db.QueryRowContext(ctx, `
		SELECT p.id, p.name, uc.share_email, u.email, u.is_verified
		FROM user_consents uc
		JOIN users u ON u.id = uc.user_id
		LEFT JOIN pseuds p ON p.id = uc.pseud_id AND p.user_id = uc.user_id
		WHERE uc.user_id = $1 AND uc.client_id = $2 AND NOT uc.is_revoked
		  AND (uc.expires_at IS NULL OR uc.expires_at > now())`, userID, clientID).Scan(
		&pseudID, &pseudName, &shareEmail, &mail, &verified)
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if wantProfile && pseudID.Valid && pseudName.Valid {
		out.Pseud = &models.PseudClaim{ID: pseudID.UUID.String(), Name: pseudName.String}
		out.Name = pseudName.String
	}
	if wantEmail && shareEmail && mail != "" {
		out.Email = mail
		out.EmailVerified = &verified
	}
	return out, nil
}

// idTokenParams is what goes into one ID token.
type idTokenParams struct {
	userID, clientID uuid.UUID
	scopes           []string
	nonce            string    // from the authorization request; may be empty
	accessToken      string    // issued alongside; gives at_hash
	authTime         time.Time // when this app's sign-in happened
}

// atHash is the OIDC at_hash: the left half of the SHA-256 of the access
// token, base64url (RS256 uses SHA-256).
func atHash(accessToken string) string {
	sum := sha256.Sum256([]byte(accessToken))
	return base64.RawURLEncoding.EncodeToString(sum[:len(sum)/2])
}

// generateIDToken signs an ID token for the app with the OIDC key. Empty
// claims are left out. auth_time is when the sign-in with this app happened,
// not the person's last login: a last-login time would be the same in every
// app and would let two apps match the person up.
func (as *AuthService) generateIDToken(ctx context.Context, p idTokenParams) (string, error) {
	issuer, err := oauthIssuer()
	if err != nil {
		return "", err
	}
	who, err := as.identityFor(ctx, p.userID, p.clientID, p.scopes)
	if err != nil {
		return "", err
	}
	now := time.Now()
	authTime := p.authTime
	if authTime.IsZero() {
		authTime = now
	}
	claims := jwt.MapClaims{
		"iss":       issuer,
		"sub":       who.Subject,
		"aud":       p.clientID.String(),
		"azp":       p.clientID.String(),
		"exp":       now.Add(time.Hour).Unix(),
		"iat":       now.Unix(),
		"auth_time": authTime.Unix(),
	}
	if p.nonce != "" {
		claims["nonce"] = p.nonce
	}
	if p.accessToken != "" {
		claims["at_hash"] = atHash(p.accessToken)
	}
	if who.Name != "" {
		claims["name"] = who.Name
	}
	if who.Pseud != nil {
		claims["pseud"] = who.Pseud
	}
	if who.Email != "" {
		claims["email"] = who.Email
		claims["email_verified"] = who.EmailVerified != nil && *who.EmailVerified
	}

	signer := as.oidcSigner()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	// Name the signing key, so apps can pick the right key from the JWKS.
	token.Header["kid"] = signer.keyID
	return token.SignedString(signer.privateKey)
}

// oidcSigner is the key set that signs ID tokens: the separate OIDC key when
// one is configured, otherwise the login key.
func (as *AuthService) oidcSigner() *JWTManager {
	if as.oidc != nil {
		return as.oidc
	}
	return as.jwt
}

// ---------------------------------------------------------------------
// Pseuds and the consent screen
// ---------------------------------------------------------------------

// pseudChoice is a pseud as the person's own consent screen lists it.
type pseudChoice struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// listPseuds returns the person's own pseuds, default first. It is only for
// showing them their own choices.
func (as *AuthService) listPseuds(ctx context.Context, userID uuid.UUID) ([]pseudChoice, error) {
	rows, err := as.db.QueryContext(ctx,
		`SELECT id, name FROM pseuds WHERE user_id = $1 ORDER BY is_default DESC, lower(name), id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []pseudChoice{}
	for rows.Next() {
		var p pseudChoice
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// defaultPseudID is the person's default pseud.
func (as *AuthService) defaultPseudID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := as.db.QueryRowContext(ctx,
		`SELECT id FROM pseuds WHERE user_id = $1 AND is_default ORDER BY created_at, id LIMIT 1`, userID).Scan(&id)
	return id, err
}

// userOwnsPseud says whether the pseud belongs to the person.
func (as *AuthService) userOwnsPseud(ctx context.Context, userID, pseudID uuid.UUID) (bool, error) {
	var ok bool
	err := as.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM pseuds WHERE id = $1 AND user_id = $2)`, pseudID, userID).Scan(&ok)
	return ok, err
}

// consentView is what a consent screen needs: the app, what it asks for, and
// the person's own pseuds to choose from. It holds no account id or username.
func (as *AuthService) consentView(ctx context.Context, consentID string, client *models.OAuthClient, scopes []string, userID uuid.UUID) (gin.H, error) {
	type scopeView struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	views := make([]scopeView, 0, len(scopes))
	for _, s := range scopes {
		views = append(views, scopeView{Name: s, Description: models.AO3OAuthScopes[s].Description})
	}
	view := gin.H{
		"consent_id": consentID,
		"app": gin.H{
			"name":     client.Name,
			"website":  client.Website,
			"logo_url": client.LogoURL,
		},
		"scopes": views,
	}
	// Pseuds are only offered when the app asks to see one.
	if contains(scopes, "profile") {
		pseuds, err := as.listPseuds(ctx, userID)
		if err != nil {
			return nil, err
		}
		view["pseuds"] = pseuds
		if def, err := as.defaultPseudID(ctx, userID); err == nil {
			view["default_pseud_id"] = def
		}
	}
	return view, nil
}

// ShowConsent returns the pending consent request as JSON for the consent
// screen. Only the signed-in person it belongs to can see it; anyone else
// gets the same answer as for a request that doesn't exist.
func (as *AuthService) ShowConsent(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	viewer := as.getAuthenticatedUser(c)
	if viewer == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	notFound := func() {
		c.JSON(http.StatusNotFound, gin.H{"error": "Consent request not found or expired"})
	}

	consentID := c.Param("consent_id")
	pending, err := as.readPendingConsent(c.Request.Context(), consentID)
	if err != nil || pending.UserID != viewer.String() {
		notFound()
		return
	}
	client, err := as.getClientByID(pending.ClientID)
	if err != nil {
		notFound()
		return
	}
	view, err := as.consentView(c.Request.Context(), consentID, client, pending.Scopes, *viewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load the consent request"})
		return
	}
	c.JSON(http.StatusOK, view)
}
