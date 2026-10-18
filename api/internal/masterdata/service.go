package masterdata

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/geo"
	"github.com/pisondev/simpanugm/api/internal/platform/settings"
)

type Service struct {
	pool     *pgxpool.Pool
	q        *sqlc.Queries
	clock    clock.Clock
	settings *settings.Store
}

func NewService(pool *pgxpool.Pool, c clock.Clock, st *settings.Store) *Service {
	return &Service{pool: pool, q: sqlc.New(pool), clock: c, settings: st}
}

type Question struct {
	Key       string `json:"key" yaml:"key"`
	Question  string `json:"question" yaml:"question"`
	Required  bool   `json:"required" yaml:"required"`
	CheckHint string `json:"check_hint" yaml:"check_hint"`
}

type Category struct {
	sqlc.Category
	Questions []Question
	Children  []Category
}

func (s *Service) Categories(ctx context.Context) ([]Category, error) {
	rows, err := s.q.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	parents := []Category{}
	index := map[uuid.UUID]int{}
	for _, r := range rows {
		if r.ParentID == nil {
			index[r.ID] = len(parents)
			parents = append(parents, Category{Category: r})
		}
	}
	for _, r := range rows {
		if r.ParentID == nil {
			continue
		}
		i, ok := index[*r.ParentID]
		if !ok {
			continue
		}
		c := Category{Category: r}
		_ = json.Unmarshal(r.VerificationQuestions, &c.Questions)
		parents[i].Children = append(parents[i].Children, c)
	}
	return parents, nil
}

type LeafCategory struct {
	sqlc.GetCategoryRow
}

func (c LeafCategory) SensitivityValue() string {
	if c.Sensitivity == nil {
		return "normal"
	}
	return *c.Sensitivity
}

// reports carry leaf categories only
func (s *Service) Leaf(ctx context.Context, q *sqlc.Queries, id uuid.UUID) (LeafCategory, bool, error) {
	c, err := q.GetCategory(ctx, id)
	if db.IsNoRows(err) {
		return LeafCategory{}, false, nil
	}
	if err != nil {
		return LeafCategory{}, false, err
	}
	// a leaf under a switched off parent is gone from the pickers too
	if !c.Active || !c.ParentActive || c.ParentID == nil {
		return LeafCategory{}, false, nil
	}
	return LeafCategory{c}, true, nil
}

func (s *Service) Zones(ctx context.Context) ([]sqlc.Zone, error) { return s.q.ListZones(ctx) }

func (s *Service) Locations(ctx context.Context, q string, zone *uuid.UUID, limit int) ([]sqlc.ListLocationsRow, error) {
	params := sqlc.ListLocationsParams{ZoneID: zone, RowLimit: int32(limit)}
	if q = strings.TrimSpace(q); q != "" {
		params.Q = &q
	}
	return s.q.ListLocations(ctx, params)
}

type Post struct {
	sqlc.ListPostsRow
	Hours     map[string][][]string
	DistanceM *float64
}

func (s *Service) Posts(ctx context.Context, near *uuid.UUID) ([]Post, error) {
	rows, err := s.q.ListPosts(ctx)
	if err != nil {
		return nil, err
	}
	var origin *sqlc.GetLocationRow
	if near != nil {
		l, err := s.q.GetLocation(ctx, *near)
		if db.IsNoRows(err) {
			return nil, apperr.Validation(apperr.FieldError{Field: "near_location_id", Code: "unknown", Message: "Lokasi tidak ditemukan"})
		}
		if err != nil {
			return nil, err
		}
		origin = &l
	}
	out := make([]Post, 0, len(rows))
	for _, r := range rows {
		p := Post{ListPostsRow: r, Hours: map[string][][]string{}}
		_ = json.Unmarshal(r.OpeningHours, &p.Hours)
		if origin != nil {
			d := geo.HaversineM(origin.Lat, origin.Lng, r.Lat, r.Lng)
			p.DistanceM = &d
		}
		out = append(out, p)
	}
	if origin != nil {
		slices.SortStableFunc(out, func(a, b Post) int {
			switch {
			case *a.DistanceM < *b.DistanceM:
				return -1
			case *a.DistanceM > *b.DistanceM:
				return 1
			}
			return 0
		})
	}
	return out, nil
}

func (s *Service) Colours(ctx context.Context) ([]sqlc.Colour, error) { return s.q.ListColours(ctx) }

func (s *Service) ColourNames(ctx context.Context, q *sqlc.Queries) (map[string]bool, error) {
	rows, err := q.ListColours(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.Name] = true
	}
	return out, nil
}
