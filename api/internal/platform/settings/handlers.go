package settings

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/openapi"
)

type Handlers struct {
	store *Store
}

func NewHandlers(store *Store) *Handlers {
	return &Handlers{store: store}
}

func (h *Handlers) AdminGetSettings(ctx context.Context, _ openapi.AdminGetSettingsRequestObject) (openapi.AdminGetSettingsResponseObject, error) {
	rows, err := h.store.ListDetailed(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]openapi.SystemSettingItem, 0, len(rows))
	for _, r := range rows {
		var v any
		_ = json.Unmarshal(r.Value, &v)
		var updatedAt *time.Time
		if !r.UpdatedAt.IsZero() {
			updatedAt = &r.UpdatedAt
		}
		items = append(items, openapi.SystemSettingItem{
			Key:           r.Key,
			Value:         v,
			UpdatedAt:     updatedAt,
			UpdatedBy:     r.UpdatedBy,
			UpdatedByName: r.UpdatedByName,
		})
	}
	return openapi.AdminGetSettings200JSONResponse{Items: items}, nil
}

func (h *Handlers) AdminUpdateSettings(ctx context.Context, req openapi.AdminUpdateSettingsRequestObject) (openapi.AdminUpdateSettingsResponseObject, error) {
	p := access.FromContext(ctx)
	var actorID *uuid.UUID
	if p != nil {
		actorID = &p.UserID
	}
	rows, err := h.store.Update(ctx, req.Body.Settings, actorID)
	if err != nil {
		return nil, err
	}
	items := make([]openapi.SystemSettingItem, 0, len(rows))
	for _, r := range rows {
		var v any
		_ = json.Unmarshal(r.Value, &v)
		var updatedAt *time.Time
		if !r.UpdatedAt.IsZero() {
			updatedAt = &r.UpdatedAt
		}
		items = append(items, openapi.SystemSettingItem{
			Key:           r.Key,
			Value:         v,
			UpdatedAt:     updatedAt,
			UpdatedBy:     r.UpdatedBy,
			UpdatedByName: r.UpdatedByName,
		})
	}
	return openapi.AdminUpdateSettings200JSONResponse{Items: items}, nil
}
