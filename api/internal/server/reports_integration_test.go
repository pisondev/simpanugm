//go:build integration

package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

type lostReport struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Status         string `json:"status"`
	Version        int    `json:"version"`
	PrivateDetails *string
	Reporter       *struct {
		FullName string `json:"full_name"`
	} `json:"reporter"`
	Places []struct {
		Location *struct{ Name string } `json:"location"`
	} `json:"places"`
	Photos []struct {
		MediaID string `json:"media_id"`
	} `json:"photos"`
	Editable bool `json:"editable"`
}

type foundReport struct {
	ID             string  `json:"id"`
	Code           string  `json:"code"`
	Status         string  `json:"status"`
	CustodyStatus  string  `json:"custody_status"`
	Version        int     `json:"version"`
	RetentionDueAt *string `json:"retention_due_at"`
	HoldingPost    *struct {
		Name string `json:"name"`
	} `json:"holding_post"`
	Editable bool `json:"editable"`
}

func lostBody(e *testenv.Env, now time.Time, media []uuid.UUID) map[string]any {
	if media == nil {
		media = []uuid.UUID{}
	}
	return map[string]any{
		"category_id":      e.Category("dompet").String(),
		"item_name":        "Dompet kulit cokelat",
		"description":      "Dompet lipat dua warna cokelat, isi KTM dan kartu ATM.",
		"private_details":  "Gantungan kunci kecil bentuk kucing",
		"colours":          []string{"cokelat"},
		"lost_from":        now.Add(-5 * time.Hour).Format(time.RFC3339),
		"lost_until":       now.Add(-2 * time.Hour).Format(time.RFC3339),
		"location_unknown": false,
		"places":           []map[string]string{{"location_id": e.Location("perpustakaan-pusat").String()}, {"location_id": e.Location("pusat-jajanan-lembah").String()}},
		"media_ids":        media,
	}
}

func foundBody(e *testenv.Env, now time.Time, category string, media []uuid.UUID, custody map[string]any) map[string]any {
	return map[string]any{
		"category_id":       e.Category(category).String(),
		"item_name":         "Barang temuan uji",
		"description":       "Ditemukan di meja baca lantai dua, kartu 4111 1111 1111 1111.",
		"hidden_details":    "Isi: KTM atas nama Rani",
		"colours":           []string{"hitam"},
		"found_at":          now.Add(-1 * time.Hour).Format(time.RFC3339),
		"found_location_id": e.Location("perpustakaan-pusat").String(),
		"media_ids":         media,
		"custody":           custody,
	}
}

var lostCode = regexp.MustCompile(`^LST-[0-9]{4}-[0-9]{4,5}$`)

