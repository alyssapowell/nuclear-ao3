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
	if got["id"] != id.String() || got["username"] != "alice" || got["email"] != "alice@example.org" {
		t.Errorf("wrong account: %v", got)
	}
	if _, leaked := got["password_hash"]; leaked {
		t.Error("password hash must never be returned")
	}
	if c := e.status("GET", "/api/v1/auth/me", "", nil); c != http.StatusUnauthorized {
		t.Errorf("me without a login: %d", c)
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
			codes <- e.status("POST", "/api/v1/auth/login", "", map[string]string{"login": "alice", "password": "secret1"})
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
