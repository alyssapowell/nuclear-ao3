package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/auth"
	"nuclear-ao3/internal/demo"
	"nuclear-ao3/internal/export"
	"nuclear-ao3/internal/notify"
	"nuclear-ao3/internal/platform"
	"nuclear-ao3/internal/search"
	"nuclear-ao3/internal/tags"
	"nuclear-ao3/internal/testdb"
	"nuclear-ao3/internal/works"
)

const demoPassword = "a-real-demo-password"

func trialEnv(t *testing.T, password string) {
	t.Helper()
	t.Setenv("TRIAL_NO_EMAIL", "true")
	t.Setenv("DEMO_PASSWORD", password)
}

func rows(t *testing.T, db *sql.DB, q string, args ...any) []string {
	t.Helper()
	r, err := db.Query(q, args...)
	if err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
	defer r.Close()
	var out []string
	for r.Next() {
		var s string
		if err := r.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

func number(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
	return n
}

// "nuclear demo-data" does nothing, and says why, when the archive is not a
// trial or the password is weak.
func TestDemoDataRefusals(t *testing.T) {
	for _, tt := range []struct {
		name  string
		trial string
		pw    string
	}{
		{"not a trial", "", demoPassword},
		{"trial turned off", "false", demoPassword},
		{"no password", "true", ""},
		{"short password", "true", "short-pw"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := testdb.New(t)
			t.Setenv("TRIAL_NO_EMAIL", tt.trial)
			t.Setenv("DEMO_PASSWORD", tt.pw)
			before := number(t, db, `SELECT count(*) FROM users`)
			if code := runCommand(db, []string{"demo-data"}); code != 2 {
				t.Errorf("exit code %d, want 2", code)
			}
			if err := loadDemoData(context.Background(), db, os.Getenv); err == nil {
				t.Error("the startup option loaded demo data")
			}
			if n := number(t, db, `SELECT count(*) FROM users`); n != before {
				t.Errorf("%d accounts were made", n-before)
			}
			if n := number(t, db, `SELECT count(*) FROM works`); n != 0 {
				t.Errorf("%d works were made", n)
			}
		})
	}
}

// demoEnv is the real app on a database with the demo data loaded.
type demoEnv struct {
	db *sql.DB
	r  *gin.Engine
}

func newDemoEnv(t *testing.T) *demoEnv {
	t.Helper()
	db := testdb.New(t)
	trialEnv(t, demoPassword)
	if code := runCommand(db, []string{"demo-data"}); code != 0 {
		t.Fatalf("demo-data exited %d", code)
	}

	gin.SetMode(gin.TestMode)
	keyFile := filepath.Join(t.TempDir(), "signing.pem")
	if _, err := auth.GenerateKeyFile(keyFile); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SIGNING_KEY_FILE", keyFile)
	t.Setenv("SITE_URL", "https://archive.example.org")
	var authSvc *auth.AuthService
	authSvc, err := auth.New(db)
	if err != nil {
		t.Fatal(err)
	}
	worksSvc, err := works.New(db)
	if err != nil {
		t.Fatal(err)
	}
	deps := platform.NewDeps(db, func(token string) (uuid.UUID, error) { return authSvc.ValidateUserToken(token) })
	r := gin.New()
	r.Use(deps.AppAccess)
	mountRoutes(r, deps, services{
		auth: authSvc, works: worksSvc, tags: tags.New(db), search: search.New(db),
		notify: notify.New(db, nil), export: export.New(db),
	})
	return &demoEnv{db: db, r: r}
}

func (e *demoEnv) login(t *testing.T, email, password string) (int, string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	w := doBody(e.r, "POST", "/api/v1/auth/login", "", string(body))
	var out struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out.AccessToken
}

func (e *demoEnv) get(url, token string) (int, string) {
	w := doReq(e.r, "GET", url, token)
	return w.Code, w.Body.String()
}

func TestDemoDataLoadsOnceAndIsIdempotent(t *testing.T) {
	e := newDemoEnv(t)
	count := func() int {
		return number(t, e.db, `SELECT (SELECT count(*) FROM users) + (SELECT count(*) FROM works) + (SELECT count(*) FROM comments) + (SELECT count(*) FROM kudos) + (SELECT count(*) FROM tags)`)
	}
	before := count()
	for _, run := range []func() int{
		func() int { return runCommand(e.db, []string{"demo-data"}) },
		func() int {
			if err := loadDemoData(context.Background(), e.db, os.Getenv); err != nil {
				t.Log(err)
				return 1
			}
			return 0
		},
	} {
		if code := run(); code != 0 {
			t.Errorf("a second run failed: %d", code)
		}
	}
	if after := count(); after != before {
		t.Errorf("a second run changed the archive: %d rows, was %d", after, before)
	}
	if n := number(t, e.db, `SELECT count(*) FROM users WHERE email ~ '^demo[0-9]{2}@example\.org$'`); n != demo.AccountCount {
		t.Errorf("%d demo accounts, want %d", n, demo.AccountCount)
	}
}

func TestDemoAccountsLogInWithTheDemoPassword(t *testing.T) {
	e := newDemoEnv(t)
	for _, n := range []int{1, 7, demo.AccountCount} {
		email := demo.AccountEmail(n)
		code, token := e.login(t, email, demoPassword)
		if code != http.StatusOK || token == "" {
			t.Fatalf("%s could not log in: %d", email, code)
		}
		// Logged in for real: the token is good for the account's own pages.
		if code, body := e.get("/api/v1/auth/me", token); code != 200 || !strings.Contains(body, email) {
			t.Errorf("%s: /auth/me gave %d %s", email, code, short(body))
		}
		if code, _ := e.get("/api/v1/my/pseuds", token); code != 200 {
			t.Errorf("%s: /my/pseuds gave %d", email, code)
		}
	}
	if code, _ := e.login(t, demo.AccountEmail(1), "not-the-password"); code != http.StatusUnauthorized {
		t.Errorf("a wrong password gave %d", code)
	}

	// An author's own list has their works, drafts included.
	_, token := e.login(t, demo.AccountEmail(2), demoPassword)
	want := number(t, e.db, `SELECT count(*) FROM works w JOIN users u ON u.id = w.user_id WHERE u.email = $1`, demo.AccountEmail(2))
	code, body := e.get("/api/v1/my/works?limit=100", token)
	if code != 200 || want == 0 || strings.Count(body, `"title"`) < want {
		t.Errorf("/my/works gave %d with %d of %d titles", code, strings.Count(body, `"title"`), want)
	}

	// Nobody has a role, so the admin pages are closed to every one of them.
	for _, path := range []string{"/api/v1/admin/works", "/api/v1/auth/admin/users"} {
		if code, _ := e.get(path, token); code != http.StatusForbidden && code != http.StatusUnauthorized {
			t.Errorf("GET %s as a demo account gave %d", path, code)
		}
	}
	if n := number(t, e.db, `SELECT count(*) FROM user_roles r JOIN users u ON u.id = r.user_id WHERE u.email ~ '^demo' AND r.role <> 'user'`); n != 0 {
		t.Errorf("%d demo accounts hold a role", n)
	}
	if n := number(t, e.db, `SELECT count(*) FROM users WHERE email ~ '^demo' AND role <> 'user'`); n != 0 {
		t.Errorf("%d demo accounts have a staff role in users.role", n)
	}
}

func TestSearchFindsDemoWorks(t *testing.T) {
	e := newDemoEnv(t)
	var id, title string
	if err := e.db.QueryRow(`SELECT id, title FROM works WHERE NOT is_draft AND NOT restricted AND NOT is_anonymous
		AND NOT in_anon_collection AND NOT in_unrevealed_collection ORDER BY chapter_count DESC, id LIMIT 1`).Scan(&id, &title); err != nil {
		t.Fatal(err)
	}
	code, body := e.get("/api/v1/search/works?q="+url.QueryEscape(title), "")
	if code != 200 || !strings.Contains(body, id) {
		t.Errorf("searching for %q: %d, work %s not in %s", title, code, id, short(body))
	}
	// A fandom, through the tag filter.
	code, body = e.get("/api/v1/search/works?q="+url.QueryEscape("Sherlock Holmes"), "")
	if code != 200 || strings.Count(body, `"id"`) < 5 {
		t.Errorf("searching for a fandom: %d %s", code, short(body))
	}
	// A tag's own page lists works.
	var tagID string
	if err := e.db.QueryRow(`SELECT id FROM tags WHERE name = 'Pride and Prejudice - Jane Austen'`).Scan(&tagID); err != nil {
		t.Fatal(err)
	}
	if code, body := e.get("/api/v1/tags/"+tagID+"/works", ""); code != 200 || !strings.Contains(body, `"total"`) || strings.Contains(body, `"total":0`) {
		t.Errorf("tag page: %d %s", code, short(body))
	}
}

// A work that hides its creator (anonymous, or in an anonymous or unrevealed
// collection) must not show who wrote it anywhere the public can look, even
// though the loader writes the creatorship, as the app does.
func TestDemoAnonymousWorksDoNotRevealTheirCreator(t *testing.T) {
	e := newDemoEnv(t)
	_, reader := e.login(t, demo.AccountEmail(25), demoPassword)

	hidden := rows(t, e.db, `SELECT id::text FROM works WHERE (is_anonymous OR in_anon_collection OR in_unrevealed_collection) AND NOT is_draft AND NOT restricted ORDER BY id`)
	if len(hidden) < 10 {
		t.Fatalf("only %d works that hide their creator", len(hidden))
	}
	// Control: on an ordinary work the same pages do name the creator, so the
	// strings looked for below are the right ones and the pages carry them.
	var plain string
	var plainName string
	if err := e.db.QueryRow(`SELECT w.id::text, p.name FROM works w JOIN creatorships c ON c.creation_id = w.id JOIN pseuds p ON p.id = c.pseud_id
		WHERE NOT w.is_draft AND NOT w.restricted AND NOT w.is_anonymous AND NOT w.in_anon_collection AND NOT w.in_unrevealed_collection
		ORDER BY w.id LIMIT 1`).Scan(&plain, &plainName); err != nil {
		t.Fatal(err)
	}
	if _, body := e.get("/api/v1/works/"+plain, ""); !strings.Contains(body, plainName) {
		t.Fatalf("an ordinary work's page does not name its creator %q, so this test would find nothing: %s", plainName, short(body))
	}
	looked := 0
	for _, id := range hidden {
		// What would give the creator away: the pseud names, the account id and the email.
		secrets := rows(t, e.db, `SELECT p.name FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id WHERE c.creation_id = $1
			UNION SELECT p.id::text FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id WHERE c.creation_id = $1
			UNION SELECT u.id::text FROM works w JOIN users u ON u.id = w.user_id WHERE w.id = $1
			UNION SELECT u.email FROM works w JOIN users u ON u.id = w.user_id WHERE w.id = $1
			UNION SELECT u.id::text FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id JOIN users u ON u.id = p.user_id WHERE c.creation_id = $1`, id)
		if len(secrets) < 4 {
			t.Fatalf("work %s: only %d things to look for", id, len(secrets))
		}
		for _, token := range []string{"", reader} {
			who := "a visitor"
			if token != "" {
				who = "a member"
			}
			for _, path := range []string{"", "/chapters", "/comments", "/kudos", "/stats"} {
				code, body := e.get("/api/v1/works/"+id+path, token)
				if code != 200 {
					continue // not shown at all is fine
				}
				looked++
				for _, s := range secrets {
					if strings.Contains(body, s) {
						t.Errorf("GET /works/%s%s as %s names %q: %s", id, path, who, s, short(body))
					}
				}
			}
		}
	}

	if looked < len(hidden)*4 {
		t.Errorf("only %d pages of hidden works were read", looked)
	}

	// In lists, the work's own entry must not name the creator, and the
	// creator's pseud page must not list the work at all.
	rowsTo := rows(t, e.db, `SELECT w.id::text || '|' || p.id || '|' || p.name FROM works w JOIN creatorships c ON c.creation_id = w.id
		JOIN pseuds p ON p.id = c.pseud_id WHERE (w.is_anonymous OR w.in_anon_collection OR w.in_unrevealed_collection) AND NOT w.is_draft AND NOT w.restricted ORDER BY 1`)
	for _, row := range rowsTo {
		parts := strings.SplitN(row, "|", 3)
		workID, pseudID, pseudName := parts[0], parts[1], parts[2]
		for _, token := range []string{"", reader} {
			code, body := e.get("/api/v1/pseuds/"+pseudID+"/works?limit=100", token)
			if code == 200 && strings.Contains(body, workID) {
				t.Errorf("the pseud page of %q lists the hidden work %s", pseudName, workID)
			}
			code, body = e.get("/api/v1/pseuds/"+pseudID+"/bookmarks", token)
			if code == 200 && strings.Contains(body, workID) {
				t.Errorf("the bookmarks of %q list the hidden work %s", pseudName, workID)
			}
		}
	}
	for _, list := range []string{"/api/v1/works?limit=100", "/api/v1/search/works?limit=100&q=the", "/api/v1/collections"} {
		for _, row := range rowsTo {
			parts := strings.SplitN(row, "|", 3)
			if entry := entryOf(t, e, list, parts[0]); entry != "" && strings.Contains(entry, parts[2]) {
				t.Errorf("%s names %q on the hidden work %s: %s", list, parts[2], parts[0], short(entry))
			}
		}
	}
	for _, cname := range rows(t, e.db, `SELECT id::text FROM collections`) {
		for _, row := range rowsTo {
			parts := strings.SplitN(row, "|", 3)
			if entry := entryOf(t, e, "/api/v1/collections/"+cname+"/works", parts[0]); entry != "" && strings.Contains(entry, parts[2]) {
				t.Errorf("collection %s names %q on the hidden work %s", cname, parts[2], parts[0])
			}
		}
	}
}

// entryOf finds the object with this id in a JSON answer, as text.
func entryOf(t *testing.T, e *demoEnv, path, id string) string {
	t.Helper()
	code, body := e.get(path, "")
	if code != 200 {
		return ""
	}
	var v any
	if json.Unmarshal([]byte(body), &v) != nil {
		return ""
	}
	var found string
	var walk func(any)
	walk = func(x any) {
		switch x := x.(type) {
		case map[string]any:
			if x["id"] == id {
				b, _ := json.Marshal(x)
				found = string(b)
			}
			for _, c := range x {
				walk(c)
			}
		case []any:
			for _, c := range x {
				walk(c)
			}
		}
	}
	walk(v)
	return found
}

// The no-username guard (see TestNoAnswerHasAUsername) over the demo data:
// accounts have no username, so no public page or account page has a key for one.
func TestDemoAnswersHaveNoUsername(t *testing.T) {
	e := newDemoEnv(t)
	_, member := e.login(t, demo.AccountEmail(4), demoPassword) // an account with three pseuds

	one := func(q string) string { return rows(t, e.db, q)[0] }
	work := one(`SELECT id::text FROM works WHERE NOT is_draft AND NOT restricted AND chapter_count > 3 ORDER BY comment_count DESC, id LIMIT 1`)
	paths := []string{
		"/api/v1/works?limit=20", "/api/v1/works/" + work, "/api/v1/works/" + work + "/chapters", "/api/v1/works/" + work + "/comments",
		"/api/v1/works/" + work + "/kudos", "/api/v1/works/" + work + "/stats",
		"/api/v1/series", "/api/v1/series/" + one(`SELECT id::text FROM series ORDER BY id LIMIT 1`),
		"/api/v1/series/" + one(`SELECT id::text FROM series ORDER BY id LIMIT 1`) + "/works",
		"/api/v1/collections", "/api/v1/collections/" + one(`SELECT id::text FROM collections WHERE name = 'quiet-hours'`),
		"/api/v1/collections/" + one(`SELECT id::text FROM collections WHERE name = 'quiet-hours'`) + "/works",
		"/api/v1/pseuds/" + one(`SELECT id::text FROM pseuds ORDER BY id LIMIT 1`) + "/works",
		"/api/v1/pseuds/" + one(`SELECT id::text FROM pseuds ORDER BY id LIMIT 1`) + "/series",
		"/api/v1/pseuds/" + one(`SELECT id::text FROM pseuds ORDER BY id LIMIT 1`) + "/bookmarks",
		"/api/v1/search/works?q=tea", "/api/v1/tags/" + one(`SELECT id::text FROM tags WHERE type = 'fandom' ORDER BY use_count DESC LIMIT 1`) + "/works",
		"/api/v1/tags/search?q=a", "/api/v1/auth/me", "/api/v1/my/works", "/api/v1/my/pseuds", "/api/v1/my/comments", "/api/v1/my/series",
		"/api/v1/my/collections", "/api/v1/bookmarks", "/api/v1/subscriptions", "/api/v1/blocks", "/api/v1/mutes", "/api/v1/notifications",
	}
	checked := 0
	for _, path := range paths {
		for _, token := range []string{"", member} {
			code, body := e.get(path, token)
			var v any
			if code != 200 || json.Unmarshal([]byte(body), &v) != nil {
				continue
			}
			checked++
			if bad := usernameKeys(v, ""); len(bad) > 0 {
				t.Errorf("GET %s has username keys %v", path, bad)
			}
		}
	}
	if checked < len(paths) {
		t.Errorf("only %d answers walked of %d paths", checked, len(paths)*2)
	}
	t.Log(fmt.Sprintf("%d answers walked", checked))
}
