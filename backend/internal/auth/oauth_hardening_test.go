package auth

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"nuclear-ao3/internal/testdb"
	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the OAuth provider hardening (security audit, phases 0 and 1).
// Each test says which item of the audit it covers.

type hEnv struct {
	t      *testing.T
	db     *sql.DB
	as     *AuthService
	router *gin.Engine
	seq    int
}

func newHEnv(t *testing.T) *hEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testdb.New(t)
	as, err := newTestAuthService(db)
	require.NoError(t, err)
	return &hEnv{t: t, db: db, as: as, router: setupRouter(as)}
}

// app is a registered OAuth app, with the secret it was given.
type app struct {
	id       string
	secret   string
	public   bool
	redirect string
}

type appOpts struct {
	public   bool
	scopes   []string
	grants   []string
	redirect string
}

func (e *hEnv) do(method, path string, form url.Values, mod func(*http.Request)) *httptest.ResponseRecorder {
	var body *strings.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if mod != nil {
		mod(req)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func (e *hEnv) asAdmin(req *http.Request) {
	require.NoError(e.t, signInAsAdmin(e.as, req))
}

func (e *hEnv) asUser(id uuid.UUID) func(*http.Request) {
	return func(req *http.Request) { require.NoError(e.t, signIn(e.as, req, id)) }
}

func (e *hEnv) adminJSON(method, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	e.asAdmin(req)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func (e *hEnv) register(body map[string]any) *httptest.ResponseRecorder {
	return e.adminJSON("POST", "/auth/register-client", body)
}

func (e *hEnv) newApp(o appOpts) app {
	e.t.Helper()
	e.seq++
	if o.scopes == nil {
		o.scopes = []string{"openid", "profile", "email", "read", "write"}
	}
	if o.grants == nil {
		o.grants = []string{"authorization_code", "refresh_token"}
	}
	if o.redirect == "" {
		o.redirect = fmt.Sprintf("https://app%d.example.org/callback", e.seq)
		if o.public {
			o.redirect = fmt.Sprintf("com.example.app%d://callback", e.seq)
		}
	}
	w := e.register(map[string]any{
		"client_name": fmt.Sprintf("App %d", e.seq), "redirect_uris": []string{o.redirect},
		"scopes": o.scopes, "grant_types": o.grants, "is_public": o.public,
	})
	require.Equal(e.t, http.StatusCreated, w.Code, w.Body.String())
	var resp models.ClientRegistrationResponse
	require.NoError(e.t, json.Unmarshal(w.Body.Bytes(), &resp))
	return app{id: resp.ClientID, secret: resp.ClientSecret, public: o.public, redirect: o.redirect}
}

func (e *hEnv) newUser(name string) *models.User {
	u, err := seedUser(e.db, name, name+"@nuclear-ao3.test", "password123")
	require.NoError(e.t, err)
	return u
}

func (e *hEnv) consent(u *models.User, a app, scopes ...string) {
	require.NoError(e.t, e.as.storeUserConsent(u.ID, uuid.MustParse(a.id), scopes))
}

// authorizeURL builds an authorization request with a valid PKCE challenge.
func (a app) authorizeURL(scope, extra string) string {
	return fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s",
		a.id, url.QueryEscape(a.redirect), url.QueryEscape(scope)) + pkceParams + extra
}

// code runs the authorization step for a person who already agreed.
func (e *hEnv) code(a app, u *models.User, scope string) string {
	e.t.Helper()
	e.consent(u, a, strings.Fields(scope)...)
	w := e.do("GET", a.authorizeURL(scope, ""), nil, e.asUser(u.ID))
	require.Equal(e.t, http.StatusFound, w.Code, w.Body.String())
	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(e.t, err)
	code := loc.Query().Get("code")
	require.NotEmpty(e.t, code, w.Header().Get("Location"))
	return code
}

// withClient adds the app's credentials to a token-style request (in the body).
func (a app) form(v url.Values) url.Values {
	v.Set("client_id", a.id)
	if a.secret != "" {
		v.Set("client_secret", a.secret)
	}
	return v
}

func (e *hEnv) tokenRequest(a app, v url.Values) (*httptest.ResponseRecorder, models.TokenResponse) {
	w := e.do("POST", "/auth/token", a.form(v), nil)
	var tr models.TokenResponse
	json.Unmarshal(w.Body.Bytes(), &tr)
	return w, tr
}

func (e *hEnv) exchange(a app, code string) (*httptest.ResponseRecorder, models.TokenResponse) {
	return e.tokenRequest(a, url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {a.redirect}, "code_verifier": {testVerifier},
	})
}

func (e *hEnv) refresh(a app, refreshToken string, scope string) (*httptest.ResponseRecorder, models.TokenResponse) {
	v := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}}
	if scope != "" {
		v.Set("scope", scope)
	}
	return e.tokenRequest(a, v)
}

// signedIn goes through the whole flow and returns the tokens.
func (e *hEnv) signedIn(a app, u *models.User, scope string) models.TokenResponse {
	e.t.Helper()
	w, tr := e.exchange(a, e.code(a, u, scope))
	require.Equal(e.t, http.StatusOK, w.Code, w.Body.String())
	return tr
}

func (e *hEnv) userinfoStatus(access string) int {
	w := e.do("GET", "/auth/userinfo", nil, func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+access) })
	return w.Code
}

func (e *hEnv) errCode(w *httptest.ResponseRecorder) string {
	var b struct {
		Error string `json:"error"`
	}
	json.Unmarshal(w.Body.Bytes(), &b)
	return b.Error
}

func (e *hEnv) count(query string, args ...any) int {
	var n int
	require.NoError(e.t, e.db.QueryRow(query, args...).Scan(&n))
	return n
}

// ---------------------------------------------------------------------
// Item 1: client_credentials is gone
// ---------------------------------------------------------------------

func TestClientCredentialsIsRefused(t *testing.T) {
	e := newHEnv(t)
	for _, grants := range [][]string{{"client_credentials"}, {"authorization_code", "client_credentials"}} {
		w := e.register(map[string]any{"client_name": "X", "redirect_uris": []string{"https://x.example.org/cb"},
			"scopes": []string{"read"}, "grant_types": grants})
		assert.Equal(t, http.StatusBadRequest, w.Code, "registering with %v", grants)
	}
	a := e.newApp(appOpts{})
	w, _ := e.tokenRequest(a, url.Values{"grant_type": {"client_credentials"}, "scope": {"read"}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "unsupported_grant_type", e.errCode(w))

	// Admin update cannot add it either.
	up := e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"grant_types": []string{"authorization_code", "client_credentials"}})
	assert.Equal(t, http.StatusBadRequest, up.Code)

	// The database refuses it, too.
	_, err := e.db.Exec(`UPDATE oauth_clients SET grant_types = ARRAY['client_credentials'] WHERE client_id = $1`, a.id)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------
// Item 2: PKCE, S256 only, for every client
// ---------------------------------------------------------------------

