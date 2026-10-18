package matching

import (
	"crypto/sha256"
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/pisondev/simpanugm/api/internal/privacy"
)

var (
	urlPattern  = regexp.MustCompile(`(?i)\b(?:https?://|www\.)\S+`)
	wordPattern = regexp.MustCompile(`[\p{L}\p{N}]+`)
)

const clipTextMaxRunes = 400

// the report fields a matching text is built from, section 3
type TextInput struct {
	ItemName     string
	CategoryName string
	ParentName   string
	Colours      []string
	Brand        string
	Description  string
	// private details of a lost report or hidden details of a found report, used inside the system only
	Private     string
	VisibleText string
}

// NFKC, URLs removed, numbers masked, whitespace collapsed
func normalise(s string) string {
	s = norm.NFKC.String(s)
	s = urlPattern.ReplaceAllString(s, " ")
	s = privacy.MaskNumbers(s)
	return strings.Join(strings.Fields(s), " ")
}

// appends canonical terms for slang found in the text, the user's own words always stay
func expand(text string, synonyms map[string]string) string {
	if len(synonyms) == 0 {
		return text
	}
	seen := map[string]bool{}
	for _, w := range wordPattern.FindAllString(strings.ToLower(text), -1) {
		seen[w] = true
	}
	var extra []string
	for _, w := range wordPattern.FindAllString(strings.ToLower(text), -1) {
		exp, ok := synonyms[w]
		if !ok {
			continue
		}
		for _, e := range strings.Fields(exp) {
			if !seen[e] {
				seen[e] = true
				extra = append(extra, e)
			}
		}
	}
	if len(extra) == 0 {
		return text
	}
	return text + " " + strings.Join(extra, " ")
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "tidak ada"
	}
	return s
}

// the e5 text: everything that describes the item, prefixed as the encoder expects
func E5Text(in TextInput, synonyms map[string]string) string {
	colours := strings.Join(in.Colours, ", ")
	parts := []string{
		in.ItemName + ".",
		in.CategoryName + " (" + in.ParentName + ").",
		"Warna: " + orNone(colours) + ".",
		"Merek: " + orNone(in.Brand) + ".",
		in.Description,
	}
	if in.Private != "" {
		parts = append(parts, in.Private)
	}
	if in.VisibleText != "" {
		parts = append(parts, in.VisibleText)
	}
	return "query: " + expand(normalise(strings.Join(parts, " ")), synonyms)
}

// the CLIP text is short because that encoder was trained on caption length text
func ClipText(in TextInput, synonyms map[string]string) string {
	parts := []string{in.ItemName + ".", in.CategoryName + "."}
	if len(in.Colours) > 0 {
		parts = append(parts, strings.Join(in.Colours, ", ")+".")
	}
	if in.Brand != "" {
		parts = append(parts, in.Brand+".")
	}
	parts = append(parts, in.Description)
	text := expand(normalise(strings.Join(parts, " ")), synonyms)
	if r := []rune(text); len(r) > clipTextMaxRunes {
		text = string(r[:clipTextMaxRunes])
	}
	return text
}

func textHash(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}
