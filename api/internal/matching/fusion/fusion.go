// Package fusion holds the scoring of matching-engine sections 6 and 7 as pure functions over one versioned configuration
package fusion

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

//go:embed fusion-linear-2026.11.0.json
var defaultDocument []byte

// the configuration of section 7.1, every number of sections 5 to 7 lives here and nowhere else
type Config struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Weights struct {
		VisualImage      float64 `json:"visual_image"`
		VisualCrossModal float64 `json:"visual_cross_modal"`
		Text             float64 `json:"text"`
		Location         float64 `json:"location"`
		Time             float64 `json:"time"`
	} `json:"weights"`
	Bands struct {
		Image      [2]float64 `json:"image"`
		CrossModal [2]float64 `json:"cross_modal"`
		E5         [2]float64 `json:"e5"`
	} `json:"bands"`
	TextMix struct {
		Semantic   float64 `json:"semantic"`
		Attributes float64 `json:"attributes"`
	} `json:"text_mix"`
	ColourPartial float64 `json:"colour_partial"`
	Location      struct {
		ScaleM  float64 `json:"scale_m"`
		ZoneCap float64 `json:"zone_cap"`
	} `json:"location"`
	Time struct {
		ToleranceBeforeH float64 `json:"tolerance_before_h"`
		HalfLifeH        float64 `json:"half_life_h"`
		PoolBeforeH      float64 `json:"pool_before_h"`
		PoolAfterD       float64 `json:"pool_after_d"`
	} `json:"time"`
	Category struct {
		SameParent float64 `json:"same_parent"`
		Other      float64 `json:"other"`
	} `json:"category"`
	IdentityNameHit struct {
		MinRatio  float64 `json:"min_ratio"`
		TextFloor float64 `json:"text_floor"`
	} `json:"identity_name_hit"`
	Tiers struct {
		High   float64 `json:"high"`
		Medium float64 `json:"medium"`
		Floor  float64 `json:"floor"`
	} `json:"tiers"`
	TopK int `json:"top_k"`
	AnnK int `json:"ann_k"`
}

func Default() (Config, []byte) {
	c, err := Parse(defaultDocument)
	if err != nil {
		panic(err)
	}
	return c, defaultDocument
}

func Parse(doc []byte) (Config, error) {
	var c Config
	if err := json.Unmarshal(doc, &c); err != nil {
		return c, fmt.Errorf("fusion: %w", err)
	}
	if c.ID == "" || c.Kind != "weighted_mean" || c.TopK < 1 || c.AnnK < 1 {
		return c, errors.New("fusion: id, kind weighted_mean, top_k and ann_k are required")
	}
	if !(c.Tiers.Floor < c.Tiers.Medium && c.Tiers.Medium < c.Tiers.High) {
		return c, errors.New("fusion: tiers must rise from floor to medium to high")
	}
	return c, nil
}

func Digest(doc []byte) []byte {
	sum := sha256.Sum256(doc)
	return sum[:]
}

type Mode string

const (
	ModeImage      Mode = "image"
	ModeCrossModal Mode = "cross_modal"
	ModeNone       Mode = "none"
)

// how two colour lists relate, section 6.2
type ColourRelation string

const (
	ColourShared   ColourRelation = "shared"
	ColourAdjacent ColourRelation = "adjacent"
	ColourOther    ColourRelation = "different"
	ColourMissing  ColourRelation = "missing"
)

type BrandRelation string

const (
	BrandSame      BrandRelation = "same"
	BrandDifferent BrandRelation = "different"
	BrandMissing   BrandRelation = "missing"
)

// raw inputs of one pair, a nil pointer is a missing signal and is left out of the mean
type Input struct {
	Mode           Mode
	VisualCosine   *float64
	SemanticCosine *float64
	Colour         ColourRelation
	Brand          BrandRelation
	NameHit        bool
	// the largest exp term over the lost places, already capped for a zone only place
	LocationScore *float64
	FoundAt       time.Time
	LostFrom      time.Time
	LostUntil     time.Time
	Category      float64
}

type Result struct {
	Mode       Mode
	Visual     *float64
	Semantic   *float64
	Attributes *float64
	Text       *float64
	Location   *float64
	Time       float64
	Category   float64
	Fused      float64
	// rounded to four decimals, the value stored and compared with the tiers
	Overall float64
	Tier    string
	NameHit bool
}

const NotStored = "not_stored"

func Norm(x float64, band [2]float64) float64 {
	lo, hi := band[0], band[1]
	if hi <= lo {
		return 0
	}
	return math.Min(1, math.Max(0, (x-lo)/(hi-lo)))
}

func (c Config) LocationTerm(distanceM float64, zoneOnly bool) float64 {
	s := math.Exp(-distanceM / c.Location.ScaleM)
	if zoneOnly {
		s = math.Min(s, c.Location.ZoneCap)
	}
	return s
}

