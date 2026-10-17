package logx

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/platform/reqctx"
)

var redacted = map[string]bool{
	"password": true, "token": true, "code": true, "cookie": true, "authorization": true,
	"hidden_details": true, "private_details": true, "answers": true, "statement": true, "description": true,
}

func New(w io.Writer, level, service, version string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: lvl,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				a.Key = "ts"
			case slog.MessageKey:
				a.Key = "msg"
			}
			if redacted[strings.ToLower(a.Key)] {
				return slog.String(a.Key, "[redacted]")
			}
			return a
		},
	})
	return slog.New(&ctxHandler{h}).With("service", service, "version", version)
}

type ctxHandler struct{ slog.Handler }

func (h *ctxHandler) Handle(ctx context.Context, r slog.Record) error {
	m := reqctx.From(ctx)
	if m.RequestID != "" {
		r.AddAttrs(slog.String("request_id", m.RequestID))
	}
	if m.UserID != uuid.Nil {
		r.AddAttrs(slog.String("user_id", m.UserID.String()), slog.String("role", m.Role))
	}
	return h.Handler.Handle(ctx, r)
}

func (h *ctxHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ctxHandler{h.Handler.WithAttrs(attrs)}
}

func (h *ctxHandler) WithGroup(name string) slog.Handler {
	return &ctxHandler{h.Handler.WithGroup(name)}
}
