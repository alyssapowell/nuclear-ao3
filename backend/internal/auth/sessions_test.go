package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nuclear-ao3/internal/testdb"
	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type sessionEnv struct {
	t      *testing.T
	as     *AuthService
	router *gin.Engine
}

func newSessionEnv(t *testing.T) *sessionEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testdb.New(t)
	as, err := New(db, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &sessionEnv{t: t, as: as, router: setupRouter(as)}
}

func (e *sessionEnv) addUser(username, email, password string) uuid.UUID {
	e.t.Helper()
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	id := uuid.New()
	_, err := e.as.db.Exec(`INSERT INTO users (id, username, email, password_hash, display_name, is_active, is_verified)
		VALUES ($1, $2, $3, $4, $5, true, true)`, id, username, email, string(hash), username)
	if err != nil {
		e.t.Fatal(err)
	}
	return id
}

func (e *sessionEnv) do(method, path, token string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "sessions-test")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func (e *sessionEnv) login(login, password string) string {
	e.t.Helper()
	w := e.do("POST", "/api/v1/auth/login", "", map[string]string{"login": login, "password": password})
	if w.Code != http.StatusOK {
		e.t.Fatalf("login %q: status %d: %s", login, w.Code, w.Body)
	}
	var resp models.AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		e.t.Fatal(err)
	}
	if resp.TokenType != "Bearer" || resp.AccessToken == "" || resp.ExpiresAt < time.Now().Add(13*24*time.Hour).Unix() {
		e.t.Fatalf("bad login response: %s", w.Body)
	}
	return resp.AccessToken
}

func (e *sessionEnv) valid(token string) bool {
	_, err := e.as.ValidateUserToken(token)
	return err == nil
}

func TestLoginByUsernameAndEmail(t *testing.T) {
	e := newSessionEnv(t)
	id := e.addUser("Alice", "Alice@example.org", "secret1")

	for _, login := range []string{"Alice", "alice", "ALICE@example.org", "alice@example.org"} {
		tok := e.login(login, "secret1")
		got, err := e.as.ValidateUserToken(tok)
		if err != nil || got != id {
			t.Errorf("login %q: token gave %v, %v", login, got, err)
		}
	}
	// The old "email" field still works as an alias.
	w := e.do("POST", "/api/v1/auth/login", "", map[string]string{"email": "alice@example.org", "password": "secret1"})
	if w.Code != http.StatusOK {
		t.Errorf("email alias: status %d", w.Code)
	}
	if bytes.Contains(w.Body.Bytes(), []byte("refresh_token")) {
		t.Error("response still has refresh_token")
	}
}

func TestLoginWrongCredentials(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	var bodies []string
	for _, login := range []string{"alice", "nobody"} {
		w := e.do("POST", "/api/v1/auth/login", "", map[string]string{"login": login, "password": "wrong!"})
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s: status %d", login, w.Code)
		}
		bodies = append(bodies, w.Body.String())
	}
	if bodies[0] != bodies[1] || !bytes.Contains([]byte(bodies[0]), []byte("invalid_credentials")) {
		t.Errorf("responses differ or wrong code: %v", bodies)
	}
}

func TestLogoutRejectsToken(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	tok := e.login("alice", "secret1")
	if !e.valid(tok) {
		t.Fatal("fresh token rejected")
	}
	if w := e.do("GET", "/api/v1/auth/me", tok, nil); w.Code != http.StatusOK {
		t.Fatalf("me: %d %s", w.Code, w.Body)
	}
	if w := e.do("POST", "/api/v1/auth/logout", tok, nil); w.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", w.Code)
	}
	if e.valid(tok) {
		t.Error("token still valid after logout")
	}
	if w := e.do("GET", "/api/v1/auth/me", tok, nil); w.Code != http.StatusUnauthorized {
		t.Errorf("me after logout: %d", w.Code)
	}
}

func TestInactiveUserRejected(t *testing.T) {
	e := newSessionEnv(t)
	id := e.addUser("alice", "alice@example.org", "secret1")
	tok := e.login("alice", "secret1")
	e.as.db.Exec(`UPDATE users SET is_active = false WHERE id = $1`, id)
	if e.valid(tok) {
		t.Error("token of deactivated user accepted")
	}
}

