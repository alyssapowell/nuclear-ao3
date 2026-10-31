package works_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// On a work that hides its creator, a block must not show. The blocked person
// sees what every other reader sees, and what they write is kept aside
// ("shadowed"): visible to them and to moderators, counted nowhere, and
// announced to nobody.

var hidingModes = []string{"is_anonymous", "in_anon_collection", "in_unrevealed_collection"}

func (w *blockWorld) hide(mode string) {
	w.e.t.Helper()
	w.e.exec(`UPDATE works SET `+mode+` = true WHERE id = $1`, w.work)
}

func (w *blockWorld) workCount(col string) int {
	return w.e.count(`SELECT `+col+` FROM works WHERE id = $1`, w.work)
}

func sortedKeys(t *testing.T, body string) []string {
	t.Helper()
	var out []string
	for k := range keysOf(t, body) {
		out = append(out, k)
	}
	// keysOf returns a set; order does not matter for comparing shapes
	return sortStrings(out)
}

func sortStrings(in []string) []string {
	for i := range in {
		for j := i + 1; j < len(in); j++ {
			if in[j] < in[i] {
				in[i], in[j] = in[j], in[i]
			}
		}
	}
	return in
}

func TestBlockIsSilentOnWorksThatHideTheirCreator(t *testing.T) {
	for _, mode := range hidingModes {
		t.Run(mode, func(t *testing.T) {
			w := newBlockWorld(t)
			e := w.e
			w.hide(mode)
			w.blockEx()
			_, modTok := e.user("mod", "moderator")
			otherID, otherTok := e.user("other_acct")
			_ = otherID
			base := "/api/v1/work/" + w.work.String()

			// Reading: same list, same count, same fields, no flag.
			blockedList := e.expect(200, "GET", base+"/comments", w.blockedTok, nil)
			readerList := e.expect(200, "GET", base+"/comments", w.readerTok, nil)
			if blockedList != readerList {
				t.Errorf("the blocked person's comment list differs:\nblocked: %s\nreader:  %s", blockedList, readerList)
			}
			if strings.Contains(blockedList, "comments_disabled_for_you") {
				t.Errorf("flag leaked: %s", blockedList)
			}
			if len(commentTexts(t, blockedList)) != 3 {
				t.Errorf("want the 3 real comments: %s", blockedList)
			}
			w.noLeak("blocked list", blockedList)

			blockedPage := e.expect(200, "GET", base, w.blockedTok, nil)
			readerPage := e.expect(200, "GET", base, w.readerTok, nil)
			if blockedPage != readerPage {
				t.Errorf("the work page differs for the blocked person:\n%s\n%s", blockedPage, readerPage)
			}
			if strings.Contains(blockedPage, "comments_disabled_for_you") || !strings.Contains(blockedPage, `"comments":3`) {
				t.Errorf("work page: %s", blockedPage)
			}
			if a, b := e.expect(200, "GET", base+"/stats", w.blockedTok, nil), e.expect(200, "GET", base+"/stats", w.readerTok, nil); a != b {
				t.Errorf("stats differ: %s vs %s", a, b)
			}

			// A comment from the blocked person: the normal answer.
			normal := e.expect(201, "POST", base+"/comments", w.readerTok, map[string]any{"content": "from the reader"})
			e.exec(`DELETE FROM notifications`)
			beforeCount := w.workCount("comment_count")
			posted := e.expect(201, "POST", base+"/comments", w.blockedTok, map[string]any{"content": "from the blocked one"})
			if !reflect.DeepEqual(sortedKeys(t, posted), sortedKeys(t, normal)) {
				t.Errorf("the answer has a different shape:\n%s\n%s", posted, normal)
			}
			w.noLeak("blocked comment answer", posted)
			reply := e.expect(201, "POST", base+"/comments", w.blockedTok,
				map[string]any{"content": "blocked reply", "parent_comment_id": w.cReader})
			_ = reply

			// Invisible to the creator and to other readers; visible to the
			// blocked person and to a moderator.
			for _, c := range []string{"from the blocked one", "blocked reply"} {
				for name, tok := range map[string]string{"creator": w.blockerTok, "reader": w.readerTok, "other": otherTok, "guest": ""} {
					if has(w.comments(w.work, tok), c) {
						t.Errorf("%s sees the shadowed comment %q", name, c)
					}
				}
				for name, tok := range map[string]string{"blocked": w.blockedTok, "moderator": modTok} {
					if !has(w.comments(w.work, tok), c) {
						t.Errorf("%s can't see their shadowed comment %q", name, c)
					}
				}
			}
			if !has(w.comments(w.work, w.readerTok), "from the reader") {
				t.Errorf("the ordinary comment should be public")
			}

			// Nobody is told, and the public count only moved for the reader's comment.
			// (notifications were cleared before the blocked person wrote)
			if n := e.count(`SELECT count(*) FROM notifications`); n != 0 {
				t.Errorf("notifications after shadowed comments = %d, want 0", n)
			}
			if got := w.workCount("comment_count"); got != beforeCount {
				t.Errorf("public comment count moved from %d to %d", beforeCount, got)
			}
			if n := e.count(`SELECT count(*) FROM comments WHERE work_id = $1 AND shadowed`, w.work); n != 2 {
				t.Errorf("shadowed comments = %d, want 2", n)
			}
			if page := e.expect(200, "GET", base, w.readerTok, nil); !strings.Contains(page, fmt.Sprintf(`"comments":%d`, beforeCount)) {
				t.Errorf("the public page count should be %d: %s", beforeCount, page)
			}

			// Editing their own old, public comment goes through and shadows it.
			beforeEdit := w.workCount("comment_count")
			edited := e.expect(200, "PUT", "/api/v1/comments/"+w.cBlockedOld.String(), w.blockedTok, map[string]any{"content": "edited quietly"})
			if strings.Contains(edited, "comments_disabled") {
				t.Errorf("edit refused: %s", edited)
			}
			if has(w.comments(w.work, w.readerTok), "edited quietly") || has(w.comments(w.work, w.blockerTok), "edited quietly") {
				t.Errorf("an edited comment of a blocked person is still public")
			}
			if !has(w.comments(w.work, w.blockedTok), "edited quietly") || !has(w.comments(w.work, modTok), "edited quietly") {
				t.Errorf("the edited comment should stay visible to its author and moderators")
			}
			if got := w.workCount("comment_count"); got != beforeEdit-1 {
				t.Errorf("count after the edit = %d, want %d", got, beforeEdit-1)
			}

			// Kudos: the normal answer, never listed or counted.
			normalKudos := e.expect(201, "POST", "/api/v1/works/"+w.work.String()+"/kudos", w.readerTok, nil)
			beforeKudos := w.workCount("kudos_count")
			kudos := e.expect(201, "POST", "/api/v1/works/"+w.work.String()+"/kudos", w.blockedTok, nil)
			if kudos != normalKudos {
				t.Errorf("kudos answer differs: %s vs %s", kudos, normalKudos)
			}
			if got := w.workCount("kudos_count"); got != beforeKudos {
				t.Errorf("public kudos count moved from %d to %d", beforeKudos, got)
			}
			if n := e.count(`SELECT count(*) FROM kudos WHERE work_id = $1 AND shadowed`, w.work); n != 1 {
				t.Errorf("shadowed kudos = %d, want 1", n)
			}
			kl := e.expect(200, "GET", "/api/v1/work/"+w.work.String()+"/kudos", w.readerTok, nil)
			if strings.Contains(kl, "Ex") && strings.Contains(kl, `"name":"Ex"`) || !strings.Contains(kl, `"total_count":1`) {
				t.Errorf("the public kudos list shows the shadowed kudos: %s", kl)
			}
			own := e.expect(200, "GET", "/api/v1/work/"+w.work.String()+"/kudos", w.blockedTok, nil)
			if !strings.Contains(own, `"has_given_kudos":true`) || !strings.Contains(own, `"total_count":2`) {
				t.Errorf("the giver should see kudos as given, counted once more than the public count: %s", own)
			}
			if code, _ := e.do("POST", "/api/v1/works/"+w.work.String()+"/kudos", w.blockedTok, nil); code != 409 {
				t.Errorf("second kudos = %d, want the normal 409", code)
			}
		})
	}
}

