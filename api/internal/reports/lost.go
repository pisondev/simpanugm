package reports

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/media"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/idem"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/textx"
)

type Place struct {
	LocationID *uuid.UUID `json:"location_id,omitempty"`
	ZoneID     *uuid.UUID `json:"zone_id,omitempty"`
}

type LostInput struct {
	CategoryID      uuid.UUID   `json:"category_id"`
	ItemName        string      `json:"item_name"`
	Description     string      `json:"description"`
	PrivateDetails  *string     `json:"private_details"`
	Colours         []string    `json:"colours"`
	Brand           *string     `json:"brand"`
	LostFrom        time.Time   `json:"lost_from"`
	LostUntil       time.Time   `json:"lost_until"`
	LocationUnknown bool        `json:"location_unknown"`
	Places          []Place     `json:"places"`
	LocationNote    *string     `json:"location_note"`
	MediaIDs        []uuid.UUID `json:"media_ids"`
	ProxyName       *string     `json:"proxy_name"`
	ProxyPhone      *string     `json:"proxy_phone"`
	AssistOutcome   []byte      `json:"-"`
}

type LostPatch struct {
	CategoryID      *uuid.UUID
	ItemName        *string
	Description     *string
	PrivateDetails  *string
	Colours         *[]string
	Brand           *string
	LostFrom        *time.Time
	LostUntil       *time.Time
	LocationUnknown *bool
	Places          *[]Place
	LocationNote    *string
	MediaIDs        *[]uuid.UUID
}

func (in *LostInput) normalize() {
	in.ItemName = textx.Value(textx.OptString(in.ItemName))
	in.Description = textx.Value(textx.OptString(in.Description))
	in.PrivateDetails = textx.Opt(in.PrivateDetails)
	in.Brand = textx.Opt(in.Brand)
	in.LocationNote = textx.Opt(in.LocationNote)
	in.ProxyName = textx.Opt(in.ProxyName)
	in.ProxyPhone = textx.Opt(in.ProxyPhone)
	if in.Colours == nil {
		in.Colours = []string{}
	}
	if in.MediaIDs == nil {
		in.MediaIDs = []uuid.UUID{}
	}
	if in.LocationUnknown {
		in.Places = nil
	}
}

func (s *Service) validateLost(ctx context.Context, q *sqlc.Queries, in *LostInput, now time.Time, proxy bool) error {
	var v apperr.Collector
	in.normalize()
	checkText(&v, textRule{"item_name", textx.OptString(in.ItemName), 3, 80, "Nama barang"})
	checkText(&v, textRule{"description", textx.OptString(in.Description), 20, 1000, "Deskripsi"})
	checkText(&v, textRule{"private_details", in.PrivateDetails, 0, 500, "Ciri khusus"})
	checkText(&v, textRule{"brand", in.Brand, 0, 40, "Merek"})
	checkText(&v, textRule{"location_note", in.LocationNote, 0, 200, "Catatan lokasi"})
	if err := s.checkColours(ctx, q, &v, in.Colours); err != nil {
		return err
	}
	if _, err := s.leaf(ctx, q, &v, in.CategoryID); err != nil {
		return err
	}
	switch {
	case in.LostFrom.After(in.LostUntil):
		v.Add("lost_until", "before_start", "Waktu akhir harus setelah waktu awal")
	case in.LostUntil.Sub(in.LostFrom) > lostWindowMax:
		v.Add("lost_until", "window_too_long", "Rentang waktu maksimal 7 hari")
	}
	if in.LostUntil.After(now.Add(futureTolerance)) {
		v.Add("lost_until", "in_future", "Waktu tidak boleh di masa depan")
	}
	if in.LostFrom.Before(now.Add(-lostLookback)) {
		v.Add("lost_from", "too_old", "Waktu kehilangan maksimal 90 hari yang lalu")
	}
	if !in.LocationUnknown {
		if len(in.Places) == 0 || len(in.Places) > 3 {
			v.Add("places", "count", "Pilih 1 sampai 3 lokasi, atau tandai tidak tahu lokasinya")
		}
		for _, pl := range in.Places {
			if err := s.checkPlace(ctx, q, &v, pl); err != nil {
				return err
			}
		}
	}
	if proxy {
		checkText(&v, textRule{"proxy.name", in.ProxyName, 2, 100, "Nama pelapor"})
		if in.ProxyPhone != nil {
			phone := textx.NormalizePhone(*in.ProxyPhone)
			v.Check(phone != "", "proxy.phone", "invalid", "Nomor telepon tidak valid")
			in.ProxyPhone = &phone
		}
	}
	return v.Err()
}

