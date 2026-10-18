package masterdata

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/geo"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
)

type SeedZone struct {
	Slug, Name string
	Lat, Lng   float64
}

type SeedLocation struct {
	Slug, Name, ZoneSlug, Kind string
	Lat, Lng                   float64
	Aliases                    []string
}

type SeedPost struct {
	Slug         string                `yaml:"slug"`
	Name         string                `yaml:"name"`
	LocationSlug string                `yaml:"location_slug"`
	Phone        string                `yaml:"phone"`
	OpeningHours map[string][][]string `yaml:"opening_hours"`
}

type SeedLeaf struct {
	Slug                  string     `yaml:"slug"`
	Name                  string     `yaml:"name"`
	Sensitivity           string     `yaml:"sensitivity"`
	RetentionDays         *int32     `yaml:"retention_days"`
	ClipPrompts           []string   `yaml:"clip_prompts"`
	VerificationQuestions []Question `yaml:"verification_questions"`
}

type SeedParent struct {
	Slug      string     `yaml:"slug"`
	Name      string     `yaml:"name"`
	SortOrder int32      `yaml:"sort_order"`
	Children  []SeedLeaf `yaml:"children"`
}

type SeedCompat struct {
	A, B   string
	Factor float64
}

type SeedColour struct {
	Name, Hex string
	Order     int32
}

type SeedBrand struct {
	Name    string
	Aliases []string
}

type SeedSynonym struct{ Term, Expansion string }

type SeedSet struct {
	Zones      []SeedZone
	Locations  []SeedLocation
	Posts      []SeedPost
	Categories []SeedParent
	Compat     []SeedCompat
	Colours    []SeedColour
	Brands     []SeedBrand
	Synonyms   []SeedSynonym
	Settings   map[string]any
	// directory each file came from, so a development overlay shows up in logs
	Sources map[string]string
}

var seedFiles = []string{
	"zones.csv", "locations.csv", "posts.yaml", "categories.yaml", "category_compatibility.csv",
	"colours.csv", "brands.csv", "synonyms.csv", "settings.yaml",
}

// the first directory that has a file wins, so the reviewed data/seed always beats a development overlay
func locate(dirs []string) map[string]string {
	out := map[string]string{}
	for _, name := range seedFiles {
		for _, d := range dirs {
			p := filepath.Join(d, name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				out[name] = p
				break
			}
		}
	}
	return out
}

func readCSV(path string, header []string) ([][]string, error) {
	f, err := os.Open(path) //nolint:gosec // seed paths come from the operator's command line
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	r := csv.NewReader(f)
	r.FieldsPerRecord = len(header)
	got, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("%s: header: %w", filepath.Base(path), err)
	}
	if !slices.Equal(got, header) {
		return nil, fmt.Errorf("%s: header must be %s", filepath.Base(path), strings.Join(header, ","))
	}
	var rows [][]string
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		for i := range rec {
			rec[i] = strings.TrimSpace(rec[i])
		}
		rows = append(rows, rec)
	}
}

func splitList(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ";") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func readYAML(path string, v any) error {
	b, err := os.ReadFile(path) //nolint:gosec // seed paths come from the operator's command line
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return nil
}

// settings whose value is an object rather than a branch of more settings
var objectSettings = map[string]bool{"notify.email_quiet_hours": true, "campus.bounding_box": true}

func flattenSettings(prefix string, in map[string]any, out map[string]any) {
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if m, ok := v.(map[string]any); ok && !objectSettings[key] {
			flattenSettings(key, m, out)
			continue
		}
		out[key] = v
	}
}

