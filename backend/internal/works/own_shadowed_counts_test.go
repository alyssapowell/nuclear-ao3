package works_test

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"nuclear-ao3/internal/platform"
	"nuclear-ao3/internal/search"
)

// A silently blocked person's own shadowed kudos and comments are left out of
// the public counts. If their view of the counts left them out too, they
// would give kudos, see the number stay the same, and know they were blocked.
// So every count a viewer is shown includes that viewer's own shadowed items,
// and everyone else sees the public count.

// seenCounts is every kudos and comment count one viewer is shown for a work.
type seenCounts struct {
	PageKudos, PageComments     int
	ListKudos, ListComments     int
	StatsKudos, StatsComments   int
	SearchKudos, SearchComments int
	KudosTotal                  int
	KudosListed                 int
}

func (w *blockWorld) mountSearch() {
	w.e.t.Helper()
	deps := platform.NewDeps(w.e.db, func(tok string) (uuid.UUID, error) {
		if id, ok := w.e.tokens[tok]; ok {
			return id, nil
		}
		return uuid.Nil, sql.ErrNoRows
	})
	search.Mount(w.e.router, search.New(w.e.db), deps)
}

func decode(t *testing.T, body string, into any) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), into); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
}

func (w *blockWorld) seen(tok string) seenCounts {
	w.e.t.Helper()
	e := w.e
	var s seenCounts
	base := "/api/v1/work/" + w.work.String()

	var page struct {
		Work struct{ Kudos, Comments int } `json:"work"`
	}
	decode(e.t, e.expect(200, "GET", base, tok, nil), &page)
	s.PageKudos, s.PageComments = page.Work.Kudos, page.Work.Comments

	// The works list, narrowed to this one work by its title.
	var list struct {
		Works []struct {
			ID              string
			Kudos, Comments int
			Hits, Bookmarks int
		} `json:"works"`
	}
	decode(e.t, e.expect(200, "GET", "/api/v1/works?q=Lighthouse", tok, nil), &list)
	for _, x := range list.Works {
		if x.ID == w.work.String() {
			s.ListKudos, s.ListComments = x.Kudos, x.Comments
		}
	}

	var stats struct{ Kudos, Comments int }
	decode(e.t, e.expect(200, "GET", base+"/stats", tok, nil), &stats)
	s.StatsKudos, s.StatsComments = stats.Kudos, stats.Comments

	var found struct {
		Works []struct {
			ID           string `json:"id"`
			KudosCount   int    `json:"kudos_count"`
			CommentCount int    `json:"comment_count"`
		} `json:"works"`
	}
	decode(e.t, e.expect(200, "GET", "/api/v1/search/works?title=Lighthouse", tok, nil), &found)
	for _, x := range found.Works {
		if x.ID == w.work.String() {
			s.SearchKudos, s.SearchComments = x.KudosCount, x.CommentCount
		}
	}

	var kudos struct {
		Total int `json:"total_count"`
		List  []struct {
			Pseud struct{ ID string } `json:"pseud"`
		} `json:"kudos"`
	}
	decode(e.t, e.expect(200, "GET", base+"/kudos", tok, nil), &kudos)
	s.KudosTotal = kudos.Total
	for _, k := range kudos.List {
		if k.Pseud.ID == w.ex.String() {
			s.KudosListed++
		}
	}
	return s
}

func TestBlockedPersonSeesCountsThatIncludeTheirOwnShadowedItems(t *testing.T) {
	for _, mode := range hidingModes {
		t.Run(mode, func(t *testing.T) {
			w := newBlockWorld(t)
			e := w.e
			w.mountSearch()
			w.hide(mode)
			w.blockEx()
			// Someone else's ordinary kudos, so the count isn't zero.
			e.expect(201, "POST", "/api/v1/works/"+w.work.String()+"/kudos", w.readerTok, nil)

			viewers := map[string]string{"blocked": w.blockedTok, "creator": w.blockerTok, "reader": w.readerTok, "guest": ""}
			before := map[string]seenCounts{}
			for name, tok := range viewers {
				before[name] = w.seen(tok)
				b := before[name]
				if b.PageKudos != 1 || b.KudosTotal != 1 || b.PageComments != 3 {
					t.Fatalf("%s starts with unexpected counts: %+v", name, b)
				}
			}

			// The blocked person leaves kudos and a comment: both shadowed.
			base := "/api/v1/work/" + w.work.String()
			e.expect(201, "POST", "/api/v1/works/"+w.work.String()+"/kudos", w.blockedTok, nil)
			e.expect(201, "POST", base+"/comments", w.blockedTok, map[string]any{"content": "hello from the blocked one"})
			if n := e.count(`SELECT count(*) FROM kudos WHERE work_id = $1 AND shadowed`, w.work); n != 1 {
				t.Fatalf("shadowed kudos = %d, want 1", n)
			}

			// They see everything counted once more, and their kudos listed.
			want := before["blocked"]
			want.PageKudos++
			want.ListKudos++
			want.StatsKudos++
			want.KudosTotal++
			want.KudosListed++
			want.PageComments++
			want.ListComments++
			want.StatsComments++
			// Search leaves out works in a collection that hasn't revealed its
			// creators, so there is nothing to count there.
			if mode != "in_unrevealed_collection" {
				want.SearchKudos++
				want.SearchComments++
			}
			if got := w.seen(w.blockedTok); got != want {
				t.Errorf("blocked person sees %+v, want %+v", got, want)
			}

			// Nobody else sees any change, the creator least of all.
			for _, name := range []string{"creator", "reader", "guest"} {
				if got := w.seen(viewers[name]); got != before[name] {
					t.Errorf("%s sees %+v, want the unchanged %+v", name, got, before[name])
				}
			}

			// The stored public counts did not move.
			if k, c := w.workCount("kudos_count"), w.workCount("comment_count"); k != 1 || c != 3 {
				t.Errorf("public counts moved: kudos %d, comments %d", k, c)
			}
		})
	}
}

