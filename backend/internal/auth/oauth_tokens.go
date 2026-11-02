package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"log"
	"net"
	"time"

	"nuclear-ao3/shared/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Issuing and rotating OAuth tokens. Tokens and codes are random values that
// are given to the app once; the database keeps only their SHA-256 (see
// hashSecret). Nothing in this file logs a token, a code or a secret.

var errInvalidToken = errors.New("invalid access token")

// tokenResult is what a successful token request returns.
type tokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	Scopes       []string
	IDToken      string
}

// grantError is a refused token request, ready to send.
type grantError struct {
	status int
	code   string
	desc   string
}

func (e *grantError) Error() string { return e.code + ": " + e.desc }

func invalidGrant(desc string) *grantError {
	return &grantError{400, "invalid_grant", desc}
}

func serverFailure(what string, err error) *grantError {
	log.Printf("oauth: %s: %v", what, err)
	return &grantError{500, "server_error", "Could not complete the request"}
}

// randomToken is 32 random bytes as URL-safe base64 (43 characters).
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateSecureToken() (string, error) { return randomToken() }

func generateClientSecret() (string, error) { return randomToken() }

// inetOrZero makes a client address fit an inet column: an address that does
// not parse (for example none, behind a unix socket) is stored as 0.0.0.0, as
// sessions do, instead of failing the whole request.
func inetOrZero(ip string) string {
	if net.ParseIP(ip) == nil {
		return "0.0.0.0"
	}
	return ip
}

func capLen(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// tokenParams describes one set of tokens to store.
type tokenParams struct {
	userID, clientID uuid.UUID
	scopes           []string
	accessTTL        int // seconds, already clamped
	refreshTTL       int // seconds, already clamped
	withRefresh      bool
	familyID         uuid.UUID
	familyCreated    time.Time // zero for a new family
	codeHash         string    // the code these came from, if any
	ip, userAgent    string
}

// insertTokens stores an access token and, if asked, a refresh token, and
// returns the raw values and the access token's id. The refresh token never
// outlives the 90-day cap on its family.
func (as *AuthService) insertTokens(ctx context.Context, tx *sql.Tx, p tokenParams) (access, refresh string, accessID uuid.UUID, err error) {
	if access, err = randomToken(); err != nil {
		return
	}
	accessID = uuid.New()
	code := sql.NullString{String: p.codeHash, Valid: p.codeHash != ""}
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO oauth_access_tokens (
			id, token_hash, user_id, client_id, scopes, token_type, expires_at,
			is_revoked, ip_address, user_agent, family_id, code_hash
		) VALUES ($1, $2, $3, $4, $5, 'Bearer', now() + $6 * interval '1 second', false, $7, $8, $9, $10)`,
		accessID, hashSecret(access), p.userID, p.clientID, pq.Array(p.scopes), p.accessTTL,
		inetOrZero(p.ip), capLen(p.userAgent, 512), p.familyID, code); err != nil {
		return
	}
	if !p.withRefresh {
		return
	}
	if refresh, err = randomToken(); err != nil {
		return
	}
	created := sql.NullTime{Time: p.familyCreated, Valid: !p.familyCreated.IsZero()}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO oauth_refresh_tokens (
			id, token_hash, access_token_id, user_id, client_id, scopes, expires_at,
			is_revoked, family_id, family_created_at, code_hash
		) VALUES ($1, $2, $3, $4, $5, $6,
			LEAST(now() + $7 * interval '1 second', COALESCE($8::timestamptz, now()) + interval '90 days'),
			false, $9, COALESCE($8::timestamptz, now()), $10)`,
		uuid.New(), hashSecret(refresh), accessID, p.userID, p.clientID, pq.Array(p.scopes),
		p.refreshTTL, created, p.familyID, code)
	return
}

