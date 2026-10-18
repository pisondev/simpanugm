//go:build integration

package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

var allWeek = map[string][][]string{
	"mon": {{"08:00", "16:00"}}, "tue": {{"08:00", "16:00"}}, "wed": {{"08:00", "16:00"}}, "thu": {{"08:00", "16:00"}},
	"fri": {{"08:00", "11:30"}, {"13:00", "16:00"}}, "sat": {}, "sun": {},
}

type adminRow struct {
	ID       uuid.UUID `json:"id"`
	Slug     string    `json:"slug"`
	Name     string    `json:"name"`
	ParentID *string   `json:"parent_id"`
	Active   bool      `json:"active"`
	EditedAt *string   `json:"edited_at"`
}

func create(t *testing.T, c *testenv.Client, path string, body any) adminRow {
	t.Helper()
	res := c.JSON(http.MethodPost, path, body)
	if res.Status != http.StatusCreated {
		t.Fatalf("POST %s: %d %s", path, res.Status, res.Body)
	}
	var r adminRow
	res.Decode(t, &r)
	return r
}

func patch(t *testing.T, c *testenv.Client, path string, body any) adminRow {
	t.Helper()
	res := c.JSON(http.MethodPatch, path, body)
	if res.Status != http.StatusOK {
		t.Fatalf("PATCH %s: %d %s", path, res.Status, res.Body)
	}
	var r adminRow
	res.Decode(t, &r)
	return r
}

func slugsOf(t *testing.T, c *testenv.Client, path string) map[string]bool {
	t.Helper()
	var l struct {
		Items []struct {
			Slug     string `json:"slug"`
			Children []struct {
				Slug string `json:"slug"`
			} `json:"children"`
		} `json:"items"`
	}
	c.JSON(http.MethodGet, path, nil).Decode(t, &l)
	out := map[string]bool{}
	for _, i := range l.Items {
		out[i.Slug] = true
		for _, ch := range i.Children {
			out[ch.Slug] = true
		}
	}
	return out
}

// whether a problem response names this field among its errors
func fieldError(t *testing.T, r testenv.Response, field string) bool {
	t.Helper()
	if r.Status != http.StatusUnprocessableEntity {
		return false
	}
	for _, f := range r.Problem(t).Errors {
		if f.Field == field {
			return true
		}
	}
	return false
}

func auditChanges(t *testing.T, e *testenv.Env, action string, id uuid.UUID) map[string]json.RawMessage {
	t.Helper()
	var raw []byte
	if err := e.Pool.QueryRow(context.Background(), "SELECT changes FROM audit_logs WHERE action = $1 AND entity_id = $2 ORDER BY id DESC LIMIT 1", action, id).Scan(&raw); err != nil {
		t.Fatalf("audit %s for %s: %v", action, id, err)
	}
	out := map[string]json.RawMessage{}
	_ = json.Unmarshal(raw, &out)
	return out
}

