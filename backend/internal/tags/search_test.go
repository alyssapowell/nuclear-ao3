package tags

import (
	"strings"
	"testing"
)

func tagNames(out map[string]any) []string {
	var names []string
	for _, t := range out["tags"].([]any) {
		names = append(names, t.(map[string]any)["name"].(string))
	}
	return names
}

func TestTagSearchFiltersSortsAndPages(t *testing.T) {
	e := newTagEnv(t)
	for _, tg := range []struct {
		name, typ string
		canon     bool
		uses      int
	}{
		{"Moon", "freeform", true, 5},
		{"Moonlight Walk", "freeform", true, 50},
		{"Blue Moon", "freeform", false, 500},
		{"Moon Hunter", "character", true, 1},
		{"100% Moon_Fic", "freeform", true, 3},
		{"Sun", "freeform", true, 9},
	} {
		if _, err := e.db.Exec(`INSERT INTO tags (name, type, is_canonical, canonical_name, use_count)
			VALUES ($1, $2, $3, CASE WHEN $3 THEN NULL ELSE 'Moon' END, $4)`, tg.name, tg.typ, tg.canon, tg.uses); err != nil {
			t.Fatal(err)
		}
	}
	get := func(q string) map[string]any { return e.expect(200, "GET", "/api/v1/tags/search?"+q, "", nil) }

	// Relevance: exact name, then names that start with it, then the rest; busier first within a group.
	if got := strings.Join(tagNames(get("q=moon")), ","); got != "Moon,Moonlight Walk,Moon Hunter,Blue Moon,100% Moon_Fic" {
		t.Errorf("relevance order: %s", got)
	}
	if got := strings.Join(tagNames(get("q=moon&sort=uses")), ","); got != "Blue Moon,Moonlight Walk,Moon,100% Moon_Fic,Moon Hunter" {
		t.Errorf("by uses: %s", got)
	}
	if got := strings.Join(tagNames(get("q=moon&sort=name&canonical=true")), ","); got != "100% Moon_Fic,Moon,Moon Hunter,Moonlight Walk" {
		t.Errorf("by name, canonical only: %s", got)
	}
	if got := tagNames(get("q=moon&type=character")); len(got) != 1 || got[0] != "Moon Hunter" {
		t.Errorf("by type: %v", got)
	}
	// % and _ are ordinary characters in a search.
	if got := tagNames(get("q=%25+moon_")); len(got) != 1 || got[0] != "100% Moon_Fic" {
		t.Errorf("wildcards leaked or literal match failed: %v", got)
	}
	if got := tagNames(get("q=_")); len(got) != 1 {
		t.Errorf("underscore matched everything: %v", got)
	}

	p1 := get("q=moon&sort=name&limit=2&page=1")
	p2 := get("q=moon&sort=name&limit=2&page=2")
	if p1["total"].(float64) != 5 || p1["pages"].(float64) != 3 || p2["page"].(float64) != 2 {
		t.Errorf("pagination: %v %v", p1, p2)
	}
	if strings.Join(append(tagNames(p1), tagNames(p2)...), ",") != "100% Moon_Fic,Blue Moon,Moon,Moon Hunter" {
		t.Errorf("pages: %v %v", tagNames(p1), tagNames(p2))
	}
	// Each result carries what the page shows.
	first := get("q=blue")["tags"].([]any)[0].(map[string]any)
	for _, k := range []string{"id", "name", "type", "is_canonical", "use_count"} {
		if _, ok := first[k]; !ok {
			t.Errorf("result has no %q: %v", k, first)
		}
	}
}
