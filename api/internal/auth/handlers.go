package auth

import (
	"context"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/cursor"
	"github.com/pisondev/simpanugm/api/internal/platform/httpx"
	"github.com/pisondev/simpanugm/api/internal/platform/reqctx"
)

type Handlers struct {
	svc     *Service
	cookies Cookies
}

func NewHandlers(svc *Service, cookies Cookies) *Handlers {
	return &Handlers{svc: svc, cookies: cookies}
}

func ack(msg string) openapi.Ack { return openapi.Ack{Message: msg} }

func (h *Handlers) Register(ctx context.Context, req openapi.RegisterRequestObject) (openapi.RegisterResponseObject, error) {
	b := req.Body
	in := RegisterInput{
		FullName: b.FullName, Email: b.Email, Password: b.Password,
		PrivacyVersion: b.PrivacyVersion, AcceptPrivacy: b.AcceptPrivacy, IP: reqctx.From(ctx).IP,
	}
	if b.NimNip != nil {
		in.NimNip = *b.NimNip
	}
	if b.Phone != nil {
		in.Phone = *b.Phone
	}
	if err := h.svc.Register(ctx, in); err != nil {
		return nil, err
	}
	return openapi.Register202JSONResponse(ack("Cek emailmu untuk tautan verifikasi.")), nil
}

func (h *Handlers) VerifyEmail(ctx context.Context, req openapi.VerifyEmailRequestObject) (openapi.VerifyEmailResponseObject, error) {
	if err := h.svc.VerifyEmail(ctx, req.Body.Token); err != nil {
		return nil, err
	}
	return openapi.VerifyEmail200JSONResponse(ack("Email berhasil diverifikasi.")), nil
}

func (h *Handlers) ResendVerification(ctx context.Context, _ openapi.ResendVerificationRequestObject) (openapi.ResendVerificationResponseObject, error) {
	if err := h.svc.ResendVerification(ctx, access.FromContext(ctx)); err != nil {
		return nil, err
	}
	return openapi.ResendVerification202JSONResponse(ack("Tautan verifikasi dikirim ulang.")), nil
}

func (h *Handlers) Login(ctx context.Context, req openapi.LoginRequestObject) (openapi.LoginResponseObject, error) {
	m := reqctx.From(ctx)
	sess, u, err := h.svc.Login(ctx, LoginInput{Email: req.Body.Email, Password: req.Body.Password, IP: m.IP, UserAgent: m.UserAgent})
	if err != nil {
		return nil, err
	}
	httpx.SetCookieCtx(ctx, h.cookies.Session(sess, h.svc.clock.Now()))
	me, err := h.me(ctx, u)
	if err != nil {
		return nil, err
	}
	return openapi.Login200JSONResponse(me), nil
}

func (h *Handlers) Logout(ctx context.Context, req openapi.LogoutRequestObject) (openapi.LogoutResponseObject, error) {
	all := req.Params.All != nil && *req.Params.All
	if err := h.svc.Logout(ctx, access.FromContext(ctx), all); err != nil {
		return nil, err
	}
	httpx.SetCookieCtx(ctx, h.cookies.Clear())
	return openapi.Logout204Response{}, nil
}

func (h *Handlers) ForgotPassword(ctx context.Context, req openapi.ForgotPasswordRequestObject) (openapi.ForgotPasswordResponseObject, error) {
	if err := h.svc.ForgotPassword(ctx, req.Body.Email); err != nil {
		return nil, err
	}
	return openapi.ForgotPassword202JSONResponse(ack("Jika email terdaftar, tautan sudah dikirim.")), nil
}

func (h *Handlers) ResetPassword(ctx context.Context, req openapi.ResetPasswordRequestObject) (openapi.ResetPasswordResponseObject, error) {
	if err := h.svc.ResetPassword(ctx, req.Body.Token, req.Body.Password); err != nil {
		return nil, err
	}
	httpx.SetCookieCtx(ctx, h.cookies.Clear())
	return openapi.ResetPassword200JSONResponse(ack("Kata sandi berhasil diubah. Silakan masuk kembali.")), nil
}

