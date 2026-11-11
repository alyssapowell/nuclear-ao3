package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"nuclear-ao3/shared/email"

	"github.com/google/uuid"
)

const authAPI = "/api/v1/auth"

func newPasswordEnv(t *testing.T) *sessionEnv {
	e := newSessionEnv(t)
	e.as.SetMail(&email.Config{FromName: "Test Archive", SiteURL: "https://archive.example.org"})
	return e
}

func (e *sessionEnv) status(method, path, token string, body any) int {
	return e.do(method, path, token, body).Code
}

// queued returns the emails waiting to be sent to address, oldest first.
func (e *sessionEnv) queued(address string) []email.Message {
	e.t.Helper()
	rows, err := e.as.db.Query(`SELECT payload FROM jobs WHERE kind = 'email' AND payload->>'to' = $1 ORDER BY id`, address)
	if err != nil {
		e.t.Fatal(err)
	}
	defer rows.Close()
	var out []email.Message
	for rows.Next() {
		var raw []byte
		var m email.Message
		if err := rows.Scan(&raw); err != nil {
			e.t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			e.t.Fatal(err)
		}
		out = append(out, m)
	}
	return out
}

// tokenIn pulls the token out of the link in an email's text.
func tokenIn(t *testing.T, m email.Message, path string) string {
	t.Helper()
	prefix := "https://archive.example.org" + path + "?token="
	i := strings.Index(m.Text, prefix)
	if i < 0 {
		t.Fatalf("no %s link in %q", path, m.Text)
	}
	rest := m.Text[i+len(prefix):]
	if j := strings.IndexAny(rest, " \n"); j >= 0 {
		rest = rest[:j]
	}
	tok, err := url.QueryUnescape(rest)
	if err != nil || len(tok) < 40 {
		t.Fatalf("bad token %q", rest)
	}
	return tok
}

func (e *sessionEnv) register(name, password string) int {
	return e.status("POST", authAPI+"/register", "", map[string]string{
		"pen_name": name, "email": name + "@example.org", "password": password})
}

func (e *sessionEnv) mustExec(query string, args ...any) {
	e.t.Helper()
	if _, err := e.as.db.Exec(query, args...); err != nil {
		e.t.Fatal(err)
	}
}

func TestRegisterAndActivate(t *testing.T) {
	e := newPasswordEnv(t)
	w := e.do("POST", authAPI+"/register", "", map[string]string{"pen_name": "newbie", "email": "newbie@example.org", "password": "secret1"})
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), "token") {
		t.Errorf("register must not return a token: %s", w.Body)
	}
	var n int
	e.as.db.QueryRow(`SELECT count(*) FROM user_sessions`).Scan(&n)
	if n != 0 {
		t.Errorf("register started %d sessions", n)
	}
	mails := e.queued("newbie@example.org")
	if len(mails) != 1 || mails[0].Subject != "Activate your account" {
		t.Fatalf("emails: %+v", mails)
	}
	token := tokenIn(t, mails[0], "/auth/activate")

	login := map[string]string{"email": "newbie@example.org", "password": "secret1"}
	w = e.do("POST", authAPI+"/login", "", login)
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "not_activated") {
		t.Fatalf("login before activation: %d %s", w.Code, w.Body)
	}
	wrong := e.do("POST", authAPI+"/login", "", map[string]string{"email": "newbie@example.org", "password": "nope-nope"})
	if wrong.Code != http.StatusUnauthorized {
		t.Errorf("wrong password on unactivated account: %d (must not reveal not_activated)", wrong.Code)
	}

	if c := e.status("POST", authAPI+"/verify-email", "", map[string]string{"token": token, "password": "secret1"}); c != http.StatusNoContent {
		t.Fatalf("verify: %d", c)
	}
	if c := e.status("POST", authAPI+"/login", "", login); c != http.StatusOK {
		t.Errorf("login after activation: %d", c)
	}
	if c := e.status("POST", authAPI+"/verify-email", "", map[string]string{"token": token, "password": "secret1"}); c != http.StatusBadRequest {
		t.Errorf("activation token reused: %d", c)
	}
	var stored string
	e.as.db.QueryRow(`SELECT token_hash FROM email_verification_tokens`).Scan(&stored)
	if stored == token || stored != hashToken(token) {
		t.Error("the token itself must not be stored")
	}
}

