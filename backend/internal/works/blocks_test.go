package works_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// A world with a blocker (account usernames and pseud names differ, so a
// leaked username can be told from a pseud name), the person they block, a
// bystander, and a work with a comment from each of them.
type blockWorld struct {
	e *fixEnv

	blocker, blocked, reader              uuid.UUID
	blockerTok, blockedTok, readerTok     string
	bee, beeTwo, ex, exTwo, rae           uuid.UUID // pseuds
	usernames                             []string
	work                                  uuid.UUID // the blocker's work
	readerWork                            uuid.UUID // the bystander's work
	blockedWork                           uuid.UUID // the blocked person's work
	cReader, cBlocker, cBlockedOld        uuid.UUID // comments on the blocker's work
	bOnReader, xOnReader, xReplyToBlocker uuid.UUID // comments on the bystander's work
}

func newBlockWorld(t *testing.T) *blockWorld {
	t.Helper()
	w := &blockWorld{e: newFixEnv(t)}
	e := w.e
	w.blocker, w.blockerTok = e.user("blocker_acct")
	w.blocked, w.blockedTok = e.user("blocked_acct")
	w.reader, w.readerTok = e.user("reader_acct")
	w.usernames = []string{"blocker_acct", "blocked_acct", "reader_acct"}
	e.exec(`UPDATE pseuds SET name = 'Bee' WHERE user_id = $1 AND is_default`, w.blocker)
	e.exec(`UPDATE pseuds SET name = 'Ex' WHERE user_id = $1 AND is_default`, w.blocked)
	e.exec(`UPDATE pseuds SET name = 'Rae' WHERE user_id = $1 AND is_default`, w.reader)
	w.bee, w.ex, w.rae = e.pseudID(w.blocker, "Bee"), e.pseudID(w.blocked, "Ex"), e.pseudID(w.reader, "Rae")
	w.beeTwo = e.newPseud(w.blocker, "Bee Two")
	w.exTwo = e.newPseud(w.blocked, "Ex Two")

	w.work = e.work(w.blocker, "Lighthouse Keepers", "published")
	w.readerWork = e.work(w.reader, "Reader Tale", "published")
	w.blockedWork = e.work(w.blocked, "Ex Story", "published")

	w.cReader = e.comment(w.work, w.reader, "reader says hello")
	w.cBlocker = e.comment(w.work, w.blocker, "blocker says thanks")
	w.cBlockedOld = w.commentAs(w.work, w.blocked, w.exTwo, "old comment from ex two", nil)
	w.bOnReader = e.comment(w.readerWork, w.blocker, "blocker on reader work")
	w.xOnReader = w.commentAs(w.readerWork, w.blocked, w.exTwo, "ex two on reader work", nil)
	w.xReplyToBlocker = w.commentAs(w.readerWork, w.blocked, w.ex, "ex replying to blocker", &w.bOnReader)
	return w
}

// commentAs writes a comment straight into the table (as an account's pseud).
func (w *blockWorld) commentAs(work, user, pseud uuid.UUID, content string, parent *uuid.UUID) uuid.UUID {
	w.e.t.Helper()
	id := uuid.New()
	w.e.exec(`INSERT INTO comments (id, work_id, user_id, pseudonym_id, parent_comment_id, content)
		VALUES ($1, $2, $3, $4, $5, $6)`, id, work, user, pseud, parent, content)
	return id
}

// blockEx has the blocker block the Ex pseud.
func (w *blockWorld) blockEx() {
	w.e.t.Helper()
	body := w.e.expect(201, "POST", "/api/v1/blocks", w.blockerTok, map[string]any{"pseud_id": w.ex})
	w.noLeak("create block", body, "Ex Two")
}

