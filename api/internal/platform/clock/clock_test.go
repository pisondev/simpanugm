package clock

import (
	"testing"
	"time"
)

func TestFakeStaysWhereItIsPut(t *testing.T) {
	start := time.Date(2026, 11, 2, 7, 0, 0, 0, time.UTC)
	f := NewFake(start)
	time.Sleep(20 * time.Millisecond)
	if !f.Now().Equal(start) {
		t.Fatalf("a frozen clock moved: %s", f.Now())
	}
	f.Advance(time.Hour)
	if !f.Now().Equal(start.Add(time.Hour)) {
		t.Fatalf("advance: %s", f.Now())
	}
}

func TestRunningClockKeepsCountingAfterAdvanceAndSet(t *testing.T) {
	start := time.Date(2026, 11, 2, 7, 0, 0, 0, time.UTC)
	f := NewRunning(start)
	time.Sleep(30 * time.Millisecond)
	if got := f.Now().Sub(start); got < 30*time.Millisecond || got > 5*time.Second {
		t.Fatalf("a running clock should move on its own: %s", got)
	}
	f.Advance(24 * time.Hour)
	if got := f.Now().Sub(start); got < 24*time.Hour || got > 24*time.Hour+5*time.Second {
		t.Fatalf("advance on a running clock: %s", got)
	}
	f.Set(start)
	time.Sleep(30 * time.Millisecond)
	if got := f.Now().Sub(start); got < 30*time.Millisecond || got > 5*time.Second {
		t.Fatalf("set must restart the count: %s", got)
	}
}
