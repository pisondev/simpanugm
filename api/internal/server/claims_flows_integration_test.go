//go:build integration

package server_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

// a further received found wallet and a lost report of the scene owner, for tests that need several items
func (s *scene) another(t *testing.T, key string) (foundReport, lostReport) {
	t.Helper()
	l := fileLost(t, s.e, s.owner, key+"-lost", nil)
	f := fileFound(t, s.e, s.finder, key+"-found", nil)
	s.receive(t, f, s.officer)
	return f, l
}

func (s *scene) claimOn(t *testing.T, f foundReport, l lostReport, key string) claimView {
	t.Helper()
	res := s.owner.Client.JSON(http.MethodPost, "/api/v1/claims", claimBody(f.ID, l.ID, s.walletAnswers()), "Idempotency-Key", key)
	if res.Status != http.StatusCreated {
		t.Fatalf("claim on %s: %d %s", f.Code, res.Status, res.Body)
	}
	var v claimView
	res.Decode(t, &v)
	return v
}

// BR-CLM-02 to BR-CLM-04 and FR-CLAIM-02
func TestClaimRulesAndRejections(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)
	s.lost = fileLost(t, e, s.owner, "lim-lost-0001", nil)
	s.found = fileFound(t, e, s.finder, "lim-found-0001", nil)
	s.receive(t, s.found, s.officer)

	own := fileLost(t, e, s.finder, "lim-lost-own", nil)
	if res := s.finder.Client.JSON(http.MethodPost, "/api/v1/claims", claimBody(s.found.ID, own.ID, s.walletAnswers()), "Idempotency-Key", "lim-key-own"); res.Status != http.StatusForbidden || res.Problem(t).Code != "own_found_report" {
		t.Fatalf("claiming one's own report: %d %s", res.Status, res.Body)
	}
	missing := claimBody(s.found.ID, s.lost.ID, s.walletAnswers()[:1])
	if res := s.owner.Client.JSON(http.MethodPost, "/api/v1/claims", missing, "Idempotency-Key", "lim-key-missing"); !fieldError(t, res, "answers.wallet_private_mark") {
		t.Fatalf("a required question left blank: %d %s", res.Status, res.Body)
	}
	short := claimBody(s.found.ID, s.lost.ID, s.walletAnswers())
	short["statement"] = "terlalu pendek"
	if res := s.owner.Client.JSON(http.MethodPost, "/api/v1/claims", short, "Idempotency-Key", "lim-key-short"); res.Status != http.StatusUnprocessableEntity && res.Status != http.StatusBadRequest {
		t.Fatalf("short statement: %d", res.Status)
	}
	if res := s.finder.Client.JSON(http.MethodPost, "/api/v1/claims", claimBody(s.found.ID, s.lost.ID, s.walletAnswers()), "Idempotency-Key", "lim-key-steal"); res.Status != http.StatusNotFound {
		t.Fatalf("claiming with someone else's lost report: %d", res.Status)
	}
	first := s.claim(t, s.owner.Client, "lim-key-0001", s.lost.ID, http.StatusCreated)
	if res := s.owner.Client.JSON(http.MethodPost, "/api/v1/claims", claimBody(s.found.ID, s.lost.ID, s.walletAnswers()), "Idempotency-Key", "lim-key-twice"); res.Problem(t).Code != "claim_exists" {
		t.Fatalf("two claims on one item: %d %s", res.Status, res.Body)
	}

	post(t, s.officer.Client, "/api/v1/claims/"+first.Code+"/pick-up", nil, http.StatusOK)
	if res := s.officer.Client.JSON(http.MethodPost, "/api/v1/claims/"+first.Code+"/reject", map[string]any{"reason": "other"}); !fieldError(t, res, "note") {
		t.Fatalf("other without a note: %d %s", res.Status, res.Body)
	}
	post(t, s.officer.Client, "/api/v1/claims/"+first.Code+"/reject", map[string]any{"reason": "details_mismatch", "note": "Isi dompet berbeda."}, http.StatusOK)
	var pair string
	_ = e.Pool.QueryRow(context.Background(), "SELECT status FROM matches WHERE lost_report_id = $1 AND found_report_id = $2", s.lost.ID, s.found.ID).Scan(&pair)
	if pair != "" && pair != "rejected" {
		t.Fatalf("M7: %s", pair)
	}
	if res := s.owner.Client.JSON(http.MethodPost, "/api/v1/claims", claimBody(s.found.ID, s.lost.ID, s.walletAnswers()), "Idempotency-Key", "lim-key-again"); res.Problem(t).Code != "claim_rejected_before" {
		t.Fatalf("a claim after a rejection: %d %s", res.Status, res.Body)
	}
	if ev := events(t, s.owner.Client); count(ev, "claim.rejected") != 1 {
		t.Fatalf("the claimant is told: %v", ev)
	}
	// K15
	if res := s.officer.Client.JSON(http.MethodPost, "/api/v1/claims/"+first.Code+"/reopen", map[string]any{"reason": "ok"}); res.Status != http.StatusUnprocessableEntity && res.Status != http.StatusBadRequest {
		t.Fatalf("a reason of two characters: %d", res.Status)
	}
	if v := post(t, s.officer.Client, "/api/v1/claims/"+first.Code+"/reopen", map[string]any{"reason": "Bukti baru dari pemilik"}, http.StatusOK); v.Status != "under_review" {
		t.Fatalf("reopened: %s", v.Status)
	}

	// BR-CLM-04
	for i := 0; i < 3; i++ {
		f, l := s.another(t, "lim-"+itoa(i))
		c := s.claimOn(t, f, l, "lim-rej-"+itoa(i))
		post(t, s.officer.Client, "/api/v1/claims/"+c.Code+"/pick-up", nil, http.StatusOK)
		post(t, s.officer.Client, "/api/v1/claims/"+c.Code+"/reject", map[string]any{"reason": "insufficient_evidence"}, http.StatusOK)
	}
	f, l := s.another(t, "lim-final")
	if res := s.owner.Client.JSON(http.MethodPost, "/api/v1/claims", claimBody(f.ID, l.ID, s.walletAnswers()), "Idempotency-Key", "lim-restricted"); res.Problem(t).Code != "claim_restricted" {
		t.Fatalf("a restricted member claims: %d %s", res.Status, res.Body)
	}
}