func TestActivationTokenExpires(t *testing.T) {
	e := newPasswordEnv(t)
	e.register("slowpoke", "secret1")
	token := tokenIn(t, e.queued("slowpoke@example.org")[0], "/auth/activate")
	e.mustExec(`UPDATE email_verification_tokens SET expires_at = now() - interval '1 second'`)
	if c := e.status("POST", authAPI+"/verify-email", "", map[string]string{"token": token, "password": "secret1"}); c != http.StatusBadRequest {
		t.Errorf("expired token: %d", c)
	}
	if c := e.status("POST", authAPI+"/verify-email", "", map[string]string{"token": "garbage", "password": "secret1"}); c != http.StatusBadRequest {
		t.Errorf("unknown token: %d", c)
	}
}

func TestRegisterWithoutMail(t *testing.T) {
	e := newSessionEnv(t) // mail is off
	if c := e.register("quiet", "secret1"); c != http.StatusCreated {
		t.Fatalf("register: %d", c)
	}
	var n int
	e.as.db.QueryRow(`SELECT count(*) FROM email_verification_tokens`).Scan(&n)
	if n != 1 {
		t.Errorf("%d tokens", n)
	}
	e.as.db.QueryRow(`SELECT count(*) FROM jobs WHERE kind = 'email'`).Scan(&n)
	if n != 0 {
		t.Errorf("%d jobs", n)
	}
}

func TestTrialSignUpIsConfirmedAtOnce(t *testing.T) {
	e := newPasswordEnv(t)
	e.as.SetTrialNoEmail(true)
	body := map[string]string{"pen_name": "tryer", "email": "tryer@example.org", "password": "secret1"}
	fresh := e.do("POST", authAPI+"/register", "", body)
	if fresh.Code != http.StatusCreated || !strings.Contains(fresh.Body.String(), "Your account is ready") {
		t.Fatalf("register: %d %s", fresh.Code, fresh.Body)
	}
	var verified bool
	e.as.db.QueryRow(`SELECT is_verified FROM users WHERE email = 'tryer@example.org'`).Scan(&verified)
	if !verified {
		t.Error("trial account is not verified")
	}
	var tokens, jobs int
	e.as.db.QueryRow(`SELECT count(*) FROM email_verification_tokens`).Scan(&tokens)
	e.as.db.QueryRow(`SELECT count(*) FROM jobs WHERE kind = 'email'`).Scan(&jobs)
	if tokens != 0 || jobs != 0 {
		t.Errorf("%d activation tokens and %d email jobs, want none", tokens, jobs)
	}
	if c := e.status("POST", authAPI+"/login", "", map[string]string{"email": "tryer@example.org", "password": "secret1"}); c != http.StatusOK {
		t.Errorf("login right after sign-up: %d", c)
	}

	// A taken address gets the very same answer.
	taken := e.do("POST", authAPI+"/register", "", body)
	if taken.Code != fresh.Code || taken.Body.String() != fresh.Body.String() {
		t.Errorf("taken: %d %s, new: %d %s", taken.Code, taken.Body, fresh.Code, fresh.Body)
	}
	other := e.do("POST", authAPI+"/register", "", map[string]string{"pen_name": "tryer2", "email": "tryer2@example.org", "password": "secret1"})
	if other.Code != fresh.Code || other.Body.String() != fresh.Body.String() {
		t.Errorf("second new account: %d %s", other.Code, other.Body)
	}
}

func TestRegisterValidation(t *testing.T) {
	e := newPasswordEnv(t)
	for _, tc := range []struct {
		pw   string
		want int
	}{
		{strings.Repeat("a", 5), http.StatusBadRequest},
		{strings.Repeat("a", 6), http.StatusCreated},
		{strings.Repeat("b", 40), http.StatusCreated},
		{strings.Repeat("c", 41), http.StatusBadRequest},
	} {
		name := "user" + string(tc.pw[0]) + string(rune('0'+len(tc.pw)/10)) + string(rune('0'+len(tc.pw)%10))
		w := e.do("POST", authAPI+"/register", "", map[string]string{"pen_name": name, "email": name + "@example.org", "password": tc.pw})
		if w.Code != tc.want {
			t.Errorf("password of %d: %d, want %d", len(tc.pw), w.Code, tc.want)
		}
		if tc.want == 400 && !strings.Contains(w.Body.String(), "weak_password") {
			t.Errorf("password of %d: %s", len(tc.pw), w.Body)
		}
	}
	if c := e.status("POST", authAPI+"/register", "", map[string]string{"pen_name": "", "email": "x@example.org", "password": "secret1"}); c != http.StatusBadRequest {
		t.Errorf("no pen name: %d", c)
	}
	if c := e.status("POST", authAPI+"/register", "", map[string]string{"email": "x@example.org", "password": "secret1"}); c != http.StatusBadRequest {
		t.Errorf("pen name missing altogether: %d", c)
	}
	if c := e.status("POST", authAPI+"/register", "", map[string]string{"pen_name": "someone", "email": "", "password": "secret1"}); c != http.StatusBadRequest {
		t.Errorf("no email: %d", c)
	}
}

