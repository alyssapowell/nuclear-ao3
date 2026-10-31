package works_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// noAccount fails if a public answer has any field naming an account, or
// contains an account id anywhere in its text.
func noAccount(t *testing.T, what, body string, accounts ...uuid.UUID) {
	t.Helper()
	neverNamesAccounts(t, what, body)
	for k := range keysOf(t, body) {
		if k == "email" {
			t.Errorf("%s has an email field: %s", what, body)
		}
	}
	for _, id := range accounts {
		if strings.Contains(body, id.String()) {
			t.Errorf("%s contains the account id %s: %s", what, id, body)
		}
	}
}

type pseudRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (e *fixEnv) newPseud(user uuid.UUID, name string) uuid.UUID {
	e.t.Helper()
	id := uuid.New()
	e.exec(`INSERT INTO pseuds (id, user_id, name) VALUES ($1, $2, $3)`, id, user, name)
	return id
}

func (e *fixEnv) credit(work, pseud uuid.UUID) {
	e.t.Helper()
	e.exec(`UPDATE creatorships SET pseud_id = $1 WHERE creation_id = $2 AND creation_type = 'Work'`, pseud, work)
}

// The pseud pages list a pseud's works, series and bookmarks, show the pseud
// as {id, name}, and never name the account.
func TestPseudPagesListWorksSeriesAndBookmarks(t *testing.T) {
	e := newFixEnv(t)
	writer, writerTok := e.user("writer")
	reader, readerTok := e.user("reader")
	pen := e.newPseud(writer, "Pen Name")
	def := e.pseudID(writer, "writer")

	w1 := e.work(writer, "Under default", "published")
	w2 := e.work(writer, "Under pen", "published")
	e.credit(w2, pen)
	secret := e.work(writer, "Anonymous", "published")
	e.exec(`UPDATE works SET is_anonymous = true WHERE id = $1`, secret)
	draft := e.work(writer, "Draft", "draft")

	// Account-keyed pages are gone: the account id is not a way in.
	for _, list := range []string{"works", "series", "bookmarks"} {
		e.expect(404, "GET", "/api/v1/users/"+writer.String()+"/"+list, readerTok, nil)
	}

	// Works.
	for _, tok := range []string{"", readerTok} {
		out := e.expect(200, "GET", "/api/v1/pseuds/"+def.String()+"/works", tok, nil)
		noAccount(t, "pseud works", out, writer, reader)
		if !strings.Contains(out, w1.String()) || strings.Contains(out, w2.String()) {
			t.Errorf("the default pseud's page should list only its own work: %s", out)
		}
		for _, hidden := range []uuid.UUID{secret, draft} {
			if strings.Contains(out, hidden.String()) {
				t.Errorf("viewer %q sees %s on the pseud's page: %s", tok, hidden, out)
			}
		}
		var got struct {
			Pseud      pseudRef
			Pagination struct{ Total int }
		}
		json.Unmarshal([]byte(out), &got)
		if got.Pseud.ID != def.String() || got.Pseud.Name != "writer" || got.Pagination.Total != 1 {
			t.Errorf("pseud and total: %+v", got)
		}
	}
	penPage := e.expect(200, "GET", "/api/v1/pseuds/"+pen.String()+"/works", readerTok, nil)
	if !strings.Contains(penPage, w2.String()) || strings.Contains(penPage, w1.String()) {
		t.Errorf("the pen name's page: %s", penPage)
	}
	// Its owner sees the anonymous work and the draft on their own pseud's page.
	own := e.expect(200, "GET", "/api/v1/pseuds/"+def.String()+"/works", writerTok, nil)
	for _, id := range []uuid.UUID{w1, secret, draft} {
		if !strings.Contains(own, id.String()) {
			t.Errorf("the owner can't see %s on their pseud's page: %s", id, own)
		}
	}
	e.expect(404, "GET", "/api/v1/pseuds/"+uuid.New().String()+"/works", "", nil)
	e.expect(400, "GET", "/api/v1/pseuds/not-a-uuid/works", "", nil)

	// Series: one with works under both pseuds, one with the secret work only,
	// and an empty one.
	mixed, onlySecret, empty := uuid.New(), uuid.New(), uuid.New()
	e.exec(`INSERT INTO series (id, user_id, title) VALUES ($1, $4, 'Mixed'), ($2, $4, 'Only Secret'), ($3, $4, 'Empty')`, mixed, onlySecret, empty, writer)
	e.exec(`INSERT INTO series_works (series_id, work_id, position) VALUES ($1, $2, 1), ($1, $3, 2), ($4, $5, 1)`, mixed, w1, w2, onlySecret, secret)
	for _, p := range []uuid.UUID{def, pen} {
		out := e.expect(200, "GET", "/api/v1/pseuds/"+p.String()+"/series", readerTok, nil)
		noAccount(t, "pseud series", out, writer, reader)
		if !strings.Contains(out, mixed.String()) || strings.Contains(out, onlySecret.String()) {
			t.Errorf("series on pseud %s: %s", p, out)
		}
		if p == def && !strings.Contains(out, empty.String()) {
			t.Errorf("an empty series belongs on the owner's default pseud: %s", out)
		}
		if p == pen && strings.Contains(out, empty.String()) {
			t.Errorf("an empty series should not be on a non-default pseud: %s", out)
		}
	}
	ownSeries := e.expect(200, "GET", "/api/v1/pseuds/"+def.String()+"/series", writerTok, nil)
	if !strings.Contains(ownSeries, onlySecret.String()) {
		t.Errorf("the owner should see their series with an anonymous work: %s", ownSeries)
	}

	// Bookmarks: reader bookmarks one open work in public, one in private,
	// and the writer bookmarks their own anonymous work.
	e.expect(201, "POST", "/api/v1/works/"+w1.String()+"/bookmark", readerTok, map[string]any{"notes": "good"})
	e.expect(201, "POST", "/api/v1/works/"+w2.String()+"/bookmark", readerTok, map[string]any{"is_private": true})
	e.expect(403, "POST", "/api/v1/works/"+w1.String()+"/bookmark", writerTok, map[string]any{"pseud_id": e.pseudIDByToken(readerTok)})
	e.expect(201, "POST", "/api/v1/works/"+secret.String()+"/bookmark", writerTok, map[string]any{"pseud_id": pen.String()})
	readerPseud := e.pseudIDByToken(readerTok)
	for _, tok := range []string{"", writerTok} {
		out := e.expect(200, "GET", "/api/v1/pseuds/"+readerPseud.String()+"/bookmarks", tok, nil)
		noAccount(t, "pseud bookmarks", out, writer, reader)
		if !strings.Contains(out, w1.String()) || strings.Contains(out, w2.String()) {
			t.Errorf("viewer %q: bookmarks %s", tok, out)
		}
		if !strings.Contains(out, `"authors":[{"id":"`+def.String()+`","name":"writer"}]`) {
			t.Errorf("viewer %q: the bookmarked work's author should be a pseud: %s", tok, out)
		}
	}
	ownMarks := e.expect(200, "GET", "/api/v1/pseuds/"+readerPseud.String()+"/bookmarks", readerTok, nil)
	if !strings.Contains(ownMarks, w2.String()) {
		t.Errorf("a bookmark's owner sees their private ones: %s", ownMarks)
	}
	// The creator's bookmark of their own anonymous work would tie the pen
	// name to it.
	for _, tok := range []string{"", readerTok} {
		out := e.expect(200, "GET", "/api/v1/pseuds/"+pen.String()+"/bookmarks", tok, nil)
		if strings.Contains(out, secret.String()) {
			t.Errorf("viewer %q: bookmarks of %s tie it to its creator: %s", tok, pen, out)
		}
	}
	if !strings.Contains(e.expect(200, "GET", "/api/v1/pseuds/"+pen.String()+"/bookmarks", writerTok, nil), secret.String()) {
		t.Errorf("the creator should still see their own bookmark")
	}
}

