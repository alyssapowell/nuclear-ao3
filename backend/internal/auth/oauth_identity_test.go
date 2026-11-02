package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nuclear-ao3/shared/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for OAuth identity (security audit, phase 2): every app sees a
// different subject for the same person, and only the one pseud they chose.

// idPerson is a person with a default pseud that is not their username, and
// a second pseud, so a leak of either is easy to spot in raw JSON.
type idPerson struct {
	user        *models.User
	shownID     uuid.UUID // the default pseud
	shownName   string
	secondID    uuid.UUID
	secondName  string
	accountName string
}

func (e *hEnv) newPerson(accountName string) idPerson {
	e.t.Helper()
	u := e.newUser(accountName)
	tag := uuid.NewString()[:6]
	p := idPerson{user: u, shownName: "Shown Pen " + tag, secondName: "Hidden Pen " + tag, accountName: accountName}
	_, err := e.db.Exec(`UPDATE pseuds SET name = $1 WHERE user_id = $2 AND is_default`, p.shownName, u.ID)
	require.NoError(e.t, err)
	require.NoError(e.t, e.db.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1 AND is_default`, u.ID).Scan(&p.shownID))
	require.NoError(e.t, e.db.QueryRow(
		`INSERT INTO pseuds (user_id, name) VALUES ($1, $2) RETURNING id`, u.ID, p.secondName).Scan(&p.secondID))
	return p
}

// leaks are the strings that must never reach an app for this person.
func (p idPerson) accountStrings() []string {
	return []string{p.user.ID.String(), p.accountName, p.user.Email}
}

func (p idPerson) otherPseudStrings() []string {
	return []string{p.secondID.String(), p.secondName}
}

func (e *hEnv) userinfoBody(access string) (int, string) {
	w := e.do("GET", "/auth/userinfo", nil, func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+access) })
	return w.Code, w.Body.String()
}

func (e *hEnv) userinfoMap(access string) map[string]any {
	e.t.Helper()
	code, body := e.userinfoBody(access)
	require.Equal(e.t, http.StatusOK, code, body)
	var m map[string]any
	require.NoError(e.t, json.Unmarshal([]byte(body), &m))
	return m
}

// idTokenPayload returns the raw payload JSON and the parsed claims.
func idTokenPayload(t *testing.T, tok string) (string, map[string]any) {
	t.Helper()
	parts := strings.Split(tok, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return string(raw), m
}

func idTokenHeader(t *testing.T, tok string) map[string]any {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(strings.Split(tok, ".")[0])
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}

// screen starts an authorization that needs consent and returns the consent
// screen's JSON and its id.
func (e *hEnv) screen(a app, u *models.User, scope, extra string) (string, string) {
	e.t.Helper()
	w := e.do("GET", a.authorizeURL(scope, extra), nil, e.asUser(u.ID))
	require.Equal(e.t, http.StatusOK, w.Code, w.Body.String())
	var v struct {
		ID string `json:"consent_id"`
	}
	require.NoError(e.t, json.Unmarshal(w.Body.Bytes(), &v))
	require.NotEmpty(e.t, v.ID)
	return v.ID, w.Body.String()
}

// approve posts the person's answer to a consent screen.
func (e *hEnv) approve(id string, u *models.User, pseud string) *httpResult {
	form := url.Values{"approved": {"true"}}
	if pseud != "" {
		form.Set("pseud_id", pseud)
	}
	w := e.do("POST", "/auth/consent/"+id, form, e.asUser(u.ID))
	return &httpResult{Code: w.Code, Location: w.Header().Get("Location"), Body: w.Body.String()}
}

type httpResult struct {
	Code     int
	Location string
	Body     string
}

func (r *httpResult) code(t *testing.T) string {
	t.Helper()
	require.Equal(t, http.StatusFound, r.Code, r.Body)
	loc, err := url.Parse(r.Location)
	require.NoError(t, err)
	c := loc.Query().Get("code")
	require.NotEmpty(t, c, r.Location)
	return c
}

// signInThroughScreen asks for consent, picks a pseud ("" for the default)
// and returns the tokens.
func (e *hEnv) signInThroughScreen(a app, u *models.User, scope, pseud, extra string) models.TokenResponse {
	e.t.Helper()
	id, _ := e.screen(a, u, scope, extra)
	code := e.approve(id, u, pseud).code(e.t)
	w, tr := e.exchange(a, code)
	require.Equal(e.t, http.StatusOK, w.Code, w.Body.String())
	return tr
}

// ---------------------------------------------------------------------
// 1. Pairwise subjects
// ---------------------------------------------------------------------

func TestSubjectsArePairwiseAndStable(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("pairwise_person")
	a, b := e.newApp(appOpts{}), e.newApp(appOpts{})

	a1 := e.signedIn(a, p.user, "openid profile")
	a2 := e.signedIn(a, p.user, "openid profile") // a second login
	b1 := e.signedIn(b, p.user, "openid profile")

	subA1 := e.userinfoMap(a1.AccessToken)["sub"].(string)
	subA2 := e.userinfoMap(a2.AccessToken)["sub"].(string)
	subB := e.userinfoMap(b1.AccessToken)["sub"].(string)

	assert.Equal(t, subA1, subA2, "the same app gets the same subject on every login")
	assert.NotEqual(t, subA1, subB, "two apps get different subjects for one person")
	for _, s := range []string{subA1, subB} {
		assert.NotEqual(t, p.user.ID.String(), s)
		assert.Len(t, s, 43, "32 random bytes, base64url")
	}

	// The ID token and introspection agree with userinfo, per app.
	_, claimsA := idTokenPayload(t, a1.IDToken)
	_, claimsA2 := idTokenPayload(t, a2.IDToken)
	_, claimsB := idTokenPayload(t, b1.IDToken)
	assert.Equal(t, subA1, claimsA["sub"])
	assert.Equal(t, subA1, claimsA2["sub"])
	assert.Equal(t, subB, claimsB["sub"])

	// A refresh keeps it too.
	rw, rtr := e.refresh(a, a1.RefreshToken, "")
	require.Equal(t, http.StatusOK, rw.Code)
	_, claimsR := idTokenPayload(t, rtr.IDToken)
	assert.Equal(t, subA1, claimsR["sub"])

	assert.Equal(t, 2, e.count(`SELECT count(*) FROM oauth_subjects`))
}

func TestIntrospectionSubjectIsThePairwiseOne(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("introspect_person")
	a, b := e.newApp(appOpts{}), e.newApp(appOpts{})
	ta, tb := e.signedIn(a, p.user, "openid profile"), e.signedIn(b, p.user, "openid profile")

	introspectSub := func(who app, tok string) string {
		w := e.do("POST", "/auth/introspect", who.form(url.Values{"token": {tok}}), nil)
		require.Equal(t, http.StatusOK, w.Code)
		var m map[string]any
		json.Unmarshal(w.Body.Bytes(), &m)
		require.Equal(t, true, m["active"])
		s, _ := m["sub"].(string)
		return s
	}
	sa, sb := introspectSub(a, ta.AccessToken), introspectSub(b, tb.AccessToken)
	assert.Equal(t, e.userinfoMap(ta.AccessToken)["sub"], sa)
	assert.Equal(t, e.userinfoMap(tb.AccessToken)["sub"], sb)
	assert.NotEqual(t, sa, sb)
	assert.NotEqual(t, p.user.ID.String(), sa)
}

func TestDeletingAnAppDeletesItsSubjects(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("subject_delete_person")
	a, b, c := e.newApp(appOpts{}), e.newApp(appOpts{}), e.newApp(appOpts{})
	for _, x := range []app{a, b, c} {
		e.signedIn(x, p.user, "openid")
	}
	require.Equal(t, 3, e.count(`SELECT count(*) FROM oauth_subjects`))

	// The admin "delete" is a soft delete; it removes the subjects itself.
	w := e.adminJSON("DELETE", "/api/v1/auth/admin/oauth/clients/"+a.id, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_subjects WHERE client_id = $1`, a.id))
	assert.Equal(t, 2, e.count(`SELECT count(*) FROM oauth_subjects`))

	// Removing the row altogether cascades.
	_, err := e.db.Exec(`DELETE FROM oauth_clients WHERE client_id = $1`, b.id)
	require.NoError(t, err)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_subjects WHERE client_id = $1`, b.id))
}

func TestSubjectIndexesAreUnique(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("unique_person")
	q := e.newPerson("unique_person_2")
	a := e.newApp(appOpts{})
	_, err := e.db.Exec(`INSERT INTO oauth_subjects (client_id, user_id, sub) VALUES ($1, $2, $3)`,
		a.id, p.user.ID, strings.Repeat("a", 43))
	require.NoError(t, err)
	_, err = e.db.Exec(`INSERT INTO oauth_subjects (client_id, user_id, sub) VALUES ($1, $2, $3)`,
		a.id, p.user.ID, strings.Repeat("b", 43))
	assert.Error(t, err, "one subject per person per app")
	_, err = e.db.Exec(`INSERT INTO oauth_subjects (client_id, user_id, sub) VALUES ($1, $2, $3)`,
		a.id, q.user.ID, strings.Repeat("a", 43))
	assert.Error(t, err, "one person per subject per app")
}

// ---------------------------------------------------------------------
// 2. No account details
// ---------------------------------------------------------------------

func TestNoAccountDetailsReachApps(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("account_name_zz")
	a := e.newApp(appOpts{})
	// Some roles and stats, which must not show up either.
	e.db.Exec(`INSERT INTO user_roles (user_id, role) VALUES ($1, 'tag_wrangler')`, p.user.ID)

	// openid profile, no email scope: the email must stay out.
	tr := e.signInThroughScreen(a, p.user, "openid profile", "", "&nonce=n-1")
	code, userinfo := e.userinfoBody(tr.AccessToken)
	require.Equal(t, http.StatusOK, code)
	idRaw, idClaims := idTokenPayload(t, tr.IDToken)

	for name, raw := range map[string]string{"userinfo": userinfo, "id_token": idRaw} {
		for _, bad := range append(p.accountStrings(), p.otherPseudStrings()...) {
			assert.NotContains(t, raw, bad, "%s must not contain %q", name, bad)
		}
		for _, bad := range []string{"ao3_", "preferred_username", "updated_at", "/users/", "roles", "tag_wrangler", "work_count"} {
			assert.NotContains(t, raw, bad, "%s must not contain %q", name, bad)
		}
		assert.Contains(t, raw, p.shownName)
		assert.Contains(t, raw, p.shownID.String())
	}

	var info map[string]any
	json.Unmarshal([]byte(userinfo), &info)
	for k := range info {
		assert.Contains(t, []string{"sub", "name", "pseud", "email", "email_verified"}, k)
	}
	standard := []string{"iss", "sub", "aud", "azp", "exp", "iat", "auth_time", "nonce", "at_hash", "name", "pseud", "email", "email_verified"}
	for k := range idClaims {
		assert.Contains(t, standard, k)
	}
	assert.NotContains(t, info, "email")
	assert.NotContains(t, idClaims, "email")

	// openid alone: only the subject (plus the standard token claims).
	only := e.signedIn(e.newApp(appOpts{}), p.user, "openid")
	assert.Equal(t, []string{"sub"}, mapKeys(e.userinfoMap(only.AccessToken)))
	_, oc := idTokenPayload(t, only.IDToken)
	assert.NotContains(t, oc, "name")
	assert.NotContains(t, oc, "pseud")
	assert.NotContains(t, oc, "email")
}

func mapKeys(m map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// Timestamps can link too: the person's last login is the same in every app.
func TestAuthTimeIsNotTheLastLogin(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("authtime_person")
	long := time.Now().Add(-72 * time.Hour).Truncate(time.Second)
	_, err := e.db.Exec(`UPDATE users SET last_login_at = $1 WHERE id = $2`, long, p.user.ID)
	require.NoError(t, err)

	a := e.newApp(appOpts{})
	tr := e.signedIn(a, p.user, "openid")
	_, claims := idTokenPayload(t, tr.IDToken)
	at := int64(claims["auth_time"].(float64))
	assert.NotEqual(t, long.Unix(), at)
	assert.InDelta(t, time.Now().Unix(), at, 30)

	rw, rtr := e.refresh(a, tr.RefreshToken, "")
	require.Equal(t, http.StatusOK, rw.Code)
	_, rc := idTokenPayload(t, rtr.IDToken)
	assert.InDelta(t, time.Now().Unix(), int64(rc["auth_time"].(float64)), 30)
}

// ---------------------------------------------------------------------
// 3. The chosen pseud
// ---------------------------------------------------------------------

func TestTheChosenPseudIsWhatTheAppSees(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("chooser")
	a := e.newApp(appOpts{})

	// Choose the second pseud.
	tr := e.signInThroughScreen(a, p.user, "openid profile", p.secondID.String(), "")
	info := e.userinfoMap(tr.AccessToken)
	pseud := info["pseud"].(map[string]any)
	assert.Equal(t, p.secondID.String(), pseud["id"])
	assert.Equal(t, p.secondName, pseud["name"])
	assert.Equal(t, p.secondName, info["name"])
	raw, _ := json.Marshal(info)
	assert.NotContains(t, string(raw), p.shownName)
	assert.NotContains(t, string(raw), p.shownID.String())
	_, claims := idTokenPayload(t, tr.IDToken)
	assert.Equal(t, info["pseud"], claims["pseud"])

	// Re-consenting can change it.
	tr2 := e.signInThroughScreen(a, p.user, "openid profile", p.shownID.String(), "&prompt=consent")
	info2 := e.userinfoMap(tr2.AccessToken)
	assert.Equal(t, p.shownID.String(), info2["pseud"].(map[string]any)["id"])
	// The sub did not change.
	assert.Equal(t, info["sub"], info2["sub"])
}

func TestPseudDefaultsToTheDefaultPseud(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("defaulter")
	a := e.newApp(appOpts{})
	tr := e.signInThroughScreen(a, p.user, "openid profile", "", "")
	assert.Equal(t, p.shownID.String(), e.userinfoMap(tr.AccessToken)["pseud"].(map[string]any)["id"])
}

func TestAnotherPersonsPseudIsRefused(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("chooser2")
	other := e.newPerson("someone_else")
	a := e.newApp(appOpts{})

	id, _ := e.screen(a, p.user, "openid profile", "")
	r := e.approve(id, p.user, other.secondID.String())
	assert.Equal(t, http.StatusForbidden, r.Code)
	assert.NotContains(t, r.Body, other.secondName)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM authorization_codes`))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM user_consents`))
	// A mistake doesn't use the request up.
	assert.Equal(t, http.StatusFound, e.approve(id, p.user, p.secondID.String()).Code)
	// Nor does a made-up id.
	id2, _ := e.screen(a, p.user, "openid profile", "&prompt=consent")
	assert.Equal(t, http.StatusForbidden, e.approve(id2, p.user, uuid.NewString()).Code)
	assert.Equal(t, http.StatusBadRequest, e.approve(id2, p.user, "not-a-uuid").Code)
}

func TestADeletedChosenPseudGivesNoPseudAndTheAppMustAskAgain(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("deleter")
	a := e.newApp(appOpts{})
	tr := e.signInThroughScreen(a, p.user, "openid profile", p.secondID.String(), "")
	require.NotNil(t, e.userinfoMap(tr.AccessToken)["pseud"])

	_, err := e.db.Exec(`DELETE FROM pseuds WHERE id = $1`, p.secondID)
	require.NoError(t, err)

	// No pseud, and not the default one either.
	info := e.userinfoMap(tr.AccessToken)
	assert.NotContains(t, info, "pseud")
	assert.NotContains(t, info, "name")
	raw, _ := json.Marshal(info)
	assert.NotContains(t, string(raw), p.shownName)
	assert.NotContains(t, string(raw), p.shownID.String())

	// A refresh no longer works, and a new request shows the screen again.
	rw, _ := e.refresh(a, tr.RefreshToken, "")
	assert.Equal(t, http.StatusBadRequest, rw.Code)
	w := e.do("GET", a.authorizeURL("openid profile", ""), nil, e.asUser(p.user.ID))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"consent_required":true`)
}

