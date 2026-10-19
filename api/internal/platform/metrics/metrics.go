package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HTTP struct {
	registry              *prometheus.Registry
	requests              *prometheus.CounterVec
	duration              *prometheus.HistogramVec
	mlRequestDuration     *prometheus.HistogramVec
	matchPipelineSeconds  prometheus.Histogram
	matchTop1Score        prometheus.Histogram
	matchCandidatesStored prometheus.Histogram
}

func NewHTTP(service string) *HTTP {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	labels := prometheus.Labels{"service": service}
	h := &HTTP{
		registry: reg,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total", Help: "HTTP requests by route, method and status.", ConstLabels: labels,
		}, []string{"route", "method", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "http_request_duration_seconds", Help: "HTTP request latency by route and method.", ConstLabels: labels,
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.5, 1, 2, 5, 10},
		}, []string{"route", "method"}),
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
	reg.MustRegister(h.requests, h.duration, h.mlRequestDuration, h.matchPipelineSeconds, h.matchTop1Score, h.matchCandidatesStored)
	return h
}

func (h *HTTP) Registry() *prometheus.Registry { return h.registry }

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