// US-07 and FR-ADMIN-02: an administrator creates, edits and switches off each master data type, pickers follow,
// existing reports keep their rows, and every change is audited with before and after
func TestAdministratorsMaintainMasterData(t *testing.T) {
	e := testenv.New(t)
	admin := e.User(access.Admin).Client
	member := e.User(access.Member)
	tag := testenv.Unique("uji")

	if r := member.Client.JSON(http.MethodGet, "/api/v1/admin/categories", nil); r.Status != http.StatusForbidden {
		t.Fatalf("members cannot manage master data: %d", r.Status)
	}
	if r := e.User(access.Officer).Client.JSON(http.MethodPost, "/api/v1/admin/zones", map[string]any{"slug": tag, "name": "Zona", "centroid_lat": -7.77, "centroid_lng": 110.37}); r.Status != http.StatusForbidden {
		t.Fatalf("officers cannot manage master data: %d", r.Status)
	}

	zone := create(t, admin, "/api/v1/admin/zones", map[string]any{"slug": "zona-" + tag, "name": "Zona Uji", "centroid_lat": -7.7701, "centroid_lng": 110.3780})
	loc := create(t, admin, "/api/v1/admin/locations", map[string]any{
		"zone_id": zone.ID, "slug": "gedung-" + tag, "name": "Gedung Uji Coba", "aliases": []string{"GUC", "Gedung UC"}, "kind": "building", "lat": -7.7702, "lng": 110.3781,
	})
	_ = loc
	if r := admin.JSON(http.MethodPost, "/api/v1/admin/locations", map[string]any{
		"zone_id": zone.ID, "slug": "gedung-" + tag, "name": "Gedung Lain", "kind": "building", "lat": -7.77, "lng": 110.378,
	}); !fieldError(t, r, "slug") {
		t.Fatalf("a taken slug is a field error: %d %s", r.Status, r.Body)
	}
	if !slugsOf(t, member.Client, "/api/v1/locations?q=GUC")["gedung-"+tag] {
		t.Fatal("a new location is found by its alias")
	}

	if r := admin.JSON(http.MethodPost, "/api/v1/admin/posts", map[string]any{
		"slug": "pos-" + tag, "name": "Pos Uji", "location_id": loc.ID, "opening_hours": map[string]any{"mon": [][]string{{"08:00", "16:00"}}},
	}); !fieldError(t, r, "opening_hours") {
		t.Fatalf("every weekday needs its hours: %d %s", r.Status, r.Body)
	}
	post := create(t, admin, "/api/v1/admin/posts", map[string]any{"slug": "pos-" + tag, "name": "Pos Uji", "location_id": loc.ID, "phone": "0274 000000", "opening_hours": allWeek})
	if !slugsOf(t, member.Client, "/api/v1/posts")["pos-"+tag] {
		t.Fatal("a new post reaches the pickers")
	}

	var parents struct {
		Items []adminRow `json:"items"`
	}
	admin.JSON(http.MethodGet, "/api/v1/admin/categories", nil).Decode(t, &parents)
	var electronics uuid.UUID
	for _, c := range parents.Items {
		if c.Slug == "elektronik" && c.ParentID == nil {
			electronics = c.ID
		}
	}
	if electronics == uuid.Nil {
		t.Fatal("seeded parent elektronik")
	}
	if r := admin.JSON(http.MethodPost, "/api/v1/admin/categories", map[string]any{"parent_id": electronics, "slug": "speaker-" + tag, "name": "Speaker"}); !fieldError(t, r, "sensitivity") || !fieldError(t, r, "clip_prompts") {
		t.Fatalf("a leaf needs sensitivity, prompts and questions: %d %s", r.Status, r.Body)
	}
	leaf := create(t, admin, "/api/v1/admin/categories", map[string]any{
		"parent_id": electronics, "slug": "speaker-" + tag, "name": "Speaker mini", "sensitivity": "valuable",
		"clip_prompts":           []string{"a photo of a portable bluetooth speaker"},
		"verification_questions": []map[string]any{{"key": "brand_colour", "question": "Sebutkan merek dan warna speaker.", "required": true}},
	})
	if !slugsOf(t, member.Client, "/api/v1/categories")["speaker-"+tag] {
		t.Fatal("a new leaf reaches the category picker")
	}

	// edits are audited field by field with the value before and after
	edited := patch(t, admin, "/api/v1/admin/categories/"+leaf.ID.String(), map[string]any{
		"name":                   "Speaker Bluetooth",
		"verification_questions": []map[string]any{{"key": "brand_colour", "question": "Sebutkan merek, warna dan stiker pada speaker.", "required": true}},
	})
	if edited.Name != "Speaker Bluetooth" || edited.EditedAt == nil {
		t.Fatalf("edited category: %+v", edited)
	}
	ch := auditChanges(t, e, "masterdata.category.update", leaf.ID)
	var name []string
	_ = json.Unmarshal(ch["name"], &name)
	if len(name) != 2 || name[0] != "Speaker mini" || name[1] != "Speaker Bluetooth" || ch["verification_questions"] == nil || ch["sort_order"] != nil {
		t.Fatalf("audit before and after: %v", ch)
	}

	// a report filed before the switch off keeps its category, a new one cannot pick it
	body := lostBody(e, e.Clock.Now(), nil)
	body["category_id"] = leaf.ID.String()
	res := member.Client.JSON(http.MethodPost, "/api/v1/lost-reports", body, "Idempotency-Key", "md-admin-"+tag)
	if res.Status != http.StatusCreated {
		t.Fatalf("lost report with the new leaf: %d %s", res.Status, res.Body)
	}
	var report lostReport
	res.Decode(t, &report)
	patch(t, admin, "/api/v1/admin/categories/"+leaf.ID.String(), map[string]any{"active": false})
	patch(t, admin, "/api/v1/admin/locations/"+loc.ID.String(), map[string]any{"active": false})
	patch(t, admin, "/api/v1/admin/posts/"+post.ID.String(), map[string]any{"active": false})
	patch(t, admin, "/api/v1/admin/zones/"+zone.ID.String(), map[string]any{"active": false})
	if slugsOf(t, member.Client, "/api/v1/categories")["speaker-"+tag] || slugsOf(t, member.Client, "/api/v1/posts")["pos-"+tag] ||
		slugsOf(t, member.Client, "/api/v1/locations?q=GUC")["gedung-"+tag] || slugsOf(t, member.Client, "/api/v1/zones")["zona-"+tag] {
		t.Fatal("switched off rows leave the pickers")
	}
	var kept struct {
		Category struct {
			Name string `json:"name"`
		} `json:"category"`
	}
	member.Client.JSON(http.MethodGet, "/api/v1/lost-reports/"+report.Code, nil).Decode(t, &kept)
	if kept.Category.Name != "Speaker Bluetooth" {
		t.Fatalf("the report keeps its category: %+v", kept)
	}
	if r := member.Client.JSON(http.MethodPost, "/api/v1/lost-reports", body, "Idempotency-Key", "md-admin-2-"+tag); !fieldError(t, r, "category_id") {
		t.Fatalf("a switched off category is refused for new reports: %d %s", r.Status, r.Body)
	}

	// a compatibility pair is switched off rather than deleted
	pair := map[string]any{"category_a": leaf.ID, "category_b": e.Category("dompet"), "factor": 0.6, "active": true}
	if r := admin.JSON(http.MethodPut, "/api/v1/admin/category-compatibility", pair); r.Status != http.StatusOK {
		t.Fatalf("put pair: %d %s", r.Status, r.Body)
	}
	pair["active"] = false
	if r := admin.JSON(http.MethodPut, "/api/v1/admin/category-compatibility", pair); r.Status != http.StatusOK {
		t.Fatalf("switch pair off: %d %s", r.Status, r.Body)
	}
	if r := admin.JSON(http.MethodPut, "/api/v1/admin/category-compatibility", map[string]any{"category_a": electronics, "category_b": leaf.ID, "factor": 0.5, "active": true}); !fieldError(t, r, "category_a") {
		t.Fatalf("pairs join leaves only: %d %s", r.Status, r.Body)
	}
}

