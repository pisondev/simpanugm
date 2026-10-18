package auth

import (
	"context"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/textx"
)

const emailChangeTTL = 24 * time.Hour

// a nil field keeps the stored value, an empty phone or NIM clears it
type ProfileInput struct {
	FullName *string
	Phone    *string
	NimNip   *string
}

// FR-AUTH-10
func (s *Service) UpdateProfile(ctx context.Context, p *access.Principal, in ProfileInput) (sqlc.User, error) {
	now := s.clock.Now()
	var out sqlc.User
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		u, err := q.GetUserByID(ctx, p.UserID)
		if err != nil {
			return err
		}
		name, phone, nim := u.FullName, u.Phone, u.NimNip
		var v apperr.Collector
		if in.FullName != nil {
			name = strings.TrimSpace(*in.FullName)
			v.Check(textx.Len(name) >= 2 && textx.Len(name) <= 100, "full_name", "invalid_length", "Nama 2 sampai 100 karakter")
		}
		if in.Phone != nil {
			phone = nil
			if raw := strings.TrimSpace(*in.Phone); raw != "" {
				normalized := textx.NormalizePhone(raw)
				v.Check(normalized != "", "phone", "invalid", "Nomor telepon tidak valid")
				phone = &normalized
			}
		}
		if in.NimNip != nil {
			nim = nil
			if raw := strings.TrimSpace(*in.NimNip); raw != "" {
				v.Check(nimPattern.MatchString(raw), "nim_nip", "invalid", "NIM atau NIP tidak valid")
				nim = &raw
			}
		}
		if err := v.Err(); err != nil {
			return err
		}
		out, err = q.UpdateProfile(ctx, sqlc.UpdateProfileParams{ID: u.ID, FullName: name, Phone: phone, NimNip: nim, UpdatedAt: now})
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "account.update_profile", EntityType: "user", EntityID: u.ID, Changes: map[string]any{
			"full_name": audit.Change(u.FullName, name), "phone": audit.Change(deref(u.Phone), deref(phone)), "nim_nip": audit.Change(deref(u.NimNip), deref(nim)),
		}})
	})
	return out, err
}

// FR-AUTH-10: the new address is confirmed by a link sent to it, the old one stays until then
func (s *Service) StartEmailChange(ctx context.Context, p *access.Principal, newEmail, password string) error {
	email := NormalizeEmail(newEmail)
	var v apperr.Collector
	if !validEmail(email) {
		v.Add("email", "invalid", "Format email tidak valid")
	} else if p.Role == access.Member && !slices.Contains(s.settings.AllowedEmailDomains(ctx), emailDomain(email)) {
		v.Add("email", "email_domain_not_allowed", "Gunakan email UGM (@mail.ugm.ac.id atau @ugm.ac.id)")
	} else if email == NormalizeEmail(p.Email) {
		v.Add("email", "unchanged", "Email baru sama dengan email sekarang")
	}
	if err := v.Err(); err != nil {
		return err
	}
	if err := s.checkCurrentPassword(ctx, p, password); err != nil {
		return err
	}
	now := s.clock.Now()
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		stats, err := q.TokenIssueStats(ctx, sqlc.TokenIssueStatsParams{UserID: p.UserID, Purpose: "change_email", Since: now.Add(-24 * time.Hour)})
		if err != nil {
			return err
		}
		if wait := stats.LastIssuedAt.Add(time.Minute).Sub(now); wait > 0 {
			return apperr.RateLimited(wait)
		}
		if stats.IssuedToday >= 5 {
			return apperr.RateLimited(24 * time.Hour)
		}
		// an address that already has an account gets nothing, the answer to the caller stays the same
		if _, err := q.GetUserByEmail(ctx, email); err == nil {
			return nil
		} else if !db.IsNoRows(err) {
			return err
		}
		u, err := q.GetUserByID(ctx, p.UserID)
		if err != nil {
			return err
		}
		if err := q.InvalidateUserTokens(ctx, sqlc.InvalidateUserTokensParams{Now: now, UserID: u.ID, Purpose: "change_email"}); err != nil {
			return err
		}
		plain, hash := ids.Token()
		if err := q.CreateAuthToken(ctx, sqlc.CreateAuthTokenParams{
			ID: ids.New(), UserID: u.ID, Purpose: "change_email", TokenHash: hash, NewEmail: &email, ExpiresAt: now.Add(emailChangeTTL), CreatedAt: now,
		}); err != nil {
			return err
		}
		link := s.mailer.BaseURL() + "/konfirmasi-email?token=" + plain
		if err := s.mailer.Enqueue(ctx, tx, notify.EmailChange, email, notify.Data{Name: u.FullName, Link: link, Expires: "24 jam"}); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "account.email_change_requested", EntityType: "user", EntityID: u.ID})
	})
}

