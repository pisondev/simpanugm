package matching

import (
	"strings"
	"testing"
)

var synonyms = map[string]string{"hp": "ponsel handphone", "casan": "charger", "tumbler": "botol minum"}

// matching-engine section 3, with the example of a lost wallet
func TestE5TextFollowsTheDocumentedShape(t *testing.T) {
	t.Parallel()
	got := E5Text(TextInput{
		ItemName: "dompet kulit", CategoryName: "Dompet", ParentName: "Dompet dan Tas Kecil", Colours: []string{"cokelat"},
		Description: "Dompet lipat dua, isi KTM dan kartu ATM.", Private: "Ada gantungan kunci kecil berbentuk kucing.",
	}, synonyms)
	want := "query: dompet kulit. Dompet (Dompet dan Tas Kecil). Warna: cokelat. Merek: tidak ada. Dompet lipat dua, isi KTM dan kartu ATM. Ada gantungan kunci kecil berbentuk kucing."
	if got != want {
		t.Fatalf("e5 text:\n%s\nwant\n%s", got, want)
	}
}

func TestTextIsNormalisedMaskedAndExpanded(t *testing.T) {
	t.Parallel()
	got := E5Text(TextInput{
		ItemName: "HP  hitam", CategoryName: "Ponsel", ParentName: "Elektronik",
		Description: "casan ikut, lihat https://contoh.id/x, nomor kartu 4111 1111 1111 1111",
	}, synonyms)
	if strings.Contains(got, "https") || strings.Contains(got, "4111") || strings.Contains(got, "  ") {
		t.Fatalf("not normalised: %s", got)
	}
	// ponsel is already in the text through the category name, so only the missing canonical words are appended
	if !strings.HasSuffix(got, "1111 handphone charger") || !strings.Contains(got, "HP hitam") {
		t.Fatalf("expansion appends and keeps the user's words: %s", got)
	}
	// a word expanded once is not repeated, an existing canonical word is not appended again
	if again := expand("hp ponsel", synonyms); again != "hp ponsel handphone" {
		t.Fatalf("expand: %q", again)
	}
}

func TestClipTextStaysShortAndLeavesPrivateDetailsOut(t *testing.T) {
	t.Parallel()
	got := ClipText(TextInput{
		ItemName: "Tumbler biru", CategoryName: "Tumbler", Colours: []string{"biru"}, Brand: "Corkcicle",
		Description: strings.Repeat("panjang ", 200), Private: "goresan rahasia",
	}, synonyms)
	if len([]rune(got)) > clipTextMaxRunes || strings.Contains(got, "rahasia") || !strings.HasPrefix(got, "Tumbler biru. Tumbler. biru. Corkcicle.") {
		t.Fatalf("clip text: %q", got)
	}
}
