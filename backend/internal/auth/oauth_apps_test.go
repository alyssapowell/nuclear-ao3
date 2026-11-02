package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nuclear-ao3/shared/email"
	"nuclear-ao3/shared/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for app vetting (security audit, phase 3): people register apps, an
// admin approves them, and only approved apps work.

// asJSON sends a JSON request signed in as the person.
func (e *hEnv) asJSON(u *models.User, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		require.NoError(e.t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	require.NoError(e.t, signIn(e.as, req, u.ID))
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func (e *hEnv) body(w *httptest.ResponseRecorder) map[string]any {
	e.t.Helper()
	var m map[string]any
	require.NoError(e.t, json.Unmarshal(w.Body.Bytes(), &m), w.Body.String())
	return m
}

func (e *hEnv) status(id string) string {
	var s string
	require.NoError(e.t, e.db.QueryRow(`SELECT status FROM oauth_clients WHERE client_id = $1`, id).Scan(&s))
	return s
}

func appBody(n int) map[string]any {
	return map[string]any{
		"client_name": fmt.Sprintf("My app %d", n), "description": "reads fic",
		"redirect_uris": []string{fmt.Sprintf("https://mine%d.example.org/cb", n)},
		"scopes":        []string{"openid", "profile", "read"},
		"grant_types":   []string{"authorization_code", "refresh_token"},
	}
}

// selfApp registers an app as the person and returns it as an app value.
func (e *hEnv) selfApp(u *models.User, body map[string]any) app {
	e.t.Helper()
	w := e.asJSON(u, "POST", "/api/v1/apps", body)
	require.Equal(e.t, http.StatusCreated, w.Code, w.Body.String())
	b := e.body(w)
	secret, _ := b["client_secret"].(string)
	uris := body["redirect_uris"].([]string)
	return app{id: b["client_id"].(string), secret: secret, public: body["is_public"] == true, redirect: uris[0]}
}

func (e *hEnv) review(action, id string, note string) *httptest.ResponseRecorder {
	var body any
	if note != "" {
		body = map[string]string{"note": note}
	}
	return e.adminJSON("POST", "/api/v1/admin/apps/"+id+"/"+action, body)
}

func TestSelfRegisteredAppIsPendingUntilApproved(t *testing.T) {
	e := newHEnv(t)
	owner, reader := e.newUser("owner1"), e.newUser("reader1")
	a := e.selfApp(owner, appBody(1))
	assert.NotEmpty(t, a.secret, "a confidential app gets its secret once")
	assert.Equal(t, "pending", e.status(a.id))
	var active bool
	var by *uuid.UUID
	require.NoError(t, e.db.QueryRow(`SELECT is_active, owner_id FROM oauth_clients WHERE client_id = $1`, a.id).Scan(&active, &by))
	assert.False(t, active, "is_active follows the status")
	assert.Equal(t, owner.ID, *by)

	// Pending: it cannot start authorization.
	e.consent(reader, a, "profile")
	w := e.do("GET", a.authorizeURL("profile", ""), nil, e.asUser(reader.ID))
	assert.NotEqual(t, http.StatusFound, w.Code, w.Body.String())
	assert.NotContains(t, w.Header().Get("Location"), "code=")

	// It cannot exchange a code, refresh or have a token accepted either.
	// Take the tokens while it is approved, then flip the status by hand
	// (no revocation), so only the status check can be what stops them.
	require.Equal(t, http.StatusOK, e.review("approve", a.id, "").Code)
	code := e.code(a, reader, "openid profile read")
	tr := e.signedIn(a, reader, "profile")
	require.Equal(t, http.StatusOK, e.userinfoStatus(tr.AccessToken))
	_, err := e.db.Exec(`UPDATE oauth_clients SET status = 'pending' WHERE client_id = $1`, a.id)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))
	rw, _ := e.refresh(a, tr.RefreshToken, "")
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
	xw, _ := e.exchange(a, code)
	assert.Equal(t, http.StatusUnauthorized, xw.Code)
	w = e.do("GET", a.authorizeURL("profile", ""), nil, e.asUser(reader.ID))
	assert.NotContains(t, w.Header().Get("Location"), "code=")

	// Approved again: the same things work.
	_, err = e.db.Exec(`UPDATE oauth_clients SET status = 'approved' WHERE client_id = $1`, a.id)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, e.userinfoStatus(tr.AccessToken))
	xw, _ = e.exchange(a, code)
	assert.Equal(t, http.StatusOK, xw.Code, xw.Body.String())
	rw, _ = e.refresh(a, tr.RefreshToken, "")
	assert.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	e.signedIn(a, reader, "profile")
}