func TestLostReportLifecycle(t *testing.T) {
	e := testenv.New(t)
	owner := e.User(access.Member)
	stranger := e.User(access.Member)
	officer := e.User(access.Officer, e.Post("pos-pusat"))
	now := e.Clock.Now()
	body := lostBody(e, now, []uuid.UUID{owner.Client.UploadID("lost_report")})

	res := owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports", body, "Idempotency-Key", "lost-key-0001")
	if res.Status != http.StatusCreated {
		t.Fatalf("create: %d %s", res.Status, res.Body)
	}
	var r lostReport
	res.Decode(t, &r)
	if !lostCode.MatchString(r.Code) || r.Status != "reported" || len(r.Places) != 2 || len(r.Photos) != 1 || !r.Editable || r.Reporter != nil {
		t.Fatalf("created report: %+v", r)
	}

	var replay lostReport
	res = owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports", body, "Idempotency-Key", "lost-key-0001")
	res.Decode(t, &replay)
	if res.Status != http.StatusCreated || replay.ID != r.ID {
		t.Fatalf("replay must return the same report: %d %s", res.Status, res.Body)
	}
	body["item_name"] = "Dompet lain"
	if res := owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports", body, "Idempotency-Key", "lost-key-0001"); res.Problem(t).Code != "idempotency_key_reused" {
		t.Fatalf("reused key with another body: %d %s", res.Status, res.Body)
	}

	if res := stranger.Client.JSON(http.MethodGet, "/api/v1/lost-reports/"+r.Code, nil); res.Status != http.StatusNotFound {
		t.Fatalf("stranger read: %d", res.Status)
	}
	var seen lostReport
	officer.Client.JSON(http.MethodGet, "/api/v1/lost-reports/"+r.Code, nil).Decode(t, &seen)
	if seen.Reporter == nil || seen.Reporter.FullName == "" {
		t.Fatal("officer must see the reporter")
	}
	var accessLogs int
	_ = e.Pool.QueryRow(context.Background(), "SELECT count(*) FROM audit_logs WHERE action = 'private.read' AND entity_id = $1", r.ID).Scan(&accessLogs)
	if accessLogs != 1 {
		t.Fatalf("private.read entries = %d, want 1 (FR-AUDIT-03)", accessLogs)
	}

	patch := map[string]any{"description": "Dompet lipat dua warna cokelat tua, isi KTM dan kartu ATM BNI."}
	if res := owner.Client.JSON(http.MethodPatch, "/api/v1/lost-reports/"+r.ID, patch, "If-Match", fmt.Sprintf(`"%d"`, r.Version+5)); res.Problem(t).Code != "stale_version" {
		t.Fatalf("stale version: %d %s", res.Status, res.Body)
	}
	var updated lostReport
	res = owner.Client.JSON(http.MethodPatch, "/api/v1/lost-reports/"+r.ID, patch, "If-Match", fmt.Sprint(r.Version))
	res.Decode(t, &updated)
	if res.Status != http.StatusOK || updated.Version != r.Version+1 {
		t.Fatalf("update: %d %s", res.Status, res.Body)
	}

	var closed lostReport
	res = owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports/"+r.ID+"/close", map[string]string{"reason": "found_by_owner"})
	res.Decode(t, &closed)
	if res.Status != http.StatusOK || closed.Status != "closed" || closed.Editable {
		t.Fatalf("close: %d %s", res.Status, res.Body)
	}
	if res := owner.Client.JSON(http.MethodPatch, "/api/v1/lost-reports/"+r.ID, patch, "If-Match", fmt.Sprint(closed.Version)); res.Problem(t).Code != "invalid_transition" {
		t.Fatalf("edit after close: %d %s", res.Status, res.Body)
	}
	var history struct {
		Items []struct {
			Transition string `json:"transition"`
			ToStatus   string `json:"to_status"`
		} `json:"items"`
	}
	owner.Client.JSON(http.MethodGet, "/api/v1/lost-reports/"+r.ID+"/history", nil).Decode(t, &history)
	if len(history.Items) != 2 || history.Items[0].Transition != "L1" || history.Items[1].Transition != "L10" {
		t.Fatalf("history: %+v", history.Items)
	}
	var list struct {
		Items []struct{ ID string } `json:"items"`
	}
	stranger.Client.JSON(http.MethodGet, "/api/v1/lost-reports", nil).Decode(t, &list)
	for _, it := range list.Items {
		if it.ID == r.ID {
			t.Fatal("a member list must only contain own reports")
		}
	}
}

func TestLostReportValidation(t *testing.T) {
	e := testenv.New(t)
	owner := e.User(access.Member)
	now := e.Clock.Now()
	body := lostBody(e, now, []uuid.UUID{})
	body["lost_from"] = now.Add(-9 * 24 * time.Hour).Format(time.RFC3339)
	body["colours"] = []string{"jingga muda"}
	body["places"] = []map[string]string{}
	res := owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports", body, "Idempotency-Key", "lost-key-invalid-1")
	got := map[string]string{}
	for _, f := range res.Problem(t).Errors {
		got[f.Field] = f.Code
	}
	if got["lost_until"] != "window_too_long" || got["colours"] != "unknown" || got["places"] != "count" {
		t.Fatalf("validation fields: %v %s", got, res.Body)
	}
}

func TestOpenLostReportLimit(t *testing.T) {
	e := testenv.New(t)
	owner := e.User(access.Member)
	now := e.Clock.Now()
	for i := 0; i < 5; i++ {
		if res := owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports", lostBody(e, now, nil), "Idempotency-Key", fmt.Sprintf("limit-key-%04d", i)); res.Status != http.StatusCreated {
			t.Fatalf("report %d: %d %s", i, res.Status, res.Body)
		}
	}
	res := owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports", lostBody(e, now, nil), "Idempotency-Key", "limit-key-9999")
	if res.Problem(t).Code != "report_limit_reached" {
		t.Fatalf("sixth open report (BR-LST-07): %d %s", res.Status, res.Body)
	}
}

