package works_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// A pseud's page head and gifts say nothing about the account or any other
// pseud of it. Gifts from the person's own account are left out.
func TestPseudHeadAndGiftsNeverTieSiblingsTogether(t *testing.T) {
	e := newFixEnv(t)
	writer, writerTok := e.user("giftwriter")
	other, otherTok := e.user("giftreader")
	def := e.pseudID(writer, "giftwriter")
	alt := e.newPseud(writer, "Sibling Alt Pen")
	e.exec(`UPDATE pseuds SET description = 'about the default' WHERE id = $1`, def)

	for _, tok := range []string{"", otherTok, writerTok} {
		out := e.expect(200, "GET", "/api/v1/pseuds/"+def.String(), tok, nil)
		noAccount(t, "pseud head", out, writer, other)
		if strings.Contains(out, alt.String()) || strings.Contains(out, "Sibling Alt Pen") {
			t.Errorf("viewer %q: the page head names a sibling pseud: %s", tok, out)
		}
		if !strings.Contains(out, "about the default") {
			t.Errorf("the description is missing: %s", out)
		}
		owned := strings.Contains(out, `"owned":true`)
		if owned != (tok == writerTok) {
			t.Errorf("viewer %q: owned = %v", tok, owned)
		}
	}
	e.expect(404, "GET", "/api/v1/pseuds/"+uuid.New().String(), "", nil)

	// other's default pseud receives a gift from the writer's alt pseud, a
	// gift of an anonymous work, and a work its own account wrote.
	recipient := e.pseudID(other, "giftreader")
	given := e.work(writer, "Given work", "published")
	e.credit(given, alt)
	anon := e.work(writer, "Given anonymously", "published")
	e.exec(`UPDATE works SET is_anonymous = true WHERE id = $1`, anon)
	own := e.work(other, "Self gift", "published")
	for _, w := range []uuid.UUID{given, anon, own} {
		e.exec(`INSERT INTO gifts (work_id, pseud_id) VALUES ($1, $2)`, w, recipient)
	}
	for _, tok := range []string{"", writerTok, otherTok} {
		out := e.expect(200, "GET", "/api/v1/pseuds/"+recipient.String()+"/gifts", tok, nil)
		noAccount(t, "pseud gifts", out, writer, other)
		if !strings.Contains(out, given.String()) || !strings.Contains(out, "Sibling Alt Pen") {
			t.Errorf("viewer %q: the gift is missing: %s", tok, out)
		}
		if strings.Contains(out, own.String()) {
			t.Errorf("viewer %q: a work by the recipient's own account is listed: %s", tok, out)
		}
		// Only the creator themselves may see their own anonymous work's author.
		if tok != writerTok && strings.Contains(out, `"name":"giftwriter"`) {
			t.Errorf("viewer %q: the writer's default pseud appears: %s", tok, out)
		}
	}
}
