package ratelimit

import (
	"sync"
	"time"
)

type entry struct {
	count int
	start time.Time
}

type Limiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	now       func() time.Time
	entries   map[string]*entry
	lastSweep time.Time
}

func NewLimiterWithClock(limit int, window time.Duration, now func() time.Time) *Limiter {
	return &Limiter{
		limit:     limit,
		window:    window,
		now:       now,
		entries:   make(map[string]*entry),
		lastSweep: now(),
	}
}

func NewLimiter(limit int, window time.Duration) *Limiter {
	return NewLimiterWithClock(limit, window, time.Now)
}

func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}
	for key, e := range l.entries {
		if now.Sub(e.start) >= l.window {
			delete(l.entries, key)
		}
	}
	l.lastSweep = now
}

// Take counts one event for key. Over the limit, it returns false and how long until key's window resets;
// a denied event isn't counted.
func (l *Limiter) Take(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	e, ok := l.entries[key]
	if !ok || now.Sub(e.start) >= l.window {
		l.entries[key] = &entry{
			count: 1,
			start: now,
		}
		return true, 0
	}

	if e.count >= l.limit {
		return false, e.start.Add(l.window).Sub(now)
	}

	e.count += 1
	return true, 0
}

// Allow counts one event for key and reports whether it is within the limit.
func (l *Limiter) Allow(key string) bool {
	ok, _ := l.Take(key)
	return ok
}

func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}
