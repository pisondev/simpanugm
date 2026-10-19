package auth

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/ratelimit"
	"github.com/pisondev/simpanugm/api/internal/platform/settings"
	"github.com/pisondev/simpanugm/api/internal/platform/textx"
)

const (
	CookieName            = "simpan_session"
	CurrentPrivacyVersion = "2026.10"

	memberSessionTTL = 30 * 24 * time.Hour
	staffSessionTTL  = 12 * time.Hour
	staffIdleTTL     = 60 * time.Minute
	memberTouchEvery = 5 * time.Minute
	staffTouchEvery  = time.Minute

	verifyTTL = 24 * time.Hour
	resetTTL  = 30 * time.Minute
	inviteTTL = 72 * time.Hour
)

var invalidCredentials = apperr.New(401, "invalid_credentials", "Gagal masuk", "Email atau kata sandi salah")

var invalidToken = apperr.Unprocessable("token_invalid", "Tautan tidak berlaku", "Tautan tidak valid atau sudah kedaluwarsa.")

var errRaceRegistered = errors.New("auth: email registered concurrently")

type Service struct {
	pool     *pgxpool.Pool
	q        *sqlc.Queries
	clock    clock.Clock
	mailer   *notify.Mailer
	settings *settings.Store

	loginPair *ratelimit.Limiter
	loginIP   *ratelimit.Limiter
	register  *ratelimit.Limiter
	forgot    *ratelimit.Limiter
	lockout   *ratelimit.Lockout
}

func NewService(pool *pgxpool.Pool, c clock.Clock, mailer *notify.Mailer, st *settings.Store) *Service {
	now := c.Now
	return &Service{
		pool: pool, q: sqlc.New(pool), clock: c, mailer: mailer, settings: st,
		loginPair: ratelimit.New(5, 15*time.Minute, now),
		loginIP:   ratelimit.New(20, 15*time.Minute, now),
		register:  ratelimit.New(5, time.Hour, now),
		forgot:    ratelimit.New(3, time.Hour, now),
		lockout:   ratelimit.NewLockout(5, 15*time.Minute, now),
	}
}

// the default of five an hour per address stays for production, the e2e stack lifts it
func (s *Service) LimitRegistrations(perHour int) {
	s.register = ratelimit.New(perHour, time.Hour, s.clock.Now)
}

type Session struct {
	Token     string
	ExpiresAt time.Time
}

type RegisterInput struct {
	FullName       string
	Email          string
	Password       string
	NimNip         string
	Phone          string
	PrivacyVersion string
	AcceptPrivacy  bool
	IP             netip.Addr
}

var nimPattern = regexp.MustCompile(`^[A-Za-z0-9./ ]{1,30}$`)

func NormalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func validEmail(s string) bool {
	local, domain, ok := strings.Cut(s, "@")
	return ok && local != "" && len(s) <= 254 && strings.Contains(domain, ".") && !strings.ContainsAny(s, " \t\r\n") && !strings.Contains(domain, "@")
}

func emailDomain(s string) string {
	_, d, _ := strings.Cut(s, "@")
	return d
}

