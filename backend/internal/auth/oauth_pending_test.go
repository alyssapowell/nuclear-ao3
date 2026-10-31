package auth

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"nuclear-ao3/internal/testdb"
)

func TestOAuthPending(t *testing.T) {
	as, err := newTestAuthService(testdb.New(t))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := as.putPending(ctx, "consent:a", []byte(`{"n":1}`)); err != nil {
		t.Fatal(err)
	}
	// Writing again replaces the value.
	if err := as.putPending(ctx, "consent:a", []byte(`{"n":2}`)); err != nil {
		t.Fatal(err)
	}
	got, err := as.getPending(ctx, "consent:a")
	if err != nil || string(got) != `{"n": 2}` {
		t.Fatalf("get = %q, %v", got, err)
	}

	// Taking it removes it, and only one caller gets it.
	if _, err := as.takePending(ctx, "consent:a"); err != nil {
		t.Fatal(err)
	}
	if _, err := as.takePending(ctx, "consent:a"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("second take: %v", err)
	}
	if _, err := as.getPending(ctx, "consent:a"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("get after take: %v", err)
	}
}

func TestOAuthPendingIgnoresExpired(t *testing.T) {
	as, err := newTestAuthService(testdb.New(t))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := as.db.Exec(`INSERT INTO oauth_pending (key, value, expires_at) VALUES ('auth_req:old', '{}', now() - interval '1 minute')`); err != nil {
		t.Fatal(err)
	}
	if _, err := as.getPending(ctx, "auth_req:old"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("get expired: %v", err)
	}
	if _, err := as.takePending(ctx, "auth_req:old"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("take expired: %v", err)
	}
	// The next write cleans expired rows up.
	if err := as.putPending(ctx, "auth_req:new", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	var n int
	as.db.QueryRow(`SELECT count(*) FROM oauth_pending WHERE key = 'auth_req:old'`).Scan(&n)
	if n != 0 {
		t.Errorf("expired row still there")
	}
}
