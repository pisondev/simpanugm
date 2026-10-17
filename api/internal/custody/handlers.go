package custody

import (
	"context"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/openapi"
)

type Handlers struct {
	svc *Service
}

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

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
			Status: openapi.FoundStatus(it.Status), ClaimState: openapi.None,
		}
		if it.ThumbMediaID != uuid.Nil {
			id := it.ThumbMediaID
			item.ThumbMediaId = &id
		}
		out.Items = append(out.Items, item)
	}
	return openapi.GetPostInventory200JSONResponse(out), nil
}
