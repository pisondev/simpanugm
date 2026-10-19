package search

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/mlclient"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/cursor"
	"github.com/pisondev/simpanugm/api/internal/reports"
)

type MLClient interface {
	EmbedText(ctx context.Context, texts []string, kinds []string) (map[string]mlclient.TextEmbeddings, error)
}

type Service struct {
	pool    *pgxpool.Pool
	q       *sqlc.Queries
	ml      MLClient
	enabled bool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, q: sqlc.New(pool)}
}

func (s *Service) WithML(ml MLClient) *Service {
	s.ml = ml
	return s
}

func (s *Service) WithSemanticSearch(enabled bool) *Service {
	s.enabled = enabled
	return s
}

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

const searchReportsSemanticSQL = `
SELECT r.type, r.id, r.code, r.item_name, r.category_name, r.status, r.custody_status, r.place_name,
       r.occurred_at, r.person_name, r.relevance, r.created_at
FROM (
    SELECT 'lost'::text AS type, lr.id, lr.code, lr.item_name, c.name AS category_name, lr.status,
           NULL::text AS custody_status,
           coalesce((SELECT coalesce(l.name, z.name)
              FROM lost_report_places pl
              LEFT JOIN locations l ON l.id = pl.location_id
              LEFT JOIN zones z ON z.id = pl.zone_id
             WHERE pl.lost_report_id = lr.id
             ORDER BY pl.position LIMIT 1), '')::text AS place_name,
           lr.lost_until AS occurred_at,
           coalesce(u.full_name, lr.proxy_name, '')::text AS person_name,
           (1 - (te.embedding <=> $12::vector))::float8 AS relevance,
           lr.created_at
    FROM lost_reports lr
    JOIN categories c ON c.id = lr.category_id
    LEFT JOIN users u ON u.id = lr.reporter_id
    JOIN LATERAL (
        SELECT embedding FROM text_embeddings_e5 te
        WHERE te.report_type = 'lost' AND te.report_id = lr.id
        ORDER BY te.updated_at DESC LIMIT 1
    ) te ON true
    WHERE $1::bool
      AND lr.hidden_at IS NULL
      AND ($11::text IS NULL OR lr.code = $11::text)
      AND $9::text IS NULL
      AND $10::uuid IS NULL
      AND ($3::uuid IS NULL OR c.id = $3::uuid OR c.parent_id = $3::uuid)
      AND ($4::uuid IS NULL OR EXISTS (
            SELECT 1 FROM lost_report_places pl LEFT JOIN locations l ON l.id = pl.location_id
             WHERE pl.lost_report_id = lr.id AND coalesce(pl.zone_id, l.zone_id) = $4::uuid))
      AND ($5::uuid IS NULL OR EXISTS (
            SELECT 1 FROM lost_report_places pl WHERE pl.lost_report_id = lr.id AND pl.location_id = $5::uuid))
      AND ($6::timestamptz IS NULL OR lr.lost_until >= $6::timestamptz)
      AND ($7::timestamptz IS NULL OR lr.lost_from < $7::timestamptz)
      AND (cardinality($8::text[]) = 0 OR lr.status = ANY($8::text[]))
    UNION ALL
    SELECT 'found'::text, fr.id, fr.code, fr.item_name, c.name, fr.status,
           fr.custody_status,
           l.name,
           fr.found_at,
           coalesce(u.full_name, fr.proxy_finder_name, '')::text,
           (1 - (te.embedding <=> $12::vector))::float8 AS relevance,
           fr.created_at
    FROM found_reports fr
    JOIN categories c ON c.id = fr.category_id
    JOIN locations l ON l.id = fr.found_location_id
    LEFT JOIN users u ON u.id = fr.finder_id
    JOIN LATERAL (
        SELECT embedding FROM text_embeddings_e5 te
        WHERE te.report_type = 'found' AND te.report_id = fr.id
        ORDER BY te.updated_at DESC LIMIT 1
    ) te ON true
    WHERE $2::bool
      AND fr.hidden_at IS NULL
      AND ($11::text IS NULL OR fr.code = $11::text)
      AND ($9::text IS NULL OR fr.custody_status = $9::text)
      AND ($10::uuid IS NULL OR fr.holding_post_id = $10::uuid OR fr.intended_post_id = $10::uuid)
      AND ($3::uuid IS NULL OR c.id = $3::uuid OR c.parent_id = $3::uuid)
      AND ($4::uuid IS NULL OR l.zone_id = $4::uuid)
      AND ($5::uuid IS NULL OR fr.found_location_id = $5::uuid)
      AND ($6::timestamptz IS NULL OR fr.found_at >= $6::timestamptz)
      AND ($7::timestamptz IS NULL OR fr.found_at < $7::timestamptz)
      AND (cardinality($8::text[]) = 0 OR fr.status = ANY($8::text[]))
) r
WHERE r.relevance > 0.0
ORDER BY r.relevance DESC, r.created_at DESC, r.id DESC
LIMIT $13;
`

