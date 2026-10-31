package search

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nuclear-ao3/internal/testdb"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// archive is a small archive to search: two authors and a handful of works
// that each test one rule.
type archive struct {
	db              *sql.DB
	author, reader  uuid.UUID
	authorPseud     uuid.UUID
	public, draft   uuid.UUID
	restricted      uuid.UUID
	anonymous       uuid.UUID
	crossover, long uuid.UUID
}

func newArchive(t *testing.T) *archive {
	db := testdb.New(t)
	a := &archive{db: db, author: uuid.New(), reader: uuid.New(), authorPseud: uuid.New()}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	for name, id := range map[string]uuid.UUID{"writer": a.author, "reader": a.reader} {
		exec(`INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, '!')`, id, name, name+"@example.org")
	}
	// The account came with a default pseud; give it the pen name and the id the tests use.
	exec(`UPDATE pseuds SET id = $1, name = 'Quill' WHERE user_id = $2 AND is_default`, a.authorPseud, a.author)

	work := func(title, summary, rating string, fandoms, tags []string, set string) uuid.UUID {
		t.Helper()
		id := uuid.New()
		exec(`INSERT INTO works (id, user_id, title, summary, rating, fandoms, freeform_tags,
				status, is_draft, published_at, word_count)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'published', false, now(), 1000)`,
			id, a.author, title, summary, rating, pq.Array(fandoms), pq.Array(tags))
		exec(`INSERT INTO creatorships (creation_id, creation_type, pseud_id) VALUES ($1, 'Work', $2)`, id, a.authorPseud)
		if set != "" {
			exec(`UPDATE works SET `+set+` WHERE id = $1`, id)
		}
		return id
	}
	a.public = work("Lighthouse Keeper", "A keeper tends the lamp through a storm.", "general",
		[]string{"Original Work"}, []string{"Fluff"}, "")
	a.draft = work("Lighthouse Draft", "Not posted yet.", "general",
		[]string{"Original Work"}, nil, "status = 'draft', is_draft = true")
	a.restricted = work("Lighthouse for Members", "Only for logged-in readers.", "teen",
		[]string{"Original Work"}, nil, "restricted_to_users = true")
	a.anonymous = work("Lighthouse Anonymous", "Posted anonymously.", "mature",
		[]string{"Original Work"}, nil, "is_anonymous = true")
	a.crossover = work("Two Worlds", "A crossover.", "teen",
		[]string{"Original Work", "Sherlock Holmes"}, []string{"Angst"}, "")
	a.long = work("Very Long Tale", "Epic.", "explicit",
		[]string{"Sherlock Holmes"}, []string{"Angst"}, "word_count = 200000, kudos_count = 50, comment_count = 7")
	return a
}

func (a *archive) search(t *testing.T, q WorkQuery, viewer *uuid.UUID) map[uuid.UUID]WorkResult {
	t.Helper()
	if q.Limit == 0 {
		q.Limit = 50
	}
	if q.Page == 0 {
		q.Page = 1
	}
	res, err := New(a.db).Search(q, viewer)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != len(res.Works) {
		t.Fatalf("total %d but %d works returned", res.Total, len(res.Works))
	}
	found := map[uuid.UUID]WorkResult{}
	for _, w := range res.Works {
		found[w.ID] = w
	}
	return found
}

func TestWhoCanFindWhat(t *testing.T) {
	a := newArchive(t)
	q := WorkQuery{Text: "lighthouse"}

	loggedOut := a.search(t, q, nil)
	if _, ok := loggedOut[a.public]; !ok {
		t.Error("a posted work should be found")
	}
	if _, ok := loggedOut[a.draft]; ok {
		t.Error("a draft must never be found, even by its author")
	}
	if _, ok := loggedOut[a.restricted]; ok {
		t.Error("a work restricted to users must not be found when logged out")
	}
	loggedIn := a.search(t, q, &a.reader)
	if _, ok := loggedIn[a.restricted]; !ok {
		t.Error("a work restricted to users should be found when logged in")
	}

	if _, err := a.db.Exec(`INSERT INTO user_blocks (blocker_id, blocked_id, block_type) VALUES ($1, $2, 'full')`,
		a.author, a.reader); err != nil {
		t.Fatal(err)
	}
	if found := a.search(t, q, &a.reader); len(found) != 0 {
		t.Errorf("someone the author blocked should find none of their works, found %d", len(found))
	}
}

