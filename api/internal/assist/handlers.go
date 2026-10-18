package assist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/openapi"
)

func hashOf(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type Handlers struct{ svc *Service }

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) AssistAnalyzeImage(ctx context.Context, req openapi.AssistAnalyzeImageRequestObject) (openapi.AssistAnalyzeImageResponseObject, error) {
	out, err := h.svc.Analyze(ctx, access.FromContext(ctx), req.Body.MediaId, string(req.Body.ReportType))
	if err != nil {
		return nil, err
	}
	return openapi.AssistAnalyzeImage200JSONResponse(out), nil
}

func (h *Handlers) AssistDescribe(ctx context.Context, req openapi.AssistDescribeRequestObject) (openapi.AssistDescribeResponseObject, error) {
	out, err := h.svc.Describe(ctx, *req.Body)
	if err != nil {
		return nil, err
	}
	return openapi.AssistDescribe200JSONResponse(out), nil
}
