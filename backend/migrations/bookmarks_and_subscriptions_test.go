package migrations

import (
	"testing"

	"github.com/google/uuid"
)

// Migration 040 moves bookmarks and author subscriptions from account ids to
// pseuds: bookmarks go under their owner's default pseud, and a subscription
// to an author now points at that author's default pseud.
func TestBookmarksAndSubscriptionsMoveToPseuds(t *testing.T) {
	db := testDB(t)
	rest := migrateBefore(t, db, "040")
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	author, fan := uuid.New(), uuid.New()
	exec(`INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'author', 'a@example.org', 'x'), ($2, 'fan', 'f@example.org', 'x')`, author, fan)
	var defaultPseud uuid.UUID
	if err := db.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1 AND is_default`, author).Scan(&defaultPseud); err != nil {
		t.Fatal(err)
	}
	work := uuid.New()
	exec(`INSERT INTO works (id, user_id, title, summary, language, rating) VALUES ($1, $2, 'w', 's', 'en', 'general')`, work, author)
	exec(`INSERT INTO bookmarks (work_id, user_id) VALUES ($1, $2)`, work, fan)
	exec(`INSERT INTO subscriptions (user_id, type, target_id, target_name) VALUES ($1, 'author', $2, 'author')`, fan, author)
	exec(`INSERT INTO content_subscriptions (id, user_id, target_type, target_id) VALUES ($1, $2, 'author', $3)`, uuid.New(), fan, author)
	// A subscription to a work is not touched.
	exec(`INSERT INTO subscriptions (user_id, type, target_id, target_name) VALUES ($1, 'work', $2, 'w')`, fan, work)

	for _, m := range rest {
		if _, err := db.Exec(m.SQL); err != nil {
			t.Fatalf("migration %s: %v", m.Version, err)
		}
	}

	var fanDefault, bookmarkPseud, subTarget uuid.UUID
	var subName string
	db.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1 AND is_default`, fan).Scan(&fanDefault)
	db.QueryRow(`SELECT pseud_id FROM bookmarks`).Scan(&bookmarkPseud)
	if bookmarkPseud != fanDefault {
		t.Errorf("bookmark pseud = %s, want the owner's default %s", bookmarkPseud, fanDefault)
	}
	db.QueryRow(`SELECT target_id, target_name FROM subscriptions WHERE type = 'author'`).Scan(&subTarget, &subName)
	if subTarget != defaultPseud || subName != "author" {
		t.Errorf("author subscription = %s %q, want %s", subTarget, subName, defaultPseud)
	}
	db.QueryRow(`SELECT target_id FROM content_subscriptions WHERE target_type = 'author'`).Scan(&subTarget)
	if subTarget != defaultPseud {
		t.Errorf("content subscription target = %s, want %s", subTarget, defaultPseud)
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM subscriptions WHERE type = 'work' AND target_id = $1`, work).Scan(&n)
	if n != 1 {
		t.Errorf("the work subscription changed")
	}
	// A bookmark made without a pseud goes under the default one.
	exec(`INSERT INTO works (id, user_id, title, summary, language, rating) VALUES ($1, $2, 'w2', 's', 'en', 'general')`, uuid.New(), author)
	var w2 uuid.UUID
	db.QueryRow(`SELECT id FROM works WHERE title = 'w2'`).Scan(&w2)
	exec(`INSERT INTO bookmarks (work_id, user_id) VALUES ($1, $2)`, w2, fan)
	db.QueryRow(`SELECT pseud_id FROM bookmarks WHERE work_id = $1`, w2).Scan(&bookmarkPseud)
	if bookmarkPseud != fanDefault {
		t.Errorf("new bookmark pseud = %s, want %s", bookmarkPseud, fanDefault)
	}
}
