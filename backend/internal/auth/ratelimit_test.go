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
	rlm.anonymous.now = now
	cfg := RateLimitConfig{Tier: RateLimitTierAnonymous, Requests: 3, Window: time.Minute}

	for i := 0; i < 3; i++ {
		h, err := rlm.checkLimitWithConfig(rlm.anonymous, "k", cfg)
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
	h, err := rlm.checkLimitWithConfig(rlm.anonymous, "k", cfg)
	if err == nil {
		t.Fatal("fourth request was allowed")
	}
	if h.Remaining != 0 || h.Limit != 3 {
		t.Errorf("blocked headers: %+v", h)
	}
	// Another key is not affected.
	if _, err := rlm.checkLimitWithConfig(rlm.anonymous, "other", cfg); err != nil {
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

func TestFullTableSharesOneBucketInsteadOfForgettingClients(t *testing.T) {
	w := newWindowCounter(5)
	now, advance := fakeClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	w.now = now
	// A client already being counted uses up most of its limit.
	for i := 0; i < 4; i++ {
		w.hit("regular", 5, time.Minute)
	}
	// A flood of new addresses fills the table and keeps going.
	allowed := 0
	for i := 0; i < 50; i++ {
		advance(time.Millisecond)
		if _, _, ok := w.hit(fmt.Sprint("flood", i), 5, time.Minute); ok {
			allowed++
		}
		if w.size() > 5 {
			t.Fatalf("size %d over cap", w.size())
		}
	}
	// The 4 that fit in the table got their own buckets; the other 46 shared
	// one, which allows 5 between them, not 5 each.
	if allowed != 4*1+5 {
		t.Errorf("flood allowed %d requests, want %d", allowed, 4+5)
	}
	// The regular client was not forgotten: it still has one request left.
	if used, _, _ := w.hit("regular", 5, time.Minute); used != 4 {
		t.Errorf("regular client's count was reset: used = %d", used)
	}
	// Once the window passes, old entries expire and new clients get their own buckets again.
	advance(time.Minute)
	if used, _, ok := w.hit("newcomer", 5, time.Minute); !ok || used != 0 {
		t.Errorf("newcomer after expiry: used %d ok %v", used, ok)
	}
}

func TestIPv6IsCountedPerNetwork(t *testing.T) {
	a := clientNetwork("2001:db8:1:2:aaaa::1")
	b := clientNetwork("2001:db8:1:2:ffff:ffff:ffff:ffff")
	c := clientNetwork("2001:db8:1:3::1")
	if a != b {
		t.Errorf("two addresses in one /64 counted apart: %s %s", a, b)
	}
	if a == c {
		t.Errorf("different /64 networks counted together: %s", a)
	}
	if got := clientNetwork("203.0.113.7"); got != "203.0.113.7" {
		t.Errorf("IPv4 changed: %s", got)
	}
}

func TestAnonymousFloodCantCrowdOutLoggedInUsers(t *testing.T) {
	rlm := newRateLimitManager("test")
	anon := &ClientRateLimitInfo{Tier: RateLimitTierAnonymous}
	user := &ClientRateLimitInfo{Tier: RateLimitTierPublic, UserID: "u1", ClientID: "user:u1"}
	rlm.CheckRateLimit(user, "198.51.100.1")
	for i := 0; i < 2000; i++ {
		rlm.CheckRateLimit(anon, fmt.Sprintf("10.0.%d.%d", i/250, i%250))
	}
	if rlm.users.size() != 1 {
		t.Errorf("logged-in table has %d entries, want 1", rlm.users.size())
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