// noLeak fails if an answer has an account id, a username, an email, or any
// of the named extra strings (other pseuds the reader shouldn't know about).
func (w *blockWorld) noLeak(what, body string, extra ...string) {
	w.e.t.Helper()
	noAccount(w.e.t, what, body, w.blocker, w.blocked, w.reader)
	for _, u := range w.usernames {
		if strings.Contains(body, u) {
			w.e.t.Errorf("%s contains the username %q: %s", what, u, body)
		}
	}
	for _, s := range extra {
		if strings.Contains(body, s) {
			w.e.t.Errorf("%s contains %q: %s", what, s, body)
		}
	}
}

// texts returns the content of every comment in a comments answer.
func commentTexts(t *testing.T, body string) []string {
	t.Helper()
	var got struct {
		Comments []struct{ Content string } `json:"comments"`
	}
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("comments answer: %v: %s", err, body)
	}
	var out []string
	for _, c := range got.Comments {
		out = append(out, c.Content)
	}
	return out
}

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func (w *blockWorld) comments(work uuid.UUID, tok string) []string {
	w.e.t.Helper()
	body := w.e.expect(200, "GET", "/api/v1/work/"+work.String()+"/comments", tok, nil)
	w.noLeak("comments of "+work.String(), body)
	return commentTexts(w.e.t, body)
}

// The block API takes a pseud, lists the pseud picked as {id, name}, and
// never gives out the account, its username or its other pseuds.
func TestBlockAPINamesPseudsOnly(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e

	w.blockEx()
	// Blocking another pseud of the same account is an ordinary new entry.
	again := e.expect(201, "POST", "/api/v1/blocks", w.blockerTok, map[string]any{"pseud_id": w.exTwo})
	w.noLeak("second block", again, "Ex\"")
	// Blocking the same pseud twice changes nothing.
	e.expect(201, "POST", "/api/v1/blocks", w.blockerTok, map[string]any{"pseud_id": w.ex})

	list := e.expect(200, "GET", "/api/v1/blocks", w.blockerTok, nil)
	w.noLeak("block list", list)
	var got struct {
		Blocks []struct {
			Pseud pseudRef `json:"pseud"`
		}
	}
	json.Unmarshal([]byte(list), &got)
	if len(got.Blocks) != 2 {
		t.Fatalf("block list: %s", list)
	}
	names := map[string]string{}
	for _, b := range got.Blocks {
		names[b.Pseud.ID] = b.Pseud.Name
	}
	if names[w.ex.String()] != "Ex" || names[w.exTwo.String()] != "Ex Two" {
		t.Errorf("block list should show the pseuds picked: %s", list)
	}

	// Only the person who blocked sees the list, and it is theirs alone.
	other := e.expect(200, "GET", "/api/v1/blocks", w.readerTok, nil)
	if strings.Contains(other, "Ex") {
		t.Errorf("someone else's block list leaked: %s", other)
	}

	e.expect(404, "POST", "/api/v1/blocks", w.blockerTok, map[string]any{"pseud_id": uuid.New()})
	e.expect(400, "POST", "/api/v1/blocks", w.blockerTok, map[string]any{"pseud_id": w.beeTwo}) // yourself
	e.expect(400, "POST", "/api/v1/blocks", w.blockerTok, map[string]any{})
	e.expect(401, "POST", "/api/v1/blocks", "", map[string]any{"pseud_id": w.ex})

	// Removing a block matches only the pseud that was picked, so it can't be
	// used to ask whether another pseud belongs to the blocked account.
	e.expect(404, "DELETE", "/api/v1/blocks/"+w.rae.String(), w.blockerTok, nil)
	e.expect(200, "DELETE", "/api/v1/blocks/"+w.exTwo.String(), w.blockerTok, nil)
	// The Ex entry still blocks the whole account.
	e.expect(403, "POST", "/api/v1/work/"+w.work.String()+"/comments", w.blockedTok, map[string]any{"content": "still blocked"})
	e.expect(200, "DELETE", "/api/v1/blocks/"+w.ex.String(), w.blockerTok, nil)
	e.expect(201, "POST", "/api/v1/work/"+w.work.String()+"/comments", w.blockedTok, map[string]any{"content": "unblocked"})

	// A block survives its pseud being deleted, and the list keeps the name.
	e.expect(201, "POST", "/api/v1/blocks", w.blockerTok, map[string]any{"pseud_id": w.exTwo})
	e.expect(200, "DELETE", "/api/v1/pseuds/"+w.exTwo.String(), w.blockedTok, nil)
	after := e.expect(200, "GET", "/api/v1/blocks", w.blockerTok, nil)
	if !strings.Contains(after, `"name":"Ex Two"`) {
		t.Errorf("after the pseud was deleted the list lost it: %s", after)
	}
	e.expect(403, "POST", "/api/v1/work/"+w.work.String()+"/comments", w.blockedTok, map[string]any{"content": "still blocked"})
}