func TestAnonymousWorksHideTheirAuthor(t *testing.T) {
	a := newArchive(t)
	found := a.search(t, WorkQuery{Text: "lighthouse"}, nil)
	anon, ok := found[a.anonymous]
	if !ok {
		t.Fatal("an anonymous work should still be found")
	}
	if !anon.Anonymous || len(anon.Authors) != 0 {
		t.Errorf("an anonymous work must not show its author, got %+v", anon.Authors)
	}
	if pub := found[a.public]; len(pub.Authors) != 1 || pub.Authors[0].Name != "Quill" {
		t.Errorf("a normal work should show its pseud, got %+v", pub.Authors)
	}

	byCreator := a.search(t, WorkQuery{Creator: "quill"}, nil)
	if _, ok := byCreator[a.anonymous]; ok {
		t.Error("searching by creator must not reveal who wrote an anonymous work")
	}
	if _, ok := byCreator[a.public]; !ok {
		t.Error("searching by creator should find their other works")
	}

	// The response must not carry account details, only the pseud.
	body, _ := json.Marshal(found[a.public])
	var fields map[string]any
	json.Unmarshal(body, &fields)
	for _, leak := range []string{"user_id", "username", "email"} {
		if _, ok := fields[leak]; ok {
			t.Errorf("search results must not include %s", leak)
		}
	}
}

func TestFilters(t *testing.T) {
	a := newArchive(t)
	cases := []struct {
		name  string
		q     WorkQuery
		want  []uuid.UUID
		avoid []uuid.UUID
	}{
		{"fandom", WorkQuery{Fandoms: []string{"Sherlock Holmes"}}, []uuid.UUID{a.crossover, a.long}, []uuid.UUID{a.public}},
		{"exclude tag", WorkQuery{Fandoms: []string{"Sherlock Holmes"}, ExcludeTags: []string{"Angst"}}, nil, []uuid.UUID{a.crossover, a.long}},
		{"no crossovers", WorkQuery{Crossovers: "exclude"}, []uuid.UUID{a.public, a.long}, []uuid.UUID{a.crossover}},
		{"only crossovers", WorkQuery{Crossovers: "only"}, []uuid.UUID{a.crossover}, []uuid.UUID{a.public}},
		{"rating", WorkQuery{Ratings: []string{"explicit"}}, []uuid.UUID{a.long}, []uuid.UUID{a.public}},
		{"word count", WorkQuery{WordCountMin: 50000}, []uuid.UUID{a.long}, []uuid.UUID{a.public}},
		{"kudos", WorkQuery{KudosMin: 10}, []uuid.UUID{a.long}, []uuid.UUID{a.public}},
		{"comments", WorkQuery{CommentsMin: 5}, []uuid.UUID{a.long}, []uuid.UUID{a.public}},
		{"tag in text", WorkQuery{Text: "fluff"}, []uuid.UUID{a.public}, []uuid.UUID{a.long}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := a.search(t, tc.q, nil)
			for _, id := range tc.want {
				if _, ok := found[id]; !ok {
					t.Errorf("expected work %s", id)
				}
			}
			for _, id := range tc.avoid {
				if _, ok := found[id]; ok {
					t.Errorf("did not expect work %s", id)
				}
			}
		})
	}
}

func TestFacetsCountTheResults(t *testing.T) {
	a := newArchive(t)
	res, err := New(a.db).Search(WorkQuery{Page: 1, Limit: 20}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Logged out: public, anonymous, crossover and long are visible.
	if res.Total != 4 {
		t.Fatalf("total = %d, want 4", res.Total)
	}
	counts := map[string]int{}
	for _, f := range res.Facets["fandoms"] {
		counts[f.Name] = f.Count
	}
	if counts["Original Work"] != 3 || counts["Sherlock Holmes"] != 2 {
		t.Errorf("fandom counts = %v", counts)
	}
}

func TestQueryParsing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	parse := func(rawQuery string) (WorkQuery, error) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/search/works?"+rawQuery, nil)
		return ParseWorkQuery(c)
	}
	q, err := parse("rating=General+Audiences&rating=teen&status=in-progress&sort=bogus")
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Ratings) != 2 || q.Ratings[0] != "general" || q.Ratings[1] != "teen" {
		t.Errorf("ratings = %v, want the stored names", q.Ratings)
	}
	if q.Complete != "in_progress" || q.Sort != "relevance" {
		t.Errorf("status %q sort %q", q.Complete, q.Sort)
	}
	if _, err := parse("page=1000&limit=100"); err == nil {
		t.Error("paging past the result limit should be refused")
	}
	if _, err := parse("word_count_min=lots"); err == nil {
		t.Error("a non-number word count should be refused")
	}
}

func TestPrefixQueryIsAlwaysValid(t *testing.T) {
	a := newArchive(t)
	for _, typed := range []string{"ligh", "o'brien", "a & b | !c", "):*", "日本語"} {
		tsq := prefixQuery(typed)
		if tsq == "" {
			continue
		}
		var ok bool
		if err := a.db.QueryRow(`SELECT to_tsquery('english', $1) IS NOT NULL`, tsq).Scan(&ok); err != nil {
			t.Errorf("%q became %q, which Postgres rejects: %v", typed, tsq, err)
		}
	}
}
