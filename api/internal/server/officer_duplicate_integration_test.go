//go:build integration

package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/openapi"
)

func TestDuplicateDetectionAndMerge(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)
	ctx := context.Background()

	// 1. Finder files found report F1
	f1 := fileFound(t, e, s.finder, "dup-found-0001", func(b map[string]any) {
		b["item_name"] = "Dompet kulit cokelat"
		b["description"] = "Dompet kulit berwarna cokelat ditemukan di meja perpustakaan"
	})

	// 2. Test POST /assist/duplicates (US-15)
	categoryID := e.Category("dompet")
	locationID := e.Location("perpustakaan-pusat")
	now := e.Clock.Now()

	// Draft with matching category, location, time, and text
	assistReq := openapi.AssistDuplicatesRequest{
		CategoryId:      categoryID,
		FoundLocationId: locationID,
		FoundAt:         now,
		ItemName:        "Dompet kulit cokelat",
	}
	desc := "Dompet kulit berwarna cokelat di meja baca"
	assistReq.Description = &desc

	var dupResp openapi.AssistDuplicatesResponse
	res := s.finder.Client.JSON(http.MethodPost, "/api/v1/assist/duplicates", assistReq)
	if res.Status != http.StatusOK {
		t.Fatalf("assist duplicates failed: %d %s", res.Status, res.Body)
	}
	res.Decode(t, &dupResp)

	if len(dupResp.Items) == 0 {
		t.Fatalf("expected at least 1 duplicate candidate, got 0")
	}
	if dupResp.Items[0].Id.String() != f1.ID {
		t.Fatalf("expected candidate ID %s, got %s", f1.ID, dupResp.Items[0].Id)
	}
	if dupResp.Items[0].DuplicateScore < 0.80 {
		t.Fatalf("expected duplicate score >= 0.80, got %f", dupResp.Items[0].DuplicateScore)
	}

	// Draft with time beyond 72h gate -> should return 0 items
	farTimeReq := assistReq
	farTimeReq.FoundAt = now.Add(80 * time.Hour)
	var emptyResp openapi.AssistDuplicatesResponse
	s.finder.Client.JSON(http.MethodPost, "/api/v1/assist/duplicates", farTimeReq).Decode(t, &emptyResp)
	if len(emptyResp.Items) != 0 {
		t.Fatalf("expected 0 candidates for time beyond 72h, got %d", len(emptyResp.Items))
	}

	// 3. Permissions check for officer duplicate endpoints
	res = s.finder.Client.JSON(http.MethodGet, "/api/v1/officer/duplicates", nil)
	if res.Status != http.StatusForbidden {
		t.Fatalf("expected 403 for member on /officer/duplicates, got %d", res.Status)
	}

	res = s.officer.Client.JSON(http.MethodGet, "/api/v1/officer/duplicates", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("officer GET /officer/duplicates failed: %d %s", res.Status, res.Body)
	}

	// 4. Finder files found report F2
	f2 := fileFound(t, e, s.finder, "dup-found-0002", func(b map[string]any) {
		b["item_name"] = "Dompet kulit cokelat kedua"
		b["description"] = "Dompet cokelat kulit sama yang dilaporkan ulang"
	})

	// 5. Test POST /officer/duplicates/merge (US-40)
	f1UUID := uuid.MustParse(f1.ID)
	f2UUID := uuid.MustParse(f2.ID)
	mergeReq := openapi.MergeDuplicatesRequest{
		ReportType:  openapi.MergeDuplicatesRequestReportTypeFound,
		SurvivorId:  f1UUID,
		DuplicateId: f2UUID,
	}

	var mergeItem openapi.OfficerDuplicateItem
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/officer/duplicates/merge", mergeReq)
	if res.Status != http.StatusOK {
		t.Fatalf("merge failed: %d %s", res.Status, res.Body)
	}
	res.Decode(t, &mergeItem)

	if mergeItem.Status != openapi.OfficerDuplicateItemStatusConfirmed {
		t.Fatalf("expected status confirmed, got %s", mergeItem.Status)
	}

	// Verify duplicate report F2 is now closed with reason duplicate and duplicate_of F1
	var f2Detail struct {
		Status        string  `json:"status"`
		CloseReason   *string `json:"close_reason"`
		CustodyStatus string  `json:"custody_status"`
	}
	s.officer.Client.JSON(http.MethodGet, "/api/v1/found-reports/"+f2.ID, nil).Decode(t, &f2Detail)
	if f2Detail.Status != "closed" {
		t.Fatalf("f2 status = %q, want 'closed'", f2Detail.Status)
	}
	if f2Detail.CloseReason == nil || *f2Detail.CloseReason != "duplicate" {
		t.Fatalf("f2 close_reason = %v, want 'duplicate'", f2Detail.CloseReason)
	}
	if f2Detail.CustodyStatus != "cancelled" {
		t.Fatalf("f2 custody_status = %q, want 'cancelled'", f2Detail.CustodyStatus)
	}

	var duplicateOf *uuid.UUID
	if err := e.Pool.QueryRow(ctx, "SELECT duplicate_of FROM found_reports WHERE id = $1", f2UUID).Scan(&duplicateOf); err != nil {
		t.Fatalf("query duplicate_of from DB: %v", err)
	}
	if duplicateOf == nil || *duplicateOf != f1UUID {
		t.Fatalf("f2 duplicate_of in DB = %v, want %s", duplicateOf, f1UUID)
	}

	// Verify survivor report F1 is still open
	var f1Detail struct {
		Status string `json:"status"`
	}
	s.officer.Client.JSON(http.MethodGet, "/api/v1/found-reports/"+f1.ID, nil).Decode(t, &f1Detail)
	if f1Detail.Status != "reported" {
		t.Fatalf("f1 status = %q, want 'reported'", f1Detail.Status)
	}

	// 6. Test reject and confirm flow on duplicate links
	// Create another duplicate link via merge of new reports F3 and F4
	f3 := fileFound(t, e, s.finder, "dup-found-0003", nil)
	f4 := fileFound(t, e, s.finder, "dup-found-0004", nil)
	f3UUID := uuid.MustParse(f3.ID)
	f4UUID := uuid.MustParse(f4.ID)

	// Insert a suggested duplicate link directly into duplicate_links
	linkID := uuid.New()
	_, err := e.Pool.Exec(ctx, `
		INSERT INTO duplicate_links (id, report_type, report_a, report_b, score, status, created_at)
		VALUES ($1, 'found', $2, $3, 0.92, 'suggested', $4)
	`, linkID, f3UUID, f4UUID, now)
	if err != nil {
		t.Fatalf("insert test duplicate_link: %v", err)
	}

	// Test GET /officer/duplicates
	var dupList openapi.OfficerDuplicateList
	s.officer.Client.JSON(http.MethodGet, "/api/v1/officer/duplicates?status=suggested", nil).Decode(t, &dupList)
	foundLink := false
	for _, it := range dupList.Items {
		if it.Id == linkID {
			foundLink = true
			if it.Score < 0.90 {
				t.Fatalf("expected score ~0.92, got %f", it.Score)
			}
			break
		}
	}
	if !foundLink {
		t.Fatalf("suggested duplicate link %s not found in list", linkID)
	}

	// Test POST /officer/duplicates/{id}/reject
	var rejectedItem openapi.OfficerDuplicateItem
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/officer/duplicates/"+linkID.String()+"/reject", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("reject failed: %d %s", res.Status, res.Body)
	}
	res.Decode(t, &rejectedItem)
	if rejectedItem.Status != openapi.OfficerDuplicateItemStatusRejected {
		t.Fatalf("expected status rejected, got %s", rejectedItem.Status)
	}

	// Test POST /officer/duplicates/{id}/confirm
	linkID2 := uuid.New()
	f5 := fileFound(t, e, s.finder, "dup-found-0005", nil)
	f6 := fileFound(t, e, s.finder, "dup-found-0006", nil)
	f5UUID := uuid.MustParse(f5.ID)
	f6UUID := uuid.MustParse(f6.ID)
	_, err = e.Pool.Exec(ctx, `
		INSERT INTO duplicate_links (id, report_type, report_a, report_b, score, status, created_at)
		VALUES ($1, 'found', $2, $3, 0.95, 'suggested', $4)
	`, linkID2, f5UUID, f6UUID, now)
	if err != nil {
		t.Fatalf("insert test duplicate_link 2: %v", err)
	}

	confirmReq := openapi.ConfirmDuplicateRequest{
		SurvivorId: &f5UUID,
	}
	var confirmedItem openapi.OfficerDuplicateItem
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/officer/duplicates/"+linkID2.String()+"/confirm", confirmReq)
	if res.Status != http.StatusOK {
		t.Fatalf("confirm failed: %d %s", res.Status, res.Body)
	}
	res.Decode(t, &confirmedItem)
	if confirmedItem.Status != openapi.OfficerDuplicateItemStatusConfirmed {
		t.Fatalf("expected status confirmed, got %s", confirmedItem.Status)
	}
}