// The blocked person still finds and reads the blocker's work, under any
// pseud, but sees comments as disabled and can't post any.
func TestBlockedPersonStillReadsButCommentsAreDisabled(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	w.blockEx()

	// Found in the work list, on the pseud page, and readable with chapters.
	list := e.expect(200, "GET", "/api/v1/works?q=Lighthouse", w.blockedTok, nil)
	w.noLeak("work list", list)
	if !strings.Contains(list, w.work.String()) {
		t.Errorf("the blocked person should still find the work in lists: %s", list)
	}
	page := e.expect(200, "GET", "/api/v1/pseuds/"+w.bee.String()+"/works", w.blockedTok, nil)
	if !strings.Contains(page, w.work.String()) {
		t.Errorf("the blocked person should still find the work on its pseud page: %s", page)
	}
	work := e.expect(200, "GET", "/api/v1/work/"+w.work.String(), w.blockedTok, nil)
	w.noLeak("work page", work)
	if !strings.Contains(work, `"comments_disabled_for_you":true`) || !strings.Contains(work, `"comments":0`) {
		t.Errorf("work page should say comments are disabled for them, with no count: %s", work)
	}
	chapters := e.expect(200, "GET", "/api/v1/work/"+w.work.String()+"/chapters", w.blockedTok, nil)
	if !strings.Contains(chapters, "chapter text") {
		t.Errorf("the blocked person should read the chapters: %s", chapters)
	}
	e.expect(200, "GET", "/api/v1/work/"+w.work.String()+"/stats", w.blockedTok, nil)

	// No comments at all, theirs or anyone's.
	body := e.expect(200, "GET", "/api/v1/work/"+w.work.String()+"/comments", w.blockedTok, nil)
	w.noLeak("comment list", body)
	if len(commentTexts(t, body)) != 0 || !strings.Contains(body, `"comments_disabled_for_you":true`) {
		t.Errorf("the blocked person's comment list should be empty and flagged: %s", body)
	}

	// Posting, replying, editing and kudos: 403 comments_disabled.
	for _, tc := range []struct {
		name, method, path string
		body               any
		code               string
	}{
		{"new comment", "POST", "/api/v1/work/" + w.work.String() + "/comments", map[string]any{"content": "hi"}, "comments_disabled"},
		{"new comment under another pseud", "POST", "/api/v1/work/" + w.work.String() + "/comments",
			map[string]any{"content": "hi", "pseudonym_id": w.exTwo}, "comments_disabled"},
		{"reply", "POST", "/api/v1/work/" + w.work.String() + "/comments",
			map[string]any{"content": "hi", "parent_comment_id": w.cReader}, "comments_disabled"},
		{"chapter comment", "POST", "/api/v1/work/" + w.work.String() + "/comments",
			map[string]any{"content": "hi", "chapter_id": e.chapterOf(w.work)}, "comments_disabled"},
		{"edit an old comment", "PUT", "/api/v1/comments/" + w.cBlockedOld.String(), map[string]any{"content": "edited"}, "comments_disabled"},
		{"kudos", "POST", "/api/v1/works/" + w.work.String() + "/kudos", nil, "kudos_disabled"},
	} {
		code, out := e.do(tc.method, tc.path, w.blockedTok, tc.body)
		if code != 403 || !strings.Contains(out, tc.code) {
			t.Errorf("%s: status %d, want 403 %s: %s", tc.name, code, tc.code, out)
		}
		w.noLeak(tc.name, out)
	}
	if n := e.count(`SELECT count(*) FROM comments WHERE content IN ('hi', 'edited')`); n != 0 {
		t.Errorf("a blocked comment was saved")
	}
	if n := e.count(`SELECT count(*) FROM kudos WHERE work_id = $1`, w.work); n != 0 {
		t.Errorf("a blocked kudos was saved")
	}

	// Comments elsewhere are still open to them.
	e.expect(201, "POST", "/api/v1/work/"+w.readerWork.String()+"/comments", w.blockedTok, map[string]any{"content": "fine here"})
	e.expect(201, "POST", "/api/v1/works/"+w.readerWork.String()+"/kudos", w.blockedTok, nil)
}

