package claims

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
)

// BR-HND-02
const maxCodeAttempts = 5

var sixDigits = regexp.MustCompile(`^[0-9]{6}$`)

type HandoverInput struct {
	ClaimID       uuid.UUID
	Code          string
	IDType        string
	IDLast4       string
	NameConfirmed bool
}

func locked() *apperr.Error {
	return apperr.Unprocessable("handover_locked", "Kode terkunci", "Terlalu banyak kode salah. Verifikasi identitas pengklaim lalu buat kode baru.")
}

// C5, K11, F8 and L8 in one transaction (FR-CUST-04). A wrong code is counted and committed before the error is returned
func (s *Service) Handover(ctx context.Context, p *access.Principal, foundRef string, in HandoverInput) (openapi.Claim, error) {
	var v apperr.Collector
	v.Check(in.NameConfirmed, "name_confirmed", "required", "Konfirmasi bahwa nama pada identitas sesuai dengan nama pengklaim")
	v.Check(len([]rune(strings.TrimSpace(in.IDLast4))) == 4, "recipient_id_last4", "length", "Isi 4 karakter terakhir identitas")
	if err := v.Err(); err != nil {
		return openapi.Claim{}, err
	}
	var wrong error
	claim, err := s.mutate(ctx, p, in.ClaimID.String(), func(tx pgx.Tx, q *sqlc.Queries, c sqlc.Claim, view sqlc.ClaimViewRow) error {
		if view.FoundCode != foundRef && view.FoundReportID.String() != foundRef {
			return apperr.NotFound()
		}
		if c.Status != "approved" {
			return stateError(c.Status)
		}
		if view.CustodyStatus != "in_custody" {
			return unprocessable("custody_not_received", "Barang belum diterima di pos", "Barang belum diterima di pos")
		}
		if err := s.requirePost(p, view); err != nil {
			return err
		}
		now := s.clock.Now()
		walkIn := c.ClaimantID == nil
		if !walkIn {
			if c.HandoverLockedAt != nil {
				return locked()
			}
			if !sixDigits.MatchString(in.Code) {
				return apperr.Validation(apperr.FieldError{Field: "code", Code: "format", Message: "Kode serah terima 6 angka"})
			}
			if !codeMatches(s.pepper, c.ID, c.HandoverNonce, in.Code) {
				failed, err := q.ClaimCodeFailed(ctx, sqlc.ClaimCodeFailedParams{ID: c.ID, MaxAttempts: maxCodeAttempts, Now: now})
				if err != nil {
					return err
				}
				if err := s.audit(ctx, q, "claim.handover.failed", c, map[string]any{"attempt": failed.HandoverFailedAttempts}); err != nil {
					return err
				}
				if failed.HandoverLockedAt != nil {
					wrong = locked()
				} else {
					wrong = apperr.Unprocessable("handover_code_wrong", "Kode salah", "Kode serah terima tidak cocok.")
				}
				return nil
			}
		}
		if err := s.step(ctx, q, c.ID, c.Status, "handover", "handed_over", nil); err != nil {
			return err
		}
		if err := q.HandOverClaim(ctx, sqlc.HandOverClaimParams{
			ID: c.ID, HandedOverAt: &now, HandedOverBy: &p.UserID, RecipientIDType: &in.IDType, RecipientIDLast4: ptr(strings.TrimSpace(in.IDLast4)),
			RecipientNameConfirmed: &in.NameConfirmed,
		}); err != nil {
			return err
		}
		return s.release(ctx, tx, q, p, c, view, in.IDType)
	})
	if err != nil {
		return openapi.Claim{}, err
	}
	if wrong != nil {
		return openapi.Claim{}, wrong
	}
	return claim, nil
}

func ptr[T any](v T) *T { return &v }

// the effects of FR-CUST-04 after the claim itself is marked
func (s *Service) release(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, p *access.Principal, c sqlc.Claim, v sqlc.ClaimViewRow, idType string) error {
	t := s.clock.Now()
	// F8 and C5, L8
	foundFrom := v.FoundStatus
	if err := q.ReleaseFoundReport(ctx, sqlc.ReleaseFoundReportParams{ID: c.FoundReportID, UpdatedAt: t}); err != nil {
		return err
	}
	if err := s.history(ctx, q, "found_report", c.FoundReportID, foundFrom, "returned", "F8", nil); err != nil {
		return err
	}
	if err := s.history(ctx, q, "custody", c.FoundReportID, "in_custody", "released", "C5", nil); err != nil {
		return err
	}
	if err := s.custody.Event(ctx, q, c.FoundReportID, "released", v.HoldingPostID, nil, p.UserID, nil); err != nil {
		return err
	}
	lostFrom := v.LostStatus
	if err := q.ReturnLostReport(ctx, sqlc.ReturnLostReportParams{ID: c.LostReportID, LastActivityAt: t}); err != nil {
		return err
	}
	if err := s.history(ctx, q, "lost_report", c.LostReportID, lostFrom, "returned", "L8", nil); err != nil {
		return err
	}
	// K9: the other claims on the item end
	others, err := q.OpenClaimsOfFound(ctx, c.FoundReportID)
	if err != nil {
		return err
	}
	reason := "item_already_returned"
	for _, o := range others {
		if o.ID == c.ID {
			continue
		}
		if err := s.step(ctx, q, o.ID, o.Status, "item_returned", "rejected", &reason); err != nil {
			return err
		}
		if err := q.RejectClaim(ctx, sqlc.RejectClaimParams{ID: o.ID, DecisionReason: &reason, DecidedAt: &t}); err != nil {
			return err
		}
		if err := s.resync(ctx, q, o.LostReportID, o.FoundReportID); err != nil {
			return err
		}
		if o.ClaimantID != nil {
			if err := s.tell(ctx, tx, notify.ClaimRejected, *o.ClaimantID, memberLink(o.Code),
				map[string]string{"claim_code": o.Code, "reason_label": reasonLabels[reason]}, o.ID); err != nil {
				return err
			}
		}
	}
	if v.ClaimantID != nil {
		if err := s.tell(ctx, tx, notify.ClaimHandedOver, *v.ClaimantID, "/laporan/hilang/"+v.LostCode, map[string]string{"claim_code": c.Code}, c.ID); err != nil {
			return err
		}
		if err := s.tell(ctx, tx, notify.LostConfirmReceipt, *v.ClaimantID, "/laporan/hilang/"+v.LostCode, map[string]string{"lost_code": v.LostCode}, c.ID); err != nil {
			return err
		}
	}
	if v.FinderID != nil {
		if err := s.tell(ctx, tx, notify.FoundReturnedThanks, *v.FinderID, "/laporan/temuan/"+v.FoundCode, map[string]string{}, c.ID); err != nil {
			return err
		}
	}
	officers, err := s.officersOf(ctx, q, v.HoldingPostID, nil)
	if err != nil {
		return err
	}
	post := ""
	if v.HoldingPostName != nil {
		post = *v.HoldingPostName
	}
	for _, o := range officers {
		if o == p.UserID {
			continue
		}
		if err := s.tell(ctx, tx, notify.CustodyReleased, o, officerLink(c.Code), map[string]string{"found_code": v.FoundCode, "post_name": post}, c.ID); err != nil {
			return err
		}
	}
	return s.audit(ctx, q, "claim.handover", c, map[string]any{"found": v.FoundCode, "id_type": idType})
}
