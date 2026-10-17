//go:build integration

package server_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

func registerBody(email string) map[string]any {
	return map[string]any{
		"full_name": "Rani Ayu", "email": email, "password": testenv.Password,
		"privacy_version": "2026.10", "accept_privacy": true,
	}
}

type me struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	Role          string   `json:"role"`
	EmailVerified bool     `json:"email_verified"`
	PostIDs       []string `json:"post_ids"`
	Permissions   []string `json:"permissions"`
}

func TestRegisterVerifyLoginLogout(t *testing.T) {
	e := testenv.New(t)
	c := e.Client()
	email := testenv.Unique("rani") + "@mail.ugm.ac.id"

	if res := c.JSON(http.MethodPost, "/api/v1/auth/register", registerBody(email)); res.Status != http.StatusAccepted {
		t.Fatalf("register: %d %s", res.Status, res.Body)
	}
	res := c.JSON(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": strings.ToUpper(email), "password": testenv.Password})
	if res.Status != http.StatusOK {
		t.Fatalf("login before verification must work (BR-ACC-02): %d %s", res.Status, res.Body)
	}
	var m me
	res.Decode(t, &m)
	if m.EmailVerified || m.Role != "member" {
		t.Fatalf("unexpected me: %+v", m)
	}
	res = c.JSON(http.MethodPost, "/api/v1/lost-reports", map[string]any{}, "Idempotency-Key", "k-unverified-1")
	if p := res.Problem(t); res.Status != http.StatusForbidden || p.Code != "account_unverified" {
		t.Fatalf("unverified create: %d %s", res.Status, res.Body)
	}

	token := testenv.TokenFrom(t, e.LastEmail(email))
	if res := c.JSON(http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": token}); res.Status != http.StatusOK {
		t.Fatalf("verify: %d %s", res.Status, res.Body)
	}
	if res := c.JSON(http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": token}); res.Problem(t).Code != "token_invalid" {
		t.Fatalf("a used token must be refused: %s", res.Body)
	}
	c.JSON(http.MethodGet, "/api/v1/me", nil).Decode(t, &m)
	if !m.EmailVerified {
		t.Fatal("email should be verified")
	}
	if res := c.JSON(http.MethodPost, "/api/v1/auth/logout", nil); res.Status != http.StatusNoContent {
		t.Fatalf("logout: %d %s", res.Status, res.Body)
	}
	if res := c.JSON(http.MethodGet, "/api/v1/me", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("me after logout: %d", res.Status)
	}
}

func TestRegisterRefusesOutsideDomainAndWeakPasswords(t *testing.T) {
	e := testenv.New(t)
	c := e.Client()
	body := registerBody(testenv.Unique("x") + "@gmail.com")
	body["password"] = "qwertyuiop"
	res := c.JSON(http.MethodPost, "/api/v1/auth/register", body)
	p := res.Problem(t)
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("status %d %s", res.Status, res.Body)
	}
	got := map[string]string{}
	for _, f := range p.Errors {
		got[f.Field] = f.Code
	}
	if got["email"] != "email_domain_not_allowed" || got["password"] != "too_common" {
		t.Fatalf("field errors %v", got)
	}
}

func TestRegisterWithExistingEmailLooksIdentical(t *testing.T) {
	e := testenv.New(t)
	u := e.User(access.Member)
	res := e.Client().JSON(http.MethodPost, "/api/v1/auth/register", registerBody(u.Email))
	if res.Status != http.StatusAccepted {
		t.Fatalf("status %d", res.Status)
	}
	if n := e.EmailCount(u.Email, "account.register_attempt"); n != 1 {
		t.Fatalf("register attempt emails = %d, want 1", n)
	}
}

