//go:build integration

package server_test

import (
	"net/http"
	"testing"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

type meBody struct {
	Email    string  `json:"email"`
	FullName string  `json:"full_name"`
	Phone    *string `json:"phone"`
	NimNip   *string `json:"nim_nip"`
}

// US-05 criterion 1, FR-AUTH-10
func TestProfileEditsAndClears(t *testing.T) {
	e := testenv.New(t)
	u := e.User(access.Member)
	res := u.Client.JSON(http.MethodPatch, "/api/v1/me", map[string]any{"full_name": "  Rani Prasetya ", "phone": "0812 3456 7890", "nim_nip": "24/543770/PA/23107"})
	var me meBody
	res.Decode(t, &me)
	if res.Status != http.StatusOK || me.FullName != "Rani Prasetya" || me.Phone == nil || *me.Phone != "+6281234567890" || me.NimNip == nil {
		t.Fatalf("update: %d %s", res.Status, res.Body)
	}
	res = u.Client.JSON(http.MethodPatch, "/api/v1/me", map[string]any{"phone": ""})
	me = meBody{}
	res.Decode(t, &me)
	if me.Phone != nil || me.FullName != "Rani Prasetya" || me.NimNip == nil {
		t.Fatalf("an empty phone clears only the phone: %s", res.Body)
	}
	if p := u.Client.JSON(http.MethodPatch, "/api/v1/me", map[string]any{"nim_nip": "24<script>"}).Problem(t); len(p.Errors) == 0 || p.Errors[0].Field != "nim_nip" {
		t.Fatalf("invalid NIM: %+v", p)
	}
	if res := u.Client.JSON(http.MethodPatch, "/api/v1/me", map[string]any{"email": "lain@mail.ugm.ac.id"}); res.Status != http.StatusBadRequest && res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("the email changes only through confirmation: %d", res.Status)
	}
}

// US-05 criterion 1: the new address confirms, the old one works until then
func TestEmailChangeNeedsConfirmationFromTheNewAddress(t *testing.T) {
	e := testenv.New(t)
	u := e.User(access.Member)
	next := testenv.Unique("baru") + "@mail.ugm.ac.id"

	if p := u.Client.JSON(http.MethodPost, "/api/v1/me/email", map[string]any{"email": next, "password": "salah sekali"}).Problem(t); len(p.Errors) == 0 || p.Errors[0].Field != "current_password" {
		t.Fatalf("wrong password: %+v", p)
	}
	if p := u.Client.JSON(http.MethodPost, "/api/v1/me/email", map[string]any{"email": "luar@gmail.com", "password": testenv.Password}).Problem(t); len(p.Errors) == 0 || p.Errors[0].Code != "email_domain_not_allowed" {
		t.Fatalf("a member keeps a campus address (BR-ACC-01): %+v", p)
	}
	if res := u.Client.JSON(http.MethodPost, "/api/v1/me/email", map[string]any{"email": next, "password": testenv.Password}); res.Status != http.StatusAccepted {
		t.Fatalf("start: %d %s", res.Status, res.Body)
	}
	token := testenv.TokenFrom(t, e.LastEmail(next))
	var me meBody
	u.Client.JSON(http.MethodGet, "/api/v1/me", nil).Decode(t, &me)
	if me.Email != u.Email {
		t.Fatal("the old address stays until the new one is confirmed")
	}
	if res := e.Client().JSON(http.MethodPost, "/api/v1/auth/email/confirm", map[string]string{"token": token}); res.Status != http.StatusOK {
		t.Fatalf("confirm: %d %s", res.Status, res.Body)
	}
	u.Client.JSON(http.MethodGet, "/api/v1/me", nil).Decode(t, &me)
	if me.Email != next {
		t.Fatalf("email after confirmation: %s", me.Email)
	}
	if res := e.Client().JSON(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": next, "password": testenv.Password}); res.Status != http.StatusOK {
		t.Fatalf("sign in with the new address: %d", res.Status)
	}
	if p := e.Client().JSON(http.MethodPost, "/api/v1/auth/email/confirm", map[string]string{"token": token}).Problem(t); p.Code != "token_invalid" {
		t.Fatalf("a used link: %+v", p)
	}
}

// the current session survives a password change, every other one ends
func TestPasswordChangeSignsOutOtherSessions(t *testing.T) {
	e := testenv.New(t)
	u := e.User(access.Member)
	other := e.Client()
	if res := other.JSON(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": u.Email, "password": testenv.Password}); res.Status != http.StatusOK {
		t.Fatal("second session")
	}
	if p := u.Client.JSON(http.MethodPut, "/api/v1/me/password", map[string]string{"current_password": "keliru sekali", "new_password": "kata sandi baru yang panjang"}).Problem(t); len(p.Errors) == 0 || p.Errors[0].Field != "current_password" {
		t.Fatalf("wrong current password: %+v", p)
	}
	if res := u.Client.JSON(http.MethodPut, "/api/v1/me/password", map[string]string{"current_password": testenv.Password, "new_password": "kata sandi baru yang panjang"}); res.Status != http.StatusNoContent {
		t.Fatalf("change: %d %s", res.Status, res.Body)
	}
	if res := u.Client.JSON(http.MethodGet, "/api/v1/me", nil); res.Status != http.StatusOK {
		t.Fatal("the session that changed the password stays")
	}
	if res := other.JSON(http.MethodGet, "/api/v1/me", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("another session after a password change: %d", res.Status)
	}
	if e.EmailCount(u.Email, "account.password_changed") != 1 {
		t.Fatal("account.password_changed is sent")
	}
}

// US-05 criterion 2, FR-AUTH-11
func TestSessionsListAndRevoke(t *testing.T) {
	e := testenv.New(t)
	u := e.User(access.Member)
	stranger := e.User(access.Member)
	phone := e.Client()
	phone.Head.Set("User-Agent", "Mozilla/5.0 (Linux; Android 14) Chrome/141.0 Mobile")
	if res := phone.JSON(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": u.Email, "password": testenv.Password}); res.Status != http.StatusOK {
		t.Fatal("phone session")
	}
	var list struct {
		Items []struct {
			ID      string `json:"id"`
			Device  string `json:"device"`
			IPHint  string `json:"ip_hint"`
			Current bool   `json:"current"`
		} `json:"items"`
	}
	u.Client.JSON(http.MethodGet, "/api/v1/me/sessions", nil).Decode(t, &list)
	if len(list.Items) != 2 {
		t.Fatalf("two sessions expected: %+v", list.Items)
	}
	var current, phoneID string
	for _, s := range list.Items {
		if s.Current {
			current = s.ID
		} else {
			phoneID = s.ID
			if s.Device != "Chrome di Android" {
				t.Fatalf("device label %q", s.Device)
			}
		}
	}
	if p := u.Client.JSON(http.MethodDelete, "/api/v1/me/sessions/"+current, nil).Problem(t); p.Code != "current_session" {
		t.Fatalf("revoking the current session from the list: %+v", p)
	}
	if res := stranger.Client.JSON(http.MethodDelete, "/api/v1/me/sessions/"+phoneID, nil); res.Status != http.StatusNotFound {
		t.Fatalf("someone else's session: %d", res.Status)
	}
	if res := u.Client.JSON(http.MethodDelete, "/api/v1/me/sessions/"+phoneID, nil); res.Status != http.StatusNoContent {
		t.Fatalf("revoke: %d %s", res.Status, res.Body)
	}
	if res := phone.JSON(http.MethodGet, "/api/v1/me", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("the revoked phone session: %d", res.Status)
	}
}
