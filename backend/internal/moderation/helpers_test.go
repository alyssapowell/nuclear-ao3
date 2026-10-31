package moderation_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/moderation"
	"nuclear-ao3/internal/platform"
	"nuclear-ao3/internal/search"
	"nuclear-ao3/internal/tags"
	"nuclear-ao3/internal/testdb"
	"nuclear-ao3/internal/works"
)

// env is a real database with the works, tags, search and moderation routes
// mounted on it, and a way to act as different users.
type env struct {
	t      *testing.T
	db     *sql.DB
	router *gin.Engine
	tokens map[string]uuid.UUID
	n      int
}

func newEnv(t *testing.T) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testdb.New(t)
	e := &env{t: t, db: db, tokens: map[string]uuid.UUID{}}
	deps := platform.NewDeps(db, func(tok string) (uuid.UUID, error) {
		if id, ok := e.tokens[tok]; ok {
			return id, nil
		}
		return uuid.Nil, sql.ErrNoRows
	})
	ws, err := works.New(db)
	if err != nil {
		t.Fatal(err)
	}
	e.router = gin.New()
	works.Mount(e.router, ws, deps)
	tags.Mount(e.router, tags.New(db), deps)
	search.Mount(e.router, search.New(db), deps)
	moderation.Mount(e.router, moderation.New(db), deps)
	return e
}

func (e *env) exec(q string, args ...any) {
	e.t.Helper()
	if _, err := e.db.Exec(q, args...); err != nil {
		e.t.Fatalf("%s: %v", q, err)
	}
}

// user makes an account and returns its id and a login token.
func (e *env) user(name string, roles ...string) (uuid.UUID, string) {
	e.t.Helper()
	id := uuid.New()
	e.exec(`INSERT INTO users (id, username, email, password_hash, is_active, is_verified)
		VALUES ($1, $2, $3, 'x', true, true)`, id, name, name+"@example.org")
	// the account already has its default pseud, named after it
	for _, r := range roles {
		e.exec(`INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, id, r)
	}
	tok := "token-" + name
	e.tokens[tok] = id
	return id, tok
}

// work makes a posted work with one chapter, created by owner, tagged with
// the fandom tag.
func (e *env) work(owner uuid.UUID, title string) uuid.UUID {
	e.t.Helper()
	id := uuid.New()
	e.exec(`INSERT INTO works (id, user_id, title, summary, language, rating, status, is_draft, published_at, fandoms)
		VALUES ($1, $2, $3, 'a summary', 'en', 'general', 'published', false, now(), $4)`,
		id, owner, title, pq.Array([]string{"Test Fandom"}))
	e.exec(`INSERT INTO creatorships (creation_id, creation_type, pseud_id)
		SELECT $1, 'Work', id FROM pseuds WHERE user_id = $2 AND is_default`, id, owner)
	e.exec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, published_at, summary, notes, end_notes)
		VALUES ($1, $2, 1, 'one', 'chapter text', false, now(), '', '', '')`, uuid.New(), id)
	e.exec(`INSERT INTO tags (name, type) VALUES ('Test Fandom', 'fandom') ON CONFLICT (name) DO NOTHING`)
	e.exec(`INSERT INTO work_tags (work_id, tag_id) SELECT $1, id FROM tags WHERE name = 'Test Fandom'`, id)
	return id
}

func (e *env) tagID() string {
	var id string
	if err := e.db.QueryRow(`SELECT id FROM tags WHERE name = 'Test Fandom'`).Scan(&id); err != nil {
		e.t.Fatal(err)
	}
	return id
}

func (e *env) useCount() int {
	var n int
	if err := e.db.QueryRow(`SELECT use_count FROM tags WHERE name = 'Test Fandom'`).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

// do sends a request. token may be empty for a logged-out reader.
func (e *env) do(method, path, token string, body any) (int, string) {
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
	return w.Code, w.Body.String()
}

func (e *env) expect(want int, method, path, token string, body any) string {
	e.t.Helper()
	code, out := e.do(method, path, token, body)
	if code != want {
		e.t.Fatalf("%s %s as %q: status %d, want %d: %s", method, path, token, code, want, out)
	}
	return out
}

func (e *env) auditRows() int {
	var n int
	if err := e.db.QueryRow(`SELECT count(*) FROM admin_actions`).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

// shows reports whether the response mentions the id.
func shows(body string, id uuid.UUID) bool { return strings.Contains(body, id.String()) }

var _ = http.StatusOK
