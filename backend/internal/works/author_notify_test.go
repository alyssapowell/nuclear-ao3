package works_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/jobs"
	"nuclear-ao3/internal/notify"
	"nuclear-ao3/shared/email"
)

// runJobs works through the queue the way the app's workers do.
func (e *fixEnv) runJobs(mail *email.Config) {
	e.t.Helper()
	q := jobs.New(e.db)
	q.Handle(notify.PostedJobKind, notify.New(e.db, mail).HandlePosted)
	q.Handle("clean-html", func(context.Context, json.RawMessage) error { return nil })
	q.Handle("email", func(context.Context, json.RawMessage) error { return nil })
	for i := 0; i < 100; i++ {
		worked, err := q.RunOne(context.Background())
		if err != nil {
			e.t.Fatal(err)
		}
		if !worked {
			return
		}
	}
	e.t.Fatal("job queue never drained")
}

func (e *fixEnv) followPseud(user, pseud uuid.UUID) {
	e.t.Helper()
	e.followTarget(user, "author", pseud)
}

func (e *fixEnv) followTarget(user uuid.UUID, kind string, target uuid.UUID) {
	e.t.Helper()
	e.exec(`INSERT INTO subscriptions (user_id, type, target_id, target_name, events)
		VALUES ($1, $2, $3, 'x', $4)`, user, kind, target, pq.Array([]string{"new_work", "work_updated", "series_updated"}))
}

