//go:build integration

package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

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

	// Journey J7: Admin activates candidate fusion version and rolls back, monitoring reflects it
	t.Run("FusionActivationAndRollback", func(t *testing.T) {
		ctx := context.Background()
		// 1. Get initial active fusion version
		var initialFusionID uuid.UUID
		err := e.Pool.QueryRow(ctx, "SELECT id FROM model_versions WHERE component = 'fusion' AND status = 'active'").Scan(&initialFusionID)
		if err != nil {
			t.Fatalf("get active fusion: %v", err)
		}

		// 2. Ensure candidate version exists
		var candidateID uuid.UUID
		err = e.Pool.QueryRow(ctx, "SELECT id FROM model_versions WHERE name = 'fusion-learned-2026.11.0'").Scan(&candidateID)
		if err != nil {
			t.Fatalf("get candidate fusion: %v", err)
		}

		// 3. Member forbidden to activate
		res := member.Client.JSON(http.MethodPost, "/api/v1/admin/model-versions/"+candidateID.String()+"/activate", nil)
		if res.Status != http.StatusForbidden {
			t.Fatalf("expected forbidden for member activating model, got %d", res.Status)
		}

		// 4. Admin activates candidate version
		res = admin.Client.JSON(http.MethodPost, "/api/v1/admin/model-versions/"+candidateID.String()+"/activate", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("admin activate candidate failed: %d %s", res.Status, res.Body)
		}

		// Verify candidate is now active, initial is retired
		var candidateStatus, initialStatus string
		_ = e.Pool.QueryRow(ctx, "SELECT status FROM model_versions WHERE id = $1", candidateID).Scan(&candidateStatus)
		_ = e.Pool.QueryRow(ctx, "SELECT status FROM model_versions WHERE id = $1", initialFusionID).Scan(&initialStatus)
		if candidateStatus != "active" || initialStatus != "retired" {
			t.Fatalf("expected candidate active and initial retired, got candidate=%s, initial=%s", candidateStatus, initialStatus)
		}

		// Verify audit log has model_version.activate entry
		var auditCount int
		err = e.Pool.QueryRow(ctx, "SELECT count(*) FROM audit_logs WHERE action = 'model_version.activate' AND entity_id = $1", candidateID).Scan(&auditCount)
		if err != nil || auditCount < 1 {
			t.Fatalf("expected audit log entry for candidate activation, count=%d, err=%v", auditCount, err)
		}

		// Verify AI monitoring active_versions reflects candidate
		res = admin.Client.JSON(http.MethodGet, "/api/v1/admin/ai/monitoring", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("get ai monitoring failed: %d %s", res.Status, res.Body)
		}
		var mon openapi.AiMonitoringResponse
		_ = json.Unmarshal(res.Body, &mon)
		foundActive := false
		for _, v := range mon.ActiveVersions {
			if v.Component == "fusion" && v.Id == candidateID {
				foundActive = true
			}
		}
		if !foundActive {
			t.Fatalf("expected active_versions in ai monitoring to reflect candidate")
		}

		// 5. Admin rolls back to initial fusion version
		res = admin.Client.JSON(http.MethodPost, "/api/v1/admin/model-versions/"+initialFusionID.String()+"/activate", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("admin rollback to initial fusion failed: %d %s", res.Status, res.Body)
		}

		_ = e.Pool.QueryRow(ctx, "SELECT status FROM model_versions WHERE id = $1", initialFusionID).Scan(&initialStatus)
		_ = e.Pool.QueryRow(ctx, "SELECT status FROM model_versions WHERE id = $1", candidateID).Scan(&candidateStatus)
		if initialStatus != "active" || candidateStatus != "retired" {
			t.Fatalf("expected initial active and candidate retired after rollback, got initial=%s, candidate=%s", initialStatus, candidateStatus)
		}

		// Verify monitoring reflects rollback
		res = admin.Client.JSON(http.MethodGet, "/api/v1/admin/ai/monitoring", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("get ai monitoring after rollback failed: %d %s", res.Status, res.Body)
		}
		_ = json.Unmarshal(res.Body, &mon)
		foundRolledBack := false
		for _, v := range mon.ActiveVersions {
			if v.Component == "fusion" && v.Id == initialFusionID {
				foundRolledBack = true
			}
		}
		if !foundRolledBack {
			t.Fatalf("expected active_versions in ai monitoring to reflect rolled-back initial fusion")
		}
	})
}
