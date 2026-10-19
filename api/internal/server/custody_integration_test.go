//go:build integration

package server_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

func ageTable(t *testing.T, e *testenv.Env, table, column, code, by string) {
	t.Helper()
	if _, err := e.Pool.Exec(context.Background(), "UPDATE "+table+" SET "+column+" = "+column+" - $2::interval WHERE code = $1", code, by); err != nil {
		t.Fatal(err)
	}
}

// C4 and BR-CUS-03
func TestCustodyTransfer(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)
	s.found = fileFound(t, e, s.finder, "tr-found-0001", nil)
	s.receive(t, s.found, s.officer)

	saintekPost := e.Post("pos-saintek")
	stranger := e.User(access.Officer, saintekPost)

	// stranger cannot transfer
	res := stranger.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/transfer", map[string]any{
		"to_post_id": saintekPost.String(),
	}, "If-Match", `"`+itoa(s.found.Version+1)+`"`)
	if res.Status != http.StatusForbidden {
		t.Fatalf("stranger transferred: %d %s", res.Status, res.Body)
	}

	// holding officer transfers
	var cur foundReport
	s.officer.Client.JSON(http.MethodGet, "/api/v1/found-reports/"+s.found.Code, nil).Decode(t, &cur)
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/transfer", map[string]any{
		"to_post_id": saintekPost.String(),
		"note":       "Dipindahkan ke pos Fakultas Teknik",
	}, "If-Match", `"`+itoa(cur.Version)+`"`)
	if res.Status != http.StatusOK {
		t.Fatalf("transfer: %d %s", res.Status, res.Body)
	}

	var holdingPostID uuid.UUID
	if err := e.Pool.QueryRow(context.Background(), "SELECT holding_post_id FROM found_reports WHERE code = $1", s.found.Code).Scan(&holdingPostID); err != nil {
		t.Fatal(err)
	}
	if holdingPostID != saintekPost {
		t.Fatalf("holding post: got %s, want %s", holdingPostID, saintekPost)
	}
}

// BR-CUS-06
func TestCustodyExtendRetention(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)
	s.found = fileFound(t, e, s.finder, "ext-found-0001", nil)
	s.receive(t, s.found, s.officer)

	var cur foundReport
	s.officer.Client.JSON(http.MethodGet, "/api/v1/found-reports/"+s.found.Code, nil).Decode(t, &cur)

	// invalid days
	res := s.officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/extend-retention", map[string]any{
		"days":   45,
		"reason": "Perlu waktu lebih lama",
	}, "If-Match", `"`+itoa(cur.Version)+`"`)
	if res.Status != http.StatusUnprocessableEntity && res.Status != http.StatusBadRequest {
		t.Fatalf("extend 45 days: %d", res.Status)
	}

	// extend 14 days
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/extend-retention", map[string]any{
		"days":   14,
		"reason": "Menunggu konfirmasi pemilik barang",
	}, "If-Match", `"`+itoa(cur.Version)+`"`)
	if res.Status != http.StatusOK {
		t.Fatalf("extend retention: %d %s", res.Status, res.Body)
	}

	var after foundReport
	res.Decode(t, &after)

	// extend again should fail
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/extend-retention", map[string]any{
		"days":   7,
		"reason": "Perpanjangan kedua kali",
	}, "If-Match", `"`+itoa(after.Version)+`"`)
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("extend second time: %d %s", res.Status, res.Body)
	}
}

