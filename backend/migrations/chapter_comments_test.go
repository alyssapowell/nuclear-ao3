package migrations

import (
	"testing"

	"github.com/google/uuid"
)

// Migration 043: a comment always belongs to a work and may also name one of
// that work's chapters. Rows that named only a chapter get the chapter's work.
func TestCommentsBelongToAWorkAndMayNameItsChapter(t *testing.T) {
	db := testDB(t)
	rest := migrateBefore(t, db, "043")
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	user := uuid.New()
	mustExec(`INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'writer', 'writer@example.org', 'x')`, user)
	newWork := func() (work, chapter uuid.UUID) {
		work, chapter = uuid.New(), uuid.New()
		mustExec(`INSERT INTO works (id, user_id, title, summary, language, rating, status, is_draft, published_at)
			VALUES ($1, $2, 'w', 's', 'en', 'general', 'published', false, now())`, work, user)
		mustExec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, published_at, summary, notes, end_notes)
			VALUES ($1, $2, 1, 'one', 'text', false, now(), '', '', '')`, chapter, work)
		return
	}
	workA, chapterA := newWork()
	_, chapterB := newWork()

	// Before: a chapter-only comment was allowed, both together were not.
	old := uuid.New()
	mustExec(`INSERT INTO comments (id, chapter_id, guest_name, content) VALUES ($1, $2, 'guest', 'old chapter comment')`, old, chapterA)
	if _, err := db.Exec(`INSERT INTO comments (work_id, chapter_id, guest_name, content) VALUES ($1, $2, 'guest', 'both')`, workA, chapterA); err == nil {
		t.Fatal("expected the old constraint to refuse work and chapter together")
	}

	for _, m := range rest {
		if _, err := db.Exec(m.SQL); err != nil {
			t.Fatalf("migration %s: %v", m.Version, err)
		}
	}

	var work uuid.UUID
	if err := db.QueryRow(`SELECT work_id FROM comments WHERE id = $1`, old).Scan(&work); err != nil || work != workA {
		t.Errorf("old chapter comment got work %v (%v), want %v", work, err, workA)
	}
	var count int
	db.QueryRow(`SELECT comment_count FROM works WHERE id = $1`, workA).Scan(&count)
	if count != 1 {
		t.Errorf("comment_count %d after the backfill, want 1", count)
	}

	// After: work and its own chapter together are fine, and counted.
	mustExec(`INSERT INTO comments (work_id, chapter_id, guest_name, content) VALUES ($1, $2, 'guest', 'chapter comment')`, workA, chapterA)
	mustExec(`INSERT INTO comments (work_id, guest_name, content) VALUES ($1, 'guest', 'work comment')`, workA)
	db.QueryRow(`SELECT comment_count FROM works WHERE id = $1`, workA).Scan(&count)
	if count != 3 {
		t.Errorf("comment_count %d, want 3", count)
	}
	// A chapter of another work, or no work at all, is refused.
	if _, err := db.Exec(`INSERT INTO comments (work_id, chapter_id, guest_name, content) VALUES ($1, $2, 'guest', 'wrong work')`, workA, chapterB); err == nil {
		t.Error("a comment on another work's chapter was accepted")
	}
	if _, err := db.Exec(`INSERT INTO comments (chapter_id, guest_name, content) VALUES ($1, 'guest', 'no work')`, chapterA); err == nil {
		t.Error("a comment with no work was accepted")
	}
	// Deleting the chapter takes its comments with it.
	mustExec(`DELETE FROM chapters WHERE id = $1`, chapterA)
	db.QueryRow(`SELECT count(*) FROM comments WHERE content LIKE '%chapter comment'`).Scan(&count)
	if count != 0 {
		t.Errorf("%d chapter comments left after deleting the chapter", count)
	}
}
