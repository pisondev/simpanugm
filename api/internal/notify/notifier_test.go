package notify

import (
	"strings"
	"testing"
	"time"

	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/settings"
)

func wib(t *testing.T, s string) time.Time {
	t.Helper()
	at, err := time.ParseInLocation("2006-01-02 15:04", s, clock.WIB())
	if err != nil {
		t.Fatal(err)
	}
	return at
}

// BR-NTF-03 with the default window 22:00 to 06:00 WIB
func TestQuietUntil(t *testing.T) {
	t.Parallel()
	q := settings.QuietHours{From: "22:00", To: "06:00"}
	cases := []struct {
		now   string
		quiet bool
		until string
	}{
		{"2026-11-02 21:59", false, ""},
		{"2026-11-02 22:00", true, "2026-11-03 06:00"},
		{"2026-11-02 23:30", true, "2026-11-03 06:00"},
		{"2026-11-03 00:10", true, "2026-11-03 06:00"},
		{"2026-11-03 05:59", true, "2026-11-03 06:00"},
		{"2026-11-03 06:00", false, ""},
		{"2026-11-03 12:00", false, ""},
	}
	for _, c := range cases {
		until, quiet := QuietUntil(wib(t, c.now).UTC(), q)
		if quiet != c.quiet {
			t.Fatalf("%s: quiet %v", c.now, quiet)
		}
		if quiet && !until.Equal(wib(t, c.until)) {
			t.Fatalf("%s: until %s", c.now, until.In(clock.WIB()))
		}
	}
	if _, quiet := QuietUntil(time.Now(), settings.QuietHours{From: "bad", To: "06:00"}); quiet {
		t.Fatal("a malformed window must never hold email back")
	}
	if until, quiet := QuietUntil(wib(t, "2026-11-02 13:00"), settings.QuietHours{From: "12:00", To: "14:00"}); !quiet || !until.Equal(wib(t, "2026-11-02 14:00")) {
		t.Fatalf("same day window: %v %v", quiet, until)
	}
}

func TestDailyAtIsSevenInTheMorningWIB(t *testing.T) {
	t.Parallel()
	d := DailyAt{Hour: 7}
	if got := d.Next(wib(t, "2026-11-02 06:59")); !got.Equal(wib(t, "2026-11-02 07:00")) {
		t.Fatalf("before seven: %s", got.In(clock.WIB()))
	}
	if got := d.Next(wib(t, "2026-11-02 07:00")); !got.Equal(wib(t, "2026-11-03 07:00")) {
		t.Fatalf("at seven: %s", got.In(clock.WIB()))
	}
}

func TestEveryCatalogEntryRendersAndStaysInIndonesian(t *testing.T) {
	t.Parallel()
	data := map[string]string{
		"lost_code": "LST-2611-0007", "found_code": "FND-2611-0012", "claim_code": "CLM-2611-0001",
		"item_name": "Dompet cokelat", "post_name": "Pos Perpustakaan", "count": "3", "score": "81%",
	}
	dashes := string([]rune{0x2013, 0x2014})
	for e, s := range catalog {
		for _, text := range []string{s.Title, s.Body} {
			out, err := render(text, data)
			if err != nil {
				t.Fatalf("%s: %v", e, err)
			}
			if strings.Contains(out, "{{") || strings.ContainsAny(out, dashes) {
				t.Fatalf("%s rendered %q", e, out)
			}
		}
		if s.Digest && s.Email {
			t.Fatalf("%s: digest events reach officers through the digest only", e)
		}
	}
	if _, err := render(catalog[MatchHighForOwner].Body, map[string]string{}); err == nil {
		t.Fatal("a missing value must fail rendering instead of printing <no value>")
	}
}

func TestGroupIsTheEventPrefix(t *testing.T) {
	t.Parallel()
	if MatchHighForOwner.Group() != "match" || FoundReceived.Group() != "found" {
		t.Fatal("group")
	}
}
