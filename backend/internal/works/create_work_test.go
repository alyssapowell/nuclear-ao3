package works_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func newWorkBody() map[string]any {
	return map[string]any{
		"title": "Posted Today", "summary": "<p>hi</p><script>x()</script>", "language": "en", "rating": "general",
		"fandoms": []string{"Brand New Fandom"}, "freeform_tags": []string{"Fluff"},
		"chapter_title": "One", "chapter_content": "Some words in the first chapter.",
	}
}

func TestCreateWorkMakesADraftWithItsFirstChapter(t *testing.T) {
	e := newFixEnv(t)
	writer, tok := e.user("writer")

	e.expect(401, "POST", "/api/v1/works", "", newWorkBody())
	out := e.expect(201, "POST", "/api/v1/works", tok, newWorkBody())
	var got struct {
		Work struct {
			ID string `json:"id"`
		} `json:"work"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if n := e.count(`SELECT count(*) FROM works WHERE id = $1 AND is_draft AND user_id = $2`, got.Work.ID, writer); n != 1 {
		t.Error("work is not a draft of the writer's")
	}
	if n := e.count(`SELECT count(*) FROM chapters WHERE work_id = $1 AND chapter_number = 1`, got.Work.ID); n != 1 {
		t.Error("no first chapter")
	}
	// Credited to the writer's default pseud
	if n := e.count(`SELECT count(*) FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id
		WHERE c.creation_id = $1 AND p.user_id = $2 AND p.is_default`, got.Work.ID, writer); n != 1 {
		t.Error("work not credited to the default pseud")
	}
	// A fandom nobody has used yet is created, unwrangled (DECISIONS 2026-10-29)
	if n := e.count(`SELECT count(*) FROM tags WHERE name = 'Brand New Fandom' AND type = 'fandom'`); n != 1 {
		t.Error("new fandom was not created")
	}
	var summary string
	e.db.QueryRow(`SELECT summary FROM works WHERE id = $1`, got.Work.ID).Scan(&summary)
	if strings.Contains(summary, "<script") || !strings.Contains(summary, "hi") {
		t.Errorf("summary stored uncleaned: %q", summary)
	}
}

func TestCreateWorkRefusesIncompleteWorks(t *testing.T) {
	e := newFixEnv(t)
	_, tok := e.user("writer")
	for name, drop := range map[string]string{
		"title": "title", "language": "language", "rating": "rating",
		"fandoms": "fandoms", "text": "chapter_content",
	} {
		body := newWorkBody()
		delete(body, drop)
		if code, out := e.do("POST", "/api/v1/works", tok, body); code != 400 {
			t.Errorf("missing %s: %d %s", name, code, out)
		}
	}
	for name, change := range map[string][2]any{
		"bad rating": {"rating", "Invalid"}, "bad language": {"language", "invalid"},
		"blank title": {"title", "   "}, "blank fandom": {"fandoms", []string{" "}},
	} {
		body := newWorkBody()
		body[change[0].(string)] = change[1]
		if code, out := e.do("POST", "/api/v1/works", tok, body); code != 400 {
			t.Errorf("%s: %d %s", name, code, out)
		}
	}
	if n := e.count(`SELECT count(*) FROM works`); n != 0 {
		t.Errorf("%d works were created from bad requests", n)
	}
}