func (e *fixEnv) chapterOf(work uuid.UUID) uuid.UUID {
	e.t.Helper()
	var id uuid.UUID
	if err := e.db.QueryRow(`SELECT id FROM chapters WHERE work_id = $1 LIMIT 1`, work).Scan(&id); err != nil {
		e.t.Fatal(err)
	}
	return id
}

// Everyone else is unaffected, and the blocker doesn't see the blocked
// account's comments, under any of its pseuds.
func TestOthersSeeCommentsAndTheBlockerDoesNotSeeBlockedOnes(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	w.blockEx() // only Ex is picked; Ex Two is the same account

	// A bystander sees everything, and the work isn't flagged for them.
	got := w.comments(w.work, w.readerTok)
	for _, want := range []string{"reader says hello", "blocker says thanks", "old comment from ex two"} {
		if !has(got, want) {
			t.Errorf("the bystander should see %q, got %v", want, got)
		}
	}
	if page := e.expect(200, "GET", "/api/v1/work/"+w.work.String(), w.readerTok, nil); strings.Contains(page, "comments_disabled_for_you") {
		t.Errorf("the bystander's work page is flagged: %s", page)
	}
	if guest := w.comments(w.work, ""); !has(guest, "old comment from ex two") {
		t.Errorf("a guest should still see the comments: %v", guest)
	}

	// The blocker's own work: the blocked account's comment is gone, under its
	// other pseud too.
	got = w.comments(w.work, w.blockerTok)
	if has(got, "old comment from ex two") {
		t.Errorf("the blocker sees a blocked account's comment: %v", got)
	}
	if !has(got, "reader says hello") || !has(got, "blocker says thanks") {
		t.Errorf("the blocker should still see everyone else's comments: %v", got)
	}

	// Elsewhere: on the bystander's work the blocker sees neither the blocked
	// account's comment nor its reply to their own comment.
	got = w.comments(w.readerWork, w.blockerTok)
	if has(got, "ex two on reader work") || has(got, "ex replying to blocker") {
		t.Errorf("the blocker sees a blocked account's comments elsewhere: %v", got)
	}
	if !has(got, "blocker on reader work") {
		t.Errorf("the blocker should see their own comment: %v", got)
	}
	// The blocked person still sees the blocker's comment (AO3 doesn't hide it).
	got = w.comments(w.readerWork, w.blockedTok)
	if !has(got, "blocker on reader work") || !has(got, "ex two on reader work") {
		t.Errorf("on a third work, the blocked person should see the thread: %v", got)
	}
	// ... but can't reply to it (AO3 comment.rb:40-46).
	code, out := e.do("POST", "/api/v1/work/"+w.readerWork.String()+"/comments", w.blockedTok,
		map[string]any{"content": "reply", "parent_comment_id": w.bOnReader})
	if code != 403 || !strings.Contains(out, "replies_disabled") {
		t.Errorf("reply to the blocker's comment: %d %s", code, out)
	}
	w.noLeak("reply refusal", out)
	// Everyone else may reply to the blocker, and the blocker's reply notices...
	e.expect(201, "POST", "/api/v1/work/"+w.readerWork.String()+"/comments", w.readerTok,
		map[string]any{"content": "reader replies", "parent_comment_id": w.bOnReader})
	// ... and the blocker can't reply to a comment they can't see.
	e.expect(404, "POST", "/api/v1/work/"+w.readerWork.String()+"/comments", w.blockerTok,
		map[string]any{"content": "reply", "parent_comment_id": w.xOnReader})
}