func (s *Service) ConfirmEmailChange(ctx context.Context, token string) error {
	now := s.clock.Now()
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		t, err := q.GetUsableAuthToken(ctx, sqlc.GetUsableAuthTokenParams{TokenHash: ids.HashToken(token), Purpose: "change_email", Now: now})
		if db.IsNoRows(err) || (err == nil && t.NewEmail == nil) {
			return invalidToken
		}
		if err != nil {
			return err
		}
		if _, err := q.GetUserByEmail(ctx, *t.NewEmail); err == nil {
			return apperr.Unprocessable("email_taken", "Email sudah terdaftar", "Alamat ini sudah dipakai akun lain.")
		} else if !db.IsNoRows(err) {
			return err
		}
		u, err := q.GetUserByID(ctx, t.UserID)
		if err != nil {
			return err
		}
		if err := q.MarkAuthTokenUsed(ctx, sqlc.MarkAuthTokenUsedParams{ID: t.ID, UsedAt: &now}); err != nil {
			return err
		}
		if err := q.ChangeUserEmail(ctx, sqlc.ChangeUserEmailParams{ID: u.ID, Email: *t.NewEmail, EmailVerifiedAt: &now}); err != nil {
			if db.IsUniqueViolation(err, "users_email_key") {
				return apperr.Unprocessable("email_taken", "Email sudah terdaftar", "Alamat ini sudah dipakai akun lain.")
			}
			return err
		}
		return audit.Record(access.WithPrincipal(ctx, &access.Principal{UserID: u.ID, Role: access.Role(u.Role)}), q, now, audit.Entry{
			Action: "account.email_changed", EntityType: "user", EntityID: u.ID, Changes: map[string]any{"email": audit.Change(u.Email, *t.NewEmail)},
		})
	})
}

// the current session survives, every other one is signed out (account.password_changed says so)
func (s *Service) ChangePassword(ctx context.Context, p *access.Principal, current, next string) error {
	if err := s.checkCurrentPassword(ctx, p, current); err != nil {
		return err
	}
	if code, msg := PasswordProblem(next, p.Email); code != "" {
		return apperr.Validation(apperr.FieldError{Field: "new_password", Code: code, Message: msg})
	}
	hash, err := HashPassword(next)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.UpdatePasswordHash(ctx, sqlc.UpdatePasswordHashParams{ID: p.UserID, PasswordHash: &hash, UpdatedAt: now}); err != nil {
			return err
		}
		if _, err := q.RevokeOtherSessions(ctx, sqlc.RevokeOtherSessionsParams{UserID: p.UserID, ID: p.SessionID, RevokedAt: &now}); err != nil {
			return err
		}
		if err := s.mailer.Enqueue(ctx, tx, notify.PasswordChanged, p.Email, notify.Data{Name: p.FullName}); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "account.password_changed", EntityType: "user", EntityID: p.UserID})
	})
}

