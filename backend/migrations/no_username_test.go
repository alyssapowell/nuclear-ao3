package migrations

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Migration 051: accounts have no username. The column stays for imported
// logins, but is optional and free-form, and nothing that lists works or
// accounts returns it.
func TestAccountsHaveNoUsername(t *testing.T) {
	db := testDB(t)
	rest := migrateBefore(t, db, "051")

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	// Before: a username is required, and short ones are refused.
	if _, err := db.Exec(`INSERT INTO users (email, password_hash) VALUES ('none@example.org', 'x')`); err == nil {
		t.Fatal("before 051 an account without a username was accepted")
	}
	if _, err := db.Exec(`INSERT INTO users (username, email, password_hash) VALUES ('ab', 'ab@example.org', 'x')`); err == nil {
		t.Fatal("before 051 a two-letter username was accepted")
	}
	legacy := uuid.New()
	exec(`INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'old_login', 'old@example.org', 'x')`, legacy)

	for _, m := range rest {
		if _, err := db.Exec(m.SQL); err != nil {
			t.Fatalf("migration %s: %v", m.Version, err)
		}
	}

	// An account needs no username, and the existing one is untouched.
	fresh := uuid.New()
	exec(`INSERT INTO users (id, email, password_hash) VALUES ($1, 'fresh@example.org', 'x')`, fresh)
	var kept string
	if err := db.QueryRow(`SELECT username FROM users WHERE id = $1`, legacy).Scan(&kept); err != nil || kept != "old_login" {
		t.Errorf("the imported username = %q, %v", kept, err)
	}
	// It is free-form: any length, any characters.
	exec(`INSERT INTO users (email, password_hash, username) VALUES ('x@example.org', 'x', 'a')`)
	exec(`INSERT INTO users (email, password_hash, username) VALUES ('y@example.org', 'x', $1)`, strings.Repeat("é ", 40))
	// It is still unique, so an imported name can't collide with another.
	if _, err := db.Exec(`INSERT INTO users (email, password_hash, username) VALUES ('z@example.org', 'x', 'OLD_LOGIN')`); err == nil {
		t.Error("two accounts have the same username")
	}

	// A new account has no pseud until sign-up makes one; an imported one
	// still gets its default pseud from its username.
	var n int
	db.QueryRow(`SELECT count(*) FROM pseuds WHERE user_id = $1`, fresh).Scan(&n)
	if n != 0 {
		t.Errorf("an account with no username got %d pseuds from nowhere", n)
	}
	exec(`INSERT INTO pseuds (user_id, name, is_default) VALUES ($1, 'Pen', true)`, fresh)
	var imported string
	exec(`INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'legacy_two', 'l2@example.org', 'x')`, uuid.New())
	db.QueryRow(`SELECT p.name FROM pseuds p JOIN users u ON u.id = p.user_id WHERE u.username = 'legacy_two'`).Scan(&imported)
	if imported != "legacy_two" {
		t.Errorf("imported account's default pseud = %q", imported)
	}

	// Nothing in the schema except the column itself mentions a username:
	// no view column and no function result.
	rows, err := db.Query(`SELECT table_name || '.' || column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND column_name ILIKE '%username%' AND table_name <> 'users'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var c string
		rows.Scan(&c)
		t.Errorf("%s still exists", c)
	}
	rows.Close()
	rows, err = db.Query(`SELECT proname FROM pg_proc WHERE pronamespace = 'public'::regnamespace
		AND pg_get_function_result(oid) ILIKE '%username%'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var f string
		rows.Scan(&f)
		t.Errorf("function %s returns a username", f)
	}
	rows.Close()
	rows, err = db.Query(`SELECT * FROM get_work_authors($1)`, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	cols, _ := rows.Columns()
	rows.Close()
	if strings.Join(cols, ",") != "pseud_id,pseud_name,user_id,is_anonymous" {
		t.Errorf("get_work_authors returns %v", cols)
	}
}
