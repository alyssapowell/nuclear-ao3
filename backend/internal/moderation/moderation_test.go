package moderation_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// everywhereAWorkAppears lists every read path that shows works, with the
// work in it, for the viewer.
func (e *env) everywhereAWorkAppears(token string, work, series, collection, bookmarker uuid.UUID) map[string]bool {
	e.t.Helper()
	paths := map[string]string{
		"browse":      "/api/v1/works",
		"search":      "/api/v1/search/works",
		"suggestions": "/api/v1/search/suggestions?q=Hidden",
		"tag page":    "/api/v1/tags/" + e.tagID() + "/works",
		"series":      "/api/v1/series/" + series.String() + "/works",
		"collection":  "/api/v1/collections/" + collection.String() + "/works",
		"bookmarks":   e.pseudPage(bookmarker, "bookmarks"),
		"work page":   "/api/v1/work/" + work.String(),
		"chapters":    "/api/v1/work/" + work.String() + "/chapters",
		"comments":    "/api/v1/work/" + work.String() + "/comments",
		"stats":       "/api/v1/work/" + work.String() + "/stats",
		"kudos":       "/api/v1/work/" + work.String() + "/kudos",
	}
	out := map[string]bool{}
	for name, p := range paths {
		code, body := e.do("GET", p, token, nil)
		// "work page", "stats" and the like don't contain the id when they
		// succeed (stats), so count a 200 as showing.
		switch name {
		case "stats", "chapters", "comments", "kudos":
			out[name] = code == http.StatusOK
		default:
			out[name] = code == http.StatusOK && shows(body, work)
		}
	}
	return out
}

func TestHiddenWorkIsInvisibleEverywhereExceptToCreatorAndAdmins(t *testing.T) {
	e := newEnv(t)
	creator, creatorTok := e.user("creator")
	_, strangerTok := e.user("stranger")
	_, adminTok := e.user("boss", "admin")
	bookmarker, _ := e.user("bookmarker")

	work := e.work(creator, "Hidden Gem")
	other := e.work(creator, "Visible Gem")
	series, collection := uuid.New(), uuid.New()
	e.exec(`INSERT INTO series (id, title, user_id) VALUES ($1, 'S', $2)`, series, creator)
	e.exec(`INSERT INTO series_works (series_id, work_id, position) VALUES ($1, $2, 1), ($1, $3, 2)`, series, work, other)
	e.exec(`INSERT INTO collections (id, name, title, user_id) VALUES ($1, 'c', 'C', $2)`, collection, creator)
	e.exec(`INSERT INTO collection_items (collection_id, work_id, is_approved) VALUES ($1, $2, true), ($1, $3, true)`, collection, work, other)
	e.exec(`INSERT INTO bookmarks (work_id, user_id, notes) VALUES ($1, $2, ''), ($3, $2, '')`, work, bookmarker, other)

	// Before hiding, a stranger and a logged-out reader see it everywhere.
	for _, tok := range []string{"", strangerTok} {
		for path, shown := range e.everywhereAWorkAppears(tok, work, series, collection, bookmarker) {
			if !shown {
				t.Errorf("before hiding, %q reader can't see the work on %s", tok, path)
			}
		}
	}
	if got := e.useCount(); got != 2 {
		t.Fatalf("tag use_count = %d before hiding, want 2", got)
	}

	// An admin hides it (this is the real endpoint, so the audit row is real).
	e.expect(200, "POST", "/api/v1/admin/works/"+work.String()+"/hide", adminTok, map[string]string{"reason": "test"})

	for _, tok := range []string{"", strangerTok} {
		for path, shown := range e.everywhereAWorkAppears(tok, work, series, collection, bookmarker) {
			if shown {
				t.Errorf("after hiding, %q reader still sees the work on %s", tok, path)
			}
		}
		// The rest of the series, collection and bookmarks are still there.
		for _, p := range []string{"/api/v1/series/" + series.String() + "/works",
			"/api/v1/collections/" + collection.String() + "/works",
			e.pseudPage(bookmarker, "bookmarks"), "/api/v1/works"} {
			if _, body := e.do("GET", p, tok, nil); !shows(body, other) {
				t.Errorf("%q reader lost the unhidden work on %s: %s", tok, p, body)
			}
		}
	}

	// The creator and an admin still see it, in listings and on its page.
	for _, tok := range []string{creatorTok, adminTok} {
		for path, shown := range e.everywhereAWorkAppears(tok, work, series, collection, bookmarker) {
			if !shown {
				t.Errorf("after hiding, %q can't see the work on %s", tok, path)
			}
		}
	}
	// And the creator is told it's hidden.
	var page struct {
		Work struct {
			Hidden bool `json:"hidden_by_admin"`
		} `json:"work"`
	}
	body := e.expect(200, "GET", "/api/v1/work/"+work.String(), creatorTok, nil)
	json.Unmarshal([]byte(body), &page)
	if !page.Work.Hidden {
		t.Errorf("creator's page doesn't say the work is hidden: %s", body)
	}

	// A hidden work's page says nothing to strangers.
	e.expect(404, "GET", "/api/v1/work/"+work.String(), strangerTok, nil)
	e.expect(404, "GET", "/api/v1/work/"+work.String(), "", nil)

	// The series page counts only what the reader can see.
	for tok, want := range map[string]int{"": 1, strangerTok: 1, creatorTok: 2, adminTok: 2} {
		var s struct {
			Series struct {
				WorkCount int `json:"work_count"`
			} `json:"series"`
		}
		body := e.expect(200, "GET", "/api/v1/series/"+series.String(), tok, nil)
		json.Unmarshal([]byte(body), &s)
		if s.Series.WorkCount != want {
			t.Errorf("series work_count for %q = %d, want %d", tok, s.Series.WorkCount, want)
		}
	}

	// Unhiding puts it back for everyone.
	e.expect(200, "POST", "/api/v1/admin/works/"+work.String()+"/unhide", adminTok, nil)
	for path, shown := range e.everywhereAWorkAppears("", work, series, collection, bookmarker) {
		if !shown {
			t.Errorf("after unhiding, logged-out reader can't see the work on %s", path)
		}
	}
}

