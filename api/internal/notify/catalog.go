package notify

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// one entry per event of docs/requirements/notifications.md that the app delivers
type Spec struct {
	App      bool
	Email    bool
	Optional bool
	// officers receive it in the 07:00 digest instead of a direct email (BR-NTF-02)
	Digest bool
	// non urgent email waits for the end of the quiet hours (BR-NTF-03)
	Quiet bool
	Title string
	Body  string
}

const (
	MatchHighForOwner     Event = "match.high_for_owner"
	MatchMediumForOwner   Event = "match.medium_digest_for_owner"
	MatchHintForFinder    Event = "match.hint_for_finder"
	MatchHighForOfficer   Event = "match.high_for_officer"
	MatchOfficerInvites   Event = "match.officer_invites_claim"
	FoundReceived         Event = "found.received"
	FoundReturnedThanks   Event = "found.returned_thanks"
	CustodyRetentionAlert Event = "custody.retention_warning"
	CustodyUnclaimed      Event = "custody.unclaimed"
	ClaimSubmitted        Event = "claim.submitted"
)

var catalog = map[Event]Spec{
	MatchHighForOwner: {
		App: true, Email: true, Optional: true, Quiet: true,
		Title: "Kandidat kuat untuk laporanmu",
		Body:  "Ada barang temuan yang sangat mirip dengan laporanmu {{.lost_code}}. Lihat kandidatnya.",
	},
	MatchMediumForOwner: {
		App: true, Optional: true,
		Title: "Kandidat baru",
		Body:  "{{.count}} kandidat baru untuk {{.item_name}}.",
	},
	MatchHintForFinder: {
		App: true, Optional: true,
		Title: "Pemiliknya mungkin sedang mencari",
		Body:  "Laporan temuanmu {{.found_code}} mungkin sudah dicari pemiliknya. Pastikan barang diserahkan ke pos.",
	},
	MatchHighForOfficer: {
		App: true, Optional: true,
		Title: "Kandidat kuat",
		Body:  "Kandidat kuat: {{.lost_code}} dan {{.found_code}}, skor {{.score}}.",
	},
	MatchOfficerInvites: {
		App: true, Email: true, Quiet: true,
		Title: "Ajukan klaim untuk barangmu",
		Body:  "Petugas menilai salah satu kandidat sangat mungkin milikmu. Ajukan klaim untuk melanjutkan.",
	},
	FoundReceived: {
		App: true, Email: true, Optional: true, Quiet: true,
		Title: "Barang sudah diterima pos",
		Body:  "Terima kasih, {{.item_name}} sudah diterima di {{.post_name}}.",
	},
	FoundReturnedThanks: {
		App: true, Email: true, Optional: true, Quiet: true,
		Title: "Barang sudah kembali",
		Body:  "Barang yang kamu temukan sudah kembali ke pemiliknya. Terima kasih sudah membantu.",
	},
	CustodyRetentionAlert: {
		App: true, Digest: true,
		Title: "Masa simpan segera habis",
		Body:  "{{.count}} barang akan melewati masa simpan dalam 7 hari.",
	},
	CustodyUnclaimed: {
		App: true, Digest: true,
		Title: "Barang tidak diklaim",
		Body:  "{{.found_code}} tidak diklaim hingga batas waktu. Tentukan tindak lanjut.",
	},
	ClaimSubmitted: {
		App: true, Digest: true,
		Title: "Klaim baru",
		Body:  "Klaim baru {{.claim_code}} untuk {{.found_code}}.",
	},
}

// the groups a user can switch off in Profil, Notifikasi (FR-NOTIF-04)
var optionalGroups = []string{"match", "found", "claim"}

func OptionalGroups() []string { return optionalGroups }

func Lookup(e Event) (Spec, bool) {
	s, ok := catalog[e]
	return s, ok
}

func (e Event) Group() string {
	g, _, _ := strings.Cut(string(e), ".")
	return g
}

func DigestEvents() []string {
	var out []string
	for e, s := range catalog {
		if s.Digest {
			out = append(out, string(e))
		}
	}
	return out
}

// rendered once at creation and stored, so a later template change never rewrites history
func render(text string, data map[string]string) (string, error) {
	t, err := template.New("n").Option("missingkey=error").Parse(text)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("notify: render: %w", err)
	}
	return b.String(), nil
}
