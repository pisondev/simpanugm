package masterdata

import (
	"context"

	"github.com/pisondev/simpanugm/api/internal/openapi"
)

type Handlers struct {
	svc *Service
}

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func toCategory(c Category) openapi.Category {
	out := openapi.Category{Id: c.ID, Slug: c.Slug, Name: c.Name, ParentId: c.ParentID}
	if c.Sensitivity != nil {
		s := openapi.Sensitivity(*c.Sensitivity)
		out.Sensitivity = &s
	}
	if c.ParentID != nil {
		qs := make([]openapi.VerificationQuestion, 0, len(c.Questions))
		for _, q := range c.Questions {
			hint := q.CheckHint
			qs = append(qs, openapi.VerificationQuestion{Key: q.Key, Question: q.Question, Required: q.Required, CheckHint: &hint})
		}
		out.VerificationQuestions = &qs
	} else {
		children := make([]openapi.Category, 0, len(c.Children))
		for _, ch := range c.Children {
			children = append(children, toCategory(ch))
		}
		out.Children = &children
	}
	return out
}

func (h *Handlers) ListCategories(ctx context.Context, _ openapi.ListCategoriesRequestObject) (openapi.ListCategoriesResponseObject, error) {
	cats, err := h.svc.Categories(ctx)
	if err != nil {
		return nil, err
	}
	out := openapi.CategoryList{Items: make([]openapi.Category, 0, len(cats))}
	for _, c := range cats {
		out.Items = append(out.Items, toCategory(c))
	}
	return openapi.ListCategories200JSONResponse(out), nil
}

func (h *Handlers) ListZones(ctx context.Context, _ openapi.ListZonesRequestObject) (openapi.ListZonesResponseObject, error) {
	zones, err := h.svc.Zones(ctx)
	if err != nil {
		return nil, err
	}
	out := openapi.ZoneList{Items: make([]openapi.Zone, 0, len(zones))}
	for _, z := range zones {
		out.Items = append(out.Items, openapi.Zone{Id: z.ID, Slug: z.Slug, Name: z.Name, CentroidLat: z.CentroidLat, CentroidLng: z.CentroidLng})
	}
	return openapi.ListZones200JSONResponse(out), nil
}

func (h *Handlers) ListLocations(ctx context.Context, req openapi.ListLocationsRequestObject) (openapi.ListLocationsResponseObject, error) {
	q := ""
	if req.Params.Q != nil {
		q = *req.Params.Q
	}
	limit := 50
	if req.Params.Limit != nil && *req.Params.Limit > 0 && *req.Params.Limit <= 200 {
		limit = *req.Params.Limit
	}
	rows, err := h.svc.Locations(ctx, q, req.Params.ZoneId, limit)
	if err != nil {
		return nil, err
	}
	out := openapi.LocationList{Items: make([]openapi.Location, 0, len(rows))}
	for _, l := range rows {
		out.Items = append(out.Items, openapi.Location{
			Id: l.ID, Slug: l.Slug, Name: l.Name, Aliases: l.Aliases, Kind: openapi.LocationKind(l.Kind),
			ZoneId: l.ZoneID, ZoneName: l.ZoneName, Lat: l.Lat, Lng: l.Lng,
		})
	}
	return openapi.ListLocations200JSONResponse(out), nil
}

func (h *Handlers) ListPosts(ctx context.Context, req openapi.ListPostsRequestObject) (openapi.ListPostsResponseObject, error) {
	posts, err := h.svc.Posts(ctx, req.Params.NearLocationId)
	if err != nil {
		return nil, err
	}
	out := openapi.PostList{Items: make([]openapi.Post, 0, len(posts))}
	for _, p := range posts {
		out.Items = append(out.Items, openapi.Post{
			Id: p.ID, Slug: p.Slug, Name: p.Name, LocationId: p.LocationID, LocationName: p.LocationName,
			ZoneId: p.ZoneID, ZoneName: p.ZoneName, Phone: p.Phone, OpeningHours: p.Hours, DistanceM: p.DistanceM,
		})
	}
	return openapi.ListPosts200JSONResponse(out), nil
}

func (h *Handlers) ListColours(ctx context.Context, _ openapi.ListColoursRequestObject) (openapi.ListColoursResponseObject, error) {
	rows, err := h.svc.Colours(ctx)
	if err != nil {
		return nil, err
	}
	out := openapi.ColourList{Items: make([]openapi.Colour, 0, len(rows))}
	for _, c := range rows {
		out.Items = append(out.Items, openapi.Colour{Name: c.Name, Hex: c.Hex})
	}
	return openapi.ListColours200JSONResponse(out), nil
}
