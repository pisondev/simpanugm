//go:build integration

package server_test

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/claims"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

type claimView struct {
	ID           string  `json:"id"`
	Code         string  `json:"code"`
	Status       string  `json:"status"`
	Version      int     `json:"version"`
	HandoverCode *string `json:"handover_code"`
	HandoverLock bool    `json:"handover_locked"`
	DecisionNote *string `json:"decision_note"`
	InternalNote *string `json:"internal_note"`
	VisitorName  *string `json:"visitor_name"`
	Actions      []string
	FoundReport  struct {
		ID            string  `json:"id"`
		Code          string  `json:"code"`
		HiddenDetails *string `json:"hidden_details"`
		StorageSlot   *string `json:"storage_slot"`
		Photos        []struct {
			Variant string `json:"variant"`
		} `json:"photos"`
	} `json:"found_report"`
	LostReport struct {
		ID             string  `json:"id"`
		Code           string  `json:"code"`
		PrivateDetails *string `json:"private_details"`
	} `json:"lost_report"`
	Claimant *struct {
		Email *string `json:"email"`
	} `json:"claimant"`
	Answers []struct{ Key, Question, Answer string }
}

type checkList struct {
	Items []struct {
		ItemKey  string `json:"item_key"`
		Required bool   `json:"required"`
		Result   string `json:"result"`
	} `json:"items"`
	Complete bool `json:"complete"`
}

// a claim scenario: a lost wallet, a found wallet at the library post, an officer of that post and a second one
type scene struct {
	e       *testenv.Env
	owner   testenv.User
	finder  testenv.User
	officer testenv.User
	other   testenv.User
	lost    lostReport
	found   foundReport
}

func newScene(t *testing.T, e *testenv.Env) *scene {
	t.Helper()
	library := e.Post("pos-perpustakaan")
	s := &scene{
		e: e, owner: e.User(access.Member), finder: e.User(access.Member),
		officer: e.User(access.Officer, library), other: e.User(access.Officer, library),
	}
	return s
}

func (s *scene) receive(t *testing.T, f foundReport, officer testenv.User) {
	t.Helper()
	res := officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+f.Code+"/receive", map[string]any{"post_id": s.e.Post("pos-perpustakaan").String(), "storage_slot": "Rak C1"})
	if res.Status != http.StatusOK {
		t.Fatalf("receive %s: %d %s", f.Code, res.Status, res.Body)
	}
}

func (s *scene) walletAnswers() []map[string]string {
	return []map[string]string{
		{"key": "wallet_contents", "answer": "KTM, kartu ATM Mandiri dan foto kucing kecil"},
		{"key": "wallet_private_mark", "answer": "Ada gantungan kunci kucing di resleting dalam"},
	}
}

func claimBody(foundID, lostID string, answers []map[string]string) map[string]any {
	return map[string]any{
		"found_report_id": foundID, "lost_report_id": lostID,
		"statement":          "Dompet ini milik saya, hilang di perpustakaan pusat kemarin sore setelah belajar.",
		"answers":            answers,
		"evidence_media_ids": []string{},
	}
}

func (s *scene) claim(t *testing.T, c *testenv.Client, key string, lostID string, status int) claimView {
	t.Helper()
	res := c.JSON(http.MethodPost, "/api/v1/claims", claimBody(s.found.ID, lostID, s.walletAnswers()), "Idempotency-Key", key)
	if res.Status != status {
		t.Fatalf("create claim: %d %s", res.Status, res.Body)
	}
	var v claimView
	if status < 300 {
		res.Decode(t, &v)
	}
	return v
}

func get(t *testing.T, c *testenv.Client, code string) claimView {
	t.Helper()
	res := c.JSON(http.MethodGet, "/api/v1/claims/"+code, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("get claim %s: %d %s", code, res.Status, res.Body)
	}
	var v claimView
	res.Decode(t, &v)
	return v
}

func post(t *testing.T, c *testenv.Client, path string, body any, status int, headers ...string) claimView {
	t.Helper()
	res := c.JSON(http.MethodPost, path, body, headers...)
	if res.Status != status {
		t.Fatalf("POST %s: want %d, got %d %s", path, status, res.Status, res.Body)
	}
	var v claimView
	if status < 300 && len(res.Body) > 0 {
		res.Decode(t, &v)
	}
	return v
}

