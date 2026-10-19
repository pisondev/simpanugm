//go:build integration

package server_test

import (
	"net/http"
	"testing"

	"github.com/pisondev/simpanugm/api/internal/openapi"
)

func TestOfficerSummaryAndQueue(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)

	// Member cannot access officer endpoints
	res := s.finder.Client.JSON(http.MethodGet, "/api/v1/officer/summary", nil)
	if res.Status != http.StatusForbidden {
		t.Fatalf("finder accessed summary: %d %s", res.Status, res.Body)
	}
	res = s.finder.Client.JSON(http.MethodGet, "/api/v1/officer/queue", nil)
	if res.Status != http.StatusForbidden {
		t.Fatalf("finder accessed queue: %d %s", res.Status, res.Body)
	}

	// 1. Initial summary for officer's post
	var initial openapi.OfficerSummary
	s.officer.Client.JSON(http.MethodGet, "/api/v1/officer/summary", nil).Decode(t, &initial)

	// 2. Report a found item with finder
	s.found = fileFound(t, e, s.finder, "of-found-0001", nil)

	var afterReport openapi.OfficerSummary
	s.officer.Client.JSON(http.MethodGet, "/api/v1/officer/summary", nil).Decode(t, &afterReport)
	if afterReport.WithFinder != initial.WithFinder+1 {
		t.Fatalf("expected with_finder=%d, got %d", initial.WithFinder+1, afterReport.WithFinder)
	}

	// 3. Officer receives the found item into custody
	s.receive(t, s.found, s.officer)

	var afterReceive openapi.OfficerSummary
	s.officer.Client.JSON(http.MethodGet, "/api/v1/officer/summary", nil).Decode(t, &afterReceive)
	if afterReceive.WithFinder != initial.WithFinder || afterReceive.InCustody != initial.InCustody+1 {
		t.Fatalf("after receive expected with_finder=%d in_custody=%d, got %+v", initial.WithFinder, initial.InCustody+1, afterReceive)
	}

	// 4. File a lost report and create a claim
	s.lost = fileLost(t, e, s.owner, "of-lost-0001", nil)
	claim := s.claim(t, s.owner.Client, "of-claim-0001", s.lost.ID, http.StatusCreated)

	var afterClaim openapi.OfficerSummary
	s.officer.Client.JSON(http.MethodGet, "/api/v1/officer/summary", nil).Decode(t, &afterClaim)
	if afterClaim.ClaimsPending != initial.ClaimsPending+1 {
		t.Fatalf("expected claims_pending=%d, got %d", initial.ClaimsPending+1, afterClaim.ClaimsPending)
	}

	// 5. Check Officer Queue
	var queue openapi.OfficerQueueList
	s.officer.Client.JSON(http.MethodGet, "/api/v1/officer/queue?limit=100", nil).Decode(t, &queue)
	if queue.Total < 1 || len(queue.Items) < 1 {
		t.Fatalf("expected at least 1 queue item, got total=%d items=%d", queue.Total, len(queue.Items))
	}

	// Find the claim item in the queue
	var foundClaimItem *openapi.OfficerQueueItem
	for i := range queue.Items {
		if queue.Items[i].Type == openapi.ClaimReview && queue.Items[i].ClaimId != nil && queue.Items[i].ClaimId.String() == claim.ID {
			foundClaimItem = &queue.Items[i]
			break
		}
	}
	if foundClaimItem == nil {
		t.Fatalf("claim %s not found in officer queue: %+v", claim.ID, queue.Items)
	}

	if foundClaimItem.ActionLabel != "Periksa klaim" {
		t.Errorf("action label = %q, want 'Periksa klaim'", foundClaimItem.ActionLabel)
	}
	if foundClaimItem.PriorityScore <= 0 {
		t.Errorf("expected positive priority score, got %v", foundClaimItem.PriorityScore)
	}
	if len(foundClaimItem.ReasonChips) == 0 {
		t.Errorf("expected reason chips for claim item")
	}

	// 6. Test filtering queue by type
	var filteredQueue openapi.OfficerQueueList
	s.officer.Client.JSON(http.MethodGet, "/api/v1/officer/queue?type=claim_review", nil).Decode(t, &filteredQueue)
	if len(filteredQueue.Items) == 0 {
		t.Fatal("expected at least 1 claim_review item")
	}
	for _, it := range filteredQueue.Items {
		if it.Type != openapi.ClaimReview {
			t.Fatalf("unexpected item type in filtered queue: %s", it.Type)
		}
	}

	// 7. Test campus scope view
	var campusSummary openapi.OfficerSummary
	s.officer.Client.JSON(http.MethodGet, "/api/v1/officer/summary?scope=campus", nil).Decode(t, &campusSummary)
	if campusSummary.InCustody < 1 {
		t.Fatalf("expected in_custody >= 1 on campus scope, got %d", campusSummary.InCustody)
	}
}
