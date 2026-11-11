package moderation_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestAccountLookupAndSessionsAreForModerators(t *testing.T) {
	e := newEnv(t)
	adminID, adminTok := e.user("boss", "admin")
	_, modTok := e.user("mod", "moderator")
	_, plainTok := e.user("plain")
	target, _ := e.user("target", "tag_wrangler")
	e.exec(`UPDATE users SET banned = true WHERE id = $1`, target)
	e.exec(`INSERT INTO user_sessions (user_id, ip_address) VALUES ($1, '127.0.0.1')`, target)
	find := "/api/v1/admin/accounts?q="

	e.expect(401, "GET", find+"target@example.org", "", nil)
	e.expect(403, "GET", find+"target@example.org", plainTok, nil)

	for _, tok := range []string{modTok, adminTok} {
		for _, q := range []string{"target@example.org", "TARGET@Example.org", target.String()} {
			var got struct {
				Account struct {
					ID     uuid.UUID
					Email  string
					Banned bool
					Roles  []string
				}
			}
			if err := json.Unmarshal([]byte(e.expect(200, "GET", find+q, tok, nil)), &got); err != nil {
				t.Fatal(err)
			}
			if got.Account.ID != target || got.Account.Email != "target@example.org" || !got.Account.Banned ||
				len(got.Account.Roles) != 1 || got.Account.Roles[0] != "tag_wrangler" {
				t.Errorf("lookup %q: %+v", q, got.Account)
			}
		}
	}
	e.expect(404, "GET", find+"nobody@example.org", modTok, nil)
	e.expect(400, "GET", find+"", modTok, nil)

	// A lapsed suspension is not reported as a suspension.
	e.exec(`UPDATE users SET banned = false, suspended = true, suspended_until = now() - interval '1 day' WHERE id = $1`, target)
	if out := e.expect(200, "GET", find+target.String(), modTok, nil); !strings.Contains(out, `"suspended":false`) {
		t.Errorf("lapsed suspension shown as current: %s", out)
	}

	// Log out everywhere: refused for a plain user, works for a moderator, audited.
	path := "/api/v1/admin/users/" + target.String() + "/logout-everywhere"
	e.expect(403, "POST", path, plainTok, nil)
	before := e.auditRows()
	e.expect(200, "POST", path, modTok, map[string]string{"reason": "lost phone"})
	var live int
	e.db.QueryRow(`SELECT count(*) FROM user_sessions WHERE user_id = $1 AND is_active`, target).Scan(&live)
	if live != 0 || e.auditRows() != before+1 {
		t.Errorf("live sessions %d, audit rows added %d", live, e.auditRows()-before)
	}
	// A moderator can't do it to staff or to themselves; an admin can do it to a moderator.
	e.expect(403, "POST", "/api/v1/admin/users/"+adminID.String()+"/logout-everywhere", modTok, nil)
	var modID uuid.UUID
	e.db.QueryRow(`SELECT id FROM users WHERE email = 'mod@example.org'`).Scan(&modID)
	e.expect(403, "POST", "/api/v1/admin/users/"+modID.String()+"/logout-everywhere", modTok, nil)
	e.expect(200, "POST", "/api/v1/admin/users/"+modID.String()+"/logout-everywhere", adminTok, nil)
	e.expect(404, "POST", "/api/v1/admin/users/"+uuid.NewString()+"/logout-everywhere", modTok, nil)
}