func TestResendThrottle(t *testing.T) {
	e := newPasswordEnv(t)
	e.register("again", "secret1")
	resend := func(address string) int {
		return e.status("POST", authAPI+"/resend-verification", "", map[string]string{"email": address})
	}
	if c := resend("again@example.org"); c != http.StatusAccepted {
		t.Fatalf("resend: %d", c)
	}
	if n := len(e.queued("again@example.org")); n != 1 {
		t.Fatalf("resend right after sign-up sent %d emails, want 1 in total", n)
	}
	e.mustExec(`UPDATE email_verification_tokens SET created_at = now() - interval '6 minutes'`)
	resend("again@example.org")
	resend("again@example.org")
	mails := e.queued("again@example.org")
	if len(mails) != 2 {
		t.Fatalf("%d emails, want 2", len(mails))
	}
	if c := resend("nobody@example.org"); c != http.StatusAccepted {
		t.Errorf("unknown address: %d", c)
	}
	// An activated account gets nothing.
	if c := e.status("POST", authAPI+"/verify-email", "", map[string]string{"token": tokenIn(t, mails[1], "/auth/activate"), "password": "secret1"}); c != http.StatusNoContent {
		t.Fatalf("verify: %d", c)
	}
	e.mustExec(`UPDATE email_verification_tokens SET created_at = now() - interval '1 hour'`)
	resend("again@example.org")
	if n := len(e.queued("again@example.org")); n != 2 {
		t.Errorf("activated account got another email (%d)", n)
	}
}

func TestPasswordResetFlow(t *testing.T) {
	e := newPasswordEnv(t)
	e.addUser("forgetful", "forgetful@example.org", "old-password")
	t1 := e.login("forgetful", "old-password")
	t2 := e.login("forgetful", "old-password")

	reset := func(address string) int {
		return e.status("POST", authAPI+"/reset-password", "", map[string]string{"email": address})
	}
	if c := reset("forgetful@example.org"); c != http.StatusAccepted {
		t.Fatalf("reset: %d", c)
	}
	reset("Forgetful@example.org") // within 5 minutes: throttled
	mails := e.queued("forgetful@example.org")
	if len(mails) != 1 || mails[0].Subject != "Reset your password" {
		t.Fatalf("emails: %+v", mails)
	}
	token := tokenIn(t, mails[0], "/auth/reset-password")

	confirm := func(token, pw string) int {
		return e.status("POST", authAPI+"/reset-password/confirm", "", map[string]string{"token": token, "password": pw})
	}
	if c := confirm(token, "short"); c != http.StatusBadRequest {
		t.Errorf("weak password: %d", c)
	}
	if c := confirm(token, "new-password"); c != http.StatusNoContent {
		t.Fatalf("confirm: %d", c)
	}
	for _, tok := range []string{t1, t2} {
		if c := e.status("GET", authAPI+"/sessions", tok, nil); c != http.StatusUnauthorized {
			t.Errorf("old session still works: %d", c)
		}
	}
	if c := e.status("POST", authAPI+"/login", "", map[string]string{"email": "forgetful@example.org", "password": "old-password"}); c != http.StatusUnauthorized {
		t.Errorf("old password: %d", c)
	}
	e.login("forgetful", "new-password")
	if c := confirm(token, "another-one"); c != http.StatusBadRequest {
		t.Errorf("reused token: %d", c)
	}
}