func optional(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func (s *Service) Register(ctx context.Context, in RegisterInput) error {
	if ok, wait := s.register.Allow("ip:" + in.IP.String()); !ok {
		return apperr.RateLimited(wait)
	}
	email := NormalizeEmail(in.Email)
	name := strings.TrimSpace(in.FullName)
	var v apperr.Collector
	v.Check(len([]rune(name)) >= 2 && len([]rune(name)) <= 100, "full_name", "invalid_length", "Nama 2 sampai 100 karakter")
	if !validEmail(email) {
		v.Add("email", "invalid", "Format email tidak valid")
	} else if !slices.Contains(s.settings.AllowedEmailDomains(ctx), emailDomain(email)) {
		v.Add("email", "email_domain_not_allowed", "Gunakan email UGM (@mail.ugm.ac.id atau @ugm.ac.id)")
	}
	if code, msg := PasswordProblem(in.Password, email); code != "" {
		v.Add("password", code, msg)
	}
	nim := strings.TrimSpace(in.NimNip)
	v.Check(nim == "" || nimPattern.MatchString(nim), "nim_nip", "invalid", "NIM atau NIP tidak valid")
	phone := ""
	if strings.TrimSpace(in.Phone) != "" {
		phone = textx.NormalizePhone(in.Phone)
		v.Check(phone != "", "phone", "invalid", "Nomor telepon tidak valid")
	}
	v.Check(in.AcceptPrivacy, "accept_privacy", "required", "Setujui pemberitahuan privasi untuk melanjutkan")
	v.Check(in.PrivacyVersion == CurrentPrivacyVersion, "privacy_version", "outdated", "Muat ulang halaman untuk membaca pemberitahuan privasi terbaru")
	if err := v.Err(); err != nil {
		return err
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	err = db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		existing, err := q.GetUserByEmail(ctx, email)
		if err == nil {
			if existing.Status == "deleted" {
				return nil
			}
			return s.mailer.Enqueue(ctx, tx, notify.RegisterAttempt, existing.Email, notify.Data{Name: existing.FullName})
		}
		if !db.IsNoRows(err) {
			return err
		}
		version := CurrentPrivacyVersion
		u, err := q.CreateUser(ctx, sqlc.CreateUserParams{
			ID: ids.New(), Email: email, FullName: name, NimNip: optional(nim), Phone: optional(phone),
			PasswordHash: &hash, Role: string(access.Member), Status: "pending_verification",
			PrivacyVersion: &version, PrivacyAcceptedAt: &now,
		})
		if db.IsUniqueViolation(err, "users_email_key") {
			return errRaceRegistered
		}
		if err != nil {
			return err
		}
		if err := s.issueToken(ctx, tx, q, u, "verify_email", verifyTTL, notify.VerifyEmail, "/verifikasi", "24 jam"); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "account.register", EntityType: "user", EntityID: u.ID})
	})
	if errors.Is(err, errRaceRegistered) {
		return nil
	}
	return err
}

func (s *Service) issueToken(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, u sqlc.User, purpose string, ttl time.Duration, event notify.Event, path, expires string) error {
	now := s.clock.Now()
	if err := q.InvalidateUserTokens(ctx, sqlc.InvalidateUserTokensParams{Now: now, UserID: u.ID, Purpose: purpose}); err != nil {
		return err
	}
	plain, hash := ids.Token()
	if err := q.CreateAuthToken(ctx, sqlc.CreateAuthTokenParams{
		ID: ids.New(), UserID: u.ID, Purpose: purpose, TokenHash: hash, ExpiresAt: now.Add(ttl), CreatedAt: now,
	}); err != nil {
		return err
	}
	link := s.mailer.BaseURL() + path + "?token=" + plain
	return s.mailer.Enqueue(ctx, tx, event, u.Email, notify.Data{Name: u.FullName, Link: link, Expires: expires})
}

func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	now := s.clock.Now()
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		t, err := q.GetUsableAuthToken(ctx, sqlc.GetUsableAuthTokenParams{TokenHash: ids.HashToken(token), Purpose: "verify_email", Now: now})
		if db.IsNoRows(err) {
			return invalidToken
		}
		if err != nil {
			return err
		}
		if err := q.MarkAuthTokenUsed(ctx, sqlc.MarkAuthTokenUsedParams{ID: t.ID, UsedAt: &now}); err != nil {
			return err
		}
		if err := q.MarkEmailVerified(ctx, sqlc.MarkEmailVerifiedParams{Now: now, ID: t.UserID}); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "account.verify_email", EntityType: "user", EntityID: t.UserID})
	})
}