// moves a claim timestamp into the past, which a clock advance cannot do without also ending every session
func age(t *testing.T, e *testenv.Env, column, code, by string) {
	t.Helper()
	if _, err := e.Pool.Exec(context.Background(), "UPDATE claims SET "+column+" = "+column+" - $2::interval WHERE code = $1", code, by); err != nil {
		t.Fatal(err)
	}
}

// BR-CLM-01
func TestFourthOpenClaimIsRefused(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)
	for i := 0; i < 3; i++ {
		f, l := s.another(t, "four-"+itoa(i))
		s.claimOn(t, f, l, "four-claim-"+itoa(i))
	}
	f, l := s.another(t, "four-last")
	res := s.owner.Client.JSON(http.MethodPost, "/api/v1/claims", claimBody(f.ID, l.ID, s.walletAnswers()), "Idempotency-Key", "four-claim-last")
	if res.Problem(t).Code != "claim_limit_reached" {
		t.Fatalf("fourth claim: %d %s", res.Status, res.Body)
	}
}

// FR-CLAIM-09, US-27 and US-28
func TestInformationRequestsMessagesAndWithdrawal(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)
	s.lost = fileLost(t, e, s.owner, "msg-lost-0001", nil)
	s.found = fileFound(t, e, s.finder, "msg-found-0001", nil)
	s.receive(t, s.found, s.officer)
	c := s.claim(t, s.owner.Client, "msg-key-0001", s.lost.ID, http.StatusCreated)
	post(t, s.officer.Client, "/api/v1/claims/"+c.Code+"/pick-up", nil, http.StatusOK)
	if res := s.officer.Client.JSON(http.MethodPost, "/api/v1/claims/"+c.Code+"/request-info", map[string]any{"message": "  "}); res.Status != http.StatusUnprocessableEntity && res.Status != http.StatusBadRequest {
		t.Fatalf("an empty question: %d", res.Status)
	}
	if v := post(t, s.officer.Client, "/api/v1/claims/"+c.Code+"/request-info", map[string]any{"message": "Foto kartu apa yang ada di dompet?"}, http.StatusOK); v.Status != "info_requested" {
		t.Fatalf("K3: %s", v.Status)
	}
	if ev := events(t, s.owner.Client); count(ev, "claim.info_requested") != 1 {
		t.Fatalf("the claimant is asked: %v", ev)
	}
	if res := s.officer.Client.JSON(http.MethodPost, "/api/v1/claims/"+c.Code+"/respond", map[string]any{"message": "x"}); res.Status < 400 {
		t.Fatalf("an officer answering: %d", res.Status)
	}
	if v := post(t, s.owner.Client, "/api/v1/claims/"+c.Code+"/respond", map[string]any{"message": "Foto kucing kecil dan KTM."}, http.StatusOK); v.Status != "under_review" {
		t.Fatalf("K4: %s", v.Status)
	}
	if ev := events(t, s.officer.Client); count(ev, "claim.info_provided") != 1 {
		t.Fatalf("the reviewer hears the answer: %v", ev)
	}
	post(t, s.owner.Client, "/api/v1/claims/"+c.Code+"/messages", map[string]any{"message": "Terima kasih, mohon dicek."}, http.StatusCreated)
	var thread struct {
		Items []struct {
			Body string `json:"body"`
			Mine bool   `json:"mine"`
		} `json:"items"`
	}
	s.officer.Client.JSON(http.MethodGet, "/api/v1/claims/"+c.Code+"/messages", nil).Decode(t, &thread)
	if len(thread.Items) < 3 || thread.Items[len(thread.Items)-1].Mine {
		t.Fatalf("thread as the officer sees it: %+v", thread)
	}
	if res := s.finder.Client.JSON(http.MethodGet, "/api/v1/claims/"+c.Code+"/messages", nil); res.Status != http.StatusNotFound && res.Status != http.StatusForbidden {
		t.Fatalf("the finder reads the thread: %d", res.Status)
	}
	if res := s.owner.Client.JSON(http.MethodPatch, "/api/v1/claims/"+c.Code+"/internal-note", map[string]any{"note": "x"}); res.Status != http.StatusForbidden {
		t.Fatalf("a member writing the internal note: %d", res.Status)
	}
	if res := s.officer.Client.JSON(http.MethodPatch, "/api/v1/claims/"+c.Code+"/internal-note", map[string]any{"note": "Cek KTM saat serah terima."}); res.Status != http.StatusOK {
		t.Fatalf("internal note: %d %s", res.Status, res.Body)
	}
	if got := get(t, s.owner.Client, c.Code); got.InternalNote != nil {
		t.Fatalf("the internal note reached the claimant")
	}
	// K14
	if v := post(t, s.owner.Client, "/api/v1/claims/"+c.Code+"/withdraw", nil, http.StatusOK); v.Status != "withdrawn" {
		t.Fatalf("K14: %s", v.Status)
	}
	if res := s.officer.Client.JSON(http.MethodPost, "/api/v1/claims/"+c.Code+"/pick-up", nil); res.Status != http.StatusConflict {
		t.Fatalf("working a withdrawn claim: %d", res.Status)
	}
}