func TestHiddenSeriesChapterAndBookmark(t *testing.T) {
	e := newEnv(t)
	creator, creatorTok := e.user("creator")
	_, strangerTok := e.user("stranger")
	_, modTok := e.user("mod", "moderator")
	bookmarker, bookmarkerTok := e.user("bookmarker")
	work := e.work(creator, "Plain Work")
	series := uuid.New()
	e.exec(`INSERT INTO series (id, title, user_id) VALUES ($1, 'S', $2)`, series, creator)
	e.exec(`INSERT INTO series_works (series_id, work_id, position) VALUES ($1, $2, 1)`, series, work)
	chapter := uuid.New()
	e.exec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, published_at, summary, notes, end_notes)
		VALUES ($1, $2, 2, 'two', 'second chapter', false, now(), '', '', '')`, chapter, work)
	bookmark := uuid.New()
	e.exec(`INSERT INTO bookmarks (id, work_id, user_id, notes) VALUES ($1, $2, $3, '')`, bookmark, work, bookmarker)

	e.expect(200, "POST", "/api/v1/admin/series/"+series.String()+"/hide", modTok, map[string]string{"reason": "r"})
	e.expect(404, "GET", "/api/v1/series/"+series.String(), strangerTok, nil)
	e.expect(404, "GET", "/api/v1/series/"+series.String(), "", nil)
	e.expect(200, "GET", "/api/v1/series/"+series.String(), creatorTok, nil)
	if _, body := e.do("GET", "/api/v1/series", strangerTok, nil); strings.Contains(body, series.String()) {
		t.Errorf("series search shows a hidden series: %s", body)
	}
	if _, body := e.do("GET", e.pseudPage(creator, "series"), strangerTok, nil); strings.Contains(body, series.String()) {
		t.Errorf("user series list shows a hidden series: %s", body)
	}

	e.expect(200, "POST", "/api/v1/admin/works/"+work.String()+"/chapters/"+chapter.String()+"/hide", modTok, map[string]string{"reason": "r"})
	_, body := e.do("GET", "/api/v1/work/"+work.String()+"/chapters", strangerTok, nil)
	if strings.Contains(body, chapter.String()) || !strings.Contains(body, "chapter text") {
		t.Errorf("stranger's chapter list should have chapter 1 only: %s", body)
	}
	_, body = e.do("GET", "/api/v1/work/"+work.String()+"/chapters", creatorTok, nil)
	if !strings.Contains(body, chapter.String()) {
		t.Errorf("creator's chapter list should have the hidden chapter: %s", body)
	}
	e.expect(404, "GET", "/api/v1/work/"+work.String()+"/chapters/2", strangerTok, nil)
	e.expect(200, "GET", "/api/v1/work/"+work.String()+"/chapters/2", creatorTok, nil)

	e.expect(200, "POST", "/api/v1/admin/bookmarks/"+bookmark.String()+"/hide", modTok, map[string]string{"reason": "r"})
	_, body = e.do("GET", e.pseudPage(bookmarker, "bookmarks"), strangerTok, nil)
	if strings.Contains(body, bookmark.String()) {
		t.Errorf("a stranger sees a hidden bookmark: %s", body)
	}
	_, body = e.do("GET", e.pseudPage(bookmarker, "bookmarks"), "", nil)
	if strings.Contains(body, bookmark.String()) {
		t.Errorf("a logged-out reader sees a hidden bookmark: %s", body)
	}
	_, body = e.do("GET", e.pseudPage(bookmarker, "bookmarks"), bookmarkerTok, nil)
	if !strings.Contains(body, bookmark.String()) {
		t.Errorf("the bookmark's owner should still see it: %s", body)
	}
	_, body = e.do("GET", e.pseudPage(bookmarker, "bookmarks"), modTok, nil)
	if !strings.Contains(body, bookmark.String()) {
		t.Errorf("a moderator should see a hidden bookmark: %s", body)
	}
}

func TestSpamWorksAreExcluded(t *testing.T) {
	e := newEnv(t)
	creator, creatorTok := e.user("creator")
	_, strangerTok := e.user("stranger")
	_, adminTok := e.user("boss", "admin")
	work := e.work(creator, "Hidden Spam")
	series, collection := uuid.New(), uuid.New()
	e.exec(`INSERT INTO series (id, title, user_id) VALUES ($1, 'S', $2)`, series, creator)
	e.exec(`INSERT INTO collections (id, name, title, user_id) VALUES ($1, 'c', 'C', $2)`, collection, creator)
	bookmarker, _ := e.user("bookmarker")

	// Marking spam hides the work too, as in AO3.
	e.expect(200, "POST", "/api/v1/admin/works/"+work.String()+"/spam", adminTok, map[string]string{"reason": "ad for pills"})
	var spam, hidden bool
	e.db.QueryRow(`SELECT spam, hidden_by_admin FROM works WHERE id = $1`, work).Scan(&spam, &hidden)
	if !spam || !hidden {
		t.Fatalf("after marking spam: spam=%v hidden=%v, want both", spam, hidden)
	}
	for _, tok := range []string{"", strangerTok} {
		for path, shown := range e.everywhereAWorkAppears(tok, work, series, collection, bookmarker) {
			if shown {
				t.Errorf("%q reader sees a spam work on %s", tok, path)
			}
		}
	}
	if got := e.useCount(); got != 0 {
		t.Errorf("a spam work still counts toward the tag: use_count = %d", got)
	}
	e.expect(200, "GET", "/api/v1/work/"+work.String(), creatorTok, nil)
	e.expect(200, "GET", "/api/v1/work/"+work.String(), adminTok, nil)

	// Unhiding a spam work is refused: unmark the spam instead.
	body := e.expect(409, "POST", "/api/v1/admin/works/"+work.String()+"/unhide", adminTok, nil)
	if !strings.Contains(body, "work_is_spam") {
		t.Errorf("unhide of a spam work: %s", body)
	}

	// Unmarking clears both flags.
	e.expect(200, "POST", "/api/v1/admin/works/"+work.String()+"/unspam", adminTok, nil)
	e.db.QueryRow(`SELECT spam, hidden_by_admin FROM works WHERE id = $1`, work).Scan(&spam, &hidden)
	if spam || hidden {
		t.Errorf("after unspam: spam=%v hidden=%v, want neither", spam, hidden)
	}
	e.expect(200, "GET", "/api/v1/work/"+work.String(), "", nil)
	if got := e.useCount(); got != 1 {
		t.Errorf("after unspam, tag use_count = %d, want 1", got)
	}

	// Data that has spam set but is not hidden (an import from AO3 can have
	// that) is still kept out of sight.
	e.exec(`UPDATE works SET spam = true WHERE id = $1`, work)
	e.expect(404, "GET", "/api/v1/work/"+work.String(), "", nil)
	if _, body := e.do("GET", "/api/v1/search/works", strangerTok, nil); shows(body, work) {
		t.Errorf("search shows an imported spam work")
	}
}

func TestTagCountsExcludeHiddenWorks(t *testing.T) {
	e := newEnv(t)
	creator, _ := e.user("creator")
	_, adminTok := e.user("boss", "admin")
	w1 := e.work(creator, "One")
	w2 := e.work(creator, "Two")
	e.work(creator, "Three")
	if got := e.useCount(); got != 3 {
		t.Fatalf("use_count = %d, want 3", got)
	}

	e.expect(200, "POST", "/api/v1/admin/works/"+w1.String()+"/hide", adminTok, map[string]string{"reason": "r"})
	if got := e.useCount(); got != 2 {
		t.Errorf("after hiding one, use_count = %d, want 2", got)
	}
	// The tag page and its total agree.
	var page struct {
		Total int `json:"total"`
	}
	body := e.expect(200, "GET", "/api/v1/tags/"+e.tagID()+"/works", "", nil)
	json.Unmarshal([]byte(body), &page)
	if page.Total != 2 || shows(body, w1) {
		t.Errorf("tag page total = %d (want 2), shows hidden work: %v", page.Total, shows(body, w1))
	}
	// Tag statistics read the same view.
	var viaView int
	e.db.QueryRow(`SELECT count(*) FROM visible_work_tags WHERE tag_id = $1`, e.tagID()).Scan(&viaView)
	if viaView != 2 {
		t.Errorf("visible_work_tags has %d rows, want 2", viaView)
	}
	var stat int
	e.db.QueryRow(`SELECT actual_use_count FROM tag_stats WHERE name = 'Test Fandom'`).Scan(&stat)
	if stat != 2 {
		t.Errorf("tag_stats.actual_use_count = %d, want 2", stat)
	}

	// Hiding twice, or unhiding what isn't hidden, changes nothing.
	e.expect(409, "POST", "/api/v1/admin/works/"+w1.String()+"/hide", adminTok, map[string]string{"reason": "r"})
	e.expect(409, "POST", "/api/v1/admin/works/"+w2.String()+"/unhide", adminTok, nil)
	if got := e.useCount(); got != 2 {
		t.Errorf("use_count = %d, want 2", got)
	}

	// Deleting a hidden work doesn't take another count away.
	e.exec(`DELETE FROM work_tags WHERE work_id = $1`, w1)
	e.exec(`DELETE FROM works WHERE id = $1`, w1)
	if got := e.useCount(); got != 2 {
		t.Errorf("after deleting the hidden work, use_count = %d, want 2", got)
	}
	// Deleting a visible one does.
	e.exec(`DELETE FROM work_tags WHERE work_id = $1`, w2)
	e.exec(`DELETE FROM works WHERE id = $1`, w2)
	if got := e.useCount(); got != 1 {
		t.Errorf("after deleting a visible work, use_count = %d, want 1", got)
	}
	// A tag attached to a work that is already hidden isn't counted.
	hidden := e.work(creator, "Four")
	e.expect(200, "POST", "/api/v1/admin/works/"+hidden.String()+"/hide", adminTok, map[string]string{"reason": "r"})
	if got := e.useCount(); got != 1 {
		t.Errorf("use_count = %d, want 1", got)
	}
	e.exec(`INSERT INTO tags (name, type) VALUES ('Other', 'freeform')`)
	e.exec(`INSERT INTO work_tags (work_id, tag_id) SELECT $1, id FROM tags WHERE name = 'Other'`, hidden)
	var other int
	e.db.QueryRow(`SELECT use_count FROM tags WHERE name = 'Other'`).Scan(&other)
	if other != 0 {
		t.Errorf("a tag on a hidden work counts %d, want 0", other)
	}
	// Facet counts in search leave hidden works out too.
	_, body = e.do("GET", "/api/v1/search/works", "", nil)
	if strings.Contains(body, `"count":3`) || strings.Contains(body, `"count":2`) {
		t.Errorf("search facets count too many works: %s", body)
	}
}

func TestUnreviewedCommentsAreForTheCreatorUntilApproved(t *testing.T) {
	e := newEnv(t)
	creator, creatorTok := e.user("creator")
	_, strangerTok := e.user("stranger")
	_, adminTok := e.user("boss", "admin")
	work := e.work(creator, "Moderated")
	e.exec(`UPDATE works SET moderate_comments = true WHERE id = $1`, work)
	path := "/api/v1/work/" + work.String() + "/comments"

	// A guest comments; it waits for review.
	var made struct {
		ID uuid.UUID `json:"id"`
	}
	body := e.expect(201, "POST", path, "", map[string]any{"content": "nice work", "guest_name": "Guesty"})
	json.Unmarshal([]byte(body), &made)
	var unreviewed bool
	e.db.QueryRow(`SELECT unreviewed FROM comments WHERE id = $1`, made.ID).Scan(&unreviewed)
	if !unreviewed {
		t.Fatal("a guest comment on a moderated work should be unreviewed")
	}
	// The creator's own comment is never held.
	body = e.expect(201, "POST", path, creatorTok, map[string]any{"content": "thanks all"})
	var mine struct {
		ID uuid.UUID `json:"id"`
	}
	json.Unmarshal([]byte(body), &mine)
	e.db.QueryRow(`SELECT unreviewed FROM comments WHERE id = $1`, mine.ID).Scan(&unreviewed)
	if unreviewed {
		t.Error("the creator's own comment should not wait for review")
	}

	for tok, want := range map[string]bool{"": false, strangerTok: false, creatorTok: true, adminTok: true} {
		_, body := e.do("GET", path, tok, nil)
		if got := shows(body, made.ID); got != want {
			t.Errorf("reader %q sees the unreviewed comment: %v, want %v", tok, got, want)
		}
	}
	// You can't reply to it while it is waiting.
	out := e.expect(403, "POST", path, creatorTok, map[string]any{"content": "reply", "parent_comment_id": made.ID})
	if !strings.Contains(out, "comment_unreviewed") {
		t.Errorf("reply to an unreviewed comment: %s", out)
	}

	// The creator approves it; now everyone sees it.
	e.expect(200, "PUT", "/api/v1/comments/"+made.ID.String()+"/moderate", creatorTok, map[string]string{"status": "published"})
	for _, tok := range []string{"", strangerTok} {
		if _, body := e.do("GET", path, tok, nil); !shows(body, made.ID) {
			t.Errorf("reader %q can't see the approved comment: %s", tok, body)
		}
	}

	// An admin can approve one too, and it is audited.
	body = e.expect(201, "POST", path, "", map[string]any{"content": "second", "guest_name": "Guesty"})
	json.Unmarshal([]byte(body), &made)
	before := e.auditRows()
	e.expect(200, "POST", "/api/v1/admin/comments/"+made.ID.String()+"/approve", adminTok, nil)
	if e.auditRows() != before+1 {
		t.Errorf("approve wrote %d audit rows, want 1", e.auditRows()-before)
	}
	if _, body := e.do("GET", path, "", nil); !shows(body, made.ID) {
		t.Errorf("the admin-approved comment is not public: %s", body)
	}
	// Strangers can't approve.
	e.expect(403, "POST", "/api/v1/admin/comments/"+made.ID.String()+"/approve", strangerTok, nil)
}

func TestHiddenAndSpamComments(t *testing.T) {
	e := newEnv(t)
	creator, creatorTok := e.user("creator")
	_, strangerTok := e.user("stranger")
	_, modTok := e.user("mod", "moderator")
	work := e.work(creator, "Comments")
	path := "/api/v1/work/" + work.String() + "/comments"

	post := func(content string, parent *uuid.UUID) uuid.UUID {
		req := map[string]any{"content": content, "guest_name": "Guesty"}
		if parent != nil {
			req["parent_comment_id"] = *parent
		}
		var c struct {
			ID uuid.UUID `json:"id"`
		}
		json.Unmarshal([]byte(e.expect(201, "POST", path, "", req)), &c)
		return c.ID
	}
	top := post("a top comment", nil)
	reply := post("a reply", &top)

	// Hidden: the comment stays in the thread but its text is withheld.
	e.expect(200, "POST", "/api/v1/admin/comments/"+top.String()+"/hide", modTok, map[string]string{"reason": "abuse"})
	for _, tok := range []string{"", strangerTok, creatorTok} {
		_, body := e.do("GET", path, tok, nil)
		if strings.Contains(body, "a top comment") {
			t.Errorf("reader %q can read a hidden comment: %s", tok, body)
		}
		if !shows(body, top) || !shows(body, reply) {
			t.Errorf("reader %q should still see the thread structure: %s", tok, body)
		}
	}
	if _, body := e.do("GET", path, modTok, nil); !strings.Contains(body, "a top comment") {
		t.Errorf("a moderator should read the hidden comment: %s", body)
	}
	out := e.expect(403, "POST", path, strangerTok, map[string]any{"content": "x", "parent_comment_id": top})
	if !strings.Contains(out, "comment_hidden") {
		t.Errorf("reply to a hidden comment: %s", out)
	}
	e.expect(200, "POST", "/api/v1/admin/comments/"+top.String()+"/unhide", modTok, nil)
	if _, body := e.do("GET", path, "", nil); !strings.Contains(body, "a top comment") {
		t.Errorf("an unhidden comment should be readable again: %s", body)
	}

	// Spam: gone for everyone but admins, and its replies go with it.
	e.expect(200, "POST", "/api/v1/admin/comments/"+top.String()+"/spam", modTok, map[string]string{"reason": "spam"})
	for _, tok := range []string{"", strangerTok, creatorTok} {
		_, body := e.do("GET", path, tok, nil)
		if shows(body, top) || shows(body, reply) {
			t.Errorf("reader %q sees a spam comment or its reply: %s", tok, body)
		}
	}
	if _, body := e.do("GET", path, modTok, nil); !shows(body, top) || !shows(body, reply) {
		t.Errorf("a moderator should see the spam thread: %s", body)
	}
	e.expect(200, "POST", "/api/v1/admin/comments/"+top.String()+"/unspam", modTok, nil)
	if _, body := e.do("GET", path, "", nil); !shows(body, reply) {
		t.Errorf("after unspam the thread is back: %s", body)
	}
}

func TestFrozenThreadsRejectReplies(t *testing.T) {
	e := newEnv(t)
	creator, creatorTok := e.user("creator")
	_, strangerTok := e.user("stranger")
	_, modTok := e.user("mod", "moderator")
	work := e.work(creator, "Frozen")
	path := "/api/v1/work/" + work.String() + "/comments"
	post := func(parent *uuid.UUID, want int) (uuid.UUID, string) {
		req := map[string]any{"content": "words " + uuid.NewString(), "guest_name": "Guesty"}
		if parent != nil {
			req["parent_comment_id"] = *parent
		}
		var c struct {
			ID uuid.UUID `json:"id"`
		}
		body := e.expect(want, "POST", path, "", req)
		json.Unmarshal([]byte(body), &c)
		return c.ID, body
	}
	top, _ := post(nil, 201)
	reply, _ := post(&top, 201)

	// Strangers can't freeze.
	e.expect(403, "PUT", "/api/v1/comments/"+top.String()+"/freeze", strangerTok, nil)
	// The work's creator can; it covers the replies too.
	e.expect(200, "PUT", "/api/v1/comments/"+top.String()+"/freeze", creatorTok, nil)
	for _, parent := range []uuid.UUID{top, reply} {
		_, body := post(&parent, 403)
		if !strings.Contains(body, "thread_frozen") {
			t.Errorf("reply to a frozen comment: %s", body)
		}
	}
	// New top-level comments are not affected.
	post(nil, 201)
	// Freezing twice says so.
	e.expect(409, "PUT", "/api/v1/comments/"+top.String()+"/freeze", creatorTok, nil)

	e.expect(200, "PUT", "/api/v1/comments/"+top.String()+"/unfreeze", creatorTok, nil)
	post(&reply, 201)

	// A moderator's freeze is an audited admin action.
	before := e.auditRows()
	e.expect(200, "PUT", "/api/v1/comments/"+top.String()+"/freeze", modTok, nil)
	if e.auditRows() != before+1 {
		t.Errorf("a moderator's freeze wrote %d audit rows, want 1", e.auditRows()-before)
	}
	post(&top, 403)
}

func TestBannedUserCannotPostButCanRead(t *testing.T) {
	e := newEnv(t)
	creator, _ := e.user("creator")
	banned, bannedTok := e.user("banned")
	_, modTok := e.user("mod", "moderator")
	work := e.work(creator, "Somebody's work")
	own := e.work(banned, "Their own")

	e.expect(200, "POST", "/api/v1/admin/users/"+banned.String()+"/ban", modTok, map[string]string{"reason": "harassment"})

	workBody := map[string]any{"title": "T", "language": "en", "rating": "general", "fandoms": []string{"F"}, "chapter_content": "text"}
	refused := func(method, path string, body any) {
		t.Helper()
		out := e.expect(403, method, path, bannedTok, body)
		if !strings.Contains(out, "account_banned") {
			t.Errorf("%s %s: %s (want account_banned)", method, path, out)
		}
	}
	refused("POST", "/api/v1/works", workBody)
	refused("PUT", "/api/v1/works/"+own.String(), map[string]any{"title": "new"})
	refused("POST", "/api/v1/works/"+own.String()+"/chapters", map[string]any{"content": "x"})
	refused("POST", "/api/v1/work/"+work.String()+"/comments", map[string]any{"content": "hi"})
	refused("POST", "/api/v1/works/"+work.String()+"/kudos", nil)
	refused("POST", "/api/v1/works/"+work.String()+"/bookmark", map[string]any{})
	refused("POST", "/api/v1/series", map[string]any{"title": "s"})
	refused("POST", "/api/v1/collections", map[string]any{"name": "c", "title": "c"})
	refused("POST", "/api/v1/pseuds", map[string]any{"name": "other"})

	// Reading is fine.
	e.expect(200, "GET", "/api/v1/work/"+work.String(), bannedTok, nil)
	e.expect(200, "GET", "/api/v1/works", bannedTok, nil)
	// AO3 lets a banned user delete their own work (only a suspension stops
	// that: application_controller.rb:452).
	code, out := e.do("DELETE", "/api/v1/works/"+own.String(), bannedTok, nil)
	if code == http.StatusForbidden && strings.Contains(out, "account_") {
		t.Errorf("a banned user is refused deleting their work: %s", out)
	}

	// Other people are not affected.
	e.expect(201, "POST", "/api/v1/work/"+work.String()+"/comments", "", map[string]any{"content": "hi", "guest_name": "G"})

	// Unbanning lets them post again.
	e.expect(200, "POST", "/api/v1/admin/users/"+banned.String()+"/unban", modTok, nil)
	e.expect(201, "POST", "/api/v1/work/"+work.String()+"/comments", bannedTok, map[string]any{"content": "hi"})
}

func TestSuspensionEndsOnTime(t *testing.T) {
	e := newEnv(t)
	creator, _ := e.user("creator")
	susp, suspTok := e.user("suspended")
	_, modTok := e.user("mod", "moderator")
	work := e.work(creator, "Somebody's work")
	own := e.work(susp, "Their own")

	// A suspension must end in the future, and needs a reason.
	e.expect(400, "POST", "/api/v1/admin/users/"+susp.String()+"/suspend", modTok,
		map[string]string{"until": "2020-01-01", "reason": "r"})
	e.expect(400, "POST", "/api/v1/admin/users/"+susp.String()+"/suspend", modTok,
		map[string]string{"until": "2099-01-01"})
	e.expect(200, "POST", "/api/v1/admin/users/"+susp.String()+"/suspend", modTok,
		map[string]string{"until": time.Now().Add(48 * time.Hour).Format("2006-01-02"), "reason": "spamming"})

	body := e.expect(403, "POST", "/api/v1/work/"+work.String()+"/comments", suspTok, map[string]any{"content": "hi"})
	var refusal struct {
		Error          string `json:"error"`
		SuspendedUntil string `json:"suspended_until"`
	}
	json.Unmarshal([]byte(body), &refusal)
	if refusal.Error != "account_suspended" || refusal.SuspendedUntil == "" {
		t.Errorf("suspended refusal: %s", body)
	}
	// A suspended user can't delete their work either (a ban doesn't stop that).
	out := e.expect(403, "DELETE", "/api/v1/works/"+own.String(), suspTok, nil)
	if !strings.Contains(out, "account_suspended") {
		t.Errorf("delete while suspended: %s", out)
	}
	e.expect(200, "GET", "/api/v1/work/"+work.String(), suspTok, nil)

	// Once the end time has passed the suspension no longer applies, whether
	// or not anything cleared the flag.
	e.exec(`UPDATE users SET suspended_until = now() - interval '1 minute' WHERE id = $1`, susp)
	var flag bool
	e.db.QueryRow(`SELECT suspended FROM users WHERE id = $1`, susp).Scan(&flag)
	if !flag {
		t.Fatal("test setup: the suspended flag should still be set")
	}
	e.expect(201, "POST", "/api/v1/work/"+work.String()+"/comments", suspTok, map[string]any{"content": "hi again"})
	var standing sql.NullString
	e.db.QueryRow(`SELECT user_standing($1)`, susp).Scan(&standing)
	if standing.Valid {
		t.Errorf("user_standing after expiry = %q, want NULL", standing.String)
	}
	// And a suspension with no end time lasts until it is lifted.
	e.exec(`UPDATE users SET suspended_until = NULL WHERE id = $1`, susp)
	e.expect(403, "POST", "/api/v1/work/"+work.String()+"/comments", suspTok, map[string]any{"content": "hi"})
	e.expect(200, "POST", "/api/v1/admin/users/"+susp.String()+"/unsuspend", modTok, nil)
	e.expect(201, "POST", "/api/v1/work/"+work.String()+"/comments", suspTok, map[string]any{"content": "back"})
}

func TestBanAndSuspendEndSessions(t *testing.T) {
	e := newEnv(t)
	u, _ := e.user("target")
	_, modTok := e.user("mod", "moderator")
	e.exec(`INSERT INTO user_sessions (user_id, ip_address) VALUES ($1, '127.0.0.1'), ($1, '127.0.0.2')`, u)
	e.expect(200, "POST", "/api/v1/admin/users/"+u.String()+"/ban", modTok, map[string]string{"reason": "r"})
	var live int
	e.db.QueryRow(`SELECT count(*) FROM user_sessions WHERE user_id = $1 AND is_active`, u).Scan(&live)
	if live != 0 {
		t.Errorf("%d sessions still live after a ban", live)
	}
	// Banning someone already banned, and the protected account, are refused.
	e.expect(409, "POST", "/api/v1/admin/users/"+u.String()+"/ban", modTok, map[string]string{"reason": "r"})
	e.expect(422, "POST", "/api/v1/admin/users/00000000-0000-0000-0000-000000000000/ban", modTok, map[string]string{"reason": "r"})
}

func TestEachAdminActionWritesExactlyOneAuditRow(t *testing.T) {
	e := newEnv(t)
	creator, _ := e.user("creator")
	target, _ := e.user("target")
	_, modTok := e.user("mod", "moderator")
	work := e.work(creator, "W")
	chapter := uuid.New()
	e.exec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, published_at)
		VALUES ($1, $2, 2, 'two', 'x', false, now())`, chapter, work)
	series, bookmark, comment := uuid.New(), uuid.New(), uuid.New()
	e.exec(`INSERT INTO series (id, title, user_id) VALUES ($1, 'S', $2)`, series, creator)
	e.exec(`INSERT INTO bookmarks (id, work_id, user_id) VALUES ($1, $2, $3)`, bookmark, work, target)
	e.exec(`INSERT INTO comments (id, work_id, content, guest_name) VALUES ($1, $2, 'c', 'G')`, comment, work)
	e.exec(`UPDATE comments SET unreviewed = true WHERE id = $1`, comment)

	w, c, s, b, u := work.String(), comment.String(), series.String(), bookmark.String(), target.String()
	until := time.Now().Add(72 * time.Hour).Format("2006-01-02")
	steps := []struct {
		name, path string
		body       any
	}{
		{"hide work", "/api/v1/admin/works/" + w + "/hide", map[string]string{"reason": "r"}},
		{"unhide work", "/api/v1/admin/works/" + w + "/unhide", nil},
		{"spam work", "/api/v1/admin/works/" + w + "/spam", map[string]string{"reason": "r"}},
		{"unspam work", "/api/v1/admin/works/" + w + "/unspam", nil},
		{"hide chapter", "/api/v1/admin/works/" + w + "/chapters/" + chapter.String() + "/hide", map[string]string{"reason": "r"}},
		{"unhide chapter", "/api/v1/admin/works/" + w + "/chapters/" + chapter.String() + "/unhide", nil},
		{"hide series", "/api/v1/admin/series/" + s + "/hide", map[string]string{"reason": "r"}},
		{"unhide series", "/api/v1/admin/series/" + s + "/unhide", nil},
		{"hide bookmark", "/api/v1/admin/bookmarks/" + b + "/hide", map[string]string{"reason": "r"}},
		{"unhide bookmark", "/api/v1/admin/bookmarks/" + b + "/unhide", nil},
		{"hide comment", "/api/v1/admin/comments/" + c + "/hide", map[string]string{"reason": "r"}},
		{"unhide comment", "/api/v1/admin/comments/" + c + "/unhide", nil},
		{"spam comment", "/api/v1/admin/comments/" + c + "/spam", map[string]string{"reason": "r"}},
		{"unspam comment", "/api/v1/admin/comments/" + c + "/unspam", nil},
		{"approve comment", "/api/v1/admin/comments/" + c + "/approve", nil},
		{"freeze comment", "/api/v1/admin/comments/" + c + "/freeze", nil},
		{"unfreeze comment", "/api/v1/admin/comments/" + c + "/unfreeze", nil},
		{"suspend", "/api/v1/admin/users/" + u + "/suspend", map[string]string{"until": until, "reason": "r"}},
		{"unsuspend", "/api/v1/admin/users/" + u + "/unsuspend", nil},
		{"ban", "/api/v1/admin/users/" + u + "/ban", map[string]string{"reason": "r"}},
		{"unban", "/api/v1/admin/users/" + u + "/unban", nil},
	}
	for _, st := range steps {
		before := e.auditRows()
		e.expect(200, "POST", st.path, modTok, st.body)
		if got := e.auditRows() - before; got != 1 {
			t.Errorf("%s wrote %d audit rows, want 1", st.name, got)
		}
	}

	// The rows say who, what and why.
	var actor, action, targetType, targetID, reason string
	var actorID uuid.UUID
	e.db.QueryRow(`SELECT actor_label, actor_user_id, action, target_type, target_id, COALESCE(reason, '')
		FROM admin_actions ORDER BY id LIMIT 1`).Scan(&actor, &actorID, &action, &targetType, &targetID, &reason)
	if actor != "mod" || action != "hide" || targetType != "work" || targetID != w || reason != "r" {
		t.Errorf("first audit row = %s %s %s %s %q", actor, action, targetType, targetID, reason)
	}

	// Actions that change nothing, fail, or are refused write nothing.
	before := e.auditRows()
	e.expect(409, "POST", "/api/v1/admin/works/"+w+"/unhide", modTok, nil)                                           // not hidden
	e.expect(400, "POST", "/api/v1/admin/works/"+w+"/hide", modTok, nil)                                             // no reason
	e.expect(404, "POST", "/api/v1/admin/works/"+uuid.NewString()+"/hide", modTok, map[string]string{"reason": "r"}) // no such work
	if e.auditRows() != before {
		t.Errorf("failed actions wrote %d audit rows", e.auditRows()-before)
	}
}

