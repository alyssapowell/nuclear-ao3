package works

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/platform"
)

type histEnv struct {
	t  *testing.T
	ws *WorkService
	r  *gin.Engine
}

func newHistEnv(t *testing.T) *histEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ws := newTestWorkService(t)
	return &histEnv{t: t, ws: ws, r: setupRouter(ws)}
}

func (e *histEnv) exec(q string, args ...any) {
	e.t.Helper()
	if _, err := e.ws.db.Exec(q, args...); err != nil {
		e.t.Fatalf("%s: %v", q, err)
	}
}

func (e *histEnv) user(name string) uuid.UUID {
	id := uuid.New()
	e.exec(`INSERT INTO users (id, username, email, password_hash, is_active, is_verified)
		VALUES ($1, $2, $3, 'x', true, true)`, id, name, name+"@example.org")
	return id
}

func (e *histEnv) work(owner uuid.UUID, title string, extra string) uuid.UUID {
	id := uuid.New()
	e.exec(`INSERT INTO works (id, user_id, title, summary, language, rating, status, is_draft, published_at, fandoms)
		VALUES ($1, $2, $3, 's', 'en', 'general', 'published', false, now(), $4)`, id, owner, title, pq.Array([]string{"F"}))
	e.exec(`INSERT INTO creatorships (creation_id, creation_type, pseud_id)
		SELECT $1, 'Work', id FROM pseuds WHERE user_id = $2 AND is_default`, id, owner)
	e.exec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, published_at, summary, notes, end_notes)
		VALUES ($1, $2, 1, 'one', 'text', false, now(), '', '', '')`, uuid.New(), id)
	if extra != "" {
		e.exec(`UPDATE works SET `+extra+` WHERE id = $1`, id)
	}
	return id
}

func (e *histEnv) do(method, path string, who uuid.UUID, body string) (int, map[string]any) {
	e.t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if who != uuid.Nil {
		req.Header.Set("Authorization", "Bearer "+who.String())
	}
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func (e *histEnv) flush() {
	e.t.Helper()
	if err := e.ws.FlushReadings(context.Background()); err != nil {
		e.t.Fatal(err)
	}
}

func (e *histEnv) readings(user uuid.UUID) int {
	var n int
	if err := e.ws.db.QueryRow(`SELECT COUNT(*) FROM readings WHERE user_id = $1`, user).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func (e *histEnv) list(user uuid.UUID, q string) []any {
	e.t.Helper()
	code, out := e.do("GET", "/api/v1/my/history"+q, user, "")
	if code != 200 {
		e.t.Fatalf("history list: %d", code)
	}
	return out["readings"].([]any)
}

func TestReadingIsRecordedOncePerWork(t *testing.T) {
	e := newHistEnv(t)
	author, reader := e.user("author"), e.user("reader")
	w := e.work(author, "Story", "")

	for i := 0; i < 3; i++ {
		if code, _ := e.do("GET", "/api/v1/works/"+w.String(), reader, ""); code != 200 {
			t.Fatalf("open work: %d", code)
		}
	}
	// Nothing is written on the reading path.
	if e.readings(reader) != 0 {
		t.Fatal("a reading was written before the flush")
	}
	e.flush()
	if e.readings(reader) != 1 {
		t.Fatalf("want one row per (user, work), got %d", e.readings(reader))
	}
	// A second flush window adds a visit and records the chapter.
	e.do("GET", fmt.Sprintf("/api/v1/works/%s/chapters/1", w), reader, "")
	e.flush()
	var views int
	var chapter *int
	e.ws.db.QueryRow(`SELECT view_count, last_chapter FROM readings WHERE user_id = $1`, reader).Scan(&views, &chapter)
	if views != 2 || chapter == nil || *chapter != 1 {
		t.Errorf("view_count=%d last_chapter=%v, want 2 and 1", views, chapter)
	}
	if e.readings(reader) != 1 {
		t.Error("a second visit made a second row")
	}
}

func TestReadingNotRecordedForGuestsAppsOrWhenDisabled(t *testing.T) {
	e := newHistEnv(t)
	author, reader := e.user("author"), e.user("reader")
	w := e.work(author, "Story", "")

	// Guest.
	if code, _ := e.do("GET", "/api/v1/works/"+w.String(), uuid.Nil, ""); code != 200 {
		t.Fatalf("guest open: %d", code)
	}
	e.flush()
	var n int
	e.ws.db.QueryRow(`SELECT COUNT(*) FROM readings`).Scan(&n)
	if n != 0 {
		t.Errorf("a guest's visit was recorded (%d rows)", n)
	}

	// An app acting for the person is a reader and records nothing.
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("user_uuid", reader)
	c.Set(platform.CtxAuthKind, "app")
	e.ws.recordReading(c, w, 1)
	e.flush()
	if e.readings(reader) != 0 {
		t.Error("an app's read was recorded")
	}
	// The same context as a login token does record.
	c.Set(platform.CtxAuthKind, "login")
	e.ws.recordReading(c, w, 1)
	e.flush()
	if e.readings(reader) != 1 {
		t.Error("a login read was not recorded")
	}

	// Turned off: stops recording, whether the visit was buffered before or after.
	e.do("DELETE", "/api/v1/my/history", reader, "")
	e.do("GET", "/api/v1/works/"+w.String(), reader, "") // buffered
	if code, out := e.do("PUT", "/api/v1/my/history-settings", reader, `{"history_enabled":false}`); code != 200 || out["history_enabled"] != false {
		t.Fatalf("turn off: %d %v", code, out)
	}
	e.do("GET", "/api/v1/works/"+w.String(), reader, "")
	e.flush()
	if e.readings(reader) != 0 {
		t.Error("history was recorded while turned off")
	}
	// Disabled between the buffering and the flush is also dropped.
	e.exec(`UPDATE users SET history_enabled = true WHERE id = $1`, reader)
	e.do("GET", "/api/v1/works/"+w.String(), reader, "")
	e.exec(`UPDATE users SET history_enabled = false WHERE id = $1`, reader)
	e.flush()
	if e.readings(reader) != 0 {
		t.Error("a buffered visit was written after history was turned off")
	}
}

func TestHistoryListVisibility(t *testing.T) {
	e := newHistEnv(t)
	author, reader := e.user("author"), e.user("reader")
	shown := e.work(author, "Shown", "")
	hidden := e.work(author, "Hidden Later", "")
	anon := e.work(author, "Anon Work", "is_anonymous = true")
	gone := e.work(author, "Deleted Later", "")
	for _, w := range []uuid.UUID{shown, hidden, anon, gone} {
		e.do("GET", "/api/v1/works/"+w.String(), reader, "")
	}
	e.flush()
	e.exec(`UPDATE works SET hidden_by_admin = true WHERE id = $1`, hidden)
	e.exec(`DELETE FROM works WHERE id = $1`, gone)

	items := e.list(reader, "")
	if len(items) != 3 {
		t.Fatalf("want 3 entries (a deleted work is omitted), got %d", len(items))
	}
	for _, it := range items {
		m := it.(map[string]any)
		switch m["work_id"] {
		case shown.String():
			if m["available"] != true || m["work"].(map[string]any)["title"] != "Shown" {
				t.Errorf("shown work: %v", m)
			}
			if a := m["work"].(map[string]any)["authors"].([]any); len(a) != 1 {
				t.Errorf("a visible work names its creator: %v", a)
			}
		case hidden.String():
			if m["available"] != false || m["work"] != nil {
				t.Errorf("a hidden work leaks: %v", m)
			}
		case anon.String():
			if a := m["work"].(map[string]any)["authors"].([]any); len(a) != 0 {
				t.Errorf("a work that hides its creator names someone: %v", a)
			}
		default:
			t.Errorf("unexpected work %v", m["work_id"])
		}
		if m["visit_count"].(float64) != 1 || m["last_visited"] == nil {
			t.Errorf("visit info missing: %v", m)
		}
	}
	// A hidden work's title is nowhere in the answer.
	_, raw := e.do("GET", "/api/v1/my/history", reader, "")
	if b, _ := json.Marshal(raw); bytes.Contains(b, []byte("Hidden Later")) || bytes.Contains(b, []byte("Deleted Later")) {
		t.Error("an unavailable work's title is in the answer")
	}
}

func TestHistoryOrderAndPages(t *testing.T) {
	e := newHistEnv(t)
	author, reader := e.user("author"), e.user("reader")
	var ids []uuid.UUID
	for i := 0; i < 5; i++ {
		w := e.work(author, fmt.Sprintf("W%d", i), "")
		ids = append(ids, w)
		e.exec(`INSERT INTO readings (user_id, work_id, last_viewed_at, view_count) VALUES ($1, $2, now() - ($3 || ' hours')::interval, 1)`, reader, w, fmt.Sprint(10-i))
	}
	items := e.list(reader, "?limit=2&page=1")
	if len(items) != 2 || items[0].(map[string]any)["work_id"] != ids[4].String() {
		t.Errorf("newest viewed should be first: %v", items)
	}
	code, out := e.do("GET", "/api/v1/my/history?limit=2&page=3", reader, "")
	if code != 200 || len(out["readings"].([]any)) != 1 || out["pagination"].(map[string]any)["total"].(float64) != 5 {
		t.Errorf("page 3: %d %v", code, out)
	}
	if code, _ := e.do("GET", "/api/v1/my/history", uuid.Nil, ""); code != 401 {
		t.Errorf("history for a guest: %d, want 401", code)
	}
}

func TestMarkForLater(t *testing.T) {
	e := newHistEnv(t)
	author, reader := e.user("author"), e.user("reader")
	a, b := e.work(author, "A", ""), e.work(author, "B", "")
	e.do("GET", "/api/v1/works/"+a.String(), reader, "")
	e.do("GET", "/api/v1/works/"+b.String(), reader, "")
	e.flush()

	code, _ := e.do("PUT", "/api/v1/my/history/"+a.String()+"/marked", reader, `{"marked":true}`)
	if code != 200 {
		t.Fatalf("mark: %d", code)
	}
	_, st := e.do("GET", "/api/v1/my/history/"+a.String(), reader, "")
	if st["marked_for_later"] != true || st["history_enabled"] != true {
		t.Errorf("status: %v", st)
	}
	to := e.list(reader, "?show=to-read")
	if len(to) != 1 || to[0].(map[string]any)["work_id"] != a.String() || to[0].(map[string]any)["marked_for_later"] != true {
		t.Errorf("to-read list: %v", to)
	}
	if len(e.list(reader, "")) != 2 {
		t.Error("marked works also stay in the history")
	}
	// Mark as read.
	e.do("PUT", "/api/v1/my/history/"+a.String()+"/marked", reader, `{"marked":false}`)
	if len(e.list(reader, "?show=to-read")) != 0 {
		t.Error("mark as read left it in the to-read list")
	}
	// Marking a work not yet in the history creates its row.
	c := e.work(author, "C", "")
	if code, _ := e.do("PUT", "/api/v1/my/history/"+c.String()+"/marked", reader, `{"marked":true}`); code != 200 {
		t.Errorf("mark an unread work: %d", code)
	}
	// A work the person can't see is not found; a bad body is refused.
	e.exec(`UPDATE works SET hidden_by_admin = true WHERE id = $1`, b)
	if code, _ := e.do("PUT", "/api/v1/my/history/"+b.String()+"/marked", reader, `{"marked":true}`); code != 404 {
		t.Errorf("mark a hidden work: %d, want 404", code)
	}
	if code, _ := e.do("PUT", "/api/v1/my/history/"+c.String()+"/marked", reader, `{}`); code != 400 {
		t.Errorf("empty body: %d, want 400", code)
	}
	// With history off the button is not offered, and the API refuses.
	e.do("PUT", "/api/v1/my/history-settings", reader, `{"history_enabled":false}`)
	if code, _ := e.do("PUT", "/api/v1/my/history/"+c.String()+"/marked", reader, `{"marked":true}`); code != 409 {
		t.Errorf("mark with history off: %d, want 409", code)
	}
	if code, _ := e.do("PUT", "/api/v1/my/history/"+c.String()+"/marked", uuid.Nil, `{"marked":true}`); code != 401 {
		t.Errorf("mark as a guest: %d, want 401", code)
	}
}

func TestDeleteAndClearHistory(t *testing.T) {
	e := newHistEnv(t)
	author, reader := e.user("author"), e.user("reader")
	a, b := e.work(author, "A", ""), e.work(author, "B", "")
	for _, w := range []uuid.UUID{a, b} {
		e.do("GET", "/api/v1/works/"+w.String(), reader, "")
	}
	e.flush()

	if code, _ := e.do("DELETE", "/api/v1/my/history/"+a.String(), reader, ""); code != 200 {
		t.Fatalf("delete one: %d", code)
	}
	items := e.list(reader, "")
	if len(items) != 1 || items[0].(map[string]any)["work_id"] != b.String() {
		t.Errorf("after deleting A: %v", items)
	}
	// Visits waiting to be written are cleared too.
	e.do("GET", "/api/v1/works/"+a.String(), reader, "")
	if code, _ := e.do("DELETE", "/api/v1/my/history", reader, ""); code != 200 {
		t.Fatal("clear")
	}
	e.flush()
	if e.readings(reader) != 0 {
		t.Error("clear left rows or a waiting visit behind")
	}
}

func TestHistoryIsPerPerson(t *testing.T) {
	e := newHistEnv(t)
	author, one, two := e.user("author"), e.user("one"), e.user("two")
	w := e.work(author, "Story", "")
	e.do("GET", "/api/v1/works/"+w.String(), one, "")
	e.flush()
	e.exec(`UPDATE readings SET marked_for_later = true WHERE user_id = $1`, one)

	if got := e.list(two, ""); len(got) != 0 {
		t.Errorf("another person sees %d entries", len(got))
	}
	if _, st := e.do("GET", "/api/v1/my/history/"+w.String(), two, ""); st["marked_for_later"] != false {
		t.Error("another person sees the mark")
	}
	// Deleting and clearing as someone else changes nothing.
	e.do("DELETE", "/api/v1/my/history/"+w.String(), two, "")
	e.do("DELETE", "/api/v1/my/history", two, "")
	e.do("PUT", "/api/v1/my/history-settings", two, `{"history_enabled":false}`)
	if e.readings(one) != 1 {
		t.Error("someone else's delete or clear reached this person's history")
	}
	if on, _ := e.ws.historyEnabled(context.Background(), one); !on {
		t.Error("someone else's setting changed this person's")
	}
}

func TestReadingsGoWithTheAccount(t *testing.T) {
	e := newHistEnv(t)
	author, reader := e.user("author"), e.user("reader")
	w := e.work(author, "Story", "")
	e.do("GET", "/api/v1/works/"+w.String(), reader, "")
	e.flush()
	if e.readings(reader) != 1 {
		t.Fatal("setup")
	}
	e.exec(`DELETE FROM users WHERE id = $1`, reader)
	var n int
	e.ws.db.QueryRow(`SELECT COUNT(*) FROM readings WHERE user_id = $1`, reader).Scan(&n)
	if n != 0 {
		t.Error("readings outlived the account")
	}
}

func TestHistoryRoutesAreNotForApps(t *testing.T) {
	for _, route := range []string{
		"GET /api/v1/my/history", "DELETE /api/v1/my/history", "GET /api/v1/my/history/:work_id",
		"DELETE /api/v1/my/history/:work_id", "PUT /api/v1/my/history/:work_id/marked",
		"GET /api/v1/my/history-settings", "PUT /api/v1/my/history-settings",
	} {
		if _, ok := platform.AppRouteScope(route); ok {
			t.Errorf("%s is open to apps", route)
		}
	}
	_ = http.StatusOK
}
