package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pisondev/simpanugm/api/internal/platform/metrics"
)

func TestMetricsExposed(t *testing.T) {
	m := metrics.NewHTTP("api")
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	m.Observe(req, "/healthz", http.StatusOK, 10*time.Millisecond)
	m.ObserveMLRequest("/v1/embed/text", http.StatusOK, 25*time.Millisecond)
	m.ObserveMatchPipeline(150 * time.Millisecond)
	m.ObserveMatchTop1(0.88)
	m.ObserveMatchCandidates(5)

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	for _, expected := range []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"ml_request_duration_seconds",
		"match_pipeline_seconds",
		"match_top1_score",
		"match_candidates_stored",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected metrics output to contain %q, but was missing", expected)
		}
	}
}
