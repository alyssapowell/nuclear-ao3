package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nuclear-ao3/shared/email"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

// loginFrom posts a login as if it came from the given address. Using a
// different address per call keeps the per-IP limit out of the way of tests
// about the per-account lock.
func (e *sessionEnv) loginFrom(addr, login, password string) *httptest.ResponseRecorder {
	return e.postFrom(addr, "/login", map[string]string{"login": login, "password": password})
}

func (e *sessionEnv) postFrom(addr, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(body)
	req := httptest.NewRequest("POST", authAPI+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = addr
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

// guess makes n wrong-password logins, each from its own address.
func (e *sessionEnv) guess(login string, n int) {
	e.t.Helper()
	for i := 0; i < n; i++ {
		addr := fmt.Sprintf("10.1.%d.%d:1000", i/250, i%250+1)
		if w := e.loginFrom(addr, login, "wrong!"); w.Code != http.StatusUnauthorized {
			e.t.Fatalf("guess %d: status %d", i, w.Code)
		}
		// Failures are recorded in the background; wait so counts are exact.
		e.as.guard.pending.Wait()
	}
}

const otherAddr = "198.51.100.7:1000"

func (e *sessionEnv) attempts(name string) (n int, locked bool) {
	e.t.Helper()
	e.as.guard.pending.Wait()
	err := e.as.db.QueryRow(`SELECT failed_login_attempts, COALESCE(locked_until > now(), false) FROM users WHERE username = $1`, name).Scan(&n, &locked)
	if err != nil {
		e.t.Fatal(err)
	}
	return n, locked
}

func TestLoginLocksAfterTooManyFailures(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")

	e.guess("alice", maxFailedLogins-1)
	if n, locked := e.attempts("alice"); n != maxFailedLogins-1 || locked {
		t.Fatalf("before the limit: %d attempts, locked %v", n, locked)
	}
	// The next wrong guess locks it.
	e.guess("alice", 1)
	if _, locked := e.attempts("alice"); !locked {
		t.Fatal("account should be locked")
	}
	// The right password no longer works, by username or by email.
	for _, login := range []string{"alice", "alice@example.org"} {
		w := e.loginFrom(otherAddr, login, "secret1")
		if w.Code != http.StatusUnauthorized || w.Body.String() != `{"error":"invalid_credentials"}` {
			t.Errorf("locked login by %s: %d %s", login, w.Code, w.Body)
		}
	}
}

func TestLockExpires(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	e.guess("alice", maxFailedLogins)
	if w := e.loginFrom(otherAddr, "alice", "secret1"); w.Code != http.StatusUnauthorized {
		t.Fatalf("should be locked: %d", w.Code)
	}
	e.mustExec(`UPDATE users SET locked_until = now() - interval '1 second' WHERE username = 'alice'`)
	if w := e.loginFrom(otherAddr, "alice", "secret1"); w.Code != http.StatusOK {
		t.Fatalf("after the lock ran out: %d %s", w.Code, w.Body)
	}
	if n, locked := e.attempts("alice"); n != 0 || locked {
		t.Errorf("good login should clear everything: %d, %v", n, locked)
	}

	// After a lock runs out, a wrong password starts counting from zero.
	e.mustExec(`UPDATE users SET failed_login_attempts = $1, locked_until = now() - interval '1 second' WHERE username = 'alice'`, maxFailedLogins)
	e.guess("alice", 1)
	if n, locked := e.attempts("alice"); n != 1 || locked {
		t.Errorf("after an expired lock: %d attempts, locked %v", n, locked)
	}
}

func TestGoodLoginResetsCounter(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	e.guess("alice", 7)
	if n, _ := e.attempts("alice"); n != 7 {
		t.Fatalf("attempts %d", n)
	}
	if w := e.loginFrom(otherAddr, "alice", "secret1"); w.Code != http.StatusOK {
		t.Fatalf("login: %d", w.Code)
	}
	if n, _ := e.attempts("alice"); n != 0 {
		t.Errorf("attempts after a good login: %d", n)
	}
}

func TestPasswordResetClearsLock(t *testing.T) {
	e := newPasswordEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	e.guess("alice", maxFailedLogins)
	if _, locked := e.attempts("alice"); !locked {
		t.Fatal("should be locked")
	}
	// Asking for a reset works while locked.
	if w := e.postFrom(otherAddr, "/reset-password", map[string]string{"email": "alice@example.org"}); w.Code != http.StatusAccepted {
		t.Fatalf("reset request: %d", w.Code)
	}
	var link email.Message
	for _, m := range e.queued("alice@example.org") {
		if m.Subject == "Reset your password" {
			link = m
		}
	}
	tok := tokenIn(t, link, "/auth/reset-password")
	if w := e.postFrom(otherAddr, "/reset-password/confirm", map[string]string{"token": tok, "password": "newsecret"}); w.Code != http.StatusNoContent {
		t.Fatalf("confirm: %d %s", w.Code, w.Body)
	}
	if n, locked := e.attempts("alice"); n != 0 || locked {
		t.Errorf("after reset: %d attempts, locked %v", n, locked)
	}
	if w := e.loginFrom(otherAddr, "alice", "newsecret"); w.Code != http.StatusOK {
		t.Errorf("login with the new password: %d", w.Code)
	}
}

func TestLockedLooksLikeWrongPassword(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	e.addUser("bob", "bob@example.org", "secret2")
	e.guess("alice", maxFailedLogins)

	// Locked (right password), unlocked (wrong password) and unknown names
	// all get the same status, body and headers.
	cases := map[string]*httptest.ResponseRecorder{
		"locked, right password":   e.loginFrom(otherAddr, "alice", "secret1"),
		"locked, wrong password":   e.loginFrom(otherAddr, "alice", "wrong!"),
		"unlocked, wrong password": e.loginFrom(otherAddr, "bob", "wrong!"),
		"unknown name":             e.loginFrom(otherAddr, "nobody", "wrong!"),
	}
	want := cases["unlocked, wrong password"]
	for name, w := range cases {
		if w.Code != want.Code || w.Body.String() != want.Body.String() || fmt.Sprint(w.Header()) != fmt.Sprint(want.Header()) {
			t.Errorf("%s: %d %s %v differs from a wrong password: %d %s %v", name, w.Code, w.Body, w.Header(), want.Code, want.Body, want.Header())
		}
	}
	if want.Code != http.StatusUnauthorized {
		t.Errorf("status %d", want.Code)
	}
}

func TestUnknownNamesCreateNoRows(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	var before, jobsBefore int
	e.as.db.QueryRow(`SELECT count(*) FROM users`).Scan(&before)
	e.as.db.QueryRow(`SELECT count(*) FROM jobs`).Scan(&jobsBefore)
	e.guess("ghost", 4)
	e.guess("ghost@example.org", 2)

	var after, touched, jobs int
	e.as.db.QueryRow(`SELECT count(*) FROM users`).Scan(&after)
	e.as.db.QueryRow(`SELECT count(*) FROM users WHERE failed_login_attempts <> 0 OR locked_until IS NOT NULL`).Scan(&touched)
	e.as.db.QueryRow(`SELECT count(*) FROM jobs`).Scan(&jobs)
	if after != before || touched != 0 || jobs != jobsBefore {
		t.Errorf("users %d -> %d, changed rows %d, jobs %d", before, after, touched, jobs)
	}
}

func TestUnknownNamesLockInMemory(t *testing.T) {
	e := newSessionEnv(t)
	failures, lockouts := testutil.ToFloat64(loginFailures), testutil.ToFloat64(loginLockouts)
	for i := 0; i < maxFailedLogins+5; i++ {
		e.as.noteUnknownFailure("Ghost")
	}
	if got := testutil.ToFloat64(loginFailures) - failures; got != maxFailedLogins+5 {
		t.Errorf("failures counted: %v", got)
	}
	if got := testutil.ToFloat64(loginLockouts) - lockouts; got != 1 {
		t.Errorf("lockouts counted: %v, want 1", got)
	}
}

func TestUnknownNameMemoryIsCapped(t *testing.T) {
	e := newSessionEnv(t)
	for i := 0; i < maxUnknownNames+50; i++ {
		e.as.noteUnknownFailure(fmt.Sprintf("ghost%d", i))
	}
	if n := e.as.guard.unknown.size(); n > maxUnknownNames {
		t.Errorf("%d names remembered, cap is %d", n, maxUnknownNames)
	}
}

func TestLockEmailQueuedOnce(t *testing.T) {
	e := newPasswordEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	e.guess("alice", maxFailedLogins-1)
	if got := len(e.queued("alice@example.org")); got != 0 {
		t.Fatalf("%d emails before the lock", got)
	}
	e.guess("alice", 1)
	// Further guesses and a correct login while locked send nothing more.
	e.guess("alice", 3)
	e.loginFrom(otherAddr, "alice", "secret1")
	got := e.queued("alice@example.org")
	if len(got) != 1 {
		t.Fatalf("%d lock emails, want 1", len(got))
	}
	if got[0].Subject != "Repeated failed logins on your account" {
		t.Errorf("subject %q", got[0].Subject)
	}
	// A second lock the same day sends no second email, so repeated locking
	// can't flood the owner's inbox.
	e.mustExec(`UPDATE users SET locked_until = now() - interval '1 second' WHERE username = 'alice'`)
	e.guess("alice", maxFailedLogins)
	if _, locked := e.attempts("alice"); !locked {
		t.Fatal("the second round of guesses should lock the account again")
	}
	if got := len(e.queued("alice@example.org")); got != 1 {
		t.Errorf("%d emails after a second lock the same day, want 1", got)
	}
	// A day later, a new lock sends one again.
	e.mustExec(`UPDATE jobs SET created_at = now() - interval '25 hours' WHERE kind = 'email'`)
	e.mustExec(`UPDATE users SET locked_until = now() - interval '1 second' WHERE username = 'alice'`)
	e.guess("alice", maxFailedLogins)
	if got := len(e.queued("alice@example.org")); got != 2 {
		t.Errorf("%d emails after a lock the next day, want 2", got)
	}
}

// A wrong password on a real account and a guess at a name that doesn't
// exist must take the same time: counting the real account's failure
// happens after the answer, so it can't be timed.
func TestFailureCountingIsNotOnTheRequestPath(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("timed", "timed@example.org", "right-password")
	// Hold the user's row so a synchronous count would block the request.
	tx, err := e.as.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`SELECT 1 FROM users WHERE username = 'timed' FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 1)
	go func() { done <- e.loginFrom(otherAddr, "timed", "wrong").Code }()
	select {
	case code := <-done:
		if code != http.StatusUnauthorized {
			t.Errorf("status %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Error("the login waited on the failure count: its timing depends on the account existing")
	}
	tx.Rollback()
	if n, _ := e.attempts("timed"); n != 1 {
		t.Errorf("the failure wasn't counted afterwards: %d", n)
	}
}

func TestLockEmailQueuedOnceUnderConcurrentGuesses(t *testing.T) {
	e := newPasswordEnv(t)
	e.addUser("alice", "alice@example.org", "secret1")
	e.mustExec(`UPDATE users SET failed_login_attempts = $1 WHERE username = 'alice'`, maxFailedLogins-1)
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			e.loginFrom(fmt.Sprintf("10.2.0.%d:1", i+1), "alice", "wrong!")
		}(i)
	}
	for i := 0; i < 8; i++ {
		<-done
	}
	if got := len(e.queued("alice@example.org")); got != 1 {
		t.Errorf("%d lock emails, want 1", got)
	}
}

func TestAttemptLimitPerIP(t *testing.T) {
	e := newSessionEnv(t)
	// Hold the clock still: the window is aligned to the wall clock, so a
	// slow run that crossed a 5-minute boundary would reset it midway.
	e.as.guard.init()
	frozen := time.Now()
	e.as.guard.byIP.now = func() time.Time { return frozen }
	ip := "203.0.113.9:4000"
	for i := 0; i < attemptsPerIP; i++ {
		w := e.loginFrom(ip, fmt.Sprintf("user%d", i), "wrong!")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %d", i, w.Code)
		}
	}
	w := e.loginFrom(ip, "another", "wrong!")
	if w.Code != http.StatusTooManyRequests || !bytes.Contains(w.Body.Bytes(), []byte(`"rate_limit_exceeded"`)) {
		t.Fatalf("attempt %d: %d %s", attemptsPerIP+1, w.Code, w.Body)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("no Retry-After header")
	}
	// Another address is unaffected.
	if w := e.loginFrom("203.0.113.10:4000", "another", "wrong!"); w.Code != http.StatusUnauthorized {
		t.Errorf("other address: %d", w.Code)
	}
	// The other endpoints share the same count.
	for _, p := range []string{"/reset-password", "/resend-verification", "/verify-email", "/reset-password/confirm"} {
		if w := e.postFrom(ip, p, map[string]string{"email": "a@example.org", "token": "x", "password": "secret1"}); w.Code != http.StatusTooManyRequests {
			t.Errorf("%s: %d, want 429", p, w.Code)
		}
	}
}

func TestAttemptLimitPerIPv6Network(t *testing.T) {
	e := newSessionEnv(t)
	// Hold the clock still: the window is aligned to the wall clock, so a
	// slow run that crossed a 5-minute boundary would reset it midway.
	e.as.guard.init()
	frozen := time.Now()
	e.as.guard.byIP.now = func() time.Time { return frozen }
	// Twenty different addresses inside one /64 use up one allowance.
	for i := 0; i < attemptsPerIP; i++ {
		w := e.loginFrom(fmt.Sprintf("[2001:db8:1:2::%x]:4000", i+1), "nobody", "wrong!")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %d", i, w.Code)
		}
	}
	if w := e.loginFrom("[2001:db8:1:2:ffff::1]:4000", "nobody", "wrong!"); w.Code != http.StatusTooManyRequests {
		t.Errorf("same /64: %d, want 429", w.Code)
	}
	if w := e.loginFrom("[2001:db8:1:3::1]:4000", "nobody", "wrong!"); w.Code != http.StatusUnauthorized {
		t.Errorf("next /64: %d, want 401", w.Code)
	}
}

func TestAttemptLimitWindowEnds(t *testing.T) {
	e := newSessionEnv(t)
	e.as.guard.init()
	now := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)
	e.as.guard.byIP.now = func() time.Time { return now }
	ip := "203.0.113.9:4000"
	for i := 0; i < attemptsPerIP; i++ {
		e.loginFrom(ip, "nobody", "wrong!")
	}
	if w := e.loginFrom(ip, "nobody", "wrong!"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("%d", w.Code)
	}
	now = now.Add(attemptWindow)
	if w := e.loginFrom(ip, "nobody", "wrong!"); w.Code != http.StatusUnauthorized {
		t.Errorf("next window: %d", w.Code)
	}
}

// As in AO3, one login name gets at most 5 tries in 20 seconds, from any
// number of addresses, and the limit is the same for names that don't exist.
func TestAttemptLimitPerName(t *testing.T) {
	e := newSessionEnv(t)
	now := time.Now()
	e.as.guard.byName.now = func() time.Time { return now }
	e.addUser("target", "target@example.org", "right-password")

	for _, name := range []string{"target", "nobody-has-this-name"} {
		for i := 0; i < attemptsPerName; i++ {
			if w := e.loginFrom(fmt.Sprintf("198.51.100.%d", i+1), name, "wrong"); w.Code != 401 {
				t.Fatalf("%s attempt %d: %d, want 401", name, i+1, w.Code)
			}
		}
		w := e.loginFrom("203.0.113.9", name, "right-password")
		if w.Code != 429 || w.Header().Get("Retry-After") == "" {
			t.Errorf("%s: attempt %d from a new address got %d, want 429 with Retry-After", name, attemptsPerName+1, w.Code)
		}
	}
	// Names are counted case-insensitively.
	if w := e.loginFrom("203.0.113.10", "TARGET", "right-password"); w.Code != 429 {
		t.Errorf("changing the case got past the limit: %d", w.Code)
	}
	// After the window, the right password works again.
	now = now.Add(nameWindow + time.Second)
	if w := e.loginFrom("203.0.113.11", "target", "right-password"); w.Code != 200 {
		t.Errorf("after the window: %d, want 200", w.Code)
	}
}
