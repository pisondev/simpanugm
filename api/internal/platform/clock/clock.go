package clock

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type System struct{}

func (System) Now() time.Time { return time.Now().UTC() }

type Fake struct {
	mu  sync.Mutex
	now time.Time
	// a running clock keeps counting from the moment it was last set
	running bool
	since   time.Time
}

func NewFake(t time.Time) *Fake { return &Fake{now: t.UTC()} }

// NewRunning starts at t and keeps moving, so a long end to end run stays in step with the browser
func NewRunning(t time.Time) *Fake { return &Fake{now: t.UTC(), running: true, since: time.Now()} }

func (f *Fake) read() time.Time {
	if f.running {
		return f.now.Add(time.Since(f.since))
	}
	return f.now
}

func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.read()
}

func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	f.now = t.UTC()
	f.since = time.Now()
	f.mu.Unlock()
}

func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	f.now = f.read().Add(d)
	f.since = time.Now()
	f.mu.Unlock()
}

var wib = mustLoad("Asia/Jakarta")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func WIB() *time.Location { return wib }
