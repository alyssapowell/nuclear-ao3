package works

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nuclear-ao3/internal/testdb"

	"github.com/gin-gonic/gin"
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

func TestHitCountsOncePerVisitorPerDay(t *testing.T) {
	var h hitSeen
	w1, w2 := uuid.New(), uuid.New()
	day := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	if h.seen(w1, "203.0.113.7", day) {
		t.Fatal("first view counted as seen")
	}
	if !h.seen(w1, "203.0.113.7", day.Add(5*time.Hour)) {
		t.Error("the same visitor's second view on the same day counted again")
	}
	if h.seen(w2, "203.0.113.7", day) || h.seen(w1, "198.51.100.1", day) {
		t.Error("another work, or another visitor, was treated as already seen")
	}
	if h.seen(w1, "203.0.113.7", day.Add(24*time.Hour)) {
		t.Error("the next day's view wasn't counted")
	}
}

func TestRecordHitCountsReadersNotCreatorsOrHiddenWorks(t *testing.T) {
	ws, workID := newHitsEnv(t)
	var creator uuid.UUID
	ws.db.QueryRow(`SELECT user_id FROM works WHERE id = $1`, workID).Scan(&creator)
	pseud := uuid.New()
	ws.db.Exec(`INSERT INTO pseuds (id, user_id, name, is_default) VALUES ($1, $2, 'writerpen', false)`, pseud, creator)
	ws.db.Exec(`INSERT INTO creatorships (creation_id, creation_type, pseud_id, approved) VALUES ($1, 'Work', $2, true)`, workID, pseud)

	gin.SetMode(gin.TestMode)
	hit := func(id uuid.UUID, ip string, user *uuid.UUID) int {
		r := gin.New()
		r.POST("/works/:work_id/hit", func(c *gin.Context) {
			if user != nil {
				c.Set("user_uuid", *user)
			}
			ws.RecordHit(c)
		})
		req := httptest.NewRequest("POST", "/works/"+id.String()+"/hit", nil)
		req.RemoteAddr = ip + ":1000"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	flushed := func() int {
		if err := ws.FlushHits(context.Background()); err != nil {
			t.Fatal(err)
		}
		return storedHits(t, ws.db, workID)
	}

	before := flushed()
	hit(workID, "203.0.113.1", nil)
	hit(workID, "203.0.113.1", nil) // same visitor, same day
	hit(workID, "2001:db8:1:2::1", nil)
	hit(workID, "2001:db8:1:2::99", nil) // same IPv6 /64
	if got := flushed() - before; got != 2 {
		t.Errorf("%d views counted, want 2 (one per visitor per day)", got)
	}
	before = flushed()
	hit(workID, "198.51.100.9", &creator)
	if got := flushed() - before; got != 0 {
		t.Errorf("the creator's own view counted (%d)", got)
	}
	ws.db.Exec(`UPDATE works SET status = 'draft' WHERE id = $1`, workID)
	if code := hit(workID, "198.51.100.10", nil); code != http.StatusNoContent {
		t.Errorf("a draft answered %d, want the same 204 as any work", code)
	}
	if code := hit(uuid.New(), "198.51.100.11", nil); code != http.StatusNoContent {
		t.Errorf("a missing work answered %d, want 204", code)
	}
	if got := flushed() - before; got != 0 {
		t.Errorf("a view of a draft counted (%d)", got)
	}
}
