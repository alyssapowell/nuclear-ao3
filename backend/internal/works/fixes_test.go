package works_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/moderation"
	"nuclear-ao3/internal/platform"
	"nuclear-ao3/internal/testdb"
	"nuclear-ao3/internal/works"
)

type fixEnv struct {
	t      *testing.T
	db     *sql.DB
	router *gin.Engine
	tokens map[string]uuid.UUID
}

func newFixEnv(t *testing.T) *fixEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testdb.New(t)
	e := &fixEnv{t: t, db: db, tokens: map[string]uuid.UUID{}}
	deps := platform.NewDeps(db, func(tok string) (uuid.UUID, error) {
		if id, ok := e.tokens[tok]; ok {
			return id, nil
		}
		return uuid.Nil, sql.ErrNoRows
	})
	ws, err := works.New(db)
	if err != nil {
		t.Fatal(err)
	}
	e.router = gin.New()
	works.Mount(e.router, ws, deps)
	moderation.Mount(e.router, moderation.New(db), deps)
	return e
}

func (e *fixEnv) exec(q string, args ...any) {
	e.t.Helper()
	if _, err := e.db.Exec(q, args...); err != nil {
		e.t.Fatalf("%s: %v", q, err)
	}
}

func (e *fixEnv) user(name string, roles ...string) (uuid.UUID, string) {
	e.t.Helper()
	id := uuid.New()
	e.exec(`INSERT INTO users (id, username, email, password_hash, is_active, is_verified)
		VALUES ($1, $2, $3, 'x', true, true)`, id, name, name+"@example.org")
	e.exec(`INSERT INTO pseuds (user_id, name, is_default) VALUES ($1, $2, true)`, id, name)
	e.exec(`INSERT INTO user_pseudonyms (user_id, name, is_default) VALUES ($1, $2, true)`, id, name)
	for _, r := range roles {
		e.exec(`INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, id, r)
	}
	tok := "token-" + name
	e.tokens[tok] = id
	return id, tok
}

// work makes a work with one chapter and one fandom tag. state is
// "published", "draft" or "hidden".
func (e *fixEnv) work(owner uuid.UUID, title, state string) uuid.UUID {
	e.t.Helper()
	id := uuid.New()
	draft := state == "draft"
	status := "published"
	if draft {
		status = "draft"
	}
	e.exec(`INSERT INTO works (id, user_id, title, summary, language, rating, status, is_draft, published_at, fandoms, hidden_by_admin)
		VALUES ($1, $2, $3, 'a summary', 'en', 'general', $4, $5, CASE WHEN $5 THEN NULL ELSE now() END, $6, $7)`,
		id, owner, title, status, draft, pq.Array([]string{"Test Fandom"}), state == "hidden")
	e.exec(`INSERT INTO creatorships (creation_id, creation_type, pseud_id)
		SELECT $1, 'Work', id FROM pseuds WHERE user_id = $2 AND is_default`, id, owner)
	e.exec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, published_at, summary, notes, end_notes)
		VALUES ($1, $2, 1, 'one', 'chapter text', $3, CASE WHEN $3 THEN NULL ELSE now() END, '', '', '')`, uuid.New(), id, draft)
	e.exec(`INSERT INTO tags (name, type) VALUES ('Test Fandom', 'fandom') ON CONFLICT (name) DO NOTHING`)
	e.exec(`INSERT INTO work_tags (work_id, tag_id) SELECT $1, id FROM tags WHERE name = 'Test Fandom'`, id)
	return id
}

func (e *fixEnv) comment(workID, author uuid.UUID, content string) uuid.UUID {
	e.t.Helper()
	id := uuid.New()
	e.exec(`INSERT INTO comments (id, work_id, user_id, pseudonym_id, content)
		SELECT $1, $2, $3, id, $4 FROM user_pseudonyms WHERE user_id = $3`, id, workID, author, content)
	return id
}

func (e *fixEnv) do(method, path, token string, body any) (int, string) {
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

func (e *fixEnv) expect(want int, method, path, token string, body any) string {
	e.t.Helper()
	code, out := e.do(method, path, token, body)
	if code != want {
		e.t.Fatalf("%s %s as %q: status %d, want %d: %s", method, path, token, code, want, out)
	}
	return out
}

func (e *fixEnv) useCount() int {
	var n int
	if err := e.db.QueryRow(`SELECT use_count FROM tags WHERE name = 'Test Fandom'`).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func TestUserWorksListForStranger(t *testing.T) {
	e := newFixEnv(t)
	creator, creatorTok := e.user("creator")
	_, strangerTok := e.user("stranger")
	pub := e.work(creator, "Public", "published")
	draft := e.work(creator, "Draft", "draft")
	hidden := e.work(creator, "Hidden", "hidden")

	for _, tok := range []string{strangerTok, ""} {
		out := e.expect(200, "GET", "/api/v1/users/"+creator.String()+"/works", tok, nil)
		if !strings.Contains(out, pub.String()) {
			t.Errorf("token %q: published work missing: %s", tok, out)
		}
		if strings.Contains(out, draft.String()) || strings.Contains(out, hidden.String()) {
			t.Errorf("token %q: draft or hidden work listed: %s", tok, out)
		}
	}
	// The creator sees their own draft too.
	out := e.expect(200, "GET", "/api/v1/users/"+creator.String()+"/works", creatorTok, nil)
	if !strings.Contains(out, draft.String()) {
		t.Errorf("creator should see their draft: %s", out)
	}
}

func TestEditComment(t *testing.T) {
	e := newFixEnv(t)
	creator, creatorTok := e.user("creator")
	author, authorTok := e.user("author")
	_, otherTok := e.user("other")
	w := e.work(creator, "Work", "published")
	id := e.comment(w, author, "first")
	path := "/api/v1/comments/" + id.String()
	stored := func() string {
		var s string
		if err := e.db.QueryRow(`SELECT content FROM comments WHERE id = $1`, id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	e.expect(403, "PUT", path, otherTok, map[string]string{"content": "hijack"})
	e.expect(403, "PUT", path, creatorTok, map[string]string{"content": "hijack"})
	e.expect(401, "PUT", path, "", map[string]string{"content": "x"})
	if stored() != "first" {
		t.Fatalf("comment changed by someone else: %q", stored())
	}

	e.expect(200, "PUT", path, authorTok, map[string]string{"content": `<p>edited</p><script>alert(1)</script>`})
	got := stored()
	if !strings.Contains(got, "edited") || strings.Contains(got, "<script") {
		t.Errorf("stored content not cleaned: %q", got)
	}

	// Frozen: refused, unchanged.
	e.exec(`UPDATE comments SET iced = true WHERE id = $1`, id)
	before := stored()
	e.expect(403, "PUT", path, authorTok, map[string]string{"content": "frozen edit"})
	e.exec(`UPDATE comments SET iced = false WHERE id = $1`, id)

	// Hidden by an admin: refused.
	e.exec(`UPDATE comments SET hidden_by_admin = true WHERE id = $1`, id)
	e.expect(403, "PUT", path, authorTok, map[string]string{"content": "hidden edit"})
	e.exec(`UPDATE comments SET hidden_by_admin = false WHERE id = $1`, id)

	// Has a reply: refused (AO3 "Comments with replies cannot be edited").
	reply := uuid.New()
	e.exec(`INSERT INTO comments (id, work_id, user_id, pseudonym_id, parent_comment_id, content)
		SELECT $1, $2, $3, id, $4, 'reply' FROM user_pseudonyms WHERE user_id = $3`, reply, w, creator, id)
	e.expect(403, "PUT", path, authorTok, map[string]string{"content": "late edit"})
	if stored() != before {
		t.Errorf("refused edits changed the comment: %q", stored())
	}
	e.exec(`DELETE FROM comments WHERE id = $1`, reply)

	// A banned author is refused like other write routes.
	e.exec(`UPDATE users SET banned = true WHERE id = $1`, author)
	e.expect(403, "PUT", path, authorTok, map[string]string{"content": "banned edit"})
}

func TestWorksListTotalCountsAllPages(t *testing.T) {
	e := newFixEnv(t)
	creator, _ := e.user("creator")
	for i := 0; i < 25; i++ {
		e.work(creator, fmt.Sprintf("Work %02d", i), "published")
	}
	e.work(creator, "Draft", "draft")
	e.work(creator, "Hidden", "hidden")

	type page struct {
		Works      []json.RawMessage `json:"works"`
		Pagination struct {
			Page  int `json:"page"`
			Limit int `json:"limit"`
			Total int `json:"total"`
			Pages int `json:"pages"`
		} `json:"pagination"`
	}
	for _, q := range []string{"", "&fandom=Test+Fandom", "&q=Work"} {
		for p, wantLen := range map[int]int{1: 10, 3: 5} {
			var got page
			out := e.expect(200, "GET", fmt.Sprintf("/api/v1/works?limit=10&page=%d%s", p, q), "", nil)
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatal(err)
			}
			if len(got.Works) != wantLen || got.Pagination.Total != 25 || got.Pagination.Pages != 3 {
				t.Errorf("query %q page %d: %d works, total %d, pages %d; want %d, 25, 3",
					q, p, len(got.Works), got.Pagination.Total, got.Pagination.Pages, wantLen)
			}
		}
	}
}

func TestDeleteTaggedWork(t *testing.T) {
	e := newFixEnv(t)
	creator, tok := e.user("creator")
	a := e.work(creator, "A", "published")
	b := e.work(creator, "B", "published")
	h := e.work(creator, "Hidden", "hidden")
	if got := e.useCount(); got != 2 {
		t.Fatalf("use_count = %d, want 2 (hidden work is not counted)", got)
	}
	e.expect(200, "DELETE", "/api/v1/works/"+a.String(), tok, nil)
	if got := e.useCount(); got != 1 {
		t.Errorf("after deleting a published work: use_count = %d, want 1", got)
	}
	e.expect(200, "DELETE", "/api/v1/works/"+h.String(), tok, nil)
	if got := e.useCount(); got != 1 {
		t.Errorf("after deleting a hidden work: use_count = %d, want 1", got)
	}
	e.expect(200, "DELETE", "/api/v1/works/"+b.String(), tok, nil)
	if got := e.useCount(); got != 0 {
		t.Errorf("after deleting every work: use_count = %d, want 0", got)
	}
	var n int
	e.db.QueryRow(`SELECT count(*) FROM work_tag_summaries`).Scan(&n)
	if n != 0 {
		t.Errorf("%d summaries left behind", n)
	}
}

func TestCreatorModeratesComments(t *testing.T) {
	e := newFixEnv(t)
	creator, creatorTok := e.user("creator")
	commenter, _ := e.user("commenter")
	_, adminTok := e.user("admin", "admin")
	w := e.work(creator, "Work", "published")

	flags := func(id uuid.UUID) (unreviewed, spam, byAdmin, hidden bool) {
		if err := e.db.QueryRow(`SELECT unreviewed, is_spam, spam_by_admin, hidden_by_admin FROM comments WHERE id = $1`, id).
			Scan(&unreviewed, &spam, &byAdmin, &hidden); err != nil {
			t.Fatal(err)
		}
		return
	}

	// The creator approves a comment held for review.
	held := e.comment(w, commenter, "held")
	e.exec(`UPDATE comments SET unreviewed = true WHERE id = $1`, held)
	e.expect(200, "PUT", "/api/v1/comments/"+held.String()+"/moderate", creatorTok, map[string]string{"status": "published"})
	if un, sp, _, _ := flags(held); un || sp {
		t.Errorf("approved comment: unreviewed=%v spam=%v", un, sp)
	}

	// The creator can reject (mark spam) and undo their own rejection.
	e.expect(200, "PUT", "/api/v1/comments/"+held.String()+"/moderate", creatorTok, map[string]string{"status": "spam"})
	if _, sp, byAdmin, hid := flags(held); !sp || byAdmin || hid {
		t.Errorf("creator spam: spam=%v by_admin=%v hidden=%v", sp, byAdmin, hid)
	}
	e.expect(200, "PUT", "/api/v1/comments/"+held.String()+"/moderate", creatorTok, map[string]string{"status": "published"})
	if _, sp, _, _ := flags(held); sp {
		t.Error("creator could not undo their own spam mark")
	}

	// An admin's spam mark is the admin's to clear.
	c := e.comment(w, commenter, "spammy")
	e.expect(200, "POST", "/api/v1/admin/comments/"+c.String()+"/spam", adminTok, map[string]string{"reason": "spam"})
	if _, sp, byAdmin, _ := flags(c); !sp || !byAdmin {
		t.Fatalf("admin spam: spam=%v by_admin=%v", sp, byAdmin)
	}
	e.expect(403, "PUT", "/api/v1/comments/"+c.String()+"/moderate", creatorTok, map[string]string{"status": "published"})
	if _, sp, byAdmin, _ := flags(c); !sp || !byAdmin {
		t.Errorf("refused request changed the comment: spam=%v by_admin=%v", sp, byAdmin)
	}
	e.expect(200, "POST", "/api/v1/admin/comments/"+c.String()+"/unspam", adminTok, nil)
	if _, sp, byAdmin, _ := flags(c); sp || byAdmin {
		t.Errorf("admin unspam: spam=%v by_admin=%v", sp, byAdmin)
	}

	// The creator's path never touches hidden_by_admin.
	e.exec(`UPDATE comments SET hidden_by_admin = true WHERE id = $1`, c)
	e.expect(200, "PUT", "/api/v1/comments/"+c.String()+"/moderate", creatorTok, map[string]string{"status": "published"})
	e.expect(200, "PUT", "/api/v1/comments/"+c.String()+"/moderate", creatorTok, map[string]string{"status": "hidden"})
	if _, _, _, hid := flags(c); !hid {
		t.Error("creator cleared hidden_by_admin")
	}

	// A banned creator is refused.
	e.exec(`UPDATE users SET banned = true WHERE id = $1`, creator)
	e.expect(403, "PUT", "/api/v1/comments/"+c.String()+"/moderate", creatorTok, map[string]string{"status": "published"})
}

// As in AO3, a comment can be deleted by its author or by the work's
// creator; anyone else is refused.
func TestDeleteComment(t *testing.T) {
	e := newFixEnv(t)
	creator, creatorTok := e.user("creator")
	author, authorTok := e.user("author")
	_, otherTok := e.user("other")
	w := e.work(creator, "Work", "published")
	deleted := func(id uuid.UUID) bool {
		var d bool
		if err := e.db.QueryRow(`SELECT is_deleted FROM comments WHERE id = $1`, id).Scan(&d); err != nil {
			t.Fatal(err)
		}
		return d
	}

	mine := e.comment(w, author, "by the author")
	e.expect(403, "DELETE", "/api/v1/comments/"+mine.String(), otherTok, nil)
	e.expect(401, "DELETE", "/api/v1/comments/"+mine.String(), "", nil)
	if deleted(mine) {
		t.Fatal("deleted by someone with no right to")
	}
	e.expect(200, "DELETE", "/api/v1/comments/"+mine.String(), authorTok, nil)
	if !deleted(mine) {
		t.Error("the author couldn't delete their comment")
	}

	onTheirWork := e.comment(w, author, "on the creator's work")
	e.expect(200, "DELETE", "/api/v1/comments/"+onTheirWork.String(), creatorTok, nil)
	if !deleted(onTheirWork) {
		t.Error("the work's creator couldn't delete a comment on it")
	}
}