func TestPKCEIsRequiredAndS256Only(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("pkce_user")
	for _, public := range []bool{false, true} {
		a := e.newApp(appOpts{public: public})
		e.consent(u, a, "read")
		base := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=read", a.id, url.QueryEscape(a.redirect))

		cases := map[string]string{
			"no challenge":        "",
			"plain":               "&code_challenge=" + testVerifier + "&code_challenge_method=plain",
			"plain, no method":    "&code_challenge=" + testVerifier,
			"method but no value": "&code_challenge_method=S256",
			"short challenge":     "&code_challenge=abc&code_challenge_method=S256",
			"long challenge":      "&code_challenge=" + testChallenge + "A&code_challenge_method=S256",
			"bad characters":      "&code_challenge=" + strings.Repeat("+", 43) + "&code_challenge_method=S256",
			"lowercase method":    "&code_challenge=" + testChallenge + "&code_challenge_method=s256",
		}
		for name, extra := range cases {
			w := e.do("GET", base+extra, nil, e.asUser(u.ID))
			require.Equal(t, http.StatusFound, w.Code, "%s (public=%v)", name, public)
			loc := w.Header().Get("Location")
			assert.Contains(t, loc, "error=invalid_request", "%s (public=%v)", name, public)
			assert.NotContains(t, loc, "code=", "%s", name)
		}

		// S256 with a good challenge works end to end.
		tr := e.signedIn(a, u, "read")
		assert.NotEmpty(t, tr.AccessToken)
	}
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM authorization_codes WHERE code_challenge_method <> 'S256'`))
}

func TestPKCEVerifierRules(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("verifier_user")
	a := e.newApp(appOpts{})

	exchangeWith := func(verifier string) *httptest.ResponseRecorder {
		code := e.code(a, u, "read")
		v := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {a.redirect}}
		if verifier != "" {
			v.Set("code_verifier", verifier)
		}
		w, _ := e.tokenRequest(a, v)
		return w
	}
	for name, v := range map[string]string{
		"missing":      "",
		"42 chars":     testVerifier[:42],
		"129 chars":    strings.Repeat("a", 129),
		"wrong value":  strings.Repeat("a", 43),
		"bad alphabet": strings.Repeat("a", 42) + "+",
	} {
		w := exchangeWith(v)
		assert.Equal(t, http.StatusBadRequest, w.Code, name)
		assert.Equal(t, "invalid_grant", e.errCode(w), name)
	}
	// 43 and 128 characters are fine when they match the challenge; here we
	// check the length rule on its own.
	assert.True(t, validPKCEVerifier(strings.Repeat("a", 43)))
	assert.True(t, validPKCEVerifier(strings.Repeat("a", 128)))
	assert.False(t, validPKCEVerifier(strings.Repeat("a", 129)))

	// The right verifier works.
	w := exchangeWith(testVerifier)
	assert.Equal(t, http.StatusOK, w.Code)

	// "plain" is not a method the checker knows.
	assert.False(t, e.as.validatePKCE(testVerifier, "plain", testVerifier))
	assert.True(t, e.as.validatePKCE(testChallenge, "S256", testVerifier))
}

// ---------------------------------------------------------------------
// Item 3: admin-only scopes are never granted
// ---------------------------------------------------------------------

func TestAdminScopesAreNeverGranted(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("scope_user")

	for _, scope := range []string{"admin", "tags:wrangle"} {
		// Registration
		w := e.register(map[string]any{"client_name": "X", "redirect_uris": []string{"https://x.example.org/cb"},
			"scopes": []string{"read", scope}})
		assert.Equal(t, http.StatusBadRequest, w.Code, "register with %s", scope)

		// Admin update
		a := e.newApp(appOpts{})
		up := e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"scopes": []string{"read", scope}})
		assert.Equal(t, http.StatusBadRequest, up.Code, "update with %s", scope)

		// Authorize, even for an app that somehow has the scope on record
		// (a row from before the migration).
		_, err := e.db.Exec(`UPDATE oauth_clients SET scopes = scopes || $2::text WHERE client_id = $1`, a.id, scope)
		require.NoError(t, err)
		e.consent(u, a, "read")
		_, err = e.db.Exec(`UPDATE user_consents SET scopes = scopes || $2::text WHERE client_id = $1`, a.id, scope)
		require.NoError(t, err)
		aw := e.do("GET", a.authorizeURL("read "+scope, ""), nil, e.asUser(u.ID))
		assert.Equal(t, http.StatusFound, aw.Code)
		assert.Contains(t, aw.Header().Get("Location"), "error=invalid_scope", "authorize with %s", scope)

		// Token: a code that already carries the scope is refused.
		code := e.code(a, u, "read")
		_, err = e.db.Exec(`UPDATE authorization_codes SET scopes = scopes || $1::text WHERE code_hash = $2`, scope, hashSecret(code))
		require.NoError(t, err)
		tw, _ := e.exchange(a, code)
		assert.Equal(t, http.StatusBadRequest, tw.Code, "token with %s", scope)
		assert.Equal(t, "invalid_scope", e.errCode(tw))

		// Refresh: asking for it, and a stored token that holds it.
		tr := e.signedIn(a, u, "read")
		rw, _ := e.refresh(a, tr.RefreshToken, "read "+scope)
		assert.Equal(t, http.StatusBadRequest, rw.Code, "refresh asking for %s", scope)
		tr = e.signedIn(a, u, "read")
		_, err = e.db.Exec(`UPDATE oauth_refresh_tokens SET scopes = scopes || $1::text WHERE token_hash = $2`, scope, hashSecret(tr.RefreshToken))
		require.NoError(t, err)
		rw, _ = e.refresh(a, tr.RefreshToken, "")
		assert.Equal(t, http.StatusBadRequest, rw.Code, "refresh holding %s", scope)
		assert.Equal(t, "invalid_scope", e.errCode(rw))
	}
}

// The code reads models.AO3OAuthScopes; the oauth_scopes table must say the
// same, or the two lists have drifted apart.
func TestScopeListsAgree(t *testing.T) {
	e := newHEnv(t)
	rows, err := e.db.Query(`SELECT name, is_admin_only FROM oauth_scopes`)
	require.NoError(t, err)
	defer rows.Close()
	seen := 0
	for rows.Next() {
		var name string
		var adminOnly bool
		require.NoError(t, rows.Scan(&name, &adminOnly))
		s, ok := models.AO3OAuthScopes[name]
		require.True(t, ok, "scope %s is in the table but not in the code", name)
		assert.Equal(t, s.IsAdminOnly, adminOnly, name)
		assert.Equal(t, adminOnly, models.IsAdminOnlyScope(name), name)
		seen++
	}
	assert.Equal(t, len(models.AO3OAuthScopes), seen)
	assert.NotContains(t, models.GrantableScopes(), "admin")
	assert.NotContains(t, models.GrantableScopes(), "tags:wrangle")
}

// ---------------------------------------------------------------------
// Item 4: grant and response type allowlists
// ---------------------------------------------------------------------

func TestGrantAndResponseTypeAllowlists(t *testing.T) {
	e := newHEnv(t)
	base := func() map[string]any {
		return map[string]any{"client_name": "X", "redirect_uris": []string{"https://x.example.org/cb"}, "scopes": []string{"read"}}
	}
	for name, mod := range map[string]map[string]any{
		"password grant":      {"grant_types": []string{"password"}},
		"implicit grant":      {"grant_types": []string{"implicit"}},
		"token response type": {"response_types": []string{"token"}},
		"id_token response":   {"response_types": []string{"code id_token"}},
		"refresh only":        {"grant_types": []string{"refresh_token"}},
	} {
		b := base()
		for k, v := range mod {
			b[k] = v
		}
		assert.Equal(t, http.StatusBadRequest, e.register(b).Code, name)
	}
	a := e.newApp(appOpts{})
	for name, body := range map[string]map[string]any{
		"update grant":    {"grant_types": []string{"implicit"}},
		"update response": {"response_types": []string{"token"}},
	} {
		assert.Equal(t, http.StatusBadRequest, e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, body).Code, name)
	}

	// A client without refresh_token gets no refresh token, and cannot use the grant.
	u := e.newUser("grant_user")
	noRefresh := e.newApp(appOpts{grants: []string{"authorization_code"}})
	tr := e.signedIn(noRefresh, u, "read")
	assert.NotEmpty(t, tr.AccessToken)
	assert.Empty(t, tr.RefreshToken)
	w, _ := e.refresh(noRefresh, "anything", "")
	assert.Equal(t, "unauthorized_client", e.errCode(w))

	// And a client without authorization_code cannot use that grant even
	// if the row somehow lists it differently: the grant must be in GrantTypes.
	_, err := e.db.Exec(`UPDATE oauth_clients SET grant_types = ARRAY['refresh_token'] WHERE client_id = $1`, noRefresh.id)
	require.NoError(t, err)
	w, _ = e.exchange(noRefresh, "x")
	assert.Equal(t, "unauthorized_client", e.errCode(w))
}

// ---------------------------------------------------------------------
// Item 5: lifetimes are clamped in code
// ---------------------------------------------------------------------

func TestLifetimesAreClamped(t *testing.T) {
	assert.Equal(t, 3600, clampAccessTTL(0))
	assert.Equal(t, 3600, clampAccessTTL(86400))
	assert.Equal(t, 300, clampAccessTTL(1))
	assert.Equal(t, 1800, clampAccessTTL(1800))
	assert.Equal(t, 30*24*3600, clampRefreshTTL(0))
	assert.Equal(t, 30*24*3600, clampRefreshTTL(10*365*24*3600))
	assert.Equal(t, 3600, clampRefreshTTL(5))

	e := newHEnv(t)
	u := e.newUser("ttl_user")
	w := e.register(map[string]any{"client_name": "Long", "redirect_uris": []string{"https://l.example.org/cb"},
		"scopes": []string{"read"}, "grant_types": []string{"authorization_code", "refresh_token"},
		"access_token_ttl": 86400, "refresh_token_ttl": 400 * 24 * 3600})
	require.Equal(t, http.StatusCreated, w.Code)
	var resp models.ClientRegistrationResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 3600, resp.AccessTokenTTL)
	assert.Equal(t, 30*24*3600, resp.RefreshTokenTTL)
	a := app{id: resp.ClientID, secret: resp.ClientSecret, redirect: "https://l.example.org/cb"}

	// Admin update clamps too.
	up := e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"access_token_ttl": 5})
	require.Equal(t, http.StatusOK, up.Code)
	assert.Equal(t, 300, e.count(`SELECT access_token_ttl FROM oauth_clients WHERE client_id = $1`, a.id))
	e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"access_token_ttl": 99999})

	tr := e.signedIn(a, u, "read")
	assert.LessOrEqual(t, tr.ExpiresIn, 3600)
	assert.Greater(t, tr.ExpiresIn, 3500)

	var codeSecs, accessSecs, refreshDays float64
	require.NoError(t, e.db.QueryRow(`SELECT extract(epoch FROM expires_at - created_at) FROM authorization_codes`).Scan(&codeSecs))
	assert.LessOrEqual(t, codeSecs, 60.5, "code lifetime")
	require.NoError(t, e.db.QueryRow(`SELECT extract(epoch FROM expires_at - created_at) FROM oauth_access_tokens`).Scan(&accessSecs))
	assert.LessOrEqual(t, accessSecs, 3600.5)
	require.NoError(t, e.db.QueryRow(`SELECT extract(epoch FROM expires_at - created_at)/86400 FROM oauth_refresh_tokens`).Scan(&refreshDays))
	assert.LessOrEqual(t, refreshDays, 30.01)
	assert.Equal(t, 60*time.Second, authCodeTTL)

	// A code older than a minute is no use.
	code := e.code(a, u, "read")
	_, err := e.db.Exec(`UPDATE authorization_codes SET expires_at = now() - interval '1 second' WHERE code_hash = $1`, hashSecret(code))
	require.NoError(t, err)
	xw, _ := e.exchange(a, code)
	assert.Equal(t, http.StatusBadRequest, xw.Code)
}

func TestRefreshChainHasAnAbsoluteCap(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("chain_user")
	a := e.newApp(appOpts{})
	tr := e.signedIn(a, u, "read")

	// Eighty-nine days into the chain: the next token cannot outlive day 90.
	_, err := e.db.Exec(`UPDATE oauth_refresh_tokens SET family_created_at = now() - interval '89 days'`)
	require.NoError(t, err)
	w, tr2 := e.refresh(a, tr.RefreshToken, "")
	require.Equal(t, http.StatusOK, w.Code)
	var daysLeft float64
	require.NoError(t, e.db.QueryRow(`SELECT extract(epoch FROM expires_at - now())/86400 FROM oauth_refresh_tokens WHERE token_hash = $1`,
		hashSecret(tr2.RefreshToken)).Scan(&daysLeft))
	assert.Less(t, daysLeft, 1.01)

	// Ninety-one days: over.
	_, err = e.db.Exec(`UPDATE oauth_refresh_tokens SET family_created_at = now() - interval '91 days', expires_at = now() + interval '5 days'`)
	require.NoError(t, err)
	w, _ = e.refresh(a, tr2.RefreshToken, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid_grant", e.errCode(w))
}

// ---------------------------------------------------------------------
// Items 6 and 7: discovery and the issuer
// ---------------------------------------------------------------------

func TestDiscoveryDocumentsAreTrimmed(t *testing.T) {
	e := newHEnv(t)

	get := func(path string) map[string]any {
		w := e.do("GET", path, nil, nil)
		require.Equal(t, http.StatusOK, w.Code)
		var m map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
		return m
	}
	for _, path := range []string{"/.well-known/openid-configuration", "/.well-known/oauth-authorization-server"} {
		doc := get(path)
		raw, _ := json.Marshal(doc)
		text := string(raw)
		for _, gone := range []string{"client_credentials", "plain", "registration_endpoint", "fragment", "form_post",
			"code id_token", "ES256", "tags:wrangle", `"admin"`, "ao3_", "example.com"} {
			assert.NotContains(t, text, gone, "%s must not mention %s", path, gone)
		}
		assert.Equal(t, "https://archive.example.org", doc["issuer"])
		assert.Equal(t, []any{"code"}, doc["response_types_supported"])
		assert.Equal(t, []any{"S256"}, doc["code_challenge_methods_supported"])
		assert.Equal(t, []any{"authorization_code", "refresh_token"}, doc["grant_types_supported"])
		assert.Equal(t, []any{"query"}, doc["response_modes_supported"])
		assert.Contains(t, doc["scopes_supported"], "read")
		assert.Contains(t, doc["token_endpoint_auth_methods_supported"], "none")
	}

	oidc := get("/.well-known/openid-configuration")
	assert.Equal(t, []any{"RS256"}, oidc["id_token_signing_alg_values_supported"])
	assert.Equal(t, "https://archive.example.org/privacy", oidc["op_policy_uri"])
	assert.Equal(t, "https://archive.example.org/terms", oidc["op_tos_uri"])
	assert.NotContains(t, oidc, "service_documentation")

	// Everything claimed is something userinfo or the ID token carries.
	u := e.newUser("claims_user")
	a := e.newApp(appOpts{})
	tr := e.signedIn(a, u, "openid profile email")
	uw := e.do("GET", "/auth/userinfo", nil, func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+tr.AccessToken) })
	require.Equal(t, http.StatusOK, uw.Code)
	var info map[string]any
	json.Unmarshal(uw.Body.Bytes(), &info)
	claims := oidc["claims_supported"].([]any)
	for k := range info {
		assert.Contains(t, claims, k, "userinfo returns %s but discovery does not list it", k)
	}
	for _, c := range claims {
		assert.NotContains(t, c, "ao3_")
	}
}

func TestIssuerComesFromSiteURL(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("issuer_user")
	a := e.newApp(appOpts{})

	bad := map[string]string{
		"unset":                "",
		"old default":          "https://ao3.example.com",
		"placeholder":          "https://your-domain.com",
		"not https":            "http://archive.example.org",
		"localhost in prod":    "https://localhost:3000",
		"userinfo in address":  "https://user:pw@archive.example.org",
		"query in address":     "https://archive.example.org?x=1",
		"no scheme or host":    "archive.example.org",
		"private network name": "https://archive.local",
	}
	for name, site := range bad {
		t.Setenv("SITE_URL", site)
		t.Setenv("BASE_URL", "https://other.example.org") // not used any more
		for _, path := range []string{"/.well-known/openid-configuration", "/.well-known/oauth-authorization-server"} {
			w := e.do("GET", path, nil, nil)
			assert.Equal(t, http.StatusServiceUnavailable, w.Code, "%s: %s", name, path)
		}
	}

	// With no usable SITE_URL an ID token is not issued, and the code is not used up.
	t.Setenv("SITE_URL", "https://archive.example.org")
	code := e.code(a, u, "openid profile")
	t.Setenv("SITE_URL", "")
	w, _ := e.exchange(a, code)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_access_tokens`))
	t.Setenv("SITE_URL", "https://archive.example.org")
	w, tr := e.exchange(a, code)
	require.Equal(t, http.StatusOK, w.Code)
	parts := strings.Split(tr.IDToken, ".")
	require.Len(t, parts, 3)
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	assert.Contains(t, string(payload), `"iss":"https://archive.example.org"`)
	header, _ := base64.RawURLEncoding.DecodeString(parts[0])
	assert.Contains(t, string(header), `"kid"`)

	// Userinfo no longer names a made-up host.
	uw := e.do("GET", "/auth/userinfo", nil, func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+tr.AccessToken) })
	assert.NotContains(t, uw.Body.String(), "ao3.example.com")
	assert.NotContains(t, uw.Body.String(), "/users/") // no link to an account page

	// In development, http to this machine is fine.
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("SITE_URL", "http://localhost:3001")
	w = e.do("GET", "/.well-known/openid-configuration", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"issuer":"http://localhost:3001"`)
}

// ---------------------------------------------------------------------
// Item 8: hashes at rest
// ---------------------------------------------------------------------

func TestTokensAndCodesAreNotStoredInPlaintext(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("hash_user")
	a := e.newApp(appOpts{})
	code := e.code(a, u, "read")

	// The code is stored as its hash while it waits.
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM authorization_codes t WHERE t::text LIKE '%' || $1 || '%'`, code))
	assert.Equal(t, 1, e.count(`SELECT count(*) FROM authorization_codes WHERE code_hash = $1`, hashSecret(code)))

	w, tr := e.exchange(a, code)
	require.Equal(t, http.StatusOK, w.Code)
	for _, table := range []string{"authorization_codes", "oauth_access_tokens", "oauth_refresh_tokens", "oauth_pending", "oauth_audit_log"} {
		for _, secret := range []string{code, tr.AccessToken, tr.RefreshToken, a.secret} {
			q := fmt.Sprintf(`SELECT count(*) FROM %s t WHERE t::text LIKE '%%' || $1 || '%%'`, table)
			assert.Equal(t, 0, e.count(q, secret), "raw value found in %s", table)
		}
	}
	assert.Equal(t, 1, e.count(`SELECT count(*) FROM oauth_access_tokens WHERE token_hash = $1`, hashSecret(tr.AccessToken)))
	assert.Equal(t, 1, e.count(`SELECT count(*) FROM oauth_refresh_tokens WHERE token_hash = $1`, hashSecret(tr.RefreshToken)))

	// The table itself refuses a plain value.
	_, err := e.db.Exec(`INSERT INTO oauth_access_tokens (id, token_hash, user_id, client_id, scopes, expires_at, family_id)
		VALUES ($1, 'plain-token', $2, $3, '{read}', now() + interval '1 hour', $1)`, uuid.New(), u.ID, a.id)
	assert.Error(t, err)

	// The raw token still works as a bearer token (it is looked up by hash).
	assert.NotEqual(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))
}

