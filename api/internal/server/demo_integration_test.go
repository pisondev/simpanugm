//go:build integration

package server_test

import (
	"context"
	"net/http"
	"path/filepath"
	"slices"
	"testing"

	"github.com/pisondev/simpanugm/api/internal/demo"
	"github.com/pisondev/simpanugm/api/internal/mlclient"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/storage"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

func TestDemoSeedIntegration(t *testing.T) {
	ctx := context.Background()
	e := testenv.New(t)
	fakeML := testenv.NewFakeML(t)
	ml := mlclient.New(fakeML.URL, "test-secret")
	store, err := storage.NewFS(t.TempDir())
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	root := testenv.RepoRoot()
	seedDirs := []string{
		filepath.Join(root, "data", "seed"),
		filepath.Join(root, "deploy", "dev-seed"),
	}

	deps := demo.Deps{
		Config:   e.Config,
		Pool:     e.Pool,
		SQLDB:    nil,
		Clock:    e.Clock,
		Store:    store,
		ML:       ml,
		Jobs:     e.Jobs,
		SeedDirs: seedDirs,
		Reset:    false,
	}

	// 1. Run demo seeder
	res, err := demo.Run(ctx, deps)
	if err != nil {
		t.Fatalf("demo.Run failed: %v", err)
	}

	if res.UsersCount < 7 {
		t.Errorf("expected at least 7 demo users, got %d", res.UsersCount)
	}
	if res.LostReportsCount < 19 {
		t.Errorf("expected at least 19 lost reports, got %d", res.LostReportsCount)
	}
	if res.FoundReportsCount < 21 {
		t.Errorf("expected at least 21 found reports, got %d", res.FoundReportsCount)
	}
	if res.ClaimsCount < 5 {
		t.Errorf("expected at least 5 claims across statuses, got %d", res.ClaimsCount)
	}
	if res.DuplicatesCount < 1 {
		t.Errorf("expected at least 1 duplicate link, got %d", res.DuplicatesCount)
	}

	// Check materialized views
	var count int
	if err := e.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM mv_reports_daily").Scan(&count); err != nil {
		t.Fatalf("query mv_reports_daily: %v", err)
	}

	// 2. Run demo seeder again to verify idempotency (no duplicate errors, clean second run)
	res2, err := demo.Run(ctx, deps)
	if err != nil {
		t.Fatalf("demo.Run second run (idempotency) failed: %v", err)
	}
	if res2.UsersCount < 7 {
		t.Errorf("expected at least 7 demo users on second run, got %d", res2.UsersCount)
	}
	if res.LostReportsCount != res2.LostReportsCount || res.FoundReportsCount != res2.FoundReportsCount || res.ClaimsCount != res2.ClaimsCount {
		t.Fatalf("demo counts changed on replay: first=%+v second=%+v", res, res2)
	}
	var claimCount int
	if err := e.Pool.QueryRow(ctx, `SELECT count(*) FROM claims c JOIN users u ON u.id = c.claimant_id
		WHERE u.email LIKE '%@demo.simpanugm.my.id'`).Scan(&claimCount); err != nil || claimCount != len(demo.DemoClaims) {
		t.Fatalf("claims duplicated or missing on replay: count=%d err=%v", claimCount, err)
	}

	var approvedCode, approvedFound, requestedCode, intakeCode string
	if err := e.Pool.QueryRow(ctx, `SELECT c.code, f.code FROM claims c
		JOIN users u ON u.id = c.claimant_id JOIN found_reports f ON f.id = c.found_report_id
		JOIN posts p ON p.id = f.holding_post_id
		WHERE c.status = 'approved' AND u.email = 'anggota.pemilik@demo.simpanugm.my.id'
		AND f.custody_status = 'in_custody' AND p.slug = 'pos-perpustakaan'`).Scan(&approvedCode, &approvedFound); err != nil {
		t.Fatalf("approved claim at the library post: %v", err)
	}
	if err := e.Pool.QueryRow(ctx, `SELECT c.code FROM claims c JOIN users u ON u.id = c.claimant_id
		WHERE c.status = 'info_requested' AND u.email = 'anggota.pemilik@demo.simpanugm.my.id'`).Scan(&requestedCode); err != nil {
		t.Fatalf("owner claim awaiting information: %v", err)
	}
	if err := e.Pool.QueryRow(ctx, `SELECT f.code FROM found_reports f JOIN posts p ON p.id = f.intended_post_id
		JOIN users u ON u.id = f.finder_id
		WHERE f.custody_status = 'with_finder' AND p.slug = 'pos-perpustakaan'
		AND u.email = 'anggota@demo.simpanugm.my.id'`).Scan(&intakeCode); err != nil {
		t.Fatalf("item awaiting physical receipt at the library: %v", err)
	}
	login := func(email string) *testenv.Client {
		t.Helper()
		client := e.Client()
		r := client.JSON(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": demo.DemoPassword})
		if r.Status != http.StatusOK {
			t.Fatalf("demo login: %d %s", r.Status, r.Body)
		}
		return client
	}
	owner := login("anggota.pemilik@demo.simpanugm.my.id")
	officer := login("petugas.pos1@demo.simpanugm.my.id")
	var approved, officerView, requested openapi.Claim
	owner.JSON(http.MethodGet, "/api/v1/claims/"+approvedCode, nil).Decode(t, &approved)
	if approved.HandoverCode == nil || len(*approved.HandoverCode) != 6 {
		t.Fatal("approved demo claim must expose a six-digit code to its owner")
	}
	officer.JSON(http.MethodGet, "/api/v1/claims/"+approvedCode, nil).Decode(t, &officerView)
	if officerView.HandoverCode != nil || !slices.Contains(officerView.Actions, "handover") {
		t.Fatal("library officer must be able to hand over without seeing the owner's code")
	}
	owner.JSON(http.MethodGet, "/api/v1/claims/"+requestedCode, nil).Decode(t, &requested)
	if !slices.Contains(requested.Actions, "respond") {
		t.Fatal("owner cannot answer the seeded information request")
	}
	if r := owner.JSON(http.MethodPost, "/api/v1/claims/"+requestedCode+"/respond", map[string]string{
		"message": "Ada stiker kartun di balik cover kalkulator.",
	}); r.Status != http.StatusOK {
		t.Fatalf("answer seeded request: %d %s", r.Status, r.Body)
	}
	if r := officer.JSON(http.MethodPost, "/api/v1/found-reports/"+intakeCode+"/receive", map[string]any{
		"post_id": e.Post("pos-perpustakaan"), "storage_slot": "Papan Kunci Demo",
	}); r.Status != http.StatusOK {
		t.Fatalf("receive seeded item: %d %s", r.Status, r.Body)
	}
	if r := officer.JSON(http.MethodPost, "/api/v1/found-reports/"+approvedFound+"/handover", map[string]any{
		"claim_id": approved.Id, "code": *approved.HandoverCode, "recipient_id_type": "ktm", "recipient_id_last4": "3107", "name_confirmed": true,
	}); r.Status != http.StatusOK {
		t.Fatalf("hand over seeded approved claim: %d %s", r.Status, r.Body)
	}
}

func TestDemoRefusedInProduction(t *testing.T) {
	ctx := context.Background()
	e := testenv.New(t)

	prodCfg := e.Config
	prodCfg.AppEnv = "production"

	deps := demo.Deps{
		Config: prodCfg,
		Pool:   e.Pool,
	}

	_, err := demo.Run(ctx, deps)
	if err == nil {
		t.Fatal("expected demo.Run to fail in production, got nil error")
	}
	if err.Error() != "simpan demo is refused in production" {
		t.Errorf("unexpected error message: %v", err)
	}
}