// ---------------------------------------------------------------------
// 4. Email
// ---------------------------------------------------------------------

func TestEmailNeedsTheScopeAndExplicitConsent(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("emailer")
	a := e.newApp(appOpts{})

	// Approved with the email scope: email and email_verified appear.
	tr := e.signInThroughScreen(a, p.user, "openid profile email", "", "")
	info := e.userinfoMap(tr.AccessToken)
	assert.Equal(t, p.user.Email, info["email"])
	assert.Contains(t, info, "email_verified")
	_, claims := idTokenPayload(t, tr.IDToken)
	assert.Equal(t, p.user.Email, claims["email"])

	// A token without the email scope never gets it, even from an approval that has it.
	b := e.newApp(appOpts{})
	e.signInThroughScreen(b, p.user, "openid profile email", "", "")
	narrow := e.tokenFor(b, p.user, "openid profile")
	assert.NotContains(t, e.userinfoMap(narrow.AccessToken), "email")

	// Without a recorded yes to email, the scope alone gives nothing, and the
	// app has to ask again.
	_, err := e.db.Exec(`UPDATE user_consents SET share_email = false WHERE client_id = $1`, a.id)
	require.NoError(t, err)
	assert.NotContains(t, e.userinfoMap(tr.AccessToken), "email")
	w := e.do("GET", a.authorizeURL("openid profile email", ""), nil, e.asUser(p.user.ID))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"consent_required":true`)
	rw, _ := e.refresh(a, tr.RefreshToken, "")
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

// tokenFor gets tokens for scopes the person already approved.
func (e *hEnv) tokenFor(a app, u *models.User, scope string) models.TokenResponse {
	e.t.Helper()
	w := e.do("GET", a.authorizeURL(scope, ""), nil, e.asUser(u.ID))
	require.Equal(e.t, http.StatusFound, w.Code, w.Body.String())
	loc, _ := url.Parse(w.Header().Get("Location"))
	xw, tr := e.exchange(a, loc.Query().Get("code"))
	require.Equal(e.t, http.StatusOK, xw.Code, xw.Body.String())
	return tr
}

// ---------------------------------------------------------------------
// 5. Trusted apps
// ---------------------------------------------------------------------

func TestTrustedAppsStillNeedConsentForProfileAndEmail(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("trusting")
	a := e.newApp(appOpts{})
	w := e.adminJSON("PUT", "/api/v1/auth/admin/oauth/clients/"+a.id, map[string]any{"is_trusted": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// openid alone skips the screen.
	w = e.do("GET", a.authorizeURL("openid", ""), nil, e.asUser(p.user.ID))
	assert.Equal(t, http.StatusFound, w.Code)

	for _, scope := range []string{"openid profile", "openid email", "profile", "openid profile email", "read"} {
		w = e.do("GET", a.authorizeURL(scope, ""), nil, e.asUser(p.user.ID))
		assert.Equal(t, http.StatusOK, w.Code, scope)
		assert.Contains(t, w.Body.String(), `"consent_required":true`, scope)
	}
	// Nothing was recorded that would let profile or email through later.
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM user_consents WHERE pseud_id IS NOT NULL OR share_email`))

	// Once the person says yes, it works as for any app.
	tr := e.signInThroughScreen(a, p.user, "openid profile", "", "")
	assert.NotNil(t, e.userinfoMap(tr.AccessToken)["pseud"])
}