func TestSelfRegistrationRulesAreForced(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("forced1")

	bad := func(name string, mod func(map[string]any)) {
		b := appBody(1)
		mod(b)
		w := e.asJSON(u, "POST", "/api/v1/apps", b)
		assert.Equal(t, http.StatusBadRequest, w.Code, "%s: %s", name, w.Body.String())
	}
	bad("admin scope", func(b map[string]any) { b["scopes"] = []string{"read", "admin"} })
	bad("moderation scope", func(b map[string]any) { b["scopes"] = []string{"moderate"} })
	bad("unknown scope", func(b map[string]any) { b["scopes"] = []string{"nope"} })
	bad("client_credentials", func(b map[string]any) { b["grant_types"] = []string{"client_credentials"} })
	bad("password grant", func(b map[string]any) { b["grant_types"] = []string{"authorization_code", "password"} })
	bad("implicit response", func(b map[string]any) { b["response_types"] = []string{"token"} })
	bad("is_trusted", func(b map[string]any) { b["is_trusted"] = true })
	bad("is_first_party", func(b map[string]any) { b["is_first_party"] = true })
	bad("status", func(b map[string]any) { b["status"] = "approved" })
	bad("is_active", func(b map[string]any) { b["is_active"] = true })
	bad("plain http redirect", func(b map[string]any) { b["redirect_uris"] = []string{"http://evil.example.org/cb"} })
	bad("javascript redirect", func(b map[string]any) { b["redirect_uris"] = []string{"javascript:alert(1)"} })
	bad("custom scheme on a confidential app", func(b map[string]any) { b["redirect_uris"] = []string{"com.example.app://cb"} })
	bad("no redirect", func(b map[string]any) { b["redirect_uris"] = []string{} })
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_clients WHERE owner_id = $1`, u.ID), "nothing was created")

	// TTLs are clamped, not trusted.
	b := appBody(2)
	b["access_token_ttl"] = 999999
	b["refresh_token_ttl"] = 999999999
	w := e.asJSON(u, "POST", "/api/v1/apps", b)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var at, rt int
	var trusted, firstParty bool
	require.NoError(t, e.db.QueryRow(`SELECT access_token_ttl, refresh_token_ttl, is_trusted, is_first_party FROM oauth_clients WHERE client_id = $1`,
		e.body(w)["client_id"]).Scan(&at, &rt, &trusted, &firstParty))
	assert.Equal(t, accessTTLMax, at)
	assert.Equal(t, refreshTTLMax, rt)
	assert.False(t, trusted)
	assert.False(t, firstParty)

	// A public app is allowed, has no secret, and may use a reverse-DNS redirect.
	pb := appBody(3)
	pb["is_public"] = true
	pb["redirect_uris"] = []string{"com.example.reader://callback"}
	w = e.asJSON(u, "POST", "/api/v1/apps", pb)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.NotContains(t, e.body(w), "client_secret")
	assert.Equal(t, "pending", e.body(w)["status"])
}

func TestSelfRegistrationNeedsAGoodStandingLogin(t *testing.T) {
	e := newHEnv(t)
	w := e.do("POST", "/api/v1/apps", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	banned := e.newUser("banned1")
	_, err := e.db.Exec(`UPDATE users SET banned = true WHERE id = $1`, banned.ID)
	require.NoError(t, err)
	w = e.asJSON(banned, "POST", "/api/v1/apps", appBody(1))
	assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	suspended := e.newUser("suspended1")
	_, err = e.db.Exec(`UPDATE users SET suspended = true, suspended_until = now() + interval '1 day' WHERE id = $1`, suspended.ID)
	require.NoError(t, err)
	w = e.asJSON(suspended, "POST", "/api/v1/apps", appBody(1))
	assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_clients WHERE owner_id IN ($1, $2)`, banned.ID, suspended.ID))
}

