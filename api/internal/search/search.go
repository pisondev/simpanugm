package search

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/cursor"
	"github.com/pisondev/simpanugm/api/internal/reports"
)

type Service struct {
	q *sqlc.Queries
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{q: sqlc.New(pool)} }

type Handlers struct {
	svc *Service
}

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

var knownStatuses = map[string]bool{
	"reported": true, "potential_match": true, "under_verification": true, "claimed": true,
	"returned": true, "unclaimed": true, "closed": true,
}

func day(d *openapi_types.Date, add int) *time.Time {
	if d == nil {
		return nil
	}
	t := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, clock.WIB()).AddDate(0, 0, add).UTC()
	return &t
}

func (h *Handlers) SearchReports(ctx context.Context, req openapi.SearchReportsRequestObject) (openapi.SearchReportsResponseObject, error) {
	p := req.Params
	offset, err := cursor.DecodeOffset(p.Cursor)
	if err != nil {
		return nil, err
	}
	limit := cursor.Limit(p.Limit)
	params := sqlc.SearchReportsParams{
		IncludeLost:  p.Type == nil || *p.Type == openapi.SearchReportsParamsTypeLost,
		IncludeFound: p.Type == nil || *p.Type == openapi.SearchReportsParamsTypeFound,
		CategoryID:   p.CategoryId, ZoneID: p.ZoneId, LocationID: p.LocationId, PostID: p.PostId,
		FromTime: day(p.From, 0), ToTime: day(p.To, 1), Statuses: []string{},
		RowOffset: int32(offset), RowLimit: int32(limit + 1),
	}
	if p.Q != nil {
		q := strings.TrimSpace(*p.Q)
		if upper := strings.ToUpper(q); reports.IsCode(upper) {
			params.Code = &upper
		} else {
			params.Term = q
		}
	}
	if p.Custody != nil {
		c := string(*p.Custody)
		params.Custody = &c
	}
	if p.Status != nil {
		for _, st := range *p.Status {
			if !knownStatuses[st] {
				return nil, apperr.Validation(apperr.FieldError{Field: "status", Code: "unknown", Message: "Status tidak dikenal: " + st})
			}
		}
		params.Statuses = *p.Status
	}
	rows, err := h.svc.q.SearchReports(ctx, params)
	if err != nil {
		return nil, err
	}
	out := openapi.SearchResultList{Items: make([]openapi.SearchResult, 0, len(rows))}
	for i, r := range rows {
		if i == limit {
			out.NextCursor = cursor.EncodeOffset(offset + limit)
			break
		}
		item := openapi.SearchResult{
			Type: openapi.SearchResultType(r.Type), Id: r.ID, Code: r.Code, ItemName: r.ItemName,
			CategoryName: r.CategoryName, Status: r.Status, OccurredAt: r.OccurredAt,
		}
		if r.CustodyStatus != nil {
			c := openapi.CustodyStatus(*r.CustodyStatus)
			item.CustodyStatus = &c
		}
		if r.PlaceName != "" {
			item.PlaceName = &r.PlaceName
		}
		if r.PersonName != "" {
			item.PersonName = &r.PersonName
		}
		if params.Term != "" {
			rel := r.Relevance
			item.Relevance = &rel
		}
		out.Items = append(out.Items, item)
	}
	return openapi.SearchReports200JSONResponse(out), nil
}