// Series answers carry their creators as pseuds, from their works, and a work
// that hides its creator adds nobody.
func TestSeriesAreCreditedToPseuds(t *testing.T) {
	e := newFixEnv(t)
	writer, writerTok := e.user("writer")
	other, _ := e.user("other")
	_, readerTok := e.user("reader")
	moon := e.newPseud(writer, "Moonlit Quill")
	def := e.pseudID(writer, "writer")
	otherPseud := e.pseudID(other, "other")

	open := e.work(writer, "Open", "published")
	secret := e.work(writer, "Secret", "published")
	e.credit(secret, moon)
	e.exec(`UPDATE works SET in_unrevealed_collection = true WHERE id = $1`, secret)
	theirs := e.work(other, "Theirs", "published")
	series, empty := uuid.New(), uuid.New()
	e.exec(`INSERT INTO series (id, user_id, title) VALUES ($1, $3, 'Shared'), ($2, $3, 'Not started')`, series, empty, writer)
	e.exec(`INSERT INTO series_works (series_id, work_id, position) VALUES ($1, $2, 1), ($1, $3, 2), ($1, $4, 3)`, series, open, secret, theirs)

	authorsOf := func(body, seriesID string) []pseudRef {
		var all struct {
			Series []struct {
				ID      string
				Authors []pseudRef
			}
		}
		json.Unmarshal([]byte(body), &all)
		for _, s := range all.Series {
			if s.ID == seriesID {
				return s.Authors
			}
		}
		t.Fatalf("series %s not in %s", seriesID, body)
		return nil
	}
	names := func(a []pseudRef) string {
		var n []string
		for _, p := range a {
			n = append(n, p.Name)
		}
		return strings.Join(n, ",")
	}

	for _, tok := range []string{"", readerTok} {
		list := e.expect(200, "GET", "/api/v1/series", tok, nil)
		noAccount(t, "series search", list, writer, other)
		if got := names(authorsOf(list, series.String())); got != "other,writer" {
			t.Errorf("viewer %q: series authors = %q, want other,writer (no Moonlit Quill)", tok, got)
		}
		if strings.Contains(list, moon.String()) || strings.Contains(list, "Moonlit Quill") {
			t.Errorf("viewer %q: the hidden creator's pseud is in the series list: %s", tok, list)
		}
		one := e.expect(200, "GET", "/api/v1/series/"+series.String(), tok, nil)
		noAccount(t, "series", one, writer, other)
		if strings.Contains(one, "Moonlit Quill") {
			t.Errorf("viewer %q: %s", tok, one)
		}
		var single struct{ Series struct{ Authors []pseudRef } }
		json.Unmarshal([]byte(one), &single)
		if names(single.Series.Authors) != "other,writer" || single.Series.Authors[1].ID != def.String() || single.Series.Authors[0].ID != otherPseud.String() {
			t.Errorf("viewer %q: series authors %v", tok, single.Series.Authors)
		}
		// A series with no works yet is credited to its owner's default pseud.
		if got := names(authorsOf(list, empty.String())); got != "writer" {
			t.Errorf("empty series authors = %q", got)
		}
	}
	// The creator of the hidden work sees their own pseud too.
	own := e.expect(200, "GET", "/api/v1/series/"+series.String(), writerTok, nil)
	noAccount(t, "own series", own, writer, other)
	if !strings.Contains(own, "Moonlit Quill") {
		t.Errorf("the creator should see their own pseud on the series: %s", own)
	}
	mine := e.expect(200, "GET", "/api/v1/my/series", writerTok, nil)
	noAccount(t, "my series", mine, writer, other)
	// Creating and updating answer the same way.
	created := e.expect(201, "POST", "/api/v1/series", writerTok, map[string]any{"title": "New", "work_ids": []string{open.String()}})
	noAccount(t, "created series", created, writer)
	var made struct{ Series struct{ ID string } }
	json.Unmarshal([]byte(created), &made)
	updated := e.expect(200, "PUT", "/api/v1/series/"+made.Series.ID, writerTok, map[string]any{"title": "Newer"})
	noAccount(t, "updated series", updated, writer)
	if !strings.Contains(updated, `"authors":[{"id":"`+def.String()+`","name":"writer"}]`) {
		t.Errorf("updated series authors: %s", updated)
	}
}

