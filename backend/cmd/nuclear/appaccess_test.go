package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/auth"
	"nuclear-ao3/internal/notify"
	"nuclear-ao3/internal/platform"
	"nuclear-ao3/internal/search"
	"nuclear-ao3/internal/tags"
	"nuclear-ao3/internal/testdb"
	"nuclear-ao3/internal/works"
	"nuclear-ao3/shared/models"
)

// routeURL fills a route pattern's parameters with a real-looking id.
func routeURL(pattern string) string {
	parts := strings.Split(pattern, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") || strings.HasPrefix(p, "*") {
			parts[i] = uuid.NewString()
		}
	}
	return strings.Join(parts, "/")
}

func stubRouter(d *platform.Deps) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(d.AppAccess)
	mountRoutes(r, d, services{
		auth:   &auth.AuthService{},
		works:  &works.WorkService{},
		tags:   &tags.TagService{},
		notify: &notify.NotificationService{},
		search: &search.Service{},
	})
	return r
}

// The "forgetting is safe" test. It walks every route that is really mounted
// and checks, without a database, that the gate answers before any handler
// runs: a route that is not in the allowlist refuses an app that holds every
// scope an app can have, and a route that is in it refuses an app that lacks
// its scope.
func TestEveryRouteRefusesApps(t *testing.T) {
	user, client := uuid.New(), uuid.New()
	var all []string
	all = append(all, models.GrantableScopes()...)
	scopes := all
	d := platform.NewDeps(nil, func(string) (uuid.UUID, error) { return uuid.Nil, errors.New("no") },
		platform.WithAppTokens(func(context.Context, string) (*platform.AppToken, error) {
			return &platform.AppToken{UserID: user, ClientID: client, Scopes: scopes}, nil
		}, nil, 0))
	r := stubRouter(d)

	mounted := map[string]bool{}
	denied, listed := 0, 0
	for _, rt := range r.Routes() {
		key := rt.Method + " " + rt.Path
		mounted[key] = true
		if platform.AppSkipped(key) {
			continue
		}
		if need, ok := platform.AppRouteScope(key); ok {
			listed++
			// Holding a different scope is not enough.
			scopes = []string{"openid"}
			w := doReq(r, rt.Method, routeURL(rt.Path), "app")
			if w.Code != 403 || !strings.Contains(w.Header().Get("WWW-Authenticate"), `error="insufficient_scope"`) ||
				!strings.Contains(w.Header().Get("WWW-Authenticate"), need) {
				t.Errorf("%s without %q: %d %q", key, need, w.Code, w.Header().Get("WWW-Authenticate"))
			}
			scopes = all
			continue
		}
		denied++
		scopes = all
		w := doReq(r, rt.Method, routeURL(rt.Path), "app")
		if w.Code != 403 || !strings.Contains(w.Body.String(), "not_available_to_apps") {
			t.Errorf("%s: an app holding every scope got %d %s", key, w.Code, w.Body.String())
		}
	}
	// Nothing in the allowlist may name a route that does not exist: that is a typo.
	for route := range platform.AppRoutes() {
		if !mounted[route] {
			t.Errorf("allowlist names %s, which is not mounted", route)
		}
	}
	t.Logf("%d routes open to apps with a scope, %d refused", listed, denied)
	if listed == 0 || denied == 0 {
		t.Fatal("the walk saw no routes")
	}

	// The routes named in the task as never reachable.
	for _, key := range []string{
		"POST /api/v1/auth/login", "POST /api/v1/auth/logout", "GET /api/v1/auth/me", "PUT /api/v1/auth/me",
		"POST /api/v1/auth/change-password", "GET /api/v1/auth/sessions", "POST /api/v1/auth/register",
		"POST /api/v1/auth/reset-password", "POST /api/v1/auth/verify-email",
		"GET /api/v1/apps", "POST /api/v1/apps", "GET /api/v1/blocks", "POST /api/v1/mutes",
		"POST /api/v1/pseuds", "PUT /api/v1/pseuds/:pseud_id", "GET /api/v1/my/works", "GET /api/v1/my/pseuds",
		"GET /api/v1/my/comments", "GET /api/v1/admin/audit",
		"GET /api/v1/oauth/authorized-applications", "POST /auth/token", "GET /auth/authorize", "POST /auth/register-client",
		"GET /api/v1/wrangling/queue", "GET /api/v1/notifications", "GET /ws",
		"POST /api/v1/support", "GET /api/v1/admin/support", "POST /api/v1/admin/support/:ticket_id/notes",
	} {
		if !mounted[key] {
			t.Errorf("expected route %s is not mounted", key)
			continue
		}
		if _, ok := platform.AppRouteScope(key); ok {
			t.Errorf("%s must never be reachable by an app", key)
		}
	}
}

