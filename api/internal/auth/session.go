package auth

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/httpx"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/reqctx"
)

type Cookies struct {
	Secure bool
}

func (c Cookies) Session(s Session, now time.Time) *http.Cookie {
	return &http.Cookie{ //nolint:gosec // Secure is configurable and only false in development over plain http
		Name: CookieName, Value: s.Token, Path: "/",
		MaxAge:   int(s.ExpiresAt.Sub(now).Seconds()),
		HttpOnly: true, Secure: c.Secure, SameSite: http.SameSiteLaxMode,
	}
}

func (c Cookies) Clear() *http.Cookie {
	return &http.Cookie{Name: CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: c.Secure, SameSite: http.SameSiteLaxMode} //nolint:gosec // see Session
}

// never rejects a request, the operation rules decide what an anonymous caller may do
func (s *Service) Authenticate(cookies Cookies) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(CookieName)
			if err != nil || c.Value == "" {
				next.ServeHTTP(w, r)
				return
			}
			p, refreshed, err := s.resolve(r.Context(), c.Value)
			if err != nil {
				slog.ErrorContext(r.Context(), "session lookup failed", "error", err.Error())
				httpx.WriteError(w, r, err)
				return
			}
			if p == nil {
				httpx.SetCookie(w, cookies.Clear())
				next.ServeHTTP(w, r)
				return
			}
			if refreshed != nil {
				httpx.SetCookie(w, cookies.Session(*refreshed, s.clock.Now()))
			}
			m := reqctx.From(r.Context())
			m.UserID, m.Role = p.UserID, string(p.Role)
			next.ServeHTTP(w, r.WithContext(access.WithPrincipal(r.Context(), p)))
		})
	}
}

func (s *Service) resolve(ctx context.Context, token string) (*access.Principal, *Session, error) {
	row, err := s.q.GetSessionByTokenHash(ctx, ids.HashToken(token))
	if db.IsNoRows(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	now := s.clock.Now()
	if row.RevokedAt != nil || !now.Before(row.ExpiresAt) || (row.IdleExpiresAt != nil && !now.Before(*row.IdleExpiresAt)) ||
		row.Status == "suspended" || row.Status == "deleted" {
		return nil, nil, nil
	}
	role := access.Role(row.Role)
	p := &access.Principal{
		UserID: row.UserID, SessionID: row.SessionID, Role: role, Status: row.Status,
		Email: row.Email, FullName: row.FullName,
		EmailVerified: row.EmailVerifiedAt != nil, ClaimRestricted: row.ClaimRestrictedAt != nil,
	}
	if role != access.Member {
		posts, err := s.q.ListUserPostIDs(ctx, row.UserID)
		if err != nil {
			return nil, nil, err
		}
		p.PostIDs = posts
	}
	var refreshed *Session
	switch {
	case role == access.Member && now.Sub(row.LastSeenAt) >= memberTouchEvery:
		expires := now.Add(memberSessionTTL)
		if err := s.q.TouchSession(ctx, sqlc.TouchSessionParams{Now: now, ExpiresAt: expires, ID: row.SessionID}); err != nil {
			return nil, nil, err
		}
		refreshed = &Session{Token: token, ExpiresAt: expires}
	case role != access.Member && now.Sub(row.LastSeenAt) >= staffTouchEvery:
		idle := now.Add(staffIdleTTL)
		if err := s.q.TouchSession(ctx, sqlc.TouchSessionParams{Now: now, ExpiresAt: row.ExpiresAt, IdleExpiresAt: &idle, ID: row.SessionID}); err != nil {
			return nil, nil, err
		}
	}
	return p, refreshed, nil
}

func (s *Service) PrincipalForUser(ctx context.Context, u sqlc.User, session uuid.UUID) (*access.Principal, error) {
	p := &access.Principal{
		UserID: u.ID, SessionID: session, Role: access.Role(u.Role), Status: u.Status, Email: u.Email, FullName: u.FullName,
		EmailVerified: u.EmailVerifiedAt != nil, ClaimRestricted: u.ClaimRestrictedAt != nil,
	}
	if p.Role != access.Member {
		posts, err := s.q.ListUserPostIDs(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		p.PostIDs = posts
	}
	return p, nil
}