func (s *Service) checkPlace(ctx context.Context, q *sqlc.Queries, v *apperr.Collector, pl Place) error {
	switch {
	case (pl.LocationID == nil) == (pl.ZoneID == nil):
		v.Add("places", "invalid", "Setiap lokasi berupa satu gedung atau satu zona")
	case pl.LocationID != nil:
		l, err := q.GetLocation(ctx, *pl.LocationID)
		if db.IsNoRows(err) || (err == nil && !l.Active) {
			v.Add("places", "unknown_location", "Lokasi tidak ditemukan")
		} else if err != nil {
			return err
		}
	default:
		z, err := q.GetZone(ctx, *pl.ZoneID)
		if db.IsNoRows(err) || (err == nil && !z.Active) {
			v.Add("places", "unknown_zone", "Zona tidak ditemukan")
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) replacePlaces(ctx context.Context, q *sqlc.Queries, id uuid.UUID, places []Place) error {
	if err := q.DeleteLostPlaces(ctx, id); err != nil {
		return err
	}
	for i, pl := range places {
		if err := q.InsertLostPlace(ctx, sqlc.InsertLostPlaceParams{LostReportID: id, Position: int16(i + 1), LocationID: pl.LocationID, ZoneID: pl.ZoneID}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) CreateLost(ctx context.Context, p *access.Principal, key string, in LostInput) (openapi.LostReport, error) {
	proxy := in.ProxyName != nil
	if proxy && !p.Role.Can(access.ReportProxyCreate) {
		return openapi.LostReport{}, apperr.Forbidden()
	}
	now := s.clock.Now()
	var id uuid.UUID
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		k := idem.Key{UserID: p.UserID, Key: key, Endpoint: "createLostReport"}
		existing, err := idem.Begin(ctx, q, k, in, now)
		if err != nil || existing != nil {
			if existing != nil {
				id = *existing
			}
			return err
		}
		if err := s.validateLost(ctx, q, &in, now, proxy); err != nil {
			return err
		}
		var reporter *uuid.UUID
		if !proxy {
			open, err := q.CountOpenLostByReporter(ctx, &p.UserID)
			if err != nil {
				return err
			}
			if open >= maxOpenLost {
				return apperr.Unprocessable("report_limit_reached", "Batas laporan tercapai", "Kamu sudah memiliki 5 laporan kehilangan yang masih terbuka.")
			}
			reporter = &p.UserID
		}
		code, err := nextCode(ctx, q, "LST", now)
		if err != nil {
			return err
		}
		id = ids.New()
		if err := q.InsertLostReport(ctx, sqlc.InsertLostReportParams{
			ID: id, Code: code, ReporterID: reporter, CreatedBy: &p.UserID, ProxyName: in.ProxyName, ProxyPhone: in.ProxyPhone,
			CategoryID: in.CategoryID, ItemName: in.ItemName, Description: in.Description, PrivateDetails: in.PrivateDetails,
			Colours: in.Colours, Brand: in.Brand, LostFrom: in.LostFrom, LostUntil: in.LostUntil,
			LocationUnknown: in.LocationUnknown, LocationNote: in.LocationNote, LastActivityAt: now, AssistOutcome: in.AssistOutcome,
		}); err != nil {
			return err
		}
		if err := s.replacePlaces(ctx, q, id, in.Places); err != nil {
			return err
		}
		if _, err := s.media.Attach(ctx, q, p, media.Target{Purpose: "lost_report", LostID: &id}, in.MediaIDs); err != nil {
			return err
		}
		tid, err := Lost.Step("", "submit", "reported")
		if err != nil {
			return err
		}
		if err := s.history(ctx, q, "lost_report", id, "", "reported", tid, nil); err != nil {
			return err
		}
		if err := audit.Record(ctx, q, now, audit.Entry{Action: "lost_report.create", EntityType: "lost_report", EntityID: id,
			Changes: map[string]any{"code": code, "proxy": proxy}}); err != nil {
			return err
		}
		if err := s.matching.ReportChanged(ctx, tx, "lost", id, in.MediaIDs); err != nil {
			return err
		}
		return idem.Complete(ctx, q, k, id)
	})
	if err != nil {
		return openapi.LostReport{}, err
	}
	return s.GetLost(ctx, p, id.String())
}

func (s *Service) loadLost(ctx context.Context, q *sqlc.Queries, ref string) (sqlc.GetLostReportRow, error) {
	id, code, err := refOf(ref)
	if err != nil {
		return sqlc.GetLostReportRow{}, err
	}
	row, err := q.GetLostReport(ctx, sqlc.GetLostReportParams{ID: id, Code: code})
	if db.IsNoRows(err) {
		return row, apperr.NotFound()
	}
	return row, err
}

func canReadLost(p *access.Principal, r sqlc.LostReport) bool {
	if r.HiddenAt != nil {
		return p.IsAdmin()
	}
	return p.IsStaff() || (r.ReporterID != nil && *r.ReporterID == p.UserID)
}

func isOwnerLost(p *access.Principal, r sqlc.LostReport) bool {
	return r.ReporterID != nil && *r.ReporterID == p.UserID
}

func lostEditable(r sqlc.LostReport) bool {
	return r.Status == "reported" || r.Status == "potential_match"
}

func (s *Service) GetLost(ctx context.Context, p *access.Principal, ref string) (openapi.LostReport, error) {
	row, err := s.loadLost(ctx, s.q, ref)
	if err != nil {
		return openapi.LostReport{}, err
	}
	if !canReadLost(p, row.LostReport) {
		return openapi.LostReport{}, apperr.NotFound()
	}
	places, err := s.q.ListLostPlaces(ctx, row.LostReport.ID)
	if err != nil {
		return openapi.LostReport{}, err
	}
	photos, err := s.media.Photos(ctx, s.q, &row.LostReport.ID, nil)
	if err != nil {
		return openapi.LostReport{}, err
	}
	owner := isOwnerLost(p, row.LostReport)
	if !owner && row.LostReport.PrivateDetails != nil {
		if err := audit.Record(ctx, s.q, s.clock.Now(), audit.Entry{Action: "private.read", EntityType: "lost_report", EntityID: row.LostReport.ID,
			Changes: map[string]any{"fields": []string{"private_details"}}}); err != nil {
			return openapi.LostReport{}, err
		}
	}
	return lostView(row, places, photos, p), nil
}

func (s *Service) UpdateLost(ctx context.Context, p *access.Principal, ref, ifMatch string, patch LostPatch) (openapi.LostReport, error) {
	version, err := parseVersion(ifMatch)
	if err != nil {
		return openapi.LostReport{}, err
	}
	now := s.clock.Now()
	var id uuid.UUID
	var removed media.Removed
	err = db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		row, err := s.loadLost(ctx, q, ref)
		if err != nil {
			return err
		}
		r := row.LostReport
		id = r.ID
		if !canReadLost(p, r) || (!isOwnerLost(p, r) && !p.IsStaff()) {
			return apperr.NotFound()
		}
		if !lostEditable(r) {
			return apperr.InvalidTransition(r.Status)
		}
		if r.Version != version {
			return apperr.StaleVersion()
		}
		places, err := q.ListLostPlaces(ctx, r.ID)
		if err != nil {
			return err
		}
		in := lostInputFrom(r, places)
		changed := patch.apply(&in)
		if err := s.validateLost(ctx, q, &in, now, false); err != nil {
			return err
		}
		if _, err := q.UpdateLostReport(ctx, sqlc.UpdateLostReportParams{
			CategoryID: in.CategoryID, ItemName: in.ItemName, Description: in.Description, PrivateDetails: in.PrivateDetails,
			Colours: in.Colours, Brand: in.Brand, LostFrom: in.LostFrom, LostUntil: in.LostUntil,
			LocationUnknown: in.LocationUnknown, LocationNote: in.LocationNote, Now: now, ID: r.ID, Version: version,
		}); db.IsNoRows(err) {
			return apperr.StaleVersion()
		} else if err != nil {
			return err
		}
		if patch.Places != nil || patch.LocationUnknown != nil {
			if err := s.replacePlaces(ctx, q, r.ID, in.Places); err != nil {
				return err
			}
		}
		var added []uuid.UUID
		if patch.MediaIDs != nil {
			before, err := s.media.Photos(ctx, q, &r.ID, nil)
			if err != nil {
				return err
			}
			added = addedMedia(before, *patch.MediaIDs)
			if removed, err = s.media.Attach(ctx, q, p, media.Target{Purpose: "lost_report", LostID: &r.ID}, *patch.MediaIDs); err != nil {
				return err
			}
		}
		if err := s.matching.ReportChanged(ctx, tx, "lost", r.ID, added); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "lost_report.update", EntityType: "lost_report", EntityID: r.ID,
			Changes: map[string]any{"fields": changed}})
	})
	if err != nil {
		return openapi.LostReport{}, err
	}
	s.media.DeleteObjects(ctx, removed)
	return s.GetLost(ctx, p, id.String())
}