func TestPerPersonAppLimits(t *testing.T) {
	e := newHEnv(t)
	u, other := e.newUser("limits1"), e.newUser("limits2")

	// Five waiting, and the sixth is refused.
	for i := 1; i <= maxPendingPerPerson; i++ {
		e.selfApp(u, appBody(i))
	}
	w := e.asJSON(u, "POST", "/api/v1/apps", appBody(6))
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "pending_limit", e.errCode(w))
	// The limit is per person.
	e.selfApp(other, appBody(7))

	// Approving some makes room for more waiting ones, up to ten in all.
	var ids []string
	rows, err := e.db.Query(`SELECT client_id FROM oauth_clients WHERE owner_id = $1`, u.ID)
	require.NoError(t, err)
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		require.Equal(t, http.StatusOK, e.review("approve", id, "").Code)
	}
	for i := 1; i <= maxPendingPerPerson; i++ {
		e.selfApp(u, appBody(20+i))
	}
	w = e.asJSON(u, "POST", "/api/v1/apps", appBody(30))
	assert.Equal(t, http.StatusConflict, w.Code)
	// Ten apps in all, of any status, is the cap: approve the waiting ones, then try.
	_, err = e.db.Exec(`UPDATE oauth_clients SET status = 'approved' WHERE owner_id = $1`, u.ID)
	require.NoError(t, err)
	w = e.asJSON(u, "POST", "/api/v1/apps", appBody(31))
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "app_limit", e.errCode(w))
	assert.Equal(t, maxAppsPerPerson, e.count(`SELECT count(*) FROM oauth_clients WHERE owner_id = $1`, u.ID))
}

func TestOwnersSeeOnlyTheirOwnAppsAndNeverTheSecret(t *testing.T) {
	e := newHEnv(t)
	alice, bob := e.newUser("alice_o"), e.newUser("bob_o")
	a := e.selfApp(alice, appBody(1))
	b := e.selfApp(bob, appBody(2))
	require.NotEmpty(t, a.secret)

	w := e.asJSON(alice, "GET", "/api/v1/apps", nil)
	require.Equal(t, http.StatusOK, w.Code)
	list := e.body(w)["apps"].([]any)
	require.Len(t, list, 1)
	got := list[0].(map[string]any)
	assert.Equal(t, a.id, got["client_id"])
	for _, k := range []string{"client_name", "status", "review_note", "redirect_uris", "scopes", "created_at", "reviewed_at"} {
		assert.Contains(t, got, k)
	}
	assert.NotContains(t, w.Body.String(), a.secret)
	assert.NotContains(t, w.Body.String(), b.id)
	assert.NotContains(t, strings.ToLower(w.Body.String()), "secret")

	// Another person's app is "not found" for every owner action.
	for _, c := range [][2]string{
		{"PATCH", "/api/v1/apps/" + b.id}, {"POST", "/api/v1/apps/" + b.id + "/reset-secret"}, {"DELETE", "/api/v1/apps/" + b.id},
	} {
		w := e.asJSON(alice, c[0], c[1], map[string]any{"client_name": "mine now"})
		assert.Equal(t, http.StatusNotFound, w.Code, "%s %s", c[0], c[1])
	}
	assert.Equal(t, "My app 2", e.nameOf(b.id))

	// The admin-only review routes are closed to owners, and owners cannot
	// change their own status.
	w = e.asJSON(alice, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"status": "approved"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "pending", e.status(a.id))
	w = e.asJSON(alice, "POST", "/api/v1/admin/apps/"+a.id+"/approve", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "pending", e.status(a.id))
}

func (e *hEnv) nameOf(id string) string {
	var n string
	require.NoError(e.t, e.db.QueryRow(`SELECT client_name FROM oauth_clients WHERE client_id = $1`, id).Scan(&n))
	return n
}

