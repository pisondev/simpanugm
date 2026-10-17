package httpx

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestClientIP(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	cases := []struct {
		name, remote, xff, want string
	}{
		{"direct client ignores the header", "203.0.113.9:5000", "1.2.3.4", "203.0.113.9"},
		{"behind the proxy", "10.0.0.5:5000", "198.51.100.7", "198.51.100.7"},
		{"forged left hop is skipped", "10.0.0.5:5000", "1.2.3.4, 198.51.100.7", "198.51.100.7"},
		{"chain of proxies", "10.0.0.5:5000", "198.51.100.7, 10.0.0.9", "198.51.100.7"},
		{"no header behind proxy", "10.0.0.5:5000", "", "10.0.0.5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = c.remote
			if c.xff != "" {
				r.Header.Set("X-Forwarded-For", c.xff)
			}
			if got := ClientIP(r, trusted).String(); got != c.want {
				t.Fatalf("got %s, want %s", got, c.want)
			}
		})
	}
}

func TestRequestIDPattern(t *testing.T) {
	if !requestIDPattern.MatchString("01JBX9Q4Z8ABCDEF") || requestIDPattern.MatchString("bad id with spaces") || requestIDPattern.MatchString("short") {
		t.Fatal("request id pattern")
	}
}

func TestCookiesWaitForHeaders(t *testing.T) {
	h := Observe(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetCookieCtx(r.Context(), &http.Cookie{Name: "simpan_session", Value: "v"})
		w.WriteHeader(http.StatusCreated)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusCreated || rec.Header().Get("Set-Cookie") == "" {
		t.Fatalf("status %d cookie %q", rec.Code, rec.Header().Get("Set-Cookie"))
	}
}

func TestPanicsBecomeProblems(t *testing.T) {
	h := Observe(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError || rec.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("status %d type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
}
