package reqctx

import (
	"context"
	"net/netip"

	"github.com/google/uuid"
)

type Meta struct {
	RequestID string
	IP        netip.Addr
	UserAgent string
	UserID    uuid.UUID
	Role      string
}

type key struct{}

func With(ctx context.Context, m *Meta) context.Context {
	return context.WithValue(ctx, key{}, m)
}

func From(ctx context.Context) *Meta {
	if m, ok := ctx.Value(key{}).(*Meta); ok {
		return m
	}
	return &Meta{}
}
