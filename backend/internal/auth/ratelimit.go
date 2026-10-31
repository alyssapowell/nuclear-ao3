package auth

import (
	"container/list"
	"sync"
	"time"
)

// Rate limit counters live in this process's memory. Each copy of the app
// counts on its own, so with N copies behind a load balancer a client can
// make up to N times the limit. See docs/DECISIONS.md ("Redis removed").
// Counts are lost on restart, which at worst gives a client a fresh window.
//
// The table never pushes out a client it is still counting: pushing out live
// entries would hand someone sending from many addresses a fresh count with
// every new one. When the table is full, the caller (RateLimitManager) counts
// new visitors by their wider network in a second table instead, and only if
// that is full too do newcomers share one overflow bucket.

const (
	// maxRateLimitKeys caps how many clients are tracked at once.
	maxRateLimitKeys = 100000
	// rateLimitSweepEvery is how often expired windows are dropped.
	rateLimitSweepEvery = time.Minute
	// rateLimitFullSweepEvery is the most often a full table is swept early,
	// so a flood can't make every request scan the whole table.
	rateLimitFullSweepEvery = time.Second
)

// windowCounter is a fixed-window counter per key. Windows line up with the
// clock (for example every minute on the minute).
type windowCounter struct {
	mu        sync.Mutex
	entries   map[string]*list.Element // key -> element holding *windowEntry
	order     *list.List               // front = most recently used
	maxKeys   int
	overflow  *windowEntry // shared by new clients while the table is full
	lastSweep time.Time
	now       func() time.Time
}

type windowEntry struct {
	key         string
	count       int
	windowStart time.Time
	window      time.Duration
}

func newWindowCounter(maxKeys int) *windowCounter {
	return &windowCounter{
		entries: map[string]*list.Element{},
		order:   list.New(),
		maxKeys: maxKeys,
		now:     time.Now,
	}
}

// hit counts one request for key. It returns the number of requests already
// counted in this window (before this one), the end of the window, and
// whether the request is allowed. Blocked requests are not counted.
func (w *windowCounter) hit(key string, limit int, window time.Duration) (used int, windowEnd time.Time, allowed bool) {
	now := w.now()
	start := now.Truncate(window)
	windowEnd = start.Add(window)

	w.mu.Lock()
	defer w.mu.Unlock()

	if now.Sub(w.lastSweep) >= rateLimitSweepEvery {
		w.sweepLocked(now)
	}

	var e *windowEntry
	if el, ok := w.entries[key]; ok {
		e = el.Value.(*windowEntry)
		w.order.MoveToFront(el)
		if !e.windowStart.Equal(start) {
			e.count = 0
			e.windowStart = start
		}
		e.window = window
	} else {
		if len(w.entries) >= w.maxKeys && now.Sub(w.lastSweep) >= rateLimitFullSweepEvery {
			w.sweepLocked(now)
		}
		if len(w.entries) >= w.maxKeys {
			e = w.overflowLocked(start, window)
		} else {
			e = &windowEntry{key: key, windowStart: start, window: window}
			w.entries[key] = w.order.PushFront(e)
		}
	}

	used = e.count
	if used >= limit {
		return used, windowEnd, false
	}
	e.count++
	return used, windowEnd, true
}

// sweepLocked drops entries whose window has ended.
func (w *windowCounter) sweepLocked(now time.Time) {
	w.lastSweep = now
	for el := w.order.Back(); el != nil; {
		prev := el.Prev()
		e := el.Value.(*windowEntry)
		if !now.Before(e.windowStart.Add(e.window)) {
			w.order.Remove(el)
			delete(w.entries, e.key)
		}
		el = prev
	}
}

// overflowLocked returns the one bucket shared by every client that arrives
// while the table is full. It sits outside the table, so it never displaces
// anyone.
func (w *windowCounter) overflowLocked(start time.Time, window time.Duration) *windowEntry {
	if w.overflow == nil || !w.overflow.windowStart.Equal(start) {
		w.overflow = &windowEntry{key: "overflow", windowStart: start, window: window}
	}
	return w.overflow
}

// tracking reports whether key has its own entry, or would get one: false
// means the table is full and key would land in the overflow bucket.
func (w *windowCounter) tracking(key string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, ok := w.entries[key]
	return ok || len(w.entries) < w.maxKeys
}

// size reports how many keys are tracked.
func (w *windowCounter) size() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.entries)
}