func (s *Service) ResendVerification(ctx context.Context, p *access.Principal) error {
	if p.EmailVerified {
		return nil
	}
	now := s.clock.Now()
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		stats, err := q.TokenIssueStats(ctx, sqlc.TokenIssueStatsParams{UserID: p.UserID, Purpose: "verify_email", Since: now.Add(-24 * time.Hour)})
		if err != nil {
			return err
		}
		if wait := stats.LastIssuedAt.Add(time.Minute).Sub(now); wait > 0 {
			return apperr.RateLimited(wait)
		}
		if stats.IssuedToday >= 5 {
			return apperr.RateLimited(24 * time.Hour)
		}
		u, err := q.GetUserByID(ctx, p.UserID)
		if err != nil {
			return err
		}
		return s.issueToken(ctx, tx, q, u, "verify_email", verifyTTL, notify.VerifyEmail, "/verifikasi", "24 jam")
	})
}

type LoginInput struct {
	Email     string
	Password  string
	IP        netip.Addr
	UserAgent string
}

func (s *Service) Login(ctx context.Context, in LoginInput) (Session, sqlc.User, error) {
	email := NormalizeEmail(in.Email)
	if ok, wait := s.loginIP.Allow("ip:" + in.IP.String()); !ok {
		return Session{}, sqlc.User{}, apperr.RateLimited(wait)
	}
	if ok, wait := s.loginPair.Allow(in.IP.String() + "|" + email); !ok {
		return Session{}, sqlc.User{}, apperr.RateLimited(wait)
	}
	if locked, wait := s.lockout.Locked(email); locked {
		burnVerify(in.Password)
		return Session{}, sqlc.User{}, apperr.RateLimited(wait)
	}
	u, err := s.q.GetUserByEmail(ctx, email)
	if db.IsNoRows(err) || (err == nil && (u.PasswordHash == nil || u.Status == "suspended" || u.Status == "deleted")) {
		burnVerify(in.Password)
		s.lockout.Fail(email)
		return Session{}, sqlc.User{}, invalidCredentials
	}
	if err != nil {
		return Session{}, sqlc.User{}, err
	}
	ok, rehash, err := VerifyPassword(in.Password, *u.PasswordHash)
	if err != nil {
		return Session{}, sqlc.User{}, err
	}
	if !ok {
		s.lockout.Fail(email)
		return Session{}, sqlc.User{}, invalidCredentials
	}
	s.lockout.Reset(email)
	now := s.clock.Now()
	var sess Session
	err = db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if rehash {
			if h, err := HashPassword(in.Password); err == nil {
				if err := q.UpdatePasswordHash(ctx, sqlc.UpdatePasswordHashParams{ID: u.ID, PasswordHash: &h, UpdatedAt: now}); err != nil {
					return err
				}
			}
		}
		var err error
		sess, err = s.createSession(ctx, q, u, in.IP, in.UserAgent)
		if err != nil {
			return err
		}
		if err := q.SetLastLogin(ctx, sqlc.SetLastLoginParams{ID: u.ID, LastLoginAt: &now}); err != nil {
			return err
		}
		return audit.Record(access.WithPrincipal(ctx, &access.Principal{UserID: u.ID, Role: access.Role(u.Role)}), q, now,
			audit.Entry{Action: "auth.login", EntityType: "user", EntityID: u.ID})
	})
	return sess, u, err
}

func (s *Service) createSession(ctx context.Context, q *sqlc.Queries, u sqlc.User, ip netip.Addr, ua string) (Session, error) {
	now := s.clock.Now()
	plain, hash := ids.Token()
	expires := now.Add(memberSessionTTL)
	var idle *time.Time
	if access.Role(u.Role) != access.Member {
		expires = now.Add(staffSessionTTL)
		t := now.Add(staffIdleTTL)
		idle = &t
	}
	var ipPtr *netip.Addr
	if ip.IsValid() {
		ipPtr = &ip
	}
	if err := q.CreateSession(ctx, sqlc.CreateSessionParams{
		ID: ids.New(), UserID: u.ID, TokenHash: hash, CreatedAt: now, ExpiresAt: expires,
		IdleExpiresAt: idle, Ip: ipPtr, UserAgent: optional(ua),
	}); err != nil {
		return Session{}, err
	}
	return Session{Token: plain, ExpiresAt: expires}, nil
}