// A block is account-wide and works both ways: the blocker is closed out of
// the blocked person's works the same way, whichever pseud wrote them.
func TestBlocksApplyToEveryPseudInBothDirections(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	w.blockEx()

	// A work credited to the blocked account's second pseud, which was never
	// picked, is closed to the blocker, who can still read it.
	secret := e.work(w.blocked, "Ex Two Story", "published")
	e.credit(secret, w.exTwo)
	page := e.expect(200, "GET", "/api/v1/work/"+secret.String(), w.blockerTok, nil)
	if !strings.Contains(page, `"comments_disabled_for_you":true`) {
		t.Errorf("the blocker should see comments disabled on the blocked account's work: %s", page)
	}
	if code, out := e.do("POST", "/api/v1/work/"+secret.String()+"/comments", w.blockerTok, map[string]any{"content": "hi"}); code != 403 || !strings.Contains(out, "comments_disabled") {
		t.Errorf("the blocker commenting on the blocked account's work: %d %s", code, out)
	}
	if code, _ := e.do("POST", "/api/v1/works/"+secret.String()+"/kudos", w.blockerTok, nil); code != 403 {
		t.Errorf("the blocker giving kudos to the blocked account's work: %d", code)
	}
	// And the blocked account's own work under its picked pseud, the same.
	if code, _ := e.do("POST", "/api/v1/work/"+w.blockedWork.String()+"/comments", w.blockerTok, map[string]any{"content": "hi"}); code != 403 {
		t.Errorf("the blocker commenting on the blocked account's other work: %d", code)
	}
	// Other pseuds of the blocker are blocked too.
	e.credit(w.work, w.beeTwo)
	if code, _ := e.do("POST", "/api/v1/work/"+w.work.String()+"/comments", w.blockedTok, map[string]any{"content": "hi", "pseudonym_id": w.ex}); code != 403 {
		t.Errorf("blocked commenting on the blocker's work under another pseud: %d", code)
	}
	// A creator is never blocked from their own work.
	e.expect(201, "POST", "/api/v1/work/"+w.work.String()+"/comments", w.blockerTok, map[string]any{"content": "my own work"})
	// A co-creator: the work is closed to the blocked person even if the
	// blocker is only one of two creators; the other creator can still comment.
	co := e.work(w.reader, "Shared Work", "published")
	e.exec(`INSERT INTO creatorships (creation_id, creation_type, pseud_id) VALUES ($1, 'Work', $2)`, co, w.bee)
	if code, _ := e.do("POST", "/api/v1/work/"+co.String()+"/comments", w.blockedTok, map[string]any{"content": "hi"}); code != 403 {
		t.Errorf("blocked commenting on a co-created work: %d", code)
	}
	e.expect(201, "POST", "/api/v1/work/"+co.String()+"/comments", w.readerTok, map[string]any{"content": "co-creator comment"})
	// Moderators are not subject to blocks.
	_, modTok := e.user("mod", "moderator")
	e.exec(`INSERT INTO user_blocks (blocker_id, blocked_id, pseud_id, pseud_name)
		SELECT $1, u.id, p.id, p.name FROM users u JOIN pseuds p ON p.user_id = u.id AND p.is_default WHERE u.username = 'mod'`, w.blocker)
	if mod := e.expect(200, "GET", "/api/v1/work/"+w.work.String(), modTok, nil); strings.Contains(mod, "comments_disabled_for_you") {
		t.Errorf("a moderator is flagged: %s", mod)
	}

	// Guests are not affected by a block.
	e.expect(201, "POST", "/api/v1/work/"+w.work.String()+"/comments", "", map[string]any{"content": "a guest", "guest_name": "Visitor"})
}

