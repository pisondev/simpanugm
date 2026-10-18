package fusion

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fixtureCase struct {
	ID       string  `json:"id"`
	Mode     Mode    `json:"mode"`
	Category float64 `json:"category_factor"`
	Visual   *struct {
		Image      *float64 `json:"image_cosine"`
		CrossModal *float64 `json:"cross_modal_cosine"`
	} `json:"visual"`
	Text struct {
		Semantic float64 `json:"semantic_cosine"`
		Colour   string  `json:"colour"`
		Brand    string  `json:"brand"`
	} `json:"text"`
	Location *struct {
		DistanceM float64 `json:"distance_m"`
	} `json:"location"`
	Time struct {
		Relation   string  `json:"relation"`
		HoursAfter float64 `json:"hours_after_window"`
		HoursFrom  float64 `json:"hours_before_start"`
	} `json:"time"`
	Expected struct {
		Visual     *float64 `json:"visual"`
		Semantic   *float64 `json:"semantic"`
		Attributes *float64 `json:"attributes"`
		Text       *float64 `json:"text"`
		Location   *float64 `json:"location"`
		Time       float64  `json:"time"`
		Fused      float64  `json:"fused"`
		Stored     float64  `json:"stored_score"`
		Tier       string   `json:"tier"`
	} `json:"expected"`
}

var colourOf = map[string]ColourRelation{"shared": ColourShared, "adjacent": ColourAdjacent, "different": ColourOther, "missing": ColourMissing}
var brandOf = map[string]BrandRelation{"same": BrandSame, "different": BrandDifferent, "missing": BrandMissing}

func near(t *testing.T, id, name string, got *float64, want *float64) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Errorf("%s %s: got %v, want %v", id, name, got, want)
		return
	}
	if got != nil && math.Abs(*got-*want) > 1e-6 {
		t.Errorf("%s %s: %f, want %f", id, name, *got, *want)
	}
}

// matching-engine sections 6 and 7 against the shared fixture, six decimals as the Python reference requires
func TestScoreAgreesWithTheSharedFixture(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "data", "fixtures", "fusion_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Version string        `json:"fusion_version"`
		Cases   []fixtureCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	cfg, _ := Default()
	if f.Version != cfg.ID {
		t.Fatalf("fixture targets %s, the embedded configuration is %s", f.Version, cfg.ID)
	}
	lostFrom := time.Date(2026, 11, 2, 1, 0, 0, 0, time.UTC)
	lostUntil := lostFrom.Add(4 * time.Hour)
	for _, c := range f.Cases {
		in := Input{Mode: c.Mode, Category: c.Category, LostFrom: lostFrom, LostUntil: lostUntil,
			Colour: colourOf[c.Text.Colour], Brand: brandOf[c.Text.Brand]}
		sem := c.Text.Semantic
		in.SemanticCosine = &sem
		if c.Visual != nil {
			if c.Visual.Image != nil {
				in.VisualCosine = c.Visual.Image
			} else {
				in.VisualCosine = c.Visual.CrossModal
			}
		}
		if c.Location != nil {
			l := cfg.LocationTerm(c.Location.DistanceM, false)
			in.LocationScore = &l
		}
		switch c.Time.Relation {
		case "in_window":
			in.FoundAt = lostUntil
		case "after_window":
			in.FoundAt = lostUntil.Add(time.Duration(c.Time.HoursAfter * float64(time.Hour)))
		case "before_window":
			in.FoundAt = lostFrom.Add(-time.Duration(c.Time.HoursFrom * float64(time.Hour)))
		default:
			t.Fatalf("%s: unknown time relation %q", c.ID, c.Time.Relation)
		}
		r := cfg.Score(in)
		near(t, c.ID, "visual", r.Visual, c.Expected.Visual)
		near(t, c.ID, "semantic", r.Semantic, c.Expected.Semantic)
		near(t, c.ID, "attributes", r.Attributes, c.Expected.Attributes)
		near(t, c.ID, "text", r.Text, c.Expected.Text)
		near(t, c.ID, "location", r.Location, c.Expected.Location)
		tm, fused, stored := r.Time, r.Fused, r.Overall
		near(t, c.ID, "time", &tm, &c.Expected.Time)
		near(t, c.ID, "fused", &fused, &c.Expected.Fused)
		near(t, c.ID, "stored", &stored, &c.Expected.Stored)
		if r.Tier != c.Expected.Tier {
			t.Errorf("%s tier: %s, want %s", c.ID, r.Tier, c.Expected.Tier)
		}
	}
}

// hand computed examples of US-19 criterion 1, four decimals
func TestHandComputedExamples(t *testing.T) {
	t.Parallel()
	cfg, _ := Default()
	from := time.Date(2026, 11, 2, 1, 0, 0, 0, time.UTC)
	until := from.Add(2 * time.Hour)
	// no photo, no embeddings, same leaf, found 150 m away inside the window: location and time only
	loc := cfg.LocationTerm(150, false)
	r := cfg.Score(Input{Mode: ModeNone, Category: 1, LocationScore: &loc, FoundAt: until, LostFrom: from, LostUntil: until})
	want := math.Round((0.15*math.Exp(-150.0/400)+0.15*1)/0.30*10000) / 10000
	if r.Overall != want || r.Tier != "high" || r.Visual != nil || r.Text != nil {
		t.Fatalf("structured only: %+v, want %f", r, want)
	}
	// a zone only place is capped at 0.6 even at the centroid
	if got := cfg.LocationTerm(0, true); got != 0.6 {
		t.Fatalf("zone cap: %f", got)
	}
	// identity card naming the owner: text lifted to 0.95 and flagged
	sem := 0.70
	r = cfg.Score(Input{Mode: ModeNone, Category: 1, SemanticCosine: &sem, NameHit: true, FoundAt: until, LostFrom: from, LostUntil: until})
	if r.Text == nil || *r.Text != 0.95 || !r.NameHit {
		t.Fatalf("name hit: %+v", r)
	}
}

func TestTierBoundaries(t *testing.T) {
	t.Parallel()
	cfg, _ := Default()
	for overall, tier := range map[float64]string{0.72: "high", 0.7199: "medium", 0.55: "medium", 0.5499: "low", 0.40: "low", 0.3999: NotStored} {
		if got := cfg.TierOf(overall); got != tier {
			t.Errorf("%f: %s, want %s", overall, got, tier)
		}
	}
}

func TestColourAndBrandRelations(t *testing.T) {
	t.Parallel()
	if Colours([]string{"Hitam"}, []string{"hitam", "biru"}) != ColourShared ||
		Colours([]string{"hitam"}, []string{"abu-abu"}) != ColourAdjacent ||
		Colours([]string{"merah"}, []string{"biru"}) != ColourOther ||
		Colours(nil, []string{"biru"}) != ColourMissing {
		t.Fatal("colour relations")
	}
	if Brands(" Eiger", "eiger ") != BrandSame || Brands("Eiger", "Consina") != BrandDifferent || Brands("", "Eiger") != BrandMissing {
		t.Fatal("brand relations")
	}
	cfg, _ := Default()
	compat := 0.70
	if cfg.CategoryFactor(false, true, &compat) != 0.85 || cfg.CategoryFactor(false, false, &compat) != 0.70 || cfg.CategoryFactor(false, false, nil) != 0.35 {
		t.Fatal("category factor")
	}
}
