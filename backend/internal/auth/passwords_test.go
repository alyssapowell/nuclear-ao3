package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"nuclear-ao3/shared/email"
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
		"username": name, "email": name + "@example.org", "password": password})
}

func (e *sessionEnv) mustExec(query string, args ...any) {
	e.t.Helper()
	if _, err := e.as.db.Exec(query, args...); err != nil {
		e.t.Fatal(err)
	}
}

func TestRegisterAndActivate(t *testing.T) {
	e := newPasswordEnv(t)
	w := e.do("POST", authAPI+"/register", "", map[string]string{"username": "newbie", "email": "newbie@example.org", "password": "secret1"})
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

	login := map[string]string{"login": "newbie", "password": "secret1"}
	w = e.do("POST", authAPI+"/login", "", login)
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "not_activated") {
		t.Fatalf("login before activation: %d %s", w.Code, w.Body)
	}
	wrong := e.do("POST", authAPI+"/login", "", map[string]string{"login": "newbie", "password": "nope-nope"})
	if wrong.Code != http.StatusUnauthorized {
		t.Errorf("wrong password on unactivated account: %d (must not reveal not_activated)", wrong.Code)
	}

	if c := e.status("POST", authAPI+"/verify-email", "", map[string]string{"token": token}); c != http.StatusNoContent {
		t.Fatalf("verify: %d", c)
	}
	if c := e.status("POST", authAPI+"/login", "", login); c != http.StatusOK {
		t.Errorf("login after activation: %d", c)
	}
	if c := e.status("POST", authAPI+"/verify-email", "", map[string]string{"token": token}); c != http.StatusBadRequest {
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
	if c := e.status("POST", authAPI+"/verify-email", "", map[string]string{"token": token}); c != http.StatusBadRequest {
		t.Errorf("expired token: %d", c)
	}
	if c := e.status("POST", authAPI+"/verify-email", "", map[string]string{"token": "garbage"}); c != http.StatusBadRequest {
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
	e.as.db.QueryRow(`SELECT count(*) FROM jobs`).Scan(&n)
	if n != 0 {
		t.Errorf("%d jobs", n)
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
		w := e.do("POST", authAPI+"/register", "", map[string]string{"username": name, "email": name + "@example.org", "password": tc.pw})
		if w.Code != tc.want {
			t.Errorf("password of %d: %d, want %d", len(tc.pw), w.Code, tc.want)
		}
		if tc.want == 400 && !strings.Contains(w.Body.String(), "weak_password") {
			t.Errorf("password of %d: %s", len(tc.pw), w.Body)
		}
	}
	if c := e.status("POST", authAPI+"/register", "", map[string]string{"username": "", "email": "x@example.org", "password": "secret1"}); c != http.StatusBadRequest {
		t.Errorf("no username: %d", c)
	}
	if c := e.status("POST", authAPI+"/register", "", map[string]string{"username": "someone", "email": "", "password": "secret1"}); c != http.StatusBadRequest {
		t.Errorf("no email: %d", c)
	}
	if c := e.register("userb40", "secret1"); c != http.StatusConflict {
		t.Errorf("duplicate: %d", c)
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
	if c := e.status("POST", authAPI+"/verify-email", "", map[string]string{"token": tokenIn(t, mails[1], "/auth/activate")}); c != http.StatusNoContent {
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
	if c := e.status("POST", authAPI+"/login", "", map[string]string{"login": "forgetful", "password": "old-password"}); c != http.StatusUnauthorized {
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
	e.as.db.QueryRow(`SELECT count(*) FROM jobs`).Scan(&n)
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
