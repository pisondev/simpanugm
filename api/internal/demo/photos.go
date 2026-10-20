package demo

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// the manifest is found from the working directory upwards, so the binary, the image and tests in any package agree
func manifestCandidates() []string {
	out := []string{}
	if dir := os.Getenv("SIMPAN_DEMO_DIR"); dir != "" {
		out = append(out, filepath.Join(dir, "SOURCES.csv"))
	}
	rel := "."
	for range 6 {
		out = append(out, filepath.Join(rel, "data", "demo", "SOURCES.csv"))
		rel = filepath.Join(rel, "..")
	}
	return append(out, filepath.Join("/app", "data", "demo", "SOURCES.csv"))
}

func discoverLocalBackgroundFoundReports(categories []string) ([]DemoFoundReport, error) {
	for _, manifestPath := range manifestCandidates() {
		manifest, err := os.Open(manifestPath)
		if err != nil {
			continue
		}
		reports, readErr := discoverBackgroundFoundReports(manifest, filepath.Join(filepath.Dir(manifestPath), "images"), categories)
		_ = manifest.Close()
		return reports, readErr
	}
	return nil, os.ErrNotExist
}

type demoPhotoSource struct {
	filename string
	source   string
	category string
	notes    string
}

type demoPhotoUploader struct {
	email   string
	officer bool
}

var demoPhotoUploaders = []demoPhotoUploader{
	{email: "petugas@demo.simpanugm.my.id", officer: true},
	{email: "petugas.pos1@demo.simpanugm.my.id", officer: true},
	{email: "petugas.pos2@demo.simpanugm.my.id", officer: true},
	{email: "petugas.foto@demo.simpanugm.my.id", officer: true},
}

func balanceDemoPhotoUploaders(reports []DemoFoundReport, lost []DemoLostReport) ([]DemoFoundReport, error) {
	counts := make(map[string]int, len(demoPhotoUploaders))
	for _, report := range lost {
		if report.PhotoFilename == "" {
			continue
		}
		email := report.OwnerEmail
		if strings.HasPrefix(report.Key, "bg-lost-") {
			email = "petugas@demo.simpanugm.my.id"
		}
		if email == "" {
			email = "anggota@demo.simpanugm.my.id"
		}
		counts[email]++
	}
	staticFound := make([]DemoFoundReport, 0, len(HeroFoundReports)+len(BackgroundFoundReports))
	staticFound = append(staticFound, HeroFoundReports...)
	staticFound = append(staticFound, BackgroundFoundReports...)
	for _, report := range staticFound {
		if report.PhotoFilename == "" {
			continue
		}
		email := report.FinderEmail
		if strings.HasPrefix(report.Key, "bg-found-") {
			email = "petugas@demo.simpanugm.my.id"
		}
		if email == "" {
			email = "anggota@demo.simpanugm.my.id"
		}
		counts[email]++
	}
	balanced := append([]DemoFoundReport(nil), reports...)
	for i := range balanced {
		selected := -1
		for actorIndex, actor := range demoPhotoUploaders {
			if counts[actor.email] >= 30 {
				continue
			}
			if selected < 0 || counts[actor.email] < counts[demoPhotoUploaders[selected].email] {
				selected = actorIndex
			}
		}
		if selected < 0 {
			return nil, fmt.Errorf("demo photo upload limit reached after %d reports", i)
		}
		actor := demoPhotoUploaders[selected]
		keySuffix := strings.TrimPrefix(balanced[i].Key, "bg-found-photo-")
		if !actor.officer {
			return nil, fmt.Errorf("demo photo uploader %s must be an officer", actor.email)
		}
		balanced[i].Key = "bg-found-photo-" + keySuffix
		balanced[i].FinderEmail = actor.email
		counts[actor.email]++
	}
	return balanced, nil
}

func discoverBackgroundFoundReports(manifest io.Reader, imageDir string, categories []string) ([]DemoFoundReport, error) {
	r := csv.NewReader(manifest)
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	columns := make(map[string]int, len(rows[0]))
	for i, name := range rows[0] {
		columns[strings.TrimSpace(name)] = i
	}
	value := func(row []string, column string) string {
		i, ok := columns[column]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
	categorySet := make(map[string]bool, len(categories))
	for _, slug := range categories {
		categorySet[slug] = true
	}
	photos := make([]demoPhotoSource, 0)
	seenContent := make(map[[32]byte]bool)
	reservedReports := make([]DemoFoundReport, 0, len(HeroFoundReports)+len(BackgroundFoundReports))
	reservedReports = append(reservedReports, HeroFoundReports...)
	reservedReports = append(reservedReports, BackgroundFoundReports...)
	for _, report := range reservedReports {
		if report.PhotoFilename == "" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(imageDir, filepath.Base(report.PhotoFilename)))
		if err == nil {
			seenContent[sha256.Sum256(content)] = true
		}
	}
	for _, row := range rows[1:] {
		filename := filepath.Base(value(row, "filename"))
		source := value(row, "source")
		category := value(row, "category")
		notes := strings.ToLower(value(row, "notes"))
		itemType := value(row, "item_type")
		if filename == "." || filename == "" || itemType != "background" ||
			(source != "wikimedia_commons" && source != "open_images") ||
			!categorySet[category] || strings.Contains(notes, "holdout") {
			continue
		}
		photoPath := filepath.Join(imageDir, filename)
		if _, err := os.Stat(photoPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		content, err := os.ReadFile(photoPath)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(content)
		if seenContent[digest] {
			continue
		}
		seenContent[digest] = true
		photos = append(photos, demoPhotoSource{filename: filename, source: source, category: category})
	}
	sort.Slice(photos, func(i, j int) bool { return photos[i].filename < photos[j].filename })

	locations := []string{"perpustakaan-pusat", "kantin-biogeo"}
	posts := []string{"pos-perpustakaan", "pos-saintek"}
	reports := make([]DemoFoundReport, 0, len(photos))
	for i, photo := range photos {
		digest := sha256.Sum256([]byte(photo.filename))
		key := "bg-found-photo-" + hex.EncodeToString(digest[:6])
		reports = append(reports, DemoFoundReport{
			Key:           key,
			FinderEmail:   "anggota@demo.simpanugm.my.id",
			ItemName:      fmt.Sprintf("Barang contoh %03d (%s)", i+1, strings.ReplaceAll(photo.category, "-", " ")),
			CategorySlug:  photo.category,
			Description:   "Foto objek contoh untuk data demo. Rincian kepemilikan dan ciri barang belum diverifikasi.",
			LocationSlug:  locations[i%len(locations)],
			PostSlug:      posts[i%len(posts)],
			CustodyMode:   "handover",
			DaysAgo:       1 + i%21,
			PhotoFilename: photo.filename,
		})
	}
	return reports, nil
}
