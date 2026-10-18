package reports

import (
	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/openapi"
)

func sensitivity(s *string) openapi.Sensitivity {
	if s == nil {
		return openapi.Normal
	}
	return openapi.Sensitivity(*s)
}

func photoRefs(media []sqlc.Medium) []openapi.PhotoRef {
	out := make([]openapi.PhotoRef, 0, len(media))
	for _, m := range media {
		out = append(out, openapi.PhotoRef{MediaId: m.ID, Position: int(m.Position)})
	}
	return out
}

func boolPtr(b bool) *bool { return &b }

func person(id *uuid.UUID, name, email, phone, nim *string) *openapi.PersonRef {
	if name == nil {
		return nil
	}
	return &openapi.PersonRef{Id: id, FullName: *name, Email: email, Phone: phone, NimNip: nim}
}

// owners see their own record, staff see everything including the reporter, nobody else reaches this
func lostView(row sqlc.GetLostReportRow, places []sqlc.ListLostPlacesRow, photos []sqlc.Medium, p *access.Principal) openapi.LostReport {
	r := row.LostReport
	out := openapi.LostReport{
		Id: r.ID, Code: r.Code, Status: openapi.LostStatus(r.Status), CloseReason: r.CloseReason, ClosedAt: r.ClosedAt,
		Category: openapi.CategoryRef{
			Id: r.CategoryID, Slug: row.CategorySlug, Name: row.CategoryName, ParentName: row.CategoryParentName,
			Sensitivity: sensitivity(row.CategorySensitivity),
		},
		ItemName: r.ItemName, Description: r.Description, PrivateDetails: r.PrivateDetails,
		Colours: r.Colours, Brand: r.Brand, LostFrom: r.LostFrom, LostUntil: r.LostUntil,
		LocationUnknown: r.LocationUnknown, LocationNote: r.LocationNote, Photos: photoRefs(photos),
		Places: make([]openapi.PlaceRef, 0, len(places)), Proxy: r.ProxyName != nil,
		Editable:       boolPtr(lostEditable(r) && (isOwnerLost(p, r) || p.IsStaff())),
		CandidateCount: int(row.CandidateCount), Version: int(r.Version), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	for _, pl := range places {
		ref := openapi.PlaceRef{Zone: &openapi.NamedRef{Id: pl.ZoneID, Name: pl.ZoneName}}
		if !pl.ZoneOnly && pl.LocationID != nil && pl.LocationName != nil {
			ref.Location = &openapi.NamedRef{Id: *pl.LocationID, Name: *pl.LocationName}
		}
		out.Places = append(out.Places, ref)
	}
	if p.IsStaff() {
		if r.ProxyName != nil {
			out.Reporter = &openapi.PersonRef{FullName: *r.ProxyName, Phone: r.ProxyPhone}
		} else {
			out.Reporter = person(r.ReporterID, row.ReporterFullName, row.ReporterEmail, row.ReporterPhone, row.ReporterNimNip)
		}
	}
	return out
}

func foundView(row sqlc.GetFoundReportRow, photos []sqlc.Medium, p *access.Principal, editable bool) openapi.FoundReport {
	r := row.FoundReport
	out := openapi.FoundReport{
		Id: r.ID, Code: r.Code, Status: openapi.FoundStatus(r.Status), CloseReason: r.CloseReason, ClosedAt: r.ClosedAt,
		Category: openapi.CategoryRef{
			Id: r.CategoryID, Slug: row.CategorySlug, Name: row.CategoryName, ParentName: row.CategoryParentName,
			Sensitivity: sensitivity(row.CategorySensitivity),
		},
		ItemName: r.ItemName, Description: r.Description, HiddenDetails: r.HiddenDetails,
		Colours: r.Colours, Brand: r.Brand, FoundAt: r.FoundAt,
		FoundLocation: openapi.LocationRef{Id: r.FoundLocationID, Name: row.LocationName, ZoneId: row.ZoneID, ZoneName: row.ZoneName},
		LocationNote:  r.LocationNote, Photos: photoRefs(photos),
		CustodyStatus: openapi.CustodyStatus(r.CustodyStatus), CustodyKeep: r.CustodyKeep,
		StorageSlot: r.StorageSlot, ReceivedAt: r.ReceivedAt, RetentionDueAt: r.RetentionDueAt,
		Proxy: r.FinderID == nil && r.CreatedBy != nil, ProxyFinderName: r.ProxyFinderName,
		Editable: boolPtr(editable), Version: int(r.Version), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.IntendedPostID != nil && row.IntendedPostName != nil {
		out.IntendedPost = &openapi.NamedRef{Id: *r.IntendedPostID, Name: *row.IntendedPostName}
	}
	if r.HoldingPostID != nil && row.HoldingPostName != nil {
		out.HoldingPost = &openapi.NamedRef{Id: *r.HoldingPostID, Name: *row.HoldingPostName}
	}
	if p.IsStaff() {
		out.Finder = person(r.FinderID, row.FinderFullName, row.FinderEmail, row.FinderPhone, row.FinderNimNip)
	} else {
		out.StorageSlot = nil
	}
	return out
}

func thumb(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func lostSummary(r sqlc.ListLostReportsRow, p *access.Principal) openapi.LostReportSummary {
	out := openapi.LostReportSummary{
		Id: r.ID, Code: r.Code, Status: openapi.LostStatus(r.Status), ItemName: r.ItemName, CategoryName: r.CategoryName,
		LostUntil: r.LostUntil, CandidateCount: int(r.CandidateCount), ThumbMediaId: thumb(r.ThumbMediaID), UpdatedAt: r.UpdatedAt,
	}
	if p.IsStaff() && r.ReporterName != "" {
		name := r.ReporterName
		out.ReporterName = &name
	}
	return out
}

func foundSummary(r sqlc.ListFoundReportsRow, p *access.Principal) openapi.FoundReportSummary {
	out := openapi.FoundReportSummary{
		Id: r.ID, Code: r.Code, Status: openapi.FoundStatus(r.Status), ItemName: r.ItemName, CategoryName: r.CategoryName,
		FoundAt: r.FoundAt, CustodyStatus: openapi.CustodyStatus(r.CustodyStatus), LocationName: r.LocationName,
		HoldingPostName: r.HoldingPostName, ThumbMediaId: thumb(r.ThumbMediaID), UpdatedAt: r.UpdatedAt,
	}
	if p.IsStaff() && r.FinderName != "" {
		name := r.FinderName
		out.FinderName = &name
	}
	return out
}

func historyList(rows []sqlc.ListStatusHistoryRow) openapi.HistoryList {
	out := openapi.HistoryList{Items: make([]openapi.HistoryEntry, 0, len(rows))}
	for _, h := range rows {
		out.Items = append(out.Items, openapi.HistoryEntry{
			Entity: openapi.HistoryEntryEntity(h.EntityType), FromStatus: h.FromStatus, ToStatus: h.ToStatus,
			Transition: h.Transition, Reason: h.Reason, ActorKind: openapi.HistoryEntryActorKind(h.ActorKind),
			ActorName: h.ActorName, CreatedAt: h.CreatedAt,
		})
	}
	return out
}
