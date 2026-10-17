package notify

import (
	"strings"
	"testing"
)

func TestEveryEventRendersBothParts(t *testing.T) {
	for event := range subjects {
		a, err := Render(event, "rani@mail.ugm.ac.id", Data{Name: "Rani", Link: "https://simpanugm.my.id/x?token=abc", Expires: "24 jam"}, "https://simpanugm.my.id")
		if err != nil {
			t.Fatalf("%s: %v", event, err)
		}
		if a.Subject == "" || !strings.Contains(a.Text, "Rani") || !strings.Contains(a.HTML, "Rani") {
			t.Fatalf("%s rendered incompletely", event)
		}
		if strings.ContainsAny(a.Text+a.HTML+a.Subject, string([]rune{0x2013, 0x2014})) {
			t.Fatalf("%s contains a typographic dash", event)
		}
	}
}

func TestHTMLEscapesNames(t *testing.T) {
	a, err := Render(VerifyEmail, "x@ugm.ac.id", Data{Name: "<script>alert(1)</script>", Link: "https://simpanugm.my.id"}, "https://simpanugm.my.id")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(a.HTML, "<script>") {
		t.Fatal("name not escaped in html")
	}
}

func TestUnknownEvent(t *testing.T) {
	if _, err := Render("account.nope", "x@ugm.ac.id", Data{}, ""); err == nil {
		t.Fatal("unknown event rendered")
	}
}
