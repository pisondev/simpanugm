package reports

import (
	"context"
	"encoding/json"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/cursor"
)

type Handlers struct {
	svc *Service
}

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func places(in *[]openapi.PlaceInput) *[]Place {
	if in == nil {
		return nil
	}
	out := make([]Place, 0, len(*in))
	for _, p := range *in {
		out = append(out, Place{LocationID: p.LocationId, ZoneID: p.ZoneId})
	}
	return &out
}

func listCursor(raw *string) (*ListCursor, error) {
	k, err := cursor.Decode(raw)
	if err != nil || k == nil {
		return nil, err
	}
	return &ListCursor{At: k.At, ID: k.ID}, nil
}

func statuses(s *openapi.StatusFilter) []string {
	if s == nil {
		return nil
	}
	return *s
}

// what the reporter did with each assistant suggestion, stored as sent after the contract validated its shape
func outcome(o *openapi.AssistOutcome) []byte {
	if o == nil {
		return nil
	}
	raw, err := json.Marshal(o)
	if err != nil {
		return nil
	}
	return raw
}

// LostInputFrom maps the request body of createLostReport, the claim form uses it for a claim from the public listing
func LostInputFrom(b openapi.LostReportCreate) LostInput {
	in := LostInput{
		CategoryID: b.CategoryId, ItemName: b.ItemName, Description: b.Description, PrivateDetails: b.PrivateDetails,
		Colours: b.Colours, Brand: b.Brand, LostFrom: b.LostFrom.UTC(), LostUntil: b.LostUntil.UTC(),
		LocationUnknown: b.LocationUnknown, LocationNote: b.LocationNote, MediaIDs: b.MediaIds, AssistOutcome: outcome(b.AssistOutcome),
	}
	if pl := places(&b.Places); pl != nil {
		in.Places = *pl
	}
	if b.Proxy != nil {
		in.ProxyName, in.ProxyPhone = &b.Proxy.Name, b.Proxy.Phone
	}
	return in
}

func (h *Handlers) CreateLostReport(ctx context.Context, req openapi.CreateLostReportRequestObject) (openapi.CreateLostReportResponseObject, error) {
	out, err := h.svc.CreateLost(ctx, access.FromContext(ctx), req.Params.IdempotencyKey, LostInputFrom(*req.Body))
	if err != nil {
		return nil, err
	}
	return openapi.CreateLostReport201JSONResponse(out), nil
}