// Someone who is not blocked is not affected: their own items are public.
func TestUnblockedReaderCountsAreThePublicCounts(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	w.mountSearch()
	w.hide("is_anonymous")
	w.blockEx()
	base := "/api/v1/work/" + w.work.String()
	e.expect(201, "POST", "/api/v1/works/"+w.work.String()+"/kudos", w.readerTok, nil)
	e.expect(201, "POST", base+"/comments", w.readerTok, map[string]any{"content": "nice"})
	got := w.seen(w.readerTok)
	if got.PageKudos != 1 || got.ListKudos != 1 || got.StatsKudos != 1 || got.SearchKudos != 1 || got.KudosTotal != 1 {
		t.Errorf("reader kudos counts: %+v", got)
	}
	if got.PageComments != 4 || got.ListComments != 4 || got.StatsComments != 4 || got.SearchComments != 4 {
		t.Errorf("reader comment counts: %+v", got)
	}
	if got != w.seen(w.blockerTok) {
		t.Errorf("the reader and the creator should see the same counts")
	}
}

// Filtering and sorting by kudos use the public count (a trade-off, see
// search/works.go): the blocked person's own shadowed kudos don't make the
// work pass a kudos_min filter.
func TestFiltersUseThePublicCount(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	w.mountSearch()
	w.hide("is_anonymous")
	w.blockEx()
	e.expect(201, "POST", "/api/v1/works/"+w.work.String()+"/kudos", w.blockedTok, nil)
	var found struct {
		Works []struct{ ID string } `json:"works"`
	}
	decode(t, e.expect(200, "GET", "/api/v1/search/works?title=Lighthouse&kudos_min=1", w.blockedTok, nil), &found)
	if len(found.Works) != 0 {
		t.Errorf("kudos_min=1 matched on a shadowed kudos: %+v", found.Works)
	}
}

// Replying to a blocker's comment: on a work that hides its creator it is the
// normal answer, kept to the replier; on a signed work it is refused.
func TestReplyToABlockersCommentOnHiddenAndSignedWorks(t *testing.T) {
	w := newBlockWorld(t)
	e := w.e
	w.blockEx()

	// The blocker's comment is on the bystander's work (w.readerWork).
	hiddenPath := "/api/v1/work/" + w.readerWork.String() + "/comments"
	reply := func() map[string]any {
		return map[string]any{"content": "replying to the blocker", "parent_comment_id": w.bOnReader}
	}

	// Signed: refused.
	out := e.expect(403, "POST", hiddenPath, w.blockedTok, reply())
	if !strings.Contains(out, "replies_disabled") {
		t.Errorf("signed work: %s", out)
	}

	// Hiding its creator: the normal 201, and the reply is shadowed.
	e.exec(`UPDATE works SET is_anonymous = true WHERE id = $1`, w.readerWork)
	e.expect(201, "POST", hiddenPath, w.blockedTok, reply())
	if n := e.count(`SELECT count(*) FROM comments WHERE work_id = $1 AND parent_comment_id = $2 AND shadowed AND content = 'replying to the blocker'`,
		w.readerWork, w.bOnReader); n != 1 {
		t.Errorf("shadowed replies = %d, want 1", n)
	}
	if has(w.comments(w.readerWork, w.readerTok), "replying to the blocker") {
		t.Errorf("another reader sees the shadowed reply")
	}
	if !has(w.comments(w.readerWork, w.blockedTok), "replying to the blocker") {
		t.Errorf("the replier can't see their own reply")
	}
}