func (s *Service) Logout(ctx context.Context, p *access.Principal, all bool) error {
	now := s.clock.Now()
	if all {
		_, err := s.q.RevokeUserSessions(ctx, sqlc.RevokeUserSessionsParams{UserID: p.UserID, RevokedAt: &now})
		return err
	}
	return s.q.RevokeSession(ctx, sqlc.RevokeSessionParams{ID: p.SessionID, RevokedAt: &now})
}

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	email = NormalizeEmail(email)
	if ok, wait := s.forgot.Allow(email); !ok {
		return apperr.RateLimited(wait)
	}
	u, err := s.q.GetUserByEmail(ctx, email)
	if db.IsNoRows(err) || (err == nil && (u.PasswordHash == nil || u.Status == "suspended" || u.Status == "deleted")) {
		return nil
	}
	if err != nil {
		return err
	}
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		return s.issueToken(ctx, tx, s.q.WithTx(tx), u, "reset_password", resetTTL, notify.PasswordReset, "/atur-sandi", "30 menit")
	})
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	now := s.clock.Now()
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		t, err := q.GetUsableAuthToken(ctx, sqlc.GetUsableAuthTokenParams{TokenHash: ids.HashToken(token), Purpose: "reset_password", Now: now})
		if db.IsNoRows(err) {
			return invalidToken
		}
		if err != nil {
			return err
		}
		u, err := q.GetUserByID(ctx, t.UserID)
		if err != nil {
			return err
		}
		if code, msg := PasswordProblem(password, u.Email); code != "" {
			return apperr.Validation(apperr.FieldError{Field: "password", Code: code, Message: msg})
		}
		hash, err := HashPassword(password)
		if err != nil {
			return err
		}
		if err := q.UpdatePasswordHash(ctx, sqlc.UpdatePasswordHashParams{ID: u.ID, PasswordHash: &hash, UpdatedAt: now}); err != nil {
			return err
		}
		if err := q.MarkAuthTokenUsed(ctx, sqlc.MarkAuthTokenUsedParams{ID: t.ID, UsedAt: &now}); err != nil {
			return err
		}
		if _, err := q.RevokeUserSessions(ctx, sqlc.RevokeUserSessionsParams{UserID: u.ID, RevokedAt: &now}); err != nil {
			return err
		}
		if err := s.mailer.Enqueue(ctx, tx, notify.PasswordChanged, u.Email, notify.Data{Name: u.FullName}); err != nil {
			return err
		}
		return audit.Record(access.WithPrincipal(ctx, &access.Principal{UserID: u.ID, Role: access.Role(u.Role)}), q, now,
			audit.Entry{Action: "auth.password_reset", EntityType: "user", EntityID: u.ID})
	})
}

func (s *Service) AcceptInvitation(ctx context.Context, token, password string, ip netip.Addr, ua string) (Session, sqlc.User, error) {
	now := s.clock.Now()
	var sess Session
	var user sqlc.User
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		t, err := q.GetUsableAuthToken(ctx, sqlc.GetUsableAuthTokenParams{TokenHash: ids.HashToken(token), Purpose: "invite", Now: now})
		if db.IsNoRows(err) {
			return invalidToken
		}
		if err != nil {
			return err
		}
		u, err := q.GetUserByID(ctx, t.UserID)
		if err != nil {
			return err
		}
		if code, msg := PasswordProblem(password, u.Email); code != "" {
			return apperr.Validation(apperr.FieldError{Field: "password", Code: code, Message: msg})
		}
		hash, err := HashPassword(password)
		if err != nil {
			return err
		}
		if err := q.ActivateInvitedUser(ctx, sqlc.ActivateInvitedUserParams{PasswordHash: &hash, Now: now, ID: u.ID}); err != nil {
			return err
		}
		if err := q.MarkAuthTokenUsed(ctx, sqlc.MarkAuthTokenUsedParams{ID: t.ID, UsedAt: &now}); err != nil {
			return err
		}
		if user, err = q.GetUserByID(ctx, u.ID); err != nil {
			return err
		}
		if sess, err = s.createSession(ctx, q, user, ip, ua); err != nil {
			return err
		}
		return audit.Record(access.WithPrincipal(ctx, &access.Principal{UserID: u.ID, Role: access.Role(u.Role)}), q, now,
			audit.Entry{Action: "account.accept_invitation", EntityType: "user", EntityID: u.ID})
	})
	return sess, user, err
}

