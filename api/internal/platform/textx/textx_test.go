package textx

import "testing"

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"0812 3456 7890":    "+6281234567890",
		"+62 812-3456-7890": "+6281234567890",
		"81234567890":       "+6281234567890",
		"62274512345":       "+62274512345",
		"12345":             "",
		"":                  "",
	}
	for in, want := range cases {
		if got := NormalizePhone(in); got != want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOpt(t *testing.T) {
	blank := "   "
	if Opt(&blank) != nil || Opt(nil) != nil {
		t.Fatal("blank is absent")
	}
	if v := OptString("  dompet "); v == nil || *v != "dompet" {
		t.Fatal("trim")
	}
}
