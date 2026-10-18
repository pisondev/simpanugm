package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	apispec "github.com/pisondev/simpanugm/api"
	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/auth"
	"github.com/pisondev/simpanugm/api/internal/custody"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/matching"
	"github.com/pisondev/simpanugm/api/internal/matching/jobargs"
	"github.com/pisondev/simpanugm/api/internal/media"
	"github.com/pisondev/simpanugm/api/internal/mlclient"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/config"
	"github.com/pisondev/simpanugm/api/internal/platform/httpx"
	"github.com/pisondev/simpanugm/api/internal/platform/metrics"
	"github.com/pisondev/simpanugm/api/internal/platform/ratelimit"
	"github.com/pisondev/simpanugm/api/internal/platform/reqctx"
	"github.com/pisondev/simpanugm/api/internal/platform/settings"
	"github.com/pisondev/simpanugm/api/internal/platform/storage"
	"github.com/pisondev/simpanugm/api/internal/reports"
	"github.com/pisondev/simpanugm/api/internal/search"
	"github.com/pisondev/simpanugm/api/migrations"
)

type Deps struct {
	Config  config.Config
	Pool    *pgxpool.Pool
	Clock   clock.Clock
	Store   storage.Store
	Jobs    *river.Client[pgx.Tx]
	Metrics *metrics.HTTP
}

type Services struct {
	Settings   *settings.Store
	Mailer     *notify.Mailer
	Notifier   *notify.Notifier
	Hub        *notify.Hub
	Notify     *notify.Service
	Auth       *auth.Service
	Masterdata *masterdata.Service
	Media      *media.Service
	Custody    *custody.Service
	Reports    *reports.Service
	Search     *search.Service
	Engine     *matching.Engine
	Matching   *matching.Service
}

func NewServices(d Deps) *Services {
	st := settings.New(sqlc.New(d.Pool), d.Clock.Now)
	mailer := notify.NewMailer(d.Jobs, strings.TrimRight(d.Config.PublicBaseURL, "/"))
	notifier := notify.NewNotifier(d.Jobs, d.Config.PublicBaseURL, st, d.Clock)
	hub := notify.NewHub(d.Pool)
	md := masterdata.NewService(d.Pool, d.Clock, st)
	med := media.NewService(d.Pool, d.Store, d.Clock)
	cs := custody.NewService(d.Pool, d.Clock, st)
	ml := mlclient.New(d.Config.MLBaseURL, d.Config.MLInternalToken)
	enq := &jobargs.Enqueuer{Jobs: d.Jobs, Inference: ml != nil}
	engine := matching.NewEngine(d.Pool, d.Clock, ml, d.Store, notifier, st, d.Jobs)
	return &Services{
		Settings: st, Mailer: mailer, Notifier: notifier, Hub: hub, Notify: notify.NewService(d.Pool, d.Clock, hub),
		Auth:       auth.NewService(d.Pool, d.Clock, mailer, st),
		Masterdata: md, Media: med, Custody: cs,
		Reports:  reports.NewService(d.Pool, d.Clock, med, md, cs, st, notifier, enq),
		Search:   search.NewService(d.Pool),
		Engine:   engine,
		Matching: matching.NewService(engine, enq),
	}
}

type (
	authHandlers       = auth.Handlers
	masterdataHandlers = masterdata.Handlers
	mediaHandlers      = media.Handlers
	reportHandlers     = reports.Handlers
	custodyHandlers    = custody.Handlers
	searchHandlers     = search.Handlers
	notifyHandlers     = notify.Handlers
	matchingHandlers   = matching.Handlers
)

type strictServer struct {
	*authHandlers
	*masterdataHandlers
	*mediaHandlers
	*reportHandlers
	*custodyHandlers
	*searchHandlers
	*notifyHandlers
	*matchingHandlers
}

var _ openapi.StrictServerInterface = strictServer{}

var masterdataPaths = map[string]bool{
	"/api/v1/categories": true, "/api/v1/zones": true, "/api/v1/locations": true, "/api/v1/posts": true, "/api/v1/colours": true,
}