func TestEditingRedirectOrScopesSendsAnApprovedAppBackToReview(t *testing.T) {
	e := newHEnv(t)
	owner, reader := e.newUser("editor1"), e.newUser("editor_reader")

	setup := func() (app, models.TokenResponse) {
		b := appBody(int(uuid.New().ID() % 100000))
		a := e.selfApp(owner, b)
		require.Equal(t, http.StatusOK, e.review("approve", a.id, "").Code)
		_, err := e.db.Exec(`UPDATE oauth_clients SET is_trusted = true WHERE client_id = $1`, a.id)
		require.NoError(t, err)
		return a, e.signedIn(a, reader, "profile read")
	}

	// Changing only the look of the app keeps it approved and keeps tokens.
	a, tr := setup()
	w := e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{
		"client_name": "Renamed", "description": "new", "website": "https://site.example.org", "logo_url": "https://site.example.org/l.png"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "approved", e.status(a.id))
	assert.Equal(t, http.StatusOK, e.userinfoStatus(tr.AccessToken))
	// Saying the same redirect again is not a change.
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"redirect_uris": []string{a.redirect}, "scopes": []string{"read", "profile", "openid"}})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "approved", e.status(a.id))

	// A changed redirect: back to pending, and everything it held is ended.
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"redirect_uris": []string{"https://elsewhere.example.org/cb"}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "pending", e.body(w)["status"])
	assert.Equal(t, "pending", e.status(a.id))
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))
	rw, _ := e.refresh(a, tr.RefreshToken, "")
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM user_consents WHERE client_id = $1 AND NOT is_revoked`, a.id), "old consents must not carry over to the new address")
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_clients WHERE client_id = $1 AND is_trusted`, a.id), "trust was for the reviewed settings")

	// Changed scopes do the same; the owner cannot add an admin scope.
	a, tr = setup()
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"scopes": []string{"read", "admin"}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "approved", e.status(a.id))
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"scopes": []string{"read", "write", "profile", "openid"}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "pending", e.status(a.id))
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))

	// After approval it works again, with a fresh sign-in.
	require.Equal(t, http.StatusOK, e.review("approve", a.id, "").Code)
	e.signedIn(a, reader, "profile")

	// A rejected app that is edited goes back to the queue; a suspended one cannot be edited.
	require.Equal(t, http.StatusOK, e.review("suspend", a.id, "abuse").Code)
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"client_name": "x"})
	assert.Equal(t, http.StatusConflict, w.Code)
	rej := e.selfApp(owner, appBody(77))
	require.Equal(t, http.StatusOK, e.review("reject", rej.id, "fix the redirect").Code)
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+rej.id, map[string]any{"redirect_uris": []string{"https://fixed.example.org/cb"}})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pending", e.status(rej.id))
}

func TestOwnerDeleteRevokesEverything(t *testing.T) {
	e := newHEnv(t)
	owner, reader := e.newUser("deleter1"), e.newUser("deleter_reader")
	a := e.selfApp(owner, appBody(1))
	require.Equal(t, http.StatusOK, e.review("approve", a.id, "").Code)
	tr := e.signedIn(a, reader, "profile")
	code := e.code(a, reader, "profile")
	require.Equal(t, http.StatusOK, e.userinfoStatus(tr.AccessToken))

	w := e.asJSON(owner, "DELETE", "/api/v1/apps/"+a.id, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))
	xw, _ := e.exchange(a, code)
	assert.Equal(t, http.StatusUnauthorized, xw.Code)
	rw, _ := e.refresh(a, tr.RefreshToken, "")
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_clients WHERE client_id = $1`, a.id))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_access_tokens WHERE client_id = $1 AND NOT is_revoked`, a.id))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM user_consents WHERE client_id = $1`, a.id))
	// The slot is free again.
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_clients WHERE owner_id = $1`, owner.ID))
}

