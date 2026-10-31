package works_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// A creator of a work that hides its creators replies to a reader. The
// reader's notification must not carry the pseud, the username or any account
// id, and must say "Anonymous Creator".
func TestReplyNotificationKeepsHiddenCreatorHidden(t *testing.T) {
	e := newFixEnv(t)
	writer, writerTok := e.user("secretwriter")
	reader, readerTok := e.user("reader")
	moon := uuid.New()
	e.exec(`INSERT INTO pseuds (id, user_id, name) VALUES ($1, $2, 'Moonlit Quill')`, moon, writer)

	for _, set := range []string{"is_anonymous", "in_anon_collection", "in_unrevealed_collection"} {
		e.exec(`DELETE FROM notifications`)
		e.exec(`UPDATE works SET is_anonymous = false, in_anon_collection = false, in_unrevealed_collection = false`)
		work := e.work(writer, "Secret "+set, "published")
		e.exec(`UPDATE creatorships SET pseud_id = $1 WHERE creation_id = $2`, moon, work)
		e.exec(`UPDATE works SET `+set+` = true WHERE id = $1`, work)
		readerComment := e.comment(work, reader, "lovely")
		e.exec(`DELETE FROM notifications`)

		e.expect(201, "POST", "/api/v1/works/"+work.String()+"/comments", writerTok, map[string]any{
			"content": "thank you", "parent_comment_id": readerComment, "pseudonym_id": moon})

		var title, message, data string
		if err := e.db.QueryRow(`SELECT title, message, data::text FROM notifications WHERE user_id = $1`, reader).
			Scan(&title, &message, &data); err != nil {
			t.Fatalf("%s: reader's notification: %v", set, err)
		}
		all := title + message + data
		for _, secret := range []string{"Moonlit Quill", "secretwriter", writer.String(), moon.String(), "commenter_id"} {
			if strings.Contains(all, secret) {
				t.Errorf("%s: notification gives away %q: %s | %s | %s", set, secret, title, message, data)
			}
		}
		if !strings.Contains(message, "Anonymous Creator") || !strings.Contains(data, `"commenter_name": "Anonymous Creator"`) {
			t.Errorf("%s: notification should say Anonymous Creator: %s | %s", set, message, data)
		}
	}

	// A work that hides nothing still names the pseud, and never the username.
	e.exec(`DELETE FROM notifications`)
	e.exec(`UPDATE works SET is_anonymous = false, in_anon_collection = false, in_unrevealed_collection = false`)
	open := e.work(writer, "Open", "published")
	e.exec(`UPDATE creatorships SET pseud_id = $1 WHERE creation_id = $2`, moon, open)
	rc := e.comment(open, reader, "hello")
	e.exec(`DELETE FROM notifications`)
	e.expect(201, "POST", "/api/v1/works/"+open.String()+"/comments", writerTok, map[string]any{
		"content": "hi", "parent_comment_id": rc, "pseudonym_id": moon})
	var message, data string
	e.db.QueryRow(`SELECT message, data::text FROM notifications WHERE user_id = $1`, reader).Scan(&message, &data)
	if !strings.Contains(message, "Moonlit Quill") || strings.Contains(message+data, "secretwriter") || strings.Contains(data, writer.String()) {
		t.Errorf("open work: %s | %s", message, data)
	}

	// A mention in a hidden creator's reply carries no account id either.
	e.exec(`DELETE FROM notifications`)
	hidden := e.work(writer, "Hidden", "published")
	e.exec(`UPDATE works SET is_anonymous = true WHERE id = $1`, hidden)
	e.exec(`INSERT INTO comments (work_id, user_id, pseudonym_id, content) VALUES ($1, $2, $3, 'thanks @reader')`, hidden, writer, moon)
	var mdata string
	if err := e.db.QueryRow(`SELECT data::text FROM notifications WHERE user_id = $1 AND type = 'comment_mention'`, reader).Scan(&mdata); err != nil {
		t.Fatalf("mention notification: %v", err)
	}
	if strings.Contains(mdata, writer.String()) || strings.Contains(mdata, "mentioning") {
		t.Errorf("mention notification names the author: %s", mdata)
	}

	// A guest is called by the guest name, then "Someone", never a username.
	e.exec(`DELETE FROM notifications`)
	e.exec(`INSERT INTO comments (work_id, guest_name, content) VALUES ($1, 'Visitor', 'hey')`, open)
	e.db.QueryRow(`SELECT message FROM notifications WHERE user_id = $1`, writer).Scan(&message)
	if !strings.Contains(message, "Visitor") {
		t.Errorf("guest notification: %q", message)
	}
	_ = readerTok
}

