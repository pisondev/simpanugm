package officer

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/openapi"
)

func TestSensitivity(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"identity", 1.0},
		{"valuable", 0.7},
		{"normal", 0.3},
		{"unknown", 0.3},
	}
	for _, tc := range cases {
		got := calculateSensitivity(tc.in)
		if got != tc.want {
			t.Errorf("calculateSensitivity(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestAgePressure(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)

	// with_finder: 24h elapsed of 48h -> 0.5
	created := now.Add(-24 * time.Hour)
	p := calculateAgePressure("with_finder", created, nil, nil, now)
	if math.Abs(p-0.5) > 0.001 {
		t.Errorf("age pressure with_finder 24h = %v, want 0.5", p)
	}

	// with_finder: 72h elapsed -> clamped to 1.0
	createdOverdue := now.Add(-72 * time.Hour)
	pOver := calculateAgePressure("with_finder", createdOverdue, nil, nil, now)
	if pOver != 1.0 {
		t.Errorf("age pressure with_finder 72h = %v, want 1.0", pOver)
	}

	// in_custody: received 10 days ago, due in 10 days (total 20 days, elapsed 10 days) -> 0.5
	received := now.Add(-10 * 24 * time.Hour)
	due := now.Add(10 * 24 * time.Hour)
	pCustody := calculateAgePressure("in_custody", time.Time{}, &received, &due, now)
	if math.Abs(pCustody-0.5) > 0.001 {
		t.Errorf("age pressure in_custody = %v, want 0.5", pCustody)
	}
}

func TestClassifyFoundItem(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	overdueThresh := now.Add(-72 * time.Hour)
	dueThresh := now.Add(7 * 24 * time.Hour)

	// unclaimed in custody
	row1 := sqlc.ListQueueFoundReportsRow{
		FoundStatus:   "unclaimed",
		CustodyStatus: "in_custody",
	}
	itemType, ok := classifyFoundItem(row1, now, overdueThresh, dueThresh)
	if !ok || itemType != openapi.UnclaimedDisposition {
		t.Errorf("unclaimed got %v, want %v", itemType, openapi.UnclaimedDisposition)
	}

	// direct return closed
	retDirect := "returned_directly"
	row2 := sqlc.ListQueueFoundReportsRow{
		FoundStatus: "closed",
		CloseReason: &retDirect,
	}
	itemType, ok = classifyFoundItem(row2, now, overdueThresh, dueThresh)
	if !ok || itemType != openapi.DirectReturnReview {
		t.Errorf("direct return got %v, want %v", itemType, openapi.DirectReturnReview)
	}

	// overdue with finder
	row3 := sqlc.ListQueueFoundReportsRow{
		FoundStatus:    "reported",
		CustodyStatus:  "with_finder",
		FoundCreatedAt: now.Add(-80 * time.Hour),
	}
	itemType, ok = classifyFoundItem(row3, now, overdueThresh, dueThresh)
	if !ok || itemType != openapi.OverdueCustody {
		t.Errorf("overdue got %v, want %v", itemType, openapi.OverdueCustody)
	}

	// retention due
	dueSoon := now.Add(2 * 24 * time.Hour)
	row4 := sqlc.ListQueueFoundReportsRow{
		FoundStatus:    "potential_match",
		CustodyStatus:  "in_custody",
		RetentionDueAt: &dueSoon,
	}
	itemType, ok = classifyFoundItem(row4, now, overdueThresh, dueThresh)
	if !ok || itemType != openapi.RetentionDue {
		t.Errorf("retention due got %v, want %v", itemType, openapi.RetentionDue)
	}

	// high match without claim
	row5 := sqlc.ListQueueFoundReportsRow{
		FoundStatus:    "reported",
		CustodyStatus:  "with_finder",
		FoundCreatedAt: now.Add(-5 * time.Hour),
		HasHighMatch:   true,
		HasActiveClaim: false,
	}
	itemType, ok = classifyFoundItem(row5, now, overdueThresh, dueThresh)
	if !ok || itemType != openapi.HighMatchNoClaim {
		t.Errorf("high match got %v, want %v", itemType, openapi.HighMatchNoClaim)
	}
}

func TestScoreBreakdown(t *testing.T) {
	// P = 0.35 * m + 0.20 * s + 0.20 * a + 0.25 * u
	m := 0.86
	s := 1.0                                           // identity
	a := 0.5                                           // half retention elapsed
	u := 1.0                                           // claim waiting > 24h
	want := 0.35*0.86 + 0.20*1.0 + 0.20*0.5 + 0.25*1.0 // 0.301 + 0.20 + 0.10 + 0.25 = 0.851
	got := roundScore(0.35*m + 0.20*s + 0.20*a + 0.25*u)
	if math.Abs(got-want) > 0.0001 {
		t.Errorf("score = %v, want %v", got, want)
	}
}

func TestBuildClaimItem(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	svc := &Service{}
	received := now.Add(-5 * 24 * time.Hour)
	due := now.Add(5 * 24 * time.Hour)

	row := sqlc.ListQueueClaimsRow{
		ClaimID:             uuid.New(),
		ClaimCode:           "C-2026-0001",
		ClaimStatus:         "submitted",
		ClaimCreatedAt:      now.Add(-48 * time.Hour),
		ClaimUpdatedAt:      now.Add(-48 * time.Hour),
		FoundID:             uuid.New(),
		FoundCode:           "F-2026-0001",
		FoundItemName:       "Dompet Hitam",
		CustodyStatus:       "in_custody",
		FoundCreatedAt:      now.Add(-10 * 24 * time.Hour),
		ReceivedAt:          &received,
		RetentionDueAt:      &due,
		FoundAt:             now.Add(-11 * 24 * time.Hour),
		CategoryName:        "Dompet",
		CategorySlug:        "dompet",
		CategorySensitivity: "valuable",
		BestMatchScore:      0.8,
	}

	item := svc.buildClaimItem(row, now)
	if item.Type != openapi.ClaimReview {
		t.Errorf("type = %v, want %v", item.Type, openapi.ClaimReview)
	}
	if item.ActionLabel != "Periksa klaim" {
		t.Errorf("action label = %q", item.ActionLabel)
	}
	if item.ActionUrl != "/petugas/klaim/C-2026-0001" {
		t.Errorf("action url = %q", item.ActionUrl)
	}
	if len(item.ReasonChips) == 0 {
		t.Error("expected reason chips")
	}
}
