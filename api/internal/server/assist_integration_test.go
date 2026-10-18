//go:build integration

package server_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

type suggestions struct {
	Categories []struct {
		Category struct {
			Slug string `json:"slug"`
		} `json:"category"`
		Score float64 `json:"score"`
	} `json:"categories"`
	Colours     []string `json:"colours"`
	Brand       *string  `json:"brand"`
	VisibleText *string  `json:"visible_text"`
}

// US-23 and FR-ASSIST-01 to 05: suggestions from a staged photo, masked text, a note instead of an error when ml is down
func TestAssistantSuggestsFromAPhoto(t *testing.T) {
	ml := testenv.NewFakeML(t)
	e := testenv.New(t, testenv.WithML(ml.URL))
	user := e.User(access.Member)
	other := e.User(access.Member)
	media := user.Client.UploadID("found_report")
	body := map[string]any{"media_id": media, "report_type": "found"}

	ml.Analysis(map[string]any{
		"categories": []map[string]any{
			{"slug": "tumbler", "score": 0.71}, {"slug": "payung", "score": 0.12}, {"slug": "tidak-ada-di-database", "score": 0.1},
			{"slug": "dompet", "score": 0.05}, {"slug": "ktm", "score": 0.02},
		},
		"colours": []map[string]any{{"name": "biru", "hex": "#1E3A8A", "share": 0.6}, {"name": "ungu-neon", "hex": "#FF00FF", "share": 0.2}, {"name": "perak", "hex": "#C0C0C0", "share": 0.1}},
		"brands":  []map[string]any{{"name": "Corkcicle", "score": 0.93}, {"name": "Lain", "score": 0.4}},
	})
	ml.ShowText("KTM 4111 1111 1111 1111")
	res := user.Client.JSON(http.MethodPost, "/api/v1/assist/analyze-image", body)
	if res.Status != http.StatusOK {
		t.Fatalf("analyze: %d %s", res.Status, res.Body)
	}
	var s suggestions
	res.Decode(t, &s)
	if len(s.Categories) != 3 || s.Categories[0].Category.Slug != "tumbler" || s.Categories[1].Category.Slug != "payung" || s.Categories[2].Category.Slug != "dompet" {
		t.Fatalf("categories keep ml order, drop unknown slugs and stop at three: %+v", s.Categories)
	}
	if strings.Join(s.Colours, ",") != "biru,perak" || s.Brand == nil || *s.Brand != "Corkcicle" {
		t.Fatalf("colours and brand: %+v", s)
	}
	if s.VisibleText == nil || strings.Contains(*s.VisibleText, "4111") || !strings.HasSuffix(*s.VisibleText, "1111") {
		t.Fatalf("the card number is masked: %v", s.VisibleText)
	}

	if r := other.Client.JSON(http.MethodPost, "/api/v1/assist/analyze-image", body); r.Status != http.StatusNotFound {
		t.Fatalf("another user's photo: %d", r.Status)
	}
	if r := user.Client.JSON(http.MethodPost, "/api/v1/assist/analyze-image", map[string]any{"media_id": media, "report_type": "lost"}); r.Status != http.StatusNotFound {
		t.Fatalf("a photo staged for the other report type: %d", r.Status)
	}
	if r := user.Client.JSON(http.MethodPost, "/api/v1/assist/analyze-image", map[string]any{"media_id": uuid.New(), "report_type": "found"}); r.Status != http.StatusNotFound {
		t.Fatalf("unknown photo: %d", r.Status)
	}

	for _, status := range []int{http.StatusServiceUnavailable, http.StatusNotImplemented} {
		ml.Fail(status)
		if r := user.Client.JSON(http.MethodPost, "/api/v1/assist/analyze-image", body); r.Status != http.StatusServiceUnavailable || r.Problem(t).Code != "assist_unavailable" {
			t.Fatalf("ml answering %d: %d %s", status, r.Status, r.Body)
		}
	}
	ml.Fail(0)

	// without an inference service at all the assistant answers the same way
	bare := testenv.New(t)
	member := bare.User(access.Member)
	if r := member.Client.JSON(http.MethodPost, "/api/v1/assist/analyze-image", map[string]any{"media_id": member.Client.UploadID("lost_report"), "report_type": "lost"}); r.Problem(t).Code != "assist_unavailable" {
		t.Fatalf("no ml configured: %d %s", r.Status, r.Body)
	}
}

// AI-04: the draft is the template, and an identity card never contributes its text
func TestAssistantDraftsADescription(t *testing.T) {
	e := testenv.New(t)
	member := e.User(access.Member)
	draft := func(body map[string]any) string {
		res := member.Client.JSON(http.MethodPost, "/api/v1/assist/describe", body)
		if res.Status != http.StatusOK {
			t.Fatalf("describe: %d %s", res.Status, res.Body)
		}
		var d struct {
			Description string `json:"description"`
		}
		res.Decode(t, &d)
		return d.Description
	}
	if got := draft(map[string]any{"category_id": e.Category("payung"), "colours": []string{"biru"}}); got != "Payung berwarna biru." {
		t.Fatalf("US-24 example: %q", got)
	}
	if got := draft(map[string]any{"category_id": e.Category("tumbler"), "colours": []string{"biru", "perak"}, "brand": "Corkcicle", "visible_text": "RANI"}); got != `Tumbler berwarna biru dan perak, merek Corkcicle. Terdapat tulisan "RANI".` {
		t.Fatalf("full template: %q", got)
	}
	if got := draft(map[string]any{"category_id": e.Category("ktm"), "visible_text": "RANI PRASETYA"}); strings.Contains(got, "RANI") {
		t.Fatalf("identity text stays out: %q", got)
	}
	if r := member.Client.JSON(http.MethodPost, "/api/v1/assist/describe", map[string]any{"category_id": uuid.New()}); !fieldError(t, r, "category_id") {
		t.Fatalf("unknown category: %d %s", r.Status, r.Body)
	}
}

// FR-ASSIST-02: what the reporter did with each suggestion is kept on the report for AI-12
func TestAssistOutcomeIsStoredWithTheReport(t *testing.T) {
	e := isolated(t)
	owner := e.User(access.Member)
	body := lostBody(e, e.Clock.Now(), nil)
	body["assist_outcome"] = map[string]any{"category": "accepted", "colours": "edited", "brand": "ignored"}
	res := owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports", body, "Idempotency-Key", "assist-outcome-0001")
	if res.Status != http.StatusCreated {
		t.Fatalf("create: %d %s", res.Status, res.Body)
	}
	var r lostReport
	res.Decode(t, &r)
	var colours string
	if err := e.Pool.QueryRow(context.Background(), "SELECT assist_outcome->>'colours' FROM lost_reports WHERE id = $1", r.ID).Scan(&colours); err != nil || colours != "edited" {
		t.Fatalf("stored outcome: %q %v", colours, err)
	}
	body["assist_outcome"] = map[string]any{"category": "guessed"}
	if r := owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports", body, "Idempotency-Key", "assist-outcome-0002"); r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
		t.Fatalf("an unknown outcome value is refused: %d", r.Status)
	}
}