func NewRouter(d Deps, s *Services) (http.Handler, error) {
	trusted, err := d.Config.TrustedPrefixes()
	if err != nil {
		return nil, err
	}
	doc, err := openapi3.NewLoader().LoadFromData(apispec.OpenAPI)
	if err != nil {
		return nil, fmt.Errorf("server: load spec: %w", err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("server: invalid spec: %w", err)
	}
	contract, err := Contract(doc)
	if err != nil {
		return nil, fmt.Errorf("server: build contract router: %w", err)
	}
	cop := http.NewCrossOriginProtection()
	if err := cop.AddTrustedOrigin(origin(d.Config.PublicBaseURL)); err != nil {
		return nil, fmt.Errorf("server: trusted origin: %w", err)
	}
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteProblem(w, r, apperr.New(http.StatusForbidden, "csrf_origin_mismatch", "Permintaan ditolak", "Asal permintaan tidak dikenali."))
	}))
	cookies := auth.Cookies{Secure: d.Config.SessionCookieSecure}
	handlers := strictServer{
		authHandlers:       auth.NewHandlers(s.Auth, cookies),
		masterdataHandlers: masterdata.NewHandlers(s.Masterdata),
		mediaHandlers:      media.NewHandlers(s.Media),
		reportHandlers:     reports.NewHandlers(s.Reports),
		custodyHandlers:    custody.NewHandlers(s.Custody),
		searchHandlers:     search.NewHandlers(s.Search),
		notifyHandlers:     notify.NewHandlers(s.Notify),
		matchingHandlers:   matching.NewHandlers(s.Matching),
	}
	strict := openapi.NewStrictHandlerWithOptions(handlers, nil, openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			httpx.WriteProblem(w, r, apperr.BadRequest("Format permintaan tidak valid.").WithCause(err))
		},
		ResponseErrorHandlerFunc: httpx.WriteError,
	})

	var observe func(*http.Request, string, int, time.Duration)
	if d.Metrics != nil {
		observe = d.Metrics.Observe
	}
	r := chi.NewRouter()
	r.Use(httpx.RequestMeta(trusted), httpx.Observe(observe))
	r.NotFound(func(w http.ResponseWriter, r *http.Request) { httpx.WriteProblem(w, r, apperr.NotFound()) })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteProblem(w, r, apperr.New(http.StatusMethodNotAllowed, "method_not_allowed", "Metode tidak didukung", "Metode ini tidak tersedia untuk alamat tersebut."))
	})
	health := func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
	r.Get("/healthz", health)
	r.Get("/api/healthz", health)
	r.Get("/readyz", Ready(d.Pool))

	userLimit := ratelimit.New(300, time.Minute, d.Clock.Now)
	ipLimit := ratelimit.New(600, time.Minute, d.Clock.Now)
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(cop.Handler)
		api.Use(httpx.LimitBody(64<<10, map[string]int64{"/api/v1/media": media.MaxBytes + 1<<20}))
		api.Use(s.Auth.Authenticate(cookies))
		api.Use(generalLimit(userLimit, ipLimit))
		api.Use(contract)
		api.Use(httpx.ETag(func(r *http.Request) bool { return masterdataPaths[r.URL.Path] }))
		api.Get("/media/{id}", s.Media.Serve)
		api.Get("/notifications/stream", s.Notify.Stream)
		openapi.HandlerWithOptions(strict, openapi.ChiServerOptions{
			BaseRouter: api,
			ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				httpx.WriteProblem(w, r, apperr.BadRequest("Parameter tidak valid.").WithCause(err))
			},
		})
	})
	return r, nil
}

func origin(base string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	return u.Scheme + "://" + u.Host
}

func generalLimit(user, ip *ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if p := access.FromContext(r.Context()); p != nil {
				if ok, wait := user.Allow(p.UserID.String()); !ok {
					httpx.WriteProblem(w, r, apperr.RateLimited(wait))
					return
				}
			}
			if ok, wait := ip.Allow(reqctx.From(r.Context()).IP.String()); !ok {
				httpx.WriteProblem(w, r, apperr.RateLimited(wait))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ready only when the schema is at the version this binary was built with
func Ready(pool *pgxpool.Pool) http.HandlerFunc {
	want := migrations.Latest()
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		var have int64
		err := pool.QueryRow(ctx, "SELECT coalesce(max(version_id), 0) FROM goose_db_version WHERE is_applied").Scan(&have)
		switch {
		case err != nil:
			slog.WarnContext(r.Context(), "readiness: database", "error", err.Error())
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "reason": "database"})
		case have < want:
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unavailable", "reason": "migrations", "have": have, "want": want})
		default:
			writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "schema_version": have})
		}
	}
}
