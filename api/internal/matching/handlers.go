package matching

import (
	"context"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/openapi"
)

type Handlers struct{ svc *Service }

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func includeLow(v *bool) bool { return v != nil && *v }

func (h *Handlers) ListLostReportMatches(ctx context.Context, req openapi.ListLostReportMatchesRequestObject) (openapi.ListLostReportMatchesResponseObject, error) {
	items, err := h.svc.ListForLost(ctx, access.FromContext(ctx), req.Id, includeLow(req.Params.IncludeLow))
	if err != nil {
		return nil, err
	}
	return openapi.ListLostReportMatches200JSONResponse{Items: items}, nil
}

func (h *Handlers) ListFoundReportMatches(ctx context.Context, req openapi.ListFoundReportMatchesRequestObject) (openapi.ListFoundReportMatchesResponseObject, error) {
	items, err := h.svc.ListForFound(ctx, access.FromContext(ctx), req.Id, includeLow(req.Params.IncludeLow))
	if err != nil {
		return nil, err
	}
	return openapi.ListFoundReportMatches200JSONResponse{Items: items}, nil
}

func (h *Handlers) GetFoundReportMatchHint(ctx context.Context, req openapi.GetFoundReportMatchHintRequestObject) (openapi.GetFoundReportMatchHintResponseObject, error) {
	items, err := h.svc.Hint(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.GetFoundReportMatchHint200JSONResponse{Items: items}, nil
}

func (h *Handlers) GetMatch(ctx context.Context, req openapi.GetMatchRequestObject) (openapi.GetMatchResponseObject, error) {
	m, err := h.svc.Get(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.GetMatch200JSONResponse(m), nil
}

func (h *Handlers) SubmitMatchFeedback(ctx context.Context, req openapi.SubmitMatchFeedbackRequestObject) (openapi.SubmitMatchFeedbackResponseObject, error) {
	if err := h.svc.Feedback(ctx, access.FromContext(ctx), req.Id, string(req.Body.Verdict), req.Body.Reason); err != nil {
		return nil, err
	}
	return openapi.SubmitMatchFeedback204Response{}, nil
}

func (h *Handlers) DismissMatch(ctx context.Context, req openapi.DismissMatchRequestObject) (openapi.DismissMatchResponseObject, error) {
	if err := h.svc.Dismiss(ctx, access.FromContext(ctx), req.Id, req.Body.Reason); err != nil {
		return nil, err
	}
	return openapi.DismissMatch204Response{}, nil
}

func (h *Handlers) RematchReport(ctx context.Context, req openapi.RematchReportRequestObject) (openapi.RematchReportResponseObject, error) {
	if err := h.svc.Rematch(ctx, access.FromContext(ctx), req.Type, req.Id); err != nil {
		return nil, err
	}
	return openapi.RematchReport202JSONResponse{Message: "Pencocokan ulang dijadwalkan."}, nil
}
