// Package fuzzy ports rapidfuzz fuzz.token_set_ratio, checked against data/fixtures/fuzzy_cases.json
package fuzzy

import (
	"slices"
	"strings"
	"unicode"
)

// TokenSetRatio matches rapidfuzz 3 without a processor: case sensitive, tokens split on whitespace
func TokenSetRatio(s1, s2 string) float64 {
	a, b := tokenSet(s1), tokenSet(s2)
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	var intersect, diffAB, diffBA []string
	for t := range a {
		if b[t] {
			intersect = append(intersect, t)
		} else {
			diffAB = append(diffAB, t)
		}
	}
	for t := range b {
		if !a[t] {
			diffBA = append(diffBA, t)
		}
	}
	// one sentence is part of the other
	if len(intersect) > 0 && (len(diffAB) == 0 || len(diffBA) == 0) {
		return 100
	}
	slices.Sort(intersect)
	slices.Sort(diffAB)
	slices.Sort(diffBA)
	ab := []rune(strings.Join(diffAB, " "))
	ba := []rune(strings.Join(diffBA, " "))
	sectLen := len([]rune(strings.Join(intersect, " ")))
	sep := 0
	if sectLen > 0 {
		sep = 1
	}
	sectAB := sectLen + sep + len(ab)
	sectBA := sectLen + sep + len(ba)
	result := normalized(indel(ab, ba), sectAB+sectBA)
	if sectLen == 0 {
		return result
	}
	result = max(result, normalized(sep+len(ab), sectLen+sectAB))
	return max(result, normalized(sep+len(ba), sectLen+sectBA))
}

// lowercased words of letters and digits, the form names and visible text are compared in
func Normalize(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }), " ")
}

func tokenSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, t := range strings.Fields(s) {
		out[t] = true
	}
	return out
}

func normalized(dist, lensum int) float64 {
	if lensum == 0 {
		return 100
	}
	return 100 * (1 - float64(dist)/float64(lensum))
}

// the Indel distance is insertions plus deletions, len(a) + len(b) - 2 * LCS
func indel(a, b []rune) int {
	if len(a) == 0 || len(b) == 0 {
		return len(a) + len(b)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				cur[j] = prev[j-1] + 1
			} else {
				cur[j] = max(prev[j], cur[j-1])
			}
		}
		prev, cur = cur, prev
	}
	return len(a) + len(b) - 2*prev[len(b)]
}
