package works

import (
	"context"
	"database/sql"
	"testing"

	"nuclear-ao3/internal/testdb"

	"github.com/google/uuid"
)

func newHitsEnv(t *testing.T) (*WorkService, uuid.UUID) {
	t.Helper()
	db := testdb.New(t)
	userID, workID := uuid.New(), uuid.New()
	if _, err := db.Exec(`INSERT INTO users (id, username, email, password_hash, display_name, is_active, is_verified)
		VALUES ($1, 'writer', 'writer@example.org', 'x', 'writer', true, true)`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO works (id, title, summary, user_id, language, rating, status, created_at, updated_at)
		VALUES ($1, 'A work', 's', $2, 'en', 'general', 'published', NOW(), NOW())`, workID, userID); err != nil {
		t.Fatal(err)
	}
	return &WorkService{db: db}, workID
}

func storedHits(t *testing.T, db *sql.DB, workID uuid.UUID) int {
	t.Helper()
	var n, onWork int
	if err := db.QueryRow(`SELECT COALESCE(SUM(hits), 0) FROM work_statistics WHERE work_id = $1`, workID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	// The work page reads works.hit_count; it must show the same number.
	if err := db.QueryRow(`SELECT hit_count FROM works WHERE id = $1`, workID).Scan(&onWork); err == nil && onWork != n {
		t.Errorf("works.hit_count = %d but work_statistics.hits = %d", onWork, n)
	}
	return n
}

func TestFlushHitsDoesNotMarkTheWorkUpdated(t *testing.T) {
	ws, workID := newHitsEnv(t)
	var before, after string
	ws.db.QueryRow(`SELECT updated_at::text FROM works WHERE id = $1`, workID).Scan(&before)
	ws.incrementHits(workID)
	if err := ws.FlushHits(context.Background()); err != nil {
		t.Fatal(err)
	}
	ws.db.QueryRow(`SELECT updated_at::text FROM works WHERE id = $1`, workID).Scan(&after)
	if before != after {
		t.Errorf("a view changed the work's updated_at from %s to %s", before, after)
	}
}

func TestFlushHitsAddsCounts(t *testing.T) {
	ws, workID := newHitsEnv(t)
	before := storedHits(t, ws.db, workID)

	for i := 0; i < 7; i++ {
		ws.incrementHits(workID)
	}
	if got := storedHits(t, ws.db, workID); got != before {
		t.Fatalf("hits written before the flush: %d", got)
	}
	if err := ws.FlushHits(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := storedHits(t, ws.db, workID); got != before+7 {
		t.Errorf("hits = %d, want %d", got, before+7)
	}

	// A second flush has nothing left to add.
	if err := ws.FlushHits(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := storedHits(t, ws.db, workID); got != before+7 {
		t.Errorf("hits after empty flush = %d, want %d", got, before+7)
	}
}

func TestFlushHitsSkipsDeletedWorks(t *testing.T) {
	ws, workID := newHitsEnv(t)
	ws.incrementHits(workID)
	ws.incrementHits(uuid.New()) // a work that no longer exists
	if err := ws.FlushHits(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := storedHits(t, ws.db, workID); got != 1 {
		t.Errorf("hits = %d, want 1", got)
	}
}

func TestFlushHitsKeepsCountsOnFailure(t *testing.T) {
	ws, workID := newHitsEnv(t)
	ws.incrementHits(workID)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ws.FlushHits(ctx); err == nil {
		t.Fatal("expected an error with a cancelled context")
	}
	if err := ws.FlushHits(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := storedHits(t, ws.db, workID); got != 1 {
		t.Errorf("hits = %d, want 1", got)
	}
}

func TestStopHitFlusherFlushes(t *testing.T) {
	ws, workID := newHitsEnv(t)
	stop := ws.StartHitFlusher()
	ws.incrementHits(workID)
	ws.incrementHits(workID)
	stop()
	if got := storedHits(t, ws.db, workID); got != 2 {
		t.Errorf("hits = %d, want 2", got)
	}
}
