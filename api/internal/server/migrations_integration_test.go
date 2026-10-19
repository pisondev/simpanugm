//go:build integration

package server_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/pisondev/simpanugm/api/internal/testenv"
	"github.com/pisondev/simpanugm/api/migrations"
)

// PRD section 9: Database migrations run forward and backward in CI
func TestMigrationsUpDownUp(t *testing.T) {
	ctx := context.Background()
	db := testenv.Shared(t)

	// Create an isolated temporary database for testing migrations up and down
	adminDB, err := sql.Open("pgx", db.AdminURL)
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	defer func() { _ = adminDB.Close() }()

	testDBName := "test_migrations_up_down"
	_, _ = adminDB.ExecContext(ctx, "DROP DATABASE IF EXISTS "+testDBName)
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+testDBName+" OWNER simpan_owner"); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+testDBName)
	})

	// Initialize extensions as superuser exactly as deploy/postgres/init.sql does
	testAdminURL := strings.Replace(db.AdminURL, "/postgres?", "/"+testDBName+"?", 1)
	testSuperDB, err := sql.Open("pgx", testAdminURL)
	if err != nil {
		t.Fatalf("open test super db: %v", err)
	}
	defer func() { _ = testSuperDB.Close() }()

	_, err = testSuperDB.ExecContext(ctx, `
		CREATE EXTENSION IF NOT EXISTS citext;
		CREATE EXTENSION IF NOT EXISTS pg_trgm;
		CREATE EXTENSION IF NOT EXISTS unaccent;
		CREATE EXTENSION IF NOT EXISTS vector;
	`)
	if err != nil {
		t.Fatalf("init extensions: %v", err)
	}

	testOwnerURL := strings.Replace(db.OwnerURL, "/simpan?", "/"+testDBName+"?", 1)
	testDB, err := sql.Open("pgx", testOwnerURL)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer func() { _ = testDB.Close() }()

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("set dialect: %v", err)
	}

	// 1. Run migrations UP
	if err := goose.UpContext(ctx, testDB, "."); err != nil {
		t.Fatalf("goose up initial: %v", err)
	}
	vUp, err := goose.GetDBVersionContext(ctx, testDB)
	if err != nil {
		t.Fatalf("get version after up: %v", err)
	}
	if vUp < 13 {
		t.Fatalf("expected at least migration version 13, got %d", vUp)
	}

	// 2. Run migrations DOWN all the way to 0
	if err := goose.DownToContext(ctx, testDB, ".", 0); err != nil {
		t.Fatalf("goose down to 0: %v", err)
	}
	vDown, err := goose.GetDBVersionContext(ctx, testDB)
	if err != nil {
		t.Fatalf("get version after down: %v", err)
	}
	if vDown != 0 {
		t.Fatalf("expected version 0 after rolling down, got %d", vDown)
	}

	// 3. Run migrations UP again to verify clean re-application
	if err := goose.UpContext(ctx, testDB, "."); err != nil {
		t.Fatalf("goose up re-apply: %v", err)
	}
	vReUp, err := goose.GetDBVersionContext(ctx, testDB)
	if err != nil {
		t.Fatalf("get version after re-up: %v", err)
	}
	if vReUp != vUp {
		t.Fatalf("expected version %d after re-up, got %d", vUp, vReUp)
	}
}
