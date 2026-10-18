package claims

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/cursor"
	"github.com/pisondev/simpanugm/api/internal/reports"
)

type Handlers struct{ svc *Service }

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) CreateClaim(ctx context.Context, req openapi.CreateClaimRequestObject) (openapi.CreateClaimResponseObject, error) {
	b := req.Body
	in := CreateInput{
		FoundID: b.FoundReportId, LostID: b.LostReportId, MatchID: b.MatchId, Statement: b.Statement, Evidence: b.EvidenceMediaIds,
		PreferredVisit: b.PreferredVisit,
	}
	for _, a := range b.Answers {
		in.Answers = append(in.Answers, AnswerInput{Key: a.Key, Answer: a.Answer})
	}
	if b.LostReport != nil {
		lost := reports.LostInputFrom(*b.LostReport)
		in.NewLost = &lost
	}
	out, err := h.svc.Create(ctx, access.FromContext(ctx), req.Params.IdempotencyKey, in)
	if err != nil {
		return nil, err
	}
	return openapi.CreateClaim201JSONResponse(out), nil
}

func (h *Handlers) ListClaims(ctx context.Context, req openapi.ListClaimsRequestObject) (openapi.ListClaimsResponseObject, error) {
	p := access.FromContext(ctx)
	cur, err := cursor.Decode(req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	limit := cursor.Limit(req.Params.Limit)
	f := ListFilter{PostID: req.Params.PostId, FoundID: req.Params.FoundReportId, Limit: limit}
	if req.Params.Status != nil {
		f.Statuses = *req.Params.Status
	}
	if cur != nil {
		f.Cursor = &struct {
			At time.Time
			ID uuid.UUID
		}{cur.At, cur.ID}
	}
	rows, err := h.svc.List(ctx, p, f)
	if err != nil {
		return nil, err
	}
	out := openapi.ClaimList{Items: make([]openapi.ClaimSummary, 0, len(rows))}
	for i, r := range rows {
		if i == limit {
			out.NextCursor = cursor.Encode(rows[limit-1].CreatedAt, rows[limit-1].ID)
			break
		}
		item := openapi.ClaimSummary{
			Id: r.ID, Code: r.Code, Status: openapi.ClaimStatus(r.Status), Source: openapi.ClaimSource(r.Source),
			FoundCode: r.FoundCode, FoundItemName: r.FoundItemName, LostCode: r.LostCode, LostItemName: r.LostItemName,
			ClaimantName: r.ClaimantName, CustodyStatus: openapi.CustodyStatus(r.CustodyStatus), HoldingPostName: r.HoldingPostName,
			ApprovedExpiresAt: r.ApprovedExpiresAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
		if r.ThumbMediaID != uuid.Nil {
			id := r.ThumbMediaID
			item.ThumbMediaId = &id
		}
		out.Items = append(out.Items, item)
	}
	return openapi.ListClaims200JSONResponse(out), nil
}

func (h *Handlers) GetClaim(ctx context.Context, req openapi.GetClaimRequestObject) (openapi.GetClaimResponseObject, error) {
	out, err := h.svc.Get(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.GetClaim200JSONResponse(out), nil
}

func (h *Handlers) PickUpClaim(ctx context.Context, req openapi.PickUpClaimRequestObject) (openapi.PickUpClaimResponseObject, error) {
	out, err := h.svc.PickUp(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.PickUpClaim200JSONResponse(out), nil
}

func (h *Handlers) RequestClaimInfo(ctx context.Context, req openapi.RequestClaimInfoRequestObject) (openapi.RequestClaimInfoResponseObject, error) {
	out, err := h.svc.RequestInfo(ctx, access.FromContext(ctx), req.Id, req.Body.Message)
	if err != nil {
		return nil, err
	}
	return openapi.RequestClaimInfo200JSONResponse(out), nil
}

func (h *Handlers) RespondToClaim(ctx context.Context, req openapi.RespondToClaimRequestObject) (openapi.RespondToClaimResponseObject, error) {
	var evidence []uuid.UUID
	if req.Body.EvidenceMediaIds != nil {
		evidence = *req.Body.EvidenceMediaIds
	}
	out, err := h.svc.Respond(ctx, access.FromContext(ctx), req.Id, req.Body.Message, evidence)
	if err != nil {
		return nil, err
	}
	return openapi.RespondToClaim200JSONResponse(out), nil
}

func (h *Handlers) GetClaimChecks(ctx context.Context, req openapi.GetClaimChecksRequestObject) (openapi.GetClaimChecksResponseObject, error) {
	out, err := h.svc.Checks(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.GetClaimChecks200JSONResponse(out), nil
}

func (h *Handlers) PutClaimChecks(ctx context.Context, req openapi.PutClaimChecksRequestObject) (openapi.PutClaimChecksResponseObject, error) {
	items := make([]CheckResult, 0, len(req.Body.Items))
	for _, it := range req.Body.Items {
		items = append(items, CheckResult{ItemKey: it.ItemKey, Result: string(it.Result), Note: it.Note})
	}
	out, err := h.svc.PutChecks(ctx, access.FromContext(ctx), req.Id, items)
	if err != nil {
		return nil, err
	}
	return openapi.PutClaimChecks200JSONResponse(out), nil
}

func (h *Handlers) ApproveClaim(ctx context.Context, req openapi.ApproveClaimRequestObject) (openapi.ApproveClaimResponseObject, error) {
	out, err := h.svc.Approve(ctx, access.FromContext(ctx), req.Id, req.Params.IfMatch, req.Body.DecisionNote)
	if err != nil {
		return nil, err
	}
	return openapi.ApproveClaim200JSONResponse(out), nil
}

func (h *Handlers) RejectClaim(ctx context.Context, req openapi.RejectClaimRequestObject) (openapi.RejectClaimResponseObject, error) {
	out, err := h.svc.Reject(ctx, access.FromContext(ctx), req.Id, string(req.Body.Reason), req.Body.Note)
	if err != nil {
		return nil, err
	}
	return openapi.RejectClaim200JSONResponse(out), nil
}

func (h *Handlers) RevokeClaim(ctx context.Context, req openapi.RevokeClaimRequestObject) (openapi.RevokeClaimResponseObject, error) {
	out, err := h.svc.Revoke(ctx, access.FromContext(ctx), req.Id, req.Body.Reason)
	if err != nil {
		return nil, err
	}
	return openapi.RevokeClaim200JSONResponse(out), nil
}

func (h *Handlers) ReopenClaim(ctx context.Context, req openapi.ReopenClaimRequestObject) (openapi.ReopenClaimResponseObject, error) {
	out, err := h.svc.Reopen(ctx, access.FromContext(ctx), req.Id, req.Body.Reason)
	if err != nil {
		return nil, err
	}
	return openapi.ReopenClaim200JSONResponse(out), nil
}

func (h *Handlers) WithdrawClaim(ctx context.Context, req openapi.WithdrawClaimRequestObject) (openapi.WithdrawClaimResponseObject, error) {
	out, err := h.svc.Withdraw(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.WithdrawClaim200JSONResponse(out), nil
}

func (h *Handlers) RegenerateHandoverCode(ctx context.Context, req openapi.RegenerateHandoverCodeRequestObject) (openapi.RegenerateHandoverCodeResponseObject, error) {
	out, err := h.svc.RegenerateCode(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.RegenerateHandoverCode200JSONResponse(out), nil
}

func (h *Handlers) ListClaimMessages(ctx context.Context, req openapi.ListClaimMessagesRequestObject) (openapi.ListClaimMessagesResponseObject, error) {
	items, err := h.svc.Messages(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.ListClaimMessages200JSONResponse{Items: items}, nil
}

func (h *Handlers) PostClaimMessage(ctx context.Context, req openapi.PostClaimMessageRequestObject) (openapi.PostClaimMessageResponseObject, error) {
	out, err := h.svc.PostMessage(ctx, access.FromContext(ctx), req.Id, req.Body.Message)
	if err != nil {
		return nil, err
	}
	return openapi.PostClaimMessage201JSONResponse(out), nil
}

func (h *Handlers) SetClaimInternalNote(ctx context.Context, req openapi.SetClaimInternalNoteRequestObject) (openapi.SetClaimInternalNoteResponseObject, error) {
	out, err := h.svc.SetInternalNote(ctx, access.FromContext(ctx), req.Id, req.Body.Note)
	if err != nil {
		return nil, err
	}
	return openapi.SetClaimInternalNote200JSONResponse(out), nil
}

func (h *Handlers) HandoverFoundReport(ctx context.Context, req openapi.HandoverFoundReportRequestObject) (openapi.HandoverFoundReportResponseObject, error) {
	b := req.Body
	in := HandoverInput{ClaimID: b.ClaimId, IDType: string(b.RecipientIdType), IDLast4: b.RecipientIdLast4, NameConfirmed: b.NameConfirmed}
	if b.Code != nil {
		in.Code = *b.Code
	}
	out, err := h.svc.Handover(ctx, access.FromContext(ctx), req.Id, in)
	if err != nil {
		return nil, err
	}
	return openapi.HandoverFoundReport200JSONResponse(out), nil
}
