package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
)

// the first administrator comes from the command line, the link is printed because email may not work yet
func (s *Service) BootstrapAdmin(ctx context.Context, email, name string) (string, error) {
	email = NormalizeEmail(email)
	if !validEmail(email) || len([]rune(name)) < 2 {
		return "", apperr.Validation(apperr.FieldError{Field: "email", Code: "invalid", Message: "Email atau nama tidak valid"})
	}
	now := s.clock.Now()
	var link string
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		u, err := q.CreateUser(ctx, sqlc.CreateUserParams{ID: ids.New(), Email: email, FullName: name, Role: string(access.Admin), Status: "pending_verification"})
		if db.IsUniqueViolation(err, "users_email_key") {
			return apperr.Unprocessable("email_taken", "Email sudah terdaftar", "Email ini sudah memiliki akun.")
		}
		if err != nil {
			return err
		}
		plain, hash := ids.Token()
		if err := q.CreateAuthToken(ctx, sqlc.CreateAuthTokenParams{
			ID: ids.New(), UserID: u.ID, Purpose: "invite", TokenHash: hash, ExpiresAt: now.Add(inviteTTL), CreatedAt: now,
		}); err != nil {
			return err
		}
		link = s.mailer.BaseURL() + "/undangan?token=" + plain
		if err := s.mailer.Enqueue(ctx, tx, notify.Invite, u.Email, notify.Data{Name: u.FullName, Link: link, Expires: "72 jam"}); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "user.bootstrap_admin", EntityType: "user", EntityID: u.ID})
	})
	return link, err
}

type ActiveUser struct {
	Email    string
	Name     string
	Role     access.Role
	Password string
	PostIDs  []uuid.UUID
}

// demo seeds and tests only: a verified account without the email round trip
func (s *Service) CreateActiveUser(ctx context.Context, in ActiveUser) (sqlc.User, error) {
	hash, err := HashPassword(in.Password)
	if err != nil {
		return sqlc.User{}, err
	}
	now := s.clock.Now()
	var u sqlc.User
	err = db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		email := NormalizeEmail(in.Email)
		existing, err := q.GetUserByEmail(ctx, email)
		if err == nil {
			u = existing
			return nil
		}
		if !db.IsNoRows(err) {
			return err
		}
		version := CurrentPrivacyVersion
		if u, err = q.CreateUser(ctx, sqlc.CreateUserParams{
			ID: ids.New(), Email: email, FullName: in.Name, PasswordHash: &hash, Role: string(in.Role), Status: "pending_verification",
			PrivacyVersion: &version, PrivacyAcceptedAt: &now,
		}); err != nil {
			return err
		}
		if err := q.MarkEmailVerified(ctx, sqlc.MarkEmailVerifiedParams{Now: now, ID: u.ID}); err != nil {
			return err
		}
		if err := s.replacePosts(ctx, q, u.ID, in.PostIDs, u.ID); err != nil {
			return err
		}
		u, err = q.GetUserByID(ctx, u.ID)
		return err
	})
	return u, err
}
