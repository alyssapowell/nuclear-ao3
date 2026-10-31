package html

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"nuclear-ao3/internal/testdb"
)

const dirty = `<p>hi</p><img src=x onerror=alert(1)><script>alert(2)</script>`

func TestMigrationQueuesTheCleanupJob(t *testing.T) {
	db := testdb.New(t)
	var n int
	db.QueryRow(`SELECT count(*) FROM jobs WHERE kind = 'clean-html' AND done_at IS NULL`).Scan(&n)
	if n != 1 {
		t.Errorf("%d clean-html jobs queued, want 1", n)
	}
}

func TestCleanStoredCleansOldRowsWithoutTouchingUpdatedAt(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%v: %s", err, q)
		}
	}

	userID, workID, chapterID, commentID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec(`INSERT INTO users (id, email, username, password_hash, is_verified, bio) VALUES ($1, 'a@example.org', 'author', 'x', true, $2)`, userID, dirty)
	exec(`INSERT INTO works (id, user_id, title, summary, notes, language, rating, status, is_draft, published_at)
		VALUES ($1, $2, 'T', $3, $3, 'en', 'general', 'published', false, now())`, workID, userID, dirty)
	exec(`INSERT INTO chapters (id, work_id, chapter_number, title, summary, notes, end_notes, content, is_draft)
		VALUES ($1, $2, 1, 'C', $3, $3, $3, $3, false)`, chapterID, workID, dirty)
	// The account came with a default pseud named after it.
	var pseudID uuid.UUID
	if err := db.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1 AND is_default`, userID).Scan(&pseudID); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO comments (id, work_id, user_id, pseudonym_id, content) VALUES ($1, $2, $3, $4, $5)`, commentID, workID, userID, pseudID, dirty)

	// Put every updated_at far in the past, so any later bump is visible.
	// The triggers would reset them, so do it the way the job does.
	past := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	tx.Exec(`SELECT set_config('nuclear.maintenance', 'on', true)`)
	for _, table := range []string{"works", "chapters", "comments", "users"} {
		if _, err := tx.Exec(`UPDATE `+table+` SET updated_at = $1`, past); err != nil {
			t.Fatal(err)
		}
	}
	tx.Commit()
	var statsBefore time.Time
	db.QueryRow(`SELECT updated_at FROM user_statistics WHERE user_id = $1`, userID).Scan(&statsBefore)
	var notificationsBefore int
	db.QueryRow(`SELECT count(*) FROM notifications`).Scan(&notificationsBefore)

	changed, err := CleanStored(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if changed != 4 {
		t.Errorf("rewrote %d rows, want 4 (work, chapter, comment, user)", changed)
	}

	get := func(q string, args ...any) string {
		t.Helper()
		var s string
		if err := db.QueryRow(q, args...).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	for name, v := range map[string]string{
		"work summary":    get(`SELECT summary FROM works WHERE id = $1`, workID),
		"work notes":      get(`SELECT notes FROM works WHERE id = $1`, workID),
		"chapter content": get(`SELECT content FROM chapters WHERE id = $1`, chapterID),
		"chapter notes":   get(`SELECT end_notes FROM chapters WHERE id = $1`, chapterID),
		"comment":         get(`SELECT content FROM comments WHERE id = $1`, commentID),
		"bio":             get(`SELECT bio FROM users WHERE id = $1`, userID),
	} {
		if v != "<p>hi</p>" {
			t.Errorf("%s = %q, want the cleaned text", name, v)
		}
	}
	if p := get(`SELECT processed_content FROM chapters WHERE id = $1`, chapterID); strings.Contains(p, "onerror") {
		t.Errorf("processed_content still dirty: %s", p)
	}

	stamp := func(table string, id uuid.UUID) time.Time {
		var ts time.Time
		db.QueryRow(`SELECT updated_at FROM `+table+` WHERE id = $1`, id).Scan(&ts)
		return ts
	}
	for table, id := range map[string]uuid.UUID{"works": workID, "chapters": chapterID, "comments": commentID, "users": userID} {
		if ts := stamp(table, id); !ts.Equal(past) {
			t.Errorf("%s.updated_at changed from %v to %v", table, past, ts)
		}
	}
	var statsAfter time.Time
	db.QueryRow(`SELECT updated_at FROM user_statistics WHERE user_id = $1`, userID).Scan(&statsAfter)
	if !statsAfter.Equal(statsBefore) {
		t.Errorf("user_statistics.updated_at changed from %v to %v", statsBefore, statsAfter)
	}
	var notificationsAfter int
	db.QueryRow(`SELECT count(*) FROM notifications`).Scan(&notificationsAfter)
	if notificationsAfter != notificationsBefore {
		t.Errorf("cleanup created %d notifications", notificationsAfter-notificationsBefore)
	}

	// A second run finds nothing to do.
	if changed, err := CleanStored(ctx, db); err != nil || changed != 0 {
		t.Errorf("second run: changed=%d err=%v", changed, err)
	}

	// Outside the job, an edit still bumps updated_at: the guard is off.
	exec(`UPDATE chapters SET notes = 'edited' WHERE id = $1`, chapterID)
	if ts := stamp("works", workID); !ts.After(past) {
		t.Error("a normal chapter edit no longer bumps works.updated_at")
	}
}

func TestCleanStoredWorksInBatches(t *testing.T) {
	db := testdb.New(t)
	var userID = uuid.New()
	mustExec(t, db, `INSERT INTO users (id, email, username, password_hash, is_verified) VALUES ($1, 'b@example.org', 'batch', 'x', true)`, userID)
	const n = batchSize*2 + 7
	for i := 0; i < n; i++ {
		mustExec(t, db, `INSERT INTO works (id, user_id, title, summary, language, rating, status)
			VALUES ($1, $2, 'T', $3, 'en', 'general', 'draft')`, uuid.New(), userID, dirty)
	}
	changed, err := CleanStored(context.Background(), db)
	if err != nil || changed != n {
		t.Fatalf("changed=%d err=%v, want %d", changed, err, n)
	}
	var left int
	db.QueryRow(`SELECT count(*) FROM works WHERE summary LIKE '%onerror%'`).Scan(&left)
	if left != 0 {
		t.Errorf("%d works still dirty", left)
	}
}

func mustExec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("%v: %s", err, q)
	}
}