// US-29, BR-CLM-05 and BR-CLM-06
func TestStalledClaimsExpire(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)
	rival := e.User(access.Member)
	s.lost = fileLost(t, e, s.owner, "exp-lost-0001", nil)
	s.found = fileFound(t, e, s.finder, "exp-found-0001", nil)
	rivalLost := fileLost(t, e, rival, "exp-lost-0002", nil)
	s.receive(t, s.found, s.officer)
	svc := e.Services.Claims
	ctx := context.Background()

	asked := s.claim(t, s.owner.Client, "exp-key-0001", s.lost.ID, http.StatusCreated)
	post(t, s.officer.Client, "/api/v1/claims/"+asked.Code+"/pick-up", nil, http.StatusOK)
	post(t, s.officer.Client, "/api/v1/claims/"+asked.Code+"/request-info", map[string]any{"message": "Sebutkan warna dompet."}, http.StatusOK)
	age(t, e, "info_requested_at", asked.Code, "6 days")
	if exp, _, err := svc.ExpireDue(ctx); err != nil || exp != 0 {
		t.Fatalf("nothing expires on day six: %d %v", exp, err)
	}
	age(t, e, "info_requested_at", asked.Code, "25 hours")
	if exp, _, err := svc.ExpireDue(ctx); err != nil || exp != 1 {
		t.Fatalf("K5: %d %v", exp, err)
	}
	if got := get(t, s.owner.Client, asked.Code); got.Status != "expired" {
		t.Fatalf("K5: %s", got.Status)
	}
	if ev := events(t, s.owner.Client); count(ev, "claim.expired") != 1 {
		t.Fatalf("the claimant is told: %v", ev)
	}

	// K12 with a rival on hold that returns to review (K7, K8)
	kept := s.claim(t, s.owner.Client, "exp-key-0002", s.lost.ID, http.StatusCreated)
	waiting := s.claim(t, rival.Client, "exp-key-0003", rivalLost.ID, http.StatusCreated)
	post(t, s.officer.Client, "/api/v1/claims/"+kept.Code+"/pick-up", nil, http.StatusOK)
	s.passChecks(t, s.officer.Client, kept.Code)
	s.approve(t, s.officer.Client, kept.Code, http.StatusOK)
	if got := get(t, rival.Client, waiting.Code); got.Status != "on_hold" {
		t.Fatalf("K7: %s", got.Status)
	}
	age(t, e, "approved_expires_at", kept.Code, "6 days 1 hour")
	if _, warned, err := svc.ExpireDue(ctx); err != nil || warned != 1 {
		t.Fatalf("the one day warning: %d %v", warned, err)
	}
	if ev := events(t, s.owner.Client); count(ev, "claim.expiring") != 1 {
		t.Fatalf("claim.expiring: %v", ev)
	}
	if _, warned, _ := svc.ExpireDue(ctx); warned != 0 {
		t.Fatalf("the warning is sent once")
	}
	age(t, e, "approved_expires_at", kept.Code, "1 day")
	if exp, _, err := svc.ExpireDue(ctx); err != nil || exp != 1 {
		t.Fatalf("K12: %d %v", exp, err)
	}
	if got := get(t, s.owner.Client, kept.Code); got.Status != "expired" || got.HandoverCode != nil {
		t.Fatalf("K12: %+v", got)
	}
	if got := get(t, rival.Client, waiting.Code); got.Status != "under_review" {
		t.Fatalf("K8: %s", got.Status)
	}
	if ev := events(t, rival.Client); count(ev, "claim.resumed") != 1 {
		t.Fatalf("the held claimant is told: %v", ev)
	}
}