// ---------------------------------------------------------------------
// 6. ShowConsent
// ---------------------------------------------------------------------

func TestShowConsentIsForTheirOwnerOnly(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("screen_owner")
	other := e.newPerson("screen_nosy")
	a := e.newApp(appOpts{})
	id, authorizeBody := e.screen(a, p.user, "openid profile email", "")

	// Not signed in
	w := e.do("GET", "/auth/consent/"+id, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NotContains(t, w.Body.String(), p.user.ID.String())

	// Another person: the same answer as for a request that doesn't exist.
	w = e.do("GET", "/auth/consent/"+id, nil, e.asUser(other.user.ID))
	assert.Equal(t, http.StatusNotFound, w.Code)
	missing := e.do("GET", "/auth/consent/"+uuid.NewString(), nil, e.asUser(other.user.ID))
	assert.Equal(t, missing.Body.String(), w.Body.String())

	// The owner gets a screen made for their own eyes, with no account fields.
	w = e.do("GET", "/auth/consent/"+id, nil, e.asUser(p.user.ID))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	for name, body := range map[string]string{"ShowConsent": w.Body.String(), "authorize": authorizeBody} {
		for _, bad := range []string{p.user.ID.String(), p.accountName, p.user.Email, "user_id", "username",
			"client_secret", "$2a$", "secret"} {
			assert.NotContains(t, body, bad, "%s must not contain %q", name, bad)
		}
	}
	var v struct {
		ID  string `json:"consent_id"`
		App struct {
			Name    string `json:"name"`
			Website string `json:"website"`
			Logo    string `json:"logo_url"`
		} `json:"app"`
		Scopes []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"scopes"`
		Pseuds []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"pseuds"`
		Default string `json:"default_pseud_id"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &v))
	assert.Equal(t, id, v.ID)
	assert.Equal(t, "App 1", v.App.Name)
	require.Len(t, v.Scopes, 3)
	for _, s := range v.Scopes {
		assert.NotEmpty(t, s.Description, s.Name)
	}
	require.Len(t, v.Pseuds, 2)
	assert.Equal(t, p.shownID.String(), v.Pseuds[0].ID, "default first")
	assert.Equal(t, p.shownID.String(), v.Default)
	// It is a JSON API: no template is involved.
	assert.True(t, strings.HasPrefix(w.Header().Get("Content-Type"), "application/json"))

	// Another person's pseuds are never in it.
	assert.NotContains(t, w.Body.String(), other.shownName)
	assert.NotContains(t, readFileForTest(t, "oauth_identity.go"), "c.HTML")
	assert.NotContains(t, readFileForTest(t, "oauth_admin.go"), "c.HTML")
}

func TestConsentScreenOffersNoPseudsWithoutTheProfileScope(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("screen_noprofile")
	a := e.newApp(appOpts{})
	id, _ := e.screen(a, p.user, "read", "")
	w := e.do("GET", "/auth/consent/"+id, nil, e.asUser(p.user.ID))
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "pseuds")
	assert.NotContains(t, w.Body.String(), p.shownName)
}

// ---------------------------------------------------------------------
// 6b. Signing: ID tokens use their own key
// ---------------------------------------------------------------------

func jwksKey(t *testing.T, jwks map[string]any, kid string) *rsa.PublicKey {
	t.Helper()
	for _, k := range jwks["keys"].([]any) {
		m := k.(map[string]any)
		if m["kid"] != kid {
			continue
		}
		n, err := base64.RawURLEncoding.DecodeString(m["n"].(string))
		require.NoError(t, err)
		eb, err := base64.RawURLEncoding.DecodeString(m["e"].(string))
		require.NoError(t, err)
		return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(eb).Int64())}
	}
	return nil
}

func (e *hEnv) jwks() map[string]any {
	w := e.do("GET", "/auth/jwks", nil, nil)
	require.Equal(e.t, http.StatusOK, w.Code)
	var m map[string]any
	require.NoError(e.t, json.Unmarshal(w.Body.Bytes(), &m))
	return m
}

func TestIDTokenFieldsAndSeparateSigningKey(t *testing.T) {
	e := newHEnv(t)
	oidc, err := NewJWTManager("nuclear-ao3", newKey(t, 2048))
	require.NoError(t, err)
	e.as.oidc = oidc
	require.NotEqual(t, e.as.jwt.keyID, oidc.keyID)

	p := e.newPerson("signer")
	a := e.newApp(appOpts{})
	tr := e.signInThroughScreen(a, p.user, "openid profile", "", "&nonce=abc123")
	require.NotEmpty(t, tr.IDToken)

	header := idTokenHeader(t, tr.IDToken)
	assert.Equal(t, oidc.keyID, header["kid"], "signed with the OIDC key")
	assert.NotEqual(t, e.as.jwt.keyID, header["kid"], "not the login key")
	assert.Equal(t, "RS256", header["alg"])

	// It validates against the published keys, and the login key is not published.
	jwks := e.jwks()
	pub := jwksKey(t, jwks, header["kid"].(string))
	require.NotNil(t, pub)
	assert.Nil(t, jwksKey(t, jwks, e.as.jwt.keyID), "the login key must not be published")
	assert.Len(t, jwks["keys"], 1)
	parsed, err := jwt.Parse(tr.IDToken, func(*jwt.Token) (interface{}, error) { return pub, nil },
		jwt.WithAudience(a.id), jwt.WithIssuer("https://archive.example.org"), jwt.WithExpirationRequired())
	require.NoError(t, err)
	claims := parsed.Claims.(jwt.MapClaims)

	assert.Equal(t, a.id, claims["azp"])
	assert.Equal(t, atHash(tr.AccessToken), claims["at_hash"])
	assert.Equal(t, "abc123", claims["nonce"])
	assert.NotEmpty(t, claims["auth_time"])

	// The login side refuses it (unknown key, wrong audience).
	_, err = e.as.jwt.ValidateToken(tr.IDToken)
	assert.Error(t, err)

	// Without a nonce there is no nonce claim, and nothing is empty.
	tr2 := e.tokenFor(a, p.user, "openid profile")
	raw, c2 := idTokenPayload(t, tr2.IDToken)
	assert.NotContains(t, c2, "nonce")
	assert.NotContains(t, raw, `""`)
	assert.Equal(t, atHash(tr2.AccessToken), c2["at_hash"])
	assert.NotEqual(t, c2["at_hash"], claims["at_hash"])

	// A refresh carries the hash of the new access token.
	rw, rtr := e.refresh(a, tr.RefreshToken, "")
	require.Equal(t, http.StatusOK, rw.Code)
	_, rc := idTokenPayload(t, rtr.IDToken)
	assert.Equal(t, atHash(rtr.AccessToken), rc["at_hash"])
	assert.Equal(t, a.id, rc["azp"])
	assert.NotContains(t, rc, "nonce")
}

func TestWithoutAnOIDCKeyTheLoginKeyIsUsed(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("fallback_signer")
	a := e.newApp(appOpts{})
	tr := e.signedIn(a, p.user, "openid")
	assert.Equal(t, e.as.jwt.keyID, idTokenHeader(t, tr.IDToken)["kid"])
	assert.NotNil(t, jwksKey(t, e.jwks(), e.as.jwt.keyID))
}

func TestLoadOIDCManager(t *testing.T) {
	login, err := NewJWTManager("nuclear-ao3", newKey(t, 2048))
	require.NoError(t, err)

	// Unset: nil, the caller falls back to the login key.
	m, err := LoadOIDCManager(login, "nuclear-ao3", "")
	assert.NoError(t, err)
	assert.Nil(t, m)

	// A key made by "nuclear keys generate" works.
	path := filepath.Join(t.TempDir(), "oidc.pem")
	kid, err := GenerateKeyFile(path)
	require.NoError(t, err)
	m, err = LoadOIDCManager(login, "nuclear-ao3", path)
	require.NoError(t, err)
	assert.Equal(t, kid, m.keyID, "the kid is the RFC 7638 thumbprint")
	assert.Equal(t, KeyID(&m.privateKey.PublicKey), m.keyID)

	// Missing or unreadable files stop the server from starting.
	_, err = LoadOIDCManager(login, "nuclear-ao3", filepath.Join(t.TempDir(), "missing.pem"))
	assert.ErrorContains(t, err, "OIDC_SIGNING_KEY_FILE")
	bad := filepath.Join(t.TempDir(), "bad.pem")
	require.NoError(t, os.WriteFile(bad, []byte("not a key"), 0o600))
	_, err = LoadOIDCManager(login, "nuclear-ao3", bad)
	assert.ErrorContains(t, err, "OIDC_SIGNING_KEY_FILE")
}

func TestDiscoveryClaimsAndSubjectTypes(t *testing.T) {
	e := newHEnv(t)
	w := e.do("GET", "/.well-known/openid-configuration", nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var doc map[string]any
	json.Unmarshal(w.Body.Bytes(), &doc)
	assert.Equal(t, []any{"pairwise"}, doc["subject_types_supported"])
	claims := doc["claims_supported"].([]any)
	for _, gone := range []string{"preferred_username", "profile", "updated_at"} {
		assert.NotContains(t, claims, gone)
	}
	for _, want := range []string{"sub", "pseud", "name", "email", "email_verified", "at_hash", "azp"} {
		assert.Contains(t, claims, want)
	}
	assert.Equal(t, []any{"client_secret_basic", "client_secret_post", "none"}, doc["revocation_endpoint_auth_methods_supported"])
	assert.Equal(t, []any{"client_secret_basic", "client_secret_post"}, doc["introspection_endpoint_auth_methods_supported"])
}

// ---------------------------------------------------------------------
// 7. Revocation for public apps
// ---------------------------------------------------------------------

func TestPublicAppsCanRevokeTheirOwnTokens(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("public_revoker")
	pubA, pubB := e.newApp(appOpts{public: true}), e.newApp(appOpts{public: true})
	conf := e.newApp(appOpts{})
	revoke := func(who app, token string) int {
		return e.do("POST", "/auth/revoke", who.form(url.Values{"token": {token}}), nil).Code
	}

	trA := e.signedIn(pubA, p.user, "openid")
	// Another public app, and a confidential one, can't touch it (and are told OK).
	assert.Equal(t, http.StatusOK, revoke(pubB, trA.RefreshToken))
	assert.Equal(t, http.StatusOK, revoke(pubB, trA.AccessToken))
	assert.Equal(t, http.StatusOK, revoke(conf, trA.RefreshToken))
	assert.Equal(t, http.StatusOK, e.userinfoStatus(trA.AccessToken), "other apps' revokes changed nothing")

	// A public app that sends a secret is still refused.
	w := e.do("POST", "/auth/revoke", url.Values{"token": {trA.RefreshToken}, "client_id": {pubA.id}, "client_secret": {"made-up"}}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, http.StatusOK, e.userinfoStatus(trA.AccessToken))

	// Its own refresh token: revoked, and its access token goes with it.
	assert.Equal(t, http.StatusOK, revoke(pubA, trA.RefreshToken))
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(trA.AccessToken))
	rw, _ := e.refresh(pubA, trA.RefreshToken, "")
	assert.Equal(t, http.StatusBadRequest, rw.Code)

	// Its own access token too.
	trB := e.signedIn(pubA, p.user, "openid")
	assert.Equal(t, http.StatusOK, revoke(pubA, trB.AccessToken))
	assert.Equal(t, http.StatusUnauthorized, e.userinfoStatus(trB.AccessToken))

	// Introspection stays confidential-only.
	iw := e.do("POST", "/auth/introspect", pubA.form(url.Values{"token": {trB.AccessToken}}), nil)
	assert.Equal(t, http.StatusUnauthorized, iw.Code)
}
