package claims

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/custody"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/media"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/textx"
)

// one result of the checklist as the officer recorded it
type CheckResult struct {
	ItemKey string
	Result  string
	Note    *string
}

type action func(tx pgx.Tx, q *sqlc.Queries, c sqlc.Claim, v sqlc.ClaimViewRow) error

// runs one change of a claim in a transaction with the row locked, then returns the claim as the caller sees it
func (s *Service) mutate(ctx context.Context, p *access.Principal, idOrCode string, fn action) (openapi.Claim, error) {
	id, code, err := ref(idOrCode)
	if err != nil {
		return openapi.Claim{}, err
	}
	var claim uuid.UUID
	err = db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		c, err := q.ClaimForUpdate(ctx, sqlc.ClaimForUpdateParams{ID: id, Code: code})
		if db.IsNoRows(err) {
			return apperr.NotFound()
		}
		if err != nil {
			return err
		}
		v, err := q.ClaimView(ctx, sqlc.ClaimViewParams{ID: &c.ID})
		if err != nil {
			return err
		}
		if !p.IsStaff() && (v.ClaimantID == nil || *v.ClaimantID != p.UserID) {
			return apperr.NotFound()
		}
		claim = c.ID
		return fn(tx, q, c, v)
	})
	if err != nil {
		return openapi.Claim{}, err
	}
	return s.view(ctx, p, claim.String())
}

func (s *Service) audit(ctx context.Context, q *sqlc.Queries, action string, c sqlc.Claim, changes map[string]any) error {
	return audit.Record(ctx, q, s.clock.Now(), audit.Entry{Action: action, EntityType: "claim", EntityID: c.ID, Changes: changes})
}

func (s *Service) requirePost(p *access.Principal, v sqlc.ClaimViewRow) error {
	if !s.assignedTo(p, v) {
		return custody.NotAssigned()
	}
	return nil
}

func stateError(status string) error { return apperr.InvalidTransition(status) }

func (s *Service) message(ctx context.Context, q *sqlc.Queries, claim uuid.UUID, author *uuid.UUID, role, body string) error {
	return q.InsertClaimMessage(ctx, sqlc.InsertClaimMessageParams{
		ID: ids.New(), ClaimID: claim, AuthorID: author, AuthorRole: role, Body: body, CreatedAt: s.clock.Now(),
	})
}

func roleOf(p *access.Principal) string { return string(p.Role) }

// K2
func (s *Service) PickUp(ctx context.Context, p *access.Principal, ref string) (openapi.Claim, error) {
	return s.mutate(ctx, p, ref, func(_ pgx.Tx, q *sqlc.Queries, c sqlc.Claim, _ sqlc.ClaimViewRow) error {
		if c.Status != "submitted" {
			return stateError(c.Status)
		}
		if err := s.step(ctx, q, c.ID, c.Status, "pick_up", "under_review", nil); err != nil {
			return err
		}
		if err := q.PickUpClaim(ctx, sqlc.PickUpClaimParams{ID: c.ID, ReviewerID: &p.UserID, UpdatedAt: s.clock.Now()}); err != nil {
			return err
		}
		return s.audit(ctx, q, "claim.pick_up", c, nil)
	})
}

// K3
func (s *Service) RequestInfo(ctx context.Context, p *access.Principal, ref, message string) (openapi.Claim, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return openapi.Claim{}, apperr.Validation(apperr.FieldError{Field: "message", Code: "required", Message: "Tulis pertanyaan untuk pengklaim"})
	}
	return s.mutate(ctx, p, ref, func(tx pgx.Tx, q *sqlc.Queries, c sqlc.Claim, v sqlc.ClaimViewRow) error {
		if c.Status != "under_review" {
			return stateError(c.Status)
		}
		now := s.clock.Now()
		if err := s.step(ctx, q, c.ID, c.Status, "request_info", "info_requested", nil); err != nil {
			return err
		}
		if err := q.SetClaimInfoRequested(ctx, sqlc.SetClaimInfoRequestedParams{ID: c.ID, InfoRequestedAt: &now}); err != nil {
			return err
		}
		if err := s.message(ctx, q, c.ID, &p.UserID, roleOf(p), message); err != nil {
			return err
		}
		if v.ClaimantID != nil {
			if err := s.tell(ctx, tx, notify.ClaimInfoRequested, *v.ClaimantID, memberLink(c.Code), map[string]string{"claim_code": c.Code}, c.ID); err != nil {
				return err
			}
		}
		return s.audit(ctx, q, "claim.request_info", c, nil)
	})
}

