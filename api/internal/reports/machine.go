package reports

import "github.com/pisondev/simpanugm/api/internal/platform/apperr"

// rows mirror docs/requirements/state-machines.md, one row per from and to pair of a numbered transition
type Transition struct {
	ID, From, Event, To string
}

type Machine struct {
	Name string
	Rows []Transition
}

func (m Machine) Step(from, event, to string) (string, error) {
	for _, r := range m.Rows {
		if r.From == from && r.Event == event && r.To == to {
			return r.ID, nil
		}
	}
	return "", apperr.InvalidTransition(from)
}

func (m Machine) Allows(from, event string) bool {
	for _, r := range m.Rows {
		if r.From == from && r.Event == event {
			return true
		}
	}
	return false
}

func expand(id string, froms []string, event string, tos ...string) []Transition {
	var out []Transition
	for _, f := range froms {
		for _, t := range tos {
			out = append(out, Transition{ID: id, From: f, Event: event, To: t})
		}
	}
	return out
}

func rows(groups ...[]Transition) []Transition {
	var out []Transition
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

var lostOpen = []string{"reported", "potential_match", "under_verification", "claimed", "returned"}

var Lost = Machine{Name: "lost_report", Rows: rows(
	expand("L1", []string{""}, "submit", "reported"),
	expand("L2", []string{"reported"}, "candidate_appeared", "potential_match"),
	expand("L3", []string{"potential_match"}, "candidates_gone", "reported"),
	expand("L4", []string{"reported", "potential_match"}, "claim_submitted", "under_verification"),
	expand("L5", []string{"under_verification"}, "claims_ended", "reported", "potential_match"),
	expand("L6", []string{"under_verification"}, "claim_approved", "claimed"),
	expand("L7", []string{"claimed"}, "approval_ended", "under_verification", "potential_match", "reported"),
	expand("L8", []string{"claimed"}, "handover", "returned"),
	expand("L9", []string{"returned"}, "receipt_confirmed", "closed"),
	expand("L9", []string{"returned"}, "confirmation_window_passed", "closed"),
	expand("L10", []string{"reported", "potential_match", "under_verification"}, "owner_close", "closed"),
	expand("L11", []string{"reported", "potential_match"}, "expire", "closed"),
	expand("L12", []string{"closed"}, "reopen", "reported"),
	expand("L13", []string{"reported", "potential_match", "under_verification"}, "mark_duplicate", "closed"),
	expand("L14", lostOpen, "account_deleted", "closed"),
)}

var Found = Machine{Name: "found_report", Rows: rows(
	expand("F1", []string{""}, "submit", "reported"),
	expand("F2", []string{"reported"}, "candidate_appeared", "potential_match"),
	expand("F3", []string{"potential_match"}, "candidates_gone", "reported"),
	expand("F4", []string{"reported", "potential_match"}, "claim_submitted", "under_verification"),
	expand("F5", []string{"under_verification"}, "claims_ended", "reported", "potential_match"),
	expand("F6", []string{"under_verification"}, "claim_approved", "claimed"),
	expand("F7", []string{"claimed"}, "approval_ended", "under_verification", "potential_match", "reported"),
	expand("F8", []string{"claimed"}, "handover", "returned"),
	expand("F9", []string{"returned"}, "receipt_confirmed", "closed"),
	expand("F9", []string{"returned"}, "confirmation_window_passed", "closed"),
	expand("F10", []string{"reported", "potential_match"}, "retention_passed", "unclaimed"),
	expand("F11", []string{"unclaimed"}, "revert", "reported"),
	expand("F12", []string{"unclaimed"}, "dispose", "closed"),
	expand("F13", []string{"reported", "potential_match"}, "finder_close", "closed"),
	expand("F14", []string{"reported", "potential_match"}, "mark_duplicate", "closed"),
)}

var Custody = Machine{Name: "custody", Rows: rows(
	expand("C1", []string{""}, "member_submit", "with_finder"),
	expand("C2", []string{""}, "proxy_submit", "in_custody"),
	expand("C3", []string{"with_finder"}, "receive", "in_custody"),
	expand("C4", []string{"in_custody"}, "transfer", "in_custody"),
	expand("C5", []string{"in_custody"}, "handover", "released"),
	expand("C6", []string{"in_custody"}, "dispose", "disposed"),
	expand("C7", []string{"with_finder"}, "returned_directly", "released"),
	expand("C8", []string{"with_finder"}, "cancel", "cancelled"),
)}