// heard returns the titles and descriptions of everything the user was told.
func (e *fixEnv) heard(user uuid.UUID) []string {
	e.t.Helper()
	rows, err := e.db.Query(`SELECT title || ' | ' || coalesce(description, '') || ' | ' || coalesce(extra_data::text, '') || coalesce(actor_name, '')
		|| coalesce(actor_id::text, '') FROM notification_items WHERE user_id = $1 ORDER BY created_at`, user)
	if err != nil {
		e.t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

func (e *fixEnv) post(tok string, work uuid.UUID) {
	e.t.Helper()
	e.expect(200, "PUT", "/api/v1/works/"+work.String(), tok, map[string]any{"status": "posted"})
}

type announceEnv struct {
	*fixEnv
	writer, fan       uuid.UUID
	writerTok         string
	moon, star, spare uuid.UUID
}

func newAnnounceEnv(t *testing.T) *announceEnv {
	e := newFixEnv(t)
	w, wt := e.user("writer")
	fan, _ := e.user("fan")
	return &announceEnv{fixEnv: e, writer: w, fan: fan, writerTok: wt,
		moon:  e.newPseud(w, "Moonlit Quill"),
		star:  e.newPseud(w, "Starfall"),
		spare: e.newPseud(w, "Unrelated Spare")}
}

func (a *announceEnv) draft(title string, pseuds ...uuid.UUID) uuid.UUID {
	w := a.work(a.writer, title, "draft")
	a.exec(`DELETE FROM creatorships WHERE creation_id = $1`, w)
	// the fixture leaves columns null that the work page's loader reads
	a.exec(`UPDATE works SET notes = '', category = '{}', warnings = '{}', characters = '{}', relationships = '{}',
		freeform_tags = '{}', max_chapters = coalesce(max_chapters, 1) WHERE id = $1`, w)
	for _, p := range pseuds {
		a.exec(`INSERT INTO creatorships (creation_id, creation_type, pseud_id) VALUES ($1, 'Work', $2)`, w, p)
	}
	return w
}

func TestPostingAWorkNotifiesPseudSubscribersOnceAndNamesOnlyFollowedPseuds(t *testing.T) {
	a := newAnnounceEnv(t)
	a.followPseud(a.fan, a.moon)
	a.followPseud(a.fan, a.star) // two pseuds of the work: still one notification
	work := a.draft("Tidal", a.moon, a.star, a.spare)

	a.runJobs(nil)
	if got := a.heard(a.fan); len(got) != 0 {
		t.Fatalf("told about a draft: %v", got)
	}
	a.post(a.writerTok, work)
	a.runJobs(nil)
	got := a.heard(a.fan)
	if len(got) != 1 {
		t.Fatalf("want one notification, got %v", got)
	}
	n := got[0]
	if !strings.Contains(n, "Moonlit Quill, Starfall") || !strings.Contains(n, "Tidal") {
		t.Errorf("should name the work and followed pseuds: %s", n)
	}
	for _, secret := range []string{"Unrelated Spare", "writer", a.writer.String()} {
		if strings.Contains(n, secret) {
			t.Errorf("gives away %q: %s", secret, n)
		}
	}
	// Re-posting after a trip back to draft is not a new work.
	a.expect(200, "PUT", "/api/v1/works/"+work.String(), a.writerTok, map[string]any{"status": "draft"})
	a.post(a.writerTok, work)
	a.runJobs(nil)
	if got := a.heard(a.fan); len(got) != 1 {
		t.Errorf("re-posting announced again: %v", got)
	}
	// The creator is told nothing about their own work.
	if got := a.heard(a.writer); len(got) != 0 {
		t.Errorf("creator told: %v", got)
	}
}

func TestOnlyFollowedPseudIsNamedWhenWorkHasSeveral(t *testing.T) {
	a := newAnnounceEnv(t)
	a.followPseud(a.fan, a.star)
	work := a.draft("Tidal", a.moon, a.star)
	a.post(a.writerTok, work)
	a.runJobs(nil)
	got := a.heard(a.fan)
	if len(got) != 1 || !strings.Contains(got[0], "Starfall") || strings.Contains(got[0], "Moonlit Quill") {
		t.Fatalf("got %v", got)
	}
}

func TestNewChapterNotifiesSubscribers(t *testing.T) {
	a := newAnnounceEnv(t)
	a.followPseud(a.fan, a.moon)
	work := a.work(a.writer, "Serial", "published")
	a.credit(work, a.moon)

	a.expect(201, "POST", "/api/v1/works/"+work.String()+"/chapters", a.writerTok,
		map[string]any{"title": "two", "content": "more", "status": "draft"})
	a.runJobs(nil)
	if got := a.heard(a.fan); len(got) != 0 {
		t.Fatalf("told about a draft chapter: %v", got)
	}
	a.expect(201, "POST", "/api/v1/works/"+work.String()+"/chapters", a.writerTok,
		map[string]any{"title": "three", "content": "more", "status": "posted"})
	a.runJobs(nil)
	got := a.heard(a.fan)
	if len(got) != 1 || !strings.Contains(got[0], "New chapter") || !strings.Contains(got[0], "Moonlit Quill") {
		t.Fatalf("got %v", got)
	}
	// Posting the draft chapter later is its own event.
	var draftCh uuid.UUID
	a.db.QueryRow(`SELECT id FROM chapters WHERE work_id = $1 AND title = 'two'`, work).Scan(&draftCh)
	a.expect(200, "PUT", "/api/v1/works/"+work.String()+"/chapters/"+draftCh.String(), a.writerTok, map[string]any{"status": "posted"})
	a.runJobs(nil)
	if got := a.heard(a.fan); len(got) != 2 {
		t.Fatalf("want two after the second chapter, got %v", got)
	}
}

func TestChaptersOfADraftWorkAreNotAnnounced(t *testing.T) {
	a := newAnnounceEnv(t)
	a.followPseud(a.fan, a.moon)
	work := a.draft("Hidden away", a.moon)
	a.expect(201, "POST", "/api/v1/works/"+work.String()+"/chapters", a.writerTok,
		map[string]any{"title": "two", "content": "more", "status": "posted"})
	a.runJobs(nil)
	if got := a.heard(a.fan); len(got) != 0 {
		t.Fatalf("told: %v", got)
	}
}

func TestSubscribersOfHiddenCreatorsAreNeverToldAndRevealDoesNotAnnounce(t *testing.T) {
	a := newAnnounceEnv(t)
	a.followPseud(a.fan, a.moon)
	for _, set := range []string{"is_anonymous", "in_anon_collection", "in_unrevealed_collection"} {
		work := a.draft("Secret "+set, a.moon)
		a.exec(`UPDATE works SET `+set+` = true WHERE id = $1`, work)
		a.post(a.writerTok, work)
		a.expect(201, "POST", "/api/v1/works/"+work.String()+"/chapters", a.writerTok,
			map[string]any{"title": "next", "content": "more", "status": "posted"})
		a.runJobs(nil)
		if got := a.heard(a.fan); len(got) != 0 {
			t.Fatalf("%s: author subscriber told: %v", set, got)
		}
		if set == "in_unrevealed_collection" {
			a.exec(`UPDATE works SET in_unrevealed_collection = false WHERE id = $1`, work)
			a.runJobs(nil)
			if got := a.heard(a.fan); len(got) != 0 {
				t.Fatalf("reveal announced: %v", got)
			}
		}
	}
}

func TestWorkSubscriberOfAnAnonymousWorkIsToldWithoutNames(t *testing.T) {
	a := newAnnounceEnv(t)
	a.followPseud(a.fan, a.moon)
	work := a.work(a.writer, "Anon serial", "published")
	a.credit(work, a.moon)
	a.exec(`UPDATE works SET is_anonymous = true WHERE id = $1`, work)
	a.followTarget(a.fan, "work", work)
	a.expect(201, "POST", "/api/v1/works/"+work.String()+"/chapters", a.writerTok,
		map[string]any{"title": "next", "content": "more", "status": "posted"})
	a.runJobs(nil)
	got := a.heard(a.fan)
	if len(got) != 1 || strings.Contains(got[0], "Moonlit") || strings.Contains(got[0], a.writer.String()) {
		t.Fatalf("got %v", got)
	}
}

func TestNoNotificationForAdminHiddenWorkOrMutedCreator(t *testing.T) {
	a := newAnnounceEnv(t)
	a.followPseud(a.fan, a.moon)

	// hidden by an admin between queueing and running
	hidden := a.draft("Hidden", a.moon)
	a.post(a.writerTok, hidden)
	a.exec(`UPDATE works SET hidden_by_admin = true WHERE id = $1`, hidden)
	a.runJobs(nil)
	if got := a.heard(a.fan); len(got) != 0 {
		t.Fatalf("hidden work announced: %v", got)
	}

	// a subscriber who can't see the work: muted pseud
	muted := a.draft("Muted", a.moon)
	a.exec(`INSERT INTO user_mutes (muter_id, muted_pseud_id) VALUES ($1, $2)`, a.fan, a.moon)
	a.post(a.writerTok, muted)
	a.runJobs(nil)
	if got := a.heard(a.fan); len(got) != 0 {
		t.Fatalf("muted creator announced: %v", got)
	}
}

func TestBlockedSubscriberGetsNothingInEitherDirection(t *testing.T) {
	for _, writerBlocks := range []bool{true, false} {
		a := newAnnounceEnv(t)
		a.followPseud(a.fan, a.moon)
		blocker, blocked := a.fan, a.writer
		if writerBlocks {
			blocker, blocked = a.writer, a.fan
		}
		a.exec(`INSERT INTO user_blocks (blocker_id, blocked_id, pseud_id, pseud_name)
			SELECT $1, $2, p.id, p.name FROM pseuds p WHERE p.user_id = $1 AND p.is_default`, blocker, blocked)
		work := a.draft("Blocked", a.moon)
		a.post(a.writerTok, work)
		a.runJobs(nil)
		if got := a.heard(a.fan); len(got) != 0 {
			t.Fatalf("writerBlocks=%v: told: %v", writerBlocks, got)
		}
	}
}

func TestSubscriberOfPseudWorkAndSeriesGetsOneNotificationPerEvent(t *testing.T) {
	a := newAnnounceEnv(t)
	work := a.work(a.writer, "Part", "published")
	a.credit(work, a.moon)
	series := uuid.New()
	a.exec(`INSERT INTO series (id, user_id, title) VALUES ($1, $2, 'Saga')`, series, a.writer)
	a.exec(`INSERT INTO series_works (series_id, work_id, position) VALUES ($1, $2, 1)`, series, work)
	a.followPseud(a.fan, a.moon)
	a.followTarget(a.fan, "work", work)
	a.followTarget(a.fan, "series", series)

	a.expect(201, "POST", "/api/v1/works/"+work.String()+"/chapters", a.writerTok,
		map[string]any{"title": "next", "content": "more", "status": "posted"})
	a.runJobs(nil)
	if got := a.heard(a.fan); len(got) != 1 {
		t.Fatalf("want one, got %v", got)
	}
	// Running the same job again (a worker died before marking it done) is quiet too.
	a.exec(`UPDATE jobs SET done_at = NULL, failed_at = NULL, locked_at = NULL`)
	a.runJobs(nil)
	if got := a.heard(a.fan); len(got) != 1 {
		t.Fatalf("a repeated job announced twice: %v", got)
	}
}

func TestAnnouncementsGoThroughTheQueueAndFollowPreferences(t *testing.T) {
	a := newAnnounceEnv(t)
	a.followPseud(a.fan, a.moon)
	work := a.draft("Queued", a.moon)
	a.post(a.writerTok, work)
	// Posting only queued a job; nobody has been told yet.
	if n := a.count(`SELECT count(*) FROM jobs WHERE kind = 'work-posted' AND done_at IS NULL`); n != 1 {
		t.Fatalf("want one queued job, got %d", n)
	}
	if got := a.heard(a.fan); len(got) != 0 {
		t.Fatalf("told before the job ran: %v", got)
	}
	// With email configured, the notification also queues an email.
	a.runJobs(&email.Config{SiteURL: "https://archive.example", FromName: "Archive"})
	if n := a.count(`SELECT count(*) FROM jobs WHERE kind = 'email'`); n != 1 {
		t.Errorf("want one email job, got %d", n)
	}
	var body string
	a.db.QueryRow(`SELECT payload::text FROM jobs WHERE kind = 'email'`).Scan(&body)
	if strings.Contains(body, "Unrelated Spare") || strings.Contains(body, a.writer.String()) {
		t.Errorf("email gives away more than the followed pseud: %s", body)
	}

	// A person who turned the event off gets neither.
	other, _ := a.user("other")
	a.followPseud(other, a.moon)
	a.exec(`INSERT INTO user_notification_preferences (user_id, event_preferences)
		VALUES ($1, '{"new_work": {"Enabled": false}}')`, other)
	work2 := a.draft("Second", a.moon)
	a.post(a.writerTok, work2)
	a.runJobs(nil)
	if got := a.heard(other); len(got) != 0 {
		t.Errorf("preferences ignored: %v", got)
	}
}
