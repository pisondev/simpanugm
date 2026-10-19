package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/auth"
	"github.com/pisondev/simpanugm/api/internal/claims"
	"github.com/pisondev/simpanugm/api/internal/custody"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/maintenance"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/matching"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/config"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/logx"
	"github.com/pisondev/simpanugm/api/internal/platform/mail"
	"github.com/pisondev/simpanugm/api/internal/platform/metrics"
	"github.com/pisondev/simpanugm/api/internal/platform/settings"
	"github.com/pisondev/simpanugm/api/internal/platform/storage"
	"github.com/pisondev/simpanugm/api/internal/reports"
	"github.com/pisondev/simpanugm/api/internal/server"
	"github.com/pisondev/simpanugm/api/migrations"
)

var version = "dev"

const usage = `simpan, the SimpanUGM backend

usage:
  simpan serve                       run the HTTP API
  simpan worker                      run background jobs
  simpan migrate [up|down|status]    apply database migrations, default up
  simpan seed [--dir path] [--overlay path] [--demo]  load master data from data/seed
  simpan admin create --email e --name n
  simpan healthcheck URL             exit 0 when URL answers 200, for container health checks
  simpan version`

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var err error
	switch args[1] {
	case "serve":
		err = serve(ctx)
	case "worker":
		err = worker(ctx)
	case "migrate":
		err = migrate(ctx, args[2:])
	case "seed":
		err = seed(ctx, args[2:])
	case "admin":
		err = admin(ctx, args[2:])
	case "version":
		fmt.Println(version)
	case "healthcheck":
		err = healthcheck(args[2:])
	default:
		fmt.Fprintln(os.Stderr, usage)
		return 2
	}
	if err != nil {
		slog.Error("command failed", "command", args[1], "error", err.Error())
		return 1
	}
	return 0
}

func setup(ctx context.Context, service string) (config.Config, *pgxpool.Pool, error) {
	cfg, err := config.Load()
	if err != nil {
		return cfg, nil, fmt.Errorf("config: %w", err)
	}
	slog.SetDefault(logx.New(os.Stdout, cfg.LogLevel, service, version))
	pool, err := db.Open(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns)
	return cfg, pool, err
}

func clockFor(cfg config.Config) clock.Clock {
	if cfg.FeatureFakeClock {
		return clock.NewFake(time.Now())
	}
	return clock.System{}
}

func serve(ctx context.Context) error {
	cfg, pool, err := setup(ctx, "api")
	if err != nil {
		return err
	}
	defer pool.Close()
	store, err := storage.NewFS(cfg.StorageFSRoot)
	if err != nil {
		return err
	}
	jobs, err := river.NewClient(riverpgxv5.New(pool), &river.Config{Logger: slog.Default()})
	if err != nil {
		return err
	}
	m := metrics.NewHTTP("api")
	deps := server.Deps{Config: cfg, Pool: pool, Clock: clockFor(cfg), Store: store, Jobs: jobs, Metrics: m}
	svcs := server.NewServices(deps)
	router, err := server.NewRouter(deps, svcs)
	if err != nil {
		return err
	}
	go svcs.Hub.Run(ctx)
	api := &http.Server{
		Addr: cfg.HTTPAddr, Handler: router,
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 120 * time.Second,
	}
	return listen(ctx, api, metricsServer(cfg.MetricsAddr, m.Handler(), pool))
}

func metricsServer(addr string, h http.Handler, pool *pgxpool.Pool) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", h)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.Handle("GET /readyz", server.Ready(pool))
	return &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}

func listen(ctx context.Context, servers ...*http.Server) error {
	errs := make(chan error, len(servers))
	for _, s := range servers {
		go func() {
			slog.Info("listening", "addr", s.Addr)
			if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errs <- err
			}
		}()
	}
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}
	slog.Info("shutting down")
	shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	for _, s := range servers {
		if err := s.Shutdown(shutdown); err != nil {
			slog.Warn("shutdown", "addr", s.Addr, "error", err.Error())
		}
	}
	return nil
}

