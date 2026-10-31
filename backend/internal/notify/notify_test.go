package notify

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/platform"
	"nuclear-ao3/internal/testdb"
)

type notifyEnv struct {
	t      *testing.T
	db     *sql.DB
	router *gin.Engine
	users  map[string]uuid.UUID
}

func newNotifyEnv(t *testing.T) *notifyEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testdb.New(t)
	e := &notifyEnv{t: t, db: db, users: map[string]uuid.UUID{}}
	deps := platform.NewDeps(db, func(tok string) (uuid.UUID, error) {
		if id, ok := e.users[tok]; ok {
			return id, nil
		}
		return uuid.Nil, sql.ErrNoRows
	})
	e.router = gin.New()
	Mount(e.router, New(db, nil), deps)
	return e
}

// user adds an account; its name is also its login token.
func (e *notifyEnv) user(name string, roles ...string) string {
	e.t.Helper()
	id := uuid.New()
	if _, err := e.db.Exec(`INSERT INTO users (id, username, email, password_hash, is_active, is_verified)
		VALUES ($1, $2, $3, 'x', true, true)`, id, name, name+"@example.org"); err != nil {
		e.t.Fatal(err)
	}
	for _, r := range roles {
		if _, err := e.db.Exec(`INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, id, r); err != nil {
			e.t.Fatal(err)
		}
	}
	e.users[name] = id
	return name
}

func (e *notifyEnv) notify(user, title string) uuid.UUID {
	e.t.Helper()
	id := uuid.New()
	if _, err := e.db.Exec(`INSERT INTO notification_items (id, user_id, event, source_id, source_type, title, description)
		VALUES ($1, $2, 'comment_received', $3, 'work', $4, 'described')`, id, e.users[user], uuid.New(), title); err != nil {
		e.t.Fatal(err)
	}
	return id
}

func (e *notifyEnv) do(method, path, token string, body any) (int, map[string]any) {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	var out map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func (e *notifyEnv) expect(want int, method, path, token string, body any) map[string]any {
	e.t.Helper()
	code, out := e.do(method, path, token, body)
	if code != want {
		e.t.Fatalf("%s %s as %q: status %d, want %d: %v", method, path, token, code, want, out)
	}
	return out
}

func TestNotificationsAreYourOwn(t *testing.T) {
	e := newNotifyEnv(t)
	alice := e.user("alice")
	bob := e.user("bob")
	a1 := e.notify("alice", "first")
	e.notify("alice", "second")
	b1 := e.notify("bob", "bobs")

	e.expect(401, "GET", "/api/v1/notifications", "", nil)
	list := e.expect(200, "GET", "/api/v1/notifications", alice, nil)["notifications"].([]any)
	if len(list) != 2 {
		t.Fatalf("alice sees %d notifications, want 2", len(list))
	}
	if c := e.expect(200, "GET", "/api/v1/notifications/unread-count", alice, nil)["count"]; c != 2.0 {
		t.Errorf("unread count %v, want 2", c)
	}

	// Marking read and deleting work on your own and are refused for others'
	e.expect(200, "PUT", "/api/v1/notifications/"+a1.String()+"/read", alice, nil)
	if c := e.expect(200, "GET", "/api/v1/notifications/unread-count", alice, nil)["count"]; c != 1.0 {
		t.Errorf("unread count after reading one: %v, want 1", c)
	}
	e.expect(400, "PUT", "/api/v1/notifications/not-an-id/read", alice, nil)
	e.expect(404, "DELETE", "/api/v1/notifications/"+b1.String(), alice, nil)
	var n int
	e.db.QueryRow(`SELECT count(*) FROM notification_items WHERE id = $1`, b1).Scan(&n)
	if n != 1 {
		t.Error("alice deleted bob's notification")
	}
	e.expect(200, "DELETE", "/api/v1/notifications/"+a1.String(), alice, nil)
	e.expect(200, "DELETE", "/api/v1/notifications/"+b1.String(), bob, nil)
}

func TestNotificationPreferences(t *testing.T) {
	e := newNotifyEnv(t)
	alice := e.user("alice")
	e.expect(401, "GET", "/api/v1/notification-preferences", "", nil)
	e.expect(200, "GET", "/api/v1/notification-preferences", alice, nil)
	// The old, unprefixed route is gone
	e.expect(404, "GET", "/api/v1/preferences", alice, nil)
}

func TestNotificationAdminRoutesAreForAdmins(t *testing.T) {
	e := newNotifyEnv(t)
	user := e.user("alice")
	admin := e.user("boss", "admin")
	for _, p := range []string{"/api/v1/admin/notifications/test", "/api/v1/admin/notifications/process-event"} {
		e.expect(401, "POST", p, "", map[string]any{})
		e.expect(403, "POST", p, user, map[string]any{})
	}
	e.expect(200, "POST", "/api/v1/admin/notifications/test", admin, map[string]any{})
	// Subscriptions live in the works package now
	e.expect(404, "GET", "/api/v1/subscriptions", user, nil)
}