// A work whose creator is shown keeps the visible behaviour; so does a block
// from a reader on a hidden-creator work (that commenter is named).
func TestBlockStaysVisibleWhenTheCreatorIsNamed(t *testing.T) {
	w := newBlockWorld(t)
	w.blockEx()
	code, out := w.e.do("POST", "/api/v1/work/"+w.work.String()+"/comments", w.blockedTok, map[string]any{"content": "hi"})
	if code != 403 || !strings.Contains(out, "comments_disabled") {
		t.Errorf("signed work: %d %s", code, out)
	}
	if n := w.e.count(`SELECT count(*) FROM comments WHERE shadowed`); n != 0 {
		t.Errorf("nothing should be shadowed on a signed work")
	}
}

// If only one of several creators blocked the person, and the work hides all
// of them, the block is still silent; a work that names its creators is not.
func TestCoCreatorBlockOnHiddenWork(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	// the reader is a second creator; only the blocker blocked the commenter
	e.exec(`INSERT INTO creatorships (creation_id, creation_type, pseud_id) VALUES ($1, 'Work', $2)`, w.work, w.rae)
	w.hide("is_anonymous")
	w.blockEx()
	e.expect(201, "POST", "/api/v1/work/"+w.work.String()+"/comments", w.blockedTok, map[string]any{"content": "quiet"})
	if has(w.comments(w.work, w.readerTok), "quiet") {
		t.Errorf("the other creator can see a shadowed comment")
	}
	// A creator is never subject to the block.
	if n := e.count(`SELECT count(*) FROM comments WHERE shadowed`); n != 1 {
		t.Errorf("shadowed = %d, want 1", n)
	}
}

