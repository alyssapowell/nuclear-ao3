package auth

import (
	"net/http"
	"strings"
	"testing"
)

// Registering makes the account's default pseud, named by the pen name given,
// in the same transaction as the account, and a refused registration leaves
// no pseud behind.
func TestRegisterCreatesDefaultPseud(t *testing.T) {
	e := newPasswordEnv(t)
	if c := e.register("Newcomer", "secret1"); c != 201 {
		t.Fatalf("register: %d", c)
	}
	rows, err := e.as.db.Query(`SELECT p.name, p.is_default FROM pseuds p JOIN users u ON u.id = p.user_id
		WHERE u.email = 'Newcomer@example.org'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var name string
		var def bool
		rows.Scan(&name, &def)
		n++
		if name != "Newcomer" || !def {
			t.Errorf("pseud = %q default=%v, want Newcomer and default", name, def)
		}
	}
	if n != 1 {
		t.Fatalf("%d pseuds, want 1", n)
	}
	var username *string
	e.as.db.QueryRow(`SELECT username FROM users WHERE email = 'newcomer@example.org'`).Scan(&username)
	if username != nil {
		t.Errorf("the account has a username %q", *username)
	}

	// A refused registration changes nothing.
	var before, after int
	e.as.db.QueryRow(`SELECT count(*) FROM pseuds`).Scan(&before)
	if c := e.status("POST", authAPI+"/register", "", map[string]string{"pen_name": "Fine", "email": "fine@example.org", "password": "x"}); c != 400 {
		t.Fatalf("weak password: %d, want 400", c)
	}
	e.as.db.QueryRow(`SELECT count(*) FROM pseuds`).Scan(&after)
	if before != after {
		t.Errorf("a refused registration left %d new pseuds", after-before)
	}
}

// Signing up with an address that already has an account looks exactly like
// signing up with a new one, creates nothing, and tells the owner by email.
func TestSignUpWithExistingEmailRevealsNothing(t *testing.T) {
	e := newPasswordEnv(t)
	fresh := e.do("POST", authAPI+"/register", "", map[string]string{
		"pen_name": "First", "email": "owner@example.org", "password": "secret1"})
	if fresh.Code != http.StatusCreated {
		t.Fatalf("first sign-up: %d %s", fresh.Code, fresh.Body)
	}
	// The owner activates, so there is a live account to protect.
	token := tokenIn(t, e.queued("owner@example.org")[0], "/auth/activate")
	if c := e.status("POST", authAPI+"/verify-email", "", map[string]string{"token": token, "password": "secret1"}); c != http.StatusNoContent {
		t.Fatalf("activate: %d", c)
	}
	var users, pseuds, tokens int
	count := func() {
		e.as.db.QueryRow(`SELECT (SELECT count(*) FROM users), (SELECT count(*) FROM pseuds),
			(SELECT count(*) FROM email_verification_tokens)`).Scan(&users, &pseuds, &tokens)
	}
	count()
	usersBefore, pseudsBefore, tokensBefore := users, pseuds, tokens

	// Same address, in another case and with spaces around it, and another
	// password and pen name.
	again := e.do("POST", authAPI+"/register", "", map[string]string{
		"pen_name": "Someone Else", "email": "  Owner@Example.org ", "password": "different1"})
	if again.Code != fresh.Code || again.Body.String() != fresh.Body.String() {
		t.Errorf("answers differ:\n new:      %d %s\n existing: %d %s", fresh.Code, fresh.Body, again.Code, again.Body)
	}
	count()
	if users != usersBefore || pseuds != pseudsBefore || tokens != tokensBefore {
		t.Errorf("sign-up with a taken address created something: users %d->%d, pseuds %d->%d, tokens %d->%d",
			usersBefore, users, pseudsBefore, pseuds, tokensBefore, tokens)
	}
	// The existing password still works and the new one doesn't.
	if c := e.status("POST", authAPI+"/login", "", map[string]string{"email": "owner@example.org", "password": "different1"}); c != http.StatusUnauthorized {
		t.Errorf("login with the stranger's password: %d", c)
	}
	if c := e.status("POST", authAPI+"/login", "", map[string]string{"email": "owner@example.org", "password": "secret1"}); c != http.StatusOK {
		t.Errorf("login with the owner's password: %d", c)
	}

	mails := e.queued("owner@example.org")
	if len(mails) != 2 {
		t.Fatalf("%d emails to the owner, want the activation and the notice: %+v", len(mails), mails)
	}
	notice := mails[1]
	if notice.Subject != "Someone tried to sign up with your email address" {
		t.Errorf("subject %q", notice.Subject)
	}
	if !strings.Contains(notice.Text, "Hello First!") || strings.Contains(notice.Text, "Someone Else") {
		t.Errorf("the notice should greet the owner by their pen name, and not name the stranger's: %q", notice.Text)
	}
	if strings.Contains(notice.Text, "token=") {
		t.Errorf("the notice carries a token: %q", notice.Text)
	}

	// A second try the same day sends no second notice.
	e.do("POST", authAPI+"/register", "", map[string]string{"pen_name": "Again", "email": "owner@example.org", "password": "different1"})
	if got := len(e.queued("owner@example.org")); got != 2 {
		t.Errorf("%d emails after a second attempt the same day, want still 2", got)
	}
}

// With mail turned off the answer is still the same and nothing is queued.
func TestSignUpWithExistingEmailWithoutMail(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("quiet", "quiet@example.org", "secret1")
	w := e.do("POST", authAPI+"/register", "", map[string]string{"pen_name": "x", "email": "quiet@example.org", "password": "secret1"})
	if w.Code != http.StatusCreated {
		t.Fatalf("sign-up: %d %s", w.Code, w.Body)
	}
	var n int
	e.as.db.QueryRow(`SELECT count(*) FROM jobs WHERE kind = 'email'`).Scan(&n)
	if n != 0 {
		t.Errorf("%d email jobs with mail off", n)
	}
}