// ---------------------------------------------------------------------
// Item 9: codes are used once, and bound to their redirect address
// ---------------------------------------------------------------------

func TestCodeUsedTwiceRevokesWhatItMade(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("replay_user")
	a := e.newApp(appOpts{})
	code := e.code(a, u, "profile")

	w, first := e.exchange(a, code)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, http.StatusOK, e.userinfoStatus(first.AccessToken))

	w, _ = e.exchange(a, code)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid_grant", e.errCode(w))

	// RFC 6749 4.1.2: what the first use produced is revoked.
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(first.AccessToken))
	rw, _ := e.refresh(a, first.RefreshToken, "")
	assert.Equal(t, http.StatusBadRequest, rw.Code)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_access_tokens WHERE NOT is_revoked`))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_refresh_tokens WHERE NOT is_revoked`))
}

func TestCodeUsedConcurrentlyGivesOneSuccess(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("race_user")
	a := e.newApp(appOpts{})
	code := e.code(a, u, "profile")

	const n = 8
	var wg sync.WaitGroup
	results := make([]*httptest.ResponseRecorder, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], _ = e.exchange(a, code)
		}(i)
	}
	close(start)
	wg.Wait()

	var winner models.TokenResponse
	ok := 0
	for _, w := range results {
		if w.Code == http.StatusOK {
			ok++
			json.Unmarshal(w.Body.Bytes(), &winner)
		} else {
			assert.Equal(t, http.StatusBadRequest, w.Code)
		}
	}
	assert.Equal(t, 1, ok, "exactly one exchange may succeed")
	require.NotEmpty(t, winner.AccessToken)
	// The losers were replays, so the winner's tokens are revoked.
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(winner.AccessToken))
	assert.Equal(t, 1, e.count(`SELECT count(*) FROM oauth_access_tokens`))
}