// userMayUseApps is true for an account that is active, not banned and not
// suspended (the same rule as user_standing(), migration 033).
func (as *AuthService) userMayUseApps(ctx context.Context, userID uuid.UUID) (bool, error) {
	var active bool
	var standing sql.NullString
	err := as.db.QueryRowContext(ctx,
		`SELECT COALESCE(is_active, false), user_standing(id) FROM users WHERE id = $1`, userID).Scan(&active, &standing)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return active && !standing.Valid, nil
}

// revokeTokensOfCode cancels every token that came from an authorization
// code. It runs when a code is presented a second time (RFC 6749 4.1.2).
func (as *AuthService) revokeTokensOfCode(ctx context.Context, codeHash string, clientID uuid.UUID) {
	for _, q := range []string{
		`UPDATE oauth_access_tokens  SET is_revoked = true, revoked_at = now() WHERE code_hash = $1 AND client_id = $2 AND NOT is_revoked`,
		`UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = now() WHERE code_hash = $1 AND client_id = $2 AND NOT is_revoked`,
	} {
		if _, err := as.db.ExecContext(ctx, q, codeHash, clientID); err != nil {
			log.Printf("oauth: revoke tokens of a reused code: %v", err)
		}
	}
}

// revokeFamily cancels every token in a refresh-token family. It runs when a
// refresh token that was already used is presented again.
func (as *AuthService) revokeFamily(ctx context.Context, familyID uuid.UUID) {
	for _, q := range []string{
		`UPDATE oauth_access_tokens  SET is_revoked = true, revoked_at = now() WHERE family_id = $1 AND NOT is_revoked`,
		`UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = now() WHERE family_id = $1 AND NOT is_revoked`,
	} {
		if _, err := as.db.ExecContext(ctx, q, familyID); err != nil {
			log.Printf("oauth: revoke a token family: %v", err)
		}
	}
}

