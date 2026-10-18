package testenv

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/auth"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/matching"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/config"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/storage"
	"github.com/pisondev/simpanugm/api/internal/server"
	"github.com/pisondev/simpanugm/api/migrations"
)

const (
	Image       = "pgvector/pgvector:pg18"
	appPassword = "app-test-password"
	ownerPass   = "owner-test-password"
)

func RepoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

type Database struct {
	AppURL   string
	OwnerURL string
	stop     func()
}

var (
	once   sync.Once
	shared *Database
	errDB  error
)

// one PostgreSQL per test binary, initialised by the real init script
func Shared(t testing.TB) *Database {
	t.Helper()
	once.Do(func() { shared, errDB = start() })
	if errDB != nil {
		t.Fatalf("testenv: %v", errDB)
	}
	return shared
}

func start() (*Database, error) {
	ctx := context.Background()
	root := RepoRoot()
	c, err := tcpostgres.Run(ctx, Image,
		tcpostgres.WithUsername("postgres"), tcpostgres.WithPassword("postgres"), tcpostgres.WithDatabase("postgres"),
		testcontainers.WithEnv(map[string]string{
			"SIMPAN_OWNER_PASSWORD": ownerPass, "SIMPAN_APP_PASSWORD": appPassword, "SIMPAN_READONLY_PASSWORD": "readonly-test-password",
		}),
		testcontainers.WithFiles(
			testcontainers.ContainerFile{HostFilePath: filepath.Join(root, "deploy", "postgres", "01-init.sh"), ContainerFilePath: "/docker-entrypoint-initdb.d/01-init.sh", FileMode: 0o755},
			testcontainers.ContainerFile{HostFilePath: filepath.Join(root, "deploy", "postgres", "init.sql"), ContainerFilePath: "/docker-entrypoint-initdb.d/init.sql.tpl", FileMode: 0o644},
		),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres: %w", err)
	}
	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := c.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return nil, err
	}
	d := &Database{
		AppURL:   fmt.Sprintf("postgres://simpan_app:%s@%s:%s/simpan?sslmode=disable", appPassword, host, port.Port()),
		OwnerURL: fmt.Sprintf("postgres://simpan_owner:%s@%s:%s/simpan?sslmode=disable", ownerPass, host, port.Port()),
		stop:     func() { _ = c.Terminate(context.Background()) },
	}
	if err := MigrateUp(ctx, d.OwnerURL); err != nil {
		return nil, err
	}
	pool, err := db.Open(ctx, d.AppURL, 4)
	if err != nil {
		return nil, err
	}
	defer pool.Close()
	set, err := masterdata.LoadSeed(filepath.Join(root, "data", "seed"), filepath.Join(root, "deploy", "dev-seed"))
	if err != nil {
		return nil, fmt.Errorf("load seed: %w", err)
	}
	if _, err := masterdata.ApplySeed(ctx, pool, set); err != nil {
		return nil, fmt.Errorf("apply seed: %w", err)
	}
	if err := matching.EnsureDefaultFusion(ctx, sqlc.New(pool), time.Now()); err != nil {
		return nil, fmt.Errorf("fusion version: %w", err)
	}
	return d, nil
}

func MigrateUp(ctx context.Context, ownerURL string) error {
	sqlDB, err := sql.Open("pgx", ownerURL)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()
	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	pool, err := db.Open(ctx, ownerURL, 2)
	if err != nil {
		return err
	}
	defer pool.Close()
	m, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return err
	}
	_, err = m.Migrate(ctx, rivermigrate.DirectionUp, nil)
	return err
}

type Env struct {
	T        testing.TB
	DB       *Database
	Pool     *pgxpool.Pool
	Clock    *clock.Fake
	Services *server.Services
	Jobs     *river.Client[pgx.Tx]
	Server   *httptest.Server
	Config   config.Config
	// wall clock time the Env was built, job helpers ignore what earlier tests left in the shared queue
	Started time.Time
}

// changes the configuration an Env is built with
type Option func(*config.Config)

// points the inference client at a fake, normally NewFakeML
func WithML(url string) Option { return func(c *config.Config) { c.MLBaseURL = url } }

