// Package claims is the claim lifecycle of state-machines section 5, the verification checklist of AI-08 and the
// handover of FR-CUST-03 and FR-CUST-04.
package claims

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"fmt"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/reports"
)

// the states in which a claim still holds its item (BR-CLM-01)
var Open = []string{"submitted", "under_review", "info_requested", "approved", "on_hold"}

// rows mirror state-machines section 5, one numbered transition each
var Claim = reports.Machine{Name: "claim", Rows: rows(
	expand("K1", []string{""}, "submit", "submitted"),
	expand("K2", []string{"submitted"}, "pick_up", "under_review"),
	expand("K3", []string{"under_review"}, "request_info", "info_requested"),
	expand("K4", []string{"info_requested"}, "respond", "under_review"),
	expand("K5", []string{"info_requested"}, "info_timeout", "expired"),
	expand("K6", []string{"under_review"}, "approve", "approved"),
	expand("K7", []string{"submitted", "under_review", "info_requested"}, "other_approved", "on_hold"),
	expand("K8", []string{"on_hold"}, "approval_ended", "under_review"),
	expand("K9", []string{"on_hold"}, "item_returned", "rejected"),
	expand("K10", []string{"under_review", "info_requested"}, "reject", "rejected"),
	expand("K11", []string{"approved"}, "handover", "handed_over"),
	expand("K12", []string{"approved"}, "approval_timeout", "expired"),
	expand("K13", []string{"approved"}, "revoke", "under_review"),
	expand("K14", Open, "withdraw", "withdrawn"),
	expand("K15", []string{"rejected"}, "reopen", "under_review"),
)}

func expand(id string, froms []string, event string, tos ...string) []reports.Transition {
	var out []reports.Transition
	for _, f := range froms {
		for _, t := range tos {
			out = append(out, reports.Transition{ID: id, From: f, Event: event, To: t})
		}
	}
	return out
}

func rows(groups ...[]reports.Transition) []reports.Transition {
	var out []reports.Transition
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// BR-HND-01: the six digit code is HMAC-SHA256(pepper, claim id || nonce) reduced to six digits. Only the random nonce is
// stored, so a database copy alone reveals no code, while the app can still show the code to the claimant
func handoverCode(pepper []byte, claim uuid.UUID, nonce []byte) string {
	mac := hmac.New(sha256.New, pepper)
	mac.Write(claim[:])
	mac.Write(nonce)
	return fmt.Sprintf("%06d", binary.BigEndian.Uint64(mac.Sum(nil)[:8])%1_000_000)
}

func codeMatches(pepper []byte, claim uuid.UUID, nonce []byte, given string) bool {
	return subtle.ConstantTimeCompare([]byte(handoverCode(pepper, claim, nonce)), []byte(given)) == 1
}

// the reasons an officer may reject with (BR-CLM-08) and what the claimant is told
var reasonLabels = map[string]string{
	"details_mismatch":      "detail yang disebutkan tidak sesuai",
	"insufficient_evidence": "bukti belum cukup",
	"item_already_returned": "barang sudah dikembalikan kepada orang lain",
	"duplicate_claim":       "klaim ganda",
	"suspected_fraud":       "klaim tidak dapat diverifikasi",
	"other":                 "alasan lain, lihat catatan petugas",
}

// reasons that label the candidate negative and count toward BR-CLM-04
var negativeReasons = map[string]bool{"details_mismatch": true, "insufficient_evidence": true, "suspected_fraud": true}
