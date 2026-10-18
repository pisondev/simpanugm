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
	ClaimSubmittedAck     Event = "claim.submitted_ack"
	ClaimInfoRequested    Event = "claim.info_requested"
	ClaimInfoProvided     Event = "claim.info_provided"
	ClaimMessage          Event = "claim.message"
	ClaimApproved         Event = "claim.approved"
	ClaimOnHold           Event = "claim.on_hold"
	ClaimResumed          Event = "claim.resumed"
	ClaimRejected         Event = "claim.rejected"
	ClaimExpiring         Event = "claim.expiring"
	ClaimExpired          Event = "claim.expired"
	ClaimHandedOver       Event = "claim.handed_over"
	CustodyReleased       Event = "custody.released"
	FoundReminder24h      Event = "found.custody_reminder_24h"
	FoundReminder48h      Event = "found.custody_reminder_48h"
	FoundOverdue          Event = "found.custody_overdue"
	LostStaleReminder     Event = "lost.stale_reminder"
	LostExpired           Event = "lost.expired"
	LostConfirmReceipt    Event = "lost.confirm_receipt"
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
	ClaimSubmittedAck: {
		App:   true,
		Title: "Klaim terkirim",
		Body:  "Klaim {{.claim_code}} terkirim. Petugas akan memeriksa.",
	},
	ClaimInfoRequested: {
		App: true, Email: true,
		Title: "Perlu info tambahan",
		Body:  "Petugas membutuhkan info tambahan untuk klaim {{.claim_code}}.",
	},
	ClaimInfoProvided: {
		App:   true,
		Title: "Klaim dibalas",
		Body:  "Pengklaim {{.claim_code}} sudah membalas.",
	},
	ClaimMessage: {
		App: true, Optional: true,
		Title: "Pesan baru pada klaim",
		Body:  "Pesan baru pada klaim {{.claim_code}}.",
	},
	ClaimApproved: {
		App: true, Email: true,
		Title: "Klaim disetujui",
		Body:  "Klaim disetujui. Datang ke {{.post_name}} dengan KTM atau KTP. Kode serah terima ada di aplikasi.",
	},
	ClaimOnHold: {
		App:   true,
		Title: "Klaim ditunda",
		Body:  "Klaimmu ditunda karena ada klaim lain yang sedang diproses.",
	},
	ClaimResumed: {
		App:   true,
		Title: "Klaim diperiksa lagi",
		Body:  "Klaimmu kembali diperiksa.",
	},
	ClaimRejected: {
		App: true, Email: true,
		Title: "Klaim ditolak",
		Body:  "Klaim {{.claim_code}} ditolak: {{.reason_label}}.",
	},
	ClaimExpiring: {
		App: true, Email: true,
		Title: "Klaim segera kedaluwarsa",
		Body:  "Klaim disetujui akan kedaluwarsa besok. Segera ambil barang di {{.post_name}}.",
	},
	ClaimExpired: {
		App: true, Email: true,
		Title: "Klaim kedaluwarsa",
		Body:  "Klaim {{.claim_code}} kedaluwarsa.",
	},
	ClaimHandedOver: {
		App: true, Email: true,
		Title: "Serah terima selesai",
		Body:  "Serah terima selesai. Semoga barangmu aman.",
	},
	CustodyReleased: {
		App: true, Digest: true,
		Title: "Barang diserahkan",
		Body:  "{{.found_code}} diserahkan kepada pemiliknya di {{.post_name}}.",
	},
	FoundReminder24h: {
		App: true, Email: true,
		Title: "Serahkan barang temuanmu",
		Body:  "Barang temuan {{.item_name}} belum diserahkan ke pos. Serahkan ke {{.post_name}} agar pemilik bisa mengambilnya.",
	},
	FoundReminder48h: {
		App: true, Email: true,
		Title: "Batas waktu penyimpanan habis",
		Body:  "Batas waktu penyimpanan oleh penemu sudah habis. Mohon serahkan hari ini ke {{.post_name}}.",
	},
	FoundOverdue: {
		App:   true,
		Title: "Barang belum diserahkan penemu",
		Body:  "{{.found_code}} belum diserahkan penemu selama 48 jam.",
	},
	LostStaleReminder: {
		App: true, Email: true, Quiet: true,
		Title: "Masih mencari barangmu?",
		Body:  "Masih mencari {{.item_name}}? Pilih tetap buka atau tutup laporan.",
	},
	LostExpired: {
		App: true, Email: true, Quiet: true,
		Title: "Laporan ditutup",
		Body:  "Laporan {{.lost_code}} ditutup karena tidak ada aktivitas. Kamu bisa membukanya lagi dalam 30 hari.",
	},
	LostConfirmReceipt: {
		App:   true,
		Title: "Konfirmasi barang diterima",
		Body:  "Sudah menerima barangmu? Konfirmasi agar kasus ditutup.",
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
