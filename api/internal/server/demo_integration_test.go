//go:build integration

package server_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pisondev/simpanugm/api/internal/demo"
	"github.com/pisondev/simpanugm/api/internal/mlclient"
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
}
