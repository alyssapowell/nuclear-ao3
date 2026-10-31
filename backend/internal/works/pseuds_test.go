package works_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// pseudID returns the id of a user's pseud with this name.
func (e *fixEnv) pseudID(user uuid.UUID, name string) uuid.UUID {
	e.t.Helper()
	var id uuid.UUID
	if err := e.db.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1 AND lower(name) = lower($2)`, user, name).Scan(&id); err != nil {
		e.t.Fatalf("pseud %q: %v", name, err)
	}
	return id
}

func (e *fixEnv) count(q string, args ...any) (n int) {
	e.t.Helper()
	if err := e.db.QueryRow(q, args...).Scan(&n); err != nil {
		e.t.Fatalf("%s: %v", q, err)
	}
	return
}

// keysOf collects every object key anywhere in a JSON document.
func keysOf(t *testing.T, body string) map[string]bool {
	t.Helper()
	var doc any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, body)
	}
	keys := map[string]bool{}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, child := range x {
				keys[k] = true
				walk(child)
			}
		case []any:
			for _, child := range x {
				walk(child)
			}
		}
	}
	walk(doc)
	return keys
}

// neverNamesAccounts fails if a public answer carries an account id or name.
func neverNamesAccounts(t *testing.T, what, body string) {
	t.Helper()
	for k := range keysOf(t, body) {
		switch k {
		case "user_id", "username", "author_user_id", "work_author_id", "guest_email", "ip_address", "pseud_name", "pseud_id":
			t.Errorf("%s has a %q field: %s", what, k, body)
		}
	}
}

func TestPseudNamesFollowAO3(t *testing.T) {
	e := newFixEnv(t)
	_, tok := e.user("writer")
	_, otherTok := e.user("other")

	create := func(token, name string) (int, string) {
		return e.do("POST", "/api/v1/pseuds", token, map[string]any{"name": name})
	}
	for _, ok := range []string{"Pen Name", "Amélie Poulain", "李小龍", "Ünal-Çağlar", "ว่าย น้ำ", "under_score", "x", strings.Repeat("é", 40)} {
		if code, out := create(tok, ok); code != 201 {
			t.Errorf("name %q: status %d: %s", ok, code, out)
		}
	}
	for name, why := range map[string]string{
		"":                      "empty",
		"   ":                   "only spaces",
		strings.Repeat("a", 41): "41 characters",
		strings.Repeat("é", 41): "41 characters, multi-byte",
		"--":                    "no letter or number",
		"a@b":                   "punctuation",
		"emoji 😀":               "emoji",
		"line\nbreak":           "control character",
		"slash/name":            "punctuation",
	} {
		if code, out := create(tok, name); code != 422 {
			t.Errorf("name %q (%s): status %d, want 422: %s", name, why, code, out)
		}
	}
	// Names are unique per user, ignoring case; another user may reuse one.
	if code, _ := create(tok, "PEN NAME"); code != 409 {
		t.Errorf("a name differing only by case: status %d, want 409", code)
	}
	if code, _ := create(otherTok, "Pen Name"); code != 201 {
		t.Errorf("another user's use of the same name: status %d, want 201", code)
	}
	// The same rules apply to renaming, and a description has a limit.
	out := e.expect(201, "POST", "/api/v1/pseuds", tok, map[string]any{"name": "Renamed"})
	var made struct{ Pseud struct{ ID string } }
	json.Unmarshal([]byte(out), &made)
	path := "/api/v1/pseuds/" + made.Pseud.ID
	e.expect(422, "PUT", path, tok, map[string]any{"name": ""})
	e.expect(409, "PUT", path, tok, map[string]any{"name": "pen name"})
	e.expect(422, "PUT", path, tok, map[string]any{"description": strings.Repeat("d", 501)})
	e.expect(422, "PUT", path, tok, map[string]any{"icon_alt_text": strings.Repeat("a", 251)})
	e.expect(422, "PUT", path, tok, map[string]any{"icon_comment_text": strings.Repeat("c", 51)})
	e.expect(422, "PUT", path, tok, map[string]any{"icon_url": "javascript:alert(1)"})
	e.expect(200, "PUT", path, tok, map[string]any{"name": "Renamed Again", "description": strings.Repeat("d", 500),
		"icon_url": "https://example.org/i.png", "icon_alt_text": "an icon", "icon_comment_text": "hello"})
	// Someone else's pseud is not found, not changed.
	e.expect(404, "PUT", path, otherTok, map[string]any{"name": "Stolen"})
	e.expect(404, "DELETE", path, otherTok, nil)
}

func TestExactlyOneDefaultPseud(t *testing.T) {
	e := newFixEnv(t)
	writer, tok := e.user("writer")
	defaults := func() int { return e.count(`SELECT count(*) FROM pseuds WHERE user_id = $1 AND is_default`, writer) }

	if defaults() != 1 {
		t.Fatalf("a new account has %d default pseuds, want 1", defaults())
	}
	out := e.expect(201, "POST", "/api/v1/pseuds", tok, map[string]any{"name": "Second", "is_default": true})
	var made struct{ Pseud struct{ ID string } }
	json.Unmarshal([]byte(out), &made)
	if defaults() != 1 {
		t.Fatalf("after making a new default there are %d, want 1", defaults())
	}
	var name string
	e.db.QueryRow(`SELECT name FROM pseuds WHERE user_id = $1 AND is_default`, writer).Scan(&name)
	if name != "Second" {
		t.Errorf("the default is %q, want Second", name)
	}
	// Asking for the default to stop being one hands it back to the pseud
	// named after the username (AO3: check_default_pseud).
	e.expect(200, "PUT", "/api/v1/pseuds/"+made.Pseud.ID, tok, map[string]any{"is_default": false})
	e.db.QueryRow(`SELECT name FROM pseuds WHERE user_id = $1 AND is_default`, writer).Scan(&name)
	if name != "writer" || defaults() != 1 {
		t.Errorf("after un-defaulting, default is %q (%d defaults), want writer", name, defaults())
	}
	// Making another pseud the default through PUT demotes the old one.
	e.expect(200, "PUT", "/api/v1/pseuds/"+made.Pseud.ID, tok, map[string]any{"is_default": true})
	if defaults() != 1 {
		t.Errorf("%d defaults after PUT, want 1", defaults())
	}
}

func TestDeletingAPseudMovesEverythingToTheDefault(t *testing.T) {
	e := newFixEnv(t)
	writer, tok := e.user("writer")
	reader, readerTok := e.user("reader")
	def := e.pseudID(writer, "writer")

	out := e.expect(201, "POST", "/api/v1/pseuds", tok, map[string]any{"name": "Old Name"})
	var made struct{ Pseud struct{ ID string } }
	json.Unmarshal([]byte(out), &made)
	old := uuid.MustParse(made.Pseud.ID)

	// A work credited to the old pseud only, one credited to both, a comment
	// under the old pseud, and a gift to it.
	only := e.work(writer, "Only Old", "published")
	both := e.work(writer, "Both", "published")
	e.exec(`UPDATE creatorships SET pseud_id = $1 WHERE creation_id = $2`, old, only)
	e.exec(`INSERT INTO creatorships (creation_id, creation_type, pseud_id, approved) VALUES ($1, 'Work', $2, false)`, both, old)
	comment := uuid.New()
	e.exec(`INSERT INTO comments (id, work_id, user_id, pseudonym_id, content) VALUES ($1, $2, $3, $4, 'hello')`, comment, only, writer, old)
	e.exec(`INSERT INTO gifts (work_id, pseud_id, recipient_name) VALUES ($1, $2, 'x')`, both, old)

	// The default and the pseud named after the username can't be deleted.
	e.expect(409, "DELETE", "/api/v1/pseuds/"+def.String(), tok, nil)
	e.expect(200, "PUT", "/api/v1/pseuds/"+old.String(), tok, map[string]any{"is_default": true})
	e.expect(409, "DELETE", "/api/v1/pseuds/"+old.String(), tok, nil) // now the default
	e.expect(200, "PUT", "/api/v1/pseuds/"+def.String(), tok, map[string]any{"is_default": true})
	// Others can't delete it.
	e.expect(404, "DELETE", "/api/v1/pseuds/"+old.String(), readerTok, nil)
	_ = reader

	e.expect(200, "DELETE", "/api/v1/pseuds/"+old.String(), tok, nil)

	if n := e.count(`SELECT count(*) FROM pseuds WHERE id = $1`, old); n != 0 {
		t.Fatal("the pseud is still there")
	}
	if n := e.count(`SELECT count(*) FROM creatorships WHERE creation_id = $1 AND pseud_id = $2 AND approved`, only, def); n != 1 {
		t.Errorf("work credited only to the old pseud now has %d credits to the default, want 1", n)
	}
	// "both": the default already had an approved credit; the unapproved
	// credit of the old pseud merges into it instead of duplicating it.
	if n := e.count(`SELECT count(*) FROM creatorships WHERE creation_id = $1`, both); n != 1 {
		t.Errorf("merged work has %d credits, want 1", n)
	}
	var moved uuid.UUID
	e.db.QueryRow(`SELECT pseudonym_id FROM comments WHERE id = $1`, comment).Scan(&moved)
	if moved != def {
		t.Errorf("comment is under %s, want the default %s", moved, def)
	}
	var gift uuid.UUID
	e.db.QueryRow(`SELECT pseud_id FROM gifts WHERE work_id = $1`, both).Scan(&gift)
	if gift != def {
		t.Errorf("gift is for %s, want the default", gift)
	}
	// The works are still there and still listed under the default.
	body := e.expect(200, "GET", "/api/v1/works/"+only.String(), readerTok, nil)
	if !strings.Contains(body, `"name":"writer"`) {
		t.Errorf("the moved work doesn't show the default pseud: %s", body)
	}
}

func TestCommentingAsSomeoneElsesPseudIsForbidden(t *testing.T) {
	e := newFixEnv(t)
	writer, _ := e.user("writer")
	_, readerTok := e.user("reader")
	work := e.work(writer, "A Work", "published")
	path := "/api/v1/works/" + work.String() + "/comments"

	theirs := e.pseudID(writer, "writer")
	e.expect(403, "POST", path, readerTok, map[string]any{"content": "hi", "pseudonym_id": theirs})
	if n := e.count(`SELECT count(*) FROM comments WHERE work_id = $1`, work); n != 0 {
		t.Fatalf("the refused comment was saved")
	}
	// Under their own pseud, by id or by default, it works.
	reader := e.pseudIDByToken(readerTok)
	e.expect(201, "POST", path, readerTok, map[string]any{"content": "hi", "pseudonym_id": reader})
	e.expect(201, "POST", path, readerTok, map[string]any{"content": "again"})
	// A pseud that doesn't exist is no one's.
	e.expect(403, "POST", path, readerTok, map[string]any{"content": "hi", "pseudonym_id": uuid.New()})
}

func (e *fixEnv) pseudIDByToken(tok string) uuid.UUID {
	e.t.Helper()
	return e.pseudID(e.tokens[tok], strings.TrimPrefix(tok, "token-"))
}

func TestPublicAnswersNameAuthorsByPseudOnly(t *testing.T) {
	e := newFixEnv(t)
	writer, writerTok := e.user("writer")
	_, readerTok := e.user("reader")
	// A pen name that is not the username, so a leak of either is visible.
	moon := uuid.New()
	e.exec(`INSERT INTO pseuds (id, user_id, name) VALUES ($1, $2, 'Moonlit Quill')`, moon, writer)
	work := e.work(writer, "A Work", "published")
	e.exec(`UPDATE creatorships SET pseud_id = $1 WHERE creation_id = $2`, moon, work)
	e.exec(`INSERT INTO comments (id, work_id, user_id, pseudonym_id, content) VALUES ($1, $2, $3, $4, 'by the author')`,
		uuid.New(), work, writer, moon)
	readerPseud := e.pseudIDByToken(readerTok)
	e.exec(`INSERT INTO comments (id, work_id, user_id, pseudonym_id, content) VALUES ($1, $2, $3, $4, 'by a reader')`,
		uuid.New(), work, e.tokens[readerTok], readerPseud)
	e.exec(`INSERT INTO comments (work_id, guest_name, guest_email, content) VALUES ($1, 'Visitor', 'visitor@example.org', 'by a guest')`, work)

	for _, viewer := range []string{"", readerTok, writerTok} {
		page := e.expect(200, "GET", "/api/v1/works/"+work.String(), viewer, nil)
		neverNamesAccounts(t, "work page", page)
		var got struct {
			Authors []map[string]any `json:"authors"`
		}
		json.Unmarshal([]byte(page), &got)
		if len(got.Authors) != 1 || got.Authors[0]["id"] != moon.String() || got.Authors[0]["name"] != "Moonlit Quill" || len(got.Authors[0]) != 2 {
			t.Errorf("authors = %v, want exactly [{id, name}]", got.Authors)
		}
		comments := e.expect(200, "GET", "/api/v1/works/"+work.String()+"/comments", viewer, nil)
		neverNamesAccounts(t, "comment list", comments)
		if strings.Contains(comments, "visitor@example.org") {
			t.Error("a guest's email is in the comment list")
		}
		if !strings.Contains(comments, `"pseud":{"id":"`+moon.String()+`","name":"Moonlit Quill"}`) {
			t.Errorf("the author's comment isn't shown as {id, name}: %s", comments)
		}
		if strings.Contains(page+comments, `"writer"`) {
			t.Errorf("the username appears in an answer for viewer %q", viewer)
		}
	}

	// Creating a comment answers in the same form.
	out := e.expect(201, "POST", "/api/v1/works/"+work.String()+"/comments", readerTok, map[string]any{"content": "new"})
	neverNamesAccounts(t, "created comment", out)
	if !strings.Contains(out, `"name":"reader"`) {
		t.Errorf("created comment lacks its pseud: %s", out)
	}
	// So does the authors endpoint.
	authors := e.expect(200, "GET", "/api/v1/works/"+work.String()+"/authors", writerTok, nil)
	neverNamesAccounts(t, "authors", authors)
}

func TestAnonymousWorksShowNoPseud(t *testing.T) {
	e := newFixEnv(t)
	writer, writerTok := e.user("writer")
	_, readerTok := e.user("reader")
	moon := uuid.New()
	e.exec(`INSERT INTO pseuds (id, user_id, name) VALUES ($1, $2, 'Moonlit Quill')`, moon, writer)

	// Three ways to hide a creator.
	hidings := map[string]string{
		"anonymous":             `is_anonymous = true`,
		"anonymous collection":  `in_anon_collection = true`,
		"unrevealed collection": `in_unrevealed_collection = true`,
	}
	for label, set := range hidings {
		work := e.work(writer, "Secret "+label, "published")
		e.exec(`UPDATE creatorships SET pseud_id = $1 WHERE creation_id = $2`, moon, work)
		e.exec(`UPDATE works SET `+set+` WHERE id = $1`, work)
		// The creator's own reply and a reader's comment.
		e.exec(`INSERT INTO comments (work_id, user_id, pseudonym_id, content) VALUES ($1, $2, $3, 'creator reply')`, work, writer, moon)
		e.exec(`INSERT INTO comments (work_id, user_id, pseudonym_id, content) VALUES ($1, $2, $3, 'reader says')`,
			work, e.tokens[readerTok], e.pseudIDByToken(readerTok))

		for _, viewer := range []string{"", readerTok, writerTok} {
			page := e.expect(200, "GET", "/api/v1/works/"+work.String(), viewer, nil)
			comments := e.expect(200, "GET", "/api/v1/works/"+work.String()+"/comments", viewer, nil)
			authors := e.expect(200, "GET", "/api/v1/works/"+work.String()+"/authors", writerTok, nil)
			for what, body := range map[string]string{"work page": page, "comments": comments, "authors": authors} {
				neverNamesAccounts(t, label+" "+what, body)
				if strings.Contains(body, "Moonlit Quill") || strings.Contains(body, moon.String()) || strings.Contains(body, `"writer"`) {
					t.Errorf("%s: %s for viewer %q reveals the creator: %s", label, what, viewer, body)
				}
			}
			var got struct {
				Authors []any `json:"authors"`
			}
			json.Unmarshal([]byte(page), &got)
			if len(got.Authors) != 0 {
				t.Errorf("%s: authors = %v, want none", label, got.Authors)
			}
			if !strings.Contains(comments, `"anonymous_creator":true`) || !strings.Contains(comments, `"reader"`) {
				t.Errorf("%s: comments should mark the creator's reply and keep the reader's name: %s", label, comments)
			}
		}
	}
}

// The creator's page, series and profile totals never give away the creator
// of a work that hides them, except to the creator.
func TestCreatorsStayHiddenInListsAndTotals(t *testing.T) {
	e := newFixEnv(t)
	writer, writerTok := e.user("writer")
	_, readerTok := e.user("reader")

	open := e.work(writer, "Open", "published")
	secret := e.work(writer, "Secret", "published")
	e.exec(`UPDATE works SET word_count = 100`)

	totals := func() (works, words int) {
		e.db.QueryRow(`SELECT works_count, words_written FROM user_statistics WHERE user_id = $1`, writer).Scan(&works, &words)
		return
	}
	if w, _ := totals(); w != 2 {
		t.Fatalf("profile counts %d works, want 2", w)
	}
	for _, set := range []string{"is_anonymous", "in_anon_collection", "in_unrevealed_collection"} {
		e.exec(`UPDATE works SET is_anonymous = false, in_anon_collection = false, in_unrevealed_collection = false`)
		e.exec(`UPDATE works SET `+set+` = true WHERE id = $1`, secret)
		if w, words := totals(); w != 1 || words != 100 {
			t.Errorf("%s: profile totals = %d works, %d words; want 1 and 100", set, w, words)
		}
		// The creator's page.
		page := e.expect(200, "GET", e.pseudPage(writer, "works"), readerTok, nil)
		if strings.Contains(page, secret.String()) || !strings.Contains(page, open.String()) {
			t.Errorf("%s: creator page for a stranger: %s", set, page)
		}
		// The creator still sees both.
		own := e.expect(200, "GET", e.pseudPage(writer, "works"), writerTok, nil)
		if !strings.Contains(own, secret.String()) {
			t.Errorf("%s: the creator can't see their own work on their page", set)
		}
	}

	// Series: one with the secret work only, one with both.
	e.exec(`UPDATE works SET is_anonymous = true WHERE id = $1`, secret)
	onlySecret, mixed := uuid.New(), uuid.New()
	e.exec(`INSERT INTO series (id, user_id, title) VALUES ($1, $3, 'Only Secret'), ($2, $3, 'Mixed')`, onlySecret, mixed, writer)
	e.exec(`INSERT INTO series_works (series_id, work_id, position) VALUES ($1, $2, 1), ($3, $2, 1), ($3, $4, 2)`, onlySecret, secret, mixed, open)

	list := e.expect(200, "GET", e.pseudPage(writer, "series"), readerTok, nil)
	if strings.Contains(list, onlySecret.String()) || !strings.Contains(list, mixed.String()) {
		t.Errorf("series list for a stranger: %s", list)
	}
	var parsed struct {
		Series []struct {
			ID        string `json:"id"`
			WorkCount int    `json:"work_count"`
		}
	}
	json.Unmarshal([]byte(list), &parsed)
	for _, s := range parsed.Series {
		if s.ID == mixed.String() && s.WorkCount != 1 {
			t.Errorf("a stranger's count for the mixed series = %d, want 1", s.WorkCount)
		}
	}
	e.expect(404, "GET", "/api/v1/series/"+onlySecret.String(), readerTok, nil)
	e.expect(200, "GET", "/api/v1/series/"+onlySecret.String(), writerTok, nil)
	inSeries := e.expect(200, "GET", "/api/v1/series/"+mixed.String()+"/works", readerTok, nil)
	if strings.Contains(inSeries, secret.String()) || !strings.Contains(inSeries, open.String()) {
		t.Errorf("works in a series for a stranger: %s", inSeries)
	}
	ownSeries := e.expect(200, "GET", "/api/v1/series/"+mixed.String()+"/works", writerTok, nil)
	if !strings.Contains(ownSeries, secret.String()) {
		t.Errorf("the creator can't see their own work in their series")
	}
	search := e.expect(200, "GET", "/api/v1/series?q=Secret", readerTok, nil)
	if strings.Contains(search, onlySecret.String()) {
		t.Errorf("series search for a stranger: %s", search)
	}
}

// pseudPage is the path of a list on the default pseud of a user's account.
func (e *fixEnv) pseudPage(user uuid.UUID, list string) string {
	e.t.Helper()
	var id uuid.UUID
	if err := e.db.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1 AND is_default`, user).Scan(&id); err != nil {
		e.t.Fatal(err)
	}
	return "/api/v1/pseuds/" + id.String() + "/" + list
}
