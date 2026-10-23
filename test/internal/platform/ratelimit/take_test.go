package ratelimit_test

import (
	"testing"
	"time"

	"github.com/revirathya/revirathya-finance-core/internal/platform/ratelimit"
)

func TestLimiterTake(t *testing.T) {
	t.Run("a denied event reports the time until the window resets", func(t *testing.T) {
		clock := newClock()
		l := ratelimit.NewLimiterWithClock(2, time.Minute, clock.now)

		for i := 1; i <= 2; i++ {
			if ok, retry := l.Take("A"); !ok || retry != 0 {
				t.Fatalf("event %d: got %v, %v; want allowed with no wait", i, ok, retry)
			}
		}
		clock.advance(20 * time.Second)

		ok, retry := l.Take("A")
		if ok || retry != 40*time.Second {
			t.Fatalf("got %v, %v; want denied with 40s left", ok, retry)
		}
	})

	t.Run("denied events don't extend the window", func(t *testing.T) {
		clock := newClock()
		l := ratelimit.NewLimiterWithClock(1, time.Minute, clock.now)
		l.Take("A")
		for range 5 {
			clock.advance(10 * time.Second)
			l.Take("A")
		}
		clock.advance(10 * time.Second) // 60s after the first event

		if ok, _ := l.Take("A"); !ok {
			t.Fatal("denied after the window ended, want a new window")
		}
	})
}