func (h *Handlers) AcceptInvitation(ctx context.Context, req openapi.AcceptInvitationRequestObject) (openapi.AcceptInvitationResponseObject, error) {
	m := reqctx.From(ctx)
	sess, u, err := h.svc.AcceptInvitation(ctx, req.Body.Token, req.Body.Password, m.IP, m.UserAgent)
	if err != nil {
		return nil, err
	}
	httpx.SetCookieCtx(ctx, h.cookies.Session(sess, h.svc.clock.Now()))
	me, err := h.me(ctx, u)
	if err != nil {
		return nil, err
	}
	return openapi.AcceptInvitation200JSONResponse(me), nil
}

func (h *Handlers) GetMe(ctx context.Context, _ openapi.GetMeRequestObject) (openapi.GetMeResponseObject, error) {
	u, err := h.svc.User(ctx, access.FromContext(ctx).UserID)
	if err != nil {
		return nil, err
	}
	me, err := h.me(ctx, u)
	if err != nil {
		return nil, err
	}
	return openapi.GetMe200JSONResponse(me), nil
}

func (h *Handlers) me(ctx context.Context, u sqlc.User) (openapi.Me, error) {
	posts := []uuid.UUID{}
	if access.Role(u.Role) != access.Member {
		var err error
		if posts, err = h.svc.PostIDs(ctx, u.ID); err != nil {
			return openapi.Me{}, err
		}
	}
	return openapi.Me{
		Id: u.ID, Email: u.Email, FullName: u.FullName, NimNip: u.NimNip, Phone: u.Phone,
		Role: openapi.Role(u.Role), Status: openapi.AccountStatus(u.Status),
		EmailVerified: u.EmailVerifiedAt != nil, ClaimRestricted: u.ClaimRestrictedAt != nil,
		PostIds: posts, Permissions: access.Role(u.Role).Permissions(),
	}, nil
}

func adminUser(u sqlc.User, posts []uuid.UUID) openapi.AdminUser {
	if posts == nil {
		posts = []uuid.UUID{}
	}
	return openapi.AdminUser{
		Id: u.ID, Email: u.Email, FullName: u.FullName, Role: openapi.Role(u.Role), Status: openapi.AccountStatus(u.Status),
		EmailVerified: u.EmailVerifiedAt != nil, PostIds: posts, CreatedAt: u.CreatedAt, LastLoginAt: u.LastLoginAt,
	}
}

