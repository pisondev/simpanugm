package ratelimit

import (
	"testing"
	"time"
)

type fakeTime struct{ t time.Time }

func (f *fakeTime) now() time.Time { return f.t }

func TestLimiterAllowsBurstThenWaits(t *testing.T) {
	clk := &fakeTime{t: time.Unix(1_800_000_000, 0)}
	l := New(5, 15*time.Minute, clk.now)
	for i := 0; i < 5; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("attempt %d refused", i+1)
		}
	}
	ok, wait := l.Allow("k")
	if ok || wait <= 0 || wait > 3*time.Minute {
		t.Fatalf("sixth attempt: ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys must be independent")
	}
	clk.t = clk.t.Add(wait)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("allowed again after waiting")
	}
}

func TestLockout(t *testing.T) {
	clk := &fakeTime{t: time.Unix(1_800_000_000, 0)}
	l := NewLockout(5, 15*time.Minute, clk.now)
	for i := 0; i < 4; i++ {
		l.Fail("a")
	}
	if locked, _ := l.Locked("a"); locked {
		t.Fatal("locked after four failures")
	}
	l.Fail("a")
	locked, wait := l.Locked("a")
	if !locked || wait != 15*time.Minute {
		t.Fatalf("after five failures: locked=%v wait=%v", locked, wait)
	}
	clk.t = clk.t.Add(16 * time.Minute)
	if locked, _ := l.Locked("a"); locked {
		t.Fatal("still locked after the window")
	}
	l.Fail("b")
	l.Reset("b")
	if locked, _ := l.Locked("b"); locked {
		t.Fatal("reset must clear")
	}
}