func TestFoundReportCustodyFlow(t *testing.T) {
	e := testenv.New(t)
	finder := e.User(access.Member)
	library := e.Post("pos-perpustakaan")
	elsewhere := e.User(access.Officer, e.Post("pos-agro"))
	officer := e.User(access.Officer, library)
	now := e.Clock.Now()

	keep := foundBody(e, now, "dompet", []uuid.UUID{finder.Client.UploadID("found_report")}, map[string]any{"mode": "keep"})
	res := finder.Client.JSON(http.MethodPost, "/api/v1/found-reports", keep, "Idempotency-Key", "found-key-keep-1")
	if p := res.Problem(t); len(p.Errors) == 0 || p.Errors[0].Code != "must_handover" {
		t.Fatalf("valuable item kept by finder (BR-FND-02): %d %s", res.Status, res.Body)
	}

	body := foundBody(e, now, "dompet", []uuid.UUID{finder.Client.UploadID("found_report")}, map[string]any{"mode": "handover", "intended_post_id": library.String()})
	res = finder.Client.JSON(http.MethodPost, "/api/v1/found-reports", body, "Idempotency-Key", "found-key-ok-1")
	if res.Status != http.StatusCreated {
		t.Fatalf("create found: %d %s", res.Status, res.Body)
	}
	var f foundReport
	res.Decode(t, &f)
	var full map[string]any
	res.Decode(t, &full)
	if f.CustodyStatus != "with_finder" || full["description"] != "Ditemukan di meja baca lantai dua, kartu ************1111." {
		t.Fatalf("found report: %+v description %v", f, full["description"])
	}

	pub := e.Client().JSON(http.MethodGet, "/api/v1/found-reports/public?limit=100", nil)
	if pub.Status != http.StatusOK {
		t.Fatalf("anonymous public listing: %d %s", pub.Status, pub.Body)
	}
	var listing struct {
		Items []map[string]any `json:"items"`
	}
	pub.Decode(t, &listing)
	var found bool
	for _, it := range listing.Items {
		if it["id"] == f.ID {
			found = true
			for _, forbidden := range []string{"description", "hidden_details", "photos", "finder", "code"} {
				if _, ok := it[forbidden]; ok {
					t.Fatalf("public listing leaks %s (FR-FOUND-12)", forbidden)
				}
			}
		}
	}
	if !found {
		t.Fatal("open found report missing from the public listing")
	}

	receive := map[string]any{"post_id": library.String(), "storage_slot": "Rak B-3", "condition_note": "Utuh"}
	if res := elsewhere.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+f.Code+"/receive", receive); res.Problem(t).Code != "post_not_assigned" {
		t.Fatalf("officer of another post (BR-CUS-03): %d %s", res.Status, res.Body)
	}
	var received foundReport
	res = officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+f.Code+"/receive", receive)
	res.Decode(t, &received)
	if res.Status != http.StatusOK || received.CustodyStatus != "in_custody" || received.HoldingPost == nil || received.RetentionDueAt == nil {
		t.Fatalf("receive: %d %s", res.Status, res.Body)
	}
	due, _ := time.Parse(time.RFC3339, *received.RetentionDueAt)
	if days := due.Sub(now).Hours() / 24; days < 59.9 || days > 60.1 {
		t.Fatalf("valuable retention is 60 days (BR-CUS-01), got %.1f", days)
	}
	if res := officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+f.Code+"/receive", receive); res.Problem(t).Code != "invalid_transition" {
		t.Fatalf("second receive: %d %s", res.Status, res.Body)
	}

	var inv struct {
		Items []struct {
			ID          string `json:"id"`
			StorageSlot string `json:"storage_slot"`
		} `json:"items"`
	}
	officer.Client.JSON(http.MethodGet, "/api/v1/posts/"+library.String()+"/inventory", nil).Decode(t, &inv)
	var inInventory bool
	for _, it := range inv.Items {
		inInventory = inInventory || (it.ID == f.ID && it.StorageSlot == "Rak B-3")
	}
	if !inInventory {
		t.Fatal("received item missing from the post inventory")
	}

	var mine foundReport
	finder.Client.JSON(http.MethodGet, "/api/v1/found-reports/"+f.ID, nil).Decode(t, &mine)
	if mine.Editable {
		t.Fatal("finder must not edit once the item is in custody")
	}
	if res := finder.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+f.ID+"/close", map[string]string{"reason": "returned_directly", "note": "Sudah dikembalikan"}); res.Problem(t).Code != "invalid_transition" {
		t.Fatalf("finder closing an item in custody: %d %s", res.Status, res.Body)
	}
	var history struct {
		Items []struct{ Transition string } `json:"items"`
	}
	finder.Client.JSON(http.MethodGet, "/api/v1/found-reports/"+f.ID+"/history", nil).Decode(t, &history)
	got := ""
	for _, h := range history.Items {
		got += h.Transition + " "
	}
	if got != "F1 C1 C3 " {
		t.Fatalf("history transitions %q", got)
	}
}

