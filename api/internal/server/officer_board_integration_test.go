//go:build integration

package server_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/openapi"
)

type boardResponse struct {
	ReportType  string                   `json:"report_type"`
	LostReport  *openapi.LostReport      `json:"lost_report,omitempty"`
	FoundReport *openapi.FoundReport     `json:"found_report,omitempty"`
	Candidates  []openapi.MatchCandidate `json:"candidates"`
}

type pinResponse struct {
	ID       uuid.UUID `json:"id"`
	Pinned   bool      `json:"pinned"`
	PinnedAt *string   `json:"pinned_at"`
}

type inviteResponse struct {
	ID      uuid.UUID `json:"id"`
	Invited bool      `json:"invited"`
}

func TestOfficerBoard_Workflow(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)

	lost := fileLost(t, e, s.owner, "board-lost-001", func(b map[string]any) {
		b["item_name"] = "Dompet cokelat kulit"
		b["description"] = "Dompet kulit warna cokelat merk Eiger, isi KTM dan uang tunai"
	})

	found := fileFound(t, e, s.finder, "board-found-001", func(b map[string]any) {
		b["item_name"] = "Dompet cokelat kulit"
		b["description"] = "Dompet kulit cokelat ditemukan di meja perpustakaan"
	})

	score(t, e)

	t.Cleanup(func() {
		_, _ = e.Pool.Exec(context.Background(), "DELETE FROM matches WHERE lost_report_id = $1 OR found_report_id = $2", lost.ID, found.ID)
	})

	// 1. Officer fetches review board for lost report
	res := s.officer.Client.JSON(http.MethodGet, "/api/v1/officer/board?report_type=lost&report_id="+lost.ID, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("GET /officer/board (lost): %d %s", res.Status, res.Body)
	}
	var lostBoard boardResponse
	res.Decode(t, &lostBoard)

	if lostBoard.ReportType != "lost" {
		t.Fatalf("expected report_type lost, got %s", lostBoard.ReportType)
	}
	if lostBoard.LostReport == nil || lostBoard.LostReport.Id.String() != lost.ID {
		t.Fatalf("expected lost report ID %s", lost.ID)
	}
	if len(lostBoard.Candidates) == 0 {
		t.Fatalf("expected candidates on board, got 0")
	}

	matchID := lostBoard.Candidates[0].Id

	// 2. Officer fetches review board for found report
	res = s.officer.Client.JSON(http.MethodGet, "/api/v1/officer/board?report_type=found&report_id="+found.ID, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("GET /officer/board (found): %d %s", res.Status, res.Body)
	}
	var foundBoard boardResponse
	res.Decode(t, &foundBoard)

	if foundBoard.ReportType != "found" {
		t.Fatalf("expected report_type found, got %s", foundBoard.ReportType)
	}
	if foundBoard.FoundReport == nil || foundBoard.FoundReport.Id.String() != found.ID {
		t.Fatalf("expected found report ID %s", found.ID)
	}
	if len(foundBoard.Candidates) == 0 {
		t.Fatalf("expected candidates on found board, got 0")
	}

	// 3. Officer pins match
	pinReq := map[string]any{"pinned": true}
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/matches/"+matchID.String()+"/pin", pinReq)
	if res.Status != http.StatusOK {
		t.Fatalf("POST /matches/{id}/pin: %d %s", res.Status, res.Body)
	}
	var pinRes pinResponse
	res.Decode(t, &pinRes)
	if !pinRes.Pinned || pinRes.PinnedAt == nil {
		t.Fatalf("expected pinned=true with timestamp, got %v", pinRes)
	}

	// Verify candidate list reflects pinned status
	res = s.officer.Client.JSON(http.MethodGet, "/api/v1/officer/board?report_type=lost&report_id="+lost.ID, nil)
	res.Decode(t, &lostBoard)
	if lostBoard.Candidates[0].Pinned == nil || !*lostBoard.Candidates[0].Pinned {
		t.Fatalf("expected candidate to be pinned")
	}

	// 4. Officer unpins match
	unpinReq := map[string]any{"pinned": false}
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/matches/"+matchID.String()+"/pin", unpinReq)
	if res.Status != http.StatusOK {
		t.Fatalf("POST /matches/{id}/pin (unpin): %d %s", res.Status, res.Body)
	}
	var unpinRes pinResponse
	res.Decode(t, &unpinRes)
	if unpinRes.Pinned || unpinRes.PinnedAt != nil {
		t.Fatalf("expected pinned=false, got %v", unpinRes)
	}

	// 5. Officer invites claim
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/matches/"+matchID.String()+"/invite-claim", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("POST /matches/{id}/invite-claim: %d %s", res.Status, res.Body)
	}
	var invRes inviteResponse
	res.Decode(t, &invRes)
	if !invRes.Invited {
		t.Fatalf("expected invited=true, got %v", invRes)
	}

	// Verify notification sent to owner
	evs := events(t, s.owner.Client)
	foundInvite := false
	for _, ev := range evs {
		if ev == string(notify.MatchOfficerInvites) {
			foundInvite = true
			break
		}
	}
	if !foundInvite {
		t.Fatalf("expected notification %s for owner, got %v", notify.MatchOfficerInvites, evs)
	}

	// Verify audit log recorded
	var auditCount int
	err := e.Pool.QueryRow(context.Background(),
		"SELECT count(*) FROM audit_logs WHERE action = 'match.invite_claim' AND entity_id = $1", matchID).Scan(&auditCount)
	if err != nil || auditCount < 1 {
		t.Fatalf("expected audit log for match.invite_claim, count=%d, err=%v", auditCount, err)
	}
}

func TestOfficerBoard_Permissions(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)

	lost := fileLost(t, e, s.owner, "board-perm-lost", nil)
	fakeMatchID := uuid.New()

	// Member cannot access board
	res := s.owner.Client.JSON(http.MethodGet, "/api/v1/officer/board?report_type=lost&report_id="+lost.ID, nil)
	if res.Status != http.StatusForbidden {
		t.Fatalf("member GET /officer/board: expected 403, got %d", res.Status)
	}

	// Member cannot pin match
	res = s.owner.Client.JSON(http.MethodPost, "/api/v1/matches/"+fakeMatchID.String()+"/pin", map[string]any{"pinned": true})
	if res.Status != http.StatusForbidden {
		t.Fatalf("member POST /matches/pin: expected 403, got %d", res.Status)
	}

	// Member cannot invite claim
	res = s.owner.Client.JSON(http.MethodPost, "/api/v1/matches/"+fakeMatchID.String()+"/invite-claim", nil)
	if res.Status != http.StatusForbidden {
		t.Fatalf("member POST /matches/invite-claim: expected 403, got %d", res.Status)
	}
}
