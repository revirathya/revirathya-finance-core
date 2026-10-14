package ratelimit_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/revirathya/revirathya-finance-core/internal/platform/ratelimit"
)

// fakeClock is a clock the test moves forward by hand.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

func TestLimiterAllow(t *testing.T) {
	const (
		limit  = 5
		window = 10 * time.Minute
	)

	t.Run("allows up to the limit, then denies", func(t *testing.T) {
		clock := newClock()
		l := ratelimit.NewLimiterWithClock(limit, window, clock.now)

		for i := 1; i <= limit; i++ {
			if !l.Allow("A") {
				t.Fatalf("attempt %d: denied, want allowed", i)
			}
		}
		if l.Allow("A") {
			t.Fatalf("attempt %d: allowed, want denied", limit+1)
		}
	})

	t.Run("stays denied until the window ends", func(t *testing.T) {
		clock := newClock()
		l := ratelimit.NewLimiterWithClock(limit, window, clock.now)
		for range limit {
			l.Allow("A")
		}

		clock.advance(window - time.Second)
		if l.Allow("A") {
			t.Fatal("allowed one second before the window ends, want denied")
		}
	})

	t.Run("allows again after the window ends", func(t *testing.T) {
		clock := newClock()
		l := ratelimit.NewLimiterWithClock(limit, window, clock.now)
		for range limit + 1 {
			l.Allow("A")
		}

		clock.advance(window)
		for i := 1; i <= limit; i++ {
			if !l.Allow("A") {
				t.Fatalf("new window, attempt %d: denied, want allowed", i)
			}
		}
		if l.Allow("A") {
			t.Fatal("new window: allowed past the limit, want denied")
		}
	})

	t.Run("keys are independent", func(t *testing.T) {
		clock := newClock()
		l := ratelimit.NewLimiterWithClock(limit, window, clock.now)
		for range limit + 1 {
			l.Allow("A")
		}

		if !l.Allow("B") {
			t.Fatal("key B denied because key A hit its limit")
		}
	})
}

func TestLimiterSweep(t *testing.T) {
	const window = 10 * time.Minute

	t.Run("removes expired keys", func(t *testing.T) {
		clock := newClock()
		l := ratelimit.NewLimiterWithClock(5, window, clock.now)
		l.Allow("A")
		l.Allow("B")

		clock.advance(2 * window)
		l.Allow("C")

		if got := l.Len(); got != 1 {
			t.Fatalf("tracked keys: got %d, want 1 (only C)", got)
		}
	})

	t.Run("keeps keys whose window is still open", func(t *testing.T) {
		clock := newClock()
		l := ratelimit.NewLimiterWithClock(5, window, clock.now)
		l.Allow("A")

		clock.advance(window / 2)
		l.Allow("B")

		clock.advance(window / 2) // A expired, B is halfway through its window
		l.Allow("C")

		if got := l.Len(); got != 2 {
			t.Fatalf("tracked keys: got %d, want 2 (B and C)", got)
		}
	})
}

func TestLimiterConcurrent(t *testing.T) {
	const (
		limit      = 5
		goroutines = 50
	)
	l := ratelimit.NewLimiter(limit, time.Hour)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)
	for i := range goroutines {
		wg.Go(func() {
			if l.Allow("A") {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
			l.Allow(fmt.Sprintf("other-%d", i))
		})
	}
	wg.Wait()

	if allowed != limit {
		t.Fatalf("allowed %d of %d concurrent attempts, want exactly %d", allowed, goroutines, limit)
	}
}
