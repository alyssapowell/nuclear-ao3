package works_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// commentList reads GET /works/:id/comments as a JSON array of objects.
func (e *fixEnv) commentList(workID uuid.UUID, token string) []map[string]any {
	e.t.Helper()
	out := e.expect(200, "GET", "/api/v1/works/"+workID.String()+"/comments", token, nil)
	var body struct {
		Comments []map[string]any `json:"comments"`
	}
	if err := json.Unmarshal([]byte(out), &body); err != nil {
		e.t.Fatalf("comment list: %v: %s", err, out)
	}
	return body.Comments
}

func TestPostAndListComments(t *testing.T) {
	e := newFixEnv(t)
	writer, _ := e.user("writer")
	_, readerTok := e.user("reader")
	work := e.work(writer, "A Work", "published")
	path := "/api/v1/works/" + work.String() + "/comments"

	// A logged-in reader comments under their default pseud.
	e.expect(201, "POST", path, readerTok, map[string]any{"content": "<p>lovely</p><script>x()</script>"})
	// A guest needs a name, and gets no account behind it.
	e.expect(400, "POST", path, "", map[string]any{"content": "no name"})
	e.expect(201, "POST", path, "", map[string]any{"content": "from a guest", "guest_name": "Pat", "guest_email": "pat@example.org"})
	// Content is required.
	e.expect(400, "POST", path, readerTok, map[string]any{"content": ""})

	var stored string
	if err := e.db.QueryRow(`SELECT content FROM comments WHERE content LIKE '%lovely%'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, "<script") {
		t.Errorf("comment stored without cleaning: %q", stored)
	}
	if n := e.count(`SELECT comment_count FROM works WHERE id = $1`, work); n != 2 {
		t.Errorf("work comment_count = %d, want 2", n)
	}
	// The creator is told about each.
	if n := e.count(`SELECT count(*) FROM notifications WHERE user_id = $1`, writer); n != 2 {
		t.Errorf("creator has %d notifications, want 2", n)
	}

	list := e.commentList(work, "")
	if len(list) != 2 {
		t.Fatalf("listed %d comments, want 2", len(list))
	}
	for _, c := range list {
		if _, leaked := c["guest_email"]; leaked {
			t.Errorf("guest email in the public list: %v", c)
		}
		for _, banned := range []string{"user_id", "username", "email"} {
			if _, ok := c[banned]; ok {
				t.Errorf("comment shows %s: %v", banned, c)
			}
		}
	}
}

func TestRepliesNestUnderTheirParent(t *testing.T) {
	e := newFixEnv(t)
	writer, _ := e.user("writer")
	_, readerTok := e.user("reader")
	work := e.work(writer, "A Work", "published")
	other := e.work(writer, "Another Work", "published")
	path := "/api/v1/works/" + work.String() + "/comments"

	var parent struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(e.expect(201, "POST", path, readerTok, map[string]any{"content": "parent"})), &parent)
	e.expect(201, "POST", path, readerTok, map[string]any{"content": "reply", "parent_comment_id": parent.ID})
	// A reply must be to a comment on the same work.
	e.expect(404, "POST", "/api/v1/works/"+other.String()+"/comments", readerTok,
		map[string]any{"content": "stray", "parent_comment_id": parent.ID})

	list := e.commentList(work, readerTok)
	if len(list) != 2 {
		t.Fatalf("listed %d comments, want 2", len(list))
	}
	if list[1]["parent_id"] != parent.ID {
		t.Errorf("reply's parent_id = %v, want %v", list[1]["parent_id"], parent.ID)
	}
}

// A comment can be left on one chapter of a work. The database used to refuse
// it (comment_target, migration 002) because the row names both the work and
// the chapter; migration 043 allows it when the chapter is the work's own.
func TestCommentOnAChapter(t *testing.T) {
	e := newFixEnv(t)
	writer, writerTok := e.user("writer")
	_, readerTok := e.user("reader")
	work := e.work(writer, "A Work", "published")
	elsewhere := e.work(writer, "Another Work", "published")
	chapter := e.chapterOf(work)
	path := "/api/v1/works/" + work.String() + "/comments"

	e.exec(`DELETE FROM notifications`)
	out := e.expect(201, "POST", path, readerTok, map[string]any{"content": "on chapter one", "chapter_id": chapter})
	var posted struct {
		ID        string `json:"id"`
		ChapterID string `json:"chapter_id"`
		WorkID    string `json:"work_id"`
	}
	if err := json.Unmarshal([]byte(out), &posted); err != nil {
		t.Fatal(err)
	}
	if posted.ChapterID != chapter.String() || posted.WorkID != work.String() {
		t.Errorf("posted comment names chapter %q, work %q", posted.ChapterID, posted.WorkID)
	}
	// Also with the chapter named but no work in the address body, and a
	// reply to it, which stays on the same chapter's work.
	e.expect(201, "POST", path, readerTok, map[string]any{"content": "reply", "chapter_id": chapter, "parent_comment_id": posted.ID})

	// Listed for everyone, with its chapter.
	for _, tok := range []string{"", readerTok, writerTok} {
		list := e.commentList(work, tok)
		if len(list) != 2 {
			t.Fatalf("token %q: listed %d comments, want 2", tok, len(list))
		}
		if list[0]["chapter_id"] != chapter.String() {
			t.Errorf("token %q: chapter_id = %v", tok, list[0]["chapter_id"])
		}
	}
	// Counted on the work, and the creator is notified of both (AO3 tells a
	// work's creators about every comment on it).
	if n := e.count(`SELECT comment_count FROM works WHERE id = $1`, work); n != 2 {
		t.Errorf("work comment_count = %d, want 2", n)
	}
	if n := e.count(`SELECT count(*) FROM notifications WHERE user_id = $1`, writer); n != 2 {
		t.Errorf("creator has %d notifications, want 2", n)
	}

	// The chapter has to be this work's.
	e.expect(400, "POST", path, readerTok, map[string]any{"content": "wrong", "chapter_id": e.chapterOf(elsewhere)})
	e.expect(404, "POST", path, readerTok, map[string]any{"content": "wrong", "chapter_id": uuid.New()})

	// A draft chapter takes comments from nobody but its creator.
	draft := uuid.New()
	e.exec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, summary, notes, end_notes)
		VALUES ($1, $2, 2, 'two', 'not yet', true, '', '', '')`, draft, work)
	e.expect(404, "POST", path, readerTok, map[string]any{"content": "early", "chapter_id": draft})
	e.expect(201, "POST", path, writerTok, map[string]any{"content": "note to self", "chapter_id": draft})
	// ...and its comments stay hidden from everyone else.
	if list := e.commentList(work, readerTok); len(list) != 2 {
		t.Errorf("reader sees %d comments with a draft chapter's comment around, want 2", len(list))
	}
	if list := e.commentList(work, writerTok); len(list) != 3 {
		t.Errorf("creator sees %d comments, want 3", len(list))
	}

	// A chapter an admin hid takes no comments, and its comments leave the
	// reader's list.
	e.exec(`UPDATE chapters SET hidden_by_admin = true WHERE id = $1`, chapter)
	e.expect(404, "POST", path, readerTok, map[string]any{"content": "late", "chapter_id": chapter})
	if list := e.commentList(work, readerTok); len(list) != 0 {
		t.Errorf("reader still sees %d comments on a hidden chapter", len(list))
	}
	e.exec(`UPDATE chapters SET hidden_by_admin = false WHERE id = $1`, chapter)

	// Deleting the chapter takes its comments with it, and the count follows.
	e.exec(`DELETE FROM chapters WHERE id = $1`, chapter)
	if n := e.count(`SELECT comment_count FROM works WHERE id = $1`, work); n != 1 {
		t.Errorf("work comment_count = %d after the chapter went, want 1 (the draft chapter's)", n)
	}
}

// Chapter comments obey blocks like any other comment on the work.
func TestChapterCommentsFollowBlocks(t *testing.T) {
	e := newFixEnv(t)
	writer, _ := e.user("writer")
	reader, readerTok := e.user("reader")
	work := e.work(writer, "A Work", "published")
	chapter := e.chapterOf(work)
	path := "/api/v1/works/" + work.String() + "/comments"

	e.exec(`INSERT INTO user_blocks (blocker_id, blocked_id, pseud_id, pseud_name)
		SELECT $1, u.id, p.id, p.name FROM users u JOIN pseuds p ON p.user_id = u.id AND p.is_default WHERE u.id = $2`, writer, reader)
	e.expect(403, "POST", path, readerTok, map[string]any{"content": "blocked", "chapter_id": chapter})
}
