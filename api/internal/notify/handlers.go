package notify

import (
	"context"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/cursor"
)

type Handlers struct{ svc *Service }

func NewHandlers(s *Service) *Handlers { return &Handlers{svc: s} }

func (h *Handlers) ListNotifications(ctx context.Context, req openapi.ListNotificationsRequestObject) (openapi.ListNotificationsResponseObject, error) {
	k, err := cursor.Decode(req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	limit := cursor.Limit(req.Params.Limit)
	rows, err := h.svc.List(ctx, access.FromContext(ctx), ListInput{UnreadOnly: req.Params.Unread != nil && *req.Params.Unread, Limit: limit, Cursor: k})
	if err != nil {
		return nil, err
	}
	out := openapi.NotificationList{Items: make([]openapi.Notification, 0, len(rows))}
	for i, n := range rows {
		if i == limit {
			last := rows[limit-1]
			out.NextCursor = cursor.Encode(last.CreatedAt, last.ID)
			break
		}
		out.Items = append(out.Items, openapi.Notification{
			Id: n.ID, Event: n.Event, Title: n.Title, Body: n.Body, Link: n.Link,
			EntityType: n.EntityType, EntityId: n.EntityID, ReadAt: n.ReadAt, CreatedAt: n.CreatedAt,
		})
	}
	return openapi.ListNotifications200JSONResponse(out), nil
}

func (h *Handlers) CountUnreadNotifications(ctx context.Context, _ openapi.CountUnreadNotificationsRequestObject) (openapi.CountUnreadNotificationsResponseObject, error) {
	n, err := h.svc.UnreadCount(ctx, access.FromContext(ctx))
	if err != nil {
		return nil, err
	}
	return openapi.CountUnreadNotifications200JSONResponse{Count: n}, nil
}

func (h *Handlers) MarkNotificationRead(ctx context.Context, req openapi.MarkNotificationReadRequestObject) (openapi.MarkNotificationReadResponseObject, error) {
	if err := h.svc.MarkRead(ctx, access.FromContext(ctx), req.Id); err != nil {
		return nil, err
	}
	return openapi.MarkNotificationRead204Response{}, nil
}

func (h *Handlers) MarkAllNotificationsRead(ctx context.Context, _ openapi.MarkAllNotificationsReadRequestObject) (openapi.MarkAllNotificationsReadResponseObject, error) {
	p := access.FromContext(ctx)
	if err := h.svc.MarkAllRead(ctx, p); err != nil {
		return nil, err
	}
	n, err := h.svc.UnreadCount(ctx, p)
	if err != nil {
		return nil, err
	}
	return openapi.MarkAllNotificationsRead200JSONResponse{Count: n}, nil
}

func (h *Handlers) GetNotificationPreferences(ctx context.Context, _ openapi.GetNotificationPreferencesRequestObject) (openapi.GetNotificationPreferencesResponseObject, error) {
	prefs, err := h.svc.Preferences(ctx, access.FromContext(ctx))
	if err != nil {
		return nil, err
	}
	return openapi.GetNotificationPreferences200JSONResponse(preferencesOut(prefs)), nil
}

func (h *Handlers) PutNotificationPreferences(ctx context.Context, req openapi.PutNotificationPreferencesRequestObject) (openapi.PutNotificationPreferencesResponseObject, error) {
	in := make([]Preference, 0, len(req.Body.Items))
	for _, p := range req.Body.Items {
		in = append(in, Preference{Group: string(p.Group), App: p.App, Email: p.Email})
	}
	prefs, err := h.svc.SavePreferences(ctx, access.FromContext(ctx), in)
	if err != nil {
		return nil, err
	}
	return openapi.PutNotificationPreferences200JSONResponse(preferencesOut(prefs)), nil
}

func preferencesOut(prefs []Preference) openapi.NotificationPreferences {
	out := openapi.NotificationPreferences{Items: make([]openapi.NotificationPreference, 0, len(prefs))}
	for _, p := range prefs {
		out.Items = append(out.Items, openapi.NotificationPreference{Group: openapi.NotificationPreferenceGroup(p.Group), App: p.App, Email: p.Email})
	}
	return out
}