// A shadowed reply to a hidden creator's own comment: the normal answer, not
// "replies_disabled", which would say the hidden creator blocked them.
func TestReplyToAHiddenCreatorsCommentIsSilent(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	w.hide("is_anonymous")
	w.blockEx()
	e.exec(`DELETE FROM notifications`)
	e.expect(201, "POST", "/api/v1/work/"+w.work.String()+"/comments", w.blockedTok,
		map[string]any{"content": "reply to creator", "parent_comment_id": w.cBlocker})
	if has(w.comments(w.work, w.blockerTok), "reply to creator") {
		t.Errorf("the creator sees the shadowed reply")
	}
	if n := e.count(`SELECT count(*) FROM notifications`); n != 0 {
		t.Errorf("a shadowed reply notified %d people", n)
	}
}

// A comment you can't see is 404 to you on every path that acts on one, with
// or without a block, and the block is never mentioned first.
func TestEditingSomeoneElsesHiddenCommentIs404(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	_, otherTok := e.user("other_acct")
	spam := e.comment(w.work, w.reader, "spammy")
	e.exec(`UPDATE comments SET is_spam = true WHERE id = $1`, spam)
	path := "/api/v1/comments/" + spam.String()

	// Without a block.
	for name, tok := range map[string]string{"blocked-to-be": w.blockedTok, "other": otherTok} {
		if code, out := e.do("PUT", path, tok, map[string]any{"content": "x"}); code != 404 {
			t.Errorf("%s before any block: %d %s", name, code, out)
		}
	}
	// With a block, on a signed work and on a hidden one.
	w.blockEx()
	for _, hidden := range []bool{false, true} {
		if hidden {
			w.hide("is_anonymous")
		}
		for name, tok := range map[string]string{"blocked": w.blockedTok, "other": otherTok} {
			if code, out := e.do("PUT", path, tok, map[string]any{"content": "x"}); code != 404 {
				t.Errorf("%s with a block (hidden=%v): %d %s", name, hidden, code, out)
			}
		}
		// A comment that is visible but not theirs is a plain 403, never the block's answer.
		if code, out := e.do("PUT", "/api/v1/comments/"+w.cBlocker.String(), otherTok, map[string]any{"content": "x"}); code != 403 || strings.Contains(out, "comments_disabled") {
			t.Errorf("someone else's visible comment: %d %s", code, out)
		}
	}
	// The blocked person cannot edit someone else's visible comment, and is not
	// told about the block when they try (signed work).
	w2 := newBlockWorld(t)
	w2.blockEx()
	if code, out := w2.e.do("PUT", "/api/v1/comments/"+w2.cReader.String(), w2.blockedTok, map[string]any{"content": "x"}); code != 404 {
		t.Errorf("blocked person on a comment they can't see: %d %s", code, out)
	}

	// A shadowed comment is 404 for everyone but its author, and a creator
	// can't delete, moderate or freeze what they can't see.
	w.e.expect(201, "POST", "/api/v1/work/"+w.work.String()+"/comments", w.blockedTok, map[string]any{"content": "quiet"})
	var shadow uuid.UUID
	if err := e.db.QueryRow(`SELECT id FROM comments WHERE shadowed LIMIT 1`).Scan(&shadow); err != nil {
		t.Fatal(err)
	}
	sp := "/api/v1/comments/" + shadow.String()
	for _, tc := range []struct{ method, path string }{
		{"PUT", sp}, {"DELETE", sp}, {"PUT", sp + "/moderate"}, {"PUT", sp + "/freeze"},
	} {
		for name, tok := range map[string]string{"creator": w.blockerTok, "other": otherTok, "reader": w.readerTok} {
			body := map[string]any{"content": "x", "status": "spam"}
			if code, out := e.do(tc.method, tc.path, tok, body); code != 404 {
				t.Errorf("%s %s as %s: %d %s", tc.method, tc.path, name, code, out)
			}
		}
	}
	if n := e.count(`SELECT count(*) FROM comments WHERE id = $1 AND NOT is_deleted AND shadowed`, shadow); n != 1 {
		t.Errorf("the shadowed comment was changed")
	}
	// Its author can delete it.
	e.expect(200, "DELETE", sp, w.blockedTok, nil)
}