type InviteInput struct {
	Email    string
	FullName string
	Role     access.Role
	PostIDs  []uuid.UUID
}

func (s *Service) Invite(ctx context.Context, in InviteInput) (sqlc.User, error) {
	email := NormalizeEmail(in.Email)
	name := strings.TrimSpace(in.FullName)
	var v apperr.Collector
	v.Check(validEmail(email), "email", "invalid", "Format email tidak valid")
	v.Check(len([]rune(name)) >= 2 && len([]rune(name)) <= 100, "full_name", "invalid_length", "Nama 2 sampai 100 karakter")
	v.Check(in.Role == access.Officer || in.Role == access.Admin, "role", "invalid", "Peran harus petugas atau admin")
	v.Check(in.Role != access.Officer || len(in.PostIDs) > 0, "post_ids", "required", "Petugas harus ditugaskan ke minimal satu pos")
	if err := v.Err(); err != nil {
		return sqlc.User{}, err
	}
	now := s.clock.Now()
	actor := access.FromContext(ctx)
	var user sqlc.User
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := s.checkPosts(ctx, q, in.PostIDs); err != nil {
			return err
		}
		var err error
		user, err = q.CreateUser(ctx, sqlc.CreateUserParams{
			ID: ids.New(), Email: email, FullName: name, Role: string(in.Role), Status: "pending_verification", InvitedBy: &actor.UserID,
		})
		if db.IsUniqueViolation(err, "users_email_key") {
			return apperr.Unprocessable("email_taken", "Email sudah terdaftar", "Email ini sudah memiliki akun. Ubah perannya dari daftar pengguna.")
		}
		if err != nil {
			return err
		}
		if err := s.replacePosts(ctx, q, user.ID, in.PostIDs, actor.UserID); err != nil {
			return err
		}
		if err := s.issueToken(ctx, tx, q, user, "invite", inviteTTL, notify.Invite, "/undangan", "72 jam"); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "user.invite", EntityType: "user", EntityID: user.ID,
			Changes: map[string]any{"role": in.Role, "post_ids": in.PostIDs}})
	})
	return user, err
}

func (s *Service) checkPosts(ctx context.Context, q *sqlc.Queries, postIDs []uuid.UUID) error {
	if len(postIDs) == 0 {
		return nil
	}
	n, err := q.CountActivePostsByIDs(ctx, uniqueIDs(postIDs))
	if err != nil {
		return err
	}
	if int(n) != len(uniqueIDs(postIDs)) {
		return apperr.Validation(apperr.FieldError{Field: "post_ids", Code: "unknown_post", Message: "Pos tidak ditemukan atau tidak aktif"})
	}
	return nil
}

func (s *Service) replacePosts(ctx context.Context, q *sqlc.Queries, user uuid.UUID, postIDs []uuid.UUID, by uuid.UUID) error {
	if err := q.DeleteOfficerPosts(ctx, user); err != nil {
		return err
	}
	now := s.clock.Now()
	for _, id := range uniqueIDs(postIDs) {
		if err := q.InsertOfficerPost(ctx, sqlc.InsertOfficerPostParams{UserID: user, PostID: id, AssignedAt: now, AssignedBy: &by}); err != nil {
			return err
		}
	}
	return nil
}

