package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"nuclear-ao3/internal/testdb"
)

// capture runs fn and returns what it printed to standard output.
func capture(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestModerationCommands(t *testing.T) {
	db := testdb.New(t)
	user := func(name string) uuid.UUID {
		id := uuid.New()
		if _, err := db.Exec(`INSERT INTO users (id, username, email, password_hash, is_active, is_verified)
			VALUES ($1, $2, $3, 'x', true, true)`, id, name, name+"@example.org"); err != nil {
			t.Fatal(err)
		}
		return id
	}
	owner, target := user("owner"), user("target")
	work := uuid.New()
	if _, err := db.Exec(`INSERT INTO works (id, user_id, title, language, rating, status, is_draft)
		VALUES ($1, $2, 'T', 'en', 'general', 'published', false)`, work, owner); err != nil {
		t.Fatal(err)
	}
	db.Exec(`INSERT INTO user_sessions (user_id, ip_address) VALUES ($1, '127.0.0.1')`, target)
	run := func(want int, args ...string) {
		t.Helper()
		if got := runCommand(db, args); got != want {
			t.Fatalf("nuclear %s: exit %d, want %d", strings.Join(args, " "), got, want)
		}
	}
	rows := func() (n int) { db.QueryRow(`SELECT count(*) FROM admin_actions`).Scan(&n); return }
	hidden := func() (h bool) { db.QueryRow(`SELECT hidden_by_admin FROM works WHERE id = $1`, work).Scan(&h); return }

	// hide needs a reason; unhide does not.
	run(1, "hide", "work", work.String())
	run(2, "hide", "work", "not-an-id", "--reason", "x")
	run(2, "hide", "gadget", work.String(), "--reason", "x")
	if rows() != 0 || hidden() {
		t.Fatal("a refused hide changed something")
	}
	run(0, "hide", "work", work.String(), "--reason", "spam ring")
	if !hidden() || rows() != 1 {
		t.Fatalf("hide: hidden=%v rows=%d", hidden(), rows())
	}
	run(0, "hide", "work", work.String(), "--reason=again") // already hidden: nothing to do
	if rows() != 1 {
		t.Errorf("hiding twice wrote a row")
	}
	run(0, "unhide", "work", work.String())
	if hidden() || rows() != 2 {
		t.Fatalf("unhide: hidden=%v rows=%d", hidden(), rows())
	}

	run(2, "suspend", "target", "--reason", "x") // no --until
	run(1, "suspend", "nobody", "--until", "2099-01-01", "--reason", "x")
	run(0, "suspend", "target", "--until", "2099-01-01", "--reason", "cooling off")
	var susp bool
	var live int
	db.QueryRow(`SELECT suspended FROM users WHERE id = $1`, target).Scan(&susp)
	db.QueryRow(`SELECT count(*) FROM user_sessions WHERE user_id = $1 AND is_active`, target).Scan(&live)
	if !susp || live != 0 {
		t.Errorf("suspend: suspended=%v live sessions=%d", susp, live)
	}
	run(0, "unsuspend", "target")
	run(1, "ban", "target") // reason required
	run(0, "ban", "target", "--reason", "harassment")
	var banned bool
	db.QueryRow(`SELECT banned FROM users WHERE id = $1`, target).Scan(&banned)
	if !banned {
		t.Error("ban did not ban")
	}
	run(0, "unban", "target")
	db.QueryRow(`SELECT banned FROM users WHERE id = $1`, target).Scan(&banned)
	if banned {
		t.Error("unban did not unban")
	}

	// grant is audited; granting twice is not.
	before := rows()
	run(0, "grant", "owner", "admin")
	run(0, "grant", "owner", "admin")
	run(1, "grant", "owner", "emperor")
	if rows() != before+1 {
		t.Errorf("grant wrote %d rows, want 1", rows()-before)
	}

	out := capture(t, func() { run(0, "roles") })
	if !strings.Contains(out, "owner") || !strings.Contains(out, "admin") || strings.Contains(out, "target") {
		t.Errorf("roles output:\n%s", out)
	}

	out = capture(t, func() { run(0, "audit", "--limit", "100") })
	for _, want := range []string{"cli:", "hide", "unhide", "suspend", "unsuspend", "ban", "unban", "grant_role", "spam ring", work.String()} {
		if !strings.Contains(out, want) {
			t.Errorf("audit output lacks %q:\n%s", want, out)
		}
	}
	out = capture(t, func() { run(0, "audit", "--since", "2999-01-01") })
	if !strings.Contains(out, "no admin actions") {
		t.Errorf("audit --since in the future:\n%s", out)
	}
	out = capture(t, func() { run(0, "audit", "--limit", "2") })
	if n := strings.Count(out, "cli:"); n != 2 {
		t.Errorf("audit --limit 2 printed %d rows", n)
	}
	run(2, "audit", "--limit", "zero")
}
