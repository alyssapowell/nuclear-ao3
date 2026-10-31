package auth

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"nuclear-ao3/internal/audit"
	"nuclear-ao3/internal/moderation"
)

func TestGrantAndRevokeRoleEndpoints(t *testing.T) {
	e := newSessionEnv(t)
	adminID := e.addUser("boss", "boss@example.org", "secret1")
	modID := e.addUser("mod", "mod@example.org", "secret1")
	userID := e.addUser("alice", "alice@example.org", "secret1")
	e.as.db.Exec(`INSERT INTO user_roles (user_id, role) VALUES ($1, 'admin')`, adminID)
	e.as.db.Exec(`INSERT INTO user_roles (user_id, role) VALUES ($1, 'moderator')`, modID)
	adminTok, modTok, userTok := e.login("boss", "secret1"), e.login("mod", "secret1"), e.login("alice", "secret1")
	rows := func() int {
		var n int
		e.as.db.QueryRow(`SELECT count(*) FROM admin_actions`).Scan(&n)
		return n
	}
	roleURL := "/api/v1/auth/admin/users/" + userID.String() + "/roles"

	// Only admins; everyone else is refused and nothing is written.
	for _, tok := range []string{userTok, modTok} {
		if w := e.do("POST", roleURL, tok, map[string]string{"role": "moderator"}); w.Code != http.StatusForbidden {
			t.Errorf("grant by non-admin: %d", w.Code)
		}
		if w := e.do("DELETE", roleURL+"/moderator", tok, nil); w.Code != http.StatusForbidden {
			t.Errorf("revoke by non-admin: %d", w.Code)
		}
	}
	if rows() != 0 {
		t.Fatalf("refused requests wrote %d rows", rows())
	}

	if w := e.do("POST", roleURL, adminTok, map[string]string{"role": "moderator"}); w.Code != http.StatusOK {
		t.Fatalf("grant: %d %s", w.Code, w.Body)
	}
	if rows() != 1 {
		t.Errorf("grant wrote %d rows, want 1", rows())
	}
	var held bool
	e.as.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role = 'moderator' AND revoked_at IS NULL)`, userID).Scan(&held)
	if !held {
		t.Error("role not granted")
	}
	if w := e.do("POST", roleURL, adminTok, map[string]string{"role": "moderator"}); w.Code != http.StatusConflict {
		t.Errorf("granting twice: %d", w.Code)
	}
	if w := e.do("POST", roleURL, adminTok, map[string]string{"role": "emperor"}); w.Code != http.StatusBadRequest {
		t.Errorf("unknown role: %d", w.Code)
	}
	// Admins can't grant themselves roles.
	if w := e.do("POST", "/api/v1/auth/admin/users/"+adminID.String()+"/roles", adminTok, map[string]string{"role": "tag_wrangler"}); w.Code != http.StatusForbidden {
		t.Errorf("self-grant: %d", w.Code)
	}
	if w := e.do("DELETE", roleURL+"/moderator", adminTok, nil); w.Code != http.StatusOK {
		t.Fatalf("revoke: %d %s", w.Code, w.Body)
	}
	if rows() != 2 {
		t.Errorf("after revoke there are %d rows, want 2", rows())
	}
	e.as.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role = 'moderator' AND revoked_at IS NULL)`, userID).Scan(&held)
	if held {
		t.Error("role still held")
	}
}

func TestBannedAndSuspendedUsersCanStillLogInButSessionsEnd(t *testing.T) {
	e := newSessionEnv(t)
	id := e.addUser("alice", "alice@example.org", "secret1")
	tok := e.login("alice", "secret1")
	svc := moderation.New(e.as.db)
	if err := svc.BanUser(nil2(), audit.CLIActor(), id, "test"); err != nil {
		t.Fatal(err)
	}
	if e.valid(tok) {
		t.Error("a ban should end the user's sessions")
	}
	// AO3 lets banned and suspended users log in; they just can't post.
	e.login("alice", "secret1")
	if err := svc.UnbanUser(nil2(), audit.CLIActor(), id, ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.SuspendUser(nil2(), audit.CLIActor(), id, time.Now().Add(time.Hour), "test"); err != nil {
		t.Fatal(err)
	}
	e.login("alice", "secret1")
	var n int
	e.as.db.QueryRow(`SELECT count(*) FROM admin_actions WHERE target_id = $1`, id.String()).Scan(&n)
	if n != 3 {
		t.Errorf("%d audit rows, want 3", n)
	}
	if w := e.do("PUT", "/api/v1/auth/me", e.login("alice", "secret1"), map[string]string{}); w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "account_suspended") {
		t.Errorf("profile edit while suspended: %d %s", w.Code, w.Body)
	}
}

func nil2() context.Context { return context.Background() }
