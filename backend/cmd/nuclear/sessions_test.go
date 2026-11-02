package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/google/uuid"

	"nuclear-ao3/internal/testdb"
)

// fakeHash is a stand-in for a stored token hash: 64 hex digits.
func fakeHash(id uuid.UUID) string {
	sum := sha256.Sum256([]byte(id.String()))
	return hex.EncodeToString(sum[:])
}

func TestRevokeSessions(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()

	mkUser := func(name string) uuid.UUID {
		id := uuid.New()
		if _, err := db.Exec(`INSERT INTO users (id, username, email, password_hash, is_active, is_verified)
			VALUES ($1, $2, $3, 'x', true, true)`, id, name, name+"@example.org"); err != nil {
			t.Fatal(err)
		}
		return id
	}
	client := uuid.New()
	if _, err := db.Exec(`INSERT INTO oauth_clients (client_id, client_name, redirect_uris, scopes)
		VALUES ($1, 'c', ARRAY['http://x/cb'], ARRAY['read'])`, client); err != nil {
		t.Fatal(err)
	}
	// Gives a user 2 sessions, an access token and a refresh token.
	give := func(uid uuid.UUID) {
		for i := 0; i < 2; i++ {
			if _, err := db.Exec(`INSERT INTO user_sessions (user_id, ip_address) VALUES ($1, '127.0.0.1')`, uid); err != nil {
				t.Fatal(err)
			}
		}
		at := uuid.New()
		if _, err := db.Exec(`INSERT INTO oauth_access_tokens (id, token_hash, user_id, client_id, scopes, expires_at, family_id)
			VALUES ($1, $2, $3, $4, ARRAY['read'], now() + interval '1 hour', $5)`, at, fakeHash(at), uid, client, uuid.New()); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO authorization_codes (code_hash, client_id, user_id, redirect_uri, scopes, code_challenge, code_challenge_method, expires_at)
			VALUES ($1, $2, $3, 'http://x/cb', ARRAY['read'], $4, 'S256', now() + interval '5 minutes')`, fakeHash(uuid.New()), client, uid, strings.Repeat("a", 43)); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO user_consents (id, user_id, client_id, scopes) VALUES ($1, $2, $3, ARRAY['read'])`,
			uuid.New(), uid, client); err != nil {
			t.Fatal(err)
		}
		rt := uuid.New()
		if _, err := db.Exec(`INSERT INTO oauth_refresh_tokens (id, token_hash, access_token_id, user_id, client_id, scopes, expires_at, family_id, family_created_at)
			VALUES ($1, $2, $3, $4, $5, ARRAY['read'], now() + interval '1 day', $6, now())`, rt, fakeHash(rt), at, uid, client, uuid.New()); err != nil {
			t.Fatal(err)
		}
	}
	live := func(uid uuid.UUID) (s, a, r int) {
		var c, k int
		db.QueryRow(`SELECT count(*) FROM authorization_codes WHERE user_id=$1 AND used_at IS NULL`, uid).Scan(&c)
		db.QueryRow(`SELECT count(*) FROM user_consents WHERE user_id=$1 AND NOT is_revoked`, uid).Scan(&k)
		db.QueryRow(`SELECT count(*) FROM user_sessions WHERE user_id=$1 AND is_active`, uid).Scan(&s)
		db.QueryRow(`SELECT count(*) FROM oauth_access_tokens WHERE user_id=$1 AND NOT is_revoked`, uid).Scan(&a)
		db.QueryRow(`SELECT count(*) FROM oauth_refresh_tokens WHERE user_id=$1 AND NOT is_revoked`, uid).Scan(&r)
		r += c + k // codes and consents count as live things too
		return
	}

	alice, bob := mkUser("alice"), mkUser("bob")
	give(alice)
	give(bob)

	if _, err := RevokeSessions(ctx, db, "nobody"); err == nil {
		t.Error("unknown username should be an error")
	}
	if s, _, _ := live(alice); s != 2 {
		t.Fatalf("a failed revoke changed data: alice has %d live sessions", s)
	}

	n, err := RevokeSessions(ctx, db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if n != (Revoked{2, 1, 1, 1, 1}) {
		t.Errorf("revoke alice = %+v", n)
	}
	if s, a, r := live(alice); s+a+r != 0 {
		t.Errorf("alice still has live %d/%d/%d", s, a, r)
	}
	if s, a, r := live(bob); s != 2 || a != 1 || r != 3 {
		t.Errorf("bob was touched: %d/%d/%d", s, a, r)
	}

	n, err = RevokeSessions(ctx, db, "")
	if err != nil {
		t.Fatal(err)
	}
	if n != (Revoked{2, 1, 1, 1, 1}) { // only bob's were still live
		t.Errorf("revoke all = %+v", n)
	}
	if s, a, r := live(bob); s+a+r != 0 {
		t.Errorf("bob still has live %d/%d/%d", s, a, r)
	}
	var revokedAt *string
	db.QueryRow(`SELECT revoked_at::text FROM oauth_access_tokens WHERE user_id=$1`, bob).Scan(&revokedAt)
	if revokedAt == nil {
		t.Error("revoked_at not set")
	}

	// The command wrapper returns exit codes.
	if code := runCommand(db, []string{"sessions", "revoke", "nobody"}); code != 1 {
		t.Errorf("exit code for unknown user = %d", code)
	}
	if code := runCommand(db, []string{"sessions", "revoke-all"}); code != 0 {
		t.Errorf("exit code for revoke-all = %d", code)
	}
}
