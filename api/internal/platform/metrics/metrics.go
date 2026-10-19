package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HTTP struct {
	service               string
	registry              *prometheus.Registry
	requests              *prometheus.CounterVec
	duration              *prometheus.HistogramVec
	inflight              prometheus.Gauge
	sseConnections        prometheus.Gauge
	authLoginTotal        *prometheus.CounterVec
	rateLimitedTotal      *prometheus.CounterVec
	domainTransitions     *prometheus.CounterVec
	riverJobsTotal        *prometheus.CounterVec
	riverJobDuration      *prometheus.HistogramVec
	riverQueueDepth       *prometheus.GaugeVec
	emailsSentTotal       *prometheus.CounterVec
	mlRequestDuration     *prometheus.HistogramVec
	matchPipelineSeconds  prometheus.Histogram
	matchTop1Score        prometheus.Histogram
	matchCandidatesStored prometheus.Histogram
}

type dbPoolCollector struct {
	pool     *pgxpool.Pool
	acquired *prometheus.Desc
	idle     *prometheus.Desc
}

func newDBPoolCollector(pool *pgxpool.Pool, labels prometheus.Labels) *dbPoolCollector {
	return &dbPoolCollector{
		pool: pool,
		acquired: prometheus.NewDesc(
			"db_pool_acquired_conns",
			"Number of currently acquired pool connections.",
			nil,
			labels,
		),
		idle: prometheus.NewDesc(
			"db_pool_idle_conns",
			"Number of currently idle pool connections.",
			nil,
			labels,
		),
	}
}

func (c *dbPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquired
	ch <- c.idle
}

func (c *dbPoolCollector) Collect(ch chan<- prometheus.Metric) {
	if c.pool == nil {
		return
	}
	stat := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(c.acquired, prometheus.GaugeValue, float64(stat.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.idle, prometheus.GaugeValue, float64(stat.IdleConns()))
}

func NewHTTP(service string) *HTTP {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	labels := prometheus.Labels{"service": service}
	h := &HTTP{
		service:  service,
		registry: reg,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total", Help: "HTTP requests by route, method and status.", ConstLabels: labels,
		}, []string{"route", "method", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "http_request_duration_seconds", Help: "HTTP request latency by route and method.", ConstLabels: labels,
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.5, 1, 2, 5, 10},
		}, []string{"route", "method"}),
		inflight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "http_inflight_requests", Help: "Current number of in-flight HTTP requests.", ConstLabels: labels,
		}),
		sseConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sse_connections", Help: "Current number of open Server-Sent Events connections.", ConstLabels: labels,
		}),
		authLoginTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "auth_login_total", Help: "Authentication login outcomes.", ConstLabels: labels,
		}, []string{"result"}),
		rateLimitedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rate_limited_total", Help: "Total rate limited requests by limit name.", ConstLabels: labels,
		}, []string{"limit"}),
		domainTransitions: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "domain_transitions_total", Help: "State machine transitions by entity and transition name.", ConstLabels: labels,
		}, []string{"machine", "transition"}),
		riverJobsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "river_jobs_total", Help: "River background jobs executed by kind and state.", ConstLabels: labels,
		}, []string{"kind", "state"}),
		riverJobDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "river_job_duration_seconds", Help: "River job processing latency by kind.", ConstLabels: labels,
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60},
		}, []string{"kind"}),
		riverQueueDepth: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "river_queue_depth", Help: "River job queue depth.", ConstLabels: labels,
		}, []string{"queue"}),
		emailsSentTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "emails_sent_total", Help: "Emails sent by event and delivery result.", ConstLabels: labels,
		}, []string{"event", "result"}),
		mlRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "ml_request_duration_seconds", Help: "ML client request duration by endpoint and status.", ConstLabels: labels,
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30},
		}, []string{"endpoint", "status"}),
		matchPipelineSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "match_pipeline_seconds", Help: "Matching pipeline duration in seconds.", ConstLabels: labels,
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60},
		}),
		matchTop1Score: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "match_top1_score", Help: "Best score per scored report.", ConstLabels: labels,
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.55, 0.6, 0.7, 0.72, 0.8, 0.85, 0.9, 0.95, 1.0},
		}),
		matchCandidatesStored: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "match_candidates_stored", Help: "Candidates stored per match.report run.", ConstLabels: labels,
			Buckets: []float64{0, 1, 2, 5, 10, 20, 50, 100},
		}),
	}
	reg.MustRegister(
		h.requests,
		h.duration,
		h.inflight,
		h.sseConnections,
		h.authLoginTotal,
		h.rateLimitedTotal,
		h.domainTransitions,
		h.riverJobsTotal,
		h.riverJobDuration,
		h.riverQueueDepth,
		h.emailsSentTotal,
		h.mlRequestDuration,
		h.matchPipelineSeconds,
		h.matchTop1Score,
		h.matchCandidatesStored,
	)
	return h
}

