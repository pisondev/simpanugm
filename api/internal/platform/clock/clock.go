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
}

func NewFake(t time.Time) *Fake { return &Fake{now: t.UTC()} }

func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	f.now = t.UTC()
	f.mu.Unlock()
}

func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
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