// AO3's other interaction rules: gifts are refused, bookmarks and
// subscriptions are not (AO3 has no block rule for them).
func TestBlocksAndGiftsBookmarksSubscriptions(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	w.blockEx()

	// work.rb:194-216: no gift to someone who has a block with a creator. The
	// giver is not told: the gift is answered like any other and kept aside
	// (shadowed), so pseuds can't be probed to find who blocked you. See
	// TestGiftToABlockerIsShadowed.
	for _, p := range []uuid.UUID{w.ex, w.exTwo} {
		code, out := e.do("POST", "/api/v1/works/"+w.work.String()+"/gift", w.blockerTok, map[string]any{"pseud_id": p})
		if code != 201 || strings.Contains(out, "gift_blocked") {
			t.Errorf("gift from the blocker to pseud %s: %d %s", p, code, out)
		}
	}
	if code, out := e.do("POST", "/api/v1/works/"+w.blockedWork.String()+"/gift", w.blockedTok, map[string]any{"pseud_id": w.beeTwo}); code != 201 {
		t.Errorf("gift from the blocked person to the blocker: %d %s", code, out)
	}
	if n := e.count(`SELECT count(*) FROM gifts WHERE shadowed`); n != 3 {
		t.Errorf("shadowed gifts = %d, want 3", n)
	}
	e.expect(201, "POST", "/api/v1/works/"+w.work.String()+"/gift", w.blockerTok, map[string]any{"pseud_id": w.rae})

	// A bookmark of the blocker's work, and a subscription to the blocker's
	// pseud, both still work for the blocked person.
	e.expect(201, "POST", "/api/v1/works/"+w.work.String()+"/bookmark", w.blockedTok, map[string]any{"notes": "good"})
	e.expect(201, "POST", "/api/v1/subscriptions", w.blockedTok, map[string]any{"type": "author", "target_id": w.bee.String()})
	if n := e.count(`SELECT count(*) FROM bookmarks WHERE user_id = $1`, w.blocked); n != 1 {
		t.Errorf("bookmarks = %d, want 1", n)
	}
}

// Nobody is notified about what an account they blocked does, under any pseud.
func TestNoNotificationsFromABlockedAccount(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	e.exec(`DELETE FROM notifications`)

	// Before the block, the reader's comment tells the blocker.
	w.commentAs(w.work, w.reader, w.rae, "from the reader", nil)
	if n := e.count(`SELECT count(*) FROM notifications WHERE user_id = $1`, w.blocker); n != 1 {
		t.Fatalf("control: the blocker has %d notifications, want 1", n)
	}

	w.blockEx()
	e.exec(`DELETE FROM notifications`)
	// The blocked account comments under its other pseud (written straight to
	// the table, as the API would refuse it), then replies to the blocker's
	// comment on a third work.
	w.commentAs(w.work, w.blocked, w.exTwo, "from ex two", nil)
	w.commentAs(w.readerWork, w.blocked, w.exTwo, "reply to blocker", &w.bOnReader)
	if n := e.count(`SELECT count(*) FROM notifications WHERE user_id = $1`, w.blocker); n != 0 {
		t.Errorf("the blocker was notified of a blocked account's comments (%d)", n)
	}
	// Others are still notified.
	if n := e.count(`SELECT count(*) FROM notifications WHERE user_id = $1`, w.reader); n == 0 {
		t.Errorf("the work's creator should still be notified")
	}
}