func worker(ctx context.Context) error {
	cfg, pool, err := setup(ctx, "worker")
	if err != nil {
		return err
	}
	defer pool.Close()
	store, err := storage.NewFS(cfg.StorageFSRoot)
	if err != nil {
		return err
	}
	sender, err := mail.NewSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPTLS, cfg.MailFrom)
	if err != nil {
		return err
	}
	insert, err := river.NewClient(riverpgxv5.New(pool), &river.Config{Logger: slog.Default()})
	if err != nil {
		return err
	}
	c := clockFor(cfg)
	svcs := server.NewServices(server.Deps{Config: cfg, Pool: pool, Clock: c, Store: store, Jobs: insert})
	workers := river.NewWorkers()
	river.AddWorker(workers, &notify.EmailWorker{Sender: sender, Q: sqlc.New(pool)})
	river.AddWorker(workers, &notify.DigestWorker{Q: sqlc.New(pool), Jobs: insert, Clock: c, BaseURL: cfg.PublicBaseURL})
	river.AddWorker(workers, &maintenance.PurgeWorker{Media: svcs.Media, Auth: svcs.Auth, Q: sqlc.New(pool), Clock: c})
	matching.AddWorkers(workers, svcs.Engine)
	river.AddWorker(workers, &claims.ExpireWorker{S: svcs.Claims})
	river.AddWorker(workers, &custody.SweepWorker{S: svcs.Custody})
	river.AddWorker(workers, &reports.SweepWorker{S: svcs.Reports})
	// scoring needs an active fusion version, a fresh database gets the embedded one
	if err := matching.EnsureDefaultFusion(ctx, sqlc.New(pool), c.Now()); err != nil {
		return err
	}
	counts, err := cfg.Queues()
	if err != nil {
		return err
	}
	queues := map[string]river.QueueConfig{}
	for name, n := range counts {
		queues[name] = river.QueueConfig{MaxWorkers: n}
	}
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: queues, Workers: workers, PeriodicJobs: slices.Concat(maintenance.Periodic(), notify.Periodic(), matching.Periodic(), claims.Periodic(), custody.Periodic(), reports.Periodic()),
		CompletedJobRetentionPeriod: time.Hour, Logger: slog.Default(),
	})
	if err != nil {
		return err
	}
	if err := client.Start(ctx); err != nil {
		return err
	}
	runErr := listen(ctx, metricsServer(cfg.MetricsAddr, metrics.NewHTTP("worker").Handler(), pool))
	stopCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := client.Stop(stopCtx); err != nil {
		slog.Warn("worker stop", "error", err.Error())
	}
	return runErr
}

func migrate(ctx context.Context, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	slog.SetDefault(logx.New(os.Stdout, cfg.LogLevel, "migrate", version))
	if cfg.DatabaseMigratorURL == "" {
		return errors.New("DATABASE_MIGRATOR_URL is required for migrate")
	}
	direction := "up"
	if len(args) > 0 {
		direction = args[0]
	}
	sqlDB, err := sql.Open("pgx", cfg.DatabaseMigratorURL)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()
	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	switch direction {
	case "up":
		if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
			return err
		}
		pool, err := db.Open(ctx, cfg.DatabaseMigratorURL, 2)
		if err != nil {
			return err
		}
		defer pool.Close()
		migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
		if err != nil {
			return err
		}
		if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
			return fmt.Errorf("river migrations: %w", err)
		}
	case "down":
		if err := goose.DownContext(ctx, sqlDB, "."); err != nil {
			return err
		}
	case "status":
		return goose.StatusContext(ctx, sqlDB, ".")
	default:
		return fmt.Errorf("unknown migrate direction %q", direction)
	}
	v, err := goose.GetDBVersionContext(ctx, sqlDB)
	if err != nil {
		return err
	}
	slog.Info("migrations applied", "direction", direction, "version", v)
	return nil
}

