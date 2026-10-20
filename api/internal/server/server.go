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
	"github.com/pisondev/simpanugm/api/internal/analytics"
	"github.com/pisondev/simpanugm/api/internal/assist"
	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/auth"
	"github.com/pisondev/simpanugm/api/internal/claims"
	"github.com/pisondev/simpanugm/api/internal/custody"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/matching"
	"github.com/pisondev/simpanugm/api/internal/matching/jobargs"
	"github.com/pisondev/simpanugm/api/internal/media"
	"github.com/pisondev/simpanugm/api/internal/mlclient"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/officer"
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
	Assist     *assist.Service
	Claims     *claims.Service
	Matching   *matching.Service
	Officer    *officer.Service
	Audit      *audit.Service
	Analytics  *analytics.Service
}

func NewServices(d Deps) *Services {
	st := settings.New(d.Pool, sqlc.New(d.Pool), d.Clock.Now)
	mailer := notify.NewMailer(d.Jobs, strings.TrimRight(d.Config.PublicBaseURL, "/"))
	notifier := notify.NewNotifier(d.Jobs, d.Config.PublicBaseURL, st, d.Clock)
	hub := notify.NewHub(d.Pool)
	authSvc := auth.NewService(d.Pool, d.Clock, mailer, st)
	if d.Config.RegisterPerHour > 0 {
		authSvc.LimitRegistrations(d.Config.RegisterPerHour)
	}
	if d.Config.LoginIPPer15Minutes > 0 {
		authSvc.LimitLoginsPerIP(d.Config.LoginIPPer15Minutes)
	}
	if d.Config.LoginPairPer15Minutes > 0 {
		authSvc.LimitLoginPairs(d.Config.LoginPairPer15Minutes)
	}
	md := masterdata.NewService(d.Pool, d.Clock, st)
	med := media.NewService(d.Pool, d.Store, d.Clock)
	cs := custody.NewService(d.Pool, d.Clock, st, notifier)
	ml := mlclient.New(d.Config.MLBaseURL, d.Config.MLInternalToken)
	if ml != nil && d.Metrics != nil {
		ml.WithObserver(d.Metrics)
	}
	enq := &jobargs.Enqueuer{Jobs: d.Jobs, Inference: ml != nil}
	rep := reports.NewService(d.Pool, d.Clock, med, md, cs, st, notifier, enq)
	engine := matching.NewEngine(d.Pool, d.Clock, ml, d.Store, notifier, st, d.Jobs).WithMetrics(d.Metrics)
	matchingSvc := matching.NewService(engine, enq)
	officerSvc := officer.NewService(d.Pool, d.Clock).WithReportsAndMatching(rep, matchingSvc)
	auditSvc := audit.NewService(sqlc.New(d.Pool))
	analyticsSvc := analytics.NewService(d.Pool, d.Clock, ml, st).WithJobs(d.Jobs)
	return &Services{
		Settings: st, Mailer: mailer, Notifier: notifier, Hub: hub, Notify: notify.NewService(d.Pool, d.Clock, hub),
		Auth:       authSvc,
		Masterdata: md, Media: med, Custody: cs,
		Reports:   rep,
		Claims:    claims.NewService(d.Pool, d.Clock, st, notifier, med, md, cs, rep, d.Config.HandoverPepper),
		Search:    search.NewService(d.Pool).WithML(ml).WithSemanticSearch(d.Config.FeatureSemanticSearch),
		Engine:    engine,
		Assist:    assist.NewService(d.Pool, ml, d.Store, md),
		Matching:  matchingSvc,
		Officer:   officerSvc,
		Audit:     auditSvc,
		Analytics: analyticsSvc,
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
	assistHandlers     = assist.Handlers
	claimsHandlers     = claims.Handlers
	officerHandlers    = officer.Handlers
	settingsHandlers   = settings.Handlers
	auditHandlers      = audit.Handlers
	analyticsHandlers  = analytics.Handlers
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
	*assistHandlers
	*claimsHandlers
	*officerHandlers
	*settingsHandlers
	*auditHandlers
	*analyticsHandlers
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
		custodyHandlers:    custody.NewHandlers(s.Custody, s.Reports),
		searchHandlers:     search.NewHandlers(s.Search),
		notifyHandlers:     notify.NewHandlers(s.Notify),
		matchingHandlers:   matching.NewHandlers(s.Matching),
		assistHandlers:     assist.NewHandlers(s.Assist),
		claimsHandlers:     claims.NewHandlers(s.Claims),
		officerHandlers:    officer.NewHandlers(s.Officer),
		settingsHandlers:   settings.NewHandlers(s.Settings),
		auditHandlers:      audit.NewHandlers(s.Audit),
		analyticsHandlers:  analytics.NewHandlers(s.Analytics),
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
	if d.Metrics != nil {
		r.Use(d.Metrics.InflightMiddleware)
	}
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

	if fake, ok := d.Clock.(*clock.Fake); ok {
		r.Route("/api/test", func(tr chi.Router) {
			tr.Post("/clock/advance", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Duration string `json:"duration"`
					Days     int    `json:"days"`
					Seconds  int64  `json:"seconds"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				if req.Days > 0 {
					fake.Advance(time.Duration(req.Days) * 24 * time.Hour)
				} else if req.Seconds > 0 {
					fake.Advance(time.Duration(req.Seconds) * time.Second)
				} else if req.Duration != "" {
					if dur, err := time.ParseDuration(req.Duration); err == nil {
						fake.Advance(dur)
					}
				}
				writeJSON(w, http.StatusOK, map[string]any{"now": fake.Now()})
			})
			tr.Get("/clock", func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusOK, map[string]any{"now": fake.Now()})
			})
			tr.Post("/clock/reset", func(w http.ResponseWriter, _ *http.Request) {
				fake.Set(time.Now())
				writeJSON(w, http.StatusOK, map[string]any{"now": fake.Now()})
			})
			tr.Post("/custody/sweep", func(w http.ResponseWriter, r *http.Request) {
				unclaimed, warned, overdue, reminders, err := s.Custody.SweepDue(r.Context())
				if err != nil {
					httpx.WriteError(w, r, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"unclaimed": unclaimed,
					"warned":    warned,
					"overdue":   overdue,
					"reminders": reminders,
				})
			})
		})
	}

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
