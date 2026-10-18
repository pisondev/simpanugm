package matching

import (
	"fmt"
	"math"
)

// what a reason line needs besides the component scores, stored with the match so a projection never recomputes it.
// It holds public attributes only, never words from private or hidden details or from visible text (BR-VIS-04)
type Facts struct {
	DistanceM     *float64 `json:"distance_m,omitempty"`
	Place         string   `json:"place,omitempty"`
	TimeRelation  string   `json:"time_relation"`
	HoursAfter    float64  `json:"hours_after,omitempty"`
	ColourShared  string   `json:"colour_shared,omitempty"`
	BrandSame     string   `json:"brand_same,omitempty"`
	SameCategory  bool     `json:"same_category"`
	CategoryLost  string   `json:"category_lost"`
	CategoryFound string   `json:"category_found"`
}

type Level string

const (
	LevelHigh   Level = "high"
	LevelMedium Level = "medium"
	LevelLow    Level = "low"
)

// section 11: 0.8 and above reads high, 0.5 and above medium
func LevelOf(s float64) Level {
	switch {
	case s >= 0.8:
		return LevelHigh
	case s >= 0.5:
		return LevelMedium
	default:
		return LevelLow
	}
}

type Line struct {
	Level   *Level
	Reason  string
	Chips   []string
	Missing bool
}

type Explanation struct {
	Visual   Line
	Text     Line
	Location Line
	Time     Line
	Category Line
}

func levelPtr(s *float64) *Level {
	if s == nil {
		return nil
	}
	l := LevelOf(*s)
	return &l
}

func pick(l *Level, high, medium, low string) string {
	switch *l {
	case LevelHigh:
		return high
	case LevelMedium:
		return medium
	default:
		return low
	}
}

// the fixed Indonesian templates of matching-engine section 11
func Explain(mode string, visual, text, location *float64, nameHit bool, f Facts) Explanation {
	var e Explanation
	e.Visual.Level = levelPtr(visual)
	switch {
	case visual == nil:
		e.Visual.Missing = true
		e.Visual.Reason = "Belum ada perbandingan foto."
	case mode == "cross_modal":
		e.Visual.Reason = pick(e.Visual.Level, "Deskripsimu sangat cocok dengan foto barang temuan.", "Deskripsimu cukup cocok dengan foto barang temuan.", "Deskripsimu kurang cocok dengan foto barang temuan.")
	default:
		e.Visual.Reason = pick(e.Visual.Level, "Foto sangat mirip.", "Foto cukup mirip.", "Foto kurang mirip.")
	}

	e.Text.Level = levelPtr(text)
	if text == nil {
		e.Text.Missing = true
		e.Text.Reason = "Belum ada perbandingan deskripsi."
	} else {
		e.Text.Reason = pick(e.Text.Level, "Deskripsi sangat mirip.", "Deskripsi cukup mirip.", "Deskripsi berbeda.")
	}
	if nameHit {
		e.Text.Reason = "Nama pada barang sesuai dengan namamu."
	}
	if f.ColourShared != "" {
		e.Text.Chips = append(e.Text.Chips, "Warna sama: "+f.ColourShared)
	}
	if f.BrandSame != "" {
		e.Text.Chips = append(e.Text.Chips, "Merek sama: "+f.BrandSame)
	}

	e.Location.Level = levelPtr(location)
	switch {
	case location == nil || f.DistanceM == nil:
		e.Location.Missing = true
		e.Location.Reason = "Lokasi kehilangan tidak diketahui."
	case *f.DistanceM < 50:
		e.Location.Reason = "Ditemukan di lokasi yang sama."
	case *f.DistanceM <= 1500:
		e.Location.Reason = fmt.Sprintf("Ditemukan sekitar %d m dari %s.", roundTo50(*f.DistanceM), f.Place)
	default:
		e.Location.Reason = "Lokasi berjauhan."
	}

	switch f.TimeRelation {
	case "in_window":
		e.Time.Reason = "Ditemukan dalam rentang waktu kehilangan."
	case "after_window":
		if f.HoursAfter < 48 {
			e.Time.Reason = fmt.Sprintf("Ditemukan %d jam setelah perkiraan waktu hilang.", max(1, int(math.Round(f.HoursAfter))))
		} else {
			e.Time.Reason = fmt.Sprintf("Ditemukan %d hari setelah perkiraan waktu hilang.", int(math.Round(f.HoursAfter/24)))
		}
	default:
		e.Time.Reason = "Ditemukan sebelum perkiraan waktu hilang."
	}

	if f.SameCategory {
		e.Category.Reason = "Kategori sama."
	} else {
		e.Category.Reason = fmt.Sprintf("Kategori berbeda: %s dan %s.", f.CategoryLost, f.CategoryFound)
	}
	return e
}

func roundTo50(m float64) int {
	return max(50, int(math.Round(m/50))*50)
}
