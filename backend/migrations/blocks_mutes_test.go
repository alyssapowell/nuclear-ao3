package migrations

import (
	"testing"

	"github.com/google/uuid"
)

// Migration 041 moves account-level blocks and mutes onto pseuds and gives
// collections an organiser pseud. Old blocks become full blocks.
func TestBlocksMutesAndCollectionsMoveToPseuds(t *testing.T) {
	db := testDB(t)
	rest := migrateBefore(t, db, "041")
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	a, b := uuid.New(), uuid.New()
	exec(`INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'alice', 'a@example.org', 'x'), ($2, 'bob', 'b@example.org', 'x')`, a, b)
	exec(`INSERT INTO pseuds (user_id, name) VALUES ($1, 'Bobby Two')`, b)
	// A comments-only block and a mute, both by account.
	exec(`INSERT INTO user_blocks (blocker_id, blocked_id, block_type, reason) VALUES ($1, $2, 'comments', 'rude')`, a, b)
	exec(`INSERT INTO user_mutes (muter_id, muted_id) VALUES ($1, $2)`, a, b)
	coll := uuid.New()
	exec(`INSERT INTO collections (id, name, title, user_id) VALUES ($1, 'c', 'C', $2)`, coll, a)
	// The mention feature has a table and functions before.
	var n int
	db.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_name = 'mentions'`).Scan(&n)
	if n != 1 {
		t.Fatalf("expected the mentions table before 041")
	}

	for _, m := range rest {
		if _, err := db.Exec(m.SQL); err != nil {
			t.Fatalf("migration %s: %v", m.Version, err)
		}
	}

	var bobDefault, alicePseud uuid.UUID
	db.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1 AND is_default`, b).Scan(&bobDefault)
	db.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1 AND is_default`, a).Scan(&alicePseud)

	var blockPseud uuid.UUID
	var blockName string
	if err := db.QueryRow(`SELECT pseud_id, pseud_name FROM user_blocks WHERE blocker_id = $1`, a).Scan(&blockPseud, &blockName); err != nil {
		t.Fatal(err)
	}
	if blockPseud != bobDefault || blockName != "bob" {
		t.Errorf("block pseud = %s %q, want the default %s %q", blockPseud, blockName, bobDefault, "bob")
	}
	var mutePseud uuid.UUID
	db.QueryRow(`SELECT muted_pseud_id FROM user_mutes WHERE muter_id = $1`, a).Scan(&mutePseud)
	if mutePseud != bobDefault {
		t.Errorf("mute pseud = %s, want the default %s", mutePseud, bobDefault)
	}
	var collPseud uuid.UUID
	db.QueryRow(`SELECT pseud_id FROM collections WHERE id = $1`, coll).Scan(&collPseud)
	if collPseud != alicePseud {
		t.Errorf("collection pseud = %s, want the owner's default %s", collPseud, alicePseud)
	}
	// A new collection with no pseud gets the owner's default.
	exec(`INSERT INTO collections (id, name, title, user_id) VALUES ($1, 'c2', 'C2', $2)`, uuid.New(), a)
	db.QueryRow(`SELECT count(*) FROM collections WHERE user_id = $1 AND pseud_id = $2`, a, alicePseud).Scan(&n)
	if n != 2 {
		t.Errorf("collections under the default pseud = %d, want 2", n)
	}

	// The block is between accounts, in both directions.
	var both bool
	db.QueryRow(`SELECT accounts_blocked($1, $2) AND accounts_blocked($2, $1)`, a, b).Scan(&both)
	if !both {
		t.Errorf("accounts_blocked should hold in both directions")
	}
	// Columns and the mention feature are gone.
	for _, col := range []struct{ table, column string }{{"user_blocks", "block_type"}, {"user_blocks", "reason"}, {"user_mutes", "muted_id"}} {
		db.QueryRow(`SELECT count(*) FROM information_schema.columns WHERE table_name = $1 AND column_name = $2`, col.table, col.column).Scan(&n)
		if n != 0 {
			t.Errorf("%s.%s should be gone", col.table, col.column)
		}
	}
	db.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_name = 'mentions'`).Scan(&n)
	if n != 0 {
		t.Errorf("the mentions table is still there")
	}
	db.QueryRow(`SELECT count(*) FROM pg_proc WHERE proname IN ('create_mentions', 'extract_mentions')`).Scan(&n)
	if n != 0 {
		t.Errorf("the mention functions are still there")
	}
}