// A mute hides one pseud's works and comments from the muter, nothing else.
func TestMutesArePerPseud(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e

	// The blocker has two pseuds, each with a work; the reader mutes one.
	second := e.work(w.blocker, "Second Pseud Work", "published")
	e.credit(second, w.beeTwo)
	anon := e.work(w.blocker, "Hidden Pseud Work", "published")
	e.exec(`UPDATE works SET is_anonymous = true WHERE id = $1`, anon)

	body := e.expect(201, "POST", "/api/v1/mutes", w.readerTok, map[string]any{"pseud_id": w.bee})
	w.noLeak("create mute", body, "Bee Two")

	list := func(tok string) string {
		out := e.expect(200, "GET", "/api/v1/works?limit=100", tok, nil)
		w.noLeak("work list", out)
		return out
	}
	muted := list(w.readerTok)
	if strings.Contains(muted, w.work.String()) {
		t.Errorf("a muted pseud's work is still listed for the muter")
	}
	if !strings.Contains(list(w.readerTok), second.String()) {
		t.Errorf("the account's other pseud was hidden by the mute")
	}
	if !strings.Contains(list(w.readerTok), anon.String()) {
		t.Errorf("an anonymous work must not be hidden by a mute: that would say who wrote it")
	}
	// Nobody else is affected.
	other := list(w.blockedTok)
	if !strings.Contains(other, w.work.String()) || !strings.Contains(other, second.String()) {
		t.Errorf("a mute hid works from someone who didn't mute: %s", other)
	}
	// The muter's page for the muted pseud, and the work page, are gone for
	// them; the other pseud's work page is open.
	e.expect(404, "GET", "/api/v1/work/"+w.work.String(), w.readerTok, nil)
	e.expect(200, "GET", "/api/v1/work/"+second.String(), w.readerTok, nil)

	// Comments: the muted pseud's are hidden from the muter, the other's are not.
	got := w.comments(w.work, w.blockedTok)
	if !has(got, "blocker says thanks") {
		t.Errorf("control: %v", got)
	}
	third := e.work(w.blocked, "Third Work", "published")
	w.commentAs(third, w.blocker, w.bee, "from bee", nil)
	w.commentAs(third, w.blocker, w.beeTwo, "from bee two on third", nil)
	got = w.comments(third, w.readerTok)
	if has(got, "from bee") || !has(got, "from bee two on third") {
		t.Errorf("a mute should hide only the muted pseud's comments: %v", got)
	}
	if got = w.comments(third, w.blockedTok); !has(got, "from bee") {
		t.Errorf("a mute hid a comment from someone who didn't mute: %v", got)
	}

	// The list shows what was picked, as {id, name}. Muting is idempotent.
	e.expect(201, "POST", "/api/v1/mutes", w.readerTok, map[string]any{"pseud_id": w.bee})
	mutes := e.expect(200, "GET", "/api/v1/mutes", w.readerTok, nil)
	w.noLeak("mute list", mutes, "Bee Two")
	var got2 struct {
		Mutes []struct {
			Pseud pseudRef `json:"pseud"`
		}
	}
	json.Unmarshal([]byte(mutes), &got2)
	if len(got2.Mutes) != 1 || got2.Mutes[0].Pseud.ID != w.bee.String() || got2.Mutes[0].Pseud.Name != "Bee" {
		t.Errorf("mute list: %s", mutes)
	}
	e.expect(400, "POST", "/api/v1/mutes", w.readerTok, map[string]any{"pseud_id": w.rae}) // yourself
	e.expect(404, "POST", "/api/v1/mutes", w.readerTok, map[string]any{"pseud_id": uuid.New()})
	// Unmuting another pseud of the account doesn't work, so it can't probe.
	e.expect(404, "DELETE", "/api/v1/mutes/"+w.beeTwo.String(), w.readerTok, nil)
	e.expect(200, "DELETE", "/api/v1/mutes/"+w.bee.String(), w.readerTok, nil)
	e.expect(200, "GET", "/api/v1/work/"+w.work.String(), w.readerTok, nil)

	// A mute is not a block: the muted person can still comment on the muter's work.
	e.expect(201, "POST", "/api/v1/mutes", w.readerTok, map[string]any{"pseud_id": w.bee})
	e.expect(201, "POST", "/api/v1/work/"+w.readerWork.String()+"/comments", w.blockerTok, map[string]any{"content": "muted but allowed"})
}

