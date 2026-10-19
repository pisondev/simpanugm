package officer

import (
	"context"

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
