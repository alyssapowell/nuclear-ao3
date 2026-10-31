package works_test

import (
	"encoding/json"
	"testing"
)

func (e *fixEnv) getJSON(path, token string, want int) map[string]any {
	e.t.Helper()
	out := e.expect(want, "GET", path, token, nil)
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		e.t.Fatalf("%s: %v: %s", path, err, out)
	}
	return m
}

func TestWorkStatsAndMyStats(t *testing.T) {
	e := newFixEnv(t)
	writer, tok := e.user("writer")
	_, strangerTok := e.user("stranger")
	work := e.work(writer, "A Work", "published")
	draft := e.work(writer, "A Draft", "draft")
	e.exec(`UPDATE works SET hit_count = 50, kudos_count = 10, comment_count = 5, bookmark_count = 3 WHERE id = $1`, work)

	// A published work's counts are public, by id; unknown and unseen ones are 404.
	for _, token := range []string{"", strangerTok, tok} {
		s := e.getJSON("/api/v1/work/"+work.String()+"/stats", token, 200)
		if s["hits"] != 50.0 || s["kudos"] != 10.0 || s["comments"] != 5.0 || s["bookmarks"] != 3.0 {
			t.Errorf("token %q: stats %v", token, s)
		}
	}
	e.expect(404, "GET", "/api/v1/work/"+draft.String()+"/stats", strangerTok, nil)
	e.expect(400, "GET", "/api/v1/work/not-an-id/stats", "", nil)

	// The writer's own totals
	e.expect(401, "GET", "/api/v1/my/stats", "", nil)
	stats := e.getJSON("/api/v1/my/stats", tok, 200)["stats"].(map[string]any)
	if stats["total_works"] != 2.0 || stats["published_works"] != 1.0 || stats["draft_works"] != 1.0 {
		t.Errorf("work counts: %v", stats)
	}
	if stats["total_hits"] != 50.0 || stats["total_kudos"] != 10.0 {
		t.Errorf("engagement totals: %v", stats)
	}
}

func TestAdminStatisticsAreForAdmins(t *testing.T) {
	e := newFixEnv(t)
	_, adminTok := e.user("boss", "admin")
	_, modTok := e.user("mod", "moderator")
	_, userTok := e.user("reader")
	e.expect(401, "GET", "/api/v1/admin/statistics", "", nil)
	e.expect(403, "GET", "/api/v1/admin/statistics", userTok, nil)
	e.expect(403, "GET", "/api/v1/admin/statistics", modTok, nil)
	e.expect(200, "GET", "/api/v1/admin/statistics", adminTok, nil)
}