func LoadSeed(dirs ...string) (SeedSet, error) {
	s := SeedSet{Sources: locate(dirs)}
	var errs []error
	parse := func(field, raw string) float64 {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s %q is not a number", field, raw))
		}
		return f
	}
	path := func(name string) (string, bool) {
		p, ok := s.Sources[name]
		if !ok {
			errs = append(errs, fmt.Errorf("%s not found in %s", name, strings.Join(dirs, ", ")))
		}
		return p, ok
	}
	if p, ok := path("zones.csv"); ok {
		if rows, err := readCSV(p, []string{"slug", "name", "centroid_lat", "centroid_lng"}); err != nil {
			errs = append(errs, err)
		} else {
			for _, r := range rows {
				s.Zones = append(s.Zones, SeedZone{Slug: r[0], Name: r[1], Lat: parse("centroid_lat", r[2]), Lng: parse("centroid_lng", r[3])})
			}
		}
	}
	if p, ok := path("locations.csv"); ok {
		if rows, err := readCSV(p, []string{"slug", "name", "zone_slug", "kind", "lat", "lng", "aliases"}); err != nil {
			errs = append(errs, err)
		} else {
			for _, r := range rows {
				s.Locations = append(s.Locations, SeedLocation{Slug: r[0], Name: r[1], ZoneSlug: r[2], Kind: r[3],
					Lat: parse("lat", r[4]), Lng: parse("lng", r[5]), Aliases: splitList(r[6])})
			}
		}
	}
	if p, ok := path("posts.yaml"); ok {
		if err := readYAML(p, &s.Posts); err != nil {
			errs = append(errs, err)
		}
	}
	if p, ok := path("categories.yaml"); ok {
		var doc struct {
			Parents []SeedParent `yaml:"parents"`
		}
		if err := readYAML(p, &doc); err != nil {
			errs = append(errs, err)
		}
		s.Categories = doc.Parents
	}
	if p, ok := path("category_compatibility.csv"); ok {
		if rows, err := readCSV(p, []string{"category_a", "category_b", "factor"}); err != nil {
			errs = append(errs, err)
		} else {
			for _, r := range rows {
				s.Compat = append(s.Compat, SeedCompat{A: r[0], B: r[1], Factor: parse("factor", r[2])})
			}
		}
	}
	if p, ok := path("colours.csv"); ok {
		if rows, err := readCSV(p, []string{"name", "hex", "sort_order"}); err != nil {
			errs = append(errs, err)
		} else {
			for _, r := range rows {
				s.Colours = append(s.Colours, SeedColour{Name: r[0], Hex: r[1], Order: int32(parse("sort_order", r[2]))})
			}
		}
	}
	if p, ok := path("brands.csv"); ok {
		if rows, err := readCSV(p, []string{"name", "aliases"}); err != nil {
			errs = append(errs, err)
		} else {
			for _, r := range rows {
				s.Brands = append(s.Brands, SeedBrand{Name: r[0], Aliases: splitList(r[1])})
			}
		}
	}
	if p, ok := path("synonyms.csv"); ok {
		if rows, err := readCSV(p, []string{"term", "expansion"}); err != nil {
			errs = append(errs, err)
		} else {
			for _, r := range rows {
				s.Synonyms = append(s.Synonyms, SeedSynonym{Term: r[0], Expansion: r[1]})
			}
		}
	}
	if p, ok := path("settings.yaml"); ok {
		raw := map[string]any{}
		if err := readYAML(p, &raw); err != nil {
			errs = append(errs, err)
		}
		s.Settings = map[string]any{}
		flattenSettings("", raw, s.Settings)
	}
	return s, errors.Join(errs...)
}

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	hexPattern  = regexp.MustCompile(`^#[0-9A-F]{6}$`)
	weekdays    = map[string]bool{"mon": true, "tue": true, "wed": true, "thu": true, "fri": true, "sat": true, "sun": true}
	clockTime   = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
	kinds       = map[string]bool{"building": true, "outdoor": true, "facility": true, "transport": true, "parking": true}
	zoneSlugs   = []string{"agro", "medika", "pusat", "saintek", "soshum"}
)