func New(t testing.TB, opts ...Option) *Env {
	t.Helper()
	d := Shared(t)
	ctx := context.Background()
	pool, err := db.Open(ctx, d.AppURL, 8)
	if err != nil {
		t.Fatalf("testenv: open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	store, err := storage.NewFS(t.TempDir())
	if err != nil {
		t.Fatalf("testenv: storage: %v", err)
	}
	jobs, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		t.Fatalf("testenv: river: %v", err)
	}
	var handler http.Handler
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler.ServeHTTP(w, r) }))
	t.Cleanup(srv.Close)
	cfg := config.Config{
		AppEnv: "test", PublicBaseURL: srv.URL, DatabaseURL: d.AppURL, StorageDriver: "fs",
		SessionCookieSecure: false, TrustedProxies: []string{"10.0.0.0/8"}, WorkerQueues: "default=1",
	}
	for _, o := range opts {
		o(&cfg)
	}
	env := &Env{T: t, DB: d, Pool: pool, Clock: clock.NewFake(time.Now().UTC().Truncate(time.Second)), Server: srv, Config: cfg, Jobs: jobs, Started: dbNow(ctx, pool)}
	deps := server.Deps{Config: cfg, Pool: pool, Clock: env.Clock, Store: store, Jobs: jobs}
	env.Services = server.NewServices(deps)
	handler, err = server.NewRouter(deps, env.Services)
	if err != nil {
		t.Fatalf("testenv: router: %v", err)
	}
	hubCtx, stopHub := context.WithCancel(context.Background())
	go env.Services.Hub.Run(hubCtx)
	t.Cleanup(stopHub)
	return env
}

var counter atomic.Int64

func Unique(prefix string) string {
	return fmt.Sprintf("%s%d%d", prefix, time.Now().UnixNano()%1_000_000, counter.Add(1))
}

const Password = "kata-sandi-uji-yang-panjang" //nolint:gosec // a throwaway test credential

type User struct {
	sqlc.User
	Client *Client
}

func (e *Env) User(role access.Role, posts ...uuid.UUID) User {
	e.T.Helper()
	domain := "mail.ugm.ac.id"
	if role != access.Member {
		domain = "ugm.ac.id"
	}
	email := Unique(string(role)) + "@" + domain
	u, err := e.Services.Auth.CreateActiveUser(context.Background(), auth.ActiveUser{
		Email: email, Name: "Pengguna Uji " + string(role), Role: role, Password: Password, PostIDs: posts,
	})
	if err != nil {
		e.T.Fatalf("testenv: create user: %v", err)
	}
	c := e.Client()
	res := c.JSON(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": Password})
	if res.Status != http.StatusOK {
		e.T.Fatalf("testenv: login %s: %d %s", email, res.Status, res.Body)
	}
	return User{User: u, Client: c}
}

func (e *Env) Post(slug string) uuid.UUID {
	e.T.Helper()
	var id uuid.UUID
	if err := e.Pool.QueryRow(context.Background(), "SELECT id FROM posts WHERE slug = $1", slug).Scan(&id); err != nil {
		e.T.Fatalf("testenv: post %s: %v", slug, err)
	}
	return id
}

func (e *Env) Location(slug string) uuid.UUID {
	e.T.Helper()
	var id uuid.UUID
	if err := e.Pool.QueryRow(context.Background(), "SELECT id FROM locations WHERE slug = $1", slug).Scan(&id); err != nil {
		e.T.Fatalf("testenv: location %s: %v", slug, err)
	}
	return id
}

func (e *Env) Category(slug string) uuid.UUID {
	e.T.Helper()
	var id uuid.UUID
	if err := e.Pool.QueryRow(context.Background(), "SELECT id FROM categories WHERE slug = $1 AND parent_id IS NOT NULL", slug).Scan(&id); err != nil {
		e.T.Fatalf("testenv: category %s: %v", slug, err)
	}
	return id
}

func (e *Env) LastEmail(to string) notify.EmailArgs {
	e.T.Helper()
	var raw []byte
	err := e.Pool.QueryRow(context.Background(),
		"SELECT args FROM river_job WHERE kind = 'email.send' AND args->>'to' = $1 ORDER BY id DESC LIMIT 1", strings.ToLower(to)).Scan(&raw)
	if err != nil {
		e.T.Fatalf("testenv: no email to %s: %v", to, err)
	}
	var a notify.EmailArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		e.T.Fatalf("testenv: email args: %v", err)
	}
	return a
}

func (e *Env) EmailCount(to, event string) int {
	e.T.Helper()
	var n int
	err := e.Pool.QueryRow(context.Background(),
		"SELECT count(*) FROM river_job WHERE kind = 'email.send' AND args->>'to' = $1 AND args->>'event' = $2", strings.ToLower(to), event).Scan(&n)
	if err != nil {
		e.T.Fatalf("testenv: count emails: %v", err)
	}
	return n
}

func (e *Env) EmailScheduledAt(to, event string) time.Time {
	e.T.Helper()
	var at time.Time
	err := e.Pool.QueryRow(context.Background(),
		"SELECT scheduled_at FROM river_job WHERE kind = 'email.send' AND args->>'to' = $1 AND args->>'event' = $2 ORDER BY id DESC LIMIT 1", strings.ToLower(to), event).Scan(&at)
	if err != nil {
		e.T.Fatalf("testenv: no %s email to %s: %v", event, to, err)
	}
	return at
}

