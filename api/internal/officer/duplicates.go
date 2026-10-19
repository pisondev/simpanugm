package officer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
)

func (s *Service) getReportSummary(ctx context.Context, q *sqlc.Queries, reportType string, id uuid.UUID) (openapi.DuplicateReportSummary, error) {
	if reportType == "found" {
		r, err := q.GetFoundReportSummary(ctx, id)
		if err != nil {
			return openapi.DuplicateReportSummary{}, err
		}
		var thumb *uuid.UUID
		if r.ThumbMediaID != uuid.Nil {
			thumb = &r.ThumbMediaID
		}
		return openapi.DuplicateReportSummary{
			Id:           r.ID,
			Code:         r.Code,
			ItemName:     r.ItemName,
			Status:       r.Status,
			CategoryName: r.CategoryName,
			LocationName: r.LocationName,
			OccurredAt:   r.FoundAt,
			ThumbMediaId: thumb,
		}, nil
	}

	r, err := q.GetLostReportSummary(ctx, id)
	if err != nil {
		return openapi.DuplicateReportSummary{}, err
	}
	var thumb *uuid.UUID
	if r.ThumbMediaID != uuid.Nil {
		thumb = &r.ThumbMediaID
	}
	return openapi.DuplicateReportSummary{
		Id:           r.ID,
		Code:         r.Code,
		ItemName:     r.ItemName,
		Status:       r.Status,
		CategoryName: r.CategoryName,
		LocationName: r.LocationName,
		OccurredAt:   r.OccurredAt,
		ThumbMediaId: thumb,
	}, nil
}

func (s *Service) mapDuplicateItem(ctx context.Context, q *sqlc.Queries, dl sqlc.DuplicateLink) (openapi.OfficerDuplicateItem, error) {
	repA, err := s.getReportSummary(ctx, q, dl.ReportType, dl.ReportA)
	if err != nil {
		return openapi.OfficerDuplicateItem{}, fmt.Errorf("report_a: %w", err)
	}
	repB, err := s.getReportSummary(ctx, q, dl.ReportType, dl.ReportB)
	if err != nil {
		return openapi.OfficerDuplicateItem{}, fmt.Errorf("report_b: %w", err)
	}

	return openapi.OfficerDuplicateItem{
		Id:         dl.ID,
		ReportType: openapi.OfficerDuplicateItemReportType(dl.ReportType),
		Score:      float32(dl.Score),
		Status:     openapi.OfficerDuplicateItemStatus(dl.Status),
		CreatedAt:  dl.CreatedAt,
		ReviewedAt: dl.ReviewedAt,
		ReviewedBy: dl.ReviewedBy,
		ReportA:    repA,
		ReportB:    repB,
	}, nil
}

