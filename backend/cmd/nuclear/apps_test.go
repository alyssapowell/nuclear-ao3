package main

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"nuclear-ao3/internal/testdb"
	"nuclear-ao3/shared/email"
)

func TestAppsCommands(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	mail := &email.Config{FromName: "Test Archive", SiteURL: "https://archive.example.org"}

	owner := uuid.New()
	if _, err := db.Exec(`INSERT INTO users (id, username, email, password_hash, is_active, is_verified)
		VALUES ($1, 'appowner', 'appowner@example.org', 'x', true, true)`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE pseuds SET name = 'Quill' WHERE user_id = $1 AND is_default`, owner); err != nil {
		t.Fatal(err)
	}
	mk := func(name string) uuid.UUID {
		id := uuid.New()
		if _, err := db.Exec(`INSERT INTO oauth_clients (client_id, client_name, redirect_uris, scopes, owner_id, status)
			VALUES ($1, $2, ARRAY['https://app.example.org/cb'], ARRAY['read'], $3, 'pending')`, id, name, owner); err != nil {
			t.Fatal(err)
		}
		return id
	}
	status := func(id uuid.UUID) (s string, active bool) {
		if err := db.QueryRow(`SELECT status, is_active FROM oauth_clients WHERE client_id = $1`, id).Scan(&s, &active); err != nil {
			t.Fatal(err)
		}
		return
	}
	audited := func(action string, id uuid.UUID) int {
		var n int
		db.QueryRow(`SELECT count(*) FROM admin_actions WHERE action = $1 AND target_id = $2 AND actor_label LIKE 'cli:%' AND actor_user_id IS NULL`,
			action, id.String()).Scan(&n)
		return n
	}
	run := func(args ...string) (code int, out string) {
		out = capture(t, func() { code = appsCommand(ctx, db, mail, args) })
		return
	}

	a, b, c := mk("First app"), mk("Second app"), mk("Third app")

	// The queue lists waiting apps, by the owner's pseud and never the username.
	code, out := run()
	if code != 0 || !strings.Contains(out, "First app") || !strings.Contains(out, "Quill") || strings.Contains(out, "appowner") {
		t.Fatalf("list: code %d, output:\n%s", code, out)
	}

	revision := func(id uuid.UUID) string {
		var r int
		if err := db.QueryRow(`SELECT revision FROM oauth_clients WHERE client_id = $1`, id).Scan(&r); err != nil {
			t.Fatal(err)
		}
		return strconv.Itoa(r)
	}
	if !strings.Contains(out, "revision "+revision(a)) {
		t.Errorf("the listing should show each app's revision:\n%s", out)
	}
	// Approving needs the revision you looked at.
	if code, _ := run("approve", a.String()); code != 2 {
		t.Errorf("approve without --revision: %d, want 2", code)
	}
	stale := revision(a)
	if _, err := db.Exec(`UPDATE oauth_clients SET redirect_uris = ARRAY['https://elsewhere.example.org/cb'] WHERE client_id = $1`, a); err != nil {
		t.Fatal(err)
	}
	if code, _ := run("approve", a.String(), "--revision", stale); code != 1 {
		t.Errorf("approve with a stale revision: %d, want 1", code)
	}
	if s, _ := status(a); s == "approved" {
		t.Fatal("an app changed after it was looked at was approved")
	}
	if code, _ := run("approve", a.String(), "--revision", revision(a)); code != 0 {
		t.Fatalf("approve: %d", code)
	}
	if s, active := status(a); s != "approved" || !active || audited("oauth_client.approve", a) != 1 {
		t.Errorf("approve: status %s active %v audited %d", s, active, audited("oauth_client.approve", a))
	}
	if _, out := run("--status", "pending"); strings.Contains(out, "First app") || !strings.Contains(out, "Second app") {
		t.Errorf("the approved app should have left the queue:\n%s", out)
	}
	if _, out := run("--status", "all"); !strings.Contains(out, "First app") {
		t.Errorf("--status all:\n%s", out)
	}

	// Reject and suspend need a note.
	if code, _ := run("reject", b.String()); code == 0 {
		t.Error("reject without a note should fail")
	}
	if code, _ := run("reject", b.String(), "--note", "redirect is wrong"); code != 0 {
		t.Fatal("reject failed")
	}
	if s, _ := status(b); s != "rejected" || audited("oauth_client.reject", b) != 1 {
		t.Errorf("reject: %s", s)
	}

	// Suspend cuts off an approved app: its tokens, codes and consents end.
	reader := uuid.New()
	db.Exec(`INSERT INTO users (id, username, email, password_hash, is_active, is_verified) VALUES ($1, 'rdr', 'rdr@example.org', 'x', true, true)`, reader)
	tok := uuid.New()
	if _, err := db.Exec(`INSERT INTO oauth_access_tokens (id, token_hash, user_id, client_id, scopes, expires_at, family_id)
		VALUES ($1, $2, $3, $4, ARRAY['read'], now() + interval '1 hour', $5)`, tok, fakeHash(tok), reader, a, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if code, _ := run("suspend", a.String(), "--note=scraping the archive"); code != 0 {
		t.Fatal("suspend failed")
	}
	var live int
	db.QueryRow(`SELECT count(*) FROM oauth_access_tokens WHERE client_id = $1 AND NOT is_revoked`, a).Scan(&live)
	if s, active := status(a); s != "suspended" || active || live != 0 || audited("oauth_client.suspend", a) != 1 {
		t.Errorf("suspend: status %s active %v live tokens %d", s, active, live)
	}
	var queued int
	db.QueryRow(`SELECT count(*) FROM jobs WHERE kind = 'email' AND payload->>'to' = 'appowner@example.org'`).Scan(&queued)
	if queued != 3 { // approved, rejected, suspended
		t.Errorf("the owner should have 3 emails queued, has %d", queued)
	}

	// Mistakes are reported, not applied.
	if code, _ := run("approve", uuid.NewString()); code == 0 {
		t.Error("an unknown app should fail")
	}
	if code, _ := run("approve", "not-an-id"); code == 0 {
		t.Error("a bad id should fail")
	}
	if code, _ := run("approve", a.String(), "--revision", revision(a)); code == 0 {
		t.Error("a suspended app can't be approved from the command line")
	}
	if code, _ := run("delete", c.String()); code == 0 {
		t.Error("an unknown action should fail")
	}
	if s, _ := status(c); s != "pending" {
		t.Errorf("untouched app changed to %s", s)
	}
}