// the box is optional here because data/scripts/check.py takes it as an argument after the location review
func (s SeedSet) campusBox() (*geo.Box, error) {
	raw, ok := s.Settings["campus.bounding_box"]
	if !ok {
		return nil, nil
	}
	b, _ := json.Marshal(raw)
	var box geo.Box
	if err := json.Unmarshal(b, &box); err != nil || box.MinLat >= box.MaxLat || box.MinLng >= box.MaxLng {
		return nil, errors.New("settings: campus.bounding_box needs min_lat < max_lat and min_lng < max_lng")
	}
	return &box, nil
}

// the seed contract of data/README.md and data/scripts/check.py, checked before anything touches the database
func (s SeedSet) Validate() error {
	var errs []error
	fail := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }
	box, err := s.campusBox()
	if err != nil {
		errs = append(errs, err)
	}
	zones := map[string]bool{}
	for _, z := range s.Zones {
		if !slugPattern.MatchString(z.Slug) || zones[z.Slug] {
			fail("zone %q: slug invalid or duplicated", z.Slug)
		}
		zones[z.Slug] = true
	}
	got := make([]string, 0, len(zones))
	for z := range zones {
		got = append(got, z)
	}
	slices.Sort(got)
	if !slices.Equal(got, zoneSlugs) {
		fail("zones: must be exactly the five documented campus zones %v, got %v", zoneSlugs, got)
	}
	locations := map[string]bool{}
	for _, l := range s.Locations {
		if !slugPattern.MatchString(l.Slug) || locations[l.Slug] {
			fail("location %q: slug invalid or duplicated", l.Slug)
		}
		locations[l.Slug] = true
		if !zones[l.ZoneSlug] {
			fail("location %q: unknown zone %q", l.Slug, l.ZoneSlug)
		}
		if !kinds[l.Kind] {
			fail("location %q: unknown kind %q", l.Slug, l.Kind)
		}
		if strings.TrimSpace(l.Name) == "" {
			fail("location %q: name is empty", l.Slug)
		}
		if box != nil && !box.Contains(l.Lat, l.Lng) {
			fail("location %q: coordinate outside the campus bounding box", l.Slug)
		}
		seen := map[string]bool{strings.ToLower(l.Name): true}
		for _, a := range l.Aliases {
			if seen[strings.ToLower(a)] {
				fail("location %q: alias %q repeats the name or another alias", l.Slug, a)
			}
			seen[strings.ToLower(a)] = true
		}
	}
	posts := map[string]bool{}
	for _, p := range s.Posts {
		if !slugPattern.MatchString(p.Slug) || posts[p.Slug] {
			fail("post %q: slug invalid or duplicated", p.Slug)
		}
		posts[p.Slug] = true
		if !locations[p.LocationSlug] {
			fail("post %q: unknown location %q", p.Slug, p.LocationSlug)
		}
		for day := range weekdays {
			if _, ok := p.OpeningHours[day]; !ok {
				fail("post %q: opening hours must list %s, use [] when closed", p.Slug, day)
			}
		}
		for day, spans := range p.OpeningHours {
			if !weekdays[day] {
				fail("post %q: unknown weekday %q", p.Slug, day)
			}
			for _, span := range spans {
				if len(span) != 2 || !clockTime.MatchString(span[0]) || !clockTime.MatchString(span[1]) || span[0] >= span[1] {
					fail("post %q: opening hours on %s must be [from, to] with from before to", p.Slug, day)
				}
			}
		}
	}
	leaves := map[string]bool{}
	parents := map[string]bool{}
	for _, parent := range s.Categories {
		if !slugPattern.MatchString(parent.Slug) || parents[parent.Slug] {
			fail("category %q: parent slug invalid or duplicated", parent.Slug)
		}
		parents[parent.Slug] = true
		if len(parent.Children) == 0 {
			fail("category %q: a parent needs at least one leaf", parent.Slug)
		}
		for _, leaf := range parent.Children {
			if !slugPattern.MatchString(leaf.Slug) || leaves[leaf.Slug] {
				fail("category %q: leaf slug invalid or duplicated", leaf.Slug)
			}
			leaves[leaf.Slug] = true
			switch leaf.Sensitivity {
			case "normal", "valuable", "identity":
			default:
				fail("category %q: sensitivity must be normal, valuable or identity", leaf.Slug)
			}
			if len(leaf.ClipPrompts) == 0 {
				fail("category %q: at least one clip prompt is required", leaf.Slug)
			}
			if len(leaf.VerificationQuestions) == 0 {
				fail("category %q: at least one verification question is required", leaf.Slug)
			}
			keys := map[string]bool{}
			for _, q := range leaf.VerificationQuestions {
				if q.Key == "" || keys[q.Key] || strings.TrimSpace(q.Question) == "" {
					fail("category %q: verification question keys must be unique and questions non empty", leaf.Slug)
				}
				keys[q.Key] = true
			}
		}
	}
	for _, c := range s.Compat {
		if !leaves[c.A] || !leaves[c.B] || c.A == c.B {
			fail("compatibility %s and %s: both must be distinct leaf categories", c.A, c.B)
		}
		if c.Factor < 0 || c.Factor > 1 {
			fail("compatibility %s and %s: factor must be between 0 and 1", c.A, c.B)
		}
	}
	colours := map[string]bool{}
	for _, c := range s.Colours {
		if c.Name == "" || colours[c.Name] || !hexPattern.MatchString(c.Hex) {
			fail("colour %q: name must be unique and hex like #1E40AF", c.Name)
		}
		colours[c.Name] = true
	}
	if len(s.Colours) != 14 {
		fail("colours.csv: the palette has exactly 14 colours, found %d", len(s.Colours))
	}
	if _, ok := s.Settings["auth.allowed_email_domains"]; !ok {
		fail("settings.yaml: auth.allowed_email_domains is required")
	}
	return errors.Join(errs...)
}