func (s *scene) passChecks(t *testing.T, c *testenv.Client, code string) {
	t.Helper()
	var list checkList
	c.JSON(http.MethodGet, "/api/v1/claims/"+code+"/checks", nil).Decode(t, &list)
	items := []map[string]string{}
	for _, it := range list.Items {
		if it.Required {
			items = append(items, map[string]string{"item_key": it.ItemKey, "result": "pass"})
		}
	}
	if res := c.JSON(http.MethodPut, "/api/v1/claims/"+code+"/checks", map[string]any{"items": items}); res.Status != http.StatusOK {
		t.Fatalf("put checks: %d %s", res.Status, res.Body)
	}
}

func statusOf(t *testing.T, e *testenv.Env, table, id string) string {
	t.Helper()
	var s string
	if err := e.Pool.QueryRow(context.Background(), "SELECT status FROM "+table+" WHERE id = $1", id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func (s *scene) approve(t *testing.T, c *testenv.Client, code string, status int) claimView {
	t.Helper()
	v := get(t, c, code)
	return post(t, c, "/api/v1/claims/"+code+"/approve", map[string]any{"decision_note": "Detail sesuai."}, status, "If-Match", `"`+itoa(v.Version)+`"`)
}

func itoa(n int) string {
	return strings.TrimSpace(strings.Repeat(" ", 0) + fmtInt(n))
}

func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

var sixDigit = regexp.MustCompile(`^[0-9]{6}$`)

// US-25 to US-28, US-31 and FR-CUST-04: the whole way from a claim to the item in the owner's hands
func TestClaimGoesFromSubmissionToHandover(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)
	rival := e.User(access.Member)
	s.lost = fileLost(t, e, s.owner, "claim-lost-0001", nil)
	s.found = fileFound(t, e, s.finder, "claim-found-0001", nil)
	rivalLost := fileLost(t, e, rival, "claim-lost-0002", nil)

	// BR-CLM-07: the item is still with its finder, a claim may be submitted but not approved
	early := s.claim(t, s.owner.Client, "claim-key-0001", s.lost.ID, http.StatusCreated)
	if early.Status != "submitted" || !regexp.MustCompile(`^CLM-[0-9]{4}-[0-9]{4}$`).MatchString(early.Code) {
		t.Fatalf("created claim: %+v", early)
	}
	if replay := s.claim(t, s.owner.Client, "claim-key-0001", s.lost.ID, http.StatusCreated); replay.ID != early.ID {
		t.Fatalf("an idempotent replay returns the same claim: %s vs %s", replay.ID, early.ID)
	}
	if st := statusOf(t, e, "lost_reports", s.lost.ID); st != "under_verification" {
		t.Fatalf("L4: %s", st)
	}
	if st := statusOf(t, e, "found_reports", s.found.ID); st != "under_verification" {
		t.Fatalf("F4: %s", st)
	}
	if ev := events(t, s.owner.Client); count(ev, "claim.submitted_ack") != 1 {
		t.Fatalf("owner acknowledgement: %v", ev)
	}
	if ev := events(t, s.officer.Client); count(ev, "claim.submitted") != 0 {
		// the item is not at a post yet, so the officers of its intended post hear about it
		t.Logf("officer events before receipt: %v", ev)
	}
	post(t, s.officer.Client, "/api/v1/claims/"+early.Code+"/pick-up", nil, http.StatusOK)
	if res := s.officer.Client.JSON(http.MethodPost, "/api/v1/claims/"+early.Code+"/approve", map[string]any{}, "If-Match", "2"); res.Problem(t).Code != "custody_not_received" {
		t.Fatalf("approve while with the finder: %d %s", res.Status, res.Body)
	}

	s.receive(t, s.found, s.officer)
	// a second member claims too, so the first approval puts it on hold
	rivalClaim := s.claim(t, rival.Client, "claim-key-0002", rivalLost.ID, http.StatusCreated)

	// the owner and the officer see different things (BR-VIS, FR-CLAIM-12)
	own, off := get(t, s.owner.Client, early.Code), get(t, s.officer.Client, early.Code)
	if own.FoundReport.HiddenDetails != nil || own.InternalNote != nil || own.HandoverCode != nil {
		t.Fatalf("claimant view leaks: %+v", own)
	}
	if off.FoundReport.HiddenDetails == nil || off.FoundReport.StorageSlot == nil || off.Claimant == nil || off.Claimant.Email == nil || off.HandoverCode != nil {
		t.Fatalf("officer view: %+v", off)
	}
	if len(off.Answers) != 2 || off.Answers[0].Question == "" {
		t.Fatalf("answers are copied with their question text: %+v", off.Answers)
	}
	if res := rival.Client.JSON(http.MethodGet, "/api/v1/claims/"+early.Code, nil); res.Status != http.StatusNotFound {
		t.Fatalf("another member reads a claim: %d", res.Status)
	}

	// BR-CLM-10: approval waits for the checklist
	var open checkList
	s.officer.Client.JSON(http.MethodGet, "/api/v1/claims/"+early.Code+"/checks", nil).Decode(t, &open)
	if open.Complete || len(open.Items) < 3 {
		t.Fatalf("checklist: %+v", open)
	}
	if res := s.officer.Client.JSON(http.MethodPost, "/api/v1/claims/"+early.Code+"/approve", map[string]any{}, "If-Match", `"`+itoa(get(t, s.officer.Client, early.Code).Version)+`"`); res.Problem(t).Code != "checklist_incomplete" {
		t.Fatalf("approve with an open checklist: %d %s", res.Status, res.Body)
	}
	if res := s.officer.Client.JSON(http.MethodPut, "/api/v1/claims/"+early.Code+"/checks", map[string]any{"items": []map[string]string{{"item_key": open.Items[0].ItemKey, "result": "not_applicable"}}}); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("not applicable needs a note: %d", res.Status)
	}
	s.passChecks(t, s.officer.Client, early.Code)

	// K6 with M5, L6 and F6, and the rival is put on hold (K7)
	approved := s.approve(t, s.officer.Client, early.Code, http.StatusOK)
	if approved.Status != "approved" || approved.HandoverCode != nil {
		t.Fatalf("approved: %+v", approved)
	}
	if st := statusOf(t, e, "lost_reports", s.lost.ID); st != "claimed" {
		t.Fatalf("L6: %s", st)
	}
	if st := statusOf(t, e, "found_reports", s.found.ID); st != "claimed" {
		t.Fatalf("F6: %s", st)
	}
	if got := get(t, rival.Client, rivalClaim.Code); got.Status != "on_hold" {
		t.Fatalf("K7: %s", got.Status)
	}
	code := get(t, s.owner.Client, early.Code).HandoverCode
	if code == nil || !sixDigit.MatchString(*code) {
		t.Fatalf("the claimant sees a six digit code: %v", code)
	}
	if again := get(t, s.owner.Client, early.Code).HandoverCode; again == nil || *again != *code {
		t.Fatalf("the code is stable until regenerated")
	}
	for _, em := range []string{s.owner.Email} {
		if mail := e.LastEmail(em); strings.Contains(mail.Text, *code) || strings.Contains(mail.HTML, *code) {
			t.Fatalf("FR-CLAIM-12: the approval email carries the code")
		}
	}
	var stored []byte
	if err := e.Pool.QueryRow(context.Background(), "SELECT handover_nonce FROM claims WHERE id = $1", early.ID).Scan(&stored); err != nil || strings.Contains(string(stored), *code) {
		t.Fatalf("the code itself is not stored: %v", err)
	}

	handover := func(c *testenv.Client, code string, status int) {
		t.Helper()
		body := map[string]any{"claim_id": early.ID, "recipient_id_type": "ktm", "recipient_id_last4": "4417", "name_confirmed": true}
		if code != "" {
			body["code"] = code
		}
		if res := c.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/handover", body); res.Status != status {
			t.Fatalf("handover with %q: want %d, got %d %s", code, status, res.Status, res.Body)
		}
	}
	// BR-HND-02: five wrong codes lock it, regenerating unlocks it with a new code
	wrong := "000000"
	if *code == wrong {
		wrong = "111111"
	}
	for i := 0; i < 4; i++ {
		handover(s.officer.Client, wrong, http.StatusUnprocessableEntity)
	}
	if res := s.officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/handover", map[string]any{
		"claim_id": early.ID, "code": wrong, "recipient_id_type": "ktm", "recipient_id_last4": "4417", "name_confirmed": true,
	}); res.Problem(t).Code != "handover_locked" {
		t.Fatalf("fifth wrong code: %d %s", res.Status, res.Body)
	}
	handover(s.officer.Client, *code, http.StatusUnprocessableEntity)
	if !get(t, s.owner.Client, early.Code).HandoverLock {
		t.Fatalf("the claimant sees that the code is locked")
	}
	if res := s.owner.Client.JSON(http.MethodPost, "/api/v1/claims/"+early.Code+"/handover-code", nil); res.Status != http.StatusForbidden {
		t.Fatalf("a member cannot regenerate: %d", res.Status)
	}
	post(t, s.officer.Client, "/api/v1/claims/"+early.Code+"/handover-code", nil, http.StatusOK)
	fresh := get(t, s.owner.Client, early.Code)
	if fresh.HandoverCode == nil || *fresh.HandoverCode == *code || fresh.HandoverLock {
		t.Fatalf("a new code replaces the old one: %+v", fresh)
	}
	handover(s.officer.Client, *code, http.StatusUnprocessableEntity)

	// the name confirmation of BR-HND-03 is mandatory
	if res := s.officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/handover", map[string]any{
		"claim_id": early.ID, "code": *fresh.HandoverCode, "recipient_id_type": "ktm", "recipient_id_last4": "4417", "name_confirmed": false,
	}); !fieldError(t, res, "name_confirmed") {
		t.Fatalf("handover without the name check: %d %s", res.Status, res.Body)
	}
	// an officer of another post cannot hand over (BR-CUS-03)
	stranger := e.User(access.Officer, e.Post("pos-saintek"))
	if res := stranger.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/handover", map[string]any{
		"claim_id": early.ID, "code": *fresh.HandoverCode, "recipient_id_type": "ktm", "recipient_id_last4": "4417", "name_confirmed": true,
	}); res.Status != http.StatusForbidden {
		t.Fatalf("handover from another post: %d", res.Status)
	}
	handover(s.officer.Client, *fresh.HandoverCode, http.StatusOK)

	done := get(t, s.owner.Client, early.Code)
	if done.Status != "handed_over" || done.HandoverCode != nil {
		t.Fatalf("handed over: %+v", done)
	}
	var custody, foundStatus, lostStatus string
	_ = e.Pool.QueryRow(context.Background(), "SELECT custody_status, status FROM found_reports WHERE id = $1", s.found.ID).Scan(&custody, &foundStatus)
	lostStatus = statusOf(t, e, "lost_reports", s.lost.ID)
	if custody != "released" || foundStatus != "returned" || lostStatus != "returned" {
		t.Fatalf("FR-CUST-04: custody %s, found %s, lost %s", custody, foundStatus, lostStatus)
	}
	if got := get(t, rival.Client, rivalClaim.Code); got.Status != "rejected" {
		t.Fatalf("K9: %s", got.Status)
	}
	if st := statusOf(t, e, "lost_reports", rivalLost.ID); st != "reported" && st != "potential_match" {
		t.Fatalf("the rival lost report is open again: %s", st)
	}
	for who, c := range map[string]*testenv.Client{"owner": s.owner.Client, "finder": s.finder.Client, "rival": rival.Client} {
		ev := events(t, c)
		want := map[string]string{"owner": "claim.handed_over", "finder": "found.returned_thanks", "rival": "claim.rejected"}[who]
		if count(ev, want) == 0 {
			t.Fatalf("%s should have been told with %s: %v", who, want, ev)
		}
	}
	if ev := events(t, s.owner.Client); count(ev, "lost.confirm_receipt") != 1 {
		t.Fatalf("US-31 criterion 3: %v", ev)
	}
	// the reversible history is kept
	var history int
	_ = e.Pool.QueryRow(context.Background(), "SELECT count(*) FROM status_history WHERE entity_type = 'claim' AND entity_id = $1", early.ID).Scan(&history)
	if history < 4 {
		t.Fatalf("claim history rows: %d", history)
	}
	if res := s.owner.Client.JSON(http.MethodPost, "/api/v1/claims/"+early.Code+"/withdraw", nil); res.Status != http.StatusConflict {
		t.Fatalf("a finished claim cannot be withdrawn: %d", res.Status)
	}
}

var _ = uuid.Nil
var _ = time.Now
var _ = claims.Open
