package reports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/custody"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/media"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/idem"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/textx"
	"github.com/pisondev/simpanugm/api/internal/privacy"
)

type ProxyFound struct {
	FinderName    *string   `json:"finder_name"`
	PostID        uuid.UUID `json:"post_id"`
	StorageSlot   string    `json:"storage_slot"`
	ConditionNote *string   `json:"condition_note"`
}

type FoundInput struct {
	CategoryID      uuid.UUID   `json:"category_id"`
	ItemName        string      `json:"item_name"`
	Description     string      `json:"description"`
	HiddenDetails   *string     `json:"hidden_details"`
	Colours         []string    `json:"colours"`
	Brand           *string     `json:"brand"`
	FoundAt         time.Time   `json:"found_at"`
	FoundLocationID uuid.UUID   `json:"found_location_id"`
	LocationNote    *string     `json:"location_note"`
	MediaIDs        []uuid.UUID `json:"media_ids"`
	CustodyMode     string      `json:"custody_mode"`
	IntendedPostID  *uuid.UUID  `json:"intended_post_id"`
	Proxy           *ProxyFound `json:"proxy"`
}

type FoundPatch struct {
	CategoryID      *uuid.UUID
	ItemName        *string
	Description     *string
	HiddenDetails   *string
	Colours         *[]string
	Brand           *string
	FoundAt         *time.Time
	FoundLocationID *uuid.UUID
	LocationNote    *string
	MediaIDs        *[]uuid.UUID
	CustodyMode     *string
	IntendedPostID  *uuid.UUID
}

func (in *FoundInput) normalize() {
	in.ItemName = textx.Value(textx.OptString(in.ItemName))
	in.Description = privacy.MaskNumbers(textx.Value(textx.OptString(in.Description)))
	in.HiddenDetails = textx.Opt(in.HiddenDetails)
	in.Brand = textx.Opt(in.Brand)
	in.LocationNote = textx.Opt(in.LocationNote)
	if in.Colours == nil {
		in.Colours = []string{}
	}
	if in.MediaIDs == nil {
		in.MediaIDs = []uuid.UUID{}
	}
}

func (s *Service) validateFound(ctx context.Context, q *sqlc.Queries, p *access.Principal, in *FoundInput, now time.Time) (masterdata.LeafCategory, error) {
	var v apperr.Collector
	in.normalize()
	checkText(&v, textRule{"item_name", textx.OptString(in.ItemName), 3, 80, "Nama barang"})
	checkText(&v, textRule{"description", textx.OptString(in.Description), 10, 500, "Deskripsi"})
	checkText(&v, textRule{"hidden_details", in.HiddenDetails, 0, 500, "Detail tersembunyi"})
	checkText(&v, textRule{"brand", in.Brand, 0, 40, "Merek"})
	checkText(&v, textRule{"location_note", in.LocationNote, 0, 200, "Catatan lokasi"})
	if err := s.checkColours(ctx, q, &v, in.Colours); err != nil {
		return masterdata.LeafCategory{}, err
	}
	cat, err := s.leaf(ctx, q, &v, in.CategoryID)
	if err != nil {
		return cat, err
	}
	if in.FoundAt.After(now.Add(futureTolerance)) {
		v.Add("found_at", "in_future", "Waktu tidak boleh di masa depan")
	}
	if in.FoundAt.Before(now.Add(-foundLookback)) {
		v.Add("found_at", "too_old", "Waktu penemuan maksimal 30 hari yang lalu")
	}
	l, err := q.GetLocation(ctx, in.FoundLocationID)
	if db.IsNoRows(err) || (err == nil && !l.Active) {
		v.Add("found_location_id", "unknown_location", "Lokasi tidak ditemukan")
	} else if err != nil {
		return cat, err
	}
	switch n := len(in.MediaIDs); {
	case n == 0:
		v.Add("media_ids", "required", "Tambahkan minimal satu foto barang")
	case n > media.MaxPerReport:
		v.Add("media_ids", "too_many", "Maksimal 5 foto per laporan")
	}
	if in.Proxy != nil {
		checkText(&v, textRule{"proxy.finder_name", textx.Opt(in.Proxy.FinderName), 0, 100, "Nama penemu"})
		if err := custody.ValidateSlot(in.Proxy.StorageSlot); err != nil {
			v.Add("proxy.storage_slot", "invalid_length", "Lokasi simpan 1 sampai 40 karakter")
		}
	} else {
		switch in.CustodyMode {
		case "keep":
			if !v.Has("category_id") && cat.SensitivityValue() != "normal" {
				v.Add("custody.mode", "must_handover", "Barang berharga dan kartu identitas wajib diserahkan ke pos")
			}
			in.IntendedPostID = nil
		case "handover":
			if in.IntendedPostID == nil {
				v.Add("custody.intended_post_id", "required", "Pilih pos tujuan penyerahan")
			} else if post, err := q.GetPost(ctx, *in.IntendedPostID); db.IsNoRows(err) || (err == nil && !post.Active) {
				v.Add("custody.intended_post_id", "unknown_post", "Pos tidak ditemukan atau tidak aktif")
			} else if err != nil {
				return cat, err
			}
		default:
			v.Add("custody.mode", "invalid", "Pilih cara penyerahan barang")
		}
	}
	return cat, v.Err()
}

