package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	h, err := HashPassword("kata-sandi-yang-panjang")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected encoding %q", h)
	}
	ok, rehash, err := VerifyPassword("kata-sandi-yang-panjang", h)
	if err != nil || !ok || rehash {
		t.Fatalf("verify: ok=%v rehash=%v err=%v", ok, rehash, err)
	}
	if ok, _, _ := VerifyPassword("kata-sandi-lain", h); ok {
		t.Fatal("wrong password accepted")
	}
	if _, _, err := VerifyPassword("x", "$2a$10$bcrypt"); err == nil {
		t.Fatal("foreign hash formats are errors")
	}
}

func TestOlderParametersAskForRehash(t *testing.T) {
	old := "$argon2id$v=19$m=8192,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$" + strings.Repeat("A", 43)
	if _, rehash, err := VerifyPassword("x", old); err != nil || rehash {
		t.Fatalf("a wrong password never reports rehash: rehash=%v err=%v", rehash, err)
	}
}

func TestPasswordProblem(t *testing.T) {
	cases := []struct{ password, email, code string }{
		{"pendek", "a@ugm.ac.id", "too_short"},
		{strings.Repeat("a", 129), "a@ugm.ac.id", "too_long"},
		{"rani@mail.ugm.ac.id", "Rani@mail.ugm.ac.id", "equals_email"},
		{"qwertyuiop", "a@ugm.ac.id", "too_common"},
		{"1234567890", "a@ugm.ac.id", "too_common"},
		{"payung-biru-di-perpus", "a@ugm.ac.id", ""},
	}
	for _, c := range cases {
		if code, _ := PasswordProblem(c.password, c.email); code != c.code {
			t.Errorf("PasswordProblem(%q) = %q, want %q", c.password, code, c.code)
		}
	}
	if len(commonPasswords) < 9000 {
		t.Fatalf("common password list looks truncated: %d", len(commonPasswords))
	}
}

func TestEmailHelpers(t *testing.T) {
	if NormalizeEmail("  Rani@Mail.UGM.ac.id ") != "rani@mail.ugm.ac.id" {
		t.Fatal("normalize")
	}
	for _, bad := range []string{"", "rani", "rani@", "@ugm.ac.id", "ra ni@ugm.ac.id", "rani@ugm"} {
		if validEmail(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
	if emailDomain("rani@mail.ugm.ac.id") != "mail.ugm.ac.id" {
		t.Fatal("domain")
	}
}