func doReq(r http.Handler, method, target, bearer string) *httptest.ResponseRecorder {
	return doBody(r, method, target, bearer, "")
}

func doBody(r http.Handler, method, target, bearer, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------
// With a database
// ---------------------------------------------------------------------

type appEnv struct {
	t      *testing.T
	db     *sql.DB
	r      *gin.Engine
	logins map[string]uuid.UUID

	alice, bob           uuid.UUID
	alicePseud, aliceAlt uuid.UUID // the default pseud, and the one shared with the app
	bobPseud             uuid.UUID
	client               uuid.UUID
	publicWork           uuid.UUID
	restrictedWork       uuid.UUID
	draftWork            uuid.UUID
}

func (e *appEnv) mustExec(q string, args ...any) {
	e.t.Helper()
	if _, err := e.db.Exec(q, args...); err != nil {
		e.t.Fatalf("%s: %v", q, err)
	}
}

func (e *appEnv) user(name string) (uuid.UUID, uuid.UUID) {
	id := uuid.New()
	e.mustExec(`INSERT INTO users (id, email, password_hash, is_active, is_verified)
		VALUES ($1, $2, 'x', true, true)`, id, name+"@example.org")
	// The pen name differs from the name in the email address, so an email
	// address in an answer is not mistaken for a pseud name.
	pseud := uuid.New()
	e.mustExec(`INSERT INTO pseuds (id, user_id, name, is_default) VALUES ($1, $2, $3, true)`, pseud, id, "Pen of "+name[:3])
	return id, pseud
}

func (e *appEnv) work(owner, pseud uuid.UUID, title string, restricted, draft bool) uuid.UUID {
	id := uuid.New()
	status := "published"
	if draft {
		status = "draft"
	}
	e.mustExec(`INSERT INTO works (id, title, summary, user_id, language, rating, status, is_draft, is_complete,
			published_at, restricted, created_at, updated_at)
		VALUES ($1, $2, 'A summary', $3, 'en', 'general', $4, $5, false, CASE WHEN $5 THEN NULL ELSE now() END, $6, now(), now())`,
		id, title, owner, status, draft, restricted)
	e.mustExec(`INSERT INTO creatorships (creation_id, creation_type, pseud_id, approved) VALUES ($1, 'Work', $2, true)`, id, pseud)
	return id
}

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// token makes an access token row and returns the raw token.
func (e *appEnv) token(user, client uuid.UUID, scopes string, expiry string) string {
	raw := "tok-" + uuid.NewString()
	e.mustExec(`INSERT INTO oauth_access_tokens (id, token_hash, user_id, client_id, scopes, expires_at, family_id)
		VALUES ($1, $2, $3, $4, string_to_array($5, ' '), now() + $6::interval, $7)`,
		uuid.New(), sha(raw), user, client, scopes, expiry, uuid.New())
	return raw
}

func (e *appEnv) newClient() uuid.UUID {
	id := uuid.New()
	e.mustExec(`INSERT INTO oauth_clients (client_id, client_name, redirect_uris, scopes, status)
		VALUES ($1, 'App', ARRAY['https://app.example.org/cb'], ARRAY['read'], 'approved')`, id)
	return id
}

func newAppEnv(t *testing.T, limit int) *appEnv {
	t.Helper()
	db := testdb.New(t)
	gin.SetMode(gin.TestMode)
	keyFile := filepath.Join(t.TempDir(), "signing.pem")
	if _, err := auth.GenerateKeyFile(keyFile); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SIGNING_KEY_FILE", keyFile)
	t.Setenv("SITE_URL", "https://archive.example.org")
	authSvc, err := auth.New(db)
	if err != nil {
		t.Fatal(err)
	}
	worksSvc, err := works.New(db)
	if err != nil {
		t.Fatal(err)
	}
	e := &appEnv{t: t, db: db, logins: map[string]uuid.UUID{}}
	d := platform.NewDeps(db,
		func(tok string) (uuid.UUID, error) {
			if id, ok := e.logins[tok]; ok {
				return id, nil
			}
			return uuid.Nil, errors.New("not a login")
		},
		platform.WithAppTokens(authSvc.ResolveAppToken, auth.NewLimiter().Allow, limit))
	r := gin.New()
	r.Use(d.AppAccess)
	mountRoutes(r, d, services{
		auth: authSvc, works: worksSvc, tags: tags.New(db), search: search.New(db),
		notify: notify.New(db, nil),
	})
	e.r = r

	e.alice, e.alicePseud = e.user("alice_acct")
	e.bob, e.bobPseud = e.user("bob_acct")
	e.mustExec(`INSERT INTO pseuds (id, user_id, name, is_default) VALUES ($1, $2, 'Alice Alt', false)`, uuid.New(), e.alice)
	if err := db.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1 AND name = 'Alice Alt'`, e.alice).Scan(&e.aliceAlt); err != nil {
		t.Fatal(err)
	}
	e.client = e.newClient()
	e.mustExec(`INSERT INTO user_consents (id, user_id, client_id, scopes, pseud_id) VALUES ($1, $2, $3, ARRAY['read'], $4)`,
		uuid.New(), e.alice, e.client, e.aliceAlt)

	e.publicWork = e.work(e.bob, e.bobPseud, "Public tale", false, false)
	e.restrictedWork = e.work(e.bob, e.bobPseud, "Members only tale", true, false)
	e.draftWork = e.work(e.alice, e.aliceAlt, "Alice private draft", false, true)
	e.logins["alice-login"] = e.alice
	return e
}

func (e *appEnv) get(path, bearer string) *httptest.ResponseRecorder {
	return doReq(e.r, "GET", path, bearer)
}

func TestAppTokenReadsAsThePerson(t *testing.T) {
	e := newAppEnv(t, 0)
	read := e.token(e.alice, e.client, "read", "1 hour")
	restricted := "/api/v1/works/" + e.restrictedWork.String()

	if w := e.get(restricted, ""); w.Code == 200 {
		t.Fatalf("a guest must not see a members-only work: %d", w.Code)
	}
	if w := e.get(restricted, read); w.Code != 200 {
		t.Fatalf("an app acting for a member sees members-only works: %d %s", w.Code, w.Body.String())
	}
	w := e.get("/api/v1/search/works?q=tale", read)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Members only tale") {
		t.Errorf("search as the member: %d %s", w.Code, w.Body.String())
	}
	if g := e.get("/api/v1/search/works?q=tale", ""); strings.Contains(g.Body.String(), "Members only tale") {
		t.Error("search as a guest showed a members-only work")
	}

	// Reading only: every write is refused with insufficient_scope.
	w = doBody(e.r, "POST", "/api/v1/works/"+e.publicWork.String()+"/comments", read, `{"content":"hello"}`)
	if w.Code != 403 {
		t.Fatalf("comment without the scope: %d", w.Code)
	}
	if h := w.Header().Get("WWW-Authenticate"); h != `Bearer error="insufficient_scope", scope="comments:write"` {
		t.Errorf("WWW-Authenticate = %q", h)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "insufficient_scope" {
		t.Errorf("body %v", body)
	}
	if w := doBody(e.r, "POST", "/api/v1/subscriptions", read, `{}`); w.Code != 403 {
		t.Errorf("subscribe without the scope: %d", w.Code)
	}
}

func TestAppCommentsUnderTheSharedPseud(t *testing.T) {
	e := newAppEnv(t, 0)
	tok := e.token(e.alice, e.client, "read comments:write", "1 hour")
	path := "/api/v1/works/" + e.publicWork.String() + "/comments"

	w := doBody(e.r, "POST", path, tok, `{"content":"lovely"}`)
	if w.Code != 201 {
		t.Fatalf("comment: %d %s", w.Code, w.Body.String())
	}
	var got string
	if err := e.db.QueryRow(`SELECT pseudonym_id::text FROM comments WHERE work_id = $1`, e.publicWork).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != e.aliceAlt.String() {
		t.Errorf("the comment is under %s, want the pseud shared with the app (%s)", got, e.aliceAlt)
	}

	// Asking for another pseud gets the same answer whether or not it is the
	// person's, so an app cannot probe which pseuds are theirs.
	mine := doBody(e.r, "POST", path, tok, `{"content":"x","pseudonym_id":"`+e.alicePseud.String()+`"}`)
	theirs := doBody(e.r, "POST", path, tok, `{"content":"x","pseudonym_id":"`+e.bobPseud.String()+`"}`)
	if mine.Code != 403 || theirs.Code != 403 || mine.Body.String() != theirs.Body.String() {
		t.Errorf("pseud probe: %d %s vs %d %s", mine.Code, mine.Body.String(), theirs.Code, theirs.Body.String())
	}
	// Asking for the shared pseud is fine.
	if w := doBody(e.r, "POST", path, tok, `{"content":"again","pseudonym_id":"`+e.aliceAlt.String()+`"}`); w.Code != 201 {
		t.Errorf("shared pseud: %d", w.Code)
	}
}

func TestAppPseudPageHidesOwnerView(t *testing.T) {
	e := newAppEnv(t, 0)
	tok := e.token(e.alice, e.client, "read", "1 hour")
	path := "/api/v1/pseuds/" + e.aliceAlt.String() + "/works"
	if w := doReq(e.r, "GET", path, "alice-login"); !strings.Contains(w.Body.String(), "Alice private draft") {
		t.Fatalf("the person's own browser should see their draft: %d %s", w.Code, w.Body.String())
	}
	w := e.get(path, tok)
	if w.Code != 200 || strings.Contains(w.Body.String(), "Alice private draft") {
		t.Errorf("an app must see the pseud's page as a stranger would: %d %s", w.Code, w.Body.String())
	}
}

func TestAppTokensThatMustBeRefused(t *testing.T) {
	e := newAppEnv(t, 0)
	work := "/api/v1/works/" + e.restrictedWork.String()
	// Subscriptions need a person, so a refused token shows as 401 there.
	probe := func(name, tok string) {
		t.Helper()
		if w := e.get("/api/v1/subscriptions", tok); w.Code != 401 {
			t.Errorf("%s: subscriptions gave %d", name, w.Code)
		}
		if w := e.get(work, tok); w.Code == 200 {
			t.Errorf("%s: a refused token read a members-only work", name)
		}
	}
	good := e.token(e.alice, e.client, "read", "1 hour")
	if w := e.get("/api/v1/subscriptions", good); w.Code != 200 {
		t.Fatalf("control: %d %s", w.Code, w.Body.String())
	}

	probe("expired", e.token(e.alice, e.client, "read", "-1 minute"))

	revoked := e.token(e.alice, e.client, "read", "1 hour")
	e.mustExec(`UPDATE oauth_access_tokens SET is_revoked = true, revoked_at = now() WHERE token_hash = $1`, sha(revoked))
	probe("revoked", revoked)

	other := e.newClient()
	suspended := e.token(e.alice, other, "read", "1 hour")
	e.mustExec(`UPDATE oauth_clients SET status = 'suspended' WHERE client_id = $1`, other)
	probe("suspended app", suspended)

	pending := e.newClient()
	pt := e.token(e.alice, pending, "read", "1 hour")
	e.mustExec(`UPDATE oauth_clients SET status = 'pending' WHERE client_id = $1`, pending)
	probe("unapproved app", pt)

	carol, _ := e.user("carol_acct")
	banned := e.token(carol, e.client, "read", "1 hour")
	e.mustExec(`UPDATE users SET banned = true WHERE id = $1`, carol)
	probe("banned person", banned)

	dave, _ := e.user("dave_acct")
	susp := e.token(dave, e.client, "read", "1 hour")
	e.mustExec(`UPDATE users SET suspended = true, suspended_until = now() + interval '1 day' WHERE id = $1`, dave)
	probe("suspended person", susp)

	erin, _ := e.user("erin_acct")
	staff := e.token(erin, e.client, "read", "1 hour")
	e.mustExec(`INSERT INTO user_roles (user_id, role) VALUES ($1, 'moderator')`, erin)
	probe("staff account", staff)

	// Revoking the person's approval of the app (the connected apps page)
	// ends its tokens as well.
	if w := e.get("/api/v1/subscriptions", good); w.Code != 200 {
		t.Fatalf("control again: %d", w.Code)
	}
	e.mustExec(`UPDATE oauth_access_tokens SET is_revoked = true WHERE user_id = $1`, e.alice)
	probe("after the person revoked it", good)

	// And a login token still reaches the same routes with no scope at all.
	if w := e.get("/api/v1/subscriptions", "alice-login"); w.Code != 200 {
		t.Errorf("login: %d", w.Code)
	}
	if w := e.get("/api/v1/my/works", "alice-login"); w.Code != 200 {
		t.Errorf("login on /my/works: %d", w.Code)
	}
}

func TestAppRateLimitIsPerAppAndPerson(t *testing.T) {
	e := newAppEnv(t, 3)
	a := e.token(e.alice, e.client, "read", "1 hour")
	a2 := e.token(e.alice, e.client, "read", "1 hour") // a second token, same (app, person)
	b := e.token(e.bob, e.client, "read", "1 hour")
	other := e.newClient()
	e.mustExec(`INSERT INTO user_consents (id, user_id, client_id, scopes) VALUES ($1, $2, $3, ARRAY['read'])`, uuid.New(), e.alice, other)
	c := e.token(e.alice, other, "read", "1 hour")

	for i := 0; i < 3; i++ {
		if w := e.get("/api/v1/tags", a); w.Code != 200 {
			t.Fatalf("request %d: %d", i+1, w.Code)
		}
	}
	w := e.get("/api/v1/tags", a)
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Errorf("past the limit: %d retry-after %q", w.Code, w.Header().Get("Retry-After"))
	}
	if w := e.get("/api/v1/tags", a2); w.Code != 429 {
		t.Errorf("a second token for the same app and person shares the count: %d", w.Code)
	}
	if w := e.get("/api/v1/tags", b); w.Code != 200 {
		t.Errorf("another person of the same app must not be limited: %d", w.Code)
	}
	if w := e.get("/api/v1/tags", c); w.Code != 200 {
		t.Errorf("another app for the same person must not be limited: %d", w.Code)
	}
	// Logged-in browsing has its own count.
	for i := 0; i < 5; i++ {
		if w := e.get("/api/v1/tags", "alice-login"); w.Code != 200 {
			t.Errorf("login request %d: %d", i+1, w.Code)
		}
	}
}

func TestAppTokenLastUsed(t *testing.T) {
	e := newAppEnv(t, 0)
	tok := e.token(e.alice, e.client, "read", "1 hour")
	last := func() sql.NullTime {
		var l sql.NullTime
		if err := e.db.QueryRow(`SELECT last_used FROM oauth_access_tokens WHERE token_hash = $1`, sha(tok)).Scan(&l); err != nil {
			t.Fatal(err)
		}
		return l
	}
	if last().Valid {
		t.Fatal("last_used starts empty")
	}
	e.get("/api/v1/tags", tok)
	first := last()
	if !first.Valid {
		t.Fatal("last_used not set by a request")
	}
	e.get("/api/v1/tags", tok)
	if !last().Time.Equal(first.Time) {
		t.Error("last_used was written twice within a minute")
	}
	e.mustExec(`UPDATE oauth_access_tokens SET last_used = now() - interval '2 minutes' WHERE token_hash = $1`, sha(tok))
	old := last()
	e.get("/api/v1/tags", tok)
	if !last().Time.After(old.Time) {
		t.Error("last_used was not refreshed after a minute")
	}
}

// Nothing an app can read may carry an account id or email address.
func TestAppAnswersHaveNoAccountDetails(t *testing.T) {
	e := newAppEnv(t, 1000)
	tok := e.token(e.alice, e.client, "read comments:write bookmarks:manage subscriptions:manage", "1 hour")

	// Give the person some activity first, as an app would.
	for _, x := range []*httptest.ResponseRecorder{
		doBody(e.r, "POST", "/api/v1/works/"+e.publicWork.String()+"/comments", tok, `{"content":"nice"}`),
		doBody(e.r, "POST", "/api/v1/works/"+e.publicWork.String()+"/bookmark", tok, `{"notes":"keeper"}`),
		doBody(e.r, "POST", "/api/v1/subscriptions", tok, `{"type":"author","target_id":"`+e.bobPseud.String()+`"}`),
	} {
		t.Logf("setup write -> %d %s", x.Code, x.Body.String())
	}

	secrets := []string{
		e.alice.String(), e.bob.String(), "alice_acct", "bob_acct", "alice_acct@example.org", "bob_acct@example.org",
	}
	w, cp := "/api/v1/works/"+e.publicWork.String(), "/api/v1/pseuds/"
	reads := []string{
		"/api/v1/works", w, w + "/chapters", w + "/comments",
		"/api/v1/work/" + e.publicWork.String(), "/api/v1/work/" + e.publicWork.String() + "/comments",
		"/api/v1/works/" + e.restrictedWork.String(),
		"/api/v1/search/works?q=tale", "/api/v1/search/suggestions?q=tal",
		"/api/v1/series", "/api/v1/collections", "/api/v1/tags", "/api/v1/tags/autocomplete?q=a",
		cp + e.bobPseud.String() + "/works", cp + e.bobPseud.String() + "/bookmarks", cp + e.bobPseud.String() + "/series",
		cp + e.aliceAlt.String() + "/works", cp + e.aliceAlt.String() + "/bookmarks",
		"/api/v1/subscriptions", "/api/v1/subscription-status?type=author&target_id=" + e.bobPseud.String(),
		"/api/v1/user/tags/followed",
	}
	checked := 0
	for _, p := range reads {
		resp := e.get(p, tok)
		if resp.Code >= 500 {
			t.Errorf("GET %s: %d %s", p, resp.Code, resp.Body.String())
		}
		checked++
		t.Logf("GET %s -> %d (%d bytes)", p, resp.Code, resp.Body.Len())
		for _, s := range secrets {
			if strings.Contains(resp.Body.String(), s) {
				t.Errorf("GET %s leaks %q: %s", p, s, resp.Body.String())
			}
		}
	}
	// And the answers to the writes.
	c := doBody(e.r, "POST", w+"/comments", tok, `{"content":"second"}`)
	if c.Code != 201 {
		t.Errorf("comment: %d %s", c.Code, c.Body.String())
	}
	for _, s := range secrets {
		if strings.Contains(c.Body.String(), s) {
			t.Errorf("the comment answer leaks %q: %s", s, c.Body.String())
		}
	}
	b := doBody(e.r, "POST", "/api/v1/works/"+e.restrictedWork.String()+"/bookmark", tok, `{}`)
	for _, s := range secrets {
		if strings.Contains(b.Body.String(), s) {
			t.Errorf("the bookmark answer leaks %q: %s", s, b.Body.String())
		}
	}
	t.Logf("checked %d reads", checked)
}