func (s *Service) CreateFound(ctx context.Context, p *access.Principal, key string, in FoundInput) (openapi.FoundReport, error) {
	proxy := in.Proxy != nil
	if proxy && !p.Role.Can(access.ReportProxyCreate) {
		return openapi.FoundReport{}, apperr.Forbidden()
	}
	now := s.clock.Now()
	var id uuid.UUID
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		k := idem.Key{UserID: p.UserID, Key: key, Endpoint: "createFoundReport"}
		existing, err := idem.Begin(ctx, q, k, in, now)
		if err != nil || existing != nil {
			if existing != nil {
				id = *existing
			}
			return err
		}
		cat, err := s.validateFound(ctx, q, p, &in, now)
		if err != nil {
			return err
		}
		params := sqlc.InsertFoundReportParams{
			Code: "", CreatedBy: &p.UserID, CategoryID: in.CategoryID, ItemName: in.ItemName, Description: in.Description,
			HiddenDetails: in.HiddenDetails, Colours: in.Colours, Brand: in.Brand, FoundAt: in.FoundAt,
			FoundLocationID: in.FoundLocationID, LocationNote: in.LocationNote, CreatedAt: now,
		}
		custodyEvent, custodyTo := "member_submit", "with_finder"
		if proxy {
			if _, err := s.custody.CheckPost(ctx, q, p, in.Proxy.PostID); err != nil {
				return err
			}
			slot := in.Proxy.StorageSlot
			due := s.custody.RetentionDue(ctx, now, cat.SensitivityValue(), cat.RetentionDays)
			params.ProxyFinderName = textx.Opt(in.Proxy.FinderName)
			params.CustodyStatus = "in_custody"
			params.HoldingPostID, params.StorageSlot, params.ReceivedAt = &in.Proxy.PostID, &slot, &now
			params.ReceivedBy, params.RetentionDueAt = &p.UserID, &due
			custodyEvent, custodyTo = "proxy_submit", "in_custody"
		} else {
			since := now.Add(-7 * 24 * time.Hour)
			n, err := q.CountFoundByFinderSince(ctx, sqlc.CountFoundByFinderSinceParams{FinderID: &p.UserID, CreatedAt: since})
			if err != nil {
				return err
			}
			if n >= maxFoundPerWeek {
				return apperr.Unprocessable("report_limit_reached", "Batas laporan tercapai", "Kamu sudah membuat 10 laporan temuan dalam 7 hari terakhir.")
			}
			params.FinderID = &p.UserID
			params.CustodyStatus = "with_finder"
			params.CustodyKeep = in.CustodyMode == "keep"
			params.IntendedPostID = in.IntendedPostID
		}
		if params.Code, err = nextCode(ctx, q, "FND", now); err != nil {
			return err
		}
		id = ids.New()
		params.ID = id
		if err := q.InsertFoundReport(ctx, params); err != nil {
			return err
		}
		if _, err := s.media.Attach(ctx, q, p, media.Target{Purpose: "found_report", FoundID: &id}, in.MediaIDs); err != nil {
			return err
		}
		tid, err := Found.Step("", "submit", "reported")
		if err != nil {
			return err
		}
		if err := s.history(ctx, q, "found_report", id, "", "reported", tid, nil); err != nil {
			return err
		}
		cid, err := Custody.Step("", custodyEvent, custodyTo)
		if err != nil {
			return err
		}
		if err := s.history(ctx, q, "custody", id, "", custodyTo, cid, nil); err != nil {
			return err
		}
		if proxy {
			if err := s.custody.Event(ctx, q, id, "received", nil, &in.Proxy.PostID, p.UserID, in.Proxy.ConditionNote); err != nil {
				return err
			}
		}
		if err := audit.Record(ctx, q, now, audit.Entry{Action: "found_report.create", EntityType: "found_report", EntityID: id,
			Changes: map[string]any{"code": params.Code, "proxy": proxy, "custody_status": custodyTo}}); err != nil {
			return err
		}
		if err := s.matching.ReportChanged(ctx, tx, "found", id, in.MediaIDs); err != nil {
			return err
		}
		return idem.Complete(ctx, q, k, id)
	})
	if err != nil {
		return openapi.FoundReport{}, err
	}
	return s.GetFound(ctx, p, id.String())
}

