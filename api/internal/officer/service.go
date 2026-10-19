package officer

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
)

type Service struct {
	pool     *pgxpool.Pool
	q        *sqlc.Queries
	clock    clock.Clock
	reports  ReportsGetter
	matching MatchesLister
}

func NewService(pool *pgxpool.Pool, clk clock.Clock) *Service {
	return &Service{
		pool:  pool,
		q:     sqlc.New(pool),
		clock: clk,
	}
}

func (s *Service) resolvePostIDs(p *access.Principal, scope string) []uuid.UUID {
	if p == nil {
		return []uuid.UUID{}
	}
	if p.Role == access.Admin {
		if scope == "campus" || len(p.PostIDs) == 0 {
			return nil
		}
		return p.PostIDs
	}
	if scope == "campus" {
		return nil
	}
	if len(p.PostIDs) == 0 {
		return []uuid.UUID{}
	}
	return p.PostIDs
}

func (s *Service) Summary(ctx context.Context, p *access.Principal, scope string) (openapi.OfficerSummary, error) {
	now := s.clock.Now()
	wib := time.FixedZone("WIB", 7*3600)
	nowWIB := now.In(wib)
	todayStart := time.Date(nowWIB.Year(), nowWIB.Month(), nowWIB.Day(), 0, 0, 0, 0, wib).UTC()
	dueThreshold := now.Add(7 * 24 * time.Hour)

	postIDs := s.resolvePostIDs(p, scope)

	row, err := s.q.GetOfficerSummary(ctx, sqlc.GetOfficerSummaryParams{
		PostIds:      postIDs,
		DueThreshold: dueThreshold,
		TodayStart:   todayStart,
	})
	if err != nil {
		return openapi.OfficerSummary{}, fmt.Errorf("officer summary: %w", err)
	}

	return openapi.OfficerSummary{
		InCustody:        int(row.InCustody),
		RetentionDue:     int(row.RetentionDue),
		WithFinder:       int(row.WithFinder),
		ClaimsPending:    int(row.ClaimsPending),
		Unclaimed:        int(row.Unclaimed),
		FoundToday:       int(row.FoundToday),
		HighMatchNoClaim: int(row.HighMatchNoClaim),
	}, nil
}

