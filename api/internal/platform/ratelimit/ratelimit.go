package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// in process limits are correct for one api instance, a shared store replaces them when api is scaled out
type Limiter struct {
	mu        sync.Mutex
	limit     rate.Limit
	burst     int
	window    time.Duration
	entries   map[string]*entry
	now       func() time.Time
	lastSweep time.Time
}

type entry struct {
	l    *rate.Limiter
	seen time.Time
}

func New(events int, per time.Duration, now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{
		limit:   rate.Limit(float64(events) / per.Seconds()),
		burst:   events,
		window:  per,
		entries: map[string]*entry{},
		now:     now,
	}
}

func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	e, ok := l.entries[key]
	if !ok {
		e = &entry{l: rate.NewLimiter(l.limit, l.burst)}
		l.entries[key] = e
	}
	e.seen = now
	r := e.l.ReserveN(now, 1)
	if !r.OK() {
		return false, l.window
	}
	if d := r.DelayFrom(now); d > 0 {
		r.CancelAt(now)
		return false, d
	}
	return true, 0
}

func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	for k, e := range l.entries {
		if now.Sub(e.seen) > 2*l.window {
			delete(l.entries, k)
		}
	}
}

// BR-SEC-02: after max failures inside the window the key waits out the window
type Lockout struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	failures map[string][]time.Time
	now      func() time.Time
}

func NewLockout(max int, window time.Duration, now func() time.Time) *Lockout {
	if now == nil {
		now = time.Now
	}
	return &Lockout{max: max, window: window, failures: map[string][]time.Time{}, now: now}
}

func (l *Lockout) Locked(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	recent := l.recent(key)
	if len(recent) < l.max {
		return false, 0
	}
	until := recent[len(recent)-1].Add(l.window)
	return true, until.Sub(l.now())
}

func (l *Lockout) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures[key] = append(l.recent(key), l.now())
}

func (l *Lockout) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, key)
}

func (l *Lockout) recent(key string) []time.Time {
	cutoff := l.now().Add(-l.window)
	kept := l.failures[key][:0]
	for _, t := range l.failures[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.failures, key)
		return nil
	}
	l.failures[key] = kept
	return kept
}
