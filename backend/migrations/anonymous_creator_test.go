package migrations

import (
	"testing"

	"github.com/google/uuid"
)

// Pseuds already named "Anonymous Creator" are renamed by migration 039, so
// the new rule can be added without breaking an existing database.
func TestReservedPseudNameRenamed(t *testing.T) {
	db := testDB(t)
	rest := migrateBefore(t, db, "039")

	user := uuid.New()
	if _, err := db.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'someone', 'someone@example.org', 'x')`, user); err != nil {
		t.Fatal(err)
	}
	ids := map[string]uuid.UUID{}
	for _, name := range []string{"Anonymous Creator", "anonymous  creator", "Fine"} {
		id := uuid.New()
		ids[name] = id
		if _, err := db.Exec(`INSERT INTO pseuds (id, user_id, name, is_default) VALUES ($1, $2, $3, false)`, id, user, name); err != nil {
			t.Fatal(err)
		}
	}
	for _, m := range rest {
		if _, err := db.Exec(m.SQL); err != nil {
			t.Fatalf("migration %s: %v", m.Version, err)
		}
	}
	for name, id := range ids {
		var now string
		if err := db.QueryRow(`SELECT name FROM pseuds WHERE id = $1`, id).Scan(&now); err != nil {
			t.Fatal(err)
		}
		if (name == "Fine") != (now == "Fine") {
			t.Errorf("%q became %q", name, now)
		}
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM pseuds WHERE lower(regexp_replace(name, '[[:space:]]', '', 'g')) = 'anonymouscreator'`).Scan(&n)
	if n != 0 {
		t.Errorf("%d pseuds still have the reserved name", n)
	}
}
