package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/httpx"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/ratelimit"
	"github.com/pisondev/simpanugm/api/internal/platform/storage"
)

const (
	MaxPerReport = 5
	stagedTTL    = 24 * time.Hour
)

type Service struct {
	pool    *pgxpool.Pool
	q       *sqlc.Queries
	store   storage.Store
	clock   clock.Clock
	sem     chan struct{}
	uploads *ratelimit.Limiter
}

func NewService(pool *pgxpool.Pool, store storage.Store, c clock.Clock) *Service {
	return &Service{
		pool: pool, q: sqlc.New(pool), store: store, clock: c,
		sem:     make(chan struct{}, 4),
		uploads: ratelimit.New(30, time.Hour, c.Now),
	}
}

type Keys map[string]string

func (s *Service) Upload(ctx context.Context, p *access.Principal, purpose string, r io.Reader) (sqlc.Medium, error) {
	if purpose != "lost_report" && purpose != "found_report" && purpose != "claim_evidence" {
		return sqlc.Medium{}, apperr.Validation(apperr.FieldError{Field: "purpose", Code: "invalid", Message: "Tujuan unggahan tidak valid"})
	}
	if ok, wait := s.uploads.Allow(p.UserID.String()); !ok {
		return sqlc.Medium{}, apperr.RateLimited(wait)
	}
	data, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return sqlc.Medium{}, apperr.BadRequest("Gagal membaca berkas.").WithCause(err)
	}
	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		return sqlc.Medium{}, ctx.Err()
	}
	processed, err := Process(data)
	<-s.sem
	if err != nil {
		return sqlc.Medium{}, err
	}
	now := s.clock.Now()
	id := ids.New()
	prefix := fmt.Sprintf("media/%04d/%02d/%s", now.Year(), int(now.Month()), id)
	keys := Keys{}
	for _, v := range processed.Variants {
		key := prefix + "/" + v.Name + "." + v.Ext
		if err := s.store.Put(ctx, key, bytes.NewReader(v.Data)); err != nil {
			s.removeKeys(ctx, keys)
			return sqlc.Medium{}, err
		}
		keys[v.Name] = key
	}
	raw, _ := json.Marshal(keys)
	params := sqlc.InsertMediaParams{
		ID: id, UploadedBy: p.UserID, Purpose: purpose, Mime: processed.Mime,
		Width: int32(processed.Width), Height: int32(processed.Height), Bytes: int32(processed.Bytes),
		Sha256: processed.SHA256, Phash: processed.PHash, StorageKeys: raw, CreatedAt: now,
	}
	if err := s.q.InsertMedia(ctx, params); err != nil {
		s.removeKeys(ctx, keys)
		return sqlc.Medium{}, err
	}
	return sqlc.Medium{
		ID: id, UploadedBy: p.UserID, Purpose: purpose, State: "staged", Mime: processed.Mime,
		Width: params.Width, Height: params.Height, Bytes: params.Bytes, CreatedAt: now,
	}, nil
}

func (s *Service) removeKeys(ctx context.Context, keys Keys) {
	for _, k := range keys {
		if err := s.store.Delete(ctx, k); err != nil {
			slog.WarnContext(ctx, "media object cleanup failed", "key", k, "error", err.Error())
		}
	}
}

// storage objects of detached media, deleted only after the transaction commits
type Removed [][]byte

func (s *Service) DeleteObjects(ctx context.Context, removed Removed) {
	for _, raw := range removed {
		var keys Keys
		if json.Unmarshal(raw, &keys) == nil {
			s.removeKeys(ctx, keys)
		}
	}
}

type Target struct {
	Purpose string
	LostID  *uuid.UUID
	FoundID *uuid.UUID
	ClaimID *uuid.UUID
}

// ids become the exact ordered photo set of the report
func (s *Service) Attach(ctx context.Context, q *sqlc.Queries, p *access.Principal, t Target, ids []uuid.UUID) (Removed, error) {
	if len(ids) > MaxPerReport {
		return nil, apperr.Unprocessable("media_limit_reached", "Terlalu banyak foto", "Maksimal 5 foto per laporan.")
	}
	if len(slices.Compact(slices.Clone(sortedIDs(ids)))) != len(ids) {
		return nil, apperr.Unprocessable("media_duplicate", "Foto ganda", "Foto ini sudah ditambahkan")
	}
	current, err := q.ListReportMedia(ctx, sqlc.ListReportMediaParams{LostReportID: t.LostID, FoundReportID: t.FoundID, ClaimID: t.ClaimID})
	if err != nil {
		return nil, err
	}
	locked, err := q.LockMedia(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]sqlc.Medium, len(locked))
	for _, m := range locked {
		byID[m.ID] = m
	}
	belongs := func(m sqlc.Medium) bool {
		return (t.LostID != nil && m.LostReportID != nil && *m.LostReportID == *t.LostID) ||
			(t.FoundID != nil && m.FoundReportID != nil && *m.FoundReportID == *t.FoundID) ||
			(t.ClaimID != nil && m.ClaimID != nil && *m.ClaimID == *t.ClaimID)
	}
	for _, id := range ids {
		m, ok := byID[id]
		usable := ok && m.Purpose == t.Purpose &&
			((m.State == "staged" && m.UploadedBy == p.UserID) || (m.State == "ready" && belongs(m)))
		if !usable {
			return nil, apperr.Validation(apperr.FieldError{Field: "media_ids", Code: "unknown_media", Message: "Foto tidak ditemukan atau sudah dipakai"})
		}
	}
	now := s.clock.Now()
	var removed Removed
	for _, m := range current {
		if !slices.Contains(ids, m.ID) {
			if err := q.MarkMediaDeleted(ctx, sqlc.MarkMediaDeletedParams{ID: m.ID, DeletedAt: &now}); err != nil {
				return nil, err
			}
			removed = append(removed, m.StorageKeys)
		}
	}
	for i, id := range ids {
		err := q.AttachMedia(ctx, sqlc.AttachMediaParams{LostReportID: t.LostID, FoundReportID: t.FoundID, ClaimID: t.ClaimID, Position: int16(i), Now: now, ID: id})
		if db.IsUniqueViolation(err, "") {
			return nil, apperr.Unprocessable("media_duplicate", "Foto ganda", "Foto ini sudah ditambahkan")
		}
		if err != nil {
			return nil, err
		}
	}
	return removed, nil
}