// K4. The claimant answers and may add evidence, which refreshes the checklist
func (s *Service) Respond(ctx context.Context, p *access.Principal, ref, message string, evidence []uuid.UUID) (openapi.Claim, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return openapi.Claim{}, apperr.Validation(apperr.FieldError{Field: "message", Code: "required", Message: "Tulis balasanmu"})
	}
	return s.mutate(ctx, p, ref, func(tx pgx.Tx, q *sqlc.Queries, c sqlc.Claim, v sqlc.ClaimViewRow) error {
		if p.IsStaff() || c.Status != "info_requested" {
			return stateError(c.Status)
		}
		current, err := s.media.Evidence(ctx, q, c.ID)
		if err != nil {
			return err
		}
		if len(evidence) > 0 {
			all := make([]uuid.UUID, 0, len(current)+len(evidence))
			for _, m := range current {
				all = append(all, m.ID)
			}
			all = append(all, evidence...)
			if len(all) > maxEvidence {
				return apperr.Validation(apperr.FieldError{Field: "evidence_media_ids", Code: "too_many", Message: "Maksimal 3 gambar bukti"})
			}
			if _, err := s.media.Attach(ctx, q, p, media.Target{Purpose: "claim_evidence", ClaimID: &c.ID}, all); err != nil {
				return err
			}
		}
		if err := s.step(ctx, q, c.ID, c.Status, "respond", "under_review", nil); err != nil {
			return err
		}
		if err := q.SetClaimUnderReview(ctx, sqlc.SetClaimUnderReviewParams{ID: c.ID, UpdatedAt: s.clock.Now()}); err != nil {
			return err
		}
		if err := s.message(ctx, q, c.ID, &p.UserID, roleOf(p), message); err != nil {
			return err
		}
		if err := s.buildChecks(ctx, q, v, len(current)+len(evidence), s.clock.Now()); err != nil {
			return err
		}
		return s.notifyReviewer(ctx, tx, q, c, v, notify.ClaimInfoProvided)
	})
}

