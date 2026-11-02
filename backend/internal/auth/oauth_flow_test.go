package auth

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"nuclear-ao3/shared/email"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the website's side of the OAuth flow (audit phase 4): the JSON
// authorize route, the JSON consent answer, the connected-apps list, and the
// rule that an admin's decision applies to the version they looked at.

func jsonAuthorizeURL(a app, scope, extra string) string {
	return strings.Replace(a.authorizeURL(scope, extra), "/auth/authorize", "/api/v1/oauth/authorize", 1)
}

func TestAuthorizeJSONAnswers(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("json_flow_person")
	a := e.newApp(appOpts{})

	// Logged out: the website is told to send the person to login. Never a redirect.
	w := e.do("GET", jsonAuthorizeURL(a, "openid profile", ""), nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, w.Header().Get("Location"))

	// Errors that must not go to the app are shown on the page: no redirect.
	unknown := strings.Replace(jsonAuthorizeURL(a, "openid", ""), a.id, uuid.NewString(), 1)
	w = e.do("GET", unknown, nil, e.asUser(p.user.ID))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid_client", e.errCode(w))
	assert.Empty(t, w.Header().Get("Location"))
	badURI := strings.Replace(jsonAuthorizeURL(a, "openid", ""), url.QueryEscape(a.redirect), url.QueryEscape("https://evil.example.org/cb"), 1)
	w = e.do("GET", badURI, nil, e.asUser(p.user.ID))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid_request", e.errCode(w))
	assert.Empty(t, w.Header().Get("Location"))

	// An invalid request the spec says to report to the app: JSON with the redirect.
	w = e.do("GET", jsonAuthorizeURL(a, "openid admin", "&state=xyz"), nil, e.asUser(p.user.ID))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Empty(t, w.Header().Get("Location"))
	loc, err := url.Parse(e.body(w)["redirect"].(string))
	require.NoError(t, err)
	assert.Equal(t, "app"+strings.TrimPrefix(a.redirect[len("https://app"):], ""), strings.TrimPrefix(loc.Scheme+"://"+loc.Host+loc.Path, "https://"))
	assert.Equal(t, "invalid_scope", loc.Query().Get("error"))
	assert.Equal(t, "xyz", loc.Query().Get("state"))

	// First time: the consent screen, with the person's own pseuds and nothing about the account.
	w = e.do("GET", jsonAuthorizeURL(a, "openid profile email", ""), nil, e.asUser(p.user.ID))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Empty(t, w.Header().Get("Location"))
	consent := e.body(w)["consent"].(map[string]any)
	assert.NotEmpty(t, consent["consent_id"])
	assert.Equal(t, p.shownID.String(), consent["default_pseud_id"])
	assert.Len(t, consent["pseuds"], 2)
	assert.Equal(t, 3, len(consent["scopes"].([]any)))
	assert.Equal(t, "App 1", consent["app"].(map[string]any)["name"])
	for _, s := range p.accountStrings() {
		assert.NotContains(t, w.Body.String(), s)
	}

	// With an approval on file the answer is the redirect with a code.
	e.consent(p.user, a, "openid", "read")
	w = e.do("GET", jsonAuthorizeURL(a, "openid read", "&state=s1"), nil, e.asUser(p.user.ID))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	loc, err = url.Parse(e.body(w)["redirect"].(string))
	require.NoError(t, err)
	assert.NotEmpty(t, loc.Query().Get("code"))
	assert.Equal(t, "s1", loc.Query().Get("state"))
}