// Kudos lists show the giver's default pseud, count guests, and carry nothing
// about the account.
func TestKudosListShowsPseudsAndCountsGuests(t *testing.T) {
	e := newFixEnv(t)
	writer, _ := e.user("writer")
	fan, fanTok := e.user("fan")
	e.newPseud(fan, "Secondary Name")
	fanDefault := e.pseudID(fan, "fan")
	work := e.work(writer, "Liked", "published")
	e.exec(`INSERT INTO kudos (work_id, user_id) VALUES ($1, $2)`, work, fan)
	e.exec(`INSERT INTO kudos (work_id, guest_session, ip_address) VALUES ($1, 'g1', '10.0.0.1'), ($1, 'g2', '10.0.0.2')`, work)

	for _, tok := range []string{"", fanTok} {
		out := e.expect(200, "GET", "/api/v1/works/"+work.String()+"/kudos", tok, nil)
		noAccount(t, "kudos", out, fan, writer)
		var got struct {
			Kudos      []struct{ Pseud pseudRef }
			GuestCount int `json:"guest_count"`
			Total      int `json:"total_count"`
		}
		json.Unmarshal([]byte(out), &got)
		if len(got.Kudos) != 1 || got.Kudos[0].Pseud.ID != fanDefault.String() || got.Kudos[0].Pseud.Name != "fan" {
			t.Errorf("kudos list: %s", out)
		}
		if got.GuestCount != 2 || got.Total != 3 {
			t.Errorf("guest_count = %d, total_count = %d, want 2 and 3", got.GuestCount, got.Total)
		}
		if strings.Contains(out, "Guest") || strings.Contains(out, "10.0.0") {
			t.Errorf("guests are only counted: %s", out)
		}
	}
	// A creator's kudos on their own hidden work would name them.
	e.exec(`UPDATE works SET is_anonymous = true WHERE id = $1`, work)
	e.exec(`INSERT INTO kudos (work_id, user_id) VALUES ($1, $2)`, work, writer)
	out := e.expect(200, "GET", "/api/v1/works/"+work.String()+"/kudos", fanTok, nil)
	if strings.Contains(out, e.pseudID(writer, "writer").String()) {
		t.Errorf("the hidden creator's kudos is listed: %s", out)
	}
}