// a wrong current password counts like a failed sign in, BR-SEC-02 applies
func (s *Service) checkCurrentPassword(ctx context.Context, p *access.Principal, password string) error {
	key := NormalizeEmail(p.Email)
	if locked, wait := s.lockout.Locked(key); locked {
		burnVerify(password)
		return apperr.RateLimited(wait)
	}
	u, err := s.q.GetUserByID(ctx, p.UserID)
	if err != nil {
		return err
	}
	if u.PasswordHash == nil {
		return wrongCurrentPassword()
	}
	ok, _, err := VerifyPassword(password, *u.PasswordHash)
	if err != nil {
		return err
	}
	if !ok {
		s.lockout.Fail(key)
		return wrongCurrentPassword()
	}
	return nil
}

func wrongCurrentPassword() error {
	return apperr.Validation(apperr.FieldError{Field: "current_password", Code: "wrong_password", Message: "Kata sandi sekarang salah"})
}

type SessionInfo struct {
	ID         uuid.UUID
	CreatedAt  time.Time
	LastSeenAt time.Time
	IPHint     string
	Device     string
	Current    bool
}

// FR-AUTH-11, the address is shortened to its network so the list never shows a full IP
func (s *Service) Sessions(ctx context.Context, p *access.Principal) ([]SessionInfo, error) {
	rows, err := s.q.ListActiveSessions(ctx, sqlc.ListActiveSessionsParams{UserID: p.UserID, Now: s.clock.Now()})
	if err != nil {
		return nil, err
	}
	out := make([]SessionInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, SessionInfo{
			ID: r.ID, CreatedAt: r.CreatedAt, LastSeenAt: r.LastSeenAt, IPHint: ipHint(r.Ip), Device: Device(deref(r.UserAgent)), Current: r.ID == p.SessionID,
		})
	}
	return out, nil
}

func (s *Service) RevokeSession(ctx context.Context, p *access.Principal, id uuid.UUID) error {
	if id == p.SessionID {
		return apperr.Unprocessable("current_session", "Sesi ini sedang dipakai", "Gunakan Keluar untuk mengakhiri sesi yang sedang kamu pakai.")
	}
	n, err := s.q.RevokeUserSession(ctx, sqlc.RevokeUserSessionParams{ID: id, UserID: p.UserID, RevokedAt: ptr(s.clock.Now())})
	if err != nil {
		return err
	}
	if n == 0 {
		return apperr.NotFound()
	}
	return audit.Record(ctx, s.q, s.clock.Now(), audit.Entry{Action: "account.session_revoked", EntityType: "session", EntityID: id})
}

func ipHint(a *netip.Addr) string {
	if a == nil || !a.IsValid() {
		return ""
	}
	if a.Is4() || a.Is4In6() {
		b := a.Unmap().As4()
		return netip.AddrFrom4([4]byte{b[0], b[1], 0, 0}).String() + "/16"
	}
	prefix, err := a.Prefix(48)
	if err != nil {
		return ""
	}
	return prefix.String()
}

// a readable "browser di sistem" label, order matters because Edge and Opera also say Chrome
func Device(ua string) string {
	l := strings.ToLower(ua)
	browser := "Peramban lain"
	switch {
	case strings.Contains(l, "edg/"):
		browser = "Edge"
	case strings.Contains(l, "opr/") || strings.Contains(l, "opera"):
		browser = "Opera"
	case strings.Contains(l, "samsungbrowser"):
		browser = "Samsung Internet"
	case strings.Contains(l, "firefox/") || strings.Contains(l, "fxios"):
		browser = "Firefox"
	case strings.Contains(l, "chrome/") || strings.Contains(l, "crios"):
		browser = "Chrome"
	case strings.Contains(l, "safari/"):
		browser = "Safari"
	}
	system := "perangkat lain"
	switch {
	case strings.Contains(l, "android"):
		system = "Android"
	case strings.Contains(l, "iphone") || strings.Contains(l, "ipad") || strings.Contains(l, "ios"):
		system = "iOS"
	case strings.Contains(l, "windows"):
		system = "Windows"
	case strings.Contains(l, "mac os") || strings.Contains(l, "macintosh"):
		system = "macOS"
	case strings.Contains(l, "linux"):
		system = "Linux"
	}
	return browser + " di " + system
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptr[T any](v T) *T { return &v }