// exchangeCode turns an authorization code into tokens. The code is used up
// in the same transaction that stores the tokens, with one UPDATE that only
// one of two simultaneous requests can win.
func (as *AuthService) exchangeCode(ctx context.Context, client *models.OAuthClient, code, redirectURI, verifier, ip, userAgent string) (*tokenResult, *grantError) {
	if code == "" || redirectURI == "" || !validPKCEVerifier(verifier) {
		return nil, invalidGrant("Invalid authorization code")
	}
	codeHash := hashSecret(code)

	// Look at the code first. Everything that can refuse the request is
	// checked here, before the code is used up, and without holding a
	// transaction open.
	var (
		userID    uuid.UUID
		scopes    []string
		nonce     string
		challenge string
		method    string
		authTime  time.Time
	)
	err := as.db.QueryRowContext(ctx, `
		SELECT user_id, scopes, COALESCE(nonce, ''), code_challenge, code_challenge_method, created_at
		FROM authorization_codes
		WHERE code_hash = $1 AND client_id = $2 AND redirect_uri = $3
		  AND used_at IS NULL AND expires_at > now()`,
		codeHash, client.ID, redirectURI).Scan(&userID, pq.Array(&scopes), &nonce, &challenge, &method, &authTime)
	if err == sql.ErrNoRows {
		// Unknown, expired, for another client or address, or already used.
		// If it was used, whatever it produced must stop working.
		as.revokeTokensOfCode(ctx, codeHash, client.ID)
		return nil, invalidGrant("Invalid authorization code")
	}
	if err != nil {
		return nil, serverFailure("read authorization code", err)
	}
	if !as.validatePKCE(challenge, method, verifier) {
		return nil, invalidGrant("Invalid code verifier")
	}
	if !scopesAllowed(scopes, client.Scopes) {
		return nil, &grantError{400, "invalid_scope", "The code asks for a scope this app may not have"}
	}
	if ok, err := as.userMayUseApps(ctx, userID); err != nil {
		return nil, serverFailure("check account", err)
	} else if !ok {
		return nil, invalidGrant("This account can't be used with apps right now")
	}

	accessTTL := clampAccessTTL(client.AccessTokenTTL)
	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, serverFailure("begin", err)
	}
	defer tx.Rollback()

	// Only one request can get a row back from this.
	var won uuid.UUID
	err = tx.QueryRowContext(ctx, `
		UPDATE authorization_codes SET used_at = now()
		WHERE code_hash = $1 AND client_id = $2 AND redirect_uri = $3
		  AND used_at IS NULL AND expires_at > now()
		RETURNING user_id`, codeHash, client.ID, redirectURI).Scan(&won)
	if err == sql.ErrNoRows {
		tx.Rollback()
		// Lost a race with another use of the same code: that is a replay.
		as.revokeTokensOfCode(ctx, codeHash, client.ID)
		return nil, invalidGrant("Invalid authorization code")
	}
	if err != nil {
		return nil, serverFailure("use authorization code", err)
	}

	access, refresh, _, err := as.insertTokens(ctx, tx, tokenParams{
		userID: userID, clientID: client.ID, scopes: scopes,
		accessTTL:   accessTTL,
		refreshTTL:  clampRefreshTTL(client.RefreshTokenTTL),
		withRefresh: contains(client.GrantTypes, "refresh_token"),
		familyID:    uuid.New(), codeHash: codeHash, ip: ip, userAgent: userAgent,
	})
	if err != nil {
		return nil, serverFailure("store tokens", err)
	}
	// The ID token carries a hash of the access token, so it is signed once
	// that exists. If signing fails nothing is committed and the code stays
	// unused.
	idToken := ""
	if contains(scopes, "openid") {
		if idToken, err = as.generateIDToken(ctx, idTokenParams{
			userID: userID, clientID: client.ID, scopes: scopes,
			nonce: nonce, accessToken: access, authTime: authTime,
		}); err != nil {
			return nil, serverFailure("sign id token", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, serverFailure("commit", err)
	}
	return &tokenResult{AccessToken: access, RefreshToken: refresh, ExpiresIn: accessTTL, Scopes: scopes, IDToken: idToken}, nil
}

// refreshTokens swaps a refresh token for a new access token and a new
// refresh token. The old refresh token is used up by one UPDATE that only one
// of two simultaneous requests can win. Presenting one that was already
// used cancels the whole family.
func (as *AuthService) refreshTokens(ctx context.Context, client *models.OAuthClient, raw, requestedScope, ip, userAgent string) (*tokenResult, *grantError) {
	if raw == "" {
		return nil, invalidGrant("Invalid refresh token")
	}
	var (
		id, familyID, userID, prevAccessID uuid.UUID
		scopes                             []string
		revoked, expired, chainOver        bool
		familyCreated                      time.Time
		codeHash                           sql.NullString
	)
	err := as.db.QueryRowContext(ctx, `
		SELECT id, family_id, user_id, access_token_id, scopes, is_revoked,
		       expires_at <= now(), family_created_at + interval '90 days' <= now(),
		       family_created_at, code_hash
		FROM oauth_refresh_tokens
		WHERE token_hash = $1 AND client_id = $2`, hashSecret(raw), client.ID).Scan(
		&id, &familyID, &userID, &prevAccessID, pq.Array(&scopes), &revoked,
		&expired, &chainOver, &familyCreated, &codeHash)
	if err == sql.ErrNoRows {
		return nil, invalidGrant("Invalid refresh token")
	}
	if err != nil {
		return nil, serverFailure("read refresh token", err)
	}
	if revoked {
		// Someone is presenting a token that was already rotated or revoked.
		// Either the app or a thief has a stale copy, so nothing from this
		// sign-in can be trusted any more.
		as.revokeFamily(ctx, familyID)
		return nil, invalidGrant("Invalid refresh token")
	}
	if expired || chainOver {
		return nil, invalidGrant("Refresh token expired")
	}

	// A refresh may keep the scopes or ask for fewer, never more.
	granted := scopes
	if requested := parseScopes(requestedScope); len(requested) > 0 {
		if !subsetOf(requested, scopes) {
			return nil, &grantError{400, "invalid_scope", "Requested scope exceeds original grant"}
		}
		granted = requested
	}
	if !scopesAllowed(granted, client.Scopes) {
		return nil, &grantError{400, "invalid_scope", "The token has a scope this app may not have"}
	}
	if ok, err := as.userMayUseApps(ctx, userID); err != nil {
		return nil, serverFailure("check account", err)
	} else if !ok {
		return nil, invalidGrant("This account can't be used with apps right now")
	}
	if !as.hasValidConsent(userID, client.ID, granted) {
		return nil, invalidGrant("The user no longer allows this app")
	}

	accessTTL := clampAccessTTL(client.AccessTokenTTL)
	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, serverFailure("begin", err)
	}
	defer tx.Rollback()

	// Only one request can get a row back from this.
	var prev uuid.UUID
	err = tx.QueryRowContext(ctx, `
		UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = now(), last_used = now()
		WHERE id = $1 AND NOT is_revoked
		RETURNING access_token_id`, id).Scan(&prev)
	if err == sql.ErrNoRows {
		tx.Rollback()
		as.revokeFamily(ctx, familyID)
		return nil, invalidGrant("Invalid refresh token")
	}
	if err != nil {
		return nil, serverFailure("rotate refresh token", err)
	}
	// The access token that came with the old refresh token stops working.
	if _, err := tx.ExecContext(ctx,
		`UPDATE oauth_access_tokens SET is_revoked = true, revoked_at = now() WHERE id = $1 AND NOT is_revoked`, prev); err != nil {
		return nil, serverFailure("revoke previous access token", err)
	}
	access, refresh, _, err := as.insertTokens(ctx, tx, tokenParams{
		userID: userID, clientID: client.ID, scopes: granted,
		accessTTL:   accessTTL,
		refreshTTL:  clampRefreshTTL(client.RefreshTokenTTL),
		withRefresh: true,
		familyID:    familyID, familyCreated: familyCreated, codeHash: codeHash.String,
		ip: ip, userAgent: userAgent,
	})
	if err != nil {
		return nil, serverFailure("store tokens", err)
	}
	idToken := ""
	if contains(granted, "openid") {
		// auth_time stays the time of the original sign-in with this app.
		if idToken, err = as.generateIDToken(ctx, idTokenParams{
			userID: userID, clientID: client.ID, scopes: granted,
			accessToken: access, authTime: familyCreated,
		}); err != nil {
			return nil, serverFailure("sign id token", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, serverFailure("commit", err)
	}
	return &tokenResult{AccessToken: access, RefreshToken: refresh, ExpiresIn: accessTTL, Scopes: granted, IDToken: idToken}, nil
}

func subsetOf(sub, super []string) bool {
	for _, s := range sub {
		if !contains(super, s) {
			return false
		}
	}
	return true
}

// validateAccessToken finds a live access token. The app must still be
// approved and the person must still be an active user who is not banned or
// suspended, so the token stops working the moment either changes.
func (as *AuthService) validateAccessToken(token string) (*models.OAuthAccessToken, error) {
	if token == "" || len(token) > 512 {
		return nil, errInvalidToken
	}
	t := &models.OAuthAccessToken{}
	var userID uuid.UUID
	err := as.db.QueryRow(`
		SELECT t.id, t.user_id, t.client_id, t.scopes, t.token_type, t.expires_at, t.created_at
		FROM oauth_access_tokens t
		JOIN oauth_clients c ON c.client_id = t.client_id AND c.status = 'approved'
		JOIN users u ON u.id = t.user_id AND u.is_active
		WHERE t.token_hash = $1 AND NOT t.is_revoked AND t.expires_at > now()
		  AND user_standing(u.id) IS NULL`, hashSecret(token)).Scan(
		&t.ID, &userID, &t.ClientID, pq.Array(&t.Scopes), &t.TokenType, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		return nil, errInvalidToken
	}
	t.UserID = &userID
	return t, nil
}