func (h *Handlers) ListLostReports(ctx context.Context, req openapi.ListLostReportsRequestObject) (openapi.ListLostReportsResponseObject, error) {
	p := access.FromContext(ctx)
	cur, err := listCursor(req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	limit := cursor.Limit(req.Params.Limit)
	rows, err := h.svc.ListLost(ctx, p, ListInput{Mine: req.Params.Mine != nil && *req.Params.Mine, Statuses: statuses(req.Params.Status), Limit: limit}, cur)
	if err != nil {
		return nil, err
	}
	out := openapi.LostReportList{Items: make([]openapi.LostReportSummary, 0, len(rows))}
	for i, r := range rows {
		if i == limit {
			last := rows[limit-1]
			out.NextCursor = cursor.Encode(last.CreatedAt, last.ID)
			break
		}
		out.Items = append(out.Items, lostSummary(r, p))
	}
	return openapi.ListLostReports200JSONResponse(out), nil
}

func (h *Handlers) GetLostReport(ctx context.Context, req openapi.GetLostReportRequestObject) (openapi.GetLostReportResponseObject, error) {
	out, err := h.svc.GetLost(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.GetLostReport200JSONResponse(out), nil
}

func (h *Handlers) UpdateLostReport(ctx context.Context, req openapi.UpdateLostReportRequestObject) (openapi.UpdateLostReportResponseObject, error) {
	b := req.Body
	patch := LostPatch{
		CategoryID: b.CategoryId, ItemName: b.ItemName, Description: b.Description, PrivateDetails: b.PrivateDetails,
		Colours: b.Colours, Brand: b.Brand, LostFrom: b.LostFrom, LostUntil: b.LostUntil,
		LocationUnknown: b.LocationUnknown, Places: places(b.Places), LocationNote: b.LocationNote, MediaIDs: b.MediaIds,
	}
	out, err := h.svc.UpdateLost(ctx, access.FromContext(ctx), req.Id, req.Params.IfMatch, patch)
	if err != nil {
		return nil, err
	}
	return openapi.UpdateLostReport200JSONResponse(out), nil
}

func (h *Handlers) CloseLostReport(ctx context.Context, req openapi.CloseLostReportRequestObject) (openapi.CloseLostReportResponseObject, error) {
	out, err := h.svc.CloseLost(ctx, access.FromContext(ctx), req.Id, string(req.Body.Reason), req.Body.Note)
	if err != nil {
		return nil, err
	}
	return openapi.CloseLostReport200JSONResponse(out), nil
}

func (h *Handlers) GetLostReportHistory(ctx context.Context, req openapi.GetLostReportHistoryRequestObject) (openapi.GetLostReportHistoryResponseObject, error) {
	rows, err := h.svc.LostHistory(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.GetLostReportHistory200JSONResponse(historyList(rows)), nil
}

func (h *Handlers) CreateFoundReport(ctx context.Context, req openapi.CreateFoundReportRequestObject) (openapi.CreateFoundReportResponseObject, error) {
	b := req.Body
	in := FoundInput{
		CategoryID: b.CategoryId, ItemName: b.ItemName, Description: b.Description, HiddenDetails: b.HiddenDetails,
		Colours: b.Colours, Brand: b.Brand, FoundAt: b.FoundAt.UTC(), FoundLocationID: b.FoundLocationId,
		LocationNote: b.LocationNote, MediaIDs: b.MediaIds, AssistOutcome: outcome(b.AssistOutcome),
		CustodyMode: string(b.Custody.Mode), IntendedPostID: b.Custody.IntendedPostId,
	}
	if b.Proxy != nil {
		in.Proxy = &ProxyFound{FinderName: b.Proxy.FinderName, PostID: b.Proxy.PostId, StorageSlot: b.Proxy.StorageSlot, ConditionNote: b.Proxy.ConditionNote}
	}
	out, err := h.svc.CreateFound(ctx, access.FromContext(ctx), req.Params.IdempotencyKey, in)
	if err != nil {
		return nil, err
	}
	return openapi.CreateFoundReport201JSONResponse(out), nil
}

func (h *Handlers) ListFoundReports(ctx context.Context, req openapi.ListFoundReportsRequestObject) (openapi.ListFoundReportsResponseObject, error) {
	p := access.FromContext(ctx)
	cur, err := listCursor(req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	limit := cursor.Limit(req.Params.Limit)
	rows, err := h.svc.ListFound(ctx, p, ListInput{Mine: req.Params.Mine != nil && *req.Params.Mine, Statuses: statuses(req.Params.Status), Limit: limit}, cur)
	if err != nil {
		return nil, err
	}
	out := openapi.FoundReportList{Items: make([]openapi.FoundReportSummary, 0, len(rows))}
	for i, r := range rows {
		if i == limit {
			last := rows[limit-1]
			out.NextCursor = cursor.Encode(last.CreatedAt, last.ID)
			break
		}
		out.Items = append(out.Items, foundSummary(r, p))
	}
	return openapi.ListFoundReports200JSONResponse(out), nil
}

// date filters are calendar days in WIB, to is inclusive
func dayStart(d *openapi_types.Date, addDays int) *time.Time {
	if d == nil {
		return nil
	}
	t := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, clock.WIB()).AddDate(0, 0, addDays).UTC()
	return &t
}

func (h *Handlers) ListPublicFoundReports(ctx context.Context, req openapi.ListPublicFoundReportsRequestObject) (openapi.ListPublicFoundReportsResponseObject, error) {
	cur, err := listCursor(req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	limit := cursor.Limit(req.Params.Limit)
	rows, err := h.svc.ListPublicFound(ctx, PublicFilter{
		CategoryID: req.Params.CategoryId, ZoneID: req.Params.ZoneId,
		From: dayStart(req.Params.From, 0), To: dayStart(req.Params.To, 1), Limit: limit,
	}, cur)
	if err != nil {
		return nil, err
	}
	out := openapi.PublicFoundItemList{Items: make([]openapi.PublicFoundItem, 0, len(rows))}
	for i, r := range rows {
		if i == limit {
			last := rows[limit-1]
			out.NextCursor = cursor.Encode(last.FoundAt, last.ID)
			break
		}
		item := openapi.PublicFoundItem{
			Id: r.ID, Category: openapi.NamedRef{Id: r.CategoryID, Name: r.CategoryName}, ItemName: r.ItemName,
			Zone:          openapi.NamedRef{Id: r.ZoneID, Name: r.ZoneName},
			FoundDate:     openapi_types.Date{Time: r.FoundAt.In(clock.WIB())},
			CustodyStatus: openapi.CustodyStatus(r.CustodyStatus),
		}
		if r.HoldingPostID != nil && r.HoldingPostName != nil {
			item.HoldingPost = &openapi.NamedRef{Id: *r.HoldingPostID, Name: *r.HoldingPostName}
		}
		out.Items = append(out.Items, item)
	}
	return openapi.ListPublicFoundReports200JSONResponse(out), nil
}

func (h *Handlers) GetFoundReport(ctx context.Context, req openapi.GetFoundReportRequestObject) (openapi.GetFoundReportResponseObject, error) {
	out, err := h.svc.GetFound(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.GetFoundReport200JSONResponse(out), nil
}

func (h *Handlers) UpdateFoundReport(ctx context.Context, req openapi.UpdateFoundReportRequestObject) (openapi.UpdateFoundReportResponseObject, error) {
	b := req.Body
	patch := FoundPatch{
		CategoryID: b.CategoryId, ItemName: b.ItemName, Description: b.Description, HiddenDetails: b.HiddenDetails,
		Colours: b.Colours, Brand: b.Brand, FoundAt: b.FoundAt, FoundLocationID: b.FoundLocationId,
		LocationNote: b.LocationNote, MediaIDs: b.MediaIds,
	}
	if b.Custody != nil {
		mode := string(b.Custody.Mode)
		patch.CustodyMode, patch.IntendedPostID = &mode, b.Custody.IntendedPostId
	}
	out, err := h.svc.UpdateFound(ctx, access.FromContext(ctx), req.Id, req.Params.IfMatch, patch)
	if err != nil {
		return nil, err
	}
	return openapi.UpdateFoundReport200JSONResponse(out), nil
}

func (h *Handlers) CloseFoundReport(ctx context.Context, req openapi.CloseFoundReportRequestObject) (openapi.CloseFoundReportResponseObject, error) {
	out, err := h.svc.CloseFound(ctx, access.FromContext(ctx), req.Id, string(req.Body.Reason), req.Body.Note)
	if err != nil {
		return nil, err
	}
	return openapi.CloseFoundReport200JSONResponse(out), nil
}

func (h *Handlers) GetFoundReportHistory(ctx context.Context, req openapi.GetFoundReportHistoryRequestObject) (openapi.GetFoundReportHistoryResponseObject, error) {
	rows, err := h.svc.FoundHistory(ctx, access.FromContext(ctx), req.Id)
	if err != nil {
		return nil, err
	}
	return openapi.GetFoundReportHistory200JSONResponse(historyList(rows)), nil
}

func (h *Handlers) ReceiveFoundReport(ctx context.Context, req openapi.ReceiveFoundReportRequestObject) (openapi.ReceiveFoundReportResponseObject, error) {
	b := req.Body
	out, err := h.svc.Receive(ctx, access.FromContext(ctx), req.Id, b.PostId, b.StorageSlot, b.ConditionNote)
	if err != nil {
		return nil, err
	}
	return openapi.ReceiveFoundReport200JSONResponse(out), nil
}