// section 6.4
func (c Config) TimeScore(foundAt, lostFrom, lostUntil time.Time) float64 {
	tolerance := time.Duration(c.Time.ToleranceBeforeH * float64(time.Hour))
	switch {
	case foundAt.Before(lostFrom.Add(-tolerance)):
		return 0
	case !foundAt.After(lostUntil):
		return 1
	default:
		hours := foundAt.Sub(lostUntil).Hours()
		return math.Pow(0.5, hours/c.Time.HalfLifeH)
	}
}

func (c Config) attributes(in Input) *float64 {
	var sum float64
	n := 0
	switch in.Colour {
	case ColourShared:
		sum, n = sum+1, n+1
	case ColourAdjacent:
		sum, n = sum+c.ColourPartial, n+1
	case ColourOther:
		n++
	}
	switch in.Brand {
	case BrandSame:
		sum, n = sum+1, n+1
	case BrandDifferent:
		n++
	}
	if n == 0 {
		return nil
	}
	v := sum / float64(n)
	return &v
}

// sections 6 and 7 for one pair
func (c Config) Score(in Input) Result {
	r := Result{Mode: in.Mode, Category: in.Category, NameHit: in.NameHit}
	var weightSum, weighted float64
	add := func(w, s float64) {
		weightSum += w
		weighted += w * s
	}
	if in.VisualCosine != nil && in.Mode != ModeNone {
		band, w := c.Bands.Image, c.Weights.VisualImage
		if in.Mode == ModeCrossModal {
			band, w = c.Bands.CrossModal, c.Weights.VisualCrossModal
		}
		v := Norm(*in.VisualCosine, band)
		r.Visual = &v
		add(w, v)
	} else {
		r.Mode = ModeNone
	}
	r.Attributes = c.attributes(in)
	if in.SemanticCosine != nil {
		sem := Norm(*in.SemanticCosine, c.Bands.E5)
		r.Semantic = &sem
		text := sem
		if r.Attributes != nil {
			text = c.TextMix.Semantic*sem + c.TextMix.Attributes**r.Attributes
		}
		r.Text = &text
	} else if r.Attributes != nil {
		// without the sentence encoder the attributes alone stand for the text signal
		text := *r.Attributes
		r.Text = &text
	}
	// the identity name rule of section 6.2, a card that names the owner lifts the text signal
	if floor := c.IdentityNameHit.TextFloor; in.NameHit && (r.Text == nil || *r.Text < floor) {
		r.Text = &floor
	}
	if r.Text != nil {
		add(c.Weights.Text, *r.Text)
	}
	if in.LocationScore != nil {
		l := *in.LocationScore
		r.Location = &l
		add(c.Weights.Location, l)
	}
	r.Time = c.TimeScore(in.FoundAt, in.LostFrom, in.LostUntil)
	add(c.Weights.Time, r.Time)
	if weightSum > 0 {
		r.Fused = in.Category * weighted / weightSum
	}
	r.Overall = math.Round(r.Fused*10000) / 10000
	r.Tier = c.TierOf(r.Overall)
	return r
}

func (c Config) TierOf(overall float64) string {
	switch {
	case overall >= c.Tiers.High:
		return "high"
	case overall >= c.Tiers.Medium:
		return "medium"
	case overall >= c.Tiers.Floor:
		return "low"
	default:
		return NotStored
	}
}

// the palette adjacency list of section 6.2, symmetric
var adjacent = map[[2]string]bool{}

func init() {
	pairs := [][2]string{
		{"hitam", "abu-abu"}, {"abu-abu", "perak"}, {"perak", "putih"}, {"putih", "krem"}, {"krem", "cokelat"},
		{"cokelat", "oranye"}, {"oranye", "merah"}, {"merah", "merah muda"}, {"oranye", "kuning"}, {"kuning", "emas"},
		{"emas", "cokelat"}, {"biru", "ungu"}, {"biru", "hijau"},
	}
	for _, p := range pairs {
		adjacent[p] = true
		adjacent[[2]string{p[1], p[0]}] = true
	}
}

func Colours(a, b []string) ColourRelation {
	if len(a) == 0 || len(b) == 0 {
		return ColourMissing
	}
	rel := ColourOther
	for _, x := range a {
		for _, y := range b {
			x, y := strings.ToLower(strings.TrimSpace(x)), strings.ToLower(strings.TrimSpace(y))
			if x == y {
				return ColourShared
			}
			if adjacent[[2]string{x, y}] {
				rel = ColourAdjacent
			}
		}
	}
	return rel
}

func Brands(a, b string) BrandRelation {
	a, b = strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b))
	switch {
	case a == "" || b == "":
		return BrandMissing
	case a == b:
		return BrandSame
	default:
		return BrandDifferent
	}
}

// section 6.5, a listed compatible pair and the same parent both apply, the larger wins
func (c Config) CategoryFactor(sameLeaf, sameParent bool, compatible *float64) float64 {
	if sameLeaf {
		return 1
	}
	g := c.Category.Other
	if sameParent {
		g = c.Category.SameParent
	}
	if compatible != nil && *compatible > g {
		g = *compatible
	}
	return g
}
