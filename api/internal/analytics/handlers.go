package analytics

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
)

type Handlers struct {
	svc *Service
}

func NewHandlers(svc *Service) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) ListModelVersions(ctx context.Context, req openapi.ListModelVersionsRequestObject) (openapi.ListModelVersionsResponseObject, error) {
	p := access.FromContext(ctx)
	if p == nil || !p.Role.Can(access.AIModelsRead) {
		return nil, apperr.Forbidden()
	}
	items, err := h.svc.ListModelVersions(ctx, req.Params.Component)
	if err != nil {
		return nil, err
	}
	return openapi.ListModelVersions200JSONResponse{Items: items}, nil
}

func (h *Handlers) ActivateModelVersion(ctx context.Context, req openapi.ActivateModelVersionRequestObject) (openapi.ActivateModelVersionResponseObject, error) {
	p := access.FromContext(ctx)
	if p == nil || !p.Role.Can(access.AIModelsActivate) {
		return nil, apperr.Forbidden()
	}
	item, err := h.svc.ActivateModelVersion(ctx, req.Id, p.UserID)
	if err != nil {
		return nil, err
	}
	return openapi.ActivateModelVersion200JSONResponse(*item), nil
}

func (h *Handlers) GetAiMonitoring(ctx context.Context, req openapi.GetAiMonitoringRequestObject) (openapi.GetAiMonitoringResponseObject, error) {
	p := access.FromContext(ctx)
	if p == nil || !p.Role.Can(access.AIMonitoringRead) {
		return nil, apperr.Forbidden()
	}
	res, err := h.svc.GetAiMonitoring(ctx, req.Params.From, req.Params.To, req.Params.ModelVersionId)
	if err != nil {
		return nil, err
	}
	return openapi.GetAiMonitoring200JSONResponse(*res), nil
}

func (h *Handlers) GetAiClusters(ctx context.Context, req openapi.GetAiClustersRequestObject) (openapi.GetAiClustersResponseObject, error) {
	p := access.FromContext(ctx)
	if p == nil || !p.Role.Can(access.AIClustersRead) {
		return nil, apperr.Forbidden()
	}
	res, err := h.svc.GetAiClusters(ctx, req.Params.RunId)
	if err != nil {
		return nil, err
	}
	return openapi.GetAiClusters200JSONResponse(*res), nil
}

func (h *Handlers) CreateAiClusterRun(ctx context.Context, _ openapi.CreateAiClusterRunRequestObject) (openapi.CreateAiClusterRunResponseObject, error) {
	p := access.FromContext(ctx)
	if p == nil || !p.Role.Can(access.AIClustersRun) {
		return nil, apperr.Forbidden()
	}
	var actorID *uuid.UUID
	if p != nil {
		actorID = &p.UserID
	}
	res, err := h.svc.CreateAiClusterRun(ctx, actorID)
	if err != nil {
		return nil, err
	}
	return openapi.CreateAiClusterRun200JSONResponse(*res), nil
}

func (h *Handlers) GetHotspots(ctx context.Context, req openapi.GetHotspotsRequestObject) (openapi.GetHotspotsResponseObject, error) {
	p := access.FromContext(ctx)
	if p == nil || !p.Role.Can(access.AnalyticsRead) {
		return nil, apperr.Forbidden()
	}
	var repType *string
	if req.Params.Type != nil {
		s := string(*req.Params.Type)
		repType = &s
	}
	res, err := h.svc.GetHotspots(ctx, req.Params.From, req.Params.To, repType, req.Params.CategoryId, req.Params.ZoneId)
	if err != nil {
		return nil, err
	}
	return openapi.GetHotspots200JSONResponse(*res), nil
}

func (h *Handlers) GetAnalyticsSummary(ctx context.Context, _ openapi.GetAnalyticsSummaryRequestObject) (openapi.GetAnalyticsSummaryResponseObject, error) {
	p := access.FromContext(ctx)
	if p == nil || !p.Role.Can(access.AnalyticsRead) {
		return nil, apperr.Forbidden()
	}
	res, err := h.svc.GetAnalyticsSummary(ctx)
	if err != nil {
		return nil, err
	}
	return openapi.GetAnalyticsSummary200JSONResponse(*res), nil
}

func (h *Handlers) GetAnalyticsTrends(ctx context.Context, req openapi.GetAnalyticsTrendsRequestObject) (openapi.GetAnalyticsTrendsResponseObject, error) {
	p := access.FromContext(ctx)
	if p == nil || !p.Role.Can(access.AnalyticsRead) {
		return nil, apperr.Forbidden()
	}
	res, err := h.svc.GetAnalyticsTrends(ctx, req.Params.From, req.Params.To, req.Params.CategoryId, req.Params.ZoneId)
	if err != nil {
		return nil, err
	}
	return openapi.GetAnalyticsTrends200JSONResponse(*res), nil
}

func (h *Handlers) ExportAnalytics(ctx context.Context, req openapi.ExportAnalyticsRequestObject) (openapi.ExportAnalyticsResponseObject, error) {
	p := access.FromContext(ctx)
	if p == nil || !p.Role.Can(access.AnalyticsExport) {
		return nil, apperr.Forbidden()
	}
	csvContent, err := h.svc.ExportAnalytics(ctx, req.Params.Month)
	if err != nil {
		return nil, err
	}
	return openapi.ExportAnalytics200TextcsvResponse{
		Body:          strings.NewReader(csvContent),
		ContentLength: int64(len(csvContent)),
	}, nil
}
