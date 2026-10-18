package auth

import (
	"net/netip"
	"testing"
)

func TestDeviceLabels(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Mozilla/5.0 (Linux; Android 14; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Mobile Safari/537.36":                       "Chrome di Android",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1": "Safari di iOS",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Safari/537.36 Edg/141.0":                   "Edge di Windows",
		"Mozilla/5.0 (X11; Linux x86_64; rv:143.0) Gecko/20100101 Firefox/143.0":                                                                  "Firefox di Linux",
		"": "Peramban lain di perangkat lain",
	}
	for ua, want := range cases {
		if got := Device(ua); got != want {
			t.Errorf("%q: %q, want %q", ua, got, want)
		}
	}
}

func TestIPHintNeverShowsTheFullAddress(t *testing.T) {
	t.Parallel()
	v4 := netip.MustParseAddr("182.253.14.7")
	if got := ipHint(&v4); got != "182.253.0.0/16" {
		t.Fatalf("v4: %s", got)
	}
	v6 := netip.MustParseAddr("2001:db8:1234:5678::1")
	if got := ipHint(&v6); got != "2001:db8:1234::/48" {
		t.Fatalf("v6: %s", got)
	}
	if ipHint(nil) != "" {
		t.Fatal("no address, no hint")
	}
}
