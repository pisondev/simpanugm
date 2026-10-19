package audit

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"strconv"
	"time"

	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/cursor"
)

type Handlers struct {
	svc *Service
}

func NewHandlers(svc *Service) *Handlers {
	return &Handlers{svc: svc}
}

type auditCursor struct {
	At time.Time `json:"t"`
	ID int64     `json:"id"`
}

func encodeAuditCursor(at time.Time, id int64) *string {
	b, _ := json.Marshal(auditCursor{At: at, ID: id})
	s := base64.RawURLEncoding.EncodeToString(b)
	return &s
}

func decodeAuditCursor(s *string) (*time.Time, *int64, error) {
	if s == nil || *s == "" {
		return nil, nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(*s)
	if err != nil {
		return nil, nil, apperr.BadRequest("Cursor tidak valid.")
	}
	var c auditCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, nil, apperr.BadRequest("Cursor tidak valid.")
	}
	return &c.At, &c.ID, nil
}

func (h *Handlers) AdminListAuditLogs(ctx context.Context, req openapi.AdminListAuditLogsRequestObject) (openapi.AdminListAuditLogsResponseObject, error) {
	isCSV := req.Params.Format != nil && string(*req.Params.Format) == "csv"
	if isCSV {
		rows, err := h.svc.Export(ctx, ListInput{
			ActorID:    req.Params.ActorId,
			Action:     req.Params.Action,
			EntityType: req.Params.EntityType,
			EntityID:   req.Params.EntityId,
			From:       req.Params.From,
			To:         req.Params.To,
			Limit:      5000,
		})
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		_ = w.Write([]string{"id", "occurred_at", "actor_id", "actor_name", "actor_role", "action", "entity_type", "entity_id", "changes", "request_id", "ip", "user_agent"})
		for _, r := range rows {
			var actorIDStr, entityIDStr, reqIDStr, ipStr, uaStr, nameStr, roleStr string
			if r.ActorID != nil {
				actorIDStr = r.ActorID.String()
			}
			if r.ActorName != nil {
				nameStr = *r.ActorName
			}
			if r.ActorRole != nil {
				roleStr = *r.ActorRole
			}
			if r.EntityID != nil {
				entityIDStr = r.EntityID.String()
			}
			if r.RequestID != nil {
				reqIDStr = *r.RequestID
			}
			if r.Ip != nil {
				ipStr = r.Ip.String()
			}
			if r.UserAgent != nil {
				uaStr = *r.UserAgent
			}
			changesStr := string(r.Changes)
			_ = w.Write([]string{
				strconv.FormatInt(r.ID, 10),
				r.OccurredAt.UTC().Format(time.RFC3339),
				actorIDStr,
				nameStr,
				roleStr,
				r.Action,
				r.EntityType,
				entityIDStr,
				changesStr,
				reqIDStr,
				ipStr,
				uaStr,
			})
		}
		w.Flush()
		return openapi.AdminListAuditLogs200TextcsvResponse{
			Body:          bytes.NewReader(buf.Bytes()),
			ContentLength: int64(buf.Len()),
		}, nil
	}

	curTime, curID, err := decodeAuditCursor(req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	limit := cursor.Limit(req.Params.Limit)
	rows, err := h.svc.List(ctx, ListInput{
		ActorID:    req.Params.ActorId,
		Action:     req.Params.Action,
		EntityType: req.Params.EntityType,
		EntityID:   req.Params.EntityId,
		From:       req.Params.From,
		To:         req.Params.To,
		CursorTime: curTime,
		CursorID:   curID,
		Limit:      limit + 1,
	})
	if err != nil {
		return nil, err
	}
	out := openapi.AuditLogList{Items: []openapi.AuditLogItem{}}
	for i, r := range rows {
		if i == limit {
			last := rows[limit-1]
			out.NextCursor = encodeAuditCursor(last.OccurredAt, last.ID)
			break
		}
		var changesPtr *map[string]interface{}
		if len(r.Changes) > 0 {
			var ch map[string]interface{}
			if err := json.Unmarshal(r.Changes, &ch); err == nil {
				changesPtr = &ch
			}
		}
		var ipStr *string
		if r.Ip != nil {
			s := r.Ip.String()
			ipStr = &s
		}
		out.Items = append(out.Items, openapi.AuditLogItem{
			Id:         int(r.ID),
			OccurredAt: r.OccurredAt,
			ActorId:    r.ActorID,
			ActorName:  r.ActorName,
			ActorRole:  r.ActorRole,
			Action:     r.Action,
			EntityType: r.EntityType,
			EntityId:   r.EntityID,
			Changes:    changesPtr,
			RequestId:  r.RequestID,
			Ip:         ipStr,
			UserAgent:  r.UserAgent,
		})
	}
	return openapi.AdminListAuditLogs200JSONResponse(out), nil
}