func TestPasswordResetEdges(t *testing.T) {
	e := newPasswordEnv(t)
	if c := e.status("POST", authAPI+"/reset-password", "", map[string]string{"email": "ghost@example.org"}); c != http.StatusAccepted {
		t.Errorf("unknown address: %d", c)
	}
	var n int
	e.as.db.QueryRow(`SELECT count(*) FROM jobs WHERE kind = 'email'`).Scan(&n)
	if n != 0 {
		t.Errorf("%d jobs queued for an unknown address", n)
	}

	// Expired link; and a reset also activates an unactivated account.
	e.register("lapsed", "secret1")
	e.mustExec(`UPDATE email_verification_tokens SET created_at = now() - interval '1 hour'`)
	e.status("POST", authAPI+"/reset-password", "", map[string]string{"email": "lapsed@example.org"})
	token := tokenIn(t, e.queued("lapsed@example.org")[1], "/auth/reset-password")
	confirm := func() int {
		return e.status("POST", authAPI+"/reset-password/confirm", "", map[string]string{"token": token, "password": "fresh-pass"})
	}
	e.mustExec(`UPDATE password_reset_tokens SET expires_at = now() - interval '1 second'`)
	if c := confirm(); c != http.StatusBadRequest {
		t.Errorf("expired reset token: %d", c)
	}
	e.mustExec(`UPDATE password_reset_tokens SET expires_at = now() + interval '1 hour'`)
	if c := confirm(); c != http.StatusNoContent {
		t.Fatalf("confirm: %d", c)
	}
	e.login("lapsed", "fresh-pass")
}

func TestChangePassword(t *testing.T) {
	e := newPasswordEnv(t)
	e.addUser("changer", "changer@example.org", "old-password")
	phone := e.login("changer", "old-password")
	laptop := e.login("changer", "old-password")
	path := authAPI + "/change-password"

	if c := e.status("POST", path, "", map[string]string{"current_password": "old-password", "new_password": "new-password"}); c != http.StatusUnauthorized {
		t.Errorf("logged out: %d", c)
	}
	w := e.do("POST", path, laptop, map[string]string{"current_password": "wrong", "new_password": "new-password"})
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "wrong_password") {
		t.Errorf("wrong current: %d %s", w.Code, w.Body)
	}
	for _, pw := range []string{"12345", strings.Repeat("x", 41)} {
		w = e.do("POST", path, laptop, map[string]string{"current_password": "old-password", "new_password": pw})
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "weak_password") {
			t.Errorf("new password of %d: %d %s", len(pw), w.Code, w.Body)
		}
	}
	if c := e.status("POST", path, laptop, map[string]string{"current_password": "old-password", "new_password": "123456"}); c != http.StatusNoContent {
		t.Fatalf("change to 6 characters: %d", c)
	}
	if c := e.status("POST", path, laptop, map[string]string{"current_password": "123456", "new_password": strings.Repeat("x", 40)}); c != http.StatusNoContent {
		t.Fatalf("change to 40 characters: %d", c)
	}
	if c := e.status("GET", authAPI+"/sessions", laptop, nil); c != http.StatusOK {
		t.Errorf("current session: %d", c)
	}
	if c := e.status("GET", authAPI+"/sessions", phone, nil); c != http.StatusUnauthorized {
		t.Errorf("other session: %d", c)
	}
	e.login("changer", strings.Repeat("x", 40))
}

// Someone who signs up with another person's address can't have that
// person activate the account for them: activating needs the password.
func TestActivationNeedsThePassword(t *testing.T) {
	e := newPasswordEnv(t)
	e.register("squatter", "attackers-pw")
	token := tokenIn(t, e.queued("squatter@example.org")[0], "/auth/activate")
	w := e.do("POST", authAPI+"/verify-email", "", map[string]string{"token": token, "password": "owners-guess"})
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "wrong_password") {
		t.Fatalf("wrong password: %d %s", w.Code, w.Body)
	}
	var verified bool
	e.as.db.QueryRow(`SELECT is_verified FROM users WHERE email = 'squatter@example.org'`).Scan(&verified)
	if verified {
		t.Fatal("activated with the wrong password")
	}
	// A wrong password doesn't use up the link.
	if c := e.status("POST", authAPI+"/verify-email", "", map[string]string{"token": token, "password": "attackers-pw"}); c != http.StatusNoContent {
		t.Errorf("right password after a wrong one: %d", c)
	}
}