func lostInputFrom(r sqlc.LostReport, places []sqlc.ListLostPlacesRow) LostInput {
	in := LostInput{
		CategoryID: r.CategoryID, ItemName: r.ItemName, Description: r.Description, PrivateDetails: r.PrivateDetails,
		Colours: r.Colours, Brand: r.Brand, LostFrom: r.LostFrom, LostUntil: r.LostUntil,
		LocationUnknown: r.LocationUnknown, LocationNote: r.LocationNote,
	}
	for _, pl := range places {
		if pl.ZoneOnly {
			z := pl.ZoneID
			in.Places = append(in.Places, Place{ZoneID: &z})
		} else {
			in.Places = append(in.Places, Place{LocationID: pl.LocationID})
		}
	}
	return in
}

// apply copies present fields into the input and returns their names for the audit entry
func (p LostPatch) apply(in *LostInput) []string {
	var changed []string
	set := func(name string, present bool) {
		if present {
			changed = append(changed, name)
		}
	}
	if p.CategoryID != nil {
		in.CategoryID = *p.CategoryID
	}
	set("category_id", p.CategoryID != nil)
	if p.ItemName != nil {
		in.ItemName = *p.ItemName
	}
	set("item_name", p.ItemName != nil)
	if p.Description != nil {
		in.Description = *p.Description
	}
	set("description", p.Description != nil)
	if p.PrivateDetails != nil {
		in.PrivateDetails = p.PrivateDetails
	}
	set("private_details", p.PrivateDetails != nil)
	if p.Colours != nil {
		in.Colours = *p.Colours
	}
	set("colours", p.Colours != nil)
	if p.Brand != nil {
		in.Brand = p.Brand
	}
	set("brand", p.Brand != nil)
	if p.LostFrom != nil {
		in.LostFrom = *p.LostFrom
	}
	set("lost_from", p.LostFrom != nil)
	if p.LostUntil != nil {
		in.LostUntil = *p.LostUntil
	}
	set("lost_until", p.LostUntil != nil)
	if p.LocationUnknown != nil {
		in.LocationUnknown = *p.LocationUnknown
	}
	set("location_unknown", p.LocationUnknown != nil)
	if p.Places != nil {
		in.Places = *p.Places
	}
	set("places", p.Places != nil)
	if p.LocationNote != nil {
		in.LocationNote = p.LocationNote
	}
	set("location_note", p.LocationNote != nil)
	set("media_ids", p.MediaIDs != nil)
	return changed
}

