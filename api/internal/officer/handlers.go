package officer

import (
	"context"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/openapi"
)

type Handlers struct {
	svc *Service
}

func NewHandlers(svc *Service) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) GetOfficerSummary(ctx context.Context, req openapi.GetOfficerSummaryRequestObject) (openapi.GetOfficerSummaryResponseObject, error) {
	p := access.FromContext(ctx)
	scope := "posts"
	if req.Params.Scope != nil {
		scope = string(*req.Params.Scope)
	}
	summary, err := h.svc.Summary(ctx, p, scope)
	if err != nil {
		return nil, err
	}
	return openapi.GetOfficerSummary200JSONResponse(summary), nil
}

func (h *Handlers) GetOfficerQueue(ctx context.Context, req openapi.GetOfficerQueueRequestObject) (openapi.GetOfficerQueueResponseObject, error) {
	p := access.FromContext(ctx)
	list, err := h.svc.Queue(ctx, p, req.Params)
	if err != nil {
		return nil, err
	}
	return openapi.GetOfficerQueue200JSONResponse(list), nil
}

func (h *Handlers) GetOfficerBoard(ctx context.Context, req openapi.GetOfficerBoardRequestObject) (openapi.GetOfficerBoardResponseObject, error) {
	p := access.FromContext(ctx)
	board, err := h.svc.Board(ctx, p, string(req.Params.ReportType), req.Params.ReportId)
	if err != nil {
		return nil, err
	}
	return openapi.GetOfficerBoard200JSONResponse(*board), nil
}

func (h *Handlers) ListOfficerDuplicates(ctx context.Context, req openapi.ListOfficerDuplicatesRequestObject) (openapi.ListOfficerDuplicatesResponseObject, error) {
	var status *string
	if req.Params.Status != nil {
		s := string(*req.Params.Status)
		status = &s
	}
	var reportType *string
	if req.Params.Type != nil {
		rt := string(*req.Params.Type)
		reportType = &rt
	}
	limit := 20
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	list, err := h.svc.ListDuplicates(ctx, status, reportType, limit)
	if err != nil {
		return nil, err
	}
	return openapi.ListOfficerDuplicates200JSONResponse(list), nil
}

func (h *Handlers) MergeOfficerDuplicates(ctx context.Context, req openapi.MergeOfficerDuplicatesRequestObject) (openapi.MergeOfficerDuplicatesResponseObject, error) {
	p := access.FromContext(ctx)
	item, err := h.svc.MergeDuplicates(ctx, p, string(req.Body.ReportType), req.Body.SurvivorId, req.Body.DuplicateId)
	if err != nil {
		return nil, err
	}
	return openapi.MergeOfficerDuplicates200JSONResponse(item), nil
}

func (h *Handlers) ConfirmOfficerDuplicate(ctx context.Context, req openapi.ConfirmOfficerDuplicateRequestObject) (openapi.ConfirmOfficerDuplicateResponseObject, error) {
	p := access.FromContext(ctx)
	var survivorID *uuid.UUID
	if req.Body != nil {
		survivorID = req.Body.SurvivorId
	}
	item, err := h.svc.ConfirmDuplicate(ctx, p, req.Id, survivorID)
	if err != nil {
		return nil, err
	}
	return openapi.ConfirmOfficerDuplicate200JSONResponse(item), nil
}

func (h *Handlers) RejectOfficerDuplicate(ctx context.Context, req openapi.RejectOfficerDuplicateRequestObject) (openapi.RejectOfficerDuplicateResponseObject, error) {
	p := access.FromContext(ctx)
	item, err := h.svc.RejectDuplicate(ctx, p, req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.RejectOfficerDuplicate200JSONResponse(item), nil
}