// the reviewed seed inserts what is missing but never overwrites a row an administrator edited
func TestSeedLeavesAdministratorEditsAlone(t *testing.T) {
	e := testenv.New(t)
	admin := e.User(access.Admin).Client
	ctx := context.Background()
	id := e.Location("wisdom-park")
	var original string
	if err := e.Pool.QueryRow(ctx, "SELECT name FROM locations WHERE id = $1", id).Scan(&original); err != nil {
		t.Fatal(err)
	}
	patch(t, admin, "/api/v1/admin/locations/"+id.String(), map[string]any{"name": original + " (Uji)"})
	t.Cleanup(func() { _, _ = e.Pool.Exec(ctx, "UPDATE locations SET name = $2 WHERE id = $1", id, original) })

	root := testenv.RepoRoot()
	set, err := masterdata.LoadSeed(filepath.Join(root, "data", "seed"), filepath.Join(root, "deploy", "dev-seed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := masterdata.ApplySeed(ctx, e.Pool, set); err != nil {
		t.Fatalf("seed after an edit: %v", err)
	}
	var name string
	if err := e.Pool.QueryRow(ctx, "SELECT name FROM locations WHERE id = $1", id).Scan(&name); err != nil || name != original+" (Uji)" {
		t.Fatalf("the edit survives the seed: %q %v", name, err)
	}
}

// coordinates outside campus.bounding_box are refused once the setting exists
func TestMasterDataCoordinatesStayOnCampus(t *testing.T) {
	e := testenv.New(t)
	ctx := context.Background()
	if _, err := e.Pool.Exec(ctx, `INSERT INTO system_settings (key, value) VALUES ('campus.bounding_box', '{"min_lat":-7.80,"max_lat":-7.75,"min_lng":110.35,"max_lng":110.40}')
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = e.Pool.Exec(ctx, "DELETE FROM system_settings WHERE key = 'campus.bounding_box'") })
	e.Services.Settings.Invalidate()
	admin := e.User(access.Admin).Client
	r := admin.JSON(http.MethodPost, "/api/v1/admin/zones", map[string]any{"slug": testenv.Unique("jauh"), "name": "Zona Jauh", "centroid_lat": -6.2, "centroid_lng": 106.8})
	if !fieldError(t, r, "centroid_lat") {
		t.Fatalf("Jakarta is not on campus: %d %s", r.Status, r.Body)
	}
}