func (s *Service) CloseLost(ctx context.Context, p *access.Principal, ref, reason string, note *string) (openapi.LostReport, error) {
	if reason != "found_by_owner" && reason != "no_longer_needed" {
		return openapi.LostReport{}, apperr.Validation(apperr.FieldError{Field: "reason", Code: "invalid", Message: "Alasan tidak valid"})
	}
	now := s.clock.Now()
	var id uuid.UUID
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		row, err := s.loadLost(ctx, q, ref)
		if err != nil {
			return err
		}
		r := row.LostReport
		id = r.ID
		if !canReadLost(p, r) || (!isOwnerLost(p, r) && !p.IsAdmin()) {
			return apperr.NotFound()
		}
		tid, err := Lost.Step(r.Status, "owner_close", "closed")
		if err != nil {
			return err
		}
		if _, err := q.SetLostStatus(ctx, sqlc.SetLostStatusParams{
			Status: "closed", CloseReason: &reason, ClosedAt: &now, Now: now, ID: r.ID, Version: r.Version,
		}); db.IsNoRows(err) {
			return apperr.StaleVersion()
		} else if err != nil {
			return err
		}
		why := reason
		if n := textx.Opt(note); n != nil {
			why = reason + ": " + *n
		}
		if err := s.history(ctx, q, "lost_report", r.ID, r.Status, "closed", tid, &why); err != nil {
			return err
		}
		if err := s.matching.ReportClosed(ctx, tx, "lost", r.ID); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "lost_report.close", EntityType: "lost_report", EntityID: r.ID,
			Changes: map[string]any{"status": audit.Change(r.Status, "closed"), "close_reason": reason}})
	})
	if err != nil {
		return openapi.LostReport{}, err
	}
	return s.GetLost(ctx, p, id.String())
}