func (s *Service) loadFound(ctx context.Context, q *sqlc.Queries, ref string) (sqlc.GetFoundReportRow, error) {
	id, code, err := refOf(ref)
	if err != nil {
		return sqlc.GetFoundReportRow{}, err
	}
	row, err := q.GetFoundReport(ctx, sqlc.GetFoundReportParams{ID: id, Code: code})
	if db.IsNoRows(err) {
		return row, apperr.NotFound()
	}
	return row, err
}

func isFinder(p *access.Principal, r sqlc.FoundReport) bool {
	return r.FinderID != nil && *r.FinderID == p.UserID
}

func canReadFound(p *access.Principal, r sqlc.FoundReport) bool {
	if r.HiddenAt != nil {
		return p.IsAdmin()
	}
	return p.IsStaff() || isFinder(p, r)
}

func foundEditable(p *access.Principal, r sqlc.FoundReport) bool {
	if p.IsStaff() {
		return r.Status != "returned" && r.Status != "closed"
	}
	return isFinder(p, r) && r.CustodyStatus == "with_finder" && (r.Status == "reported" || r.Status == "potential_match")
}

func (s *Service) GetFound(ctx context.Context, p *access.Principal, ref string) (openapi.FoundReport, error) {
	row, err := s.loadFound(ctx, s.q, ref)
	if err != nil {
		return openapi.FoundReport{}, err
	}
	if !canReadFound(p, row.FoundReport) {
		return openapi.FoundReport{}, apperr.NotFound()
	}
	photos, err := s.media.Photos(ctx, s.q, nil, &row.FoundReport.ID)
	if err != nil {
		return openapi.FoundReport{}, err
	}
	if !isFinder(p, row.FoundReport) && row.FoundReport.HiddenDetails != nil {
		if err := audit.Record(ctx, s.q, s.clock.Now(), audit.Entry{Action: "private.read", EntityType: "found_report", EntityID: row.FoundReport.ID,
			Changes: map[string]any{"fields": []string{"hidden_details"}}}); err != nil {
			return openapi.FoundReport{}, err
		}
	}
	return foundView(row, photos, p, foundEditable(p, row.FoundReport)), nil
}

