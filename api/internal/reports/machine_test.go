package reports

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
)

// every numbered transition in the state machine page has rows here, and no row exists that the page does not number
func TestMachinesMatchTheDocumentedTables(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "requirements", "state-machines.md"))
	if err != nil {
		t.Fatal(err)
	}
	documented := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^\| ([LFC][0-9]+) \|`).FindAllStringSubmatch(string(raw), -1) {
		documented[m[1]] = true
	}
	inCode := map[string]bool{}
	for _, m := range []Machine{Lost, Found, Custody} {
		for _, r := range m.Rows {
			inCode[r.ID] = true
		}
	}
	for id := range documented {
		if !inCode[id] {
			t.Errorf("documented transition %s has no row", id)
		}
	}
	for id := range inCode {
		if !documented[id] {
			t.Errorf("row %s is not documented", id)
		}
	}
}

func TestStep(t *testing.T) {
	cases := []struct {
		m               Machine
		from, event, to string
		want            string
	}{
		{Lost, "", "submit", "reported", "L1"},
		{Lost, "potential_match", "owner_close", "closed", "L10"},
		{Lost, "claimed", "owner_close", "closed", ""},
		{Lost, "closed", "reopen", "reported", "L12"},
		{Found, "reported", "finder_close", "closed", "F13"},
		{Found, "under_verification", "finder_close", "closed", ""},
		{Custody, "with_finder", "receive", "in_custody", "C3"},
		{Custody, "in_custody", "receive", "in_custody", ""},
		{Custody, "in_custody", "returned_directly", "released", ""},
	}
	for _, c := range cases {
		got, err := c.m.Step(c.from, c.event, c.to)
		if c.want == "" {
			if err == nil {
				t.Errorf("%s %s -%s-> %s must be refused", c.m.Name, c.from, c.event, c.to)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("%s %s -%s-> %s = %q, %v; want %s", c.m.Name, c.from, c.event, c.to, got, err, c.want)
		}
	}
}

func TestTargetsAreValidStates(t *testing.T) {
	states := map[string][]string{
		"lost_report":  lostStatuses,
		"found_report": foundStatuses,
		"custody":      {"with_finder", "in_custody", "released", "disposed", "cancelled"},
	}
	for _, m := range []Machine{Lost, Found, Custody} {
		for _, r := range m.Rows {
			if !slices.Contains(states[m.Name], r.To) || (r.From != "" && !slices.Contains(states[m.Name], r.From)) {
				t.Errorf("%s row %s uses an unknown state: %s to %s", m.Name, r.ID, r.From, r.To)
			}
		}
	}
}

func TestCodePattern(t *testing.T) {
	for code, ok := range map[string]bool{"LST-2611-0007": true, "FND-2612-12345": true, "LST-2611-07": false, "lst-2611-0007": false, "XYZ-2611-0007": false} {
		if IsCode(code) != ok {
			t.Errorf("IsCode(%q) != %v", code, ok)
		}
	}
}
