package notify

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// fans PostgreSQL notifications out to the open streams of this process (notifications section 6)
type Hub struct {
	pool *pgxpool.Pool
	mu   sync.Mutex
	subs map[uuid.UUID]map[chan struct{}]struct{}
}

func NewHub(pool *pgxpool.Pool) *Hub {
	return &Hub{pool: pool, subs: map[uuid.UUID]map[chan struct{}]struct{}{}}
}

// the channel carries no data, a wake up means "read what is new", so a dropped signal costs nothing
func (h *Hub) Subscribe(user uuid.UUID) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	if h.subs[user] == nil {
		h.subs[user] = map[chan struct{}]struct{}{}
	}
	h.subs[user][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs[user], ch)
		if len(h.subs[user]) == 0 {
			delete(h.subs, user)
		}
		h.mu.Unlock()
	}
}

func (h *Hub) wake(user uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[user] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (h *Hub) wakeAll() {
	h.mu.Lock()
	users := make([]uuid.UUID, 0, len(h.subs))
	for u := range h.subs {
		users = append(users, u)
	}
	h.mu.Unlock()
	for _, u := range users {
		h.wake(u)
	}
}

// keeps one LISTEN connection open until ctx ends, reconnecting with backoff
func (h *Hub) Run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		if err := h.listen(ctx); err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "notification listener lost, reconnecting", "error", err.Error(), "in", backoff.String())
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, 30*time.Second)
			continue
		}
		backoff = time.Second
	}
}

func (h *Hub) listen(ctx context.Context) error {
	acquired, err := h.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	// taken out of the pool, a connection still listening must never serve other queries
	conn := acquired.Hijack()
	defer func() { _ = conn.Close(context.Background()) }()
	if _, err := conn.Exec(ctx, "LISTEN notifications"); err != nil {
		return err
	}
	// anything sent while the listener was down is caught up by every stream
	h.wakeAll()
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		if user, err := uuid.Parse(n.Payload); err == nil {
			h.wake(user)
		}
	}
}
