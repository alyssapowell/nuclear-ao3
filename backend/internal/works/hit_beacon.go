package works

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/auth"
	"nuclear-ao3/internal/platform"
)

// RecordHit answers POST /works/:work_id/hit, which the work page sends once
// when it opens. It is the only thing that counts a view (a hit), so every
// read address can be cached by a CDN while views are still counted: this
// one is a POST and is never cached.
//
// Like AO3, a view counts once per visitor per work per day (a visitor is
// their network: an IPv4 address or an IPv6 /64), and the work's own
// creators' views don't count. Only a work the visitor may read is counted.
// The answer is always 204, so it says nothing about the work.
func (ws *WorkService) RecordHit(c *gin.Context) {
	defer c.Status(http.StatusNoContent)
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		return
	}
	visitor := auth.ClientNetwork(c.ClientIP())
	if ws.hitSeen.seen(workID, visitor, time.Now()) {
		return
	}
	viewer, asReader := platform.ReaderViewer(c)
	var count bool
	err = ws.db.QueryRowContext(c.Request.Context(), `
		SELECT can_user_view_work($1, $2, $3) AND NOT ($2::uuid IS NOT NULL AND is_work_creator($1, $2))`,
		workID, viewer, asReader).Scan(&count)
	if err != nil {
		log.Printf("hit: work %s: %v", workID, err)
		return
	}
	if count {
		ws.incrementHits(workID)
	}
}

// hitSeen remembers which visitors' views of which works were already
// counted today. It is cleared when the day changes. Each copy of the app
// keeps its own, so with several copies a visitor can count once per copy a
// day; that is the price of keeping it out of the database. If it grows past
// maxHitSeen (a flood of made-up visitors) it starts again rather than use
// more memory, which can only let a few extra views count.
type hitSeen struct {
	mu   sync.Mutex
	day  string
	keys map[string]struct{}
}

const maxHitSeen = 1 << 20

// seen reports whether this visitor's view of the work was counted today,
// and marks it as counted.
func (h *hitSeen) seen(workID uuid.UUID, visitor string, now time.Time) bool {
	day := now.UTC().Format("2006-01-02")
	key := workID.String() + "|" + visitor
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.day != day || h.keys == nil || len(h.keys) >= maxHitSeen {
		h.day, h.keys = day, map[string]struct{}{}
	}
	if _, ok := h.keys[key]; ok {
		return true
	}
	h.keys[key] = struct{}{}
	return false
}