// Collections name their organiser as a pseud {id, name, role}, never an
// account id or username.
func TestCollectionsNameOrganisersAsPseuds(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e

	// Default pseud when none is chosen.
	made := e.expect(201, "POST", "/api/v1/collections", w.blockerTok,
		map[string]any{"name": "lighthouse-fest", "title": "Lighthouse Fest", "is_open": true})
	w.noLeak("created collection", made)
	if !strings.Contains(made, `"organisers":[{"id":"`+w.bee.String()+`","name":"Bee","role":"owner"}]`) {
		t.Errorf("created collection should name the default pseud: %s", made)
	}
	// A chosen pseud; someone else's pseud is refused.
	chosen := e.expect(201, "POST", "/api/v1/collections", w.blockerTok,
		map[string]any{"name": "tower-fest", "title": "Tower Fest", "pseud_id": w.beeTwo})
	w.noLeak("created collection", chosen, "\"Bee\"")
	if !strings.Contains(chosen, w.beeTwo.String()) || !strings.Contains(chosen, "Bee Two") {
		t.Errorf("created collection should name the chosen pseud: %s", chosen)
	}
	e.expect(403, "POST", "/api/v1/collections", w.blockerTok,
		map[string]any{"name": "stolen", "title": "Stolen", "pseud_id": w.rae})

	var one struct {
		Collection struct{ ID string } `json:"collection"`
	}
	json.Unmarshal([]byte(made), &one)

	for _, c := range []struct{ what, path, tok string }{
		{"collection search", "/api/v1/collections", ""},
		{"collection search (logged in)", "/api/v1/collections", w.readerTok},
		{"collection page", "/api/v1/collections/" + one.Collection.ID, ""},
		{"my collections", "/api/v1/my/collections", w.blockerTok},
	} {
		out := e.expect(200, "GET", c.path, c.tok, nil)
		w.noLeak(c.what, out)
		if !strings.Contains(out, `"organisers"`) || strings.Contains(out, "maintainer") {
			t.Errorf("%s should name organisers: %s", c.what, out)
		}
	}

	// Changing the pseud, and deleting a pseud moves the collection to the default.
	updated := e.expect(200, "PUT", "/api/v1/collections/"+one.Collection.ID, w.blockerTok, map[string]any{"pseud_id": w.beeTwo})
	w.noLeak("updated collection", updated)
	if !strings.Contains(updated, "Bee Two") {
		t.Errorf("update should change the organiser: %s", updated)
	}
	e.expect(403, "PUT", "/api/v1/collections/"+one.Collection.ID, w.blockerTok, map[string]any{"pseud_id": w.rae})
	e.expect(200, "DELETE", "/api/v1/pseuds/"+w.beeTwo.String(), w.blockerTok, nil)
	after := e.expect(200, "GET", "/api/v1/collections/"+one.Collection.ID, "", nil)
	if !strings.Contains(after, `"name":"Bee"`) {
		t.Errorf("after the pseud was deleted the organiser should be the default: %s", after)
	}
}

// AO3 has no @mentions. A comment keeps its text and notifies nobody for a
// name written in it.
func TestAtNamesInCommentsNotifyNobody(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	e.exec(`DELETE FROM notifications`)

	text := "thanks @blocker_acct and @Bee, and @reader_acct"
	out := e.expect(201, "POST", "/api/v1/work/"+w.readerWork.String()+"/comments", w.blockedTok, map[string]any{"content": text})
	if !strings.Contains(out, text) {
		t.Errorf("the comment text changed: %s", out)
	}
	if n := e.count(`SELECT count(*) FROM notifications WHERE user_id = $1`, w.blocker); n != 0 {
		t.Errorf("an @name notified the person named (%d)", n)
	}
	if n := e.count(`SELECT count(*) FROM notifications WHERE type LIKE '%mention%'`); n != 0 {
		t.Errorf("mention notifications were created: %d", n)
	}
	if n := e.count(`SELECT count(*) FROM pg_proc WHERE proname IN ('create_mentions', 'extract_mentions')`); n != 0 {
		t.Errorf("mention functions still exist")
	}
	if n := e.count(`SELECT count(*) FROM information_schema.tables WHERE table_name = 'mentions'`); n != 0 {
		t.Errorf("the mentions table still exists")
	}
}
