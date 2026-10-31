package works

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// How often counted views are written to the database.
const hitFlushInterval = 30 * time.Second

// hitCounter holds views that have not been written to the database yet.
// A crash loses at most one flush interval of views. The zero value is ready
// to use.
type hitCounter struct {
	mu      sync.Mutex
	pending map[uuid.UUID]int
}

func (h *hitCounter) add(workID uuid.UUID) {
	h.addN(workID, 1)
}

func (h *hitCounter) addN(workID uuid.UUID, n int) {
	h.mu.Lock()
	if h.pending == nil {
		h.pending = map[uuid.UUID]int{}
	}
	h.pending[workID] += n
	h.mu.Unlock()
}

// take returns everything counted so far and starts again from zero.
func (h *hitCounter) take() map[uuid.UUID]int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.pending) == 0 {
		return nil
	}
	out := h.pending
	h.pending = nil
	return out
}

// FlushHits adds the counted views to work_statistics.hits in one statement.
// If the write fails the counts are kept for the next flush.
func (ws *WorkService) FlushHits(ctx context.Context) error {
	counts := ws.hits.take()
	if len(counts) == 0 {
		return nil
	}

	// Sorted so copies of the app flushing at once lock rows in the same order.
	ids := make([]uuid.UUID, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	idStrs := make([]string, len(ids))
	adds := make([]int64, len(ids))
	for i, id := range ids {
		idStrs[i] = id.String()
		adds[i] = int64(counts[id])
	}

	// Hits are kept in two places that different pages read: works.hit_count
	// (the work page, search) and work_statistics.hits (listings, profile
	// totals). Both are added to together so every page shows the same
	// number. Joining works skips views of works deleted since they were
	// counted.
	err := func() error {
		tx, err := ws.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, `
			UPDATE works w SET hit_count = w.hit_count + v.n
			FROM unnest($1::uuid[], $2::int[]) AS v(id, n)
			WHERE w.id = v.id`, pq.Array(idStrs), pq.Array(adds)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO work_statistics (work_id, hits, kudos, comments, bookmarks, updated_at)
			SELECT w.id, v.n, 0, 0, 0, NOW()
			FROM unnest($1::uuid[], $2::int[]) AS v(id, n)
			JOIN works w ON w.id = v.id
			ON CONFLICT (work_id)
			DO UPDATE SET hits = work_statistics.hits + EXCLUDED.hits, updated_at = NOW()`,
			pq.Array(idStrs), pq.Array(adds)); err != nil {
			return err
		}
		return tx.Commit()
	}()
	if err != nil {
		for id, n := range counts {
			ws.hits.addN(id, n)
		}
		return fmt.Errorf("flush hits: %w", err)
	}
	return nil
}

// StartHitFlusher writes counted views to the database every 30 seconds. The
// returned function stops it and does a last flush, so call it on shutdown.
func (ws *WorkService) StartHitFlusher() (stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(hitFlushInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := ws.FlushHits(ctx); err != nil {
					log.Printf("works: %v", err)
				}
			}
		}
	}()
	return func() {
		cancel()
		<-done
		flushCtx, flushCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer flushCancel()
		if err := ws.FlushHits(flushCtx); err != nil {
			log.Printf("works: final %v", err)
		}
	}
}