// K13, BR-CLM-09 and the walk in visitor of BR-HND-04
func TestRevokeAndWalkInClaims(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)
	s.lost = fileLost(t, e, s.owner, "rev-lost-0001", nil)
	s.found = fileFound(t, e, s.finder, "rev-found-0001", nil)
	s.receive(t, s.found, s.officer)
	c := s.claim(t, s.owner.Client, "rev-key-0001", s.lost.ID, http.StatusCreated)
	post(t, s.officer.Client, "/api/v1/claims/"+c.Code+"/pick-up", nil, http.StatusOK)
	s.passChecks(t, s.officer.Client, c.Code)
	s.approve(t, s.officer.Client, c.Code, http.StatusOK)
	if v := post(t, s.officer.Client, "/api/v1/claims/"+c.Code+"/revoke", map[string]any{"reason": "Identitas tidak cocok di loket"}, http.StatusOK); v.Status != "under_review" {
		t.Fatalf("K13: %s", v.Status)
	}
	if got := get(t, s.owner.Client, c.Code); got.HandoverCode != nil {
		t.Fatalf("a revoked code is gone")
	}
	if st := statusOf(t, e, "found_reports", s.found.ID); st != "under_verification" {
		t.Fatalf("F7: %s", st)
	}
	post(t, s.officer.Client, "/api/v1/claims/"+c.Code+"/reject", map[string]any{"reason": "duplicate_claim"}, http.StatusOK)

	body := lostBody(e, e.Clock.Now(), nil)
	body["proxy"] = map[string]string{"name": "Pengunjung Loket", "phone": "081200000000"}
	vis := s.officer.Client.JSON(http.MethodPost, "/api/v1/lost-reports", body, "Idempotency-Key", "rev-visitor-lost")
	if vis.Status != http.StatusCreated {
		t.Fatalf("proxy lost report: %d %s", vis.Status, vis.Body)
	}
	var vl lostReport
	vis.Decode(t, &vl)
	walk := s.officer.Client.JSON(http.MethodPost, "/api/v1/claims", claimBody(s.found.ID, vl.ID, s.walletAnswers()), "Idempotency-Key", "rev-visitor-claim")
	if walk.Status != http.StatusCreated {
		t.Fatalf("walk in claim: %d %s", walk.Status, walk.Body)
	}
	var wc claimView
	walk.Decode(t, &wc)
	if wc.VisitorName == nil || *wc.VisitorName != "Pengunjung Loket" {
		t.Fatalf("visitor: %+v", wc)
	}
	post(t, s.officer.Client, "/api/v1/claims/"+wc.Code+"/pick-up", nil, http.StatusOK)
	s.passChecks(t, s.officer.Client, wc.Code)
	v := get(t, s.officer.Client, wc.Code)
	if res := s.officer.Client.JSON(http.MethodPost, "/api/v1/claims/"+wc.Code+"/approve", map[string]any{}, "If-Match", `"`+itoa(v.Version)+`"`); res.Problem(t).Code != "same_officer" {
		t.Fatalf("the creator approving: %d %s", res.Status, res.Body)
	}
	s.approve(t, s.other.Client, wc.Code, http.StatusOK)
	res := s.other.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/handover", map[string]any{
		"claim_id": wc.ID, "recipient_id_type": "ktm", "recipient_id_last4": "9912", "name_confirmed": true,
	})
	if res.Status != http.StatusOK {
		t.Fatalf("walk in handover: %d %s", res.Status, res.Body)
	}
}
