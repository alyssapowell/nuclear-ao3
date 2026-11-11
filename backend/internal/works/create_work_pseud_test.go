package works_test

import (
	"encoding/json"
	"testing"
)

// A work can be posted under one of the writer's own pseuds, never anyone else's.
func TestCreateWorkUnderChosenPseud(t *testing.T) {
	e := newFixEnv(t)
	writer, tok := e.user("choosy")
	other, _ := e.user("someoneelse")
	pen := e.newPseud(writer, "Chosen Pen")
	theirs := e.newPseud(other, "Not Yours")

	body := newWorkBody()
	body["pseud_id"] = pen.String()
	out := e.expect(201, "POST", "/api/v1/works", tok, body)
	var got struct{ Work struct{ ID string } }
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if n := e.count(`SELECT count(*) FROM creatorships WHERE creation_id = $1 AND pseud_id = $2`, got.Work.ID, pen); n != 1 {
		t.Error("the work is not credited to the chosen pseud")
	}
	if n := e.count(`SELECT count(*) FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id
		WHERE c.creation_id = $1 AND p.is_default`, got.Work.ID); n != 0 {
		t.Error("the work is also credited to the default pseud")
	}

	// Someone else's pseud and a pseud that doesn't exist get the same answer.
	body["pseud_id"] = theirs.String()
	a := e.expect(403, "POST", "/api/v1/works", tok, body)
	body["pseud_id"] = "00000000-0000-0000-0000-000000000001"
	b := e.expect(403, "POST", "/api/v1/works", tok, body)
	if a != b {
		t.Errorf("a pseud that is not yours must look like one that doesn't exist: %s vs %s", a, b)
	}
}