func defaultSeedDir() string {
	if d := os.Getenv("SEED_DIR"); d != "" {
		return d
	}
	for _, d := range []string{"data/seed", "../data/seed", "/app/seed"} {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			return d
		}
	}
	return "data/seed"
}

func seed(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	dir := fs.String("dir", defaultSeedDir(), "reviewed seed directory, normally data/seed")
	overlay := fs.String("overlay", os.Getenv("SEED_OVERLAY_DIR"), "development files used only where the seed directory has none, refused in production")
	demo := fs.Bool("demo", false, "also create demo accounts, refused in production")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, pool, err := setup(ctx, "seed")
	if err != nil {
		return err
	}
	defer pool.Close()
	dirs := []string{*dir}
	if *overlay != "" {
		if cfg.IsProduction() {
			return errors.New("seed --overlay is refused in production, promote reviewed files into data/seed")
		}
		dirs = append(dirs, *overlay)
	}
	set, err := masterdata.LoadSeed(dirs...)
	if err != nil {
		return err
	}
	sum, err := masterdata.ApplySeed(ctx, pool, set)
	if err != nil {
		return err
	}
	if err := matching.EnsureDefaultFusion(ctx, sqlc.New(pool), time.Now()); err != nil {
		return err
	}
	abs, _ := filepath.Abs(*dir)
	slog.Info("seed applied", "dir", abs, "sources", set.Sources, "summary", sum)
	if !*demo {
		return nil
	}
	if cfg.IsProduction() {
		return errors.New("seed --demo is refused in production")
	}
	return demoAccounts(ctx, cfg, pool)
}

func demoAccounts(ctx context.Context, cfg config.Config, pool *pgxpool.Pool) error {
	jobs, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		return err
	}
	svc := auth.NewService(pool, clock.System{}, notify.NewMailer(jobs, cfg.PublicBaseURL), settings.New(sqlc.New(pool), time.Now))
	posts, err := sqlc.New(pool).ListPosts(ctx)
	if err != nil {
		return err
	}
	postIDs := make([]uuid.UUID, 0, len(posts))
	for _, p := range posts {
		postIDs = append(postIDs, p.ID)
	}
	accounts := []auth.ActiveUser{
		{Email: "anggota@demo.simpanugm.my.id", Name: "Anggota Demo", Role: access.Member},
		{Email: "petugas@demo.simpanugm.my.id", Name: "Petugas Demo", Role: access.Officer, PostIDs: postIDs},
		{Email: "admin@demo.simpanugm.my.id", Name: "Admin Demo", Role: access.Admin},
	}
	for _, a := range accounts {
		a.Password = randomPassword()
		u, err := svc.CreateActiveUser(ctx, a)
		if err != nil {
			return err
		}
		if u.Email == a.Email {
			fmt.Printf("%-8s %-32s %s\n", a.Role, a.Email, a.Password)
		}
	}
	fmt.Println("passwords are shown once, existing demo accounts keep their old password")
	return nil
}

func randomPassword() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func admin(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "create" {
		return errors.New("usage: simpan admin create --email e --name n")
	}
	fs := flag.NewFlagSet("admin create", flag.ContinueOnError)
	email := fs.String("email", "", "administrator email")
	name := fs.String("name", "", "full name")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	cfg, pool, err := setup(ctx, "admin")
	if err != nil {
		return err
	}
	defer pool.Close()
	jobs, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		return err
	}
	svc := auth.NewService(pool, clock.System{}, notify.NewMailer(jobs, cfg.PublicBaseURL), settings.New(sqlc.New(pool), time.Now))
	link, err := svc.BootstrapAdmin(ctx, *email, *name)
	if err != nil {
		return err
	}
	fmt.Println("administrator created, set the password within 72 hours at:")
	fmt.Println(link)
	return nil
}

func healthcheck(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: simpan healthcheck URL")
	}
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Get(args[0]) //nolint:gosec,noctx // the URL is the container's own endpoint
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", res.StatusCode)
	}
	return nil
}
