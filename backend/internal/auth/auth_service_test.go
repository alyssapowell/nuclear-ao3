package auth

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"
)

// Registering, logging in, sessions, passwords and roles are tested in
// passwords_test.go, sessions_test.go and roles_test.go. This file keeps the
// account details a user can read about themselves.

func TestGetProfileReturnsOwnAccountOnly(t *testing.T) {
	e := newSessionEnv(t)
	id := e.addUser("alice", "alice@example.org", "secret1")
	e.addUser("bob", "bob@example.org", "secret1")
	tok := e.login("alice", "secret1")

	w := e.do("GET", "/api/v1/auth/me", tok, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("me: %d %s", w.Code, w.Body)
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["id"] != id.String() || got["default_pseud"].(map[string]any)["name"] != "alice" || got["email"] != "alice@example.org" {
		t.Errorf("wrong account: %v", got)
	}
	if _, leaked := got["password_hash"]; leaked {
		t.Error("password hash must never be returned")
	}
	if c := e.status("GET", "/api/v1/auth/me", "", nil); c != http.StatusUnauthorized {
		t.Errorf("me without a login: %d", c)
	}
}

func TestMeListsOnlyTheCallersOwnRoles(t *testing.T) {
	e := newSessionEnv(t)
	id := e.addUser("alice", "alice@example.org", "secret1")
	bob := e.addUser("bob", "bob@example.org", "secret1")
	if _, err := e.as.db.Exec(`INSERT INTO user_roles (user_id, role) VALUES ($1, 'moderator'), ($1, 'tag_wrangler'), ($2, 'admin')`, id, bob); err != nil {
		t.Fatal(err)
	}
	if _, err := e.as.db.Exec(`INSERT INTO user_roles (user_id, role, revoked_at) VALUES ($1, 'admin', now())`, id); err != nil {
		t.Fatal(err)
	}
	w := e.do("GET", "/api/v1/auth/me", e.login("alice", "secret1"), nil)
	var got struct {
		Roles []string `json:"roles"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Roles) != 2 || got.Roles[0] != "moderator" || got.Roles[1] != "tag_wrangler" {
		t.Errorf("roles = %v, want moderator and tag_wrangler (not a revoked admin, not bob's)", got.Roles)
	}
}

func TestAdminUserListIsForAdminsOnly(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	tok := e.login("alice", "secret1")
	if c := e.status("GET", "/api/v1/auth/admin/users", tok, nil); c != http.StatusForbidden {
		t.Errorf("admin list as an ordinary user: %d", c)
	}
	if c := e.status("GET", "/api/v1/auth/admin/users", "", nil); c != http.StatusUnauthorized {
		t.Errorf("admin list without a login: %d", c)
	}
}

func TestConcurrentLogins(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	const n = 20
	codes := make(chan int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- e.status("POST", "/api/v1/auth/login", "", map[string]string{"email": "alice@example.org", "password": "secret1"})
		}()
	}
	wg.Wait()
	close(codes)
	for c := range codes {
		if c != http.StatusOK {
			t.Errorf("concurrent login: %d", c)
		}
	}
	var sessions int
	e.as.db.QueryRow(`SELECT count(*) FROM user_sessions WHERE is_active`).Scan(&sessions)
	if sessions != n {
		t.Errorf("%d sessions for %d logins", sessions, n)
	}
}
