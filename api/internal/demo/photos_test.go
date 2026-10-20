package demo

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverBackgroundFoundReportsFiltersAndKeysAreStable(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"bg-wallet-a.jpg", "bg-wallet-a-copy.jpg", "bg-wallet-b.jpg", "hero-wallet-lost.jpg"} {
		content := []byte(name)
		if name == "bg-wallet-a-copy.jpg" {
			content = []byte("bg-wallet-a.jpg")
		}
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manifest := "filename,source,author,license,source_url,category,item_type,notes\n" +
		"bg-wallet-b.jpg,wikimedia_commons,creator,CC BY 4.0,https://example.test/b,dompet,background,usable\n" +
		"bg-wallet-a.jpg,open_images,creator,CC0,https://example.test/a,dompet,background,usable\n" +
		"bg-wallet-a-copy.jpg,wikimedia_commons,creator,CC BY 4.0,https://example.test/copy,dompet,background,duplicate content\n" +
		"hero-wallet-lost.jpg,derived,creator,CC BY 4.0,https://example.test/hero,dompet,hero_lost,derived pair\n" +
		"bg-holdout.jpg,wikimedia_commons,creator,CC BY 4.0,https://example.test/holdout,dompet,background,holdout reserved\n" +
		"bg-missing.jpg,wikimedia_commons,creator,CC BY 4.0,https://example.test/missing,dompet,background,usable\n" +
		"bg-fixture.jpg,team_fixture,team,fixture,local,dompet,background,placeholder\n"
	got, err := discoverBackgroundFoundReports(strings.NewReader(manifest), dir, []string{"dompet"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d reports, want 2", len(got))
	}
	if got[0].PhotoFilename != "bg-wallet-a.jpg" || got[1].PhotoFilename != "bg-wallet-b.jpg" {
		t.Fatalf("reports not sorted or include an ineligible photo: %#v", got)
	}
	if got[0].Key == got[1].Key || !strings.HasPrefix(got[0].Key, "bg-found-photo-") {
		t.Fatalf("unexpected stable keys: %q, %q", got[0].Key, got[1].Key)
	}
	gotAgain, err := discoverBackgroundFoundReports(strings.NewReader(manifest), dir, []string{"dompet"})
	if err != nil {
		t.Fatal(err)
	}
	keys := func(reports []DemoFoundReport) []string {
		result := make([]string, len(reports))
		for i, report := range reports {
			result[i] = report.Key
		}
		return result
	}
	if !reflect.DeepEqual(keys(got), keys(gotAgain)) {
		t.Fatalf("keys changed across identical discovery: %v, %v", keys(got), keys(gotAgain))
	}
}

func TestPhotoDiscoveryPreservesHeroReferences(t *testing.T) {
	if len(HeroFoundReports) == 0 || HeroFoundReports[0].PhotoFilename != "hero-earbuds-found.jpg" {
		t.Fatalf("hero found photo references changed: %#v", HeroFoundReports)
	}
	if len(HeroLostReports) == 0 || HeroLostReports[0].PhotoFilename != "hero-earbuds-lost.jpg" {
		t.Fatalf("hero lost photo references changed: %#v", HeroLostReports)
	}
}

func TestBalanceDemoPhotoUploadersStaysWithinUploadLimits(t *testing.T) {
	reports := make([]DemoFoundReport, 90)
	for i := range reports {
		reports[i] = DemoFoundReport{Key: "bg-found-photo-" + strings.Repeat("a", i%12+1)}
	}
	balanced, err := balanceDemoPhotoUploaders(reports, append(append([]DemoLostReport(nil), HeroLostReports...), BackgroundLostReports...))
	if err != nil {
		t.Fatal(err)
	}
	counts := make(map[string]int)
	for _, report := range append(append([]DemoLostReport(nil), HeroLostReports...), BackgroundLostReports...) {
		if report.PhotoFilename != "" {
			email := report.OwnerEmail
			if strings.HasPrefix(report.Key, "bg-lost-") {
				email = "petugas@demo.simpanugm.my.id"
			}
			if email == "" {
				email = "anggota@demo.simpanugm.my.id"
			}
			counts[email]++
		}
	}
	for _, report := range append(append([]DemoFoundReport(nil), HeroFoundReports...), BackgroundFoundReports...) {
		if report.PhotoFilename == "" {
			continue
		}
		email := report.FinderEmail
		if strings.HasPrefix(report.Key, "bg-found-") {
			email = "petugas@demo.simpanugm.my.id"
		}
		counts[email]++
	}
	for _, report := range balanced {
		counts[report.FinderEmail]++
		if !strings.HasPrefix(report.Key, "bg-found-photo-") && !strings.HasPrefix(report.Key, "found-photo-") {
			t.Fatalf("unstable report key prefix %q", report.Key)
		}
	}
	for email, count := range counts {
		if count > 30 {
			t.Fatalf("uploader %s exceeds limit: %d", email, count)
		}
	}
}