func sortedIDs(in []uuid.UUID) []uuid.UUID {
	out := slices.Clone(in)
	slices.SortFunc(out, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	return out
}

func (s *Service) Photos(ctx context.Context, q *sqlc.Queries, lost, found *uuid.UUID) ([]sqlc.Medium, error) {
	return q.ListReportMedia(ctx, sqlc.ListReportMediaParams{LostReportID: lost, FoundReportID: found})
}

// the evidence images of a claim
func (s *Service) Evidence(ctx context.Context, q *sqlc.Queries, claim uuid.UUID) ([]sqlc.Medium, error) {
	return q.ListReportMedia(ctx, sqlc.ListReportMediaParams{ClaimID: &claim})
}

func (s *Service) DeleteStaged(ctx context.Context, p *access.Principal, id uuid.UUID) error {
	row, err := s.q.GetMediaForAccess(ctx, sqlc.GetMediaForAccessParams{ID: id, Viewer: p.UserID})
	if db.IsNoRows(err) || (err == nil && (row.UploadedBy != p.UserID || row.State == "deleted")) {
		return apperr.NotFound()
	}
	if err != nil {
		return err
	}
	if row.State != "staged" {
		return apperr.Unprocessable("media_attached", "Foto sudah terpasang", "Hapus foto dari laporan dengan mengubah laporannya.")
	}
	now := s.clock.Now()
	if err := s.q.MarkMediaDeleted(ctx, sqlc.MarkMediaDeletedParams{ID: id, DeletedAt: &now}); err != nil {
		return err
	}
	s.DeleteObjects(ctx, Removed{row.StorageKeys})
	return nil
}

var variants = map[string]bool{"display": true, "thumb": true, "original": true, "thumb_blur": true}

// a plain handler because the body is binary, still behind the getMedia rule
func (s *Service) Serve(w http.ResponseWriter, r *http.Request) {
	p := access.FromContext(r.Context())
	if err := access.Check(p, "getMedia"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, apperr.NotFound())
		return
	}
	variant := r.URL.Query().Get("variant")
	if variant == "" {
		variant = "display"
	}
	if !variants[variant] {
		httpx.WriteError(w, r, apperr.BadRequest("Varian tidak dikenal."))
		return
	}
	row, err := s.q.GetMediaForAccess(r.Context(), sqlc.GetMediaForAccessParams{ID: id, Viewer: p.UserID})
	if db.IsNoRows(err) {
		httpx.WriteError(w, r, apperr.NotFound())
		return
	}
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if !CanView(p, row, variant) {
		httpx.WriteError(w, r, apperr.NotFound())
		return
	}
	var keys Keys
	if err := json.Unmarshal(row.StorageKeys, &keys); err != nil || keys[variant] == "" {
		httpx.WriteError(w, r, apperr.NotFound())
		return
	}
	etag := `"` + id.String() + "-" + variant + `"`
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	body, info, err := s.store.Get(r.Context(), keys[variant])
	if err != nil {
		httpx.WriteError(w, r, apperr.NotFound().WithCause(err))
		return
	}
	defer func() { _ = body.Close() }()
	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if _, err := io.Copy(w, body); err != nil {
		slog.WarnContext(r.Context(), "media stream interrupted", "error", err.Error())
	}
}

// BR-VIS: uploader, report owner and staff see every variant. The owner of a lost report with this found item as a
// visible candidate sees the display variant, or only the blurred thumbnail for sensitivity identity
func CanView(p *access.Principal, row sqlc.GetMediaForAccessRow, variant string) bool {
	if p == nil || row.State == "deleted" {
		return false
	}
	switch {
	case p.IsStaff(), row.UploadedBy == p.UserID:
		return true
	case row.LostReporterID != nil && *row.LostReporterID == p.UserID:
		return true
	case row.FoundFinderID != nil && *row.FoundFinderID == p.UserID:
		return true
	case row.Claimant && row.State == "ready" && row.FoundReportID != nil:
		// a claimant sees the item they are claiming, but an identity card only blurred until the handover
		if row.Sensitivity == "identity" {
			return variant == "thumb_blur"
		}
		return variant == "display" || variant == "thumb" || variant == "thumb_blur"
	case row.CandidateOwner && row.State == "ready":
		if row.Sensitivity == "identity" {
			return variant == "thumb_blur"
		}
		return variant == "display" || variant == "thumb" || variant == "thumb_blur"
	}
	return false
}

func (s *Service) PurgeStaged(ctx context.Context) (int, error) {
	rows, err := s.q.ListStagedMediaBefore(ctx, s.clock.Now().Add(-stagedTTL))
	if err != nil {
		return 0, err
	}
	now := s.clock.Now()
	for _, r := range rows {
		s.DeleteObjects(ctx, Removed{r.StorageKeys})
		if err := s.q.MarkMediaDeleted(ctx, sqlc.MarkMediaDeletedParams{ID: r.ID, DeletedAt: &now}); err != nil {
			return 0, err
		}
	}
	return len(rows), nil
}
