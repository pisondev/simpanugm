package assist

import "testing"

// AI-04 and US-24 criteria 1 and 2
func TestDraftFollowsTheTemplate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		item     string
		colours  []string
		brand    string
		text     string
		identity bool
		want     string
	}{
		{"colour only", "Payung", []string{"biru"}, "", "", false, "Payung berwarna biru."},
		{"two colours and brand", "Botol minum", []string{"biru", "perak"}, "Corkcicle", "", false, "Botol minum berwarna biru dan perak, merek Corkcicle."},
		{"no attributes", "Payung", nil, "", "", false, "Payung."},
		{"short text", "Tumbler", []string{"hijau"}, "", "RANI", false, `Tumbler berwarna hijau. Terdapat tulisan "RANI".`},
		{"long text is left out", "Tumbler", []string{"hijau"}, "", "tulisan yang terlalu panjang untuk dimasukkan ke deskripsi", false, "Tumbler berwarna hijau."},
		{"masked number is left out", "Kartu", []string{"putih"}, "", "4111 1111 1111 1111", false, "Kartu berwarna putih."},
		{"identity never shows text", "KTM", nil, "", "RANI", true, "KTM."},
	}
	for _, c := range cases {
		if got := Draft(c.item, c.colours, c.brand, c.text, c.identity); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
