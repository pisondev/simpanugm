package claims

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
)

// the semantic bands of V1 (ai-features AI-08)
const (
	semanticHigh   = 0.7
	semanticMedium = 0.4
	lowLocation    = 0.1
	historyDays    = 90
)

type suggestion struct {
	// tinggi, sedang, rendah, periksa or cocok, empty when the item only informs
	Level string `json:"level,omitempty"`
	Text  string `json:"text,omitempty"`
	// codes of other claims on the same item, for V9
	Links []string `json:"links,omitempty"`
}

type item struct {
	key      string
	label    string
	required bool
	infoOnly bool
	sug      suggestion
}

func level(v float64) string {
	switch {
	case v >= semanticHigh:
		return "tinggi"
	case v >= semanticMedium:
		return "sedang"
	default:
		return "rendah"
	}
}

func overlap(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if strings.EqualFold(strings.TrimSpace(x), strings.TrimSpace(y)) {
				return true
			}
		}
	}
	return false
}

func jsonColours(raw []byte) []string {
	var cs []struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(raw, &cs)
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Name)
	}
	return out
}

func jsonBrands(raw []byte) []string {
	var bs []struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(raw, &bs)
	out := make([]string, 0, len(bs))
	for _, b := range bs {
		out = append(out, b.Name)
	}
	return out
}

// builds or refreshes the checklist of AI-08 for a claim, keeping the results an officer already recorded
func (s *Service) buildChecks(ctx context.Context, q *sqlc.Queries, v sqlc.ClaimViewRow, evidence int, now time.Time) error {
	var items []item
	pair, err := q.PairScores(ctx, sqlc.PairScoresParams{LostReportID: v.LostReportID, FoundReportID: v.FoundReportID})
	havePair := err == nil
	if err != nil && !db.IsNoRows(err) {
		return err
	}

	v1 := suggestion{Level: "rendah"}
	if havePair && pair.SSemantic != nil {
		v1.Level = level(*pair.SSemantic)
		v1.Text = fmt.Sprintf("Kemiripan makna jawaban dengan detail tersembunyi: %s.", v1.Level)
	} else {
		v1.Level, v1.Text = "", "Belum ada skor kemiripan, bandingkan jawaban dengan detail tersembunyi secara manual."
	}
	items = append(items, item{key: "V1", label: "Jawaban ciri khusus sesuai dengan detail tersembunyi barang", required: true, sug: v1})

	if v.FoundSensitivity != nil && *v.FoundSensitivity == "identity" {
		sug := suggestion{Level: "periksa", Text: "Nama pada kartu belum cocok otomatis, periksa kartu langsung."}
		if havePair && slices.Contains(pair.Flags, "name_hit") {
			sug = suggestion{Level: "cocok", Text: "Nama pada kartu cocok dengan nama pemilik laporan."}
		}
		items = append(items, item{key: "V2", label: "Nama pada kartu atau dokumen sesuai dengan nama pengklaim", required: true, sug: sug})
	}

	attrs, err := q.FoundMediaAttributes(ctx, &v.FoundReportID)
	if err != nil {
		return err
	}
	var detectedColours, detectedBrands []string
	for _, a := range attrs {
		detectedColours = append(detectedColours, jsonColours(a.Colours)...)
		detectedBrands = append(detectedBrands, jsonBrands(a.Brands)...)
	}
	mismatch := (len(v.LostColours) > 0 && len(detectedColours) > 0 && !overlap(v.LostColours, detectedColours)) ||
		(v.LostBrand != nil && *v.LostBrand != "" && len(detectedBrands) > 0 && !overlap([]string{*v.LostBrand}, detectedBrands))
	v3 := suggestion{Level: "cocok", Text: "Warna dan merek yang disebut tidak bertentangan dengan foto."}
	if mismatch {
		v3 = suggestion{
			Level: "periksa",
			Text: fmt.Sprintf("Disebut: %s. Terdeteksi: %s.", strings.Join(append(slices.Clone(v.LostColours), brandOf(v.LostBrand)...), ", "),
				strings.Join(append(slices.Clone(detectedColours), detectedBrands...), ", ")),
		}
	}
	items = append(items, item{key: "V3", label: "Warna dan merek yang disebut sesuai dengan foto", required: mismatch, sug: v3})

	timeZero := havePair && pair.STime != nil && *pair.STime == 0
	v4 := suggestion{Level: "cocok", Text: "Waktu penemuan berada dalam atau sesudah rentang kehilangan."}
	if timeZero {
		v4 = suggestion{Level: "periksa", Text: "Barang ditemukan sebelum rentang waktu kehilangan yang disebut."}
	}
	items = append(items, item{key: "V4", label: "Waktu kehilangan masuk akal", required: timeZero, sug: v4})

	v5 := suggestion{Level: "cocok", Text: "Lokasi kehilangan dan penemuan berdekatan."}
	if havePair && pair.SLocation != nil && *pair.SLocation < lowLocation {
		v5 = suggestion{Level: "periksa", Text: "Lokasi kehilangan dan penemuan berjauhan."}
	}
	items = append(items, item{key: "V5", label: "Lokasi kehilangan masuk akal", sug: v5})

	var questions []masterdata.Question
	_ = json.Unmarshal(v.VerificationQuestions, &questions)
	for _, qn := range questions {
		if strings.TrimSpace(qn.CheckHint) == "" {
			continue
		}
		items = append(items, item{key: "V6_" + qn.Key, label: qn.CheckHint, required: true, sug: suggestion{Text: qn.Question}})
	}

	if evidence > 0 {
		items = append(items, item{key: "V7", label: "Bukti pendukung relevan", sug: suggestion{Text: fmt.Sprintf("%d gambar bukti dilampirkan.", evidence)}})
	}

	if v.ClaimantID != nil {
		n, err := q.ClaimantRejectedCount90(ctx, sqlc.ClaimantRejectedCount90Params{ClaimantID: v.ClaimantID, DecidedAt: ptrTime(now.AddDate(0, 0, -historyDays))})
		if err != nil {
			return err
		}
		items = append(items, item{key: "V8", label: "Riwayat pengklaim", infoOnly: true, sug: suggestion{Text: fmt.Sprintf("%d klaim ditolak dalam %d hari terakhir.", n, historyDays)}})
	}

	others, err := q.OtherOpenClaimsOfFound(ctx, sqlc.OtherOpenClaimsOfFoundParams{FoundReportID: v.FoundReportID, ID: v.ID})
	if err != nil {
		return err
	}
	var codes []string
	for _, o := range others {
		codes = append(codes, o.Code)
	}
	items = append(items, item{key: "V9", label: "Klaim lain atas barang ini", infoOnly: true, sug: suggestion{Text: fmt.Sprintf("%d klaim lain sedang aktif.", len(others)), Links: codes}})

	keep := make([]string, 0, len(items))
	for i, it := range items {
		raw, _ := json.Marshal(it.sug)
		if err := q.InsertClaimCheck(ctx, sqlc.InsertClaimCheckParams{
			ClaimID: v.ID, ItemKey: it.key, Position: int16(i), Label: it.label, Required: it.required, InfoOnly: it.infoOnly, Suggestion: raw,
		}); err != nil {
			return err
		}
		keep = append(keep, it.key)
	}
	return q.DeleteStaleClaimChecks(ctx, sqlc.DeleteStaleClaimChecksParams{ClaimID: v.ID, Keep: keep})
}

func brandOf(b *string) []string {
	if b == nil || *b == "" {
		return nil
	}
	return []string{*b}
}

func ptrTime(t time.Time) *time.Time { return &t }

var _ = uuid.Nil
