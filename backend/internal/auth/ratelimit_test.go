package auth

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func fakeClock(start time.Time) (func() time.Time, func(time.Duration)) {
	var mu sync.Mutex
	now := start
	return func() time.Time { mu.Lock(); defer mu.Unlock(); return now },
		func(d time.Duration) { mu.Lock(); now = now.Add(d); mu.Unlock() }
}

func TestRateLimitBlocksAtLimitWithHeaders(t *testing.T) {
	rlm := newRateLimitManager("test")
	now, _ := fakeClock(time.Date(2026, 1, 1, 12, 0, 10, 0, time.UTC))
	rlm.counter.now = now
	cfg := RateLimitConfig{Tier: RateLimitTierAnonymous, Requests: 3, Window: time.Minute}

	for i := 0; i < 3; i++ {
		h, err := rlm.checkLimitWithConfig("k", cfg)
		if err != nil {
			t.Fatalf("request %d blocked: %v", i, err)
		}
		if h.Remaining != 2-i || h.Limit != 3 || h.Tier != "anonymous" {
			t.Errorf("request %d headers: %+v", i, h)
		}
		if want := time.Date(2026, 1, 1, 12, 1, 0, 0, time.UTC).Unix(); h.Reset != want {
			t.Errorf("reset = %d, want %d", h.Reset, want)
		}
	}
	h, err := rlm.checkLimitWithConfig("k", cfg)
	if err == nil {
		t.Fatal("fourth request was allowed")
	}
	if h.Remaining != 0 || h.Limit != 3 {
		t.Errorf("blocked headers: %+v", h)
	}
	// Another key is not affected.
	if _, err := rlm.checkLimitWithConfig("other", cfg); err != nil {
		t.Errorf("other key blocked: %v", err)
	}
}

func TestRateLimitWindowResets(t *testing.T) {
	w := newWindowCounter(100)
	now, advance := fakeClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	w.now = now
	for i := 0; i < 2; i++ {
		w.hit("k", 2, time.Minute)
	}
	if _, _, ok := w.hit("k", 2, time.Minute); ok {
		t.Fatal("allowed over the limit")
	}
	advance(time.Minute)
	if _, _, ok := w.hit("k", 2, time.Minute); !ok {
		t.Fatal("still blocked in the next window")
	}
}

func TestRateLimitEvictsIdleKeys(t *testing.T) {
	w := newWindowCounter(100)
	now, advance := fakeClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	w.now = now
	for i := 0; i < 10; i++ {
		w.hit(fmt.Sprint("k", i), 5, time.Minute)
	}
	if w.size() != 10 {
		t.Fatalf("size = %d", w.size())
	}
	advance(3 * time.Minute)
	w.hit("fresh", 5, time.Minute) // triggers the sweep
	if w.size() != 1 {
		t.Errorf("size after sweep = %d, want 1", w.size())
	}
}

func TestRateLimitKeyCapEvictsOldestAndKeepsServing(t *testing.T) {
	w := newWindowCounter(5)
	now, advance := fakeClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	w.now = now
	for i := 0; i < 20; i++ {
		advance(time.Millisecond)
		if _, _, ok := w.hit(fmt.Sprint("k", i), 5, time.Minute); !ok {
			t.Fatalf("request for new key %d refused", i)
		}
		if w.size() > 5 {
			t.Fatalf("size %d over cap", w.size())
		}
	}
	// The newest keys survived, the oldest did not.
	if used, _, _ := w.hit("k19", 5, time.Minute); used != 1 {
		t.Errorf("newest key lost its count: used = %d", used)
	}
	if used, _, _ := w.hit("k0", 5, time.Minute); used != 0 {
		t.Errorf("oldest key kept its count: used = %d", used)
	}
}

func TestRateLimitConcurrent(t *testing.T) {
	w := newWindowCounter(50)
	var allowed int64
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				// One shared key, plus many distinct keys to force eviction.
				if _, _, ok := w.hit("shared", 100, time.Hour); ok {
					atomic.AddInt64(&allowed, 1)
				}
				w.hit(fmt.Sprint("g", g, "-", i), 10, time.Hour)
			}
		}(g)
	}
	wg.Wait()
	if allowed > 100 {
		t.Errorf("shared key allowed %d, limit 100", allowed)
	}
	if w.size() > 50 {
		t.Errorf("size %d over cap", w.size())
	}
}
