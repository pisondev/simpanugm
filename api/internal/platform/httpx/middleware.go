package httpx

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/reqctx"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

func RequestMeta(trusted []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-Id")
			if !requestIDPattern.MatchString(id) {
				id = ids.New().String()
			}
			w.Header().Set("X-Request-Id", id)
			ua := r.UserAgent()
			if len(ua) > 300 {
				ua = ua[:300]
			}
			m := &reqctx.Meta{RequestID: id, IP: ClientIP(r, trusted), UserAgent: ua}
			next.ServeHTTP(w, r.WithContext(reqctx.With(r.Context(), m)))
		})
	}
}

// the first untrusted hop from the right is the client, so a forged header cannot skip the proxy
func ClientIP(r *http.Request, trusted []netip.Prefix) netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remote, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	remote = remote.Unmap()
	if !isTrusted(remote, trusted) {
		return remote
	}
	hops := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		a, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			continue
		}
		a = a.Unmap()
		if !isTrusted(a, trusted) {
			return a
		}
	}
	return remote
}

func isTrusted(a netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

type statusRecorder struct {
	http.ResponseWriter
	status  int
	bytes   int
	cookies []*http.Cookie
	wrote   bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.wrote {
		return
	}
	s.wrote = true
	s.status = code
	for _, c := range s.cookies {
		http.SetCookie(s.ResponseWriter, c)
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wrote {
		s.WriteHeader(http.StatusOK)
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		if !s.wrote {
			s.WriteHeader(http.StatusOK)
		}
		f.Flush()
	}
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func (s *statusRecorder) addCookie(c *http.Cookie) { s.cookies = append(s.cookies, c) }

// strict handlers cannot touch headers, so cookies wait here until the status is written
func SetCookie(w http.ResponseWriter, c *http.Cookie) bool {
	for {
		if s, ok := w.(*statusRecorder); ok {
			s.addCookie(c)
			return true
		}
		u, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return false
		}
		w = u.Unwrap()
	}
}

type recorderKey struct{}

func SetCookieCtx(ctx context.Context, c *http.Cookie) bool {
	rec, ok := ctx.Value(recorderKey{}).(*statusRecorder)
	if ok {
		rec.addCookie(c)
	}
	return ok
}

func Observe(observe func(r *http.Request, route string, status int, d time.Duration)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			r = r.WithContext(context.WithValue(r.Context(), recorderKey{}, rec))
			defer func() {
				if v := recover(); v != nil {
					slog.ErrorContext(r.Context(), "panic", "panic", v, "stack", string(debug.Stack()))
					if !rec.wrote {
						WriteProblem(rec, r, apperr.New(http.StatusInternalServerError, "internal", "Terjadi kesalahan", "Terjadi kesalahan di sisi kami."))
					}
				}
				route := "unmatched"
				if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
					route = rc.RoutePattern()
				}
				d := time.Since(start)
				if observe != nil {
					observe(r, route, rec.status, d)
				}
				level := slog.LevelInfo
				if rec.status >= 500 {
					level = slog.LevelError
				}
				if route == "/healthz" || route == "/readyz" || route == "/api/healthz" {
					level = slog.LevelDebug
				}
				slog.Log(r.Context(), level, "request",
					"method", r.Method, "route", route, "status", rec.status,
					"duration_ms", d.Milliseconds(), "bytes", rec.bytes, "ip", reqctx.From(r.Context()).IP.String())
			}()
			next.ServeHTTP(rec, r)
		})
	}
}

func LimitBody(defaultLimit int64, overrides map[string]int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit := defaultLimit
			if l, ok := overrides[r.URL.Path]; ok {
				limit = l
			}
			if r.ContentLength > limit {
				WriteProblem(w, r, apperr.New(http.StatusRequestEntityTooLarge, "payload_too_large", "Data terlalu besar", "Ukuran data melebihi batas."))
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// conditional GET for master data, FR-MD-08
func ETag(match func(r *http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || !match(r) {
				next.ServeHTTP(w, r)
				return
			}
			buf := &bufferedWriter{header: http.Header{}, status: http.StatusOK}
			next.ServeHTTP(buf, r)
			for k, v := range buf.header {
				w.Header()[k] = v
			}
			if buf.status == http.StatusOK {
				sum := sha256.Sum256(buf.body.Bytes())
				tag := `"` + hex.EncodeToString(sum[:12]) + `"`
				w.Header().Set("ETag", tag)
				w.Header().Set("Cache-Control", "private, no-cache")
				if r.Header.Get("If-None-Match") == tag {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
			w.WriteHeader(buf.status)
			_, _ = w.Write(buf.body.Bytes())
		})
	}
}

type bufferedWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (b *bufferedWriter) Header() http.Header         { return b.header }
func (b *bufferedWriter) WriteHeader(code int)        { b.status = code }
func (b *bufferedWriter) Write(p []byte) (int, error) { return b.body.Write(p) }