// "Anonymous Creator" is the label for a hidden creator, so no pseud may have
// it, and a comment by one that got it anyway does not pass as a hidden creator.
func TestAnonymousCreatorNameIsReserved(t *testing.T) {
	e := newFixEnv(t)
	writer, tok := e.user("writer")
	_, otherTok := e.user("other")

	for _, name := range []string{"Anonymous Creator", "anonymous creator", "ANONYMOUS  CREATOR", " AnonymousCreator", "Anonymous\tCreator"} {
		if code, out := e.do("POST", "/api/v1/pseuds", tok, map[string]any{"name": name}); code != 422 {
			t.Errorf("creating %q: status %d, want 422: %s", name, code, out)
		}
	}
	out := e.expect(201, "POST", "/api/v1/pseuds", tok, map[string]any{"name": "Fine Name"})
	var made struct{ Pseud struct{ ID string } }
	json.Unmarshal([]byte(out), &made)
	e.expect(422, "PUT", "/api/v1/pseuds/"+made.Pseud.ID, tok, map[string]any{"name": "anonymous creator"})

	// The database refuses it too, whatever the API does.
	if _, err := e.db.Exec(`INSERT INTO pseuds (user_id, name) VALUES ($1, 'Anonymous   Creator')`, writer); err == nil {
		t.Errorf("the database accepted a pseud named Anonymous Creator")
	}
	// An account whose username would give that default pseud gets another name.
	var def string
	e.exec(`INSERT INTO users (username, email, password_hash) VALUES ('AnonymousCreator', 'ac@example.org', 'x')`)
	e.db.QueryRow(`SELECT p.name FROM pseuds p JOIN users u ON u.id = p.user_id WHERE u.username = 'AnonymousCreator'`).Scan(&def)
	if def == "" || strings.EqualFold(strings.ReplaceAll(def, " ", ""), "anonymouscreator") {
		t.Errorf("default pseud of an account called AnonymousCreator = %q", def)
	}

	// Force the name in SQL (dropping the constraint) on a work that hides nothing.
	e.exec(`ALTER TABLE pseuds DROP CONSTRAINT pseud_name_not_reserved`)
	e.exec(`UPDATE pseuds SET name = 'Anonymous Creator' WHERE id = $1`, made.Pseud.ID)
	work := e.work(writer, "Open Work", "published")
	e.exec(`INSERT INTO comments (work_id, user_id, pseudonym_id, content) VALUES ($1, $2, $3, 'I am not hidden')`, work, writer, made.Pseud.ID)
	for _, path := range []string{"/api/v1/works/" + work.String() + "/comments", "/api/v1/works/" + work.String()} {
		body := e.expect(200, "GET", path, otherTok, nil)
		if strings.Contains(body, "I am not hidden") && strings.Contains(body, `"anonymous_creator":true`) {
			t.Errorf("%s: a pseud named Anonymous Creator passed as a hidden creator: %s", path, body)
		}
	}
	posted := e.expect(201, "POST", "/api/v1/works/"+work.String()+"/comments", tok, map[string]any{
		"content": "again", "pseudonym_id": made.Pseud.ID})
	if strings.Contains(posted, `"anonymous_creator":true`) {
		t.Errorf("created comment passed as a hidden creator: %s", posted)
	}
}