func (h *HTTP) Registry() *prometheus.Registry { return h.registry }

func (h *HTTP) RegisterPool(pool *pgxpool.Pool) {
	if h != nil && pool != nil {
		h.registry.MustRegister(newDBPoolCollector(pool, prometheus.Labels{"service": h.service}))
	}
}

func (h *HTTP) InflightMiddleware(next http.Handler) http.Handler {
	if h == nil || h.inflight == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.inflight.Inc()
		defer h.inflight.Dec()
		next.ServeHTTP(w, r)
	})
}

func (h *HTTP) IncSSE() {
	if h != nil && h.sseConnections != nil {
		h.sseConnections.Inc()
	}
}

func (h *HTTP) DecSSE() {
	if h != nil && h.sseConnections != nil {
		h.sseConnections.Dec()
	}
}

func (h *HTTP) ObserveLogin(result string) {
	if h != nil && h.authLoginTotal != nil {
		h.authLoginTotal.WithLabelValues(result).Inc()
	}
}

func (h *HTTP) ObserveRateLimited(limit string) {
	if h != nil && h.rateLimitedTotal != nil {
		h.rateLimitedTotal.WithLabelValues(limit).Inc()
	}
}

func (h *HTTP) ObserveDomainTransition(machine, transition string) {
	if h != nil && h.domainTransitions != nil {
		h.domainTransitions.WithLabelValues(machine, transition).Inc()
	}
}

func (h *HTTP) ObserveRiverJob(kind, state string, d time.Duration) {
	if h != nil {
		if h.riverJobsTotal != nil {
			h.riverJobsTotal.WithLabelValues(kind, state).Inc()
		}
		if h.riverJobDuration != nil {
			h.riverJobDuration.WithLabelValues(kind).Observe(d.Seconds())
		}
	}
}

func (h *HTTP) SetRiverQueueDepth(queue string, depth int) {
	if h != nil && h.riverQueueDepth != nil {
		h.riverQueueDepth.WithLabelValues(queue).Set(float64(depth))
	}
}

func (h *HTTP) ObserveEmailSent(event, result string) {
	if h != nil && h.emailsSentTotal != nil {
		h.emailsSentTotal.WithLabelValues(event, result).Inc()
	}
}

func (h *HTTP) Observe(r *http.Request, route string, status int, d time.Duration) {
	h.requests.WithLabelValues(route, r.Method, strconv.Itoa(status)).Inc()
	h.duration.WithLabelValues(route, r.Method).Observe(d.Seconds())
}

func (h *HTTP) ObserveMLRequest(endpoint string, status int, d time.Duration) {
	if h != nil && h.mlRequestDuration != nil {
		h.mlRequestDuration.WithLabelValues(endpoint, strconv.Itoa(status)).Observe(d.Seconds())
	}
}

func (h *HTTP) ObserveMatchPipeline(d time.Duration) {
	if h != nil && h.matchPipelineSeconds != nil {
		h.matchPipelineSeconds.Observe(d.Seconds())
	}
}

func (h *HTTP) ObserveMatchTop1(score float64) {
	if h != nil && h.matchTop1Score != nil {
		h.matchTop1Score.Observe(score)
	}
}

func (h *HTTP) ObserveMatchCandidates(count int) {
	if h != nil && h.matchCandidatesStored != nil {
		h.matchCandidatesStored.Observe(float64(count))
	}
}

func (h *HTTP) Handler() http.Handler {
	return promhttp.HandlerFor(h.registry, promhttp.HandlerOpts{})
}
