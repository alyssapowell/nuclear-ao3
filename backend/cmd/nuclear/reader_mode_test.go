package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"

	"nuclear-ao3/internal/platform"
)

// The guard for "apps get the reader's view".
//
// An app acts for a person, but it must see exactly what any other logged-in
// reader sees, never what that person sees as a creator. Otherwise the
// difference (a 200 instead of a 404, an extra work in a list, a bigger count)
// tells the app which anonymous, unrevealed, draft or moderator-hidden works
// the person wrote.
//
// This test builds a person (alice) who owns exactly that kind of content,
// plus a second ordinary reader (carol) who owns nothing. For every GET route
// in the app allowlist it asks as alice's app and as carol, and the answers
// must be identical once volatile fields (times, hit counts) are removed. It
// then asks as alice at her own browser and checks the site still shows her
// her own hidden content, since creator views must keep working for the site
// itself.
//
// A GET route added to the allowlist must be either compared here or named in
// readerSkips with a reason, or the test fails: nobody can open a route to
// apps and forget this check.

// readerSkips are allowlisted GET routes that are not compared, with why.
var readerSkips = map[string]string{
	"GET /api/v1/subscriptions":       "the person's own follows; names no creator content and has no reader equivalent",
	"GET /api/v1/subscription-status": "the person's own follow state for an id they name",
	"GET /api/v1/user/tags/followed":  "the person's own follows",
}

type readerFx struct {
	e *appEnv

	carol    uuid.UUID
	carolTok string

	// Alice's content that hides or is hidden.
	anon, anonColl, unrevealed, draft, hidden uuid.UUID
	plain                                     uuid.UUID // alice's ordinary public work
	daveAnon                                  uuid.UUID // bob's anonymous work; bob has silently blocked alice
	allWorks                                  []uuid.UUID

	onlySecret, mixed, emptySeries, hiddenSeries uuid.UUID
	collection                                   uuid.UUID
	fandomTag, charTag, relTag                   uuid.UUID
}

func (f *readerFx) exec(q string, args ...any) { f.e.mustExec(q, args...) }