func (s *Service) UpdateFound(ctx context.Context, p *access.Principal, ref, ifMatch string, patch FoundPatch) (openapi.FoundReport, error) {
	version, err := parseVersion(ifMatch)
	if err != nil {
		return openapi.FoundReport{}, err
	}
	now := s.clock.Now()
	var id uuid.UUID
	var removed media.Removed
	err = db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		row, err := s.loadFound(ctx, q, ref)
		if err != nil {
			return err
		}
		r := row.FoundReport
		id = r.ID
		if !canReadFound(p, r) {
			return apperr.NotFound()
		}
		if !foundEditable(p, r) {
			return apperr.InvalidTransition(r.Status)
		}
		if r.Version != version {
			return apperr.StaleVersion()
		}
		photos, err := s.media.Photos(ctx, q, nil, &r.ID)
		if err != nil {
			return err
		}
		in := foundInputFrom(r, photos)
		changed := patch.apply(&in)
		if patch.CustodyMode != nil && r.CustodyStatus != "with_finder" {
			return apperr.Unprocessable("custody_locked", "Penyerahan sudah tercatat", "Cara penyerahan hanya bisa diubah selama barang masih dibawa penemu.")
		}
		if r.CustodyStatus != "with_finder" {
			in.CustodyMode = "handover"
			if in.IntendedPostID == nil {
				in.IntendedPostID = r.HoldingPostID
			}
		}
		if _, err := s.validateFound(ctx, q, p, &in, now); err != nil {
			return err
		}
		if _, err := q.UpdateFoundReport(ctx, sqlc.UpdateFoundReportParams{
			CategoryID: in.CategoryID, ItemName: in.ItemName, Description: in.Description, HiddenDetails: in.HiddenDetails,
			Colours: in.Colours, Brand: in.Brand, FoundAt: in.FoundAt, FoundLocationID: in.FoundLocationID,
			LocationNote: in.LocationNote, CustodyKeep: in.CustodyMode == "keep" && r.CustodyStatus == "with_finder",
			IntendedPostID: in.IntendedPostID, Now: now, ID: r.ID, Version: version,
		}); db.IsNoRows(err) {
			return apperr.StaleVersion()
		} else if err != nil {
			return err
		}
		var added []uuid.UUID
		if patch.MediaIDs != nil {
			added = addedMedia(photos, *patch.MediaIDs)
			if removed, err = s.media.Attach(ctx, q, p, media.Target{Purpose: "found_report", FoundID: &r.ID}, *patch.MediaIDs); err != nil {
				return err
			}
		}
		if err := s.matching.ReportChanged(ctx, tx, "found", r.ID, added); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "found_report.update", EntityType: "found_report", EntityID: r.ID,
			Changes: map[string]any{"fields": changed}})
	})
	if err != nil {
		return openapi.FoundReport{}, err
	}
	s.media.DeleteObjects(ctx, removed)
	return s.GetFound(ctx, p, id.String())
}

func foundInputFrom(r sqlc.FoundReport, photos []sqlc.Medium) FoundInput {
	in := FoundInput{
		CategoryID: r.CategoryID, ItemName: r.ItemName, Description: r.Description, HiddenDetails: r.HiddenDetails,
		Colours: r.Colours, Brand: r.Brand, FoundAt: r.FoundAt, FoundLocationID: r.FoundLocationID,
		LocationNote: r.LocationNote, IntendedPostID: r.IntendedPostID, CustodyMode: "handover",
	}
	if r.CustodyKeep {
		in.CustodyMode = "keep"
	}
	for _, m := range photos {
		in.MediaIDs = append(in.MediaIDs, m.ID)
	}
	return in
}