// The pen name is the name of the first pseud, so it follows the rules for a
// pseud name; it may be anything a pseud may be, including what would once
// have been a username.
func TestPenNameFollowsPseudRules(t *testing.T) {
	e := newPasswordEnv(t)
	i := 0
	for name, want := range map[string]int{
		"": http.StatusBadRequest, "   ": http.StatusBadRequest, "a@b.org": http.StatusBadRequest,
		"Anonymous Creator": http.StatusBadRequest, "anonymouscreator": http.StatusBadRequest,
		"!!!": http.StatusBadRequest, "---": http.StatusBadRequest, strings.Repeat("a", 41): http.StatusBadRequest,
		"a": http.StatusCreated, "ab": http.StatusCreated, "has space": http.StatusCreated, "under_score9": http.StatusCreated,
		"_lead": http.StatusCreated, "Ünïcode 日本語": http.StatusCreated, strings.Repeat("b", 40): http.StatusCreated,
	} {
		i++
		w := e.do("POST", authAPI+"/register", "", map[string]string{
			"pen_name": name, "email": fmt.Sprintf("pen%d@example.org", i), "password": "secret1"})
		if w.Code != want {
			t.Errorf("pen name %q: %d, want %d (%s)", name, w.Code, want, w.Body)
		}
		if want == http.StatusBadRequest && !strings.Contains(w.Body.String(), "invalid_pen_name") {
			t.Errorf("pen name %q: %s", name, w.Body)
		}
	}
}

// Changing or resetting a password also cuts off OAuth apps and cancels
// reset links sent earlier.
func TestPasswordChangeRevokesAppsAndResetLinks(t *testing.T) {
	e := newPasswordEnv(t)
	e.addUser("connected", "connected@example.org", "old-password")
	tok := e.login("connected", "old-password")
	var userID uuid.UUID
	e.as.db.QueryRow(`SELECT id FROM users WHERE email = 'connected@example.org'`).Scan(&userID)
	client := uuid.New()
	e.mustExec(`INSERT INTO oauth_clients (client_id, client_name, redirect_uris, scopes) VALUES ($1, 'Reader app', '{https://app.example}', '{read}')`, client)
	addAppToken := func() {
		e.mustExec(`INSERT INTO oauth_access_tokens (id, token_hash, user_id, client_id, scopes, expires_at, family_id)
			VALUES ($1, $2, $3, $4, '{read}', now() + interval '1 hour', $5)`, uuid.New(), hashSecret(uuid.NewString()), userID, client, uuid.New())
	}
	liveAppTokens := func() (n int) {
		e.as.db.QueryRow(`SELECT
			(SELECT count(*) FROM oauth_access_tokens WHERE user_id = $1 AND NOT is_revoked) +
			(SELECT count(*) FROM authorization_codes WHERE user_id = $1 AND used_at IS NULL) +
			(SELECT count(*) FROM user_consents WHERE user_id = $1 AND NOT is_revoked)`, userID).Scan(&n)
		return
	}
	e.mustExec(`INSERT INTO authorization_codes (code_hash, client_id, user_id, redirect_uri, scopes, code_challenge, code_challenge_method, expires_at)
		VALUES ($3, $1, $2, 'https://app.example', '{read}', $4, 'S256', now() + interval '10 minutes')`, client, userID, hashSecret("unexchanged"), testChallenge)
	e.mustExec(`INSERT INTO user_consents (id, user_id, client_id, scopes, granted_at) VALUES ($1, $2, $3, '{read}', now())`,
		uuid.New(), userID, client)

	addAppToken()
	e.status("POST", authAPI+"/reset-password", "", map[string]string{"email": "connected@example.org"})
	resetLink := tokenIn(t, e.queued("connected@example.org")[0], "/auth/reset-password")
	if c := e.status("POST", authAPI+"/change-password", tok, map[string]string{"current_password": "old-password", "new_password": "new-password"}); c != http.StatusNoContent {
		t.Fatalf("change: %d", c)
	}
	if n := liveAppTokens(); n != 0 {
		t.Errorf("%d app tokens still work after a password change", n)
	}
	if c := e.status("POST", authAPI+"/reset-password/confirm", "", map[string]string{"token": resetLink, "password": "sneaky-pass"}); c != http.StatusBadRequest {
		t.Errorf("a reset link from before the change still worked: %d", c)
	}

	addAppToken()
	e.mustExec(`UPDATE password_reset_tokens SET created_at = now() - interval '10 minutes'`)
	e.status("POST", authAPI+"/reset-password", "", map[string]string{"email": "connected@example.org"})
	mails := e.queued("connected@example.org")
	link := tokenIn(t, mails[len(mails)-1], "/auth/reset-password")
	if c := e.status("POST", authAPI+"/reset-password/confirm", "", map[string]string{"token": link, "password": "reset-pass"}); c != http.StatusNoContent {
		t.Fatalf("reset: %d", c)
	}
	if n := liveAppTokens(); n != 0 {
		t.Errorf("%d app tokens still work after a reset", n)
	}
}
