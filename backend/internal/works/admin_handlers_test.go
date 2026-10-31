package works_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// The older /admin endpoints (status changes, deletes, the report queue) are
// for administrators. They once refused everyone, because they read a
// users.role column that nothing sets instead of user_roles.

type adminWorld struct {
	e       *fixEnv
	admin   string // tokens
	mod     string
	reader  string
	author  uuid.UUID
	work    uuid.UUID
	comment uuid.UUID
}

func newAdminWorld(t *testing.T) *adminWorld {
	e := newFixEnv(t)
	w := &adminWorld{e: e}
	_, w.admin = e.user("boss", "admin")
	_, w.mod = e.user("mod", "moderator")
	_, w.reader = e.user("reader")
	w.author, _ = e.user("author")
	w.work = e.work(w.author, "A Work", "published")
	w.comment = e.comment(w.work, w.author, "a comment")
	return w
}

func (w *adminWorld) actions(action string) int {
	return w.e.count(`SELECT count(*) FROM admin_actions WHERE action = $1`, action)
}

func TestAdminEndpointsAreForAdminsOnly(t *testing.T) {
	w := newAdminWorld(t)
	e := w.e
	work := "/api/v1/admin/works/" + w.work.String()
	comment := "/api/v1/admin/comments/" + w.comment.String()
	for _, tok := range []string{"", w.reader, w.mod} {
		want := 403
		if tok == "" {
			want = 401
		}
		e.expect(want, "GET", "/api/v1/admin/reports", tok, nil)
		e.expect(want, "GET", "/api/v1/admin/comments", tok, nil)
		e.expect(want, "PUT", work+"/status", tok, map[string]any{"status": "abandoned"})
		e.expect(want, "DELETE", work, tok, map[string]any{"reason": "x", "confirm": true})
		e.expect(want, "PUT", comment+"/status", tok, map[string]any{"status": "hidden"})
		e.expect(want, "DELETE", comment, tok, map[string]any{"reason": "x"})
	}
	if n := e.count(`SELECT count(*) FROM admin_actions`); n != 0 {
		t.Errorf("refused requests wrote %d audit rows", n)
	}
	if n := e.count(`SELECT count(*) FROM works WHERE id = $1 AND status = 'published'`, w.work); n != 1 {
		t.Error("a refused request changed the work")
	}
}

func TestAdminUpdateWorkStatus(t *testing.T) {
	w := newAdminWorld(t)
	e := w.e
	path := "/api/v1/admin/works/" + w.work.String() + "/status"

	e.expect(400, "PUT", path, w.admin, map[string]any{"status": "bogus"})
	e.expect(200, "PUT", path, w.admin, map[string]any{"status": "abandoned", "reason": "inappropriate content"})
	var status string
	e.db.QueryRow(`SELECT status FROM works WHERE id = $1`, w.work).Scan(&status)
	if status != "abandoned" {
		t.Errorf("status = %q, want abandoned", status)
	}
	// Moderation is logged and audited, and the author is told.
	if n := e.count(`SELECT count(*) FROM moderation_logs WHERE target_id = $1`, w.work); n != 1 {
		t.Errorf("%d moderation log rows, want 1", n)
	}
	if n := w.actions("set_status"); n != 1 {
		t.Errorf("%d audit rows, want 1", n)
	}
	if n := e.count(`SELECT count(*) FROM notifications WHERE user_id = $1 AND type = 'moderator_action'`, w.author); n != 1 {
		t.Errorf("author has %d moderator notifications, want 1", n)
	}
	// Restoring is allowed; the same state again is not a valid change.
	e.expect(200, "PUT", path, w.admin, map[string]any{"status": "published"})
	e.expect(400, "PUT", path, w.admin, map[string]any{"status": "published"})
	e.expect(404, "PUT", "/api/v1/admin/works/"+uuid.New().String()+"/status", w.admin, map[string]any{"status": "abandoned"})
}

func TestAdminDeleteWork(t *testing.T) {
	w := newAdminWorld(t)
	e := w.e
	path := "/api/v1/admin/works/" + w.work.String()

	// A reason and an explicit yes are both needed.
	e.expect(400, "DELETE", path, w.admin, map[string]any{"reason": "tos"})
	e.expect(400, "DELETE", path, w.admin, map[string]any{"reason": "tos", "confirm": false})
	if n := e.count(`SELECT count(*) FROM works WHERE id = $1`, w.work); n != 1 {
		t.Fatal("work deleted without confirmation")
	}
	e.expect(200, "DELETE", path, w.admin, map[string]any{"reason": "Terms of service violation", "confirm": true})
	if n := e.count(`SELECT count(*) FROM works WHERE id = $1`, w.work); n != 0 {
		t.Error("work still exists")
	}
	if n := w.actions("delete"); n != 1 {
		t.Errorf("%d audit rows for the delete, want 1", n)
	}
}

func TestAdminCommentModeration(t *testing.T) {
	w := newAdminWorld(t)
	e := w.e
	base := "/api/v1/admin/comments/" + w.comment.String()

	list := e.expect(200, "GET", "/api/v1/admin/comments", w.admin, nil)
	if !strings.Contains(list, w.comment.String()) {
		t.Errorf("comment missing from the admin list: %s", list)
	}
	e.expect(400, "PUT", base+"/status", w.admin, map[string]any{"status": "approved"}) // not a status
	e.expect(200, "PUT", base+"/status", w.admin, map[string]any{"status": "hidden", "reason": "review"})
	var status string
	e.db.QueryRow(`SELECT status FROM comments WHERE id = $1`, w.comment).Scan(&status)
	if status != "hidden" {
		t.Errorf("status = %q, want hidden", status)
	}
	if n := w.actions("set_status"); n != 1 {
		t.Errorf("%d audit rows, want 1", n)
	}

	// A comment with replies needs delete_replies, then goes with them.
	reply := uuid.New()
	e.exec(`INSERT INTO comments (id, work_id, user_id, pseudonym_id, parent_comment_id, content)
		SELECT $1, $2, $3, id, $4, 'reply' FROM pseuds WHERE user_id = $3 AND is_default`, reply, w.work, w.author, w.comment)
	e.expect(400, "DELETE", base, w.admin, map[string]any{"reason": "spam"})
	e.expect(200, "DELETE", base, w.admin, map[string]any{"reason": "spam", "delete_replies": true})
	// The comment and its replies are removed for good, and it is audited.
	if n := e.count(`SELECT count(*) FROM comments WHERE id IN ($1, $2)`, w.comment, reply); n != 0 {
		t.Errorf("%d of the comment and its reply remain", n)
	}
	if n := w.actions("delete"); n != 1 {
		t.Errorf("%d audit rows for the delete, want 1", n)
	}
}

func TestAdminReportQueue(t *testing.T) {
	w := newAdminWorld(t)
	e := w.e
	reader := e.tokens[w.reader]
	e.exec(`INSERT INTO reports (target_type, target_id, reporter_id, reason, description)
		VALUES ('work', $1, $2, 'inappropriate_content', 'Test report')`, w.work, reader)
	out := e.expect(200, "GET", "/api/v1/admin/reports?status=pending", w.admin, nil)
	if !strings.Contains(out, w.work.String()) || !strings.Contains(out, "inappropriate_content") {
		t.Errorf("report missing from the queue: %s", out)
	}
	e.expect(200, "GET", "/api/v1/admin/reports?status=resolved", w.admin, nil)
}