// the reviewer, or the officers of the post when nobody picked the claim up
func (s *Service) notifyReviewer(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, c sqlc.Claim, v sqlc.ClaimViewRow, event notify.Event) error {
	to := []uuid.UUID{}
	if c.ReviewerID != nil {
		to = append(to, *c.ReviewerID)
	} else {
		officers, err := s.officersOf(ctx, q, v.HoldingPostID, v.IntendedPostID)
		if err != nil {
			return err
		}
		to = officers
	}
	for _, o := range to {
		if err := s.tell(ctx, tx, event, o, officerLink(c.Code), map[string]string{"claim_code": c.Code}, c.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Checks(ctx context.Context, p *access.Principal, ref string) (openapi.ClaimCheckList, error) {
	v, err := s.load(ctx, s.q, p, ref)
	if err != nil {
		return openapi.ClaimCheckList{}, err
	}
	return s.checkList(ctx, s.q, v.ID)
}

func (s *Service) checkList(ctx context.Context, q *sqlc.Queries, claim uuid.UUID) (openapi.ClaimCheckList, error) {
	rows, err := q.ListClaimChecks(ctx, claim)
	if err != nil {
		return openapi.ClaimCheckList{}, err
	}
	out := openapi.ClaimCheckList{Items: make([]openapi.ClaimCheck, 0, len(rows)), Complete: true}
	for _, r := range rows {
		var sug suggestion
		_ = json.Unmarshal(r.Suggestion, &sug)
		item := openapi.ClaimCheck{
			ItemKey: r.ItemKey, Label: r.Label, Required: r.Required, InfoOnly: r.InfoOnly, Result: openapi.ClaimCheckResult(r.Result), Note: r.Note,
			Suggestion: openapi.ClaimCheckSuggestion{Level: strPtr(sug.Level), Text: strPtr(sug.Text)},
		}
		if len(sug.Links) > 0 {
			links := sug.Links
			item.Suggestion.Links = &links
		}
		out.Items = append(out.Items, item)
		if r.Required && !resolved(r.Result, r.Note) {
			out.Complete = false
		}
	}
	return out, nil
}

// BR-CLM-10: pass, or not applicable with a note
func resolved(result string, note *string) bool {
	return result == "pass" || (result == "not_applicable" && note != nil && strings.TrimSpace(*note) != "")
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *Service) PutChecks(ctx context.Context, p *access.Principal, ref string, items []CheckResult) (openapi.ClaimCheckList, error) {
	var out openapi.ClaimCheckList
	_, err := s.mutate(ctx, p, ref, func(_ pgx.Tx, q *sqlc.Queries, c sqlc.Claim, _ sqlc.ClaimViewRow) error {
		if !slices.Contains(Open, c.Status) {
			return stateError(c.Status)
		}
		now := s.clock.Now()
		for _, it := range items {
			note := textx.Opt(it.Note)
			if it.Result == "not_applicable" && note == nil {
				return apperr.Validation(apperr.FieldError{Field: "items." + it.ItemKey, Code: "note_required", Message: "Tidak berlaku butuh catatan"})
			}
			n, err := q.SetClaimCheck(ctx, sqlc.SetClaimCheckParams{ClaimID: c.ID, ItemKey: it.ItemKey, Result: it.Result, Note: note, DecidedBy: &p.UserID, DecidedAt: &now})
			if err != nil {
				return err
			}
			if n == 0 {
				return apperr.Validation(apperr.FieldError{Field: "items." + it.ItemKey, Code: "unknown", Message: "Butir tidak dikenal"})
			}
		}
		list, err := s.checkList(ctx, q, c.ID)
		out = list
		return err
	})
	return out, err
}

func newNonce() ([]byte, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	return b, err
}

func parseVersion(raw string) (int32, error) {
	raw = strings.Trim(strings.TrimSpace(raw), `"`)
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 1 {
		return 0, apperr.BadRequest("Versi tidak valid.")
	}
	return int32(n), nil
}

// K6. Needs the item at a post, a different officer for a walk in claim, and a resolved checklist
func (s *Service) Approve(ctx context.Context, p *access.Principal, ref, ifMatch string, note *string) (openapi.Claim, error) {
	version, err := parseVersion(ifMatch)
	if err != nil {
		return openapi.Claim{}, err
	}
	return s.mutate(ctx, p, ref, func(tx pgx.Tx, q *sqlc.Queries, c sqlc.Claim, v sqlc.ClaimViewRow) error {
		if c.Status != "under_review" {
			return stateError(c.Status)
		}
		if c.Version != version {
			return apperr.StaleVersion()
		}
		if v.CustodyStatus != "in_custody" {
			return unprocessable("custody_not_received", "Barang belum diterima di pos", "Barang belum diterima di pos")
		}
		if err := s.requirePost(p, v); err != nil {
			return err
		}
		if c.Source == "proxy" && c.CreatedBy == p.UserID {
			return unprocessable("same_officer", "Butuh petugas lain", "Klaim atas nama pengunjung harus disetujui petugas yang berbeda dari pembuatnya.")
		}
		checks, err := s.checkList(ctx, q, c.ID)
		if err != nil {
			return err
		}
		if !checks.Complete {
			return unprocessable("checklist_incomplete", "Daftar periksa belum lengkap", "Selesaikan semua butir wajib sebelum menyetujui.")
		}
		nonce, err := newNonce()
		if err != nil {
			return err
		}
		now := s.clock.Now()
		if err := s.step(ctx, q, c.ID, c.Status, "approve", "approved", nil); err != nil {
			return err
		}
		if err := q.ApproveClaim(ctx, sqlc.ApproveClaimParams{
			ID: c.ID, DecidedBy: &p.UserID, DecidedAt: &now, HandoverNonce: nonce, ApprovedExpiresAt: ptrTime(now.Add(s.approvedFor(ctx))), DecisionNote: textx.Opt(note),
		}); err != nil {
			if db.IsUniqueViolation(err, "claims_one_approved") {
				return unprocessable("already_approved", "Barang sudah disetujui", "Klaim lain atas barang ini sudah disetujui.")
			}
			return err
		}
		// K7: every other open claim on the item waits
		others, err := q.OpenClaimsOfFound(ctx, c.FoundReportID)
		if err != nil {
			return err
		}
		for _, o := range others {
			if o.ID == c.ID || o.Status == "on_hold" {
				continue
			}
			if err := s.step(ctx, q, o.ID, o.Status, "other_approved", "on_hold", nil); err != nil {
				return err
			}
			if err := q.SetClaimOnHold(ctx, sqlc.SetClaimOnHoldParams{ID: o.ID, UpdatedAt: now}); err != nil {
				return err
			}
			if o.ClaimantID != nil {
				if err := s.tell(ctx, tx, notify.ClaimOnHold, *o.ClaimantID, memberLink(o.Code), map[string]string{"claim_code": o.Code}, o.ID); err != nil {
					return err
				}
			}
		}
		if err := s.resync(ctx, q, c.LostReportID, c.FoundReportID); err != nil {
			return err
		}
		if err := s.pair(ctx, q, c.LostReportID, c.FoundReportID, []string{"active"}, "confirmed"); err != nil {
			return err
		}
		if v.ClaimantID != nil {
			post := ""
			if v.HoldingPostName != nil {
				post = *v.HoldingPostName
			}
			if err := s.tell(ctx, tx, notify.ClaimApproved, *v.ClaimantID, memberLink(c.Code), map[string]string{"claim_code": c.Code, "post_name": post}, c.ID); err != nil {
				return err
			}
		}
		return s.audit(ctx, q, "claim.approve", c, map[string]any{"checks": checks.Complete})
	})
}

// K10
func (s *Service) Reject(ctx context.Context, p *access.Principal, ref, reason string, note *string) (openapi.Claim, error) {
	if _, ok := reasonLabels[reason]; !ok {
		return openapi.Claim{}, apperr.Validation(apperr.FieldError{Field: "reason", Code: "invalid", Message: "Pilih alasan penolakan"})
	}
	note = textx.Opt(note)
	if reason == "other" && (note == nil || textx.Len(*note) < 5) {
		return openapi.Claim{}, apperr.Validation(apperr.FieldError{Field: "note", Code: "required", Message: "Alasan lain butuh catatan minimal 5 karakter"})
	}
	return s.mutate(ctx, p, ref, func(tx pgx.Tx, q *sqlc.Queries, c sqlc.Claim, v sqlc.ClaimViewRow) error {
		if c.Status != "under_review" && c.Status != "info_requested" {
			return stateError(c.Status)
		}
		now := s.clock.Now()
		if err := s.step(ctx, q, c.ID, c.Status, "reject", "rejected", &reason); err != nil {
			return err
		}
		if err := q.RejectClaim(ctx, sqlc.RejectClaimParams{ID: c.ID, DecisionReason: &reason, DecisionNote: note, DecidedBy: &p.UserID, DecidedAt: &now}); err != nil {
			return err
		}
		if negativeReasons[reason] {
			// M7 and the label for the evaluation set
			pair, err := q.PairScores(ctx, sqlc.PairScoresParams{LostReportID: c.LostReportID, FoundReportID: c.FoundReportID})
			if err == nil {
				if err := s.pair(ctx, q, c.LostReportID, c.FoundReportID, []string{"active", "superseded"}, "rejected"); err != nil {
					return err
				}
				if err := q.InsertMatchFeedback(ctx, sqlc.InsertMatchFeedbackParams{
					ID: ids.New(), MatchID: pair.ID, ActorID: &p.UserID, ActorRole: roleOf(p), Verdict: "not_relevant", Reason: note, CreatedAt: now,
				}); err != nil {
					return err
				}
			} else if !db.IsNoRows(err) {
				return err
			}
			if err := s.flagRepeatedRejections(ctx, q, c, now); err != nil {
				return err
			}
		}
		if err := s.resync(ctx, q, c.LostReportID, c.FoundReportID); err != nil {
			return err
		}
		if v.ClaimantID != nil {
			if err := s.tell(ctx, tx, notify.ClaimRejected, *v.ClaimantID, memberLink(c.Code),
				map[string]string{"claim_code": c.Code, "reason_label": reasonLabels[reason]}, c.ID); err != nil {
				return err
			}
		}
		return s.audit(ctx, q, "claim.reject", c, map[string]any{"reason": reason})
	})
}

// BR-CLM-04: three rejections in 30 days restrict the member until an administrator clears the mark
func (s *Service) flagRepeatedRejections(ctx context.Context, q *sqlc.Queries, c sqlc.Claim, now time.Time) error {
	if c.ClaimantID == nil {
		return nil
	}
	window := time.Duration(s.settings.Int(ctx, "claim.reject_flag_window_days")) * 24 * time.Hour
	n, err := q.RejectedClaimsSince(ctx, sqlc.RejectedClaimsSinceParams{ClaimantID: c.ClaimantID, DecidedAt: ptrTime(now.Add(-window))})
	if err != nil {
		return err
	}
	if int(n) >= s.settings.Int(ctx, "claim.reject_flag_count") {
		return q.SetClaimRestricted(ctx, sqlc.SetClaimRestrictedParams{ID: *c.ClaimantID, ClaimRestrictedAt: ptrTime(now)})
	}
	return nil
}

// the claims on hold return to review when the approved claim ends, K8
func (s *Service) resumeHeld(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, foundID, except uuid.UUID) error {
	others, err := q.OpenClaimsOfFound(ctx, foundID)
	if err != nil {
		return err
	}
	for _, o := range others {
		if o.ID == except || o.Status != "on_hold" {
			continue
		}
		if err := s.step(ctx, q, o.ID, o.Status, "approval_ended", "under_review", nil); err != nil {
			return err
		}
		if err := q.SetClaimUnderReview(ctx, sqlc.SetClaimUnderReviewParams{ID: o.ID, UpdatedAt: s.clock.Now()}); err != nil {
			return err
		}
		if o.ClaimantID != nil {
			if err := s.tell(ctx, tx, notify.ClaimResumed, *o.ClaimantID, memberLink(o.Code), map[string]string{"claim_code": o.Code}, o.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// what ending an approved claim undoes: the match goes back to active and held claims resume
func (s *Service) releaseApproval(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, c sqlc.Claim) error {
	if err := s.pair(ctx, q, c.LostReportID, c.FoundReportID, []string{"confirmed"}, "active"); err != nil {
		return err
	}
	if err := s.resumeHeld(ctx, tx, q, c.FoundReportID, c.ID); err != nil {
		return err
	}
	return s.resync(ctx, q, c.LostReportID, c.FoundReportID)
}

// K13
func (s *Service) Revoke(ctx context.Context, p *access.Principal, ref, reason string) (openapi.Claim, error) {
	return s.mutate(ctx, p, ref, func(tx pgx.Tx, q *sqlc.Queries, c sqlc.Claim, v sqlc.ClaimViewRow) error {
		if c.Status != "approved" {
			return stateError(c.Status)
		}
		if err := s.requirePost(p, v); err != nil {
			return err
		}
		if err := s.step(ctx, q, c.ID, c.Status, "revoke", "under_review", &reason); err != nil {
			return err
		}
		if err := q.ReopenClaim(ctx, sqlc.ReopenClaimParams{ID: c.ID, ReopenedAt: ptrTime(s.clock.Now())}); err != nil {
			return err
		}
		if err := s.message(ctx, q, c.ID, &p.UserID, roleOf(p), "Persetujuan dicabut: "+reason); err != nil {
			return err
		}
		if err := s.releaseApproval(ctx, tx, q, c); err != nil {
			return err
		}
		return s.audit(ctx, q, "claim.revoke", c, map[string]any{"reason": reason})
	})
}

// K15, an exception to BR-CLM-02 that is recorded with its reason
func (s *Service) Reopen(ctx context.Context, p *access.Principal, ref, reason string) (openapi.Claim, error) {
	return s.mutate(ctx, p, ref, func(_ pgx.Tx, q *sqlc.Queries, c sqlc.Claim, _ sqlc.ClaimViewRow) error {
		if c.Status != "rejected" {
			return stateError(c.Status)
		}
		if err := s.step(ctx, q, c.ID, c.Status, "reopen", "under_review", &reason); err != nil {
			return err
		}
		if err := q.ReopenClaim(ctx, sqlc.ReopenClaimParams{ID: c.ID, ReopenedAt: ptrTime(s.clock.Now())}); err != nil {
			return err
		}
		if err := s.resync(ctx, q, c.LostReportID, c.FoundReportID); err != nil {
			return err
		}
		return s.audit(ctx, q, "claim.reopen", c, map[string]any{"reason": reason})
	})
}

// K14
func (s *Service) Withdraw(ctx context.Context, p *access.Principal, ref string) (openapi.Claim, error) {
	return s.mutate(ctx, p, ref, func(tx pgx.Tx, q *sqlc.Queries, c sqlc.Claim, _ sqlc.ClaimViewRow) error {
		if p.IsStaff() && !p.IsAdmin() {
			return apperr.Forbidden()
		}
		if !slices.Contains(Open, c.Status) {
			return stateError(c.Status)
		}
		if err := s.step(ctx, q, c.ID, c.Status, "withdraw", "withdrawn", nil); err != nil {
			return err
		}
		if err := q.EndClaim(ctx, sqlc.EndClaimParams{ID: c.ID, Status: "withdrawn", UpdatedAt: s.clock.Now()}); err != nil {
			return err
		}
		if c.Status == "approved" {
			if err := s.releaseApproval(ctx, tx, q, c); err != nil {
				return err
			}
		} else if err := s.resync(ctx, q, c.LostReportID, c.FoundReportID); err != nil {
			return err
		}
		return s.audit(ctx, q, "claim.withdraw", c, nil)
	})
}

// BR-HND-02: an officer who verified the person at the desk issues a new code, which unlocks a locked one
func (s *Service) RegenerateCode(ctx context.Context, p *access.Principal, ref string) (openapi.Claim, error) {
	return s.mutate(ctx, p, ref, func(tx pgx.Tx, q *sqlc.Queries, c sqlc.Claim, v sqlc.ClaimViewRow) error {
		if c.Status != "approved" {
			return stateError(c.Status)
		}
		if err := s.requirePost(p, v); err != nil {
			return err
		}
		nonce, err := newNonce()
		if err != nil {
			return err
		}
		if err := q.RegenerateHandoverCode(ctx, sqlc.RegenerateHandoverCodeParams{
			ID: c.ID, HandoverNonce: nonce, HandoverCodeIssuedAt: ptrTime(s.clock.Now()), ApprovedExpiresAt: c.ApprovedExpiresAt,
		}); err != nil {
			return err
		}
		if v.ClaimantID != nil {
			if err := s.tell(ctx, tx, notify.ClaimMessage, *v.ClaimantID, memberLink(c.Code), map[string]string{"claim_code": c.Code}, c.ID); err != nil {
				return err
			}
		}
		return s.audit(ctx, q, "claim.handover_code.regenerate", c, nil)
	})
}

func (s *Service) Messages(ctx context.Context, p *access.Principal, ref string) ([]openapi.ClaimMessage, error) {
	v, err := s.load(ctx, s.q, p, ref)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListClaimMessages(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	out := make([]openapi.ClaimMessage, 0, len(rows))
	for _, r := range rows {
		out = append(out, openapi.ClaimMessage{
			Id: r.ID, AuthorRole: openapi.ClaimMessageAuthorRole(r.AuthorRole), AuthorName: r.AuthorName, Body: r.Body,
			Mine: r.AuthorID != nil && *r.AuthorID == p.UserID, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

// FR-CLAIM-09: the thread is between the claimant and officers, the finder never takes part
func (s *Service) PostMessage(ctx context.Context, p *access.Principal, ref, body string) (openapi.ClaimMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return openapi.ClaimMessage{}, apperr.Validation(apperr.FieldError{Field: "message", Code: "required", Message: "Tulis pesanmu"})
	}
	var posted openapi.ClaimMessage
	_, err := s.mutate(ctx, p, ref, func(tx pgx.Tx, q *sqlc.Queries, c sqlc.Claim, v sqlc.ClaimViewRow) error {
		if !slices.Contains(Open, c.Status) {
			return stateError(c.Status)
		}
		id := ids.New()
		now := s.clock.Now()
		if err := q.InsertClaimMessage(ctx, sqlc.InsertClaimMessageParams{ID: id, ClaimID: c.ID, AuthorID: &p.UserID, AuthorRole: roleOf(p), Body: body, CreatedAt: now}); err != nil {
			return err
		}
		posted = openapi.ClaimMessage{Id: id, AuthorRole: openapi.ClaimMessageAuthorRole(roleOf(p)), AuthorName: p.FullName, Body: body, Mine: true, CreatedAt: now}
		if p.IsStaff() {
			if v.ClaimantID != nil {
				return s.tell(ctx, tx, notify.ClaimMessage, *v.ClaimantID, memberLink(c.Code), map[string]string{"claim_code": c.Code}, c.ID)
			}
			return nil
		}
		return s.notifyReviewer(ctx, tx, q, c, v, notify.ClaimMessage)
	})
	return posted, err
}

func (s *Service) SetInternalNote(ctx context.Context, p *access.Principal, ref, note string) (openapi.Claim, error) {
	return s.mutate(ctx, p, ref, func(_ pgx.Tx, q *sqlc.Queries, c sqlc.Claim, _ sqlc.ClaimViewRow) error {
		return q.SetClaimInternalNote(ctx, sqlc.SetClaimInternalNoteParams{ID: c.ID, InternalNote: textx.OptString(note), UpdatedAt: s.clock.Now()})
	})
}