func TestCodeIsBoundToItsRedirectURI(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("redirect_user")
	a := e.newApp(appOpts{})
	// A second registered address for the same app.
	other := a.redirect + "/other"
	up := e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"redirect_uris": []string{a.redirect, other}})
	require.Equal(t, http.StatusOK, up.Code)

	code := e.code(a, u, "read")
	for _, wrong := range []string{other, a.redirect + "/", strings.ToUpper(a.redirect), ""} {
		w, _ := e.tokenRequest(a, url.Values{"grant_type": {"authorization_code"}, "code": {code},
			"redirect_uri": {wrong}, "code_verifier": {testVerifier}})
		assert.Equal(t, http.StatusBadRequest, w.Code, "redirect_uri %q", wrong)
	}
	// A wrong address does not use the code up.
	w, _ := e.exchange(a, code)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCodeBelongsToOneClient(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("two_clients_user")
	a, b := e.newApp(appOpts{}), e.newApp(appOpts{})
	code := e.code(a, u, "read")
	w, _ := e.tokenRequest(b, url.Values{"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {a.redirect}, "code_verifier": {testVerifier}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w, _ = e.exchange(a, code)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------
// Item 10: refresh rotation, families, standing
// ---------------------------------------------------------------------

func TestRefreshRotatesAndRevokesThePreviousAccessToken(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("rotate_user")
	a := e.newApp(appOpts{})
	first := e.signedIn(a, u, "profile")

	w, second := e.refresh(a, first.RefreshToken, "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotEqual(t, first.RefreshToken, second.RefreshToken)
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(first.AccessToken), "previous access token")
	assert.Equal(t, http.StatusOK, e.userinfoStatus(second.AccessToken))
	assert.Equal(t, 1, e.count(`SELECT count(DISTINCT family_id) FROM oauth_refresh_tokens`), "one family")
}

func TestRefreshReplayRevokesTheFamily(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("family_user")
	a := e.newApp(appOpts{})
	first := e.signedIn(a, u, "profile")
	w, second := e.refresh(a, first.RefreshToken, "")
	require.Equal(t, http.StatusOK, w.Code)
	w, third := e.refresh(a, second.RefreshToken, "")
	require.Equal(t, http.StatusOK, w.Code)

	// Someone presents the first token again.
	w, _ = e.refresh(a, first.RefreshToken, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid_grant", e.errCode(w))

	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(third.AccessToken))
	w, _ = e.refresh(a, third.RefreshToken, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_refresh_tokens WHERE NOT is_revoked`))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_access_tokens WHERE NOT is_revoked`))

	// Another sign-in is a different family and is untouched.
	other := e.signedIn(a, u, "profile")
	w, _ = e.refresh(a, first.RefreshToken, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, http.StatusOK, e.userinfoStatus(other.AccessToken))
}

func TestRefreshUsedConcurrentlyGivesOneSuccess(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("refresh_race_user")
	a := e.newApp(appOpts{})
	first := e.signedIn(a, u, "profile")

	const n = 8
	var wg sync.WaitGroup
	results := make([]*httptest.ResponseRecorder, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], _ = e.refresh(a, first.RefreshToken, "")
		}(i)
	}
	close(start)
	wg.Wait()

	ok := 0
	var winner models.TokenResponse
	for _, w := range results {
		if w.Code == http.StatusOK {
			ok++
			json.Unmarshal(w.Body.Bytes(), &winner)
		}
	}
	assert.Equal(t, 1, ok, "exactly one refresh may succeed")
	// The replays revoked the family, including what the winner got.
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(winner.AccessToken))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_refresh_tokens WHERE NOT is_revoked`))
}

func TestRefreshNeedsLiveConsentActiveClientAndGoodStanding(t *testing.T) {
	e := newHEnv(t)
	a := e.newApp(appOpts{})

	check := func(name string, breakIt func(u *models.User), fixIt func(u *models.User)) {
		u := e.newUser("standing_" + strings.ReplaceAll(name, " ", "_"))
		tr := e.signedIn(a, u, "profile")
		breakIt(u)
		w, _ := e.refresh(a, tr.RefreshToken, "")
		assert.Equal(t, http.StatusBadRequest, w.Code, name)
		assert.Equal(t, "invalid_grant", e.errCode(w), name)
		if fixIt != nil {
			// The refused attempt did not use the token up, so once the
			// reason is gone the same token works.
			fixIt(u)
			w, _ = e.refresh(a, tr.RefreshToken, "")
			assert.Equal(t, http.StatusOK, w.Code, name+" after fixing")
		}
	}
	exec := func(q string) func(u *models.User) {
		return func(u *models.User) {
			_, err := e.db.Exec(q, u.ID)
			require.NoError(t, err)
		}
	}
	check("banned user", exec(`UPDATE users SET banned = true WHERE id = $1`), exec(`UPDATE users SET banned = false WHERE id = $1`))
	check("suspended user", exec(`UPDATE users SET suspended = true, suspended_until = now() + interval '1 day' WHERE id = $1`),
		exec(`UPDATE users SET suspended_until = now() - interval '1 second' WHERE id = $1`))
	check("indefinitely suspended", exec(`UPDATE users SET suspended = true, suspended_until = NULL WHERE id = $1`), nil)
	check("inactive user", exec(`UPDATE users SET is_active = false WHERE id = $1`), exec(`UPDATE users SET is_active = true WHERE id = $1`))
	check("consent revoked", exec(`UPDATE user_consents SET is_revoked = true WHERE user_id = $1`), nil)
	check("consent expired", exec(`UPDATE user_consents SET expires_at = now() - interval '1 second' WHERE user_id = $1`), nil)
	check("consent no longer covers", exec(`UPDATE user_consents SET scopes = '{read}' WHERE user_id = $1`), nil)

	// Inactive client: it cannot even authenticate, and the token is dead.
	u := e.newUser("standing_client")
	b := e.newApp(appOpts{})
	tr := e.signedIn(b, u, "profile")
	_, err := e.db.Exec(`UPDATE oauth_clients SET is_active = false WHERE client_id = $1`, b.id)
	require.NoError(t, err)
	w, _ := e.refresh(b, tr.RefreshToken, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))
}

func TestAccessTokenStopsWorkingWhenTheUserIsNotInGoodStanding(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("good_standing")
	a := e.newApp(appOpts{})
	tr := e.signedIn(a, u, "profile")
	require.Equal(t, http.StatusOK, e.userinfoStatus(tr.AccessToken))

	_, err := e.db.Exec(`UPDATE users SET banned = true WHERE id = $1`, u.ID)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))
	_, err = e.db.Exec(`UPDATE users SET banned = false, suspended = true, suspended_until = now() + interval '1 hour' WHERE id = $1`, u.ID)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))
	_, err = e.db.Exec(`UPDATE users SET suspended = false WHERE id = $1`, u.ID)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, e.userinfoStatus(tr.AccessToken))
}

// ---------------------------------------------------------------------
// Item 11: consent revocation
// ---------------------------------------------------------------------

func TestRevokingConsentKillsRefreshAndCodes(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("consent_user")
	a := e.newApp(appOpts{})
	tr := e.signedIn(a, u, "profile")
	pending := e.code(a, u, "profile") // a code not yet exchanged

	var consentID string
	require.NoError(t, e.db.QueryRow(`SELECT id FROM user_consents WHERE user_id = $1`, u.ID).Scan(&consentID))

	// Someone else cannot revoke it.
	stranger := e.newUser("consent_stranger")
	w := e.do("DELETE", "/auth/consents/"+consentID, nil, e.asUser(stranger.ID))
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, http.StatusOK, e.userinfoStatus(tr.AccessToken))

	w = e.do("DELETE", "/auth/consents/"+consentID, nil, e.asUser(u.ID))
	require.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))
	rw, _ := e.refresh(a, tr.RefreshToken, "")
	assert.Equal(t, http.StatusBadRequest, rw.Code)
	xw, _ := e.exchange(a, pending)
	assert.Equal(t, http.StatusBadRequest, xw.Code)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_refresh_tokens WHERE NOT is_revoked`))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM authorization_codes WHERE used_at IS NULL`))
}

// ---------------------------------------------------------------------
// Item 12: client deactivation, deletion and secret reset
// ---------------------------------------------------------------------

func TestDeactivatingOrDeletingAClientKillsItsTokens(t *testing.T) {
	for _, how := range []string{"deactivate", "delete"} {
		t.Run(how, func(t *testing.T) {
			e := newHEnv(t)
			u := e.newUser("deactivate_user")
			a := e.newApp(appOpts{})
			tr := e.signedIn(a, u, "profile")
			pending := e.code(a, u, "profile")

			var w *httptest.ResponseRecorder
			if how == "deactivate" {
				w = e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"is_active": false})
			} else {
				w = e.adminJSON("DELETE", "/api/v1/auth/admin/oauth/clients/"+a.id, nil)
			}
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())

			assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))
			assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_access_tokens WHERE NOT is_revoked`))
			assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_refresh_tokens WHERE NOT is_revoked`))
			assert.Equal(t, 0, e.count(`SELECT count(*) FROM user_consents WHERE NOT is_revoked`))
			assert.Equal(t, 0, e.count(`SELECT count(*) FROM authorization_codes WHERE used_at IS NULL`))
			xw, _ := e.exchange(a, pending)
			assert.Equal(t, http.StatusUnauthorized, xw.Code)

			// An admin can still see it.
			gw := e.do("GET", "/api/v1/auth/admin/oauth/clients/"+a.id, nil, e.asAdmin)
			assert.Equal(t, http.StatusOK, gw.Code)
			assert.Contains(t, gw.Body.String(), `"is_active":false`)
			// But the authorize step treats it as unknown, and stores nothing.
			aw := e.do("GET", a.authorizeURL("read", ""), nil, nil)
			assert.Equal(t, http.StatusBadRequest, aw.Code)
			assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_pending`))
		})
	}
}

