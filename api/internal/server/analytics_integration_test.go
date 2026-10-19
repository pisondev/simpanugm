//go:build integration

package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/openapi"
)

func TestAnalyticsAndAIMonitoringFlows(t *testing.T) {
	e := isolated(t)
	admin := e.User(access.Admin)
	officer := e.User(access.Officer, e.Post("pos-perpustakaan"))
	member := e.User(access.Member)

	// US-49: Model Versions & Monitoring
	t.Run("ModelVersionsAndMonitoring", func(t *testing.T) {
		// Non-admin forbidden
		res := member.Client.JSON(http.MethodGet, "/api/v1/admin/model-versions", nil)
		if res.Status != http.StatusForbidden {
			t.Fatalf("expected forbidden for member on model versions, got %d", res.Status)
		}

		res = officer.Client.JSON(http.MethodGet, "/api/v1/admin/model-versions", nil)
		if res.Status != http.StatusForbidden {
			t.Fatalf("expected forbidden for officer on model versions, got %d", res.Status)
		}

		// Admin can list model versions
		res = admin.Client.JSON(http.MethodGet, "/api/v1/admin/model-versions", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("admin list model versions failed: %d %s", res.Status, res.Body)
		}

		var mvList openapi.ModelVersionList
		if err := json.Unmarshal(res.Body, &mvList); err != nil {
			t.Fatalf("decode model versions: %v", err)
		}
		if len(mvList.Items) == 0 {
			t.Fatalf("expected seeded model versions, got 0")
		}

		// Activate model version
		targetVersion := mvList.Items[0]
		res = admin.Client.JSON(http.MethodPost, "/api/v1/admin/model-versions/"+targetVersion.Id.String()+"/activate", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("admin activate model version failed: %d %s", res.Status, res.Body)
		}

		// AI Monitoring endpoint
		res = member.Client.JSON(http.MethodGet, "/api/v1/admin/ai/monitoring", nil)
		if res.Status != http.StatusForbidden {
			t.Fatalf("expected forbidden for member on ai monitoring, got %d", res.Status)
		}

		res = admin.Client.JSON(http.MethodGet, "/api/v1/admin/ai/monitoring", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("admin get ai monitoring failed: %d %s", res.Status, res.Body)
		}

		var mon openapi.AiMonitoringResponse
		if err := json.Unmarshal(res.Body, &mon); err != nil {
			t.Fatalf("decode ai monitoring response: %v", err)
		}
		if mon.ConfirmedRanks.ConfirmedTotal < 0 {
			t.Fatalf("invalid confirmed total: %d", mon.ConfirmedRanks.ConfirmedTotal)
		}
	})

	// US-48: Case Clustering
	t.Run("CaseClustering", func(t *testing.T) {
		// Non-admin forbidden
		res := member.Client.JSON(http.MethodPost, "/api/v1/admin/ai/clusters/runs", nil)
		if res.Status != http.StatusForbidden {
			t.Fatalf("expected forbidden for member on cluster run, got %d", res.Status)
		}

		// Admin can trigger cluster run
		res = admin.Client.JSON(http.MethodPost, "/api/v1/admin/ai/clusters/runs", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("admin trigger cluster run failed: %d %s", res.Status, res.Body)
		}

		var runSummary openapi.ClusterRunSummary
		if err := json.Unmarshal(res.Body, &runSummary); err != nil {
			t.Fatalf("decode cluster run summary: %v", err)
		}
		if runSummary.Status != openapi.Done {
			t.Fatalf("expected status done, got %s", runSummary.Status)
		}

		// Get cluster results
		res = admin.Client.JSON(http.MethodGet, "/api/v1/admin/ai/clusters?run_id="+runSummary.Id.String(), nil)
		if res.Status != http.StatusOK {
			t.Fatalf("admin get clusters failed: %d %s", res.Status, res.Body)
		}

		var clusterResp openapi.AiClustersResponse
		if err := json.Unmarshal(res.Body, &clusterResp); err != nil {
			t.Fatalf("decode clusters response: %v", err)
		}
		if clusterResp.Run == nil || clusterResp.Run.Id != runSummary.Id {
			t.Fatalf("cluster response run mismatch")
		}
	})

	// US-46 & US-47: Analytics Summary, Trends, Hotspots, and Export
	t.Run("AnalyticsAndHotspots", func(t *testing.T) {
		// Member forbidden from analytics
		res := member.Client.JSON(http.MethodGet, "/api/v1/analytics/summary", nil)
		if res.Status != http.StatusForbidden {
			t.Fatalf("expected forbidden for member on analytics summary, got %d", res.Status)
		}

		// Officer can view analytics summary
		res = officer.Client.JSON(http.MethodGet, "/api/v1/analytics/summary", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("officer get analytics summary failed: %d %s", res.Status, res.Body)
		}

		var summary openapi.AnalyticsSummary
		if err := json.Unmarshal(res.Body, &summary); err != nil {
			t.Fatalf("decode analytics summary: %v", err)
		}

		// File reports to test hotspot count and BR-PRV-05
		l := fileLost(t, e, member, "hotspot-lost-01", nil)
		f := fileFound(t, e, member, "hotspot-found-01", nil)
		if l.ID == "" || f.ID == "" {
			t.Fatalf("reports not created")
		}

		// Officer checks hotspots
		res = officer.Client.JSON(http.MethodGet, "/api/v1/analytics/hotspots", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("officer get hotspots failed: %d %s", res.Status, res.Body)
		}

		var hotspots openapi.HotspotsResponse
		if err := json.Unmarshal(res.Body, &hotspots); err != nil {
			t.Fatalf("decode hotspots response: %v", err)
		}
		if len(hotspots.Locations) == 0 {
			t.Fatalf("expected locations in hotspots")
		}

		// Verify BR-PRV-05: any cell with 1 or 2 reports shows "< 3"
		hasPrivacyCheck := false
		for _, loc := range hotspots.Locations {
			if loc.TotalCount > 0 && loc.TotalCount < 3 {
				if loc.DisplayTotal != "< 3" {
					t.Fatalf("expected display '< 3' for total count %d, got %s", loc.TotalCount, loc.DisplayTotal)
				}
				hasPrivacyCheck = true
			}
		}
		if !hasPrivacyCheck {
			t.Log("Note: all location counts were 0 or >= 3")
		}

		// Operational trends
		res = officer.Client.JSON(http.MethodGet, "/api/v1/analytics/trends", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("officer get trends failed: %d %s", res.Status, res.Body)
		}
		var trends openapi.AnalyticsTrendsResponse
		if err := json.Unmarshal(res.Body, &trends); err != nil {
			t.Fatalf("decode trends response: %v", err)
		}
		if len(trends.Weekly) != 12 {
			t.Fatalf("expected 12 weeks of trends, got %d", len(trends.Weekly))
		}

		// Monthly CSV export (Admin only)
		res = officer.Client.JSON(http.MethodGet, "/api/v1/analytics/export?month=2026-10", nil)
		if res.Status != http.StatusForbidden {
			t.Fatalf("expected forbidden for officer on export, got %d", res.Status)
		}

		res = admin.Client.JSON(http.MethodGet, "/api/v1/analytics/export?month=2026-10", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("admin export analytics failed: %d %s", res.Status, string(res.Body))
		}
		if !strings.Contains(string(res.Body), "Tanggal,Tipe Laporan,Nomor Laporan,Kategori,Lokasi,Status") {
			t.Fatalf("invalid CSV header: %s", string(res.Body))
		}
	})
}
