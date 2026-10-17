//go:build integration

package server_test

import (
	"bytes"
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

func TestMasterdataReadsAndConditionalRequests(t *testing.T) {
	e := testenv.New(t)
	c := e.User(access.Member).Client

	var cats struct {
		Items []struct {
			Slug     string `json:"slug"`
			Children []struct {
				Slug        string `json:"slug"`
				Sensitivity string `json:"sensitivity"`
				Questions   []any  `json:"verification_questions"`
			} `json:"children"`
		} `json:"items"`
	}
	c.JSON(http.MethodGet, "/api/v1/categories", nil).Decode(t, &cats)
	if len(cats.Items) < 10 {
		t.Fatalf("expected the seeded parents, got %d", len(cats.Items))
	}
	for _, p := range cats.Items {
		for _, l := range p.Children {
			if l.Sensitivity == "" || len(l.Questions) == 0 {
				t.Fatalf("leaf %s lacks sensitivity or questions", l.Slug)
			}
		}
	}

	for _, q := range []string{"perpus", "Perpustakaan", "perpus%20pusat"} {
		var locs struct {
			Items []struct{ Slug string } `json:"items"`
		}
		c.JSON(http.MethodGet, "/api/v1/locations?q="+q, nil).Decode(t, &locs)
		if len(locs.Items) == 0 || locs.Items[0].Slug != "perpustakaan-pusat" {
			t.Fatalf("q=%s: first result %+v", q, locs.Items)
		}
	}

	var posts struct {
		Items []struct {
			Slug      string   `json:"slug"`
			DistanceM *float64 `json:"distance_m"`
		} `json:"items"`
	}
	c.JSON(http.MethodGet, "/api/v1/posts?near_location_id="+e.Location("fmipa").String(), nil).Decode(t, &posts)
	for i := 1; i < len(posts.Items); i++ {
		if *posts.Items[i-1].DistanceM > *posts.Items[i].DistanceM {
			t.Fatalf("posts not sorted by distance: %+v", posts.Items)
		}
	}

	first := c.JSON(http.MethodGet, "/api/v1/zones", nil)
	tag := first.Header.Get("ETag")
	if tag == "" {
		t.Fatal("zones response has no ETag")
	}
	if again := c.JSON(http.MethodGet, "/api/v1/zones", nil, "If-None-Match", tag); again.Status != http.StatusNotModified {
		t.Fatalf("conditional request: %d", again.Status)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	e := testenv.New(t)
	count := func() (n int) {
		_ = e.Pool.QueryRow(context.Background(), "SELECT (SELECT count(*) FROM categories) + (SELECT count(*) FROM locations) + (SELECT count(*) FROM posts)").Scan(&n)
		return n
	}
	before := count()
	set, err := masterdata.LoadSeed(filepath.Join(testenv.RepoRoot(), "data", "seed"), filepath.Join(testenv.RepoRoot(), "deploy", "dev-seed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := masterdata.ApplySeed(context.Background(), e.Pool, set); err != nil {
		t.Fatal(err)
	}
	if after := count(); after != before {
		t.Fatalf("rows changed from %d to %d", before, after)
	}
}

func TestUploadStripsMetadataAndGuardsAccess(t *testing.T) {
	e := testenv.New(t)
	owner := e.User(access.Member)
	stranger := e.User(access.Member)
	officer := e.User(access.Officer, e.Post("pos-pusat"))

	res := owner.Client.Upload("lost_report", testenv.JPEG(t, 800, 600, testenv.ExifGPS()))
	if res.Status != http.StatusCreated {
		t.Fatalf("upload: %d %s", res.Status, res.Body)
	}
	var m struct {
		ID    string `json:"id"`
		State string `json:"state"`
	}
	res.Decode(t, &m)
	if m.State != "staged" {
		t.Fatalf("state %s", m.State)
	}
	orig := owner.Client.JSON(http.MethodGet, "/api/v1/media/"+m.ID+"?variant=original", nil)
	if orig.Status != http.StatusOK || orig.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("original: %d %s", orig.Status, orig.Header.Get("Content-Type"))
	}
	if bytes.Contains(orig.Body, []byte("GPSLatitude")) || bytes.Contains(orig.Body, []byte("Exif")) {
		t.Fatal("metadata survived re-encoding (FR-MEDIA-04)")
	}
	if !strings.HasPrefix(orig.Header.Get("Cache-Control"), "private") {
		t.Fatalf("cache control %q", orig.Header.Get("Cache-Control"))
	}
	for _, v := range []string{"display", "thumb", "thumb_blur"} {
		if r := owner.Client.JSON(http.MethodGet, "/api/v1/media/"+m.ID+"?variant="+v, nil); r.Header.Get("Content-Type") != "image/webp" {
			t.Fatalf("variant %s: %d %s", v, r.Status, r.Header.Get("Content-Type"))
		}
	}
	if r := stranger.Client.JSON(http.MethodGet, "/api/v1/media/"+m.ID, nil); r.Status != http.StatusNotFound {
		t.Fatalf("stranger reading a staged upload: %d", r.Status)
	}
	if r := officer.Client.JSON(http.MethodGet, "/api/v1/media/"+m.ID, nil); r.Status != http.StatusOK {
		t.Fatalf("officer reading: %d", r.Status)
	}
	if r := e.Client().JSON(http.MethodGet, "/api/v1/media/"+m.ID, nil); r.Status != http.StatusUnauthorized {
		t.Fatalf("anonymous reading: %d", r.Status)
	}
}

func TestUploadRejectsNonImagesAndTinyImages(t *testing.T) {
	e := testenv.New(t)
	c := e.User(access.Member).Client
	if r := c.Upload("lost_report", []byte("bukan gambar, hanya teks biasa")); r.Problem(t).Code != "media_invalid_type" {
		t.Fatalf("text upload: %d %s", r.Status, r.Body)
	}
	if r := c.Upload("lost_report", testenv.JPEG(t, 300, 200, nil)); r.Problem(t).Code != "media_too_small" {
		t.Fatalf("tiny upload: %d %s", r.Status, r.Body)
	}
	if r := c.Upload("claim_evidence", testenv.JPEG(t, 640, 480, nil)); r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
		t.Fatalf("unknown purpose: %d %s", r.Status, r.Body)
	}
}

func TestStagedUploadsArePurgedAfterADay(t *testing.T) {
	e := testenv.New(t)
	c := e.User(access.Member).Client
	id := c.UploadID("found_report")
	if r := c.JSON(http.MethodDelete, "/api/v1/media/"+c.UploadID("found_report").String(), nil); r.Status != http.StatusNoContent {
		t.Fatalf("delete staged: %d %s", r.Status, r.Body)
	}
	e.Clock.Advance(25 * time.Hour)
	if _, err := e.Services.Media.PurgeStaged(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r := c.JSON(http.MethodGet, "/api/v1/media/"+id.String(), nil); r.Status != http.StatusNotFound {
		t.Fatalf("purged upload still served: %d", r.Status)
	}
}