func (p FoundPatch) apply(in *FoundInput) []string {
	var changed []string
	mark := func(name string, present bool) {
		if present {
			changed = append(changed, name)
		}
	}
	if p.CategoryID != nil {
		in.CategoryID = *p.CategoryID
	}
	mark("category_id", p.CategoryID != nil)
	if p.ItemName != nil {
		in.ItemName = *p.ItemName
	}
	mark("item_name", p.ItemName != nil)
	if p.Description != nil {
		in.Description = *p.Description
	}
	mark("description", p.Description != nil)
	if p.HiddenDetails != nil {
		in.HiddenDetails = p.HiddenDetails
	}
	mark("hidden_details", p.HiddenDetails != nil)
	if p.Colours != nil {
		in.Colours = *p.Colours
	}
	mark("colours", p.Colours != nil)
	if p.Brand != nil {
		in.Brand = p.Brand
	}
	mark("brand", p.Brand != nil)
	if p.FoundAt != nil {
		in.FoundAt = *p.FoundAt
	}
	mark("found_at", p.FoundAt != nil)
	if p.FoundLocationID != nil {
		in.FoundLocationID = *p.FoundLocationID
	}
	mark("found_location_id", p.FoundLocationID != nil)
	if p.LocationNote != nil {
		in.LocationNote = p.LocationNote
	}
	mark("location_note", p.LocationNote != nil)
	if p.MediaIDs != nil {
		in.MediaIDs = *p.MediaIDs
	}
	mark("media_ids", p.MediaIDs != nil)
	if p.CustodyMode != nil {
		in.CustodyMode = *p.CustodyMode
		in.IntendedPostID = p.IntendedPostID
	}
	mark("custody", p.CustodyMode != nil)
	return changed
}

func (s *Service) CloseFound(ctx context.Context, p *access.Principal, ref, reason, note string) (openapi.FoundReport, error) {
	var custodyEvent, custodyTo string
	switch reason {
	case "returned_directly":
		custodyEvent, custodyTo = "returned_directly", "released"
	case "reported_in_error":
		custodyEvent, custodyTo = "cancel", "cancelled"
	default:
		return openapi.FoundReport{}, apperr.Validation(apperr.FieldError{Field: "reason", Code: "invalid", Message: "Alasan tidak valid"})
	}
	clean := textx.Opt(&note)
	if clean == nil || textx.Len(*clean) < closeNoteMinChars || textx.Len(*clean) > 500 {
		return openapi.FoundReport{}, apperr.Validation(apperr.FieldError{Field: "note", Code: "invalid_length", Message: "Catatan 5 sampai 500 karakter"})
	}
	now := s.clock.Now()
	var id uuid.UUID
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		row, err := s.loadFound(ctx, q, ref)
		if err != nil {
			return err
		}
		r := row.FoundReport
		id = r.ID
		if !canReadFound(p, r) || (!isFinder(p, r) && !p.IsAdmin()) {
			return apperr.NotFound()
		}
		tid, err := Found.Step(r.Status, "finder_close", "closed")
		if err != nil {
			return err
		}
		cid, err := Custody.Step(r.CustodyStatus, custodyEvent, custodyTo)
		if err != nil {
			return err
		}
		if _, err := q.CloseFoundReport(ctx, sqlc.CloseFoundReportParams{
			CloseReason: &reason, CloseNote: clean, Now: now, CustodyStatus: custodyTo, ID: r.ID, Version: r.Version,
		}); db.IsNoRows(err) {
			return apperr.StaleVersion()
		} else if err != nil {
			return err
		}
		if err := s.history(ctx, q, "found_report", r.ID, r.Status, "closed", tid, &reason); err != nil {
			return err
		}
		if err := s.history(ctx, q, "custody", r.ID, r.CustodyStatus, custodyTo, cid, &reason); err != nil {
			return err
		}
		if err := s.matching.ReportClosed(ctx, tx, "found", r.ID); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "found_report.close", EntityType: "found_report", EntityID: r.ID,
			Changes: map[string]any{"status": audit.Change(r.Status, "closed"), "custody_status": audit.Change(r.CustodyStatus, custodyTo), "close_reason": reason}})
	})
	if err != nil {
		return openapi.FoundReport{}, err
	}
	return s.GetFound(ctx, p, id.String())
}

