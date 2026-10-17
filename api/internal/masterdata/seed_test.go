package masterdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	seedDir    = filepath.Join("..", "..", "..", "data", "seed")
	overlayDir = filepath.Join("..", "..", "..", "deploy", "dev-seed")
)

func TestRepositorySeedIsValid(t *testing.T) {
	s, err := LoadSeed(seedDir, overlayDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(s.Zones) != 5 || len(s.Colours) != 14 || len(s.Posts) < 5 || len(s.Locations) < 30 || len(s.Categories) < 10 {
		t.Fatalf("unexpected sizes: zones %d colours %d posts %d locations %d parents %d",
			len(s.Zones), len(s.Colours), len(s.Posts), len(s.Locations), len(s.Categories))
	}
	if s.Settings["custody.retention_days.valuable"] != 60 {
		t.Fatalf("nested settings are flattened to dotted keys, got %v", s.Settings["custody.retention_days.valuable"])
	}
	if _, ok := s.Settings["notify.email_quiet_hours"].(map[string]any); !ok {
		t.Fatal("object settings keep their shape")
	}
}

func TestReviewedSeedWinsOverOverlay(t *testing.T) {
	official := t.TempDir()
	if err := os.WriteFile(filepath.Join(official, "zones.csv"), []byte("slug,name,centroid_lat,centroid_lng\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, _ := LoadSeed(official, overlayDir)
	if s.Sources["zones.csv"] != filepath.Join(official, "zones.csv") {
		t.Fatalf("zones.csv came from %s", s.Sources["zones.csv"])
	}
	if s.Sources["locations.csv"] != filepath.Join(overlayDir, "locations.csv") {
		t.Fatalf("locations.csv came from %s", s.Sources["locations.csv"])
	}
}

func merged(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, src := range []string{overlayDir, seedDir} {
		entries, err := os.ReadDir(src)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			b, err := os.ReadFile(filepath.Join(src, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, e.Name()), b, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	return dir
}

func TestValidateCatchesBrokenSeeds(t *testing.T) {
	cases := []struct {
		name, file, old, new, want string
	}{
		{"unknown zone", "locations.csv", "fmipa,Fakultas MIPA,saintek", "fmipa,Fakultas MIPA,nowhere", "unknown zone"},
		{"alias repeats the name", "locations.csv", "fmipa,Fakultas MIPA,saintek,building,-7.767359,110.376550,mipa;fmipa", "fmipa,Fakultas MIPA,saintek,building,-7.767359,110.376550,mipa;Fakultas MIPA", "repeats the name"},
		{"leaf without valid sensitivity", "categories.yaml", "sensitivity: valuable", "sensitivity: precious", "sensitivity must be"},
		{"compatibility with unknown leaf", "category_compatibility.csv", "dompet,pouch,0.70", "dompet,koper,0.70", "both must be distinct leaf categories"},
		{"bad hex", "colours.csv", "hitam,#111827,1", "hitam,111827,1", "hex like"},
		{"post at unknown location", "posts.yaml", "location_slug: balairung", "location_slug: menara-gading", "unknown location"},
		{"post missing a weekday", "posts.yaml", "    sun: []\n", "", "must list sun"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := merged(t)
			path := filepath.Join(dir, c.file)
			b, _ := os.ReadFile(path)
			if !strings.Contains(string(b), c.old) {
				t.Fatalf("fixture text %q not found in %s", c.old, c.file)
			}
			_ = os.WriteFile(path, []byte(strings.Replace(string(b), c.old, c.new, 1)), 0o600)
			s, err := LoadSeed(dir)
			if err == nil {
				err = s.Validate()
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want error containing %q, got %v", c.want, err)
			}
		})
	}
}

func TestBoundingBoxIsCheckedWhenConfigured(t *testing.T) {
	s, err := LoadSeed(seedDir, overlayDir)
	if err != nil {
		t.Fatal(err)
	}
	s.Settings["campus.bounding_box"] = map[string]any{"min_lat": -7.70, "max_lat": -7.60, "min_lng": 110.30, "max_lng": 110.40}
	if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "outside the campus bounding box") {
		t.Fatalf("expected bbox failures, got %v", err)
	}
}
