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
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
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
	}
	reg.MustRegister(h.requests, h.duration)
	return h
}

func (h *HTTP) Registry() *prometheus.Registry { return h.registry }

func (h *HTTP) Observe(r *http.Request, route string, status int, d time.Duration) {
	h.requests.WithLabelValues(route, r.Method, strconv.Itoa(status)).Inc()
	h.duration.WithLabelValues(route, r.Method).Observe(d.Seconds())
}

func (h *HTTP) Handler() http.Handler {
	return promhttp.HandlerFor(h.registry, promhttp.HandlerOpts{})
}