func TestAuditLogIsAppendOnly(t *testing.T) {
	e := newEnv(t)
	e.exec(`INSERT INTO admin_actions (actor_label, action, target_type, target_id) VALUES ('cli:test', 'hide', 'work', 'x')`)
	for _, stmt := range []string{
		`UPDATE admin_actions SET reason = 'edited'`,
		`UPDATE admin_actions SET action = 'unhide' WHERE id = 1`,
		`DELETE FROM admin_actions`,
		`DELETE FROM admin_actions WHERE id = 1`,
		`TRUNCATE admin_actions`,
	} {
		_, err := e.db.Exec(stmt)
		if err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Errorf("%s: err = %v, want an append-only refusal", stmt, err)
		}
	}
	if n := e.auditRows(); n != 1 {
		t.Errorf("%d rows left, want 1", n)
	}
	// Deleting an account that acted does not touch the log.
	id, _ := e.user("ex-admin")
	e.exec(`INSERT INTO admin_actions (actor_user_id, actor_label, action, target_type, target_id) VALUES ($1, 'ex-admin', 'ban', 'user', 'y')`, id)
	e.exec(`DELETE FROM users WHERE id = $1`, id)
	if n := e.auditRows(); n != 2 {
		t.Errorf("%d rows after deleting the actor's account, want 2", n)
	}
}