func TestResetSecretRevokesAccessAndRefreshTokens(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("reset_user")
	a := e.newApp(appOpts{})
	tr := e.signedIn(a, u, "profile")

	w := e.adminJSON("POST", "/api/v1/auth/admin/oauth/clients/"+a.id+"/reset-secret", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var out struct {
		Secret string `json:"client_secret"`
	}
	json.Unmarshal(w.Body.Bytes(), &out)
	require.NotEmpty(t, out.Secret)
	require.NotEqual(t, a.secret, out.Secret)

	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_refresh_tokens WHERE NOT is_revoked`))
	rw, _ := e.refresh(a, tr.RefreshToken, "") // old secret
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
	a.secret = out.Secret
	rw, _ = e.refresh(a, tr.RefreshToken, "") // new secret, but the token is revoked
	assert.Equal(t, http.StatusBadRequest, rw.Code)
	// The person's approval is untouched: a fresh sign-in works with no new consent.
	w2 := e.do("GET", a.authorizeURL("profile", ""), nil, e.asUser(u.ID))
	assert.Equal(t, http.StatusFound, w2.Code)
	assert.Contains(t, w2.Header().Get("Location"), "code=")
}

// ---------------------------------------------------------------------
// Item 13: revocation endpoint
// ---------------------------------------------------------------------

func TestRevokeIsScopedToTheCallingClientAndTakesItsSibling(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("revoke_user")
	a, b := e.newApp(appOpts{}), e.newApp(appOpts{})
	revoke := func(who app, token string) *httptest.ResponseRecorder {
		return e.do("POST", "/auth/revoke", who.form(url.Values{"token": {token}}), nil)
	}

	tr := e.signedIn(a, u, "profile")
	// Another app cannot revoke it (and is told "ok" all the same, per RFC 7009).
	for _, tok := range []string{tr.AccessToken, tr.RefreshToken} {
		assert.Equal(t, http.StatusOK, revoke(b, tok).Code)
	}
	assert.Equal(t, http.StatusOK, e.userinfoStatus(tr.AccessToken))
	rw, _ := e.refresh(a, tr.RefreshToken, "")
	require.Equal(t, http.StatusOK, rw.Code, "the other app's revoke must not have touched it")
	var tr2 models.TokenResponse
	json.Unmarshal(rw.Body.Bytes(), &tr2)

	// Revoking the access token takes the refresh token with it.
	assert.Equal(t, http.StatusOK, revoke(a, tr2.AccessToken).Code)
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr2.AccessToken))
	rw, _ = e.refresh(a, tr2.RefreshToken, "")
	assert.Equal(t, http.StatusBadRequest, rw.Code)

	// Revoking the refresh token takes its access token with it.
	tr3 := e.signedIn(a, u, "profile")
	assert.Equal(t, http.StatusOK, revoke(a, tr3.RefreshToken).Code)
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr3.AccessToken))

	// A public app has no secret and may revoke its own tokens with just its
	// client_id (see TestPublicAppsCanRevokeTheirOwnTokens).
	pub := e.newApp(appOpts{public: true})
	ptr := e.signedIn(pub, u, "read")
	assert.Equal(t, http.StatusOK, revoke(pub, ptr.AccessToken).Code)
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(ptr.AccessToken))
}

// ---------------------------------------------------------------------
// Item 14: client authentication
// ---------------------------------------------------------------------

func basic(id, secret string) func(*http.Request) {
	return func(r *http.Request) {
		r.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(id+":"+secret)))
	}
}

func TestBasicAuthOnlyClientWorks(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("basic_user")
	a := e.newApp(appOpts{})
	code := e.code(a, u, "profile")

	// No client_id or client_secret in the body at all.
	w := e.do("POST", "/auth/token", url.Values{"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {a.redirect}, "code_verifier": {testVerifier}}, basic(a.id, a.secret))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var tr models.TokenResponse
	json.Unmarshal(w.Body.Bytes(), &tr)

	// Introspection and revocation take Basic too.
	iw := e.do("POST", "/auth/introspect", url.Values{"token": {tr.AccessToken}}, basic(a.id, a.secret))
	assert.Contains(t, iw.Body.String(), `"active":true`)
	rw := e.do("POST", "/auth/revoke", url.Values{"token": {tr.AccessToken}}, basic(a.id, a.secret))
	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))

	// Wrong secret.
	w = e.do("POST", "/auth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tr.RefreshToken}}, basic(a.id, "nope"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Header().Get("WWW-Authenticate"), "Basic")

	// Basic and a form secret together is a mistake.
	w = e.do("POST", "/auth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tr.RefreshToken},
		"client_secret": {a.secret}}, basic(a.id, a.secret))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	// A form client_id that disagrees with Basic is refused.
	w = e.do("POST", "/auth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tr.RefreshToken},
		"client_id": {uuid.NewString()}}, basic(a.id, a.secret))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBasicCredentialsAreURLDecoded(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("decode_user")
	a := e.newApp(appOpts{})
	// A secret with characters that must be percent-encoded in Basic.
	weird := "p@ss+w%rd /é"
	hash, err := bcryptHash(weird)
	require.NoError(t, err)
	_, err = e.db.Exec(`UPDATE oauth_clients SET client_secret = $1 WHERE client_id = $2`, hash, a.id)
	require.NoError(t, err)

	tok := func() (*httptest.ResponseRecorder, string) {
		code := e.code(a, u, "profile")
		return nil, code
	}
	_, code := tok()
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {a.redirect}, "code_verifier": {testVerifier}}

	// Encoded the way RFC 6749 2.3.1 says (form-url-encoding): accepted.
	w := e.do("POST", "/auth/token", form, basic(url.QueryEscape(a.id), url.QueryEscape(weird)))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Sent raw, the secret is read as an encoded string and does not match.
	_, code = tok()
	form.Set("code", code)
	w = e.do("POST", "/auth/token", form, basic(a.id, weird))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// A broken escape is refused, not mistaken for a character.
	w = e.do("POST", "/auth/token", form, basic(a.id, "%zz"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecretsInTheQueryStringAreRefused(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("query_user")
	a := e.newApp(appOpts{})
	tr := e.signedIn(a, u, "profile")

	for _, path := range []string{"/auth/token", "/auth/introspect", "/auth/revoke"} {
		// The secret in the address is refused outright, even with a good body.
		w := e.do("POST", path+"?client_secret="+url.QueryEscape(a.secret), a.form(url.Values{
			"grant_type": {"refresh_token"}, "refresh_token": {tr.RefreshToken}, "token": {tr.AccessToken}}), nil)
		assert.Equal(t, http.StatusBadRequest, w.Code, path)
		assert.Equal(t, "invalid_request", e.errCode(w), path)

		// And the address alone is not a way to authenticate.
		w = e.do("POST", fmt.Sprintf("%s?client_id=%s&client_secret=%s&token=%s&refresh_token=%s&grant_type=refresh_token",
			path, a.id, url.QueryEscape(a.secret), tr.AccessToken, tr.RefreshToken), url.Values{}, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code, path)
	}
	// The same values in the body work.
	w, _ := e.refresh(a, tr.RefreshToken, "")
	assert.Equal(t, http.StatusOK, w.Code)

	// A client_id in the address is simply not read.
	w = e.do("POST", "/auth/token?client_id="+a.id, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"x"}}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// A JSON body is not a form.
	jb := httptest.NewRequest("POST", "/auth/token", strings.NewReader(`{"grant_type":"refresh_token"}`))
	jb.Header.Set("Content-Type", "application/json")
	jw := httptest.NewRecorder()
	e.router.ServeHTTP(jw, jb)
	assert.Equal(t, http.StatusBadRequest, jw.Code)
}

func TestUnknownClientsCostAsMuchAsKnownOnes(t *testing.T) {
	e := newHEnv(t)
	spendBcrypt("warm up")
	start := time.Now()
	w := e.do("POST", "/auth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"x"},
		"client_id": {uuid.NewString()}, "client_secret": {"guess"}}, nil)
	took := time.Since(start)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	// A bcrypt check at the default cost takes tens of milliseconds; skipping
	// it would answer in well under one.
	assert.Greater(t, took, 5*time.Millisecond)
	// Same for an id that is not even a UUID.
	start = time.Now()
	w = e.do("POST", "/auth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {"nope"}, "client_secret": {"guess"}}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Greater(t, time.Since(start), 5*time.Millisecond)
}

func TestPublicClientsAuthenticateWithoutASecretOnlyAtTheTokenEndpoint(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("public_user")
	pub := e.newApp(appOpts{public: true})
	tr := e.signedIn(pub, u, "read")
	assert.NotEmpty(t, tr.RefreshToken)

	// Refresh needs no secret.
	w, tr2 := e.refresh(pub, tr.RefreshToken, "")
	assert.Equal(t, http.StatusOK, w.Code)

	// Introspection is for confidential clients only.
	iw := e.do("POST", "/auth/introspect", pub.form(url.Values{"token": {tr2.AccessToken}}), nil)
	assert.Equal(t, http.StatusUnauthorized, iw.Code)

	// A public client that sends a secret is refused.
	w = e.do("POST", "/auth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tr2.RefreshToken},
		"client_id": {pub.id}, "client_secret": {"made-up"}}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------
// Item 15: introspection
// ---------------------------------------------------------------------

func TestIntrospectionOnlyAnswersForTheOwningClient(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("introspect_user")
	a, b := e.newApp(appOpts{}), e.newApp(appOpts{})
	tr := e.signedIn(a, u, "profile")

	introspect := func(who app, token string) map[string]any {
		w := e.do("POST", "/auth/introspect", who.form(url.Values{"token": {token}}), nil)
		require.Equal(t, http.StatusOK, w.Code)
		var m map[string]any
		json.Unmarshal(w.Body.Bytes(), &m)
		return m
	}
	own := introspect(a, tr.AccessToken)
	assert.Equal(t, true, own["active"])
	assert.Equal(t, a.id, own["client_id"])
	assert.NotContains(t, own, "username")
	assert.NotEqual(t, u.ID.String(), own["sub"]) // pairwise: see oauth_identity_test.go

	other := introspect(b, tr.AccessToken)
	assert.Equal(t, map[string]any{"active": false}, other)
	assert.Equal(t, map[string]any{"active": false}, introspect(a, "not-a-token"))
}

// ---------------------------------------------------------------------
// Item 16: rate limits and request size caps
// ---------------------------------------------------------------------

func TestOAuthEndpointsAreRateLimited(t *testing.T) {
	e := newHEnv(t)
	e.as.oauthRates = map[string]oauthRate{
		"token":      {PerIP: 3, PerClient: 1 << 20},
		"introspect": {PerIP: 3, PerClient: 1 << 20},
		"revoke":     {PerIP: 3, PerClient: 1 << 20},
		"authorize":  {PerIP: 3, PerClient: 1 << 20},
	}
	a := e.newApp(appOpts{})
	for _, ep := range []struct {
		method, path string
		form         url.Values
	}{
		{"POST", "/auth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"x"}}},
		{"POST", "/auth/introspect", url.Values{"token": {"x"}}},
		{"POST", "/auth/revoke", url.Values{"token": {"x"}}},
		{"GET", a.authorizeURL("read", ""), nil},
	} {
		for i := 1; i <= 3; i++ {
			w := e.do(ep.method, ep.path, ep.form, nil)
			require.NotEqual(t, http.StatusTooManyRequests, w.Code, "%s request %d", ep.path, i)
		}
		w := e.do(ep.method, ep.path, ep.form, nil)
		assert.Equal(t, http.StatusTooManyRequests, w.Code, ep.path)
		assert.NotEmpty(t, w.Header().Get("Retry-After"), ep.path)
	}
	// Other addresses are counted separately.
	w := e.do("POST", "/auth/revoke", url.Values{"token": {"x"}}, func(r *http.Request) { r.RemoteAddr = "203.0.113.9:1234" })
	assert.NotEqual(t, http.StatusTooManyRequests, w.Code)
}

// The per-app limit counts only requests where the app proved who it is:
// strangers naming a real app's (public) client_id can't lock it out.
func TestOAuthEndpointsAreRateLimitedPerClientID(t *testing.T) {
	e := newHEnv(t)
	e.as.oauthRates = map[string]oauthRate{"introspect": {PerIP: 1 << 20, PerClient: 2}}
	a := e.newApp(appOpts{})
	introspect := func(ip, secret string) int {
		return e.do("POST", "/auth/introspect", url.Values{"token": {"whatever"}, "client_id": {a.id}, "client_secret": {secret}},
			func(r *http.Request) { r.RemoteAddr = ip + ":1" }).Code
	}
	// A stranger with the id but not the secret, from many addresses.
	for i := 0; i < 10; i++ {
		assert.Equal(t, http.StatusUnauthorized, introspect(fmt.Sprintf("198.51.100.%d", i+1), "wrong"))
	}
	// The real app is not locked out by that, and is limited on its own use.
	assert.Equal(t, http.StatusOK, introspect("203.0.113.1", a.secret))
	assert.Equal(t, http.StatusOK, introspect("203.0.113.2", a.secret))
	assert.Equal(t, http.StatusTooManyRequests, introspect("203.0.113.3", a.secret), "the app is counted across addresses")
	// Authorize has no per-app limit: it's the page people are sent to.
	e.as.oauthRates = map[string]oauthRate{"authorize": {PerIP: 1 << 20, PerClient: 1}}
	for i := 0; i < 3; i++ {
		w := e.do("GET", a.authorizeURL("read", ""), nil, func(r *http.Request) { r.RemoteAddr = fmt.Sprintf("192.0.2.%d:1", i+1) })
		assert.NotEqual(t, http.StatusTooManyRequests, w.Code)
	}
}

func TestAuthorizeParameterSizeCapsAndPendingRows(t *testing.T) {
	e := newHEnv(t)
	a := e.newApp(appOpts{})
	long := func(n int) string { return strings.Repeat("a", n) }

	for name, extra := range map[string]string{
		"state":      "&state=" + long(513),
		"nonce":      "&nonce=" + long(513),
		"login_hint": "&login_hint=" + long(513),
	} {
		w := e.do("GET", a.authorizeURL("read", extra), nil, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code, name)
	}
	w := e.do("GET", fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&scope=read&redirect_uri=%s", a.id,
		url.QueryEscape("https://app.example.org/"+long(2040))), nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code, "redirect_uri")
	// At the limit is fine.
	w = e.do("GET", a.authorizeURL("read", "&state="+long(512)+"&nonce="+long(512)+"&login_hint="+long(512)), nil, nil)
	assert.Equal(t, http.StatusFound, w.Code)
	// Logged out: the request travels in the address to the website's own
	// authorize page, and nothing is stored on the server.
	assert.True(t, strings.HasPrefix(w.Header().Get("Location"), "/oauth/authorize?client_id=") ||
		strings.HasPrefix(w.Header().Get("Location"), "/oauth/authorize?response_type=code&client_id="), w.Header().Get("Location")[:60])
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_pending`))

	// Unknown, malformed and switched-off apps leave nothing behind.
	e.db.Exec(`DELETE FROM oauth_pending`)
	unknown := fmt.Sprintf("/auth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=read%s", uuid.NewString(), url.QueryEscape(a.redirect), pkceParams)
	assert.Equal(t, http.StatusBadRequest, e.do("GET", unknown, nil, nil).Code)
	bad := strings.Replace(unknown, unknown[strings.Index(unknown, "client_id=")+10:][:36], "not-a-uuid", 1)
	assert.Equal(t, http.StatusBadRequest, e.do("GET", bad, nil, nil).Code)
	_, err := e.db.Exec(`UPDATE oauth_clients SET is_active = false WHERE client_id = $1`, a.id)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, e.do("GET", a.authorizeURL("read", ""), nil, nil).Code)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_pending`))
}

// ---------------------------------------------------------------------
// Item 17: redirect addresses
// ---------------------------------------------------------------------

func TestRedirectURIRules(t *testing.T) {
	cases := []struct {
		uri     string
		public  bool
		dev     bool
		allowed bool
	}{
		{"https://app.example.org/cb", false, false, true},
		{"https://app.example.org/cb?x=1", false, false, true},
		{"http://app.example.org/cb", false, false, false},
		{"http://localhost:3000/cb", false, false, false},
		{"http://localhost:3000/cb", false, true, true},
		{"http://127.0.0.1:8080/cb", true, true, true},
		{"http://[::1]:8080/cb", true, true, true},
		{"http://evil.example.net/cb", false, true, false},
		{"http://localhost.evil.example.net/cb", false, true, false},
		{"javascript:alert(1)", true, true, false},
		{"JavaScript:alert(1)", true, true, false},
		{"data:text/html,hi", true, true, false},
		{"file:///etc/passwd", true, true, false},
		{"vbscript:msgbox(1)", true, true, false},
		{"https://app.example.org/cb#frag", false, false, false},
		{"https://app.example.org/cb#", false, false, false},
		{"https://user@app.example.org/cb", false, false, false},
		{"https://user:pw@app.example.org/cb", false, false, false},
		{"https:///cb", false, false, false},
		{"https://app.example.org/a b", false, false, false},
		{"https://app.example.org\\@evil.example.net/", false, false, false},
		{"com.example.app://callback", true, false, true},
		{"com.example.app:/callback", true, false, true},
		{"com.example.app://callback", false, false, false}, // confidential apps use https
		{"myapp://callback", true, false, false},            // no dot in the scheme
		{"", false, false, false},
		{"https://app.example.org/" + strings.Repeat("a", 2048), false, false, false},
	}
	for _, c := range cases {
		if c.dev {
			t.Setenv("ENVIRONMENT", "development")
		} else {
			t.Setenv("ENVIRONMENT", "")
		}
		assert.Equal(t, c.allowed, validRedirectURI(c.uri, c.public), "%q public=%v dev=%v", c.uri, c.public, c.dev)
	}
	t.Setenv("ENVIRONMENT", "")

	// Registration and admin update use the same rule.
	e := newHEnv(t)
	w := e.register(map[string]any{"client_name": "X", "redirect_uris": []string{"https://ok.example.org/cb", "javascript:alert(1)"}, "scopes": []string{"read"}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	a := e.newApp(appOpts{})
	up := e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"redirect_uris": []string{"http://evil.example.net/cb"}})
	assert.Equal(t, http.StatusBadRequest, up.Code)
	up = e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"redirect_uris": []string{}})
	assert.Equal(t, http.StatusBadRequest, up.Code)
}

func TestRedirectURIsAreRecheckedAtAuthorize(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("old_uri_user")
	a := e.newApp(appOpts{})
	// An address stored before the rules got stricter is no longer honoured.
	_, err := e.db.Exec(`UPDATE oauth_clients SET redirect_uris = ARRAY['http://old.example.net/cb'] WHERE client_id = $1`, a.id)
	require.NoError(t, err)
	a.redirect = "http://old.example.net/cb"
	e.consent(u, a, "read")
	w := e.do("GET", a.authorizeURL("read", ""), nil, e.asUser(u.ID))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, w.Header().Get("Location"))
}

func TestErrorAndConsentRedirectsEscapeState(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("state_user")
	a := e.newApp(appOpts{})
	evil := "x&error=none&code=stolen#frag?y=1"

	// An error redirect (a scope the app was not given).
	w := e.do("GET", a.authorizeURL("read tags:wrangle", "&state="+url.QueryEscape(evil)), nil, e.asUser(u.ID))
	require.Equal(t, http.StatusFound, w.Code)
	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, evil, loc.Query().Get("state"))
	assert.Equal(t, "invalid_scope", loc.Query().Get("error"))
	assert.Empty(t, loc.Query().Get("code"))
	assert.Empty(t, loc.Fragment)
	assert.Equal(t, a.redirect, loc.Scheme+"://"+loc.Host+loc.Path)

	// The consent screen's cancel link.
	cw := e.do("GET", a.authorizeURL("read", "&state="+url.QueryEscape(evil)), nil, e.asUser(u.ID))
	require.Equal(t, http.StatusOK, cw.Code, cw.Body.String())
	var body struct {
		Cancel string `json:"cancel_url"`
	}
	json.Unmarshal(cw.Body.Bytes(), &body)
	cu, err := url.Parse(body.Cancel)
	require.NoError(t, err)
	assert.Equal(t, evil, cu.Query().Get("state"))
	assert.Equal(t, "access_denied", cu.Query().Get("error"))
	assert.Len(t, cu.Query(), 2, "only error and state")
	assert.Empty(t, cu.Fragment)

	// The redirect with a code keeps an existing query on the registered address.
	assert.Equal(t, "https://x.example.org/cb?code=abc&keep=1&state=s%26t",
		e.as.buildCallbackURL("https://x.example.org/cb?keep=1", "abc", "s&t"))
}

// ---------------------------------------------------------------------
// Item 18: admin client API
// ---------------------------------------------------------------------

func TestAdminListBoundsAndInactiveClients(t *testing.T) {
	e := newHEnv(t)
	a := e.newApp(appOpts{})
	e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"is_active": false})
	e.newApp(appOpts{})

	for _, q := range []string{"limit=0", "limit=-5", "limit=abc", "page=0", "page=-1", "limit=100000", "page=99999999999999999999"} {
		w := e.do("GET", "/api/v1/auth/admin/oauth/clients?"+q, nil, e.asAdmin)
		require.Equal(t, http.StatusOK, w.Code, q)
		var out struct {
			Clients    []map[string]any `json:"clients"`
			Pagination struct{ Page, Limit, Pages int }
		}
		json.Unmarshal(w.Body.Bytes(), &out)
		assert.GreaterOrEqual(t, out.Pagination.Limit, 1, q)
		assert.LessOrEqual(t, out.Pagination.Limit, 100, q)
		assert.GreaterOrEqual(t, out.Pagination.Page, 1, q)
	}
	w := e.do("GET", "/api/v1/auth/admin/oauth/clients", nil, e.asAdmin)
	assert.Contains(t, w.Body.String(), a.id, "the switched-off app is listed")
	assert.Contains(t, w.Body.String(), `"is_active":false`)

	tw := e.do("GET", "/api/v1/auth/admin/oauth/tokens?limit=0&page=0", nil, e.asAdmin)
	assert.Equal(t, http.StatusOK, tw.Code)
	assert.Equal(t, http.StatusBadRequest, e.do("GET", "/api/v1/auth/admin/oauth/tokens?client_id=nope", nil, e.asAdmin).Code)
}

func TestAdminUpdateValidatesItsInput(t *testing.T) {
	e := newHEnv(t)
	a := e.newApp(appOpts{})
	put := func(body any) *httptest.ResponseRecorder {
		return e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, body)
	}
	for name, body := range map[string]any{
		"empty name":         map[string]any{"client_name": "  "},
		"long name":          map[string]any{"client_name": strings.Repeat("n", 101)},
		"long description":   map[string]any{"description": strings.Repeat("d", 501)},
		"javascript logo":    map[string]any{"logo_url": "javascript:alert(1)"},
		"http website":       map[string]any{"website": "http://plain.example.net"},
		"unknown scope":      map[string]any{"scopes": []string{"read", "nonsense"}},
		"no scopes":          map[string]any{"scopes": []string{}},
		"is_active string":   map[string]any{"is_active": "yes"},
		"ttl string":         map[string]any{"access_token_ttl": "1h"},
		"changing is_public": map[string]any{"is_public": true},
		"changing secret":    map[string]any{"client_secret": "x"},
		"changing owner":     map[string]any{"is_first_party": true},
		"nothing":            map[string]any{},
		"not an object":      []int{1},
	} {
		assert.Equal(t, http.StatusBadRequest, put(body).Code, name)
	}
	assert.Equal(t, http.StatusNotFound, e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+uuid.NewString(), map[string]any{"client_name": "x"}).Code)

	ok := put(map[string]any{"client_name": "Renamed", "description": "d", "website": "https://w.example.org",
		"scopes": []string{"read"}, "grant_types": []string{"authorization_code"}, "refresh_token_ttl": 7200})
	assert.Equal(t, http.StatusOK, ok.Code, ok.Body.String())
	var name string
	var refreshTTL int
	require.NoError(t, e.db.QueryRow(`SELECT client_name, refresh_token_ttl FROM oauth_clients WHERE client_id = $1`, a.id).Scan(&name, &refreshTTL))
	assert.Equal(t, "Renamed", name)
	assert.Equal(t, 7200, refreshTTL)

	// Taking a scope away takes it from tokens already issued.
	u := e.newUser("narrow_user")
	b := e.newApp(appOpts{scopes: []string{"read", "write"}})
	tr := e.signedIn(b, u, "read write")
	e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+b.id, map[string]any{"scopes": []string{"read"}})
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(tr.AccessToken))
	rw, _ := e.refresh(b, tr.RefreshToken, "")
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAdminClientActionsAreAudited(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("audit_user")

	actions := func() []string {
		rows, err := e.db.Query(`SELECT action FROM admin_actions WHERE target_type IN ('oauth_client', 'oauth_token') ORDER BY id`)
		require.NoError(t, err)
		defer rows.Close()
		var out []string
		for rows.Next() {
			var a string
			rows.Scan(&a)
			out = append(out, a)
		}
		return out
	}
	assert.Empty(t, actions())

	a := e.newApp(appOpts{}) // create
	assert.Equal(t, []string{"oauth_client.create"}, actions())

	e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"description": "hello"}) // update
	e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"is_trusted": true})     // update + trust
	e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"is_trusted": true})     // no change in trust
	e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"is_trusted": false})    // update + untrust
	e.adminJSON("POST", "/api/v1/auth/admin/oauth/clients/"+a.id+"/reset-secret", nil)                   // reset
	tr := e.signedIn(e.newApp(appOpts{}), u, "read")
	var tokenID string
	require.NoError(t, e.db.QueryRow(`SELECT id FROM oauth_access_tokens WHERE token_hash = $1`, hashSecret(tr.AccessToken)).Scan(&tokenID))
	e.adminJSON("DELETE", "/api/v1/auth/admin/oauth/tokens/"+tokenID, nil) // token revoke
	e.adminJSON("DELETE", "/api/v1/auth/admin/oauth/clients/"+a.id, nil)   // delete

	assert.Equal(t, []string{
		"oauth_client.create", // a
		"oauth_client.update", // description
		"oauth_client.update", "oauth_client.trust",
		"oauth_client.update",
		"oauth_client.update", "oauth_client.untrust",
		"oauth_client.reset_secret",
		"oauth_client.create", // the second app, made by signedIn
		"oauth_token.revoke",
		"oauth_client.delete",
	}, actions())

	// Rows name the admin and the client, and never hold a secret.
	var label, target, details string
	require.NoError(t, e.db.QueryRow(`SELECT actor_label, target_id, COALESCE(details::text, '') FROM admin_actions WHERE action = 'oauth_client.reset_secret'`).Scan(&label, &target, &details))
	assert.Equal(t, "client_admin", label)
	assert.Equal(t, a.id, target)
	assert.NotContains(t, details, a.secret)
	var withSecret int
	e.db.QueryRow(`SELECT count(*) FROM admin_actions WHERE details::text LIKE '%' || $1 || '%'`, a.secret).Scan(&withSecret)
	assert.Equal(t, 0, withSecret)

	// A change that fails writes no row.
	before := len(actions())
	e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"scopes": []string{"admin"}})
	assert.Equal(t, before, len(actions()))

	// Only admins reach these routes.
	w := e.do("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, nil, e.asUser(u.ID))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestTrustedAppsStillLeaveARecordedConsent(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("trusted_user")
	a := e.newApp(appOpts{})
	e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"is_trusted": true})

	// No consent screen, but the person's approval is on record, so they can
	// see and revoke it and a refresh needs it.
	// (Only for openid alone; see TestTrustedAppsStillNeedConsentForProfileAndEmail.)
	w := e.do("GET", a.authorizeURL("openid", ""), nil, e.asUser(u.ID))
	require.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, 1, e.count(`SELECT count(*) FROM user_consents WHERE user_id = $1 AND NOT is_revoked`, u.ID))
	tr := e.signedIn(a, u, "openid")
	rw, _ := e.refresh(a, tr.RefreshToken, "")
	assert.Equal(t, http.StatusOK, rw.Code)
}

func TestConsentScreenFlowStillWorks(t *testing.T) {
	e := newHEnv(t)
	u := e.newUser("screen_user")
	a := e.newApp(appOpts{})

	w := e.do("GET", a.authorizeURL("read profile", "&state=s1"), nil, e.asUser(u.ID))
	require.Equal(t, http.StatusOK, w.Code)
	var prompt struct {
		ID string `json:"consent_id"`
	}
	json.Unmarshal(w.Body.Bytes(), &prompt)
	require.NotEmpty(t, prompt.ID)

	// Deny
	dw := e.do("POST", "/auth/consent/"+prompt.ID, url.Values{"approved": {"false"}}, e.asUser(u.ID))
	require.Equal(t, http.StatusFound, dw.Code)
	assert.Contains(t, dw.Header().Get("Location"), "error=access_denied")

	// Approve, from a fresh screen
	w = e.do("GET", a.authorizeURL("read profile", "&state=s1"), nil, e.asUser(u.ID))
	json.Unmarshal(w.Body.Bytes(), &prompt)
	aw := e.do("POST", "/auth/consent/"+prompt.ID, url.Values{"approved": {"true"}}, e.asUser(u.ID))
	require.Equal(t, http.StatusFound, aw.Code)
	loc, _ := url.Parse(aw.Header().Get("Location"))
	code := loc.Query().Get("code")
	require.NotEmpty(t, code)
	xw, _ := e.exchange(a, code)
	assert.Equal(t, http.StatusOK, xw.Code)

	// An app switched off while the screen was open gets no code.
	e.db.Exec(`UPDATE user_consents SET is_revoked = true`)
	w = e.do("GET", a.authorizeURL("read", "&state=s2"), nil, e.asUser(u.ID))
	json.Unmarshal(w.Body.Bytes(), &prompt)
	e.db.Exec(`UPDATE oauth_clients SET is_active = false WHERE client_id = $1`, a.id)
	aw = e.do("POST", "/auth/consent/"+prompt.ID, url.Values{"approved": {"true"}}, e.asUser(u.ID))
	assert.Equal(t, http.StatusBadRequest, aw.Code)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM authorization_codes WHERE used_at IS NULL`))
}

// Tokens are sent only to the person's own browser session, never logged:
// this keeps the log lines of the OAuth code free of the raw values.
func TestOAuthSourceDoesNotLogSecrets(t *testing.T) {
	// A guard for review: the files that handle raw tokens must not print them.
	// (The handlers log errors only, never request values.)
	for _, f := range []string{"oauth_tokens.go", "oauth_handlers.go", "oauth_helpers.go", "oauth_clientauth.go", "oauth_admin.go"} {
		src := readFileForTest(t, f)
		for _, line := range strings.Split(src, "\n") {
			if !strings.Contains(line, "log.Printf") {
				continue
			}
			for _, bad := range []string{"access)", "refresh)", "code)", "secret)", "Secret)", "raw)", "token)"} {
				assert.False(t, strings.Contains(line, bad), "%s logs a secret value: %s", f, strings.TrimSpace(line))
			}
		}
	}
}