// The delete path answers 404 for a comment a stranger can't see, not 403.
func TestDeleteChecksVisibilityFirst(t *testing.T) {
	e := newFixEnv(t)
	owner, _ := e.user("owner_acct")
	_, strangerTok := e.user("stranger_acct")
	author, _ := e.user("author_acct")
	work := e.work(owner, "Delete Tale", "published")
	hidden := e.comment(work, author, "spam")
	e.exec(`UPDATE comments SET is_spam = true WHERE id = $1`, hidden)
	visible := e.comment(work, author, "fine")
	if code, out := e.do("DELETE", "/api/v1/comments/"+hidden.String(), strangerTok, nil); code != 404 {
		t.Errorf("stranger deleting a hidden comment: %d %s", code, out)
	}
	if code, out := e.do("DELETE", "/api/v1/comments/"+visible.String(), strangerTok, nil); code != 403 {
		t.Errorf("stranger deleting a visible comment: %d %s", code, out)
	}
}

// A gift to a recipient who blocked a creator is answered like any gift.
func TestGiftToABlockerIsShadowed(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	w.blockEx()
	e.exec(`DELETE FROM notifications`)

	normal := e.expect(201, "POST", "/api/v1/works/"+w.work.String()+"/gift", w.blockerTok, map[string]any{"pseud_id": w.rae})
	var shape func(string) []string
	shape = func(b string) []string {
		var m map[string]map[string]any
		if err := json.Unmarshal([]byte(b), &m); err != nil {
			t.Fatal(err)
		}
		var ks []string
		for k := range m["gift"] {
			ks = append(ks, k)
		}
		return sortStrings(ks)
	}
	for _, p := range []uuid.UUID{w.ex, w.exTwo} {
		out := e.expect(201, "POST", "/api/v1/works/"+w.work.String()+"/gift", w.blockerTok, map[string]any{"pseud_id": p})
		if !reflect.DeepEqual(shape(out), shape(normal)) || strings.Contains(out, "blocked") || !strings.Contains(out, `"rejected":false`) {
			t.Errorf("gift answer to pseud %s: %s (normal: %s)", p, out, normal)
		}
	}
	// The other direction (the blocked person gifting to the blocker).
	e.expect(201, "POST", "/api/v1/works/"+w.blockedWork.String()+"/gift", w.blockedTok, map[string]any{"pseud_id": w.beeTwo})

	if n := e.count(`SELECT count(*) FROM gifts WHERE shadowed`); n != 3 {
		t.Errorf("shadowed gifts = %d, want 3", n)
	}
	list := func(work uuid.UUID, tok string) string {
		return e.expect(200, "GET", "/api/v1/works/"+work.String()+"/gifts", tok, nil)
	}
	for name, tok := range map[string]string{"recipient": w.blockedTok, "reader": w.readerTok} {
		if out := list(w.work, tok); strings.Contains(out, w.ex.String()) || strings.Contains(out, w.exTwo.String()) {
			t.Errorf("%s sees a shadowed gift: %s", name, out)
		}
	}
	if out := list(w.work, w.blockerTok); !strings.Contains(out, w.ex.String()) {
		t.Errorf("the giver should see their own gift: %s", out)
	}
	if out := list(w.work, w.readerTok); !strings.Contains(out, w.rae.String()) {
		t.Errorf("an ordinary gift should be listed: %s", out)
	}
	if n := e.count(`SELECT count(*) FROM notifications`); n != 0 {
		t.Errorf("a gift notified %d people", n)
	}
}

func TestGiftsAreLimitedPerAccount(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	got429 := -1
	for i := 1; i <= 35; i++ {
		code, out := e.do("POST", "/api/v1/works/"+w.work.String()+"/gift", w.blockerTok, map[string]any{"pseud_id": w.rae})
		if code == 429 {
			got429 = i
			if !strings.Contains(out, "rate_limit_exceeded") {
				t.Errorf("429 body: %s", out)
			}
			break
		}
		if code != 201 {
			t.Fatalf("gift %d: %d %s", i, code, out)
		}
	}
	if got429 != 31 {
		t.Errorf("first 429 at gift %d, want 31", got429)
	}
	// Another account is not affected, and a blocked recipient counts the same.
	e.expect(201, "POST", "/api/v1/works/"+w.blockedWork.String()+"/gift", w.blockedTok, map[string]any{"pseud_id": w.rae})
}
