package fuzzy

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

type fixture struct {
	Metric    string  `json:"metric"`
	Threshold float64 `json:"identity_name_threshold"`
	Cases     []struct {
		ID      string  `json:"id"`
		Left    string  `json:"left"`
		Right   string  `json:"right"`
		Ratio   float64 `json:"expected_ratio"`
		NameHit bool    `json:"expected_name_hit"`
	} `json:"cases"`
}

// the shared fixture was produced with rapidfuzz, the Go port must agree to six decimals
func TestTokenSetRatioAgreesWithRapidfuzz(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "data", "fixtures", "fuzzy_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if f.Metric != "rapidfuzz.fuzz.token_set_ratio" || len(f.Cases) == 0 {
		t.Fatalf("unexpected fixture %q with %d cases", f.Metric, len(f.Cases))
	}
	for _, c := range f.Cases {
		got := TokenSetRatio(c.Left, c.Right)
		if math.Abs(got-c.Ratio) > 1e-6 {
			t.Errorf("%s: %f, want %f", c.ID, got, c.Ratio)
		}
		if (got >= f.Threshold) != c.NameHit {
			t.Errorf("%s: name hit %v, want %v", c.ID, got >= f.Threshold, c.NameHit)
		}
	}
}

func TestEdgeCases(t *testing.T) {
	t.Parallel()
	if TokenSetRatio("", "Rani") != 0 || TokenSetRatio("Rani", "  ") != 0 {
		t.Fatal("an empty side scores zero")
	}
	if TokenSetRatio("abc", "xyz") != 0 {
		t.Fatal("nothing in common scores zero")
	}
	if Normalize("  KTM: RANI   Prasetya, 24/543770 ") != "ktm rani prasetya 24 543770" {
		t.Fatalf("normalize: %q", Normalize("  KTM: RANI   Prasetya, 24/543770 "))
	}
}