func TokenFrom(t testing.TB, a notify.EmailArgs) string {
	t.Helper()
	_, after, ok := strings.Cut(a.Text, "token=")
	if !ok {
		t.Fatalf("testenv: no token in email %q", a.Event)
	}
	return strings.Fields(after)[0]
}

func (e *Env) InTx(fn func(pgx.Tx) error) error { return db.InTx(context.Background(), e.Pool, fn) }

type Client struct {
	t    testing.TB
	base string
	http *http.Client
	Head http.Header
}

func (e *Env) Client() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{t: e.T, base: e.Server.URL, http: &http.Client{Jar: jar, Timeout: 30 * time.Second}, Head: http.Header{}}
}

type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

func (r Response) Decode(t testing.TB, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %d response: %v: %s", r.Status, err, r.Body)
	}
}

type Problem struct {
	Code   string `json:"code"`
	Status int    `json:"status"`
	Errors []struct {
		Field string `json:"field"`
		Code  string `json:"code"`
	} `json:"errors"`
}

func (r Response) Problem(t testing.TB) Problem {
	t.Helper()
	var p Problem
	r.Decode(t, &p)
	return p
}

func (r Response) HasField(field string) bool {
	var m map[string]any
	if json.Unmarshal(r.Body, &m) != nil {
		return false
	}
	_, ok := m[field]
	return ok
}

func (c *Client) Do(method, path string, body io.Reader, contentType string, headers ...string) Response {
	c.t.Helper()
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		c.t.Fatalf("request: %v", err)
	}
	for k, v := range c.Head {
		req.Header[k] = v
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()
	b, _ := io.ReadAll(res.Body)
	return Response{Status: res.StatusCode, Header: res.Header, Body: b}
}

// an open response for streams, the caller closes the body
func (c *Client) Open(path string, headers ...string) *http.Response {
	c.t.Helper()
	req, err := http.NewRequest(http.MethodGet, c.base+path, nil)
	if err != nil {
		c.t.Fatalf("request: %v", err)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	streaming := &http.Client{Jar: c.http.Jar}
	res, err := streaming.Do(req)
	if err != nil {
		c.t.Fatalf("GET %s: %v", path, err)
	}
	return res
}

func (c *Client) JSON(method, path string, body any, headers ...string) Response {
	c.t.Helper()
	var r io.Reader
	ct := ""
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal: %v", err)
		}
		r, ct = bytes.NewReader(b), "application/json"
	}
	return c.Do(method, path, r, ct, headers...)
}

func (c *Client) Upload(purpose string, data []byte) Response {
	c.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("purpose", purpose)
	fw, _ := w.CreateFormFile("file", "foto.jpg")
	_, _ = fw.Write(data)
	_ = w.Close()
	return c.Do(http.MethodPost, "/api/v1/media", &buf, w.FormDataContentType())
}

func (c *Client) UploadID(purpose string) uuid.UUID {
	c.t.Helper()
	res := c.Upload(purpose, JPEG(c.t, 640, 480, nil))
	if res.Status != http.StatusCreated {
		c.t.Fatalf("upload: %d %s", res.Status, res.Body)
	}
	var m struct {
		ID uuid.UUID `json:"id"`
	}
	res.Decode(c.t, &m)
	return m.ID
}

var seed atomic.Int64

// a unique pattern per call so uploads never collide on hash, extra goes right after SOI
func JPEG(t testing.TB, w, h int, extra []byte) []byte {
	t.Helper()
	n := seed.Add(1)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8((x + int(n)*37) % 256), G: uint8((y * 3) % 256), B: uint8((x*y + int(n)) % 256), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatalf("jpeg: %v", err)
	}
	b := buf.Bytes()
	if extra == nil {
		return b
	}
	out := append([]byte{}, b[:2]...)
	out = append(out, extra...)
	return append(out, b[2:]...)
}

// an APP1 segment with a GPS marker, enough to prove metadata is dropped
func ExifGPS() []byte {
	payload := append([]byte("Exif\x00\x00"), []byte("MM\x00\x2a\x00\x00\x00\x08GPSLatitude-7.7693GPSLongitude110.3779")...)
	size := len(payload) + 2
	return append([]byte{0xFF, 0xE1, byte(size >> 8), byte(size)}, payload...)
}

// river_job timestamps come from the database clock, so the cut-off does too
func dbNow(ctx context.Context, pool *pgxpool.Pool) time.Time {
	var now time.Time
	_ = pool.QueryRow(ctx, "SELECT now()").Scan(&now)
	return now
}
