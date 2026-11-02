package auth

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/google/uuid"

	"nuclear-ao3/internal/platform"
)

// This file is the resource side of the OAuth provider: it answers "who is
// this access token for?" when an app calls the API. The route checks that
// follow (scopes, the denylist, rate limits) are in internal/platform.

// appLastUsedEvery is the most often a token's last_used is written, the same
// as for sessions (see sessions.go), so a busy app does not turn every
// request into a database write.
const appLastUsedEvery = time.Minute

// ResolveAppToken turns an opaque access token into the person and app it was
// issued to. It refuses a token that is unknown, expired or revoked, whose
// app is not approved, or whose person is not active and in good standing
// (validateAccessToken does that in one query, so any of these stops working
// the moment it changes).
//
// It also refuses people who hold a staff role (moderator, tag wrangler,
// admin). Visibility is decided by what the person may see, and staff may see
// hidden works; an app must never see more than an ordinary reader, whatever
// scope it was given. Staff use the website, or a separate ordinary account,
// with apps.
func (as *AuthService) ResolveAppToken(ctx context.Context, token string) (*platform.AppToken, error) {
	t, err := as.validateAccessToken(token)
	if err != nil {
		return nil, err
	}
	var staff bool
	if err := as.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role <> 'user' AND revoked_at IS NULL)`,
		*t.UserID).Scan(&staff); err != nil {
		return nil, err
	}
	if staff {
		return nil, errInvalidToken
	}

	// The pseud the person shared with this app, if it still exists and is
	// still theirs. If it is gone there is none; no other is swapped in.
	var pseud uuid.NullUUID
	err = as.db.QueryRowContext(ctx, `
		SELECT p.id FROM user_consents uc
		JOIN pseuds p ON p.id = uc.pseud_id AND p.user_id = uc.user_id
		WHERE uc.user_id = $1 AND uc.client_id = $2 AND NOT uc.is_revoked`,
		*t.UserID, t.ClientID).Scan(&pseud)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if _, err := as.db.ExecContext(ctx, `
		UPDATE oauth_access_tokens SET last_used = now()
		WHERE id = $1 AND (last_used IS NULL OR last_used < now() - $2::interval)`,
		t.ID, appLastUsedEvery.String()); err != nil {
		log.Printf("update app token last_used: %v", err)
	}

	out := &platform.AppToken{UserID: *t.UserID, ClientID: t.ClientID, Scopes: t.Scopes}
	if pseud.Valid {
		id := pseud.UUID
		out.PseudID = &id
	}
	return out, nil
}