func TestSessionsListAndRevoke(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	e.addUser("bob", "bob@example.org", "secret2")
	a1 := e.login("alice", "secret1")
	a2 := e.login("alice", "secret1")
	b1 := e.login("bob", "secret2")

	list := func(tok string) []sessionInfo {
		w := e.do("GET", "/api/v1/auth/sessions", tok, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("sessions: %d %s", w.Code, w.Body)
		}
		var out []sessionInfo
		json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	sessions := list(a1)
	if len(sessions) != 2 {
		t.Fatalf("want 2 sessions, got %d", len(sessions))
	}
	var other, mine uuid.UUID
	for _, s := range sessions {
		if s.UserAgent != "sessions-test" || s.IPAddress == "" {
			t.Errorf("bad session info %+v", s)
		}
		if s.Current {
			mine = s.ID
		} else {
			other = s.ID
		}
	}
	if mine == uuid.Nil || other == uuid.Nil {
		t.Fatalf("current flag wrong: %+v", sessions)
	}
	if !sessions[0].CreatedAt.After(sessions[1].CreatedAt) {
		t.Error("not newest first")
	}

	// Bob cannot revoke Alice's session.
	if w := e.do("DELETE", "/api/v1/auth/sessions/"+other.String(), b1, nil); w.Code != http.StatusNotFound {
		t.Errorf("revoking someone else's: %d", w.Code)
	}
	if !e.valid(a2) || !e.valid(a1) {
		t.Fatal("alice's sessions died")
	}
	// Alice revokes her other session; only it dies.
	if w := e.do("DELETE", "/api/v1/auth/sessions/"+other.String(), a1, nil); w.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d", w.Code)
	}
	if !e.valid(a1) || !e.valid(b1) {
		t.Error("wrong session killed")
	}
	if e.valid(a2) {
		t.Error("revoked session still valid")
	}
	if len(list(a1)) != 1 {
		t.Error("revoked session still listed")
	}
	if w := e.do("DELETE", "/api/v1/auth/sessions/"+other.String(), a1, nil); w.Code != http.StatusNotFound {
		t.Errorf("revoking twice: %d", w.Code)
	}
}

func TestRevokeSessionsExcept(t *testing.T) {
	e := newSessionEnv(t)
	id := e.addUser("alice", "alice@example.org", "secret1")
	e.addUser("bob", "bob@example.org", "secret2")
	a1 := e.login("alice", "secret1")
	a2 := e.login("alice", "secret1")
	a3 := e.login("alice", "secret1")
	b1 := e.login("bob", "secret2")

	_, keep, err := e.as.validateSession(context.Background(), a2)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.as.revokeSessions(context.Background(), e.as.db, id, keep); err != nil {
		t.Fatal(err)
	}
	if e.valid(a1) || e.valid(a3) || !e.valid(a2) || !e.valid(b1) {
		t.Error("except did not keep exactly the given session")
	}
	if err := e.as.revokeSessions(context.Background(), e.as.db, id, uuid.Nil); err != nil {
		t.Fatal(err)
	}
	if e.valid(a2) || !e.valid(b1) {
		t.Error("uuid.Nil should revoke all of the user's sessions only")
	}
}

func TestTokenWithoutSessionRejected(t *testing.T) {
	e := newSessionEnv(t)
	id := e.addUser("alice", "alice@example.org", "secret1")
	old, err := e.as.jwt.GenerateToken(id, LoginAudience, []string{"user"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if e.valid(old) {
		t.Error("token with no sid accepted")
	}
	if w := e.do("GET", "/api/v1/auth/me", old, nil); w.Code != http.StatusUnauthorized {
		t.Errorf("middleware accepted token with no sid: %d", w.Code)
	}
	// A token naming a session that does not exist is rejected too.
	ghost, _ := e.as.jwt.GenerateLoginToken(id, uuid.New(), time.Hour)
	if e.valid(ghost) {
		t.Error("token with unknown session accepted")
	}
}

func TestNotImplementedStubs(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	tok := e.login("alice", "secret1")
	if w := e.do("PUT", "/api/v1/auth/me", tok, map[string]string{}); w.Code != http.StatusNotImplemented {
		t.Errorf("PUT /me: %d", w.Code)
	}
	for _, p := range []string{"/api/v1/auth/refresh", "/api/v1/auth/security-events"} {
		if w := e.do("GET", p, tok, nil); w.Code != http.StatusNotFound {
			t.Errorf("%s still routed: %d", p, w.Code)
		}
	}
}
