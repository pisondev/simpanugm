package privacy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMaskNumbersMatchesTheSharedFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "data", "fixtures", "masking_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Rule  string `json:"rule"`
		Cases []struct {
			ID       string `json:"id"`
			Input    string `json:"input"`
			Expected string `json:"expected"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if f.Rule != "BR-PRV-03" || len(f.Cases) == 0 {
		t.Fatalf("unexpected fixture: rule %q, %d cases", f.Rule, len(f.Cases))
	}
	for _, c := range f.Cases {
		t.Run(c.ID, func(t *testing.T) {
			if got := MaskNumbers(c.Input); got != c.Expected {
				t.Fatalf("MaskNumbers(%q) = %q, want %q", c.Input, got, c.Expected)
			}
		})
	}
}