func TestNonAdminsGet403OnAdminEndpoints(t *testing.T) {
	e := newEnv(t)
	creator, _ := e.user("creator")
	target, _ := e.user("target")
	_, userTok := e.user("plainuser")
	_, wranglerTok := e.user("wrangler", "tag_wrangler")
	_, modTok := e.user("mod", "moderator")
	work := e.work(creator, "W")
	w, u := work.String(), target.String()

	moderation := []struct{ method, path string }{
		{"POST", "/api/v1/admin/works/" + w + "/hide"},
		{"POST", "/api/v1/admin/works/" + w + "/unhide"},
		{"POST", "/api/v1/admin/works/" + w + "/spam"},
		{"POST", "/api/v1/admin/works/" + w + "/unspam"},
		{"POST", "/api/v1/admin/works/" + w + "/chapters/" + uuid.NewString() + "/hide"},
		{"POST", "/api/v1/admin/series/" + uuid.NewString() + "/hide"},
		{"POST", "/api/v1/admin/bookmarks/" + uuid.NewString() + "/hide"},
		{"POST", "/api/v1/admin/comments/" + uuid.NewString() + "/hide"},
		{"POST", "/api/v1/admin/comments/" + uuid.NewString() + "/spam"},
		{"POST", "/api/v1/admin/comments/" + uuid.NewString() + "/approve"},
		{"POST", "/api/v1/admin/users/" + u + "/suspend"},
		{"POST", "/api/v1/admin/users/" + u + "/unsuspend"},
		{"POST", "/api/v1/admin/users/" + u + "/ban"},
		{"POST", "/api/v1/admin/users/" + u + "/unban"},
	}
	before := e.auditRows()
	for _, tok := range []string{userTok, wranglerTok} {
		for _, r := range moderation {
			e.expect(403, r.method, r.path, tok, map[string]string{"reason": "r", "until": "2099-01-01"})
		}
		e.expect(403, "GET", "/api/v1/admin/audit", tok, nil)
	}
	// Logged-out requests are asked to log in.
	for _, r := range moderation {
		e.expect(401, r.method, r.path, "", map[string]string{"reason": "r"})
	}
	e.expect(401, "GET", "/api/v1/admin/audit", "", nil)
	// Moderators can moderate but not read the audit log (administrators only).
	e.expect(403, "GET", "/api/v1/admin/audit", modTok, nil)
	if e.auditRows() != before {
		t.Errorf("refused requests wrote %d audit rows", e.auditRows()-before)
	}
	var hidden bool
	e.db.QueryRow(`SELECT hidden_by_admin FROM works WHERE id = $1`, work).Scan(&hidden)
	if hidden {
		t.Error("a refused request hid the work")
	}
}

