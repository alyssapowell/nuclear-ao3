package tags

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/platform"
	"nuclear-ao3/internal/testdb"
)

type tagEnv struct {
	t      *testing.T
	db     *sql.DB
	router *gin.Engine
	users  map[string]uuid.UUID
}

func newTagEnv(t *testing.T) *tagEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testdb.New(t)
	e := &tagEnv{t: t, db: db, users: map[string]uuid.UUID{}}
	deps := platform.NewDeps(db, func(tok string) (uuid.UUID, error) {
		if id, ok := e.users[tok]; ok {
			return id, nil
		}
		return uuid.Nil, sql.ErrNoRows
	})
	e.router = gin.New()
	Mount(e.router, New(db), deps)
	return e
}

// user adds an account; its name is also its login token.
func (e *tagEnv) user(name string, roles ...string) string {
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

func (e *tagEnv) do(method, path, token string, body any) (int, map[string]any) {
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

func (e *tagEnv) expect(want int, method, path, token string, body any) map[string]any {
	e.t.Helper()
	code, out := e.do(method, path, token, body)
	if code != want {
		e.t.Fatalf("%s %s: status %d, want %d: %v", method, path, code, want, out)
	}
	return out
}

func TestCreateTag(t *testing.T) {
	e := newTagEnv(t)
	tok := e.user("tagger")

	e.expect(401, "POST", "/api/v1/tags", "", map[string]any{"name": "Fluff", "type": "freeform"})
	out := e.expect(201, "POST", "/api/v1/tags", tok, map[string]any{"name": "Fluff", "type": "freeform"})
	tag := out["tag"].(map[string]any)
	// A tag someone makes up is not canonical until a wrangler says so
	if tag["is_canonical"] != false || tag["name"] != "Fluff" {
		t.Errorf("new tag: %v", tag)
	}
	dup := e.expect(409, "POST", "/api/v1/tags", tok, map[string]any{"name": "Fluff", "type": "freeform"})
	if dup["existing_id"] != tag["id"] {
		t.Errorf("duplicate did not name the existing tag: %v", dup)
	}
	e.expect(400, "POST", "/api/v1/tags", tok, map[string]any{"name": "Odd", "type": "bogus"})
	// Anyone can start a new fandom (DECISIONS 2026-10-29)
	e.expect(201, "POST", "/api/v1/tags", tok, map[string]any{"name": "A Small New Fandom", "type": "fandom"})
}

func TestGetAndSearchTags(t *testing.T) {
	e := newTagEnv(t)
	tok := e.user("tagger")
	e.expect(201, "POST", "/api/v1/tags", tok, map[string]any{"name": "Fluffy Morning", "type": "freeform"})
	id := e.expect(201, "POST", "/api/v1/tags", tok, map[string]any{"name": "Angsty Night", "type": "freeform"})["tag"].(map[string]any)["id"].(string)

	got := e.expect(200, "GET", "/api/v1/tags/"+id, "", nil)
	if !strings.Contains(toJSON(got), "Angsty Night") {
		t.Errorf("get tag: %v", got)
	}
	e.expect(400, "GET", "/api/v1/tags/not-a-uuid", "", nil)
	e.expect(404, "GET", "/api/v1/tags/"+uuid.New().String(), "", nil)

	found := e.expect(200, "GET", "/api/v1/tags/search?q=fluff", "", nil)
	if list := found["tags"].([]any); len(list) != 1 {
		t.Errorf("search for fluff found %d tags, want 1", len(list))
	}
	// Nothing found is an empty list, not null
	none := e.expect(200, "GET", "/api/v1/tags/search?q=fluff&type=fandom", "", nil)
	if list, ok := none["tags"].([]any); !ok || len(list) != 0 {
		t.Errorf("empty search: %v", none["tags"])
	}
}

func TestAutocompleteOffersFilterableTagsOnly(t *testing.T) {
	e := newTagEnv(t)
	tok := e.user("tagger")
	e.expect(201, "POST", "/api/v1/tags", tok, map[string]any{"name": "Fluffy Morning", "type": "freeform"})
	e.expect(201, "POST", "/api/v1/tags", tok, map[string]any{"name": "Fluffy Night", "type": "freeform"})
	e.db.Exec(`UPDATE tags SET is_filterable = true WHERE name = 'Fluffy Morning'`)

	e.expect(400, "GET", "/api/v1/tags/autocomplete", "", nil)
	out := e.expect(200, "GET", "/api/v1/tags/autocomplete?q=fluffy", "", nil)
	list := out["suggestions"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["name"] != "Fluffy Morning" {
		t.Errorf("suggestions: %v", list)
	}
	empty := e.expect(200, "GET", "/api/v1/tags/autocomplete?q=zzz", "", nil)
	if l, ok := empty["suggestions"].([]any); !ok || len(l) != 0 {
		t.Errorf("no match should be an empty list: %v", empty["suggestions"])
	}
}

func TestWranglingIsForWranglers(t *testing.T) {
	e := newTagEnv(t)
	tok := e.user("tagger")
	wrangler := e.user("wrangler", "tag_wrangler")
	e.expect(401, "GET", "/api/v1/wrangling/queue", "", nil)
	e.expect(403, "GET", "/api/v1/wrangling/queue", tok, nil)
	e.expect(200, "GET", "/api/v1/wrangling/queue", wrangler, nil)
	e.expect(403, "GET", "/api/v1/admin/tags", wrangler, nil)
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