func TestFoundReportRules(t *testing.T) {
	e := testenv.New(t)
	finder := e.User(access.Member)
	now := e.Clock.Now()
	noPhoto := foundBody(e, now, "payung", []uuid.UUID{}, map[string]any{"mode": "keep"})
	res := finder.Client.JSON(http.MethodPost, "/api/v1/found-reports", noPhoto, "Idempotency-Key", "found-nophoto-1")
	if p := res.Problem(t); len(p.Errors) == 0 || p.Errors[0].Field != "media_ids" {
		t.Fatalf("found report without photo: %d %s", res.Status, res.Body)
	}
	umbrella := foundBody(e, now, "payung", []uuid.UUID{finder.Client.UploadID("found_report")}, map[string]any{"mode": "keep"})
	var f foundReport
	res = finder.Client.JSON(http.MethodPost, "/api/v1/found-reports", umbrella, "Idempotency-Key", "found-umbrella-1")
	res.Decode(t, &f)
	if res.Status != http.StatusCreated || !f.Editable {
		t.Fatalf("umbrella kept by finder: %d %s", res.Status, res.Body)
	}
	var closed foundReport
	res = finder.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+f.ID+"/close", map[string]string{"reason": "returned_directly", "note": "Pemiliknya teman sekelas"})
	res.Decode(t, &closed)
	if res.Status != http.StatusOK || closed.Status != "closed" || closed.CustodyStatus != "released" {
		t.Fatalf("returned directly (F13, C7): %d %s", res.Status, res.Body)
	}
}

func TestProxyReportsAreForOfficersOnly(t *testing.T) {
	e := testenv.New(t)
	member := e.User(access.Member)
	post := e.Post("pos-soshum")
	officer := e.User(access.Officer, post)
	now := e.Clock.Now()

	lost := lostBody(e, now, nil)
	lost["proxy"] = map[string]string{"name": "Pengunjung Kampus", "phone": "0812 3456 7890"}
	if res := member.Client.JSON(http.MethodPost, "/api/v1/lost-reports", lost, "Idempotency-Key", "proxy-member-1"); res.Status != http.StatusForbidden {
		t.Fatalf("member proxy: %d %s", res.Status, res.Body)
	}
	var r lostReport
	res := officer.Client.JSON(http.MethodPost, "/api/v1/lost-reports", lost, "Idempotency-Key", "proxy-officer-1")
	res.Decode(t, &r)
	if res.Status != http.StatusCreated || r.Reporter == nil || r.Reporter.FullName != "Pengunjung Kampus" {
		t.Fatalf("proxy lost: %d %s", res.Status, res.Body)
	}

	found := foundBody(e, now, "ktm", []uuid.UUID{officer.Client.UploadID("found_report")}, map[string]any{"mode": "handover", "intended_post_id": post.String()})
	found["proxy"] = map[string]string{"post_id": post.String(), "storage_slot": "Laci 1"}
	var f foundReport
	res = officer.Client.JSON(http.MethodPost, "/api/v1/found-reports", found, "Idempotency-Key", "proxy-officer-2")
	res.Decode(t, &f)
	if res.Status != http.StatusCreated || f.CustodyStatus != "in_custody" {
		t.Fatalf("proxy found (C2): %d %s", res.Status, res.Body)
	}
}

func TestSearchFindsReportsForOfficers(t *testing.T) {
	e := testenv.New(t)
	owner := e.User(access.Member)
	officer := e.User(access.Officer, e.Post("pos-pusat"))
	now := e.Clock.Now()
	body := lostBody(e, now, nil)
	body["item_name"] = "Tumbler biru dongker"
	body["category_id"] = e.Category("tumbler").String()
	var r lostReport
	owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports", body, "Idempotency-Key", "search-key-1").Decode(t, &r)

	var out struct {
		Items []struct {
			Code       string `json:"code"`
			PersonName string `json:"person_name"`
		} `json:"items"`
	}
	officer.Client.JSON(http.MethodGet, "/api/v1/search?q=tumbler%20dongker&type=lost", nil).Decode(t, &out)
	if !containsCode(out.Items, r.Code) {
		t.Fatalf("text search missed %s: %+v", r.Code, out.Items)
	}
	officer.Client.JSON(http.MethodGet, "/api/v1/search?q="+r.Code, nil).Decode(t, &out)
	if len(out.Items) != 1 || out.Items[0].Code != r.Code {
		t.Fatalf("exact code search: %+v", out.Items)
	}
	officer.Client.JSON(http.MethodGet, "/api/v1/search?q=tumblr%20biru", nil).Decode(t, &out)
	if !containsCode(out.Items, r.Code) {
		t.Fatalf("typo tolerant search missed %s", r.Code)
	}
	if res := owner.Client.JSON(http.MethodGet, "/api/v1/search?q=tumbler", nil); res.Status != http.StatusForbidden {
		t.Fatalf("member search: %d", res.Status)
	}
}