type ListInput struct {
	Mine     bool
	Statuses []string
	Cursor   *string
	Limit    int
}

func (s *Service) ListLost(ctx context.Context, p *access.Principal, in ListInput, cur *ListCursor) ([]sqlc.ListLostReportsRow, error) {
	if err := validStatuses(in.Statuses, lostStatuses); err != nil {
		return nil, err
	}
	params := sqlc.ListLostReportsParams{Statuses: nonNil(in.Statuses), RowLimit: int32(in.Limit + 1)}
	if !p.IsStaff() || in.Mine {
		params.ReporterID = &p.UserID
	}
	if cur != nil {
		params.CursorCreated, params.CursorID = &cur.At, &cur.ID
	}
	return s.q.ListLostReports(ctx, params)
}

type ListCursor struct {
	At time.Time
	ID uuid.UUID
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return slices.Clone(s)
}

func (s *Service) LostHistory(ctx context.Context, p *access.Principal, ref string) ([]sqlc.ListStatusHistoryRow, error) {
	row, err := s.loadLost(ctx, s.q, ref)
	if err != nil {
		return nil, err
	}
	if !canReadLost(p, row.LostReport) {
		return nil, apperr.NotFound()
	}
	return s.q.ListStatusHistory(ctx, sqlc.ListStatusHistoryParams{EntityID: row.LostReport.ID, EntityTypes: []string{"lost_report"}})
}