// C6, F11, F12, BR-CUS-04
func TestCustodyDisposeAndRevert(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)
	s.found = fileFound(t, e, s.finder, "disp-found-0001", nil)
	s.receive(t, s.found, s.officer)

	var cur foundReport
	s.officer.Client.JSON(http.MethodGet, "/api/v1/found-reports/"+s.found.Code, nil).Decode(t, &cur)

	// cannot dispose while reported
	res := s.officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/dispose", map[string]any{
		"kind": "donated",
		"note": "Disumbangkan ke panti asuhan",
	}, "If-Match", `"`+itoa(cur.Version)+`"`)
	if res.Status != http.StatusConflict {
		t.Fatalf("dispose while reported: %d %s", res.Status, res.Body)
	}

	// move to unclaimed via DB
	ageTable(t, e, "found_reports", "retention_due_at", s.found.Code, "90 days")
	unclaimed, _, _, _, err := e.Services.Custody.SweepDue(context.Background())
	if err != nil || unclaimed < 1 {
		t.Fatalf("sweep unclaimed: %d %v", unclaimed, err)
	}

	// test revert unclaimed (F11)
	s.officer.Client.JSON(http.MethodGet, "/api/v1/found-reports/"+s.found.Code, nil).Decode(t, &cur)
	if cur.Status != "unclaimed" {
		t.Fatalf("expected unclaimed, got %s", cur.Status)
	}
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/revert-unclaimed", nil, "If-Match", `"`+itoa(cur.Version)+`"`)
	if res.Status != http.StatusOK {
		t.Fatalf("revert unclaimed: %d %s", res.Status, res.Body)
	}
	res.Decode(t, &cur)
	if cur.Status != "reported" {
		t.Fatalf("after revert: %s", cur.Status)
	}

	// move to unclaimed again for dispose
	ageTable(t, e, "found_reports", "retention_due_at", s.found.Code, "90 days")
	if _, _, _, _, err := e.Services.Custody.SweepDue(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	s.officer.Client.JSON(http.MethodGet, "/api/v1/found-reports/"+s.found.Code, nil).Decode(t, &cur)

	// dispose regular item
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+s.found.Code+"/dispose", map[string]any{
		"kind": "donated",
		"note": "Disumbangkan ke panti asuhan",
	}, "If-Match", `"`+itoa(cur.Version)+`"`)
	if res.Status != http.StatusOK {
		t.Fatalf("dispose regular: %d %s", res.Status, res.Body)
	}
	res.Decode(t, &cur)
	if cur.Status != "closed" || cur.CustodyStatus != "disposed" {
		t.Fatalf("after dispose: status %s custody %s", cur.Status, cur.CustodyStatus)
	}

	// BR-CUS-04: identity item (KTM) must be transferred to unit
	ktmCat := e.Category("ktm")
	ktmFound := fileFound(t, e, s.finder, "ktm-found-0001", func(b map[string]any) {
		b["category_id"] = ktmCat.String()
		b["item_name"] = "KTM Mahasiswa Teknik"
	})
	s.receive(t, ktmFound, s.officer)
	ageTable(t, e, "found_reports", "retention_due_at", ktmFound.Code, "90 days")
	if _, _, _, _, err := e.Services.Custody.SweepDue(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	var ktmCur foundReport
	s.officer.Client.JSON(http.MethodGet, "/api/v1/found-reports/"+ktmFound.Code, nil).Decode(t, &ktmCur)

	// donated fails for identity
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+ktmFound.Code+"/dispose", map[string]any{
		"kind": "donated",
		"note": "Disumbangkan",
	}, "If-Match", `"`+itoa(ktmCur.Version)+`"`)
	if res.Status != http.StatusUnprocessableEntity && res.Status != http.StatusBadRequest {
		t.Fatalf("dispose identity as donated: %d", res.Status)
	}

	// transferred_to_unit succeeds for identity
	res = s.officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+ktmFound.Code+"/dispose", map[string]any{
		"kind":      "transferred_to_unit",
		"recipient": "Kantor Akademik FT UGM",
		"note":      "Diserahkan ke bagian akademik fakultas",
	}, "If-Match", `"`+itoa(ktmCur.Version)+`"`)
	if res.Status != http.StatusOK {
		t.Fatalf("dispose identity transferred_to_unit: %d %s", res.Status, res.Body)
	}
}