func TestOwnerResetsSecret(t *testing.T) {
	e := newHEnv(t)
	owner, reader := e.newUser("resetter1"), e.newUser("resetter_reader")
	a := e.selfApp(owner, appBody(1))
	require.Equal(t, http.StatusOK, e.review("approve", a.id, "").Code)
	tr := e.signedIn(a, reader, "profile")

	w := e.asJSON(owner, "POST", "/api/v1/apps/"+a.id+"/reset-secret", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	newSecret, _ := e.body(w)["client_secret"].(string)
	require.NotEmpty(t, newSecret)
	assert.NotEqual(t, a.secret, newSecret)
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken), "tokens from the old secret are ended")
	old, _ := e.refresh(a, tr.RefreshToken, "")
	assert.Equal(t, http.StatusUnauthorized, old.Code)

	pub := appBody(2)
	pub["is_public"] = true
	pub["redirect_uris"] = []string{"com.example.p://cb"}
	p := e.selfApp(owner, pub)
	w = e.asJSON(owner, "POST", "/api/v1/apps/"+p.id+"/reset-secret", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminQueueShowsTheOwnerAsAPseudOnly(t *testing.T) {
	e := newHEnv(t)
	owner := e.newUser("queue_owner")
	// The default pseud is renamed so the pen name differs from the username.
	_, err := e.db.Exec(`UPDATE pseuds SET name = 'Quill' WHERE user_id = $1 AND is_default`, owner.ID)
	require.NoError(t, err)
	a := e.selfApp(owner, appBody(1))
	e.selfApp(e.newUser("queue_other"), appBody(2))
	require.Equal(t, http.StatusOK, e.review("approve", e.selfApp(e.newUser("queue_third"), appBody(3)).id, "").Code)

	w := e.adminJSON("GET", "/api/v1/admin/apps?status=pending", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	apps := e.body(w)["apps"].([]any)
	require.Len(t, apps, 2, "only the waiting ones")

	var mine map[string]any
	for _, x := range apps {
		if x.(map[string]any)["client_id"] == a.id {
			mine = x.(map[string]any)
		}
	}
	require.NotNil(t, mine)
	var pseudID string
	require.NoError(t, e.db.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1 AND is_default`, owner.ID).Scan(&pseudID))
	assert.Equal(t, map[string]any{"id": pseudID, "name": "Quill"}, mine["owner"])
	for _, k := range []string{"redirect_uris", "scopes", "website", "client_name", "status"} {
		assert.Contains(t, mine, k)
	}
	raw := w.Body.String()
	assert.NotContains(t, raw, owner.ID.String(), "no account id")
	assert.NotContains(t, raw, "queue_owner", "no username")
	assert.NotContains(t, raw, owner.Email)
	for _, k := range []string{"owner_id", "user_id", "username", "email", "reviewed_by"} {
		assert.NotContains(t, raw, `"`+k+`"`)
	}

	assert.Equal(t, http.StatusBadRequest, e.adminJSON("GET", "/api/v1/admin/apps?status=bogus", nil).Code)
	all := e.body(e.adminJSON("GET", "/api/v1/admin/apps?status=all", nil))["apps"].([]any)
	assert.Len(t, all, 3)
	// Not for ordinary people.
	assert.Equal(t, http.StatusForbidden, e.asJSON(owner, "GET", "/api/v1/admin/apps", nil).Code)
	assert.Equal(t, http.StatusUnauthorized, e.do("GET", "/api/v1/admin/apps", nil, nil).Code)
}

func (e *hEnv) auditCount(action, target string) int {
	return e.count(`SELECT count(*) FROM admin_actions WHERE action = $1 AND target_type = 'oauth_client' AND target_id = $2`, action, target)
}

func TestAdminReviewActions(t *testing.T) {
	e := newHEnv(t)
	e.as.mail = &email.Config{FromName: "Test Archive", SiteURL: "https://archive.example.org"}
	owner, reader := e.newUser("review_owner"), e.newUser("review_reader")
	a := e.selfApp(owner, appBody(1))

	// Non-admins get 403 on every action; nothing changes.
	for _, act := range []string{"approve", "reject", "suspend", "reinstate"} {
		w := e.asJSON(owner, "POST", "/api/v1/admin/apps/"+a.id+"/"+act, map[string]string{"note": "x"})
		assert.Equal(t, http.StatusForbidden, w.Code, act)
		w = e.asJSON(reader, "POST", "/api/v1/admin/apps/"+a.id+"/"+act, map[string]string{"note": "x"})
		assert.Equal(t, http.StatusForbidden, w.Code, act)
	}
	assert.Equal(t, "pending", e.status(a.id))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM admin_actions WHERE target_id = $1`, a.id))

	// Reject needs a note; wrong states are refused; unknown ids are 404.
	assert.Equal(t, http.StatusBadRequest, e.review("reject", a.id, "").Code)
	assert.Equal(t, http.StatusConflict, e.review("reinstate", a.id, "").Code)
	assert.Equal(t, http.StatusNotFound, e.review("approve", uuid.NewString(), "").Code)
	assert.Equal(t, http.StatusBadRequest, e.review("reject", a.id, strings.Repeat("x", maxReviewNoteLen+1)).Code)

	// Reject: audited with the note, emailed, listed with the note to the owner.
	require.Equal(t, http.StatusOK, e.review("reject", a.id, "redirect looks wrong").Code)
	assert.Equal(t, "rejected", e.status(a.id))
	assert.Equal(t, 1, e.auditCount("oauth_client.reject", a.id))
	var reason string
	require.NoError(t, e.db.QueryRow(`SELECT reason FROM admin_actions WHERE action = 'oauth_client.reject' AND target_id = $1`, a.id).Scan(&reason))
	assert.Equal(t, "redirect looks wrong", reason)
	mine := e.body(e.asJSON(owner, "GET", "/api/v1/apps", nil))["apps"].([]any)[0].(map[string]any)
	assert.Equal(t, "rejected", mine["status"])
	assert.Equal(t, "redirect looks wrong", mine["review_note"])
	assert.NotNil(t, mine["reviewed_at"])
	assert.Equal(t, 1, e.count(`SELECT count(*) FROM jobs WHERE kind = 'email' AND payload->>'to' = $1 AND payload->>'subject' = 'Your app was rejected' AND payload->>'text' LIKE '%redirect looks wrong%'`, owner.Email))

	// Approve (from rejected), emailed.
	require.Equal(t, http.StatusOK, e.review("approve", a.id, "").Code)
	assert.Equal(t, "approved", e.status(a.id))
	assert.Equal(t, 1, e.auditCount("oauth_client.approve", a.id))
	assert.Equal(t, 1, e.count(`SELECT count(*) FROM jobs WHERE kind = 'email' AND payload->>'subject' = 'Your app was approved'`))
	var reviewedBy *uuid.UUID
	require.NoError(t, e.db.QueryRow(`SELECT reviewed_by FROM oauth_clients WHERE client_id = $1`, a.id).Scan(&reviewedBy))
	assert.NotNil(t, reviewedBy)

	// Suspend: tokens, codes and consents all end, and the owner is told.
	tr := e.signedIn(a, reader, "profile read")
	code := e.code(a, reader, "profile")
	require.Equal(t, http.StatusOK, e.userinfoStatus(tr.AccessToken))
	require.Equal(t, http.StatusOK, e.review("suspend", a.id, "scraping").Code)
	assert.Equal(t, "suspended", e.status(a.id))
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))
	rw, _ := e.refresh(a, tr.RefreshToken, "")
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
	xw, _ := e.exchange(a, code)
	assert.Equal(t, http.StatusUnauthorized, xw.Code)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_access_tokens WHERE client_id = $1 AND NOT is_revoked`, a.id))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_refresh_tokens WHERE client_id = $1 AND NOT is_revoked`, a.id))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM authorization_codes WHERE client_id = $1 AND used_at IS NULL`, a.id))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM user_consents WHERE client_id = $1 AND NOT is_revoked`, a.id))
	assert.Equal(t, 1, e.auditCount("oauth_client.suspend", a.id))
	assert.Equal(t, 1, e.count(`SELECT count(*) FROM jobs WHERE kind = 'email' AND payload->>'subject' = 'Your app was suspended'`))
	w := e.do("GET", a.authorizeURL("profile", ""), nil, e.asUser(reader.ID))
	assert.NotContains(t, w.Header().Get("Location"), "code=")

	// Reinstate works again (no email for that), and a suspend of a pending
	// app is allowed.
	emails := e.count(`SELECT count(*) FROM jobs WHERE kind = 'email'`)
	require.Equal(t, http.StatusOK, e.review("reinstate", a.id, "").Code)
	assert.Equal(t, "approved", e.status(a.id))
	assert.Equal(t, 1, e.auditCount("oauth_client.reinstate", a.id))
	assert.Equal(t, emails, e.count(`SELECT count(*) FROM jobs WHERE kind = 'email'`))
	e.signedIn(a, reader, "profile")
}