func TestLoginFailuresAreUniformAndRateLimited(t *testing.T) {
	e := testenv.New(t)
	u := e.User(access.Member)
	c := e.Client()
	unknown := c.JSON(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "tidak-ada@mail.ugm.ac.id", "password": "salah-salah-salah"})
	wrong := c.JSON(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": u.Email, "password": "salah-salah-salah"})
	if unknown.Status != http.StatusUnauthorized || wrong.Status != http.StatusUnauthorized ||
		unknown.Problem(t).Code != wrong.Problem(t).Code {
		t.Fatalf("unknown %d %s, wrong %d %s", unknown.Status, unknown.Body, wrong.Status, wrong.Body)
	}
	var last testenv.Response
	for i := 0; i < 5; i++ {
		last = c.JSON(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": u.Email, "password": "salah-salah-salah"})
	}
	if last.Status != http.StatusTooManyRequests || last.Header.Get("Retry-After") == "" {
		t.Fatalf("expected 429 with Retry-After after repeated failures, got %d", last.Status)
	}
}

func TestPasswordResetRevokesEverySession(t *testing.T) {
	e := testenv.New(t)
	u := e.User(access.Member)
	other := e.Client()
	if res := other.JSON(http.MethodPost, "/api/v1/auth/password/forgot", map[string]string{"email": u.Email}); res.Status != http.StatusAccepted {
		t.Fatalf("forgot: %d", res.Status)
	}
	token := testenv.TokenFrom(t, e.LastEmail(u.Email))
	res := other.JSON(http.MethodPost, "/api/v1/auth/password/reset", map[string]string{"token": token, "password": "kata-sandi-baru-yang-panjang"})
	if res.Status != http.StatusOK {
		t.Fatalf("reset: %d %s", res.Status, res.Body)
	}
	if res := u.Client.JSON(http.MethodGet, "/api/v1/me", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("old session must be revoked, got %d", res.Status)
	}
	if n := e.EmailCount(u.Email, "account.password_changed"); n != 1 {
		t.Fatalf("password changed emails = %d", n)
	}
}

func TestInvitationAndPostAssignment(t *testing.T) {
	e := testenv.New(t)
	admin := e.User(access.Admin)
	email := testenv.Unique("petugas") + "@gmail.com"
	library, saintek := e.Post("pos-perpustakaan"), e.Post("pos-saintek")
	res := admin.Client.JSON(http.MethodPost, "/api/v1/admin/users/invitations", map[string]any{
		"email": email, "full_name": "Pak Slamet", "role": "officer", "post_ids": []string{library.String()},
	})
	if res.Status != http.StatusCreated {
		t.Fatalf("invite: %d %s", res.Status, res.Body)
	}
	var invited struct {
		ID string `json:"id"`
	}
	res.Decode(t, &invited)
	c := e.Client()
	res = c.JSON(http.MethodPost, "/api/v1/auth/invitations/accept", map[string]string{"token": testenv.TokenFrom(t, e.LastEmail(email)), "password": testenv.Password})
	if res.Status != http.StatusOK {
		t.Fatalf("accept: %d %s", res.Status, res.Body)
	}
	var m me
	res.Decode(t, &m)
	if m.Role != "officer" || len(m.PostIDs) != 1 || m.PostIDs[0] != library.String() {
		t.Fatalf("officer after accept: %+v", m)
	}
	res = admin.Client.JSON(http.MethodPatch, "/api/v1/admin/users/"+invited.ID, map[string]any{"post_ids": []string{library.String(), saintek.String()}})
	if res.Status != http.StatusOK {
		t.Fatalf("update posts: %d %s", res.Status, res.Body)
	}
	c.JSON(http.MethodGet, "/api/v1/me", nil).Decode(t, &m)
	if len(m.PostIDs) != 2 {
		t.Fatalf("posts after update: %v", m.PostIDs)
	}
	if res := c.JSON(http.MethodGet, "/api/v1/admin/users", nil); res.Status != http.StatusForbidden {
		t.Fatalf("officer listing users: %d", res.Status)
	}
}

func TestLastActiveAdminCannotBeDemoted(t *testing.T) {
	e := testenv.New(t)
	admin := e.User(access.Admin)
	ctx := context.Background()
	if _, err := e.Pool.Exec(ctx, "UPDATE users SET status = 'suspended' WHERE role = 'admin' AND status = 'active' AND id <> $1", admin.ID); err != nil {
		t.Fatal(err)
	}
	defer e.Pool.Exec(ctx, "UPDATE users SET status = 'active' WHERE role = 'admin' AND status = 'suspended'")
	res := admin.Client.JSON(http.MethodPatch, "/api/v1/admin/users/"+admin.ID.String(), map[string]any{"role": "member"})
	if p := res.Problem(t); p.Code != "last_admin" {
		t.Fatalf("expected last_admin, got %d %s", res.Status, res.Body)
	}
}

func TestStaffSessionIdleTimeout(t *testing.T) {
	e := testenv.New(t)
	officer := e.User(access.Officer, e.Post("pos-pusat"))
	e.Clock.Advance(61 * time.Minute)
	if res := officer.Client.JSON(http.MethodGet, "/api/v1/me", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("officer idle 61 minutes: %d", res.Status)
	}
}

func TestMemberSessionSlides(t *testing.T) {
	e := testenv.New(t)
	member := e.User(access.Member)
	e.Clock.Advance(10 * 24 * time.Hour)
	res := member.Client.JSON(http.MethodGet, "/api/v1/me", nil)
	if res.Status != http.StatusOK || !strings.Contains(res.Header.Get("Set-Cookie"), "simpan_session=") {
		t.Fatalf("expected a refreshed cookie, got %d %q", res.Status, res.Header.Get("Set-Cookie"))
	}
	e.Clock.Advance(25 * 24 * time.Hour)
	if res := member.Client.JSON(http.MethodGet, "/api/v1/me", nil); res.Status != http.StatusOK {
		t.Fatalf("35 days after login with activity at day 10 the session lives: %d", res.Status)
	}
}

func TestCrossSiteRequestsAreRefused(t *testing.T) {
	e := testenv.New(t)
	member := e.User(access.Member)
	res := member.Client.JSON(http.MethodPost, "/api/v1/auth/logout", nil, "Origin", "https://evil.example", "Sec-Fetch-Site", "cross-site")
	if p := res.Problem(t); res.Status != http.StatusForbidden || p.Code != "csrf_origin_mismatch" {
		t.Fatalf("cross site: %d %s", res.Status, res.Body)
	}
}

func TestContractRejectsUnknownFieldsAndBadJSON(t *testing.T) {
	e := testenv.New(t)
	c := e.Client()
	res := c.JSON(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "a@ugm.ac.id", "password": "x", "admin": "true"})
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("unknown field: %d %s", res.Status, res.Body)
	}
	res = c.Do(http.MethodPost, "/api/v1/auth/login", strings.NewReader("{not json"), "application/json")
	if res.Status != http.StatusBadRequest && res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("bad json: %d %s", res.Status, res.Body)
	}
	if res := c.JSON(http.MethodGet, "/api/v1/tidak-ada", nil); res.Status != http.StatusNotFound {
		t.Fatalf("unknown path: %d", res.Status)
	}
}