func TestAuditEndpoint(t *testing.T) {
	e := newEnv(t)
	creator, _ := e.user("creator")
	_, adminTok := e.user("boss", "admin")
	w := e.work(creator, "W")
	e.expect(200, "POST", "/api/v1/admin/works/"+w.String()+"/hide", adminTok, map[string]string{"reason": "because"})
	e.expect(200, "POST", "/api/v1/admin/works/"+w.String()+"/unhide", adminTok, nil)

	var out struct {
		Actions []struct {
			Actor, Action, TargetType, TargetID, Reason string
		} `json:"actions"`
	}
	body := e.expect(200, "GET", "/api/v1/admin/audit?limit=10", adminTok, nil)
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Actions) != 2 || out.Actions[0].Action != "unhide" || out.Actions[1].Action != "hide" ||
		out.Actions[1].Actor != "boss" || out.Actions[1].Reason != "because" {
		t.Errorf("audit listing: %s", body)
	}
	e.expect(200, "GET", "/api/v1/admin/audit?limit=1", adminTok, nil)
	body = e.expect(200, "GET", "/api/v1/admin/audit?since=2999-01-01", adminTok, nil)
	if !strings.Contains(body, `"actions":[]`) {
		t.Errorf("since in the future: %s", body)
	}
	e.expect(400, "GET", "/api/v1/admin/audit?since=yesterday", adminTok, nil)
	e.expect(400, "GET", "/api/v1/admin/audit?limit=0", adminTok, nil)
}

// A block does not hide the blocker's works. The blocked reader and a
// moderator both read the work.
func TestBlocksDoNotHideWorks(t *testing.T) {
	e := newEnv(t)
	creator, _ := e.user("creator")
	blocked, blockedTok := e.user("blocked")
	mod, modTok := e.user("mod", "moderator")
	work := e.work(creator, "Blocked from this")
	e.exec(`INSERT INTO user_blocks (blocker_id, blocked_id, pseud_id, pseud_name)
		SELECT $1, u.id, p.id, p.name FROM users u JOIN pseuds p ON p.user_id = u.id AND p.is_default
		WHERE u.id IN ($2, $3)`, creator, blocked, mod)
	e.expect(200, "GET", "/api/v1/work/"+work.String(), blockedTok, nil)
	e.expect(200, "GET", "/api/v1/work/"+work.String(), modTok, nil)
}