func newReaderFx(t *testing.T) *readerFx {
	e := newAppEnv(t, 1_000_000)
	f := &readerFx{e: e}
	f.carol, _ = e.user("carol_acct")
	f.carolTok = "carol-login"
	e.logins[f.carolTok] = f.carol

	zephyr := func(title string, pseud uuid.UUID, draft bool) uuid.UUID {
		return e.work(e.alice, pseud, "Zephyr "+title, false, draft)
	}
	f.anon = zephyr("anonymous", e.aliceAlt, false)
	f.anonColl = zephyr("in an anonymous collection", e.alicePseud, false)
	f.unrevealed = zephyr("in an unrevealed collection", e.aliceAlt, false)
	f.draft = zephyr("draft", e.aliceAlt, true)
	f.hidden = zephyr("hidden by a moderator", e.alicePseud, false)
	f.plain = zephyr("plain", e.aliceAlt, false)
	f.exec(`UPDATE works SET is_anonymous = true WHERE id = $1`, f.anon)
	f.exec(`UPDATE works SET in_anon_collection = true WHERE id = $1`, f.anonColl)
	f.exec(`UPDATE works SET in_unrevealed_collection = true WHERE id = $1`, f.unrevealed)
	f.exec(`UPDATE works SET hidden_by_admin = true WHERE id = $1`, f.hidden)

	// Dave wrote an anonymous work and blocked alice (he has nothing else, so
	// the block changes nothing else alice sees). Alice's comment and kudos
	// on it are "shadowed": she sees them, nobody else does.
	dave, davePseud := e.user("dave_acct")
	f.daveAnon = e.work(dave, davePseud, "Zephyr dave anonymous", false, false)
	f.exec(`UPDATE works SET is_anonymous = true WHERE id = $1`, f.daveAnon)
	f.exec(`INSERT INTO user_blocks (blocker_id, blocked_id, pseud_id, pseud_name)
		SELECT $1, $2, p.id, p.name FROM pseuds p WHERE p.user_id = $2 AND p.is_default`, dave, e.alice)
	f.exec(`INSERT INTO comments (work_id, user_id, pseudonym_id, content, shadowed) VALUES ($1, $2, $3, 'alice shadowed', true)`,
		f.daveAnon, e.alice, e.aliceAlt)
	f.exec(`INSERT INTO kudos (work_id, user_id, shadowed) VALUES ($1, $2, true)`, f.daveAnon, e.alice)
	f.exec(`INSERT INTO comments (work_id, user_id, pseudonym_id, content) VALUES ($1, $2, (SELECT id FROM pseuds WHERE user_id = $2 AND is_default), 'carol public')`,
		f.daveAnon, f.carol)
	f.exec(`INSERT INTO kudos (work_id, user_id) VALUES ($1, $2)`, f.daveAnon, f.carol)

	f.allWorks = []uuid.UUID{f.anon, f.anonColl, f.unrevealed, f.draft, f.hidden, f.plain, f.daveAnon, e.publicWork, e.restrictedWork, e.draftWork}

	// Chapters: one posted chapter each; a draft chapter and a moderator-hidden
	// chapter on works readers can otherwise read.
	for _, w := range f.allWorks {
		f.exec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, published_at, summary, notes, end_notes)
			SELECT $1, $2, 1, 'one', 'chapter text', w.is_draft, CASE WHEN w.is_draft THEN NULL ELSE now() END, '', '', ''
			FROM works w WHERE w.id = $2`, uuid.New(), w)
	}
	f.exec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, summary, notes, end_notes)
		VALUES ($1, $2, 2, 'draft chapter', 'secret', true, '', '', '')`, uuid.New(), f.anon)
	f.exec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, published_at, hidden_by_admin, summary, notes, end_notes)
		VALUES ($1, $2, 2, 'hidden chapter', 'hidden', false, now(), true, '', '', '')`, uuid.New(), f.plain)

	// Comments that only a creator or the writer sees: an unreviewed comment
	// by carol on alice's anonymous work, and an unreviewed one by alice on a
	// stranger's work. Plus ordinary ones everyone sees.
	f.exec(`INSERT INTO comments (work_id, user_id, pseudonym_id, content, unreviewed)
		VALUES ($1, $2, (SELECT id FROM pseuds WHERE user_id = $2 AND is_default), 'carol waiting for review', true)`, f.anon, f.carol)
	f.exec(`INSERT INTO comments (work_id, user_id, pseudonym_id, content)
		VALUES ($1, $2, (SELECT id FROM pseuds WHERE user_id = $2 AND is_default), 'carol approved')`, f.anon, f.carol)
	f.exec(`INSERT INTO comments (work_id, user_id, pseudonym_id, content, unreviewed) VALUES ($1, $2, $3, 'alice waiting for review', true)`,
		e.publicWork, e.alice, e.aliceAlt)
	f.exec(`INSERT INTO comments (work_id, user_id, pseudonym_id, content) VALUES ($1, $2, $3, 'alice reply visible')`,
		e.publicWork, e.alice, e.aliceAlt)

	// Series: only hidden things; hidden things plus an ordinary work; empty;
	// and one a moderator hid.
	f.onlySecret, f.mixed, f.emptySeries, f.hiddenSeries = uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for id, title := range map[uuid.UUID]string{f.onlySecret: "Zephyr only secret", f.mixed: "Zephyr mixed", f.emptySeries: "Zephyr empty", f.hiddenSeries: "Zephyr series hidden"} {
		f.exec(`INSERT INTO series (id, title, user_id) VALUES ($1, $2, $3)`, id, title, e.alice)
	}
	f.exec(`UPDATE series SET hidden_by_admin = true WHERE id = $1`, f.hiddenSeries)
	for i, w := range []uuid.UUID{f.anon, f.anonColl, f.unrevealed, f.draft, f.hidden} {
		f.exec(`INSERT INTO series_works (series_id, work_id, position) VALUES ($1, $2, $3)`, f.onlySecret, w, i+1)
	}
	f.exec(`INSERT INTO series_works (series_id, work_id, position) VALUES ($1, $2, 1), ($1, $3, 2)`, f.mixed, f.plain, f.anon)
	f.exec(`INSERT INTO series_works (series_id, work_id, position) VALUES ($1, $2, 1)`, f.hiddenSeries, f.plain)

	// Bookmarks by alice's pseud: a private one, one of her own anonymous
	// work, one of her ordinary work, and one of a stranger's work.
	for _, b := range []struct {
		work    uuid.UUID
		private bool
	}{{e.publicWork, true}, {f.anon, false}, {f.plain, false}, {e.restrictedWork, false}, {f.hidden, false}} {
		f.exec(`INSERT INTO bookmarks (id, work_id, user_id, pseud_id, is_private) VALUES ($1, $2, $3, $4, $5)`,
			uuid.New(), b.work, e.alice, e.aliceAlt, b.private)
	}

	// A collection (bob's) holding alice's works, and tags on everything.
	f.collection = uuid.New()
	f.exec(`INSERT INTO collections (id, name, title, user_id) VALUES ($1, 'zephyr-coll', 'Zephyr collection', $2)`, f.collection, e.bob)
	for _, w := range []uuid.UUID{f.anon, f.draft, f.hidden, f.plain, f.unrevealed} {
		f.exec(`INSERT INTO collection_items (collection_id, work_id, is_approved) VALUES ($1, $2, true)`, f.collection, w)
	}
	for _, tg := range []struct {
		dst  *uuid.UUID
		name string
		kind string
	}{{&f.fandomTag, "Zephyr Fandom", "fandom"}, {&f.charTag, "Zephyr Character", "character"}, {&f.relTag, "Zephyr/Pairing", "relationship"}} {
		*tg.dst = uuid.New()
		f.exec(`INSERT INTO tags (id, name, type) VALUES ($1, $2, $3)`, *tg.dst, tg.name, tg.kind)
		for _, w := range []uuid.UUID{f.anon, f.anonColl, f.unrevealed, f.draft, f.hidden, f.plain, f.daveAnon} {
			f.exec(`INSERT INTO work_tags (work_id, tag_id) VALUES ($1, $2)`, w, *tg.dst)
		}
	}
	return f
}

// urls fills a route pattern with every fixture id it can be called with.
func (f *readerFx) urls(pattern string) []string {
	path := strings.TrimPrefix(pattern, "GET ")
	values := map[string][]string{
		"work_id":         nil,
		"chapter_id":      {"1", "2"},
		"series_id":       {f.onlySecret.String(), f.mixed.String(), f.emptySeries.String(), f.hiddenSeries.String()},
		"collection_id":   {f.collection.String()},
		"pseud_id":        {f.e.aliceAlt.String(), f.e.alicePseud.String(), f.e.bobPseud.String()},
		"tag_id":          {f.fandomTag.String(), f.charTag.String()},
		"fandom_id":       {f.fandomTag.String()},
		"character_id":    {f.charTag.String()},
		"relationship_id": {f.relTag.String()},
	}
	for _, w := range f.allWorks {
		values["work_id"] = append(values["work_id"], w.String())
	}
	out := []string{""}
	for _, part := range strings.Split(path, "/") {
		var next []string
		if strings.HasPrefix(part, ":") {
			vs, ok := values[part[1:]]
			if !ok {
				return nil
			}
			for _, prefix := range out {
				for _, v := range vs {
					next = append(next, prefix+"/"+v)
				}
			}
		} else if part == "" {
			next = out
		} else {
			for _, prefix := range out {
				next = append(next, prefix+"/"+part)
			}
		}
		out = next
	}
	// Routes that list things are asked with a search that matches the fixture.
	query := map[string][]string{
		"/api/v1/works":              {"?limit=100"},
		"/api/v1/series":             {"?limit=100", "?q=Zephyr&limit=100"},
		"/api/v1/collections":        {"?limit=100"},
		"/api/v1/search/works":       {"?q=Zephyr", "?q=Zephyr&sort=kudos"},
		"/api/v1/search/suggestions": {"?q=Zeph"},
		"/api/v1/tags":               {"?q=Zephyr"},
		"/api/v1/tags/search":        {"?q=Zephyr"},
		"/api/v1/tags/autocomplete":  {"?q=Zephyr"},
	}[path]
	if len(query) == 0 {
		return out
	}
	var withQuery []string
	for _, u := range out {
		for _, q := range query {
			withQuery = append(withQuery, u+q)
		}
	}
	return withQuery
}

// normal removes what legitimately differs between two identical requests:
// times and hit counters.
func normal(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			if strings.HasSuffix(k, "_at") || k == "hits" || k == "hit_count" {
				delete(x, k)
				continue
			}
			x[k] = normal(val)
		}
	case []any:
		for i := range x {
			x[i] = normal(x[i])
		}
	}
	return v
}

func normalBody(body string) string {
	var v any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return body
	}
	b, _ := json.Marshal(normal(v)) // map keys are written in sorted order
	return string(b)
}

func (f *readerFx) answer(url, token string) (int, string) {
	w := f.e.get(url, token)
	return w.Code, normalBody(w.Body.String())
}

func short(s string) string {
	if len(s) > 700 {
		return s[:700] + "..."
	}
	return s
}

func TestAppSeesWhatAReaderSees(t *testing.T) {
	f := newReaderFx(t)
	e := f.e
	appTok := e.token(e.alice, e.client, "read comments:write bookmarks:manage subscriptions:manage", "1 hour")

	var routes []string
	for route := range platform.AppRoutes() {
		if strings.HasPrefix(route, "GET ") {
			routes = append(routes, route)
		}
	}
	sort.Strings(routes)

	compared, differsForCreator := 0, 0
	for _, route := range routes {
		if why, ok := readerSkips[route]; ok {
			t.Logf("not compared: %s (%s)", route, why)
			continue
		}
		urls := f.urls(route)
		if len(urls) == 0 {
			t.Errorf("%s: the test has no fixture ids for this route's parameters; add them or add the route to readerSkips with a reason", route)
			continue
		}
		for _, u := range urls {
			appCode, appBody := f.answer(u, appTok)
			readerCode, readerBody := f.answer(u, f.carolTok)
			compared++
			if appCode != readerCode || appBody != readerBody {
				t.Errorf("%s answers alice's app differently from an ordinary reader\n  app    (%d): %s\n  reader (%d): %s",
					u, appCode, short(appBody), readerCode, short(readerBody))
			}
			ownCode, ownBody := f.answer(u, "alice-login")
			if ownCode != readerCode || ownBody != readerBody {
				differsForCreator++
			}
		}
	}
	t.Logf("compared %d requests over %d allowlisted GET routes; %d of them differ between alice at her own browser and a reader",
		compared, len(routes)-len(readerSkips), differsForCreator)
	// The fixture must exercise the creator exceptions; if it stops doing so
	// the comparison above proves nothing.
	if differsForCreator < 25 {
		t.Errorf("only %d requests differ for the creator herself; the fixture no longer exercises creator-only views", differsForCreator)
	}
}

// The site itself must keep showing a creator her own hidden things.
func TestCreatorStillSeesOwnContentAtOwnBrowser(t *testing.T) {
	f := newReaderFx(t)
	e := f.e
	appTok := e.token(e.alice, e.client, "read", "1 hour")
	const own = "alice-login"

	for name, id := range map[string]uuid.UUID{"draft": f.draft, "hidden": f.hidden, "anonymous": f.anon, "unrevealed": f.unrevealed} {
		u := "/api/v1/works/" + id.String()
		if w := e.get(u, own); w.Code != 200 {
			t.Errorf("alice cannot open her own %s work: %d %s", name, w.Code, w.Body.String())
		}
	}
	for name, id := range map[string]uuid.UUID{"draft": f.draft, "hidden": f.hidden} {
		u := "/api/v1/works/" + id.String()
		if w := e.get(u, appTok); w.Code != 404 {
			t.Errorf("an app opened alice's %s work: %d", name, w.Code)
		}
		if w := e.get(u, f.carolTok); w.Code != 404 {
			t.Errorf("a reader opened alice's %s work: %d", name, w.Code)
		}
	}
	if w := e.get("/api/v1/series/"+f.onlySecret.String(), own); w.Code != 200 {
		t.Errorf("alice cannot open her own series: %d %s", w.Code, w.Body.String())
	}
	if w := e.get("/api/v1/series/"+f.onlySecret.String(), appTok); w.Code != 404 {
		t.Errorf("an app opened a series made only of hidden works: %d", w.Code)
	}
	if w := e.get("/api/v1/series/"+f.hiddenSeries.String(), own); w.Code != 200 {
		t.Errorf("alice cannot open her own moderator-hidden series: %d", w.Code)
	}

	// Her pseud page at her own browser lists drafts and hidden-creator works.
	list := e.get("/api/v1/pseuds/"+e.aliceAlt.String()+"/works", own).Body.String()
	for _, title := range []string{"Zephyr anonymous", "Zephyr draft", "Zephyr in an unrevealed collection"} {
		if !strings.Contains(list, title) {
			t.Errorf("alice's own pseud page lacks %q: %s", title, short(list))
		}
		if strings.Contains(e.get("/api/v1/pseuds/"+e.aliceAlt.String()+"/works", appTok).Body.String(), title) {
			t.Errorf("an app's view of alice's pseud page lists %q", title)
		}
	}
	// But a member still sees a members-only work through the app.
	if w := e.get("/api/v1/works/"+e.restrictedWork.String(), appTok); w.Code != 200 {
		t.Errorf("an app for a member must read members-only works: %d", w.Code)
	}

	// Unreviewed comments on her own work, and her own shadowed ones.
	comments := e.get("/api/v1/works/"+f.anon.String()+"/comments", own).Body.String()
	if !strings.Contains(comments, "carol waiting for review") {
		t.Errorf("a creator must see comments waiting for review: %s", short(comments))
	}
	if body := e.get("/api/v1/works/"+f.anon.String()+"/comments", appTok).Body.String(); strings.Contains(body, "waiting for review") {
		t.Errorf("an app saw a comment waiting for review: %s", short(body))
	}
	shadow := e.get("/api/v1/works/"+f.daveAnon.String()+"/comments", own).Body.String()
	if !strings.Contains(shadow, "alice shadowed") {
		t.Errorf("a silently blocked person keeps seeing her own comment at her own browser: %s", short(shadow))
	}
	if body := e.get("/api/v1/works/"+f.daveAnon.String()+"/comments", appTok).Body.String(); strings.Contains(body, "alice shadowed") {
		t.Errorf("an app saw the person's shadowed comment, which shows the silent block: %s", short(body))
	}
}

// Writes an app may make are checked as a reader too: a draft or hidden work
// gives the same answer to the person's app as to any other reader.
func TestAppWritesAreCheckedAsAReader(t *testing.T) {
	f := newReaderFx(t)
	e := f.e
	appTok := e.token(e.alice, e.client, "read comments:write bookmarks:manage subscriptions:manage", "1 hour")
	// Carol needs a login token to compare with; she is not blocked anywhere.
	for _, tc := range []struct {
		name, path, body string
		work             uuid.UUID
	}{
		{"comment on a draft", "/comments", `{"content":"hi"}`, f.draft},
		{"comment on a hidden work", "/comments", `{"content":"hi"}`, f.hidden},
		{"bookmark a draft", "/bookmark", `{}`, f.draft},
		{"bookmark a hidden work", "/bookmark", `{}`, f.hidden},
	} {
		url := "/api/v1/works/" + tc.work.String() + tc.path
		a := doBody(e.r, "POST", url, appTok, tc.body)
		r := doBody(e.r, "POST", url, f.carolTok, tc.body)
		if a.Code != r.Code || normalBody(a.Body.String()) != normalBody(r.Body.String()) {
			t.Errorf("%s: app got %d %s, reader got %d %s", tc.name, a.Code, short(a.Body.String()), r.Code, short(r.Body.String()))
		}
	}

	// A comment on a work with moderated comments waits for review for an app,
	// as for any reader, even on a work the person co-created.
	e.mustExec(`UPDATE works SET moderate_comments = true WHERE id = $1`, f.plain)
	url := "/api/v1/works/" + f.plain.String() + "/comments"
	a := doBody(e.r, "POST", url, appTok, `{"content":"mine"}`)
	r := doBody(e.r, "POST", url, f.carolTok, `{"content":"theirs"}`)
	if a.Code != r.Code || strings.Contains(a.Body.String(), `"unreviewed":false`) != strings.Contains(r.Body.String(), `"unreviewed":false`) {
		t.Errorf("moderated comment: app %d %s, reader %d %s", a.Code, short(a.Body.String()), r.Code, short(r.Body.String()))
	}
	var unreviewed bool
	if err := e.db.QueryRow(`SELECT unreviewed FROM comments WHERE work_id = $1 AND content = 'mine'`, f.plain).Scan(&unreviewed); err != nil || !unreviewed {
		t.Errorf("an app's comment on a moderated work should wait for review like anyone's (unreviewed=%v err=%v)", unreviewed, err)
	}

	// Following a work or series names it only if the person may see it.
	for _, tc := range []struct{ typ, id string }{{"work", f.draft.String()}, {"work", f.hidden.String()}, {"series", f.hiddenSeries.String()}} {
		body := fmt.Sprintf(`{"type":%q,"target_id":%q}`, tc.typ, tc.id)
		a := doBody(e.r, "POST", "/api/v1/subscriptions", appTok, body)
		if a.Code != 201 || strings.Contains(a.Body.String(), "Zephyr") {
			t.Errorf("subscribing to hidden %s via an app: %d %s", tc.typ, a.Code, short(a.Body.String()))
		}
	}
}

// usernameKeys lists the paths of every JSON key in v that names a username.
func usernameKeys(v any, path string) []string {
	var out []string
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			if strings.Contains(strings.ToLower(k), "username") {
				out = append(out, path+"/"+k)
			}
			out = append(out, usernameKeys(val, path+"/"+k)...)
		}
	case []any:
		for i, val := range x {
			out = append(out, usernameKeys(val, fmt.Sprintf("%s/%d", path, i))...)
		}
	}
	return out
}

// Accounts have no username, so no answer may have a key for one, whoever
// asks: an app, another reader, or the person at their own browser. The
// allowlisted routes are walked with every fixture id, and then every other
// GET route that needs no ids, asked as the person herself, which is where
// her own account details are.
func TestNoAnswerHasAUsername(t *testing.T) {
	f := newReaderFx(t)
	e := f.e
	appTok := e.token(e.alice, e.client, "read comments:write bookmarks:manage subscriptions:manage", "1 hour")

	check := func(who, url, token string) {
		t.Helper()
		w := e.get(url, token)
		var v any
		if json.Unmarshal(w.Body.Bytes(), &v) != nil {
			return // not JSON (an error page or a file); no keys to read
		}
		if bad := usernameKeys(v, ""); len(bad) > 0 {
			t.Errorf("GET %s as %s has username keys %v: %s", url, who, bad, short(w.Body.String()))
		}
	}

	asked := map[string]bool{}
	for route := range platform.AppRoutes() {
		if !strings.HasPrefix(route, "GET ") {
			continue
		}
		for _, u := range f.urls(route) {
			asked[u] = true
			check("an app", u, appTok)
			check("a reader", u, f.carolTok)
			check("the person", u, "alice-login")
		}
	}
	own := 0
	for _, r := range e.r.Routes() {
		if r.Method != "GET" || strings.Contains(r.Path, ":") || strings.Contains(r.Path, "*") || asked[r.Path] {
			continue
		}
		own++
		check("the person", r.Path, "alice-login")
	}
	t.Logf("checked %d allowlisted requests and %d other routes", len(asked), own)
	if len(asked) < 50 || own < 10 {
		t.Errorf("only %d + %d requests walked; the walk no longer covers the API", len(asked), own)
	}
}
