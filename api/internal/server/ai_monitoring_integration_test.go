//go:build integration

package server_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/testenv"
)

func TestAIMonitoringViews(t *testing.T) {
	ctx := context.Background()
	e := testenv.New(t)

	// Fetch active fusion version and categories for valid test rows
	var fusionID uuid.UUID
	if err := e.Pool.QueryRow(ctx, "SELECT id FROM model_versions WHERE component = 'fusion' AND status = 'active' LIMIT 1").Scan(&fusionID); err != nil {
		t.Fatalf("failed to get active fusion version: %v", err)
	}

	var catID uuid.UUID
	if err := e.Pool.QueryRow(ctx, "SELECT id FROM categories WHERE parent_id IS NOT NULL LIMIT 1").Scan(&catID); err != nil {
		t.Fatalf("failed to get leaf category: %v", err)
	}

	now := time.Now()
	// Insert lost report with assist_outcome
	lostID := uuid.New()
	assistOutcomeJSON, _ := json.Marshal(map[string]string{
		"category": "accepted",
		"colours":  "edited",
		"brand":    "ignored",
	})
	_, err := e.Pool.Exec(ctx, `
		INSERT INTO lost_reports (
			id, code, proxy_name, category_id, item_name, description, lost_from, lost_until,
			status, assist_outcome, last_activity_at, created_at, updated_at
		) VALUES (
			$1, $2, 'Budi Santoso', $3, 'Dompet Cokelat', 'Dompet kulit warna cokelat hilang di kantin fakultas', $4, $4,
			'reported', $5, $4, $4, $4
		)
	`, lostID, "L-TEST-001", catID, now, assistOutcomeJSON)
	if err != nil {
		t.Fatalf("insert lost report: %v", err)
	}

	var locID uuid.UUID
	if err := e.Pool.QueryRow(ctx, "SELECT id FROM locations LIMIT 1").Scan(&locID); err != nil {
		t.Fatalf("failed to get location: %v", err)
	}

	// Insert found report
	foundID1 := uuid.New()
	foundID2 := uuid.New()
	for i, fID := range []uuid.UUID{foundID1, foundID2} {
		code := "F-TEST-001"
		if i == 1 {
			code = "F-TEST-002"
		}
		_, err := e.Pool.Exec(ctx, `
			INSERT INTO found_reports (
				id, code, category_id, found_location_id, item_name, description, found_at,
				status, custody_status, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, 'Dompet Ditemukan', 'Ditemukan di meja', $5,
				'reported', 'with_finder', $5, $5
			)
		`, fID, code, catID, locID, now)
		if err != nil {
			t.Fatalf("insert found report: %v", err)
		}
	}

	// Insert match 1: confirmed, tier high, rank 1, score 0.88
	matchID1 := uuid.New()
	_, err = e.Pool.Exec(ctx, `
		INSERT INTO matches (
			id, lost_report_id, found_report_id, visual_mode, category_factor,
			overall, tier, rank_for_lost, rank_for_found, status,
			fusion_version_id, first_scored_at, scored_at
		) VALUES (
			$1, $2, $3, 'none', 1.0,
			0.8800, 'high', 1, 1, 'confirmed',
			$4, $5, $5
		)
	`, matchID1, lostID, foundID1, fusionID, now)
	if err != nil {
		t.Fatalf("insert match 1: %v", err)
	}

	// Insert match 2: dismissed by owner, tier medium, rank 2, score 0.65
	matchID2 := uuid.New()

	t.Cleanup(func() {
		_, _ = e.Pool.Exec(context.Background(), "DELETE FROM matches WHERE id IN ($1, $2)", matchID1, matchID2)
		_, _ = e.Pool.Exec(context.Background(), "DELETE FROM lost_reports WHERE id = $1", lostID)
		_, _ = e.Pool.Exec(context.Background(), "DELETE FROM found_reports WHERE id IN ($1, $2)", foundID1, foundID2)
	})

	_, err = e.Pool.Exec(ctx, `
		INSERT INTO matches (
			id, lost_report_id, found_report_id, visual_mode, category_factor,
			overall, tier, rank_for_lost, rank_for_found, status,
			owner_dismissed_at, fusion_version_id, first_scored_at, scored_at
		) VALUES (
			$1, $2, $3, 'none', 1.0,
			0.6500, 'medium', 2, 1, 'active',
			$4, $5, $4, $4
		)
	`, matchID2, lostID, foundID2, now, fusionID)
	if err != nil {
		t.Fatalf("insert match 2: %v", err)
	}

	// 1. Verify single unified query on ai_monitoring_summary
	var assistOutcomesRaw, dismissRatesRaw, confirmedRanksRaw, top1HistRaw []byte
	err = e.Pool.QueryRow(ctx, `
		SELECT assist_outcomes, dismiss_rates, confirmed_ranks, top1_histogram
		FROM ai_monitoring_summary
	`).Scan(&assistOutcomesRaw, &dismissRatesRaw, &confirmedRanksRaw, &top1HistRaw)
	if err != nil {
		t.Fatalf("query ai_monitoring_summary: %v", err)
	}

	var assistOutcomes []map[string]any
	if err := json.Unmarshal(assistOutcomesRaw, &assistOutcomes); err != nil {
		t.Fatalf("decode assist_outcomes: %v", err)
	}
	if len(assistOutcomes) < 3 {
		t.Errorf("expected at least 3 assist outcome entries, got %d", len(assistOutcomes))
	}

	var dismissRates []map[string]any
	if err := json.Unmarshal(dismissRatesRaw, &dismissRates); err != nil {
		t.Fatalf("decode dismiss_rates: %v", err)
	}
	foundMedium := false
	for _, dr := range dismissRates {
		if dr["tier"] == "medium" {
			foundMedium = true
			if dr["owner_dismissed"].(float64) < 1 {
				t.Errorf("expected medium owner_dismissed >= 1, got %v", dr["owner_dismissed"])
			}
		}
	}
	if !foundMedium {
		t.Errorf("expected medium tier in dismiss_rates, got %v", dismissRates)
	}

	var confirmedRanks map[string]any
	if err := json.Unmarshal(confirmedRanksRaw, &confirmedRanks); err != nil {
		t.Fatalf("decode confirmed_ranks: %v", err)
	}
	if confirmedRanks["confirmed_total"].(float64) < 1 {
		t.Errorf("expected confirmed_total >= 1, got %v", confirmedRanks["confirmed_total"])
	}
	if confirmedRanks["rank_1_count"].(float64) < 1 {
		t.Errorf("expected rank_1_count >= 1, got %v", confirmedRanks["rank_1_count"])
	}
	if confirmedRanks["recall_at_1"].(float64) != 1.0 {
		t.Errorf("expected recall_at_1 = 1.0, got %v", confirmedRanks["recall_at_1"])
	}

	var top1Hist []map[string]any
	if err := json.Unmarshal(top1HistRaw, &top1Hist); err != nil {
		t.Fatalf("decode top1_histogram: %v", err)
	}
	if len(top1Hist) == 0 {
		t.Errorf("expected non-empty top1_histogram buckets")
	}

	// 2. Also verify individual views can be queried directly
	var assistCount int
	if err := e.Pool.QueryRow(ctx, "SELECT count(*) FROM ai_monitoring_assist_outcomes").Scan(&assistCount); err != nil || assistCount < 3 {
		t.Errorf("direct query on ai_monitoring_assist_outcomes failed or empty: %v, count=%d", err, assistCount)
	}

	var confirmedCount int
	if err := e.Pool.QueryRow(ctx, "SELECT confirmed_total FROM ai_monitoring_confirmed_ranks").Scan(&confirmedCount); err != nil || confirmedCount < 1 {
		t.Errorf("direct query on ai_monitoring_confirmed_ranks failed or empty: %v, count=%d", err, confirmedCount)
	}
}