func TestNoEmailForAnAdminsOwnApp(t *testing.T) {
	e := newHEnv(t)
	e.as.mail = &email.Config{FromName: "Test Archive", SiteURL: "https://archive.example.org"}
	// An admin-created app is approved at once: the admin is the reviewer.
	a := e.newApp(appOpts{})
	assert.Equal(t, "approved", e.status(a.id))
	var active bool
	require.NoError(t, e.db.QueryRow(`SELECT is_active FROM oauth_clients WHERE client_id = $1`, a.id).Scan(&active))
	assert.True(t, active)
	u := e.newUser("admin_made_reader")
	e.signedIn(a, u, "profile")

	// Suspending the admin's own app from the admin's own login sends no email.
	require.Equal(t, http.StatusOK, e.review("suspend", a.id, "testing").Code)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM jobs WHERE kind = 'email'`))
	assert.Equal(t, 1, e.auditCount("oauth_client.suspend", a.id))
}

func TestOldAdminRoutesRespectStatus(t *testing.T) {
	e := newHEnv(t)
	owner := e.newUser("old_routes_owner")
	a := e.selfApp(owner, appBody(1))

	// Listed with its status, in every status.
	w := e.adminJSON("GET", "/api/v1/auth/admin/oauth/clients", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"pending"`)
	w = e.adminJSON("GET", "/api/v1/auth/admin/oauth/clients/"+a.id, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pending", e.body(w)["client"].(map[string]any)["status"])

	// Switching a waiting app "on" through is_active is not an approval.
	w = e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"is_active": true})
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Equal(t, "pending", e.status(a.id))
	// And the database refuses it, too.
	_, err := e.db.Exec(`UPDATE oauth_clients SET is_active = true WHERE client_id = $1`, a.id)
	assert.Error(t, err)

	// Off means suspended, on again means approved (a reinstatement).
	require.Equal(t, http.StatusOK, e.review("approve", a.id, "").Code)
	w = e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"is_active": false})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "suspended", e.status(a.id))
	w = e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"is_active": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "approved", e.status(a.id))

	// Trust stays an admin-only, audited setting.
	w = e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"is_trusted": true})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, e.auditCount("oauth_client.trust", a.id))

	// Admin delete leaves it unusable and not approved.
	require.Equal(t, http.StatusOK, e.adminJSON("DELETE", "/api/v1/auth/admin/oauth/clients/"+a.id, nil).Code)
	assert.Equal(t, "suspended", e.status(a.id))
}