func containsCode(items []struct {
	Code       string `json:"code"`
	PersonName string `json:"person_name"`
}, code string) bool {
	for _, it := range items {
		if it.Code == code {
			return true
		}
	}
	return false
}

func TestSearchReportsSemantic_US41(t *testing.T) {
	fake := testenv.NewFakeML(t)
	e := testenv.New(t, testenv.WithML(fake.URL), testenv.WithSemanticSearch())
	ctx := context.Background()
	now := e.Clock.Now()
	owner := e.User(access.Member)
	officer := e.User(access.Officer, e.Post("pos-pusat"))

	// US-41: "tempat minum biru" finds a report described as "tumbler warna biru dongker" in top five
	body := lostBody(e, now, nil)
	body["item_name"] = "tumbler warna biru dongker"
	body["category_id"] = e.Category("tumbler").String()
	var r lostReport
	owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports", body, "Idempotency-Key", "us41-search-key").Decode(t, &r)

	// Register text_encoder model version
	var versionID uuid.UUID
	err := e.Pool.QueryRow(ctx, "SELECT id FROM model_versions WHERE component = 'text_encoder' AND status = 'active' LIMIT 1").Scan(&versionID)
	if err != nil {
		versionID = uuid.New()
		if _, err := e.Pool.Exec(ctx, "INSERT INTO model_versions (id, component, name, status, created_at) VALUES ($1, 'text_encoder', 'test-e5-r1', 'active', $2)", versionID, now); err != nil {
			t.Fatal(err)
		}
	}

	// Fetch query vector for "query: tempat minum biru" from FakeML
	var embOut struct {
		E5 struct {
			Embeddings [][]float32 `json:"embeddings"`
		} `json:"e5"`
	}
	reqBody := `{"texts": ["query: tempat minum biru"], "kinds": ["e5"]}`
	resp, err := http.Post(fake.URL+"/v1/embed/text", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&embOut); err != nil {
		t.Fatal(err)
	}

	var vecLitBuilder strings.Builder
	vecLitBuilder.WriteByte('[')
	for i, x := range embOut.E5.Embeddings[0] {
		if i > 0 {
			vecLitBuilder.WriteByte(',')
		}
		fmt.Fprintf(&vecLitBuilder, "%f", x)
	}
	vecLitBuilder.WriteByte(']')

	// Store embedding for the lost report
	_, err = e.Pool.Exec(ctx, "INSERT INTO text_embeddings_e5 (report_type, report_id, model_version_id, embedding, text_sha256, created_at, updated_at) VALUES ('lost', $1, $2, $3::vector, $4, $5, $5)", r.ID, versionID, vecLitBuilder.String(), []byte("sha256"), now)
	if err != nil {
		t.Fatal(err)
	}

	// Semantic search with semantic=true finds the report in top five
	var out struct {
		Items []struct {
			Code       string   `json:"code"`
			ItemName   string   `json:"item_name"`
			Relevance  *float64 `json:"relevance"`
			PersonName string   `json:"person_name"`
		} `json:"items"`
	}
	officer.Client.JSON(http.MethodGet, "/api/v1/search?q=tempat%20minum%20biru&type=lost&semantic=true", nil).Decode(t, &out)
	foundInTop5 := false
	for i, it := range out.Items {
		if i < 5 && it.Code == r.Code {
			foundInTop5 = true
			if it.Relevance == nil || *it.Relevance <= 0.0 {
				t.Fatalf("expected positive relevance, got %+v", it.Relevance)
			}
			break
		}
	}
	if !foundInTop5 {
		t.Fatalf("semantic search US-41 failed to find %s in top five: %+v", r.Code, out.Items)
	}
}

func TestAuditLogIsAppendOnly(t *testing.T) {
	e := testenv.New(t)
	ctx := context.Background()
	if _, err := e.Pool.Exec(ctx, "UPDATE audit_logs SET action = 'tampered' WHERE id = (SELECT min(id) FROM audit_logs)"); err == nil {
		t.Fatal("the application role updated an audit row (FR-AUDIT-02)")
	}
	if _, err := e.Pool.Exec(ctx, "DELETE FROM audit_logs"); err == nil {
		t.Fatal("the application role deleted audit rows (FR-AUDIT-02)")
	}
}
