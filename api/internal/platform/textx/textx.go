package textx

import (
	"strings"
	"unicode/utf8"
)

func Len(s string) int { return utf8.RuneCountInString(s) }

func Opt(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func OptString(s string) *string { return Opt(&s) }

func Value(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// turns 0812..., 812... and 62812... into +62812..., "" when the input is not a phone number
func NormalizePhone(s string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
	switch {
	case strings.HasPrefix(digits, "0"):
		digits = "62" + digits[1:]
	case strings.HasPrefix(digits, "8"):
		digits = "62" + digits
	}
	if len(digits) < 9 || len(digits) > 15 {
		return ""
	}
	return "+" + digits
}