func TestConsentRoundTripGivesACode(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("round_trip_person")
	a := e.newApp(appOpts{})

	screen := func(scope string) string {
		w := e.do("GET", jsonAuthorizeURL(a, scope, "&state=st&prompt=consent"), nil, e.asUser(p.user.ID))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		return e.body(w)["consent"].(map[string]any)["consent_id"].(string)
	}
	answer := func(id string, body map[string]any) map[string]any {
		w := e.asJSON(p.user, "POST", "/api/v1/oauth/consent/"+id, body)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Empty(t, w.Header().Get("Location"), "JSON mode never sends an HTTP redirect")
		return e.body(w)
	}

	// Allow, the second pseud, email declined: a code, no email in what the app gets.
	id := screen("openid profile email")
	loc, _ := url.Parse(answer(id, map[string]any{"approved": true, "pseud_id": p.secondID.String(), "share_email": false})["redirect"].(string))
	code := loc.Query().Get("code")
	require.NotEmpty(t, code)
	assert.Equal(t, "st", loc.Query().Get("state"))
	xw, tr := e.exchange(a, code)
	require.Equal(t, http.StatusOK, xw.Code, xw.Body.String())
	assert.NotContains(t, strings.Fields(tr.Scope), "email")
	status, body := e.userinfoBody(tr.AccessToken)
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, p.secondID.String())
	assert.NotContains(t, body, p.user.Email)
	assert.NotContains(t, body, p.shownID.String())

	// Allow with email shared: the address is there.
	id = screen("openid profile email")
	loc, _ = url.Parse(answer(id, map[string]any{"approved": true, "share_email": true})["redirect"].(string))
	xw, tr = e.exchange(a, loc.Query().Get("code"))
	require.Equal(t, http.StatusOK, xw.Code, xw.Body.String())
	_, body = e.userinfoBody(tr.AccessToken)
	assert.Contains(t, body, p.user.Email)
	assert.Contains(t, body, p.shownID.String(), "no pseud chosen means the default")

	// The email choice defaults to no when left out.
	id = screen("email")
	loc, _ = url.Parse(answer(id, map[string]any{"approved": true})["redirect"].(string))
	assert.Equal(t, "access_denied", loc.Query().Get("error"), "only email was asked for and it was declined")

	// Deny: the app hears access_denied, with its state.
	id = screen("openid profile")
	loc, _ = url.Parse(answer(id, map[string]any{"approved": false})["redirect"].(string))
	assert.Equal(t, "access_denied", loc.Query().Get("error"))
	assert.Equal(t, "st", loc.Query().Get("state"))
	assert.Empty(t, loc.Query().Get("code"))

	// A screen can be answered once, only by its owner, and only with the person's own pseud.
	id = screen("openid profile")
	other := e.newPerson("round_trip_other")
	w := e.asJSON(other.user, "POST", "/api/v1/oauth/consent/"+id, map[string]any{"approved": true})
	assert.Equal(t, http.StatusForbidden, w.Code)
	w = e.asJSON(p.user, "POST", "/api/v1/oauth/consent/"+id, map[string]any{"approved": true, "pseud_id": other.shownID.String()})
	assert.Equal(t, http.StatusForbidden, w.Code)
	answer(id, map[string]any{"approved": true})
	w = e.asJSON(p.user, "POST", "/api/v1/oauth/consent/"+id, map[string]any{"approved": true})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Logged out answers are refused.
	w = e.do("POST", "/api/v1/oauth/consent/"+id, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestConnectedAppsIncludeRefreshOnlyAndConsentOnlyApps(t *testing.T) {
	e := newHEnv(t)
	p := e.newPerson("connected_person")
	other := e.newPerson("connected_other")

	withToken := e.newApp(appOpts{})
	refreshOnly := e.newApp(appOpts{})
	consentOnly := e.newApp(appOpts{})
	gone := e.newApp(appOpts{})

	trA := e.signedIn(withToken, p.user, "openid profile email")
	trR := e.signedIn(refreshOnly, p.user, "openid profile")
	e.consent(p.user, consentOnly, "openid", "read")
	e.signedIn(gone, p.user, "openid")
	e.signedIn(withToken, other.user, "openid") // someone else's connection never shows up

	// The refresh-only app: its access token is revoked, but the refresh token still works.
	_, err := e.db.Exec(`UPDATE oauth_access_tokens SET is_revoked = true WHERE client_id = $1`, refreshOnly.id)
	require.NoError(t, err)
	// The consent-only app has no tokens at all. The last app is disconnected.
	require.Equal(t, http.StatusOK, e.asJSON(p.user, "DELETE", "/api/v1/oauth/authorized-applications/"+gone.id, nil).Code)
	require.NotEmpty(t, trA.RefreshToken)

	w := e.asJSON(p.user, "GET", "/api/v1/oauth/authorized-applications", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	list := e.body(w)["applications"].([]any)
	byID := map[string]map[string]any{}
	for _, it := range list {
		m := it.(map[string]any)
		byID[m["client_id"].(string)] = m
	}
	require.Len(t, byID, 3, w.Body.String())
	assert.Contains(t, byID, withToken.id)
	assert.Contains(t, byID, refreshOnly.id)
	assert.Contains(t, byID, consentOnly.id)
	assert.NotContains(t, byID, gone.id)

	m := byID[withToken.id]
	assert.Equal(t, true, m["share_email"])
	assert.Equal(t, p.shownID.String(), m["pseud"].(map[string]any)["id"])
	assert.Equal(t, p.shownName, m["pseud"].(map[string]any)["name"])
	assert.ElementsMatch(t, []any{"openid", "profile", "email"}, m["scopes"])
	assert.NotNil(t, m["granted_at"])
	assert.NotNil(t, m["client_name"])
	assert.Nil(t, byID[consentOnly.id]["pseud"], "no profile scope means no pseud is shared")
	assert.Equal(t, false, byID[consentOnly.id]["share_email"])
	assert.ElementsMatch(t, []any{"openid", "read"}, byID[consentOnly.id]["scopes"])
	assert.ElementsMatch(t, []any{"openid", "profile"}, byID[refreshOnly.id]["scopes"])
	for _, s := range append(p.accountStrings(), other.accountStrings()...) {
		assert.NotContains(t, w.Body.String(), s)
	}

	// Disconnecting the refresh-only app uses the full revoke path.
	require.Equal(t, http.StatusOK, e.asJSON(p.user, "DELETE", "/api/v1/oauth/authorized-applications/"+refreshOnly.id, nil).Code)
	rw, _ := e.refresh(refreshOnly, trR.RefreshToken, "")
	assert.Equal(t, http.StatusBadRequest, rw.Code)
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM user_consents WHERE client_id = $1 AND user_id = $2 AND NOT is_revoked`, refreshOnly.id, p.user.ID))
	w = e.asJSON(p.user, "GET", "/api/v1/oauth/authorized-applications", nil)
	assert.Len(t, e.body(w)["applications"], 2)
	// The older address still works and answers the same.
	assert.Equal(t, http.StatusOK, e.do("GET", "/auth/authorized-applications", nil, e.asUser(p.user.ID)).Code)
}

// ---------------------------------------------------------------------
// Decisions apply to the version the admin looked at
// ---------------------------------------------------------------------

func (e *hEnv) urisOf(id string) []string {
	var s string
	require.NoError(e.t, e.db.QueryRow(`SELECT array_to_string(redirect_uris, ' ') FROM oauth_clients WHERE client_id = $1`, id).Scan(&s))
	return strings.Fields(s)
}

func TestApprovalNeedsTheRevisionTheAdminSaw(t *testing.T) {
	e := newHEnv(t)
	owner := e.newUser("rev_owner")
	a := e.selfApp(owner, appBody(501))

	// The queue shows the revision, and so do the old admin views.
	q := e.body(e.adminJSON("GET", "/api/v1/admin/apps", nil))["apps"].([]any)
	var seen float64
	for _, it := range q {
		if it.(map[string]any)["client_id"] == a.id {
			seen = it.(map[string]any)["revision"].(float64)
		}
	}
	require.NotZero(t, seen)
	assert.NotNil(t, e.body(e.adminJSON("GET", "/api/v1/auth/admin/oauth/clients/"+a.id, nil))["client"].(map[string]any)["revision"])
	assert.NotNil(t, e.body(e.adminJSON("GET", "/api/v1/auth/admin/oauth/clients", nil))["clients"].([]any)[0].(map[string]any)["revision"])

	// A decision without a revision is refused.
	w := e.adminJSON("POST", "/api/v1/admin/apps/"+a.id+"/approve", map[string]any{})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Between the queue read and the decision the owner edits the app.
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"redirect_uris": []string{"https://changed.example.org/cb"}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = e.reviewRev("approve", a.id, "", int(seen))
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "app_changed", e.errCode(w))
	assert.Equal(t, "pending", e.status(a.id), "a stale approval must not approve")
	assert.Equal(t, 0, e.auditCount("oauth_client.approve", a.id))

	// After looking again the same decision works, and the queue shows the new settings.
	require.Equal(t, http.StatusOK, e.review("approve", a.id, "").Code)
	assert.Equal(t, []string{"https://changed.example.org/cb"}, e.urisOf(a.id))

	// Stale revisions are refused for every action.
	old := e.revisionOf(a.id) - 1
	for _, act := range []string{"reject", "suspend", "reinstate", "approve-display", "reject-display"} {
		assert.Equal(t, http.StatusConflict, e.reviewRev(act, a.id, "n", old).Code, act)
	}
	assert.Equal(t, "approved", e.status(a.id))
}

func TestConcurrentEditAndApproveNeverApproveUnseenSettings(t *testing.T) {
	e := newHEnv(t)
	owner := e.newUser("race_owner")
	for round := 0; round < 8; round++ {
		b := appBody(600 + round)
		a := e.selfApp(owner, b)
		original := b["redirect_uris"].([]string)
		rev := e.revisionOf(a.id)

		var wg sync.WaitGroup
		var approve, edit int
		wg.Add(2)
		go func() {
			defer wg.Done()
			approve = e.reviewRev("approve", a.id, "", rev).Code
		}()
		go func() {
			defer wg.Done()
			edit = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{
				"redirect_uris": []string{"https://swapped.example.org/cb"}, "scopes": []string{"openid"}}).Code
		}()
		wg.Wait()

		assert.Equal(t, http.StatusOK, edit, "round %d", round)
		if e.status(a.id) == "approved" {
			assert.Equal(t, original, e.urisOf(a.id), "round %d: approved with settings the admin never saw", round)
		}
		if approve == http.StatusOK {
			// The approval won, then the edit sent the app back to review.
			assert.Equal(t, "pending", e.status(a.id), "round %d", round)
		} else {
			assert.Equal(t, http.StatusConflict, approve, "round %d", round)
			assert.Equal(t, "pending", e.status(a.id), "round %d", round)
		}
		// Then the edit is cleared: delete to stay under the per-person limit.
		require.Equal(t, http.StatusOK, e.asJSON(owner, "DELETE", "/api/v1/apps/"+a.id, nil).Code)
	}
}

func TestOwnerEditSetsPendingWhateverTheStatusExceptSuspended(t *testing.T) {
	e := newHEnv(t)
	owner := e.newUser("pending_edit_owner")
	a := e.selfApp(owner, appBody(701))
	// A pending app edited stays pending, and a sensitive edit says so explicitly.
	w := e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"scopes": []string{"openid"}})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pending", e.status(a.id))
	// Rejected becomes pending; suspended refuses.
	require.Equal(t, http.StatusOK, e.review("reject", a.id, "no").Code)
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"redirect_uris": []string{"https://again.example.org/cb"}})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pending", e.status(a.id))
	require.Equal(t, http.StatusOK, e.review("suspend", a.id, "no").Code)
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"redirect_uris": []string{"https://more.example.org/cb"}})
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "suspended", e.status(a.id))
	assert.Equal(t, []string{"https://again.example.org/cb"}, e.urisOf(a.id))
}

// ---------------------------------------------------------------------
// Display changes on an approved app wait for an admin
// ---------------------------------------------------------------------

func (e *hEnv) consentAppName(a app, u string, person idPerson) string {
	w := e.do("GET", jsonAuthorizeURL(a, "openid profile", "&prompt=consent"), nil, e.asUser(person.user.ID))
	require.Equal(e.t, http.StatusOK, w.Code, w.Body.String())
	return e.body(w)["consent"].(map[string]any)["app"].(map[string]any)["name"].(string)
}

func TestDisplayChangesOnAnApprovedAppWaitForAnAdmin(t *testing.T) {
	e := newHEnv(t)
	e.as.mail = &email.Config{FromName: "Test Archive", SiteURL: "https://archive.example.org"}
	owner := e.newUser("display_owner")
	p := e.newPerson("display_person")
	b := appBody(801)
	b["client_name"] = "Honest Reader"
	a := e.selfApp(owner, b)
	require.Equal(t, http.StatusOK, e.review("approve", a.id, "").Code)
	tr := e.signedIn(a, p.user, "openid profile")
	_, err := e.db.Exec(`UPDATE oauth_clients SET is_trusted = true WHERE client_id = $1`, a.id)
	require.NoError(t, err)

	// A rename does not reach the consent screen, and nobody is signed out.
	w := e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"client_name": "Official AO3 Support", "website": "https://lookalike.example.org"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, true, e.body(w)["display_review_pending"])
	assert.Equal(t, "approved", e.status(a.id))
	assert.Equal(t, "Honest Reader", e.nameOf(a.id))
	assert.Equal(t, "Honest Reader", e.consentAppName(a, "", p))
	assert.Equal(t, http.StatusOK, e.userinfoStatus(tr.AccessToken))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_clients WHERE client_id = $1 AND is_trusted`, a.id), "any edit drops trust")
	assert.Equal(t, 1, e.auditCount("oauth_client.owner_update", a.id))

	// The owner sees it as waiting; the admin queue shows old and new side by side.
	mine := e.body(e.asJSON(owner, "GET", "/api/v1/apps", nil))["apps"].([]any)[0].(map[string]any)
	assert.Equal(t, true, mine["display_review_pending"])
	assert.Equal(t, "Honest Reader", mine["client_name"])
	assert.Equal(t, "Official AO3 Support", mine["proposed"].(map[string]any)["client_name"])
	q := e.body(e.adminJSON("GET", "/api/v1/admin/apps?status=display", nil))["apps"].([]any)
	require.Len(t, q, 1)
	row := q[0].(map[string]any)
	assert.Equal(t, "Honest Reader", row["client_name"])
	assert.Equal(t, "Official AO3 Support", row["proposed"].(map[string]any)["client_name"])
	assert.Equal(t, "https://lookalike.example.org", row["proposed"].(map[string]any)["website"])

	// A second edit builds on the waiting proposal.
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"description": "now with words"})
	require.Equal(t, http.StatusOK, w.Code)
	row = e.body(e.adminJSON("GET", "/api/v1/admin/apps?status=display", nil))["apps"].([]any)[0].(map[string]any)
	assert.Equal(t, "Official AO3 Support", row["proposed"].(map[string]any)["client_name"])
	assert.Equal(t, "now with words", row["proposed"].(map[string]any)["description"])

	// Rejecting needs a note and the right revision; it discards the proposal.
	assert.Equal(t, http.StatusBadRequest, e.review("reject-display", a.id, "").Code)
	assert.Equal(t, http.StatusConflict, e.reviewRev("reject-display", a.id, "no", e.revisionOf(a.id)-1).Code)
	require.Equal(t, http.StatusOK, e.review("reject-display", a.id, "looks like impersonation").Code)
	assert.Equal(t, "Honest Reader", e.nameOf(a.id))
	assert.Equal(t, 0, e.count(`SELECT count(*) FROM oauth_clients WHERE client_id = $1 AND (display_review_pending OR proposed_name IS NOT NULL OR proposed_website IS NOT NULL)`, a.id))
	assert.Equal(t, "approved", e.status(a.id))
	assert.Equal(t, http.StatusOK, e.userinfoStatus(tr.AccessToken))
	assert.Equal(t, 1, e.auditCount("oauth_client.reject_display", a.id))
	assert.Equal(t, 1, e.count(`SELECT count(*) FROM jobs WHERE kind = 'email' AND payload->>'to' = $1 AND payload->>'subject' = 'Your app''s display changes were rejected' AND payload->>'text' LIKE '%impersonation%'`, owner.Email))
	// With nothing waiting, another decision has nothing to decide.
	assert.Equal(t, http.StatusConflict, e.review("approve-display", a.id, "").Code)

	// Approving applies the new details; the consent screen shows them from then on.
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"client_name": "Honest Reader 2", "logo_url": "https://logo.example.org/l.png"})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Honest Reader", e.consentAppName(a, "", p))
	require.Equal(t, http.StatusOK, e.review("approve-display", a.id, "").Code)
	assert.Equal(t, "Honest Reader 2", e.nameOf(a.id))
	assert.Equal(t, "Honest Reader 2", e.consentAppName(a, "", p))
	assert.Equal(t, "approved", e.status(a.id))
	assert.Equal(t, http.StatusOK, e.userinfoStatus(tr.AccessToken), "existing tokens keep working throughout")
	rw, _ := e.refresh(a, tr.RefreshToken, "")
	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, 1, e.auditCount("oauth_client.approve_display", a.id))
	assert.Equal(t, 1, e.count(`SELECT count(*) FROM jobs WHERE kind = 'email' AND payload->>'subject' = 'Your app''s display changes were approved'`))

	// Putting the old details back clears the proposal by itself.
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"client_name": "Something else"})
	require.Equal(t, http.StatusOK, w.Code)
	w = e.asJSON(owner, "PATCH", "/api/v1/apps/"+a.id, map[string]any{"client_name": "Honest Reader 2", "logo_url": "https://logo.example.org/l.png"})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, false, e.body(w)["display_review_pending"])

	// Non-admins can't decide.
	assert.Equal(t, http.StatusForbidden, e.asJSON(owner, "POST", "/api/v1/admin/apps/"+a.id+"/approve-display", map[string]any{"revision": 1}).Code)
}