func (s *Service) ListDuplicates(ctx context.Context, status *string, reportType *string, limit int) (openapi.OfficerDuplicateList, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.q.ListDuplicateLinks(ctx, sqlc.ListDuplicateLinksParams{
		Status:     status,
		ReportType: reportType,
		Lim:        int32(limit),
	})
	if err != nil {
		return openapi.OfficerDuplicateList{Items: []openapi.OfficerDuplicateItem{}, Total: 0}, err
	}

	items := make([]openapi.OfficerDuplicateItem, 0, len(rows))
	for _, row := range rows {
		item, err := s.mapDuplicateItem(ctx, s.q, sqlc.DuplicateLink(row))
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return openapi.OfficerDuplicateList{
		Items: items,
		Total: len(items),
	}, nil
}

func (s *Service) executeMerge(ctx context.Context, q *sqlc.Queries, p *access.Principal, reportType string, survivorID, duplicateID uuid.UUID, now time.Time) error {
	if reportType == "found" {
		survivor, err := q.GetFoundReport(ctx, sqlc.GetFoundReportParams{ID: &survivorID})
		if err != nil {
			if db.IsNoRows(err) {
				return apperr.NotFound()
			}
			return err
		}
		duplicate, err := q.GetFoundReport(ctx, sqlc.GetFoundReportParams{ID: &duplicateID})
		if err != nil {
			if db.IsNoRows(err) {
				return apperr.NotFound()
			}
			return err
		}
		if survivor.FoundReport.Status == "closed" || duplicate.FoundReport.Status == "closed" {
			return apperr.New(http.StatusConflict, "cannot_merge", "Laporan tidak dapat digabungkan", "Laporan yang sudah ditutup tidak dapat digabungkan.")
		}

		if err := q.MergeCloseFoundReport(ctx, sqlc.MergeCloseFoundReportParams{
			ClosedAt:    now,
			SurvivorID:  &survivorID,
			DuplicateID: duplicateID,
		}); err != nil {
			return err
		}

		if err := q.MoveFoundMediaToSurvivor(ctx, sqlc.MoveFoundMediaToSurvivorParams{
			SurvivorID:  &survivorID,
			DuplicateID: &duplicateID,
		}); err != nil {
			return err
		}

		if err := q.DeleteUnmovedFoundMedia(ctx, sqlc.DeleteUnmovedFoundMediaParams{
			DuplicateID: &duplicateID,
			Now:         now,
		}); err != nil {
			return err
		}

		if err := q.MoveFoundClaimsToSurvivor(ctx, sqlc.MoveFoundClaimsToSurvivorParams{
			SurvivorID:  survivorID,
			DuplicateID: duplicateID,
		}); err != nil {
			return err
		}

		if err := q.RejectConflictingFoundClaims(ctx, sqlc.RejectConflictingFoundClaimsParams{
			DuplicateID: duplicateID,
			Now:         now,
		}); err != nil {
			return err
		}

		if err := q.SupersedeFoundMatches(ctx, duplicateID); err != nil {
			return err
		}

		return audit.Record(ctx, q, now, audit.Entry{
			Action:     "found_report.merge",
			EntityType: "found_report",
			EntityID:   duplicateID,
			Changes: map[string]any{
				"status":         audit.Change(duplicate.FoundReport.Status, "closed"),
				"close_reason":   "duplicate",
				"duplicate_of":   survivorID,
				"custody_status": audit.Change(duplicate.FoundReport.CustodyStatus, "cancelled"),
			},
		})
	}

	// Lost report merge
	survivor, err := q.GetLostReport(ctx, sqlc.GetLostReportParams{ID: &survivorID})
	if err != nil {
		if db.IsNoRows(err) {
			return apperr.NotFound()
		}
		return err
	}
	duplicate, err := q.GetLostReport(ctx, sqlc.GetLostReportParams{ID: &duplicateID})
	if err != nil {
		if db.IsNoRows(err) {
			return apperr.NotFound()
		}
		return err
	}
	if survivor.LostReport.Status == "closed" || duplicate.LostReport.Status == "closed" {
		return apperr.New(http.StatusConflict, "cannot_merge", "Laporan tidak dapat digabungkan", "Laporan yang sudah ditutup tidak dapat digabungkan.")
	}

	// L13: survivor is open and same owner
	if survivor.LostReport.ReporterID != duplicate.LostReport.ReporterID {
		return apperr.New(http.StatusConflict, "cannot_merge", "Laporan tidak dapat digabungkan", "Hanya laporan hilang dari pemilik yang sama yang dapat digabungkan.")
	}

	if err := q.MergeCloseLostReport(ctx, sqlc.MergeCloseLostReportParams{
		ClosedAt:    now,
		SurvivorID:  &survivorID,
		DuplicateID: duplicateID,
	}); err != nil {
		return err
	}

	if err := q.MoveLostMediaToSurvivor(ctx, sqlc.MoveLostMediaToSurvivorParams{
		SurvivorID:  &survivorID,
		DuplicateID: &duplicateID,
	}); err != nil {
		return err
	}

	if err := q.DeleteUnmovedLostMedia(ctx, sqlc.DeleteUnmovedLostMediaParams{
		DuplicateID: &duplicateID,
		Now:         now,
	}); err != nil {
		return err
	}

	if err := q.MoveLostClaimsToSurvivor(ctx, sqlc.MoveLostClaimsToSurvivorParams{
		SurvivorID:  survivorID,
		DuplicateID: duplicateID,
	}); err != nil {
		return err
	}

	if err := q.SupersedeLostMatches(ctx, duplicateID); err != nil {
		return err
	}

	return audit.Record(ctx, q, now, audit.Entry{
		Action:     "lost_report.merge",
		EntityType: "lost_report",
		EntityID:   duplicateID,
		Changes: map[string]any{
			"status":       audit.Change(duplicate.LostReport.Status, "closed"),
			"close_reason": "duplicate",
			"duplicate_of": survivorID,
		},
	})
}

func (s *Service) ConfirmDuplicate(ctx context.Context, p *access.Principal, id uuid.UUID, survivorID *uuid.UUID) (openapi.OfficerDuplicateItem, error) {
	dl, err := s.q.GetDuplicateLink(ctx, id)
	if err != nil {
		if db.IsNoRows(err) {
			return openapi.OfficerDuplicateItem{}, apperr.NotFound()
		}
		return openapi.OfficerDuplicateItem{}, err
	}

	chosenSurvivor := dl.ReportA
	chosenDuplicate := dl.ReportB
	if survivorID != nil {
		if *survivorID == dl.ReportB {
			chosenSurvivor = dl.ReportB
			chosenDuplicate = dl.ReportA
		} else if *survivorID != dl.ReportA {
			return openapi.OfficerDuplicateItem{}, apperr.Validation(apperr.FieldError{
				Field:   "survivor_id",
				Code:    "invalid",
				Message: "survivor_id harus berupa salah satu dari laporan yang dipasangkan",
			})
		}
	}

	now := s.clock.Now()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return openapi.OfficerDuplicateItem{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	if err := s.executeMerge(ctx, qtx, p, dl.ReportType, chosenSurvivor, chosenDuplicate, now); err != nil {
		return openapi.OfficerDuplicateItem{}, err
	}

	updated, err := qtx.UpdateDuplicateLinkStatus(ctx, sqlc.UpdateDuplicateLinkStatusParams{
		ID:         id,
		Status:     "confirmed",
		ReviewedBy: &p.UserID,
		ReviewedAt: &now,
	})
	if err != nil {
		return openapi.OfficerDuplicateItem{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return openapi.OfficerDuplicateItem{}, err
	}

	return s.mapDuplicateItem(ctx, s.q, sqlc.DuplicateLink(updated))
}

func (s *Service) RejectDuplicate(ctx context.Context, p *access.Principal, id uuid.UUID) (openapi.OfficerDuplicateItem, error) {
	now := s.clock.Now()
	updated, err := s.q.UpdateDuplicateLinkStatus(ctx, sqlc.UpdateDuplicateLinkStatusParams{
		ID:         id,
		Status:     "rejected",
		ReviewedBy: &p.UserID,
		ReviewedAt: &now,
	})
	if err != nil {
		if db.IsNoRows(err) {
			return openapi.OfficerDuplicateItem{}, apperr.NotFound()
		}
		return openapi.OfficerDuplicateItem{}, err
	}

	return s.mapDuplicateItem(ctx, s.q, sqlc.DuplicateLink(updated))
}

func (s *Service) MergeDuplicates(ctx context.Context, p *access.Principal, reportType string, survivorID, duplicateID uuid.UUID) (openapi.OfficerDuplicateItem, error) {
	if survivorID == duplicateID {
		return openapi.OfficerDuplicateItem{}, apperr.Validation(apperr.FieldError{
			Field:   "duplicate_id",
			Code:    "invalid",
			Message: "Laporan yang bertahan dan duplikat tidak boleh sama",
		})
	}

	now := s.clock.Now()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return openapi.OfficerDuplicateItem{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	if err := s.executeMerge(ctx, qtx, p, reportType, survivorID, duplicateID, now); err != nil {
		return openapi.OfficerDuplicateItem{}, err
	}

	linkID := uuid.New()
	upserted, err := qtx.UpsertDuplicateLink(ctx, sqlc.UpsertDuplicateLinkParams{
		ID:         linkID,
		ReportType: reportType,
		ReportA:    survivorID,
		ReportB:    duplicateID,
		Score:      1.0,
		Status:     "confirmed",
		ReviewedBy: &p.UserID,
		ReviewedAt: &now,
		CreatedAt:  now,
	})
	if err != nil {
		return openapi.OfficerDuplicateItem{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return openapi.OfficerDuplicateItem{}, err
	}

	return s.mapDuplicateItem(ctx, s.q, sqlc.DuplicateLink(upserted))
}
