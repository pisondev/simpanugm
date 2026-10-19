package custody

import (
	"context"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/openapi"
)

type ReportService interface {
	Transfer(ctx context.Context, p *access.Principal, ref string, toPost uuid.UUID, note *string) (openapi.FoundReport, error)
	ExtendRetention(ctx context.Context, p *access.Principal, ref string, days int32, reason string) (openapi.FoundReport, error)
	Dispose(ctx context.Context, p *access.Principal, ref, kind string, recipient *string, note string) (openapi.FoundReport, error)
	RevertUnclaimed(ctx context.Context, p *access.Principal, ref string) (openapi.FoundReport, error)
}

type Handlers struct {
	svc *Service
	rep ReportService
}

func NewHandlers(svc *Service, rep ReportService) *Handlers { return &Handlers{svc: svc, rep: rep} }

func (h *Handlers) GetPostInventory(ctx context.Context, req openapi.GetPostInventoryRequestObject) (openapi.GetPostInventoryResponseObject, error) {
	sort := ""
	if req.Params.Sort != nil {
		sort = string(*req.Params.Sort)
	}
	post, items, err := h.svc.Inventory(ctx, req.Id, sort)
	if err != nil {
		return nil, err
	}
	out := openapi.InventoryList{Post: openapi.NamedRef{Id: post.ID, Name: post.Name}, Items: make([]openapi.InventoryItem, 0, len(items))}
	for _, it := range items {
		item := openapi.InventoryItem{
			Id: it.ID, Code: it.Code, ItemName: it.ItemName, CategoryName: it.CategoryName,
			Sensitivity: openapi.Sensitivity(it.Sensitivity), StorageSlot: it.StorageSlot,
			ReceivedAt: it.ReceivedAt, RetentionDueAt: it.RetentionDueAt, AgeDays: it.AgeDays,
			Status: openapi.FoundStatus(it.Status), ClaimState: openapi.InventoryItemClaimStateNone,
		}
		if it.ThumbMediaID != uuid.Nil {
			id := it.ThumbMediaID
			item.ThumbMediaId = &id
		}
		out.Items = append(out.Items, item)
	}
	return openapi.GetPostInventory200JSONResponse(out), nil
}

func (h *Handlers) TransferCustody(ctx context.Context, req openapi.TransferCustodyRequestObject) (openapi.TransferCustodyResponseObject, error) {
	out, err := h.rep.Transfer(ctx, access.FromContext(ctx), req.Id, req.Body.ToPostId, req.Body.Note)
	if err != nil {
		return nil, err
	}
	return openapi.TransferCustody200JSONResponse(out), nil
}

func (h *Handlers) ExtendRetention(ctx context.Context, req openapi.ExtendRetentionRequestObject) (openapi.ExtendRetentionResponseObject, error) {
	out, err := h.rep.ExtendRetention(ctx, access.FromContext(ctx), req.Id, int32(req.Body.Days), req.Body.Reason)
	if err != nil {
		return nil, err
	}
	return openapi.ExtendRetention200JSONResponse(out), nil
}

func (h *Handlers) DisposeFoundReport(ctx context.Context, req openapi.DisposeFoundReportRequestObject) (openapi.DisposeFoundReportResponseObject, error) {
	out, err := h.rep.Dispose(ctx, access.FromContext(ctx), req.Id, string(req.Body.Kind), req.Body.Recipient, req.Body.Note)
	if err != nil {
		return nil, err
	}
	return openapi.DisposeFoundReport200JSONResponse(out), nil
}

func (h *Handlers) RevertUnclaimed(ctx context.Context, req openapi.RevertUnclaimedRequestObject) (openapi.RevertUnclaimedResponseObject, error) {
	out, err := h.rep.RevertUnclaimed(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.RevertUnclaimed200JSONResponse(out), nil
}