func (s *Service) Queue(ctx context.Context, p *access.Principal, params openapi.GetOfficerQueueParams) (openapi.OfficerQueueList, error) {
	now := s.clock.Now()
	scope := "posts"
	if params.Scope != nil {
		scope = string(*params.Scope)
	}

	postIDs := s.resolvePostIDs(p, scope)

	var specificPostID *uuid.UUID
	if params.PostId != nil {
		specificPostID = params.PostId
	}

	dueThreshold := now.Add(7 * 24 * time.Hour)
	overdueThreshold := now.Add(-72 * time.Hour)

	claimRows, err := s.q.ListQueueClaims(ctx, sqlc.ListQueueClaimsParams{
		PostIds:        postIDs,
		SpecificPostID: specificPostID,
	})
	if err != nil {
		return openapi.OfficerQueueList{}, fmt.Errorf("list queue claims: %w", err)
	}

	foundRows, err := s.q.ListQueueFoundReports(ctx, sqlc.ListQueueFoundReportsParams{
		OverdueThreshold: overdueThreshold,
		DueThreshold:     dueThreshold,
		PostIds:          postIDs,
		SpecificPostID:   specificPostID,
	})
	if err != nil {
		return openapi.OfficerQueueList{}, fmt.Errorf("list queue found: %w", err)
	}

	items := make([]openapi.OfficerQueueItem, 0, len(claimRows)+len(foundRows))

	for _, c := range claimRows {
		item := s.buildClaimItem(c, now)
		if params.Type != nil && item.Type != *params.Type {
			continue
		}
		items = append(items, item)
	}

	for _, f := range foundRows {
		itemType, ok := classifyFoundItem(f, now, overdueThreshold, dueThreshold)
		if !ok {
			continue
		}
		if params.Type != nil && itemType != *params.Type {
			continue
		}
		item := s.buildFoundItem(f, itemType, now)
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].PriorityScore != items[j].PriorityScore {
			return items[i].PriorityScore > items[j].PriorityScore
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	total := len(items)

	limit := 20
	if params.Limit != nil && *params.Limit > 0 {
		limit = *params.Limit
	}
	if limit > len(items) {
		limit = len(items)
	}

	return openapi.OfficerQueueList{
		Items: items[:limit],
		Total: total,
	}, nil
}

func classifyFoundItem(f sqlc.ListQueueFoundReportsRow, now, overdueThreshold, dueThreshold time.Time) (openapi.OfficerQueueItemType, bool) {
	if f.FoundStatus == "unclaimed" && f.CustodyStatus == "in_custody" {
		return openapi.UnclaimedDisposition, true
	}
	if f.FoundStatus == "closed" && f.CloseReason != nil && *f.CloseReason == "returned_directly" {
		return openapi.DirectReturnReview, true
	}
	if f.CustodyStatus == "with_finder" && (f.FoundStatus == "reported" || f.FoundStatus == "potential_match") {
		if f.FoundCreatedAt.Before(overdueThreshold) || f.FoundCreatedAt.Equal(overdueThreshold) || f.OverdueFlaggedAt != nil {
			return openapi.OverdueCustody, true
		}
	}
	if f.CustodyStatus == "in_custody" && f.RetentionDueAt != nil && !f.HasActiveClaim {
		if f.FoundStatus != "returned" && f.FoundStatus != "unclaimed" && f.FoundStatus != "closed" {
			if f.RetentionDueAt.Before(dueThreshold) || f.RetentionDueAt.Equal(dueThreshold) {
				return openapi.RetentionDue, true
			}
		}
	}
	if (f.FoundStatus == "reported" || f.FoundStatus == "potential_match") && f.HasHighMatch && !f.HasActiveClaim {
		return openapi.HighMatchNoClaim, true
	}
	return "", false
}

func calculateSensitivity(sensitivity string) float64 {
	switch sensitivity {
	case "identity":
		return 1.0
	case "valuable":
		return 0.7
	default:
		return 0.3
	}
}

func calculateAgePressure(custodyStatus string, createdAt time.Time, receivedAt, retentionDueAt *time.Time, now time.Time) float64 {
	switch custodyStatus {
	case "in_custody":
		if receivedAt != nil && retentionDueAt != nil {
			totalDuration := retentionDueAt.Sub(*receivedAt)
			if totalDuration <= 0 {
				return 1.0
			}
			elapsed := now.Sub(*receivedAt)
			if elapsed <= 0 {
				return 0.0
			}
			fraction := float64(elapsed) / float64(totalDuration)
			if fraction > 1.0 {
				return 1.0
			}
			return fraction
		}
		return 0.0
	case "with_finder":
		elapsed := now.Sub(createdAt)
		if elapsed <= 0 {
			return 0.0
		}
		fraction := float64(elapsed) / float64(48*time.Hour)
		if fraction > 1.0 {
			return 1.0
		}
		return fraction
	default:
		return 0.0
	}
}

func roundScore(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func (s *Service) buildClaimItem(c sqlc.ListQueueClaimsRow, now time.Time) openapi.OfficerQueueItem {
	m := clamp(c.BestMatchScore, 0.0, 1.0)
	sens := calculateSensitivity(c.CategorySensitivity)
	a := calculateAgePressure(c.CustodyStatus, c.FoundCreatedAt, c.ReceivedAt, c.RetentionDueAt, now)

	waitingDuration := now.Sub(c.ClaimCreatedAt)
	u := 0.5
	if waitingDuration > 24*time.Hour {
		u = 1.0
	}

	total := roundScore(0.35*m + 0.20*sens + 0.20*a + 0.25*u)

	chips := make([]string, 0, 4)
	if m > 0 {
		chips = append(chips, fmt.Sprintf("Skor kecocokan %d%%", int(math.Round(m*100))))
	}
	switch sens {
	case 1.0:
		chips = append(chips, "Kartu identitas")
	case 0.7:
		chips = append(chips, "Barang berharga")
	}
	waitingDays := int(waitingDuration.Hours() / 24)
	if waitingDays >= 1 {
		chips = append(chips, fmt.Sprintf("Klaim menunggu %d hari", waitingDays))
	} else {
		chips = append(chips, "Klaim baru menunggu verifikasi")
	}

	postID := c.PostID
	var postName *string
	if c.PostName != "" {
		postName = &c.PostName
	}

	var thumbID *uuid.UUID
	if c.ThumbMediaID != uuid.Nil {
		thumbID = &c.ThumbMediaID
	}

	claimID := c.ClaimID
	claimCode := c.ClaimCode

	return openapi.OfficerQueueItem{
		Id:            fmt.Sprintf("claim-%s", c.ClaimID),
		Type:          openapi.ClaimReview,
		PriorityScore: total,
		ScoreBreakdown: openapi.OfficerQueueScoreBreakdown{
			M:     roundScore(m),
			S:     roundScore(sens),
			A:     roundScore(a),
			U:     roundScore(u),
			Total: total,
		},
		ReasonChips:         chips,
		ActionLabel:         "Periksa klaim",
		ActionUrl:           fmt.Sprintf("/petugas/klaim/%s", c.ClaimCode),
		Title:               c.FoundItemName,
		FoundId:             c.FoundID,
		FoundCode:           c.FoundCode,
		ClaimId:             &claimID,
		ClaimCode:           &claimCode,
		CategoryName:        c.CategoryName,
		CategorySlug:        c.CategorySlug,
		CategorySensitivity: openapi.Sensitivity(c.CategorySensitivity),
		CustodyStatus:       openapi.CustodyStatus(c.CustodyStatus),
		PostId:              postID,
		PostName:            postName,
		ThumbMediaId:        thumbID,
		OccurredAt:          &c.FoundAt,
		CreatedAt:           c.ClaimCreatedAt,
		UpdatedAt:           &c.ClaimUpdatedAt,
	}
}

func (s *Service) buildFoundItem(f sqlc.ListQueueFoundReportsRow, itemType openapi.OfficerQueueItemType, now time.Time) openapi.OfficerQueueItem {
	m := clamp(f.BestMatchScore, 0.0, 1.0)
	sens := calculateSensitivity(f.CategorySensitivity)
	a := calculateAgePressure(f.CustodyStatus, f.FoundCreatedAt, f.ReceivedAt, f.RetentionDueAt, now)

	var u float64
	var actionLabel string
	chips := make([]string, 0, 4)

	if m > 0 {
		chips = append(chips, fmt.Sprintf("Skor kecocokan %d%%", int(math.Round(m*100))))
	}
	switch sens {
	case 1.0:
		chips = append(chips, "Kartu identitas")
	case 0.7:
		chips = append(chips, "Barang berharga")
	}

	switch itemType {
	case openapi.OverdueCustody:
		u = 1.0
		actionLabel = "Hubungi penemu"
		hours := int(now.Sub(f.FoundCreatedAt).Hours())
		chips = append(chips, fmt.Sprintf("Belum diserahkan penemu (%d jam)", hours))

	case openapi.RetentionDue:
		if f.RetentionDueAt != nil && !f.RetentionDueAt.After(now.Add(3*24*time.Hour)) {
			u = 1.0
		} else {
			u = 0.0
		}
		actionLabel = "Lihat barang"
		if f.RetentionDueAt != nil {
			daysLeft := int(math.Ceil(f.RetentionDueAt.Sub(now).Hours() / 24))
			if daysLeft <= 0 {
				chips = append(chips, "Masa simpan habis")
			} else {
				chips = append(chips, fmt.Sprintf("Masa simpan habis %d hari lagi", daysLeft))
			}
		}

	case openapi.UnclaimedDisposition:
		u = 1.0
		actionLabel = "Tentukan tindak lanjut"
		chips = append(chips, "Barang tak terambil")

	case openapi.DirectReturnReview:
		u = 0.0
		actionLabel = "Tinjau"
		chips = append(chips, "Diserahkan langsung oleh penemu")

	case openapi.HighMatchNoClaim:
		u = 0.0
		actionLabel = "Undang pemilik"
		chips = append(chips, "Kandidat kuat tanpa klaim")

	default:
		u = 0.0
		actionLabel = "Lihat barang"
	}

	total := roundScore(0.35*m + 0.20*sens + 0.20*a + 0.25*u)

	postID := f.PostID
	var postName *string
	if f.PostName != "" {
		postName = &f.PostName
	}

	var thumbID *uuid.UUID
	if f.ThumbMediaID != uuid.Nil {
		thumbID = &f.ThumbMediaID
	}

	updatedAt := f.FoundUpdatedAt

	return openapi.OfficerQueueItem{
		Id:            fmt.Sprintf("%s-%s", itemType, f.FoundID),
		Type:          itemType,
		PriorityScore: total,
		ScoreBreakdown: openapi.OfficerQueueScoreBreakdown{
			M:     roundScore(m),
			S:     roundScore(sens),
			A:     roundScore(a),
			U:     roundScore(u),
			Total: total,
		},
		ReasonChips:         chips,
		ActionLabel:         actionLabel,
		ActionUrl:           fmt.Sprintf("/petugas/laporan/temuan/%s", f.FoundCode),
		Title:               f.FoundItemName,
		FoundId:             f.FoundID,
		FoundCode:           f.FoundCode,
		CategoryName:        f.CategoryName,
		CategorySlug:        f.CategorySlug,
		CategorySensitivity: openapi.Sensitivity(f.CategorySensitivity),
		CustodyStatus:       openapi.CustodyStatus(f.CustodyStatus),
		PostId:              postID,
		PostName:            postName,
		ThumbMediaId:        thumbID,
		OccurredAt:          &f.FoundAt,
		CreatedAt:           f.FoundCreatedAt,
		UpdatedAt:           &updatedAt,
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
