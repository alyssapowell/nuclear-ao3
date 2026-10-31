package migrations

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// migrateBefore applies every migration older than version, directly, so a
// test can put rows in the old schema and then run the migration under test.
func migrateBefore(t *testing.T, db *sql.DB, version string) []Migration {
	t.Helper()
	all, err := All()
	if err != nil {
		t.Fatal(err)
	}
	var rest []Migration
	for _, m := range all {
		if m.Version >= version {
			rest = append(rest, m)
			continue
		}
		if _, err := db.Exec(m.SQL); err != nil {
			t.Fatalf("migration %s: %v", m.Version, err)
		}
	}
	return rest
}

// TestOnePseudTable seeds both old pseud tables, runs migration 037, and
// checks that the rows merged, the comments followed their pseuds, and the
// AO3 rules hold afterwards.
func TestOnePseudTable(t *testing.T) {
	db := testDB(t)
	rest := migrateBefore(t, db, "037")

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	user := func(name string) uuid.UUID {
		id := uuid.New()
		exec(`INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, 'x')`,
			id, name, name+"@example.org")
		return id
	}
	// The old schema has no trigger, so users have no pseud unless a test adds one.
	alice, bob, cee, dee, eve := user("alice"), user("bob"), user("cee"), user("dee"), user("eve")

	// alice: in both tables under the same name (different case), plus a pen
	// name only comments knew about.
	alicePseud, aliceOld, aliceOldPen := uuid.New(), uuid.New(), uuid.New()
	exec(`INSERT INTO pseuds (id, user_id, name, is_default) VALUES ($1, $2, 'alice', true)`, alicePseud, alice)
	exec(`INSERT INTO user_pseudonyms (id, user_id, name, is_default, description, icon_url)
		VALUES ($1, $2, 'Alice', true, 'about alice', 'https://example.org/a.png')`, aliceOld, alice)
	exec(`INSERT INTO user_pseudonyms (id, user_id, name, is_default) VALUES ($1, $2, 'Pen Name', false)`, aliceOldPen, alice)
	// bob: only in user_pseudonyms, with a name too long for a pseud.
	bobOld, bobLong := uuid.New(), uuid.New()
	exec(`INSERT INTO user_pseudonyms (id, user_id, name, is_default) VALUES ($1, $2, 'bob', true)`, bobOld, bob)
	exec(`INSERT INTO user_pseudonyms (id, user_id, name) VALUES ($1, $2, $3)`, bobLong, bob, strings.Repeat("b", 50))
	// cee: only in pseuds, with two names that differ only by case, no default.
	ceeA, ceeB := uuid.New(), uuid.New()
	exec(`INSERT INTO pseuds (id, user_id, name, created_at) VALUES ($1, $2, 'Cee', now() - interval '2 days')`, ceeA, cee)
	exec(`INSERT INTO pseuds (id, user_id, name, created_at) VALUES ($1, $2, 'cee', now() - interval '1 day')`, ceeB, cee)
	// dee: pseuds says one default, user_pseudonyms says another name is.
	deePseud, deeOld := uuid.New(), uuid.New()
	exec(`INSERT INTO pseuds (id, user_id, name, is_default) VALUES ($1, $2, 'dee', true)`, deePseud, dee)
	exec(`INSERT INTO user_pseudonyms (id, user_id, name, is_default) VALUES ($1, $2, 'D', true)`, deeOld, dee)
	// eve: no pseud anywhere.

	work := uuid.New()
	exec(`INSERT INTO works (id, user_id, title, language, rating, status, is_draft)
		VALUES ($1, $2, 'W', 'en', 'general', 'published', false)`, work, alice)
	exec(`INSERT INTO creatorships (creation_id, creation_type, pseud_id) VALUES ($1, 'Work', $2)`, work, alicePseud)
	comment := func(user, pseud uuid.UUID) uuid.UUID {
		id := uuid.New()
		exec(`INSERT INTO comments (id, work_id, user_id, pseudonym_id, content) VALUES ($1, $2, $3, $4, 'hi')`,
			id, work, user, pseud)
		return id
	}
	onAlice, onPen, onBobLong, onDee := comment(alice, aliceOld), comment(alice, aliceOldPen), comment(bob, bobLong), comment(dee, deeOld)
	kudos := uuid.New()
	exec(`INSERT INTO comment_kudos (id, comment_id, user_id, pseudonym_id) VALUES ($1, $2, $3, $4)`, kudos, onAlice, bob, bobOld)

	// Run the migration.
	for _, m := range rest {
		if m.Version != "037_one_pseud_table" {
			continue
		}
		if _, err := db.Exec(m.SQL); err != nil {
			t.Fatalf("037: %v", err)
		}
	}

	var n int
	if err := db.QueryRow(`SELECT count(*) FROM pg_class WHERE relname = 'user_pseudonyms'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("user_pseudonyms still exists (n=%d, err=%v)", n, err)
	}
	pseudOf := func(comment uuid.UUID) (id uuid.UUID, name string) {
		t.Helper()
		if err := db.QueryRow(`SELECT p.id, p.name FROM comments c JOIN pseuds p ON p.id = c.pseudonym_id WHERE c.id = $1`,
			comment).Scan(&id, &name); err != nil {
			t.Fatalf("comment %s lost its pseud: %v", comment, err)
		}
		return
	}

	// alice's two rows are one pseud; comments moved to it; description kept.
	if id, name := pseudOf(onAlice); id != alicePseud || name != "alice" {
		t.Errorf("alice's comment is on %s %q, want the existing pseud", id, name)
	}
	var desc, icon sql.NullString
	db.QueryRow(`SELECT description, icon_url FROM pseuds WHERE id = $1`, alicePseud).Scan(&desc, &icon)
	if desc.String != "about alice" || icon.String != "https://example.org/a.png" {
		t.Errorf("alice's description/icon not carried over: %q %q", desc.String, icon.String)
	}
	// The pen name had no pseud, so it became one and kept its id.
	if id, name := pseudOf(onPen); id != aliceOldPen || name != "Pen Name" {
		t.Errorf("pen name comment is on %s %q, want the old id kept", id, name)
	}
	// bob's long name was shortened to fit and kept its id.
	if id, name := pseudOf(onBobLong); id != bobLong || len([]rune(name)) > 40 || name == "" {
		t.Errorf("long name comment is on %s %q", id, name)
	}
	// dee's "D" was a different name from her pseud, so it became its own pseud.
	if id, name := pseudOf(onDee); id != deeOld || name != "D" {
		t.Errorf("dee's comment is on %s %q, want the old id kept", id, name)
	}
	var kudosPseud uuid.UUID
	db.QueryRow(`SELECT pseudonym_id FROM comment_kudos WHERE id = $1`, kudos).Scan(&kudosPseud)
	if kudosPseud != bobOld {
		t.Errorf("comment kudos moved to %s, want %s", kudosPseud, bobOld)
	}
	// Authorship is untouched.
	db.QueryRow(`SELECT count(*) FROM creatorships WHERE creation_id = $1 AND pseud_id = $2`, work, alicePseud).Scan(&n)
	if n != 1 {
		t.Errorf("creatorship lost")
	}

	// Case-only duplicates were separated, keeping the older name.
	var cleanName string
	db.QueryRow(`SELECT name FROM pseuds WHERE id = $1`, ceeA).Scan(&cleanName)
	if cleanName != "Cee" {
		t.Errorf("older of the case duplicates was renamed to %q", cleanName)
	}
	db.QueryRow(`SELECT count(*) FROM pseuds WHERE user_id = $1`, cee).Scan(&n)
	if n != 2 {
		t.Errorf("cee has %d pseuds, want 2", n)
	}

	// Exactly one default for every user, including eve, who had none.
	rows, err := db.Query(`SELECT u.username, count(p.id) FILTER (WHERE p.is_default)
		FROM users u LEFT JOIN pseuds p ON p.user_id = u.id GROUP BY u.id, u.username`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var defaults int
		rows.Scan(&name, &defaults)
		if defaults != 1 {
			t.Errorf("%s has %d default pseuds, want 1", name, defaults)
		}
	}
	var defName string
	db.QueryRow(`SELECT name FROM pseuds WHERE user_id = $1 AND is_default`, dee).Scan(&defName)
	if defName != "dee" {
		t.Errorf("dee's default is %q; the pseud that authorship used should win", defName)
	}
	db.QueryRow(`SELECT name FROM pseuds WHERE user_id = $1 AND is_default`, eve).Scan(&defName)
	if defName != "eve" {
		t.Errorf("eve's new default is %q, want her username", defName)
	}

	// The rules now in force.
	reject := func(what, q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err == nil {
			t.Errorf("accepted %s", what)
		}
	}
	reject("a second default", `INSERT INTO pseuds (user_id, name, is_default) VALUES ($1, 'second', true)`, alice)
	reject("a name differing only by case", `INSERT INTO pseuds (user_id, name) VALUES ($1, 'PEN NAME')`, alice)
	reject("an empty name", `INSERT INTO pseuds (user_id, name) VALUES ($1, '')`, alice)
	reject("a blank name", `INSERT INTO pseuds (user_id, name) VALUES ($1, '  ')`, alice)
	reject("41 characters", `INSERT INTO pseuds (user_id, name) VALUES ($1, $2)`, alice, strings.Repeat("x", 41))
	reject("a comment on a missing pseud", `UPDATE comments SET pseudonym_id = $1 WHERE id = $2`, uuid.New(), onAlice)
	for _, ok := range []string{"Amélie Poulain", "李小龍", "Ünal-Çağlar", "ว่าย น้ำ", strings.Repeat("é", 40)} {
		exec(`INSERT INTO pseuds (user_id, name) VALUES ($1, $2)`, bob, ok)
	}
	// The same name is fine for a different user.
	exec(`INSERT INTO pseuds (user_id, name) VALUES ($1, 'Pen Name')`, bob)

	// A new account gets its default pseud with it.
	fresh := user("newcomer")
	var newName string
	var newDefault bool
	if err := db.QueryRow(`SELECT name, is_default FROM pseuds WHERE user_id = $1`, fresh).Scan(&newName, &newDefault); err != nil ||
		newName != "newcomer" || !newDefault {
		t.Errorf("new account's pseud = %q default=%v err=%v", newName, newDefault, err)
	}
}

// TestOnePseudTableOnEmptyDatabase checks the migration on a database with no
// pseuds or users but the seeded orphan account.
func TestOnePseudTableOnEmptyDatabase(t *testing.T) {
	db := testDB(t)
	if _, err := Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var defaults int
	if err := db.QueryRow(`SELECT count(*) FROM pseuds WHERE is_default AND name = 'orphan_account'`).Scan(&defaults); err != nil || defaults != 1 {
		t.Fatalf("orphan_account default pseud count = %d, err %v", defaults, err)
	}
}