func TestAbuseReportQueue(t *testing.T) {
	e := newEnv(t)
	_, adminTok := e.user("boss", "admin")
	_, modTok := e.user("mod", "moderator")
	_, plainTok := e.user("plain")
	author, _ := e.user("author")
	work := e.work(author, "Reported work")
	comment := uuid.New()
	e.exec(`INSERT INTO comments (id, work_id, content, guest_name) VALUES ($1, $2, 'rude words here', 'G')`, comment, work)
	wr, cr := uuid.New(), uuid.New()
	e.exec(`INSERT INTO work_reports (id, work_id, reason, description) VALUES ($1, $2, 'spam', 'it is spam')`, wr, work)
	e.exec(`INSERT INTO comment_reports (id, comment_id, reason) VALUES ($1, $2, 'harassment')`, cr, comment)

	e.expect(401, "GET", "/api/v1/admin/abuse-reports", "", nil)
	e.expect(403, "GET", "/api/v1/admin/abuse-reports", plainTok, nil)

	type list struct {
		Reports []struct {
			Kind        string
			TargetTitle string `json:"target_title"`
			ID          uuid.UUID
			WorkID      uuid.UUID `json:"work_id"`
		}
		HasMore bool `json:"has_more"`
	}
	get := func(q string) list {
		var l list
		if err := json.Unmarshal([]byte(e.expect(200, "GET", "/api/v1/admin/abuse-reports"+q, modTok, nil)), &l); err != nil {
			t.Fatal(err)
		}
		return l
	}
	l := get("")
	if len(l.Reports) != 2 {
		t.Fatalf("pending reports: %+v", l)
	}
	if l := get("?kind=comment"); len(l.Reports) != 1 || l.Reports[0].ID != cr || l.Reports[0].WorkID != work ||
		l.Reports[0].TargetTitle != "rude words here" {
		t.Errorf("comment reports: %+v", l)
	}
	if l := get("?limit=1"); len(l.Reports) != 1 || !l.HasMore {
		t.Errorf("paging: %+v", l)
	}
	e.expect(400, "GET", "/api/v1/admin/abuse-reports?status=bogus", modTok, nil)
	e.expect(400, "GET", "/api/v1/admin/abuse-reports?kind=tag", modTok, nil)

	// Closing needs the moderator role, writes one audit row, and is not repeatable.
	path := "/api/v1/admin/abuse-reports/work/" + wr.String()
	e.expect(403, "POST", path+"/resolve", plainTok, nil)
	before := e.auditRows()
	e.expect(200, "POST", path+"/resolve", modTok, map[string]string{"reason": "hid it"})
	if e.auditRows() != before+1 {
		t.Error("closing a report did not write one audit row")
	}
	e.expect(409, "POST", path+"/resolve", modTok, nil)
	e.expect(200, "POST", path+"/dismiss", adminTok, nil)
	e.expect(404, "POST", "/api/v1/admin/abuse-reports/work/"+uuid.NewString()+"/dismiss", modTok, nil)
	e.expect(400, "POST", "/api/v1/admin/abuse-reports/tag/"+cr.String()+"/dismiss", modTok, nil)
	if l := get(""); len(l.Reports) != 1 {
		t.Errorf("pending after closing one: %+v", l)
	}
	if l := get("?status=all"); len(l.Reports) != 2 {
		t.Errorf("all: %+v", l)
	}
}

func TestChapterHideByIdAndAuditFilters(t *testing.T) {
	e := newEnv(t)
	_, adminTok := e.user("boss", "admin")
	_, modTok := e.user("mod", "moderator")
	_, plainTok := e.user("plain")
	author, _ := e.user("author")
	work := e.work(author, "W")
	var chapter uuid.UUID
	e.db.QueryRow(`SELECT id FROM chapters WHERE work_id = $1`, work).Scan(&chapter)

	path := "/api/v1/admin/chapters/" + chapter.String()
	e.expect(403, "POST", path+"/hide", plainTok, map[string]string{"reason": "x"})
	e.expect(200, "POST", path+"/hide", modTok, map[string]string{"reason": "x"})
	e.expect(200, "POST", path+"/unhide", modTok, nil)
	e.expect(200, "POST", "/api/v1/admin/works/"+work.String()+"/hide", modTok, map[string]string{"reason": "y"})

	// The log: admins only; filters by action, actor and date; pages.
	e.expect(403, "GET", "/api/v1/admin/audit", modTok, nil)
	e.expect(403, "GET", "/api/v1/admin/audit", plainTok, nil)
	type page struct {
		Actions []struct{ Action, Actor string }
		HasMore bool `json:"has_more"`
	}
	get := func(q string) page {
		var p page
		if err := json.Unmarshal([]byte(e.expect(200, "GET", "/api/v1/admin/audit"+q, adminTok, nil)), &p); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if p := get(""); len(p.Actions) != 3 || p.HasMore {
		t.Errorf("all: %+v", p)
	}
	if p := get("?action=hide"); len(p.Actions) != 2 {
		t.Errorf("action=hide: %+v", p)
	}
	if p := get("?actor=mod%40example"); len(p.Actions) != 3 {
		t.Errorf("actor by label: %+v", p)
	}
	if p := get("?actor=boss"); len(p.Actions) != 0 {
		t.Errorf("actor boss did nothing: %+v", p)
	}
	if p := get("?actor=%25"); len(p.Actions) != 0 {
		t.Errorf("a percent sign must not match everything: %+v", p)
	}
	if p := get("?limit=2"); len(p.Actions) != 2 || !p.HasMore {
		t.Errorf("first page: %+v", p)
	}
	if p := get("?limit=2&offset=2"); len(p.Actions) != 1 || p.HasMore {
		t.Errorf("second page: %+v", p)
	}
	if p := get("?since=2999-01-01"); len(p.Actions) != 0 {
		t.Errorf("future since: %+v", p)
	}
	if p := get("?until=2000-01-01"); len(p.Actions) != 0 {
		t.Errorf("past until: %+v", p)
	}
	if p := get("?since=2000-01-01&until=2999-01-01"); len(p.Actions) != 3 {
		t.Errorf("wide range: %+v", p)
	}
	e.expect(400, "GET", "/api/v1/admin/audit?until=nope", adminTok, nil)
	e.expect(400, "GET", "/api/v1/admin/audit?offset=-1", adminTok, nil)
}
