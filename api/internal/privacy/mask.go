package privacy

import (
	"regexp"
	"strings"
)

// digit runs may be grouped by single spaces or hyphens, as printed on cards
var digitRun = regexp.MustCompile(`\d(?:[ -]?\d){12,18}`)

// BR-PRV-03
func MaskNumbers(s string) string {
	return digitRun.ReplaceAllStringFunc(s, func(run string) string {
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, run)
		if len(digits) == 16 || luhn(digits) {
			return strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
		}
		return run
	})
}

func luhn(digits string) bool {
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	sum := 0
	double := false
	for i := len(digits) - 1; i >= 0; i-- {
		d := int(digits[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}