func (s *Service) Receive(ctx context.Context, p *access.Principal, ref string, post uuid.UUID, slot string, note *string) (openapi.FoundReport, error) {
	now := s.clock.Now()
	var id uuid.UUID
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		row, err := s.loadFound(ctx, q, ref)
		if err != nil {
			return err
		}
		r := row.FoundReport
		id = r.ID
		if !canReadFound(p, r) {
			return apperr.NotFound()
		}
		if r.Status == "closed" || r.Status == "returned" {
			return apperr.InvalidTransition(r.Status)
		}
		cid, err := Custody.Step(r.CustodyStatus, "receive", "in_custody")
		if err != nil {
			return err
		}
		cat, _, err := s.md.Leaf(ctx, q, r.CategoryID)
		if err != nil {
			return err
		}
		due, err := s.custody.Receive(ctx, q, p, custody.ReceiveInput{
			FoundID: r.ID, Version: r.Version, PostID: post, StorageSlot: slot, ConditionNote: note,
			Sensitivity: cat.SensitivityValue(), RetentionDays: cat.RetentionDays,
		})
		if err != nil {
			return err
		}
		if err := s.history(ctx, q, "custody", r.ID, r.CustodyStatus, "in_custody", cid, nil); err != nil {
			return err
		}
		// US-30 criterion 2, the finder learns the item reached a post
		if r.FinderID != nil {
			postRow, err := q.GetPost(ctx, post)
			if err != nil {
				return err
			}
			if err := s.notifier.Send(ctx, tx, notify.Message{
				Event: notify.FoundReceived, To: *r.FinderID, Link: "/laporan/temuan/" + r.Code,
				Data:       map[string]string{"item_name": r.ItemName, "post_name": postRow.Name},
				EntityType: "found_report", EntityID: &r.ID,
			}); err != nil {
				return err
			}
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "custody.receive", EntityType: "found_report", EntityID: r.ID,
			Changes: map[string]any{"custody_status": audit.Change(r.CustodyStatus, "in_custody"), "post_id": post, "retention_due_at": due}})
	})
	if err != nil {
		return openapi.FoundReport{}, err
	}
	return s.GetFound(ctx, p, id.String())
}

func (s *Service) ListFound(ctx context.Context, p *access.Principal, in ListInput, cur *ListCursor) ([]sqlc.ListFoundReportsRow, error) {
	if err := validStatuses(in.Statuses, foundStatuses); err != nil {
		return nil, err
	}
	params := sqlc.ListFoundReportsParams{Statuses: nonNil(in.Statuses), RowLimit: int32(in.Limit + 1)}
	if !p.IsStaff() || in.Mine {
		params.FinderID = &p.UserID
	}
	if cur != nil {
		params.CursorCreated, params.CursorID = &cur.At, &cur.ID
	}
	return s.q.ListFoundReports(ctx, params)
}

type PublicFilter struct {
	CategoryID *uuid.UUID
	ZoneID     *uuid.UUID
	From, To   *time.Time
	Limit      int
}

func (s *Service) ListPublicFound(ctx context.Context, f PublicFilter, cur *ListCursor) ([]sqlc.ListPublicFoundReportsRow, error) {
	params := sqlc.ListPublicFoundReportsParams{CategoryID: f.CategoryID, ZoneID: f.ZoneID, FromTime: f.From, ToTime: f.To, RowLimit: int32(f.Limit + 1)}
	if cur != nil {
		params.CursorFound, params.CursorID = &cur.At, &cur.ID
	}
	return s.q.ListPublicFoundReports(ctx, params)
}

func (s *Service) FoundHistory(ctx context.Context, p *access.Principal, ref string) ([]sqlc.ListStatusHistoryRow, error) {
	row, err := s.loadFound(ctx, s.q, ref)
	if err != nil {
		return nil, err
	}
	if !canReadFound(p, row.FoundReport) {
		return nil, apperr.NotFound()
	}
	return s.q.ListStatusHistory(ctx, sqlc.ListStatusHistoryParams{EntityID: row.FoundReport.ID, EntityTypes: []string{"found_report", "custody"}})
}