// Author subscriptions are made and listed by pseud; the account id and the
// username are never taken or given.
func TestAuthorSubscriptionsUseAPseudNotAnAccount(t *testing.T) {
	e := newFixEnv(t)
	writer, _ := e.user("writer")
	fan, fanTok := e.user("fan")
	_, otherTok := e.user("other")
	pen := e.newPseud(writer, "Pen Name")
	def := e.pseudID(writer, "writer")

	made := e.expect(201, "POST", "/api/v1/subscriptions", fanTok, map[string]any{
		"type": "author", "target_id": pen.String(), "target_name": "client says another name"})
	noAccount(t, "created subscription", made, writer, fan)
	if !strings.Contains(made, `"target_name":"Pen Name"`) || !strings.Contains(made, pen.String()) {
		t.Errorf("created subscription: %s", made)
	}
	list := e.expect(200, "GET", "/api/v1/subscriptions", fanTok, nil)
	noAccount(t, "subscription list", list, writer, fan)
	var got struct {
		Subscriptions []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			TargetID string `json:"target_id"`
			Name     string `json:"target_name"`
		}
	}
	json.Unmarshal([]byte(list), &got)
	if len(got.Subscriptions) != 1 || got.Subscriptions[0].TargetID != pen.String() || got.Subscriptions[0].Name != "Pen Name" {
		t.Fatalf("subscription list: %s", list)
	}
	// The status check is per pseud: the writer's other pseud is not subscribed,
	// so the answer can't be used to tell the two belong together.
	status := func(p uuid.UUID) string {
		return e.expect(200, "GET", "/api/v1/subscription-status?type=author&target_id="+p.String(), fanTok, nil)
	}
	if s := status(pen); !strings.Contains(s, `"subscribed":true`) {
		t.Errorf("status of the subscribed pseud: %s", s)
	}
	if s := status(def); !strings.Contains(s, `"subscribed":false`) {
		t.Errorf("status of the writer's other pseud: %s", s)
	}
	// Subscribing again is not a second subscription; another pseud is.
	e.expect(201, "POST", "/api/v1/subscriptions", fanTok, map[string]any{"type": "author", "target_id": pen.String()})
	e.expect(201, "POST", "/api/v1/subscriptions", fanTok, map[string]any{"type": "author", "target_id": def.String()})
	if n := e.count(`SELECT count(*) FROM subscriptions WHERE user_id = $1`, fan); n != 2 {
		t.Errorf("subscriptions = %d, want 2", n)
	}
	// An account id is not a pseud, and neither is a made-up id.
	e.expect(404, "POST", "/api/v1/subscriptions", fanTok, map[string]any{"type": "author", "target_id": writer.String()})
	e.expect(404, "POST", "/api/v1/subscriptions", fanTok, map[string]any{"type": "author", "target_id": uuid.New().String()})
	// A renamed pseud shows its new name.
	e.exec(`UPDATE pseuds SET name = 'Renamed Pen' WHERE id = $1`, pen)
	if !strings.Contains(e.expect(200, "GET", "/api/v1/subscriptions", fanTok, nil), "Renamed Pen") {
		t.Errorf("the list should show the pseud's current name")
	}
	// Nobody else sees it, and it can be removed.
	if strings.Contains(e.expect(200, "GET", "/api/v1/subscriptions", otherTok, nil), pen.String()) {
		t.Errorf("another user's list shows the subscription")
	}
	e.expect(200, "DELETE", "/api/v1/subscriptions/"+got.Subscriptions[0].ID, fanTok, nil)
	if n := e.count(`SELECT count(*) FROM subscriptions WHERE user_id = $1 AND target_id = $2`, fan, pen); n != 0 {
		t.Errorf("subscription not deleted")
	}
	// Deleting a pseud ends subscriptions to it, rather than moving them to the
	// default pseud and showing the two are the same person.
	writerTok := "token-writer"
	e.expect(201, "POST", "/api/v1/subscriptions", fanTok, map[string]any{"type": "author", "target_id": pen.String()})
	e.expect(200, "DELETE", "/api/v1/pseuds/"+pen.String(), writerTok, nil)
	if n := e.count(`SELECT count(*) FROM subscriptions WHERE user_id = $1 AND target_id = $2`, fan, pen); n != 0 {
		t.Errorf("a deleted pseud still has subscriptions")
	}
	if n := e.count(`SELECT count(*) FROM subscriptions WHERE user_id = $1 AND target_id = $2`, fan, def); n != 1 {
		t.Errorf("the subscription to the default pseud should be untouched")
	}
}
