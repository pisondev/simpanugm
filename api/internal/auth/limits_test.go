package auth

import (
	"testing"
	"time"

	"github.com/pisondev/simpanugm/api/internal/platform/clock"
)

func TestLoginIPBudget(t *testing.T) {
	for _, budget := range []int{20, 500} {
		fake := clock.NewFake(time.Now())
		svc := NewService(nil, fake, nil, nil)
		if budget != 20 {
			svc.LimitLoginsPerIP(budget)
		}
		for i := range budget {
			if ok, _ := svc.loginIP.Allow("ip"); !ok {
				t.Fatalf("budget %d refused attempt %d", budget, i+1)
			}
		}
		if ok, wait := svc.loginIP.Allow("ip"); ok || wait <= 0 {
			t.Fatalf("budget %d failed to limit: allowed=%v wait=%s", budget, ok, wait)
		}
		if ok, _ := svc.loginIP.Allow("other-ip"); !ok {
			t.Fatal("one IP exhausted another address's budget")
		}
		fake.Advance(15 * time.Minute)
		if ok, _ := svc.loginIP.Allow("ip"); !ok {
			t.Fatal("IP budget did not refill")
		}
		for range 5 {
			if ok, _ := svc.loginPair.Allow("ip|email"); !ok {
				t.Fatal("default pair budget changed")
			}
		}
		if ok, _ := svc.loginPair.Allow("ip|email"); ok {
			t.Fatal("raising the IP budget must preserve the pair limit")
		}
	}
}

func TestConfiguredLoginPairBudget(t *testing.T) {
	svc := NewService(nil, clock.NewFake(time.Now()), nil, nil)
	svc.LimitLoginPairs(500)
	for range 500 {
		if ok, _ := svc.loginPair.Allow("ip|email"); !ok {
			t.Fatal("configured pair budget refused a permitted attempt")
		}
	}
	if ok, _ := svc.loginPair.Allow("ip|email"); ok {
		t.Fatal("configured pair budget failed to limit")
	}
}
