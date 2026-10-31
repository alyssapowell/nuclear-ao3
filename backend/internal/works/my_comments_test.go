package works_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
)

func TestMyComments(t *testing.T) {
	e := newFixEnv(t)
	owner, _ := e.user("owner")
	me, tok := e.user("myself")
	other, _ := e.user("other")
	work := e.work(owner, "Test Work", "published")

	first := e.comment(work, me, "First comment")
	e.comment(work, other, "Someone else's comment") // never listed for me
	reply := uuid.New()
	e.exec(`INSERT INTO comments (id, work_id, user_id, pseudonym_id, parent_comment_id, content)
		SELECT $1, $2, $3, id, $4, 'My reply' FROM pseuds WHERE user_id = $3 AND is_default`, reply, work, me, first)
	for i := 0; i < 3; i++ {
		e.comment(work, me, fmt.Sprintf("Comment %d", i))
	}
	e.exec(`UPDATE comments SET status = 'pending' WHERE id = $1`, first)

	e.expect(401, "GET", "/api/v1/my/comments", "", nil)

	all := e.getJSON("/api/v1/my/comments", tok, 200)
	items := all["comments"].([]any)
	if len(items) != 5 {
		t.Fatalf("listed %d comments, want my 5", len(items))
	}
	replies := 0
	for _, it := range items {
		m := it.(map[string]any)
		if m["work_title"] != "Test Work" || m["work_id"] != work.String() {
			t.Errorf("work info missing: %v", m)
		}
		if m["is_reply"] == true {
			replies++
		}
	}
	if replies != 1 {
		t.Errorf("%d replies, want 1", replies)
	}

	// Filter by status
	if got := e.getJSON("/api/v1/my/comments?status=pending", tok, 200)["comments"].([]any); len(got) != 1 {
		t.Errorf("pending filter: %d, want 1", len(got))
	}
	// Pages, and limits that are out of range or not numbers fall back
	if got := e.getJSON("/api/v1/my/comments?limit=2&page=1", tok, 200)["comments"].([]any); len(got) != 2 {
		t.Errorf("page 1 of 2: %d", len(got))
	}
	if got := e.getJSON("/api/v1/my/comments?limit=2&page=3", tok, 200)["comments"].([]any); len(got) != 1 {
		t.Errorf("page 3 of 2: %d, want 1", len(got))
	}
	for _, q := range []string{"limit=200", "limit=invalid"} {
		if got := e.getJSON("/api/v1/my/comments?"+q, tok, 200)["comments"].([]any); len(got) != 5 {
			t.Errorf("%s: %d comments, want all 5", q, len(got))
		}
	}
	// Nothing written yet is an empty list, not null
	_, newTok := e.user("newcomer")
	if got := e.getJSON("/api/v1/my/comments", newTok, 200)["comments"].([]any); len(got) != 0 {
		t.Errorf("newcomer has %d comments", len(got))
	}
}