type SeedSummary struct {
	Zones, Locations, Posts, Categories, Compat, Colours, Brands, Synonyms, Settings int
}

// upserts by natural key and never deletes, so it is safe on a live database
// an upsert returns no row when the existing row belongs to an administrator (US-07), the id is then looked up
func kept(id uuid.UUID, err error) func(func() (uuid.UUID, error)) (uuid.UUID, error) {
	return func(lookup func() (uuid.UUID, error)) (uuid.UUID, error) {
		if db.IsNoRows(err) {
			return lookup()
		}
		return id, err
	}
}

func ApplySeed(ctx context.Context, pool *pgxpool.Pool, s SeedSet) (SeedSummary, error) {
	if err := s.Validate(); err != nil {
		return SeedSummary{}, err
	}
	var sum SeedSummary
	err := db.InTx(ctx, pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		zoneIDs := map[string]uuid.UUID{}
		for _, z := range s.Zones {
			id, err := kept(q.UpsertZone(ctx, sqlc.UpsertZoneParams{ID: ids.New(), Slug: z.Slug, Name: z.Name, CentroidLat: z.Lat, CentroidLng: z.Lng}))(func() (uuid.UUID, error) {
				return q.ZoneIDBySlug(ctx, z.Slug)
			})
			if err != nil {
				return fmt.Errorf("zone %s: %w", z.Slug, err)
			}
			zoneIDs[z.Slug] = id
			sum.Zones++
		}
		locIDs := map[string]uuid.UUID{}
		for _, l := range s.Locations {
			id, err := kept(q.UpsertLocation(ctx, sqlc.UpsertLocationParams{
				ID: ids.New(), ZoneID: zoneIDs[l.ZoneSlug], Slug: l.Slug, Name: l.Name, Aliases: l.Aliases, Kind: l.Kind, Lat: l.Lat, Lng: l.Lng,
			}))(func() (uuid.UUID, error) { return q.LocationIDBySlug(ctx, l.Slug) })
			if err != nil {
				return fmt.Errorf("location %s: %w", l.Slug, err)
			}
			locIDs[l.Slug] = id
			sum.Locations++
		}
		for _, p := range s.Posts {
			hours, _ := json.Marshal(p.OpeningHours)
			var phone *string
			if p.Phone != "" {
				phone = &p.Phone
			}
			if _, err := q.UpsertPost(ctx, sqlc.UpsertPostParams{
				ID: ids.New(), Slug: p.Slug, Name: p.Name, LocationID: locIDs[p.LocationSlug], Phone: phone, OpeningHours: hours,
			}); err != nil && !db.IsNoRows(err) {
				return fmt.Errorf("post %s: %w", p.Slug, err)
			}
			sum.Posts++
		}
		leafIDs := map[string]uuid.UUID{}
		for i, parent := range s.Categories {
			order := parent.SortOrder
			if order == 0 {
				order = int32(i + 1)
			}
			pid, err := kept(q.UpsertParentCategory(ctx, sqlc.UpsertParentCategoryParams{ID: ids.New(), Slug: parent.Slug, Name: parent.Name, SortOrder: order}))(
				func() (uuid.UUID, error) { return q.ParentCategoryIDBySlug(ctx, parent.Slug) })
			if err != nil {
				return fmt.Errorf("category %s: %w", parent.Slug, err)
			}
			sum.Categories++
			for j, leaf := range parent.Children {
				qs, _ := json.Marshal(leaf.VerificationQuestions)
				sens := leaf.Sensitivity
				id, err := kept(q.UpsertLeafCategory(ctx, sqlc.UpsertLeafCategoryParams{
					ID: ids.New(), ParentID: &pid, Slug: leaf.Slug, Name: leaf.Name, Sensitivity: &sens,
					RetentionDays: leaf.RetentionDays, VerificationQuestions: qs, ClipPrompts: leaf.ClipPrompts, SortOrder: int32(j + 1),
				}))(func() (uuid.UUID, error) { return q.LeafCategoryIDBySlug(ctx, leaf.Slug) })
				if err != nil {
					return fmt.Errorf("category %s: %w", leaf.Slug, err)
				}
				leafIDs[leaf.Slug] = id
				sum.Categories++
			}
		}
		for _, c := range s.Compat {
			a, b := leafIDs[c.A], leafIDs[c.B]
			if a.String() > b.String() {
				a, b = b, a
			}
			if err := q.UpsertCompatibility(ctx, sqlc.UpsertCompatibilityParams{CategoryA: a, CategoryB: b, Factor: c.Factor}); err != nil {
				return fmt.Errorf("compatibility %s %s: %w", c.A, c.B, err)
			}
			sum.Compat++
		}
		for _, c := range s.Colours {
			if err := q.UpsertColour(ctx, sqlc.UpsertColourParams{Name: c.Name, Hex: c.Hex, SortOrder: c.Order}); err != nil {
				return err
			}
			sum.Colours++
		}
		for _, b := range s.Brands {
			if err := q.UpsertBrand(ctx, sqlc.UpsertBrandParams{Name: b.Name, Aliases: b.Aliases}); err != nil {
				return err
			}
			sum.Brands++
		}
		for _, syn := range s.Synonyms {
			if err := q.UpsertSynonym(ctx, sqlc.UpsertSynonymParams{Term: syn.Term, Expansion: syn.Expansion}); err != nil {
				return err
			}
			sum.Synonyms++
		}
		keys := make([]string, 0, len(s.Settings))
		for k := range s.Settings {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			b, err := json.Marshal(s.Settings[k])
			if err != nil {
				return fmt.Errorf("setting %s: %w", k, err)
			}
			if err := q.InsertSettingIfMissing(ctx, sqlc.InsertSettingIfMissingParams{Key: k, Value: b}); err != nil {
				return err
			}
			sum.Settings++
		}
		return nil
	})
	return sum, err
}