func uniqueIDs(in []uuid.UUID) []uuid.UUID {
	out := slices.Clone(in)
	slices.SortFunc(out, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return slices.Compact(out)
}

type UpdateUserInput struct {
	Role    *access.Role
	PostIDs *[]uuid.UUID
}

func (s *Service) UpdateUser(ctx context.Context, id uuid.UUID, in UpdateUserInput) (sqlc.User, error) {
	if in.Role != nil && !in.Role.Valid() {
		return sqlc.User{}, apperr.Validation(apperr.FieldError{Field: "role", Code: "invalid", Message: "Peran tidak valid"})
	}
	now := s.clock.Now()
	actor := access.FromContext(ctx)
	var out sqlc.User
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		u, err := q.GetUserByID(ctx, id)
		if db.IsNoRows(err) || (err == nil && u.Status == "deleted") {
			return apperr.NotFound()
		}
		if err != nil {
			return err
		}
		changes := map[string]any{}
		if in.Role != nil && string(*in.Role) != u.Role {
			if u.Role == string(access.Admin) && u.Status == "active" {
				admins, err := q.CountActiveAdmins(ctx)
				if err != nil {
					return err
				}
				if admins <= 1 {
					return apperr.Unprocessable("last_admin", "Admin terakhir", "Admin aktif terakhir tidak dapat diturunkan perannya.")
				}
			}
			if err := q.UpdateUserRole(ctx, sqlc.UpdateUserRoleParams{ID: id, Role: string(*in.Role), UpdatedAt: now}); err != nil {
				return err
			}
			if _, err := q.RevokeUserSessions(ctx, sqlc.RevokeUserSessionsParams{UserID: id, RevokedAt: &now}); err != nil {
				return err
			}
			changes["role"] = audit.Change(u.Role, *in.Role)
		}
		if in.PostIDs != nil {
			if err := s.checkPosts(ctx, q, *in.PostIDs); err != nil {
				return err
			}
			if err := s.replacePosts(ctx, q, id, *in.PostIDs, actor.UserID); err != nil {
				return err
			}
			changes["post_ids"] = uniqueIDs(*in.PostIDs)
		}
		if len(changes) > 0 {
			if err := audit.Record(ctx, q, now, audit.Entry{Action: "user.update", EntityType: "user", EntityID: id, Changes: changes}); err != nil {
				return err
			}
		}
		out, err = q.GetUserByID(ctx, id)
		return err
	})
	return out, err
}

func (s *Service) PostIDs(ctx context.Context, user uuid.UUID) ([]uuid.UUID, error) {
	return s.q.ListUserPostIDs(ctx, user)
}

func (s *Service) User(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	u, err := s.q.GetUserByID(ctx, id)
	if db.IsNoRows(err) {
		return u, apperr.NotFound()
	}
	return u, err
}

type ListUsersInput struct {
	Q      string
	Role   string
	Cursor *Cursor
	Limit  int
}

type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

func (s *Service) ListUsers(ctx context.Context, in ListUsersInput) ([]sqlc.User, map[uuid.UUID][]uuid.UUID, error) {
	params := sqlc.ListUsersParams{RowLimit: int32(in.Limit)}
	if in.Q != "" {
		params.Q = &in.Q
	}
	if in.Role != "" {
		params.Role = &in.Role
	}
	if in.Cursor != nil {
		params.CursorCreated, params.CursorID = &in.Cursor.CreatedAt, &in.Cursor.ID
	}
	users, err := s.q.ListUsers(ctx, params)
	if err != nil {
		return nil, nil, err
	}
	userIDs := make([]uuid.UUID, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	rows, err := s.q.ListPostIDsForUsers(ctx, userIDs)
	if err != nil {
		return nil, nil, err
	}
	posts := map[uuid.UUID][]uuid.UUID{}
	for _, r := range rows {
		posts[r.UserID] = append(posts[r.UserID], r.PostID)
	}
	return users, posts, nil
}

func (s *Service) Purge(ctx context.Context) (users int64, sessions int64, err error) {
	now := s.clock.Now()
	if users, err = s.q.DeleteUnverifiedUsersBefore(ctx, now.Add(-7*24*time.Hour)); err != nil {
		return 0, 0, fmt.Errorf("auth: purge unverified: %w", err)
	}
	if sessions, err = s.q.DeleteExpiredSessions(ctx, now.Add(-30*24*time.Hour)); err != nil {
		return users, 0, fmt.Errorf("auth: purge sessions: %w", err)
	}
	return users, sessions, nil
}