func (h *Handlers) AdminListUsers(ctx context.Context, req openapi.AdminListUsersRequestObject) (openapi.AdminListUsersResponseObject, error) {
	c, err := cursor.Decode(req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	limit := cursor.Limit(req.Params.Limit)
	in := ListUsersInput{Limit: limit + 1}
	if req.Params.Q != nil {
		in.Q = *req.Params.Q
	}
	if req.Params.Role != nil {
		in.Role = string(*req.Params.Role)
	}
	if c != nil {
		in.Cursor = &Cursor{CreatedAt: c.At, ID: c.ID}
	}
	users, posts, err := h.svc.ListUsers(ctx, in)
	if err != nil {
		return nil, err
	}
	out := openapi.AdminUserList{Items: []openapi.AdminUser{}}
	for i, u := range users {
		if i == limit {
			last := users[limit-1]
			out.NextCursor = cursor.Encode(last.CreatedAt, last.ID)
			break
		}
		out.Items = append(out.Items, adminUser(u, posts[u.ID]))
	}
	return openapi.AdminListUsers200JSONResponse(out), nil
}

func (h *Handlers) AdminInviteUser(ctx context.Context, req openapi.AdminInviteUserRequestObject) (openapi.AdminInviteUserResponseObject, error) {
	b := req.Body
	u, err := h.svc.Invite(ctx, InviteInput{Email: b.Email, FullName: b.FullName, Role: access.Role(b.Role), PostIDs: b.PostIds})
	if err != nil {
		return nil, err
	}
	posts, err := h.svc.PostIDs(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return openapi.AdminInviteUser201JSONResponse(adminUser(u, posts)), nil
}

func (h *Handlers) AdminUpdateUser(ctx context.Context, req openapi.AdminUpdateUserRequestObject) (openapi.AdminUpdateUserResponseObject, error) {
	b := req.Body
	in := UpdateUserInput{PostIDs: b.PostIds}
	if b.Role != nil {
		r := access.Role(*b.Role)
		in.Role = &r
	}
	if in.Role == nil && in.PostIDs == nil {
		return nil, apperr.BadRequest("Tidak ada perubahan.")
	}
	u, err := h.svc.UpdateUser(ctx, req.Id, in)
	if err != nil {
		return nil, err
	}
	posts, err := h.svc.PostIDs(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return openapi.AdminUpdateUser200JSONResponse(adminUser(u, posts)), nil
}

func (h *Handlers) UpdateMe(ctx context.Context, req openapi.UpdateMeRequestObject) (openapi.UpdateMeResponseObject, error) {
	u, err := h.svc.UpdateProfile(ctx, access.FromContext(ctx), ProfileInput{FullName: req.Body.FullName, Phone: req.Body.Phone, NimNip: req.Body.NimNip})
	if err != nil {
		return nil, err
	}
	me, err := h.me(ctx, u)
	if err != nil {
		return nil, err
	}
	return openapi.UpdateMe200JSONResponse(me), nil
}

func (h *Handlers) StartEmailChange(ctx context.Context, req openapi.StartEmailChangeRequestObject) (openapi.StartEmailChangeResponseObject, error) {
	if err := h.svc.StartEmailChange(ctx, access.FromContext(ctx), req.Body.Email, req.Body.Password); err != nil {
		return nil, err
	}
	return openapi.StartEmailChange202JSONResponse(ack("Jika alamat itu bisa dipakai, tautan konfirmasi sudah dikirim ke sana.")), nil
}

func (h *Handlers) ConfirmEmailChange(ctx context.Context, req openapi.ConfirmEmailChangeRequestObject) (openapi.ConfirmEmailChangeResponseObject, error) {
	if err := h.svc.ConfirmEmailChange(ctx, req.Body.Token); err != nil {
		return nil, err
	}
	return openapi.ConfirmEmailChange200JSONResponse(ack("Email akunmu sudah diganti.")), nil
}

func (h *Handlers) ChangePassword(ctx context.Context, req openapi.ChangePasswordRequestObject) (openapi.ChangePasswordResponseObject, error) {
	if err := h.svc.ChangePassword(ctx, access.FromContext(ctx), req.Body.CurrentPassword, req.Body.NewPassword); err != nil {
		return nil, err
	}
	return openapi.ChangePassword204Response{}, nil
}

func (h *Handlers) ListMySessions(ctx context.Context, _ openapi.ListMySessionsRequestObject) (openapi.ListMySessionsResponseObject, error) {
	rows, err := h.svc.Sessions(ctx, access.FromContext(ctx))
	if err != nil {
		return nil, err
	}
	out := openapi.SessionList{Items: make([]openapi.SessionInfo, 0, len(rows))}
	for _, s := range rows {
		out.Items = append(out.Items, openapi.SessionInfo{Id: s.ID, CreatedAt: s.CreatedAt, LastSeenAt: s.LastSeenAt, Device: s.Device, IpHint: s.IPHint, Current: s.Current})
	}
	return openapi.ListMySessions200JSONResponse(out), nil
}

func (h *Handlers) RevokeMySession(ctx context.Context, req openapi.RevokeMySessionRequestObject) (openapi.RevokeMySessionResponseObject, error) {
	if err := h.svc.RevokeSession(ctx, access.FromContext(ctx), req.Id); err != nil {
		return nil, err
	}
	return openapi.RevokeMySession204Response{}, nil
}
