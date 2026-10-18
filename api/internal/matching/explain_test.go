package matching

import (
	"strings"
	"testing"
)

func f64(v float64) *float64 { return &v }

// US-20 criteria 2 and 3, with the reason lines of matching-engine section 11
func TestReasonLines(t *testing.T) {
	t.Parallel()
	e := Explain("cross_modal", f64(0.85), f64(0.6), f64(0.7), false, Facts{
		DistanceM: f64(137), Place: "Perpustakaan Pusat UGM", TimeRelation: "after_window", HoursAfter: 5,
		ColourShared: "hitam", BrandSame: "Eiger", SameCategory: true,
	})
	if e.Visual.Reason != "Deskripsimu sangat cocok dengan foto barang temuan." {
		t.Fatalf("visual: %s", e.Visual.Reason)
	}
	if e.Location.Reason != "Ditemukan sekitar 150 m dari Perpustakaan Pusat UGM." {
		t.Fatalf("location: %s", e.Location.Reason)
	}
	if e.Time.Reason != "Ditemukan 5 jam setelah perkiraan waktu hilang." {
		t.Fatalf("time: %s", e.Time.Reason)
	}
	if strings.Join(e.Text.Chips, "|") != "Warna sama: hitam|Merek sama: Eiger" || e.Text.Reason != "Deskripsi cukup mirip." {
		t.Fatalf("text: %+v", e.Text)
	}
	if e.Category.Reason != "Kategori sama." {
		t.Fatalf("category: %s", e.Category.Reason)
	}

	e = Explain("none", nil, nil, nil, true, Facts{TimeRelation: "after_window", HoursAfter: 80, CategoryLost: "Dompet", CategoryFound: "Pouch"})
	if !e.Visual.Missing || !e.Location.Missing || e.Location.Reason != "Lokasi kehilangan tidak diketahui." {
		t.Fatalf("missing signals: %+v", e)
	}
	if e.Text.Reason != "Nama pada barang sesuai dengan namamu." {
		t.Fatalf("name hit: %s", e.Text.Reason)
	}
	if e.Time.Reason != "Ditemukan 3 hari setelah perkiraan waktu hilang." || e.Category.Reason != "Kategori berbeda: Dompet dan Pouch." {
		t.Fatalf("time %q category %q", e.Time.Reason, e.Category.Reason)
	}
	if LevelOf(0.8) != LevelHigh || LevelOf(0.79) != LevelMedium || LevelOf(0.49) != LevelLow {
		t.Fatal("levels")
	}
}