func TestStatusAndIsActiveStayInStep(t *testing.T) {
	e := newHEnv(t)
	a := e.newApp(appOpts{})
	set := func(sql string) {
		_, err := e.db.Exec(sql, a.id)
		require.NoError(t, err)
	}
	state := func() (string, bool) {
		var s string
		var act bool
		require.NoError(t, e.db.QueryRow(`SELECT status, is_active FROM oauth_clients WHERE client_id = $1`, a.id).Scan(&s, &act))
		return s, act
	}
	for _, st := range appStatuses {
		set(`UPDATE oauth_clients SET status = '` + st + `' WHERE client_id = $1`)
		s, act := state()
		assert.Equal(t, st, s)
		assert.Equal(t, st == "approved", act, st)
	}
	set(`UPDATE oauth_clients SET status = 'approved' WHERE client_id = $1`)
	set(`UPDATE oauth_clients SET is_active = false WHERE client_id = $1`)
	s, act := state()
	assert.Equal(t, "suspended", s)
	assert.False(t, act)
	_, err := e.db.Exec(`UPDATE oauth_clients SET status = 'banana' WHERE client_id = $1`, a.id)
	assert.Error(t, err)

	// A row written the old way (no status) is approved, or suspended if inactive.
	id1, id2 := uuid.New(), uuid.New()
	_, err = e.db.Exec(`INSERT INTO oauth_clients (client_id, client_name, redirect_uris, scopes) VALUES ($1, 'old', '{https://a.example.org}', '{read}')`, id1)
	require.NoError(t, err)
	_, err = e.db.Exec(`INSERT INTO oauth_clients (client_id, client_name, redirect_uris, scopes, is_active) VALUES ($1, 'old off', '{https://a.example.org}', '{read}', false)`, id2)
	require.NoError(t, err)
	assert.Equal(t, "approved", e.status(id1.String()))
	assert.Equal(t, "suspended", e.status(id2.String()))
}
