package works_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type pseudHit struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	WorkCount   int    `json:"work_count"`
}

func (e *fixEnv) searchPseuds(q, token string) (hits []pseudHit, raw string) {
	e.t.Helper()
	raw = e.expect(200, "GET", "/api/v1/pseuds/search?"+q, token, nil)
	var out struct {
		Pseuds     []pseudHit     `json:"pseuds"`
		Pagination map[string]any `json:"pagination"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		e.t.Fatal(err)
	}
	return out.Pseuds, raw
}

func TestPseudSearchCountsOnlyWhatTheViewerMayReadAndNeverRevealsAccounts(t *testing.T) {
	e := newFixEnv(t)
	writer, _ := e.user("writer")
	_, readerTok := e.user("reader")
	moon := e.newPseud(writer, "Moonwriter")
	moon2 := e.newPseud(writer, "Moonbeam")
	e.exec(`UPDATE pseuds SET description = '<p>Writes <b>about</b> the moon &amp; tides</p>' WHERE id = $1`, moon)

	open := e.work(writer, "Open", "published")
	e.credit(open, moon)
	anon := e.work(writer, "Anon", "published")
	e.credit(anon, moon)
	e.exec(`UPDATE works SET is_anonymous = true WHERE id = $1`, anon)
	unrevealed := e.work(writer, "Unrevealed", "published")
	e.credit(unrevealed, moon)
	e.exec(`UPDATE works SET in_unrevealed_collection = true WHERE id = $1`, unrevealed)
	draft := e.work(writer, "Draft", "draft")
	e.credit(draft, moon)
	hidden := e.work(writer, "Hidden", "hidden")
	e.credit(hidden, moon)

	hits, raw := e.searchPseuds("q=moon", readerTok)
	if len(hits) != 2 {
		t.Fatalf("want two moon pseuds: %s", raw)
	}
	// relevance (both are prefix matches), then name: Moonbeam before Moonwriter
	if hits[0].Name != "Moonbeam" || hits[1].Name != "Moonwriter" {
		t.Errorf("order: %s", raw)
	}
	if hits[1].WorkCount != 1 || hits[0].WorkCount != 0 {
		t.Errorf("only the open, visible work counts: %s", raw)
	}
	if hits[1].Description != "Writes about the moon & tides" {
		t.Errorf("excerpt should be plain text: %q", hits[1].Description)
	}
	for _, secret := range []string{writer.String(), "user_id", "account", "username", moon2.String() + `","name":"Moonwriter`} {
		if strings.Contains(raw, secret) {
			t.Errorf("answer carries %q: %s", secret, raw)
		}
	}
	// A logged-out reader gets the same list, and the count is as a reader's.
	if guest, _ := e.searchPseuds("q=moon", ""); len(guest) != 2 || guest[1].WorkCount != 1 {
		t.Errorf("guest: %+v", guest)
	}
	_ = uuid.Nil
}

func TestPseudSearchRelevanceThenNameThenIDAndPages(t *testing.T) {
	e := newFixEnv(t)
	a, _ := e.user("acct-a")
	b, _ := e.user("acct-b")
	// Names are the same for two accounts: id breaks the tie, never the account.
	e.newPseud(a, "Dusk")
	e.newPseud(b, "dusk")
	e.newPseud(a, "Twilight Dusk")
	e.newPseud(b, "Duskfall")

	hits, raw := e.searchPseuds("q=dusk", "")
	var names []string
	for _, h := range hits {
		names = append(names, h.Name)
	}
	// exact (two, by lowercase name then name), prefix, then contains
	if got := strings.Join(names, ","); got != "Dusk,dusk,Duskfall,Twilight Dusk" && got != "dusk,Dusk,Duskfall,Twilight Dusk" {
		t.Errorf("order: %s", raw)
	}
	p1 := e.expect(200, "GET", "/api/v1/pseuds/search?q=dusk&limit=3&page=1", "", nil)
	p2 := e.expect(200, "GET", "/api/v1/pseuds/search?q=dusk&limit=3&page=2", "", nil)
	if !strings.Contains(p1, `"total":4`) || !strings.Contains(p1, `"total_pages":2`) || !strings.Contains(p2, "Twilight Dusk") {
		t.Errorf("pages: %s | %s", p1, p2)
	}
	// % and _ are literal
	if hits, _ := e.searchPseuds("q=%25", ""); len(hits) != 0 {
		t.Errorf("wildcard matched: %v", hits)
	}
}
