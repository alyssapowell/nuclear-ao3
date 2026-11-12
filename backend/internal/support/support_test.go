package support_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/platform"
	"nuclear-ao3/internal/support"
	"nuclear-ao3/internal/testdb"
)

type env struct {
	t      *testing.T
	db     *sql.DB
	router *gin.Engine
	tokens map[string]uuid.UUID
}

func newEnv(t *testing.T, supportEmail string, opts ...support.Option) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testdb.New(t)
	e := &env{t: t, db: db, tokens: map[string]uuid.UUID{}}
	deps := platform.NewDeps(db, func(tok string) (uuid.UUID, error) {
		if id, ok := e.tokens[tok]; ok {
			return id, nil
		}
		return uuid.Nil, sql.ErrNoRows
	})
	e.router = gin.New()
	support.Mount(e.router, support.New(db, supportEmail), deps, opts...)
	return e
}

func (e *env) user(name string, roles ...string) (uuid.UUID, string) {
	e.t.Helper()
	id := uuid.New()
	exec := func(q string, a ...any) {
		if _, err := e.db.Exec(q, a...); err != nil {
			e.t.Fatal(err)
		}
	}
	exec(`INSERT INTO users (id, email, password_hash, is_active, is_verified) VALUES ($1, $2, 'x', true, true)`, id, name+"@example.org")
	exec(`INSERT INTO pseuds (user_id, name, is_default) VALUES ($1, $2, true)`, id, name)
	for _, r := range roles {
		exec(`INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, id, r)
	}
	e.tokens["token-"+name] = id
	return id, "token-" + name
}

func (e *env) do(method, path, token string, body any) (int, string) {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

func (e *env) expect(want int, method, path, token string, body any) string {
	e.t.Helper()
	code, out := e.do(method, path, token, body)
	if code != want {
		e.t.Fatalf("%s %s as %q: status %d, want %d: %s", method, path, token, code, want, out)
	}
	return out
}

func (e *env) count(q string, a ...any) int {
	var n int
	if err := e.db.QueryRow(q, a...).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

// emails returns the queued email jobs as to -> subject lines.
func (e *env) emails() map[string][]string {
	rows, err := e.db.Query(`SELECT payload FROM jobs WHERE kind = 'email' ORDER BY id`)
	if err != nil {
		e.t.Fatal(err)
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var p []byte
		rows.Scan(&p)
		var m struct{ To, Subject, Text string }
		json.Unmarshal(p, &m)
		out[m.To] = append(out[m.To], m.Subject+"\n"+m.Text)
	}
	return out
}

func (e *env) ticketID() string {
	var id string
	if err := e.db.QueryRow(`SELECT id FROM support_tickets ORDER BY created_at DESC LIMIT 1`).Scan(&id); err != nil {
		e.t.Fatal(err)
	}
	return id
}

func TestGuestSubmits(t *testing.T) {
	e := newEnv(t, "help@archive.example")
	out := e.expect(201, "POST", "/api/v1/support", "", map[string]any{
		"kind": "abuse", "subject": "  A   problem ", "body": "Line one\r\nLine two\x00\x07",
		"email": "guest@example.org", "url": "https://archive.example/works/1",
	})
	if strings.Contains(out, "guest@example.org") || strings.Contains(out, "id") {
		t.Errorf("response says too much: %s", out)
	}
	var kind, subject, body, to, url string
	var user sql.NullString
	if err := e.db.QueryRow(`SELECT kind, subject, body, reporter_email, user_id::text, url FROM support_tickets`).
		Scan(&kind, &subject, &body, &to, &user, &url); err != nil {
		t.Fatal(err)
	}
	if kind != "abuse" || subject != "A problem" || body != "Line one\nLine two" || to != "guest@example.org" || user.Valid {
		t.Errorf("stored %q %q %q %q %v", kind, subject, body, to, user)
	}
	em := e.emails()
	if len(em["help@archive.example"]) != 1 || !strings.Contains(em["help@archive.example"][0], "guest@example.org") ||
		!strings.HasPrefix(em["help@archive.example"][0], "[Policy & Abuse] A problem") {
		t.Errorf("support email: %v", em["help@archive.example"])
	}
	if len(em["guest@example.org"]) != 1 {
		t.Errorf("confirmation: %v", em)
	}
}

func TestNoSupportEmailStillStoresAndConfirms(t *testing.T) {
	e := newEnv(t, "")
	e.expect(201, "POST", "/api/v1/support", "", map[string]any{"subject": "Hi", "body": "Hello", "email": "g@example.org"})
	if n := e.count(`SELECT count(*) FROM jobs WHERE kind = 'email'`); n != 1 {
		t.Errorf("%d emails queued, want only the confirmation", n)
	}
	if e.count(`SELECT count(*) FROM support_tickets WHERE kind = 'support'`) != 1 {
		t.Error("kind should default to support")
	}
}

func TestLoggedInUsesAccountEmail(t *testing.T) {
	e := newEnv(t, "help@archive.example")
	id, tok := e.user("alice")
	out := e.expect(201, "POST", "/api/v1/support", tok, map[string]any{
		"subject": "Hi", "body": "Hello", "email": "someone.else@example.org"})
	if strings.Contains(out, "alice@example.org") || strings.Contains(out, id.String()) {
		t.Errorf("response leaks the account: %s", out)
	}
	var to string
	var user uuid.UUID
	if err := e.db.QueryRow(`SELECT reporter_email, user_id FROM support_tickets`).Scan(&to, &user); err != nil {
		t.Fatal(err)
	}
	if to != "alice@example.org" || user != id {
		t.Errorf("stored %q %v", to, user)
	}
	if len(e.emails()["alice@example.org"]) != 1 || len(e.emails()["someone.else@example.org"]) != 0 {
		t.Errorf("confirmation went to the wrong place: %v", e.emails())
	}
}

func TestValidation(t *testing.T) {
	e := newEnv(t, "", support.WithSendLimit(1000, time.Hour))
	ok := map[string]any{"subject": "S", "body": "B", "email": "g@example.org"}
	with := func(k string, v any) map[string]any {
		m := map[string]any{}
		for a, b := range ok {
			m[a] = b
		}
		m[k] = v
		return m
	}
	for name, body := range map[string]map[string]any{
		"no subject":     with("subject", "  "),
		"long subject":   with("subject", strings.Repeat("a", 201)),
		"no body":        with("body", ""),
		"long body":      with("body", strings.Repeat("é", 10001)),
		"bad kind":       with("kind", "other"),
		"no email":       with("email", ""),
		"bad email":      with("email", "not an email"),
		"display name":   with("email", "Bob <bob@example.org>"),
		"header inject":  with("email", "a@example.org\r\nBcc: b@example.org"),
		"javascript url": with("url", "javascript:alert(1)"),
		"relative url":   with("url", "/works/1"),
	} {
		t.Run(name, func(t *testing.T) { e.expect(400, "POST", "/api/v1/support", "", body) })
	}
	e.expect(201, "POST", "/api/v1/support", "", with("body", strings.Repeat("é", 10000)))
	e.expect(400, "POST", "/api/v1/support", "", strings.Repeat("x", 100))
	if e.count(`SELECT count(*) FROM support_tickets`) != 1 {
		t.Error("only the valid ticket should be stored")
	}
}

func TestBodyIsStoredAsTypedNotAsHTML(t *testing.T) {
	e := newEnv(t, "")
	e.expect(201, "POST", "/api/v1/support", "", map[string]any{
		"subject": "<b>x</b>", "body": "<script>alert(1)</script>", "email": "g@example.org"})
	var s, b string
	e.db.QueryRow(`SELECT subject, body FROM support_tickets`).Scan(&s, &b)
	if s != "<b>x</b>" || b != "<script>alert(1)</script>" {
		t.Errorf("stored %q %q", s, b)
	}
}

func TestRateLimit(t *testing.T) {
	e := newEnv(t, "", support.WithSendLimit(2, time.Hour))
	m := map[string]any{"subject": "S", "body": "B", "email": "g@example.org"}
	e.expect(201, "POST", "/api/v1/support", "", m)
	e.expect(201, "POST", "/api/v1/support", "", m)
	e.expect(429, "POST", "/api/v1/support", "", m)
	// A refused request stores nothing and queues nothing. (The two kept
	// messages share an address, so they get one confirmation between them.)
	if e.count(`SELECT count(*) FROM support_tickets`) != 2 || e.count(`SELECT count(*) FROM jobs WHERE kind = 'email'`) != 1 {
		t.Error("a limited request left something behind")
	}
}

func TestAdminAccessAndFlow(t *testing.T) {
	e := newEnv(t, "")
	_, plain := e.user("plain")
	modID, mod := e.user("mod", "moderator")
	_, adm := e.user("adm", "admin")
	reporter, _ := e.user("rep")
	_ = reporter
	e.expect(201, "POST", "/api/v1/support", "token-rep", map[string]any{"subject": "Broken", "body": "It broke"})
	e.expect(201, "POST", "/api/v1/support", "", map[string]any{"kind": "abuse", "subject": "Bad", "body": "Bad thing", "email": "g@example.org"})
	id := e.ticketID()

	for _, tok := range []string{"", plain} {
		want := 403
		if tok == "" {
			want = 401
		}
		e.expect(want, "GET", "/api/v1/admin/support", tok, nil)
		e.expect(want, "GET", "/api/v1/admin/support/"+id, tok, nil)
		e.expect(want, "POST", "/api/v1/admin/support/"+id+"/notes", tok, map[string]any{"body": "x"})
		e.expect(want, "POST", "/api/v1/admin/support/"+id+"/close", tok, nil)
		e.expect(want, "POST", "/api/v1/admin/support/"+id+"/reopen", tok, nil)
	}

	var list struct {
		Total   int
		Tickets []struct {
			ID, Kind, Status string
			Body             string
			ReporterEmail    string `json:"reporter_email"`
		}
	}
	json.Unmarshal([]byte(e.expect(200, "GET", "/api/v1/admin/support", mod, nil)), &list)
	if list.Total != 2 || len(list.Tickets) != 2 || list.Tickets[0].Body != "" {
		t.Fatalf("list: %+v", list)
	}
	json.Unmarshal([]byte(e.expect(200, "GET", "/api/v1/admin/support?kind=abuse&status=open", adm, nil)), &list)
	if list.Total != 1 || list.Tickets[0].Kind != "abuse" {
		t.Fatalf("filtered list: %+v", list)
	}
	e.expect(400, "GET", "/api/v1/admin/support?status=weird", mod, nil)

	var tk struct {
		Body  string
		Notes []struct{ Kind, Body, Author string }
	}
	json.Unmarshal([]byte(e.expect(200, "GET", "/api/v1/admin/support/"+id, mod, nil)), &tk)
	if tk.Body != "Bad thing" {
		t.Errorf("ticket: %+v", tk)
	}
	e.expect(404, "GET", "/api/v1/admin/support/"+uuid.NewString(), mod, nil)

	before := e.count(`SELECT count(*) FROM jobs`)
	e.expect(201, "POST", "/api/v1/admin/support/"+id+"/notes", mod, map[string]any{"body": "internal only"})
	if e.count(`SELECT count(*) FROM jobs`) != before {
		t.Error("an internal note must not send email")
	}
	e.expect(201, "POST", "/api/v1/admin/support/"+id+"/notes", mod, map[string]any{"body": "We looked into it", "reply": true})
	if got := e.emails()["g@example.org"]; len(got) != 2 || !strings.HasPrefix(got[1], "Re: Bad") {
		t.Errorf("reply email: %v", got)
	}
	e.expect(400, "POST", "/api/v1/admin/support/"+id+"/notes", mod, map[string]any{"body": " "})
	e.expect(404, "POST", "/api/v1/admin/support/"+uuid.NewString()+"/notes", mod, map[string]any{"body": "x"})

	e.expect(200, "POST", "/api/v1/admin/support/"+id+"/close", mod, map[string]any{"reason": "handled"})
	e.expect(409, "POST", "/api/v1/admin/support/"+id+"/close", mod, nil)
	var closedBy uuid.UUID
	e.db.QueryRow(`SELECT closed_by FROM support_tickets WHERE id = $1`, id).Scan(&closedBy)
	if closedBy != modID {
		t.Errorf("closed_by = %v", closedBy)
	}
	e.expect(200, "GET", "/api/v1/admin/support?status=closed", mod, nil)
	e.expect(200, "POST", "/api/v1/admin/support/"+id+"/reopen", adm, nil)
	e.expect(409, "POST", "/api/v1/admin/support/"+id+"/reopen", adm, nil)

	// Every change is in the audit log: note, reply, close, reopen.
	for action, want := range map[string]int{"support_note": 1, "support_reply": 1, "support_close": 1, "support_reopen": 1} {
		if n := e.count(`SELECT count(*) FROM admin_actions WHERE action = $1 AND target_type = 'support_ticket' AND target_id = $2`, action, id); n != want {
			t.Errorf("%s: %d audit rows, want %d", action, n, want)
		}
	}
	if n := e.count(`SELECT count(*) FROM admin_actions WHERE details::text LIKE '%internal only%'`); n != 0 {
		t.Error("note text must not be copied into the audit log")
	}
	// Failed changes leave no audit row.
	if n := e.count(`SELECT count(*) FROM admin_actions`); n != 4 {
		t.Errorf("%d audit rows, want 4", n)
	}
}

func TestNotesAreAppendOnly(t *testing.T) {
	e := newEnv(t, "")
	_, mod := e.user("mod", "moderator")
	e.expect(201, "POST", "/api/v1/support", "", map[string]any{"subject": "S", "body": "B", "email": "g@example.org"})
	id := e.ticketID()
	e.expect(201, "POST", "/api/v1/admin/support/"+id+"/notes", mod, map[string]any{"body": "n"})
	if _, err := e.db.Exec(`UPDATE support_ticket_notes SET body = 'changed'`); err == nil {
		t.Error("UPDATE was allowed")
	}
	if _, err := e.db.Exec(`DELETE FROM support_ticket_notes`); err == nil {
		t.Error("DELETE was allowed")
	}
}

func TestDeletingTheAccountKeepsTheTicket(t *testing.T) {
	e := newEnv(t, "")
	id, tok := e.user("gone")
	e.expect(201, "POST", "/api/v1/support", tok, map[string]any{"subject": "S", "body": "B"})
	if _, err := e.db.Exec(`DELETE FROM users WHERE id = $1`, id); err != nil {
		t.Fatalf("deleting the account: %v", err)
	}
	if e.count(`SELECT count(*) FROM support_tickets WHERE user_id IS NULL`) != 1 {
		t.Error("ticket should remain, without the account")
	}
}

// The confirmation goes to an address nobody has checked, so it must not
// carry anything the sender wrote.
func TestConfirmationRepeatsNothingTheSenderWrote(t *testing.T) {
	e := newEnv(t, "help@archive.example")
	e.expect(201, "POST", "/api/v1/support", "", map[string]any{
		"subject": "Cheap pills at spam.example", "body": "Visit spam.example today", "email": "stranger@example.org",
	})
	got := e.emails()["stranger@example.org"]
	if len(got) != 1 {
		t.Fatalf("confirmation: %v", got)
	}
	if strings.Contains(got[0], "spam.example") || strings.Contains(got[0], "pills") {
		t.Errorf("the confirmation repeats the message:\n%s", got[0])
	}
}

func TestRateLimitCannotBeDodged(t *testing.T) {
	e := newEnv(t, "", support.WithSendLimit(2, time.Hour))
	from := func(addr, to string) int {
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(map[string]any{"subject": "S", "body": "B", "email": to})
		req := httptest.NewRequest("POST", "/api/v1/support", &buf)
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = addr
		w := httptest.NewRecorder()
		e.router.ServeHTTP(w, req)
		return w.Code
	}
	// One IPv6 connection has a whole /64 to itself: new addresses in it
	// are the same visitor.
	if from("[2001:db8:1:2::1]:1000", "a@example.org") != 201 || from("[2001:db8:1:2::2]:1000", "b@example.org") != 201 {
		t.Fatal("first two refused")
	}
	if code := from("[2001:db8:1:2:ffff::9]:1000", "c@example.org"); code != 429 {
		t.Errorf("a third address in the same /64 got %d, want 429", code)
	}
	// Typing someone else's address over and over neither blocks them from
	// writing in nor floods their inbox: every message is kept, and the
	// address gets one confirmation an hour.
	for i := 1; i <= 3; i++ {
		if code := from(fmt.Sprintf("198.51.100.%d:1000", i), "target@example.org"); code != 201 {
			t.Fatalf("message %d naming the target got %d, want 201", i, code)
		}
	}
	if n := len(e.emails()["target@example.org"]); n != 1 {
		t.Errorf("%d confirmations to one address within the hour, want 1", n)
	}
	if n := e.count(`SELECT count(*) FROM support_tickets WHERE reporter_email = 'target@example.org'`); n != 3 {
		t.Errorf("%d messages kept, want 3", n)
	}
}