func (s *Service) searchSemantic(ctx context.Context, p sqlc.SearchReportsParams, vecLit string, limit int32) ([]sqlc.SearchReportsRow, error) {
	if s.pool == nil {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, searchReportsSemanticSQL,
		p.IncludeLost,
		p.IncludeFound,
		p.CategoryID,
		p.ZoneID,
		p.LocationID,
		p.FromTime,
		p.ToTime,
		p.Statuses,
		p.Custody,
		p.PostID,
		p.Code,
		vecLit,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []sqlc.SearchReportsRow
	for rows.Next() {
		var r sqlc.SearchReportsRow
		if err := rows.Scan(
			&r.Type,
			&r.ID,
			&r.Code,
			&r.ItemName,
			&r.CategoryName,
			&r.Status,
			&r.CustodyStatus,
			&r.PlaceName,
			&r.OccurredAt,
			&r.PersonName,
			&r.Relevance,
			&r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type rankedItem struct {
	row       sqlc.SearchReportsRow
	score     float64
	relevance float64
}

func fuseRRF(kwRows, semRows []sqlc.SearchReportsRow, offset, limit int) []sqlc.SearchReportsRow {
	const k = 60.0
	itemsMap := make(map[string]*rankedItem)

	for rank, r := range kwRows {
		key := r.Type + ":" + r.ID.String()
		it, ok := itemsMap[key]
		if !ok {
			it = &rankedItem{row: r, relevance: r.Relevance}
			itemsMap[key] = it
		}
		it.score += 1.0 / (k + float64(rank+1))
	}

	for rank, r := range semRows {
		key := r.Type + ":" + r.ID.String()
		it, ok := itemsMap[key]
		if !ok {
			it = &rankedItem{row: r, relevance: r.Relevance}
			itemsMap[key] = it
		} else if r.Relevance > it.relevance {
			it.relevance = r.Relevance
		}
		it.score += 1.0 / (k + float64(rank+1))
	}

	all := make([]*rankedItem, 0, len(itemsMap))
	for _, it := range itemsMap {
		all = append(all, it)
	}

	sort.Slice(all, func(i, j int) bool {
		if math.Abs(all[i].score-all[j].score) > 1e-9 {
			return all[i].score > all[j].score
		}
		if !all[i].row.CreatedAt.Equal(all[j].row.CreatedAt) {
			return all[i].row.CreatedAt.After(all[j].row.CreatedAt)
		}
		return all[i].row.ID.String() > all[j].row.ID.String()
	})

	var paged []sqlc.SearchReportsRow
	for i := offset; i < len(all) && i < offset+limit+1; i++ {
		row := all[i].row
		row.Relevance = all[i].relevance
		paged = append(paged, row)
	}
	return paged
}

func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.Grow(len(v) * 10)
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
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

	var rows []sqlc.SearchReportsRow
	useSemantic := (h.svc.enabled || (p.Semantic != nil && *p.Semantic)) && h.svc.ml != nil && params.Term != ""
	if useSemantic {
		prompt := "query: " + params.Term
		embRes, err := h.svc.ml.EmbedText(ctx, []string{prompt}, []string{"e5"})
		if err == nil && len(embRes["e5"].Embeddings) > 0 {
			vecLit := vectorLiteral(embRes["e5"].Embeddings[0])
			fetchLimit := int32(offset + limit + 50)
			if fetchLimit < 100 {
				fetchLimit = 100
			}
			kwParams := params
			kwParams.RowOffset = 0
			kwParams.RowLimit = fetchLimit
			kwRows, kwErr := h.svc.q.SearchReports(ctx, kwParams)
			semRows, semErr := h.svc.searchSemantic(ctx, params, vecLit, fetchLimit)
			if kwErr == nil && semErr == nil {
				rows = fuseRRF(kwRows, semRows, offset, limit)
			}
		}
	}
	if rows == nil {
		var err error
		rows, err = h.svc.q.SearchReports(ctx, params)
		if err != nil {
			return nil, err
		}
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