// US-11, BR-LST-02, BR-LST-03, BR-LST-04, L9, L11, L12
func TestLostReportReopenAndKeepOpenAndConfirm(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)
	s.lost = fileLost(t, e, s.owner, "lst-life-0001", nil)

	// keep open
	res := s.owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports/"+s.lost.Code+"/keep-open", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("keep open: %d %s", res.Status, res.Body)
	}

	// stale reminder after 60 days
	ageTable(t, e, "lost_reports", "last_activity_at", s.lost.Code, "61 days")
	reminded, _, err := e.Services.Reports.SweepDue(context.Background())
	if err != nil || reminded < 1 {
		t.Fatalf("sweep stale: %d %v", reminded, err)
	}
	if ev := events(t, s.owner.Client); count(ev, "lost.stale_reminder") != 1 {
		t.Fatalf("owner told stale: %v", ev)
	}

	// expiry after 14 days without answer
	ageTable(t, e, "lost_reports", "stale_reminded_at", s.lost.Code, "15 days")
	_, expired, err := e.Services.Reports.SweepDue(context.Background())
	if err != nil || expired < 1 {
		t.Fatalf("sweep expiry: %d %v", expired, err)
	}
	st, _ := lostStatus(t, s.owner.Client, s.lost.Code)
	if st != "closed" {
		t.Fatalf("status after expire: %s", st)
	}

	// reopen within 30 days (L12)
	res = s.owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports/"+s.lost.Code+"/reopen", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("reopen lost: %d %s", res.Status, res.Body)
	}
	st, _ = lostStatus(t, s.owner.Client, s.lost.Code)
	if st != "reported" {
		t.Fatalf("status after reopen: %s", st)
	}

	// reopen past 30 days fails
	s.owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports/"+s.lost.Code+"/close", map[string]any{"reason": "found_by_owner"})
	ageTable(t, e, "lost_reports", "closed_at", s.lost.Code, "31 days")
	res = s.owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports/"+s.lost.Code+"/reopen", nil)
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("reopen past 30 days: %d %s", res.Status, res.Body)
	}

	// confirm receipt (L9)
	confirmLost := fileLost(t, e, s.owner, "lst-conf-0001", nil)
	if _, err := e.Pool.Exec(context.Background(), "UPDATE lost_reports SET status = 'returned' WHERE id = $1", confirmLost.ID); err != nil {
		t.Fatal(err)
	}
	res = s.owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports/"+confirmLost.Code+"/confirm-receipt", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("confirm receipt: %d %s", res.Status, res.Body)
	}
	st, _ = lostStatus(t, s.owner.Client, confirmLost.Code)
	if st != "closed" {
		t.Fatalf("status after confirm receipt: %s", st)
	}
}

// US-16, BR-FND-03, BR-CUS-05
func TestFinderCustodyRemindersAndRetentionWarning(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)

	// found item of sensitivity normal kept with finder
	flashdiskCat := e.Category("flashdisk")
	finderFound := fileFound(t, e, s.finder, "rem-found-0001", func(b map[string]any) {
		b["category_id"] = flashdiskCat.String()
		b["item_name"] = "Flashdisk Kingston Hitam"
		b["custody"] = map[string]any{"mode": "keep"}
	})

	// 24h reminder
	ageTable(t, e, "found_reports", "created_at", finderFound.Code, "25 hours")
	_, _, _, reminders, err := e.Services.Custody.SweepDue(context.Background())
	if err != nil || reminders < 1 {
		t.Fatalf("24h sweep: %d %v", reminders, err)
	}
	if ev := events(t, s.finder.Client); count(ev, "found.custody_reminder_24h") != 1 {
		t.Fatalf("finder 24h event: %v", ev)
	}

	// 48h reminder to finder
	ageTable(t, e, "found_reports", "created_at", finderFound.Code, "24 hours")
	_, _, _, reminders, err = e.Services.Custody.SweepDue(context.Background())
	if err != nil || reminders < 1 {
		t.Fatalf("48h sweep: %d %v", reminders, err)
	}
	if ev := events(t, s.finder.Client); count(ev, "found.custody_reminder_48h") != 1 {
		t.Fatalf("finder 48h event: %v", ev)
	}

	// 72h overdue flag
	ageTable(t, e, "found_reports", "created_at", finderFound.Code, "24 hours")
	_, _, overdue, _, err := e.Services.Custody.SweepDue(context.Background())
	if err != nil || overdue < 1 {
		t.Fatalf("72h sweep: %d %v", overdue, err)
	}

	// BR-CUS-05 retention 7-day warning
	inCustody := fileFound(t, e, s.finder, "warn-found-0001", nil)
	s.receive(t, inCustody, s.officer)
	ageTable(t, e, "found_reports", "retention_due_at", inCustody.Code, "55 days")
	_, warned, _, _, err := e.Services.Custody.SweepDue(context.Background())
	if err != nil || warned < 1 {
		t.Fatalf("retention warning: %d %v", warned, err)
	}
	if ev := events(t, s.officer.Client); count(ev, "custody.retention_warning") < 1 {
		t.Fatalf("officer retention warning event: %v", ev)
	}
}
