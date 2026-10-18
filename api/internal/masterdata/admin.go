package masterdata

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
)

// US-07 and FR-ADMIN-02: administrators create, edit and switch off master data. Rows are never deleted, so every report
// keeps its category, location and post (FR-MD-07), and every change is audited with its before and after values

func text(s string) string { return strings.Join(strings.Fields(s), " ") }

func checkName(v *apperr.Collector, field, name string) {
	v.Check(len([]rune(name)) >= 2, field, "too_short", "Minimal 2 karakter")
}

func checkSlug(v *apperr.Collector, slug string) {
	v.Check(slugPattern.MatchString(slug), "slug", "invalid", "Gunakan huruf kecil, angka dan tanda hubung, misalnya pos-teknik")
}

func (s *Service) checkPoint(ctx context.Context, v *apperr.Collector, prefix string, lat, lng float64) {
	box, ok := s.settings.CampusBox(ctx)
	if ok && !box.Contains(lat, lng) {
		v.Add(prefix+"lat", "outside_campus", "Koordinat berada di luar area kampus")
	}
}

func slugTaken(err error) error {
	return apperr.Validation(apperr.FieldError{Field: "slug", Code: "taken", Message: "Slug sudah dipakai"}).WithCause(err)
}

func changes(before, after map[string]any) map[string]any {
	out := map[string]any{}
	for k, a := range after {
		b := before[k]
		bj, _ := json.Marshal(b)
		aj, _ := json.Marshal(a)
		if string(bj) != string(aj) {
			out[k] = audit.Change(b, a)
		}
	}
	return out
}

func questions(in []openapi.QuestionInput) []Question {
	out := make([]Question, 0, len(in))
	for _, q := range in {
		hint := ""
		if q.CheckHint != nil {
			hint = text(*q.CheckHint)
		}
		out = append(out, Question{Key: strings.TrimSpace(q.Key), Question: text(q.Question), Required: q.Required, CheckHint: hint})
	}
	return out
}

func prompts(in []string) []string {
	out := []string{}
	for _, p := range in {
		if p = text(p); p != "" && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

var questionKey = slugPattern

func checkLeaf(v *apperr.Collector, sens *string, qs []Question, clip []string) {
	v.Check(sens != nil, "sensitivity", "required", "Pilih tingkat kerahasiaan")
	v.Check(len(clip) > 0, "clip_prompts", "required", "Minimal satu prompt untuk klasifikasi foto")
	v.Check(len(qs) > 0, "verification_questions", "required", "Minimal satu pertanyaan verifikasi")
	keys := map[string]bool{}
	for _, q := range qs {
		if !questionKey.MatchString(strings.ReplaceAll(q.Key, "_", "-")) || keys[q.Key] || q.Question == "" {
			v.Add("verification_questions", "invalid", "Setiap pertanyaan butuh kunci unik dan teks pertanyaan")
			break
		}
		keys[q.Key] = true
	}
}

func categoryView(c sqlc.AdminListCategoriesRow) openapi.AdminCategory {
	var qs []Question
	_ = json.Unmarshal(c.VerificationQuestions, &qs)
	out := openapi.AdminCategory{
		Id: c.ID, ParentId: c.ParentID, Slug: c.Slug, Name: c.Name, ClipPrompts: c.ClipPrompts, SortOrder: int(c.SortOrder),
		Active: c.Active, EditedAt: c.EditedAt, UpdatedAt: c.UpdatedAt, VerificationQuestions: []openapi.VerificationQuestion{},
	}
	if c.ClipPrompts == nil {
		out.ClipPrompts = []string{}
	}
	if c.ParentName != "" {
		name := c.ParentName
		out.ParentName = &name
	}
	if c.Sensitivity != nil {
		v := openapi.Sensitivity(*c.Sensitivity)
		out.Sensitivity = &v
	}
	if c.RetentionDays != nil {
		d := int(*c.RetentionDays)
		out.RetentionDays = &d
	}
	for _, q := range qs {
		hint := q.CheckHint
		out.VerificationQuestions = append(out.VerificationQuestions, openapi.VerificationQuestion{Key: q.Key, Question: q.Question, Required: q.Required, CheckHint: &hint})
	}
	return out
}

func (s *Service) AdminCategories(ctx context.Context) ([]openapi.AdminCategory, error) {
	rows, err := s.q.AdminListCategories(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]openapi.AdminCategory, 0, len(rows))
	for _, r := range rows {
		out = append(out, categoryView(r))
	}
	return out, nil
}

func (s *Service) adminCategory(ctx context.Context, id uuid.UUID) (openapi.AdminCategory, error) {
	rows, err := s.q.AdminListCategories(ctx)
	if err != nil {
		return openapi.AdminCategory{}, err
	}
	for _, r := range rows {
		if r.ID == id {
			return categoryView(r), nil
		}
	}
	return openapi.AdminCategory{}, apperr.NotFound()
}

func categoryFields(c sqlc.Category) map[string]any {
	var qs []Question
	_ = json.Unmarshal(c.VerificationQuestions, &qs)
	return map[string]any{
		"name": c.Name, "sensitivity": c.Sensitivity, "retention_days": c.RetentionDays, "verification_questions": qs,
		"clip_prompts": c.ClipPrompts, "sort_order": c.SortOrder, "active": c.Active,
	}
}

func (s *Service) CreateCategory(ctx context.Context, in openapi.CategoryCreate) (openapi.AdminCategory, error) {
	var v apperr.Collector
	slug, name := strings.TrimSpace(in.Slug), text(in.Name)
	checkSlug(&v, slug)
	checkName(&v, "name", name)
	var qs []Question
	var clip []string
	var sens *string
	if in.VerificationQuestions != nil {
		qs = questions(*in.VerificationQuestions)
	}
	if in.ClipPrompts != nil {
		clip = prompts(*in.ClipPrompts)
	}
	if in.Sensitivity != nil {
		v := string(*in.Sensitivity)
		sens = &v
	}
	var retention *int32
	if in.RetentionDays != nil {
		d := int32(*in.RetentionDays)
		retention = &d
	}
	order := int32(0)
	if in.SortOrder != nil {
		order = int32(*in.SortOrder)
	}
	id := ids.New()
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if in.ParentId != nil {
			parent, err := q.CategoryForUpdate(ctx, *in.ParentId)
			if db.IsNoRows(err) || (err == nil && parent.ParentID != nil) {
				v.Add("parent_id", "unknown", "Induk kategori tidak ditemukan")
			} else if err != nil {
				return err
			}
			checkLeaf(&v, sens, qs, clip)
		} else {
			v.Check(sens == nil && len(qs) == 0 && len(clip) == 0 && retention == nil, "parent_id", "parent_fields",
				"Kategori induk tidak memakai kerahasiaan, pertanyaan atau prompt")
		}
		if err := v.Err(); err != nil {
			return err
		}
		raw, _ := json.Marshal(qs)
		if qs == nil {
			raw = []byte("[]")
		}
		if clip == nil {
			clip = []string{}
		}
		if err := q.InsertCategory(ctx, sqlc.InsertCategoryParams{
			ID: id, ParentID: in.ParentId, Slug: slug, Name: name, Sensitivity: sens, RetentionDays: retention,
			VerificationQuestions: raw, ClipPrompts: clip, SortOrder: order, Now: s.clock.Now(),
		}); db.IsUniqueViolation(err, "categories_parent_slug") || db.IsUniqueViolation(err, "categories_leaf_slug") {
			return slugTaken(err)
		} else if err != nil {
			return err
		}
		return audit.Record(ctx, q, s.clock.Now(), audit.Entry{Action: "masterdata.category.create", EntityType: "category", EntityID: id,
			Changes: map[string]any{"slug": slug, "name": name, "parent_id": in.ParentId, "sensitivity": sens}})
	})
	if err != nil {
		return openapi.AdminCategory{}, err
	}
	return s.adminCategory(ctx, id)
}

// US-07 criterion 2: questions change the form for new claims only, a claim keeps the questions it was asked
func (s *Service) UpdateCategory(ctx context.Context, id uuid.UUID, in openapi.CategoryUpdate) (openapi.AdminCategory, error) {
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		c, err := q.CategoryForUpdate(ctx, id)
		if db.IsNoRows(err) {
			return apperr.NotFound()
		}
		if err != nil {
			return err
		}
		before := categoryFields(c)
		var v apperr.Collector
		if in.Name != nil {
			c.Name = text(*in.Name)
			checkName(&v, "name", c.Name)
		}
		if in.SortOrder != nil {
			c.SortOrder = int32(*in.SortOrder)
		}
		if in.Active != nil {
			c.Active = *in.Active
		}
		leaf := c.ParentID != nil
		if !leaf && (in.Sensitivity != nil || in.RetentionDays != nil || in.VerificationQuestions != nil || in.ClipPrompts != nil) {
			v.Add("sensitivity", "parent_fields", "Kategori induk tidak memakai kerahasiaan, pertanyaan atau prompt")
		}
		if in.Sensitivity != nil {
			sv := string(*in.Sensitivity)
			c.Sensitivity = &sv
		}
		if in.RetentionDays != nil {
			if *in.RetentionDays == 0 {
				c.RetentionDays = nil
			} else {
				d := int32(*in.RetentionDays)
				c.RetentionDays = &d
			}
		}
		var qs []Question
		_ = json.Unmarshal(c.VerificationQuestions, &qs)
		if in.VerificationQuestions != nil {
			qs = questions(*in.VerificationQuestions)
		}
		if in.ClipPrompts != nil {
			c.ClipPrompts = prompts(*in.ClipPrompts)
		}
		if leaf {
			checkLeaf(&v, c.Sensitivity, qs, c.ClipPrompts)
		}
		if err := v.Err(); err != nil {
			return err
		}
		raw, _ := json.Marshal(qs)
		if qs == nil {
			raw = []byte("[]")
		}
		c.VerificationQuestions = raw
		now := s.clock.Now()
		if err := q.UpdateCategory(ctx, sqlc.UpdateCategoryParams{
			ID: id, Name: c.Name, Sensitivity: c.Sensitivity, RetentionDays: c.RetentionDays, VerificationQuestions: raw,
			ClipPrompts: c.ClipPrompts, SortOrder: c.SortOrder, Active: c.Active, Now: now,
		}); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "masterdata.category.update", EntityType: "category", EntityID: id,
			Changes: changes(before, categoryFields(c))})
	})
	if err != nil {
		return openapi.AdminCategory{}, err
	}
	return s.adminCategory(ctx, id)
}

func (s *Service) AdminCompatibility(ctx context.Context) ([]openapi.CompatibilityPair, error) {
	rows, err := s.q.AdminListCompatibility(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]openapi.CompatibilityPair, 0, len(rows))
	for _, r := range rows {
		out = append(out, openapi.CompatibilityPair{CategoryA: r.CategoryA, CategoryB: r.CategoryB, NameA: r.NameA, NameB: r.NameB, Factor: float32(r.Factor), Active: r.Active})
	}
	return out, nil
}

func (s *Service) PutCompatibility(ctx context.Context, in openapi.CompatibilityPut) (openapi.CompatibilityPair, error) {
	a, b := in.CategoryA, in.CategoryB
	if a.String() > b.String() {
		a, b = b, a
	}
	factor := float64(in.Factor)
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var v apperr.Collector
		v.Check(a != b, "category_b", "same", "Pilih dua kategori yang berbeda")
		for _, id := range []uuid.UUID{a, b} {
			c, err := q.CategoryForUpdate(ctx, id)
			if db.IsNoRows(err) || (err == nil && c.ParentID == nil) {
				v.Add("category_a", "not_leaf", "Kompatibilitas hanya untuk subkategori")
			} else if err != nil {
				return err
			}
		}
		if err := v.Err(); err != nil {
			return err
		}
		before := map[string]any{}
		if prev, err := q.CompatibilityForUpdate(ctx, sqlc.CompatibilityForUpdateParams{CategoryA: a, CategoryB: b}); err == nil {
			before = map[string]any{"factor": prev.Factor, "active": prev.Active}
		} else if !db.IsNoRows(err) {
			return err
		}
		now := s.clock.Now()
		if err := q.PutCompatibility(ctx, sqlc.PutCompatibilityParams{CategoryA: a, CategoryB: b, Factor: factor, Active: in.Active, Now: &now}); err != nil {
			return err
		}
		ch := changes(before, map[string]any{"factor": factor, "active": in.Active})
		ch["category_b"] = b
		return audit.Record(ctx, q, now, audit.Entry{Action: "masterdata.compatibility.put", EntityType: "category", EntityID: a, Changes: ch})
	})
	if err != nil {
		return openapi.CompatibilityPair{}, err
	}
	pairs, err := s.AdminCompatibility(ctx)
	if err != nil {
		return openapi.CompatibilityPair{}, err
	}
	for _, p := range pairs {
		if p.CategoryA == a && p.CategoryB == b {
			return p, nil
		}
	}
	return openapi.CompatibilityPair{}, apperr.NotFound()
}

func zoneView(z sqlc.Zone) openapi.AdminZone {
	return openapi.AdminZone{
		Id: z.ID, Slug: z.Slug, Name: z.Name, CentroidLat: z.CentroidLat, CentroidLng: z.CentroidLng, Active: z.Active,
		EditedAt: z.EditedAt, UpdatedAt: z.UpdatedAt,
	}
}

func zoneFields(z sqlc.Zone) map[string]any {
	return map[string]any{"name": z.Name, "centroid_lat": z.CentroidLat, "centroid_lng": z.CentroidLng, "active": z.Active}
}

func (s *Service) AdminZones(ctx context.Context) ([]openapi.AdminZone, error) {
	rows, err := s.q.AdminListZones(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]openapi.AdminZone, 0, len(rows))
	for _, z := range rows {
		out = append(out, zoneView(z))
	}
	return out, nil
}

func (s *Service) CreateZone(ctx context.Context, in openapi.ZoneCreate) (openapi.AdminZone, error) {
	var v apperr.Collector
	slug, name := strings.TrimSpace(in.Slug), text(in.Name)
	checkSlug(&v, slug)
	checkName(&v, "name", name)
	s.checkPoint(ctx, &v, "centroid_", in.CentroidLat, in.CentroidLng)
	if err := v.Err(); err != nil {
		return openapi.AdminZone{}, err
	}
	id := ids.New()
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.InsertZone(ctx, sqlc.InsertZoneParams{ID: id, Slug: slug, Name: name, CentroidLat: in.CentroidLat, CentroidLng: in.CentroidLng, Now: s.clock.Now()}); db.IsUniqueViolation(err, "zones_slug_key") {
			return slugTaken(err)
		} else if err != nil {
			return err
		}
		return audit.Record(ctx, q, s.clock.Now(), audit.Entry{Action: "masterdata.zone.create", EntityType: "zone", EntityID: id,
			Changes: map[string]any{"slug": slug, "name": name, "centroid_lat": in.CentroidLat, "centroid_lng": in.CentroidLng}})
	})
	if err != nil {
		return openapi.AdminZone{}, err
	}
	z, err := s.q.GetZone(ctx, id)
	return zoneView(z), err
}

func (s *Service) UpdateZone(ctx context.Context, id uuid.UUID, in openapi.ZoneUpdate) (openapi.AdminZone, error) {
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		z, err := q.ZoneForUpdate(ctx, id)
		if db.IsNoRows(err) {
			return apperr.NotFound()
		}
		if err != nil {
			return err
		}
		before := zoneFields(z)
		var v apperr.Collector
		if in.Name != nil {
			z.Name = text(*in.Name)
			checkName(&v, "name", z.Name)
		}
		if in.CentroidLat != nil {
			z.CentroidLat = *in.CentroidLat
		}
		if in.CentroidLng != nil {
			z.CentroidLng = *in.CentroidLng
		}
		if in.Active != nil {
			z.Active = *in.Active
		}
		s.checkPoint(ctx, &v, "centroid_", z.CentroidLat, z.CentroidLng)
		if err := v.Err(); err != nil {
			return err
		}
		now := s.clock.Now()
		if err := q.UpdateZone(ctx, sqlc.UpdateZoneParams{ID: id, Name: z.Name, CentroidLat: z.CentroidLat, CentroidLng: z.CentroidLng, Active: z.Active, Now: now}); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "masterdata.zone.update", EntityType: "zone", EntityID: id, Changes: changes(before, zoneFields(z))})
	})
	if err != nil {
		return openapi.AdminZone{}, err
	}
	z, err := s.q.GetZone(ctx, id)
	return zoneView(z), err
}

func locationView(l sqlc.AdminListLocationsRow) openapi.AdminLocation {
	out := openapi.AdminLocation{
		Id: l.ID, ZoneId: l.ZoneID, ZoneName: l.ZoneName, Slug: l.Slug, Name: l.Name, Aliases: l.Aliases, Kind: openapi.LocationKind(l.Kind),
		Lat: l.Lat, Lng: l.Lng, Active: l.Active, EditedAt: l.EditedAt, UpdatedAt: l.UpdatedAt,
	}
	if out.Aliases == nil {
		out.Aliases = []string{}
	}
	return out
}

func locationFields(l sqlc.Location) map[string]any {
	return map[string]any{"zone_id": l.ZoneID, "name": l.Name, "aliases": l.Aliases, "kind": l.Kind, "lat": l.Lat, "lng": l.Lng, "active": l.Active}
}

func (s *Service) AdminLocations(ctx context.Context, qtext *string, zone *uuid.UUID) ([]openapi.AdminLocation, error) {
	if qtext != nil {
		if t := strings.TrimSpace(*qtext); t == "" {
			qtext = nil
		} else {
			qtext = &t
		}
	}
	rows, err := s.q.AdminListLocations(ctx, sqlc.AdminListLocationsParams{ZoneID: zone, Q: qtext})
	if err != nil {
		return nil, err
	}
	out := make([]openapi.AdminLocation, 0, len(rows))
	for _, l := range rows {
		out = append(out, locationView(l))
	}
	return out, nil
}

func (s *Service) adminLocation(ctx context.Context, id uuid.UUID) (openapi.AdminLocation, error) {
	l, err := s.q.GetLocation(ctx, id)
	if err != nil {
		return openapi.AdminLocation{}, err
	}
	return locationView(sqlc.AdminListLocationsRow(l)), nil
}

func aliases(name string, in []string, v *apperr.Collector) []string {
	out := []string{}
	seen := map[string]bool{strings.ToLower(name): true}
	for _, a := range in {
		a = text(a)
		if a == "" {
			continue
		}
		if seen[strings.ToLower(a)] {
			v.Add("aliases", "duplicate", "Alias tidak boleh mengulang nama atau alias lain")
			continue
		}
		seen[strings.ToLower(a)] = true
		out = append(out, a)
	}
	return out
}

func (s *Service) zoneExists(ctx context.Context, q *sqlc.Queries, v *apperr.Collector, id uuid.UUID) error {
	if _, err := q.GetZone(ctx, id); db.IsNoRows(err) {
		v.Add("zone_id", "unknown", "Zona tidak ditemukan")
	} else if err != nil {
		return err
	}
	return nil
}

func (s *Service) CreateLocation(ctx context.Context, in openapi.LocationCreate) (openapi.AdminLocation, error) {
	var v apperr.Collector
	slug, name := strings.TrimSpace(in.Slug), text(in.Name)
	checkSlug(&v, slug)
	checkName(&v, "name", name)
	var al []string
	if in.Aliases != nil {
		al = aliases(name, *in.Aliases, &v)
	} else {
		al = []string{}
	}
	s.checkPoint(ctx, &v, "", in.Lat, in.Lng)
	id := ids.New()
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := s.zoneExists(ctx, q, &v, in.ZoneId); err != nil {
			return err
		}
		if err := v.Err(); err != nil {
			return err
		}
		now := s.clock.Now()
		if err := q.InsertLocation(ctx, sqlc.InsertLocationParams{
			ID: id, ZoneID: in.ZoneId, Slug: slug, Name: name, Aliases: al, Kind: string(in.Kind), Lat: in.Lat, Lng: in.Lng, Now: now,
		}); db.IsUniqueViolation(err, "locations_slug_key") {
			return slugTaken(err)
		} else if err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "masterdata.location.create", EntityType: "location", EntityID: id,
			Changes: map[string]any{"slug": slug, "name": name, "zone_id": in.ZoneId, "kind": in.Kind, "lat": in.Lat, "lng": in.Lng, "aliases": al}})
	})
	if err != nil {
		return openapi.AdminLocation{}, err
	}
	return s.adminLocation(ctx, id)
}

func (s *Service) UpdateLocation(ctx context.Context, id uuid.UUID, in openapi.LocationUpdate) (openapi.AdminLocation, error) {
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		l, err := q.LocationForUpdate(ctx, id)
		if db.IsNoRows(err) {
			return apperr.NotFound()
		}
		if err != nil {
			return err
		}
		before := locationFields(l)
		var v apperr.Collector
		if in.Name != nil {
			l.Name = text(*in.Name)
			checkName(&v, "name", l.Name)
		}
		if in.Aliases != nil {
			l.Aliases = aliases(l.Name, *in.Aliases, &v)
		}
		if in.Kind != nil {
			l.Kind = string(*in.Kind)
		}
		if in.Lat != nil {
			l.Lat = *in.Lat
		}
		if in.Lng != nil {
			l.Lng = *in.Lng
		}
		if in.Active != nil {
			l.Active = *in.Active
		}
		if in.ZoneId != nil {
			l.ZoneID = *in.ZoneId
			if err := s.zoneExists(ctx, q, &v, l.ZoneID); err != nil {
				return err
			}
		}
		s.checkPoint(ctx, &v, "", l.Lat, l.Lng)
		if err := v.Err(); err != nil {
			return err
		}
		now := s.clock.Now()
		if err := q.UpdateLocation(ctx, sqlc.UpdateLocationParams{
			ID: id, ZoneID: l.ZoneID, Name: l.Name, Aliases: l.Aliases, Kind: l.Kind, Lat: l.Lat, Lng: l.Lng, Active: l.Active, Now: now,
		}); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "masterdata.location.update", EntityType: "location", EntityID: id,
			Changes: changes(before, locationFields(l))})
	})
	if err != nil {
		return openapi.AdminLocation{}, err
	}
	return s.adminLocation(ctx, id)
}

func postView(p sqlc.AdminListPostsRow) openapi.AdminPost {
	hours := openapi.OpeningHours{}
	_ = json.Unmarshal(p.OpeningHours, &hours)
	return openapi.AdminPost{
		Id: p.ID, Slug: p.Slug, Name: p.Name, LocationId: p.LocationID, LocationName: p.LocationName, ZoneName: p.ZoneName,
		Phone: p.Phone, OpeningHours: hours, Active: p.Active, EditedAt: p.EditedAt, UpdatedAt: p.UpdatedAt,
	}
}

func postFields(p sqlc.Post) map[string]any {
	var hours map[string][][]string
	_ = json.Unmarshal(p.OpeningHours, &hours)
	return map[string]any{"name": p.Name, "location_id": p.LocationID, "phone": p.Phone, "opening_hours": hours, "active": p.Active}
}

func (s *Service) AdminPosts(ctx context.Context) ([]openapi.AdminPost, error) {
	rows, err := s.q.AdminListPosts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]openapi.AdminPost, 0, len(rows))
	for _, p := range rows {
		out = append(out, postView(p))
	}
	return out, nil
}

func (s *Service) adminPost(ctx context.Context, id uuid.UUID) (openapi.AdminPost, error) {
	rows, err := s.AdminPosts(ctx)
	if err != nil {
		return openapi.AdminPost{}, err
	}
	for _, p := range rows {
		if p.Id == id {
			return p, nil
		}
	}
	return openapi.AdminPost{}, apperr.NotFound()
}

// the opening hours rule of the seed contract: all seven weekdays, each span from before to
func checkHours(v *apperr.Collector, hours openapi.OpeningHours) []byte {
	for day := range weekdays {
		if _, ok := hours[day]; !ok {
			v.Add("opening_hours", "incomplete", "Isi jam buka untuk ketujuh hari, kosongkan hari libur")
			break
		}
	}
	for day, spans := range hours {
		if !weekdays[day] {
			v.Add("opening_hours", "unknown_day", "Hari tidak dikenal")
			continue
		}
		for _, span := range spans {
			if len(span) != 2 || !clockTime.MatchString(span[0]) || !clockTime.MatchString(span[1]) || span[0] >= span[1] {
				v.Add("opening_hours", "invalid_span", "Jam buka harus berupa jam mulai sebelum jam tutup, misalnya 08:00 sampai 16:00")
				break
			}
		}
	}
	raw, _ := json.Marshal(hours)
	return raw
}

func phone(p *string) *string {
	if p == nil {
		return nil
	}
	t := strings.TrimSpace(*p)
	if t == "" {
		return nil
	}
	return &t
}

func (s *Service) locationExists(ctx context.Context, q *sqlc.Queries, v *apperr.Collector, id uuid.UUID) error {
	if l, err := q.GetLocation(ctx, id); db.IsNoRows(err) || (err == nil && !l.Active) {
		v.Add("location_id", "unknown", "Lokasi tidak ditemukan atau tidak aktif")
	} else if err != nil {
		return err
	}
	return nil
}

func (s *Service) CreatePost(ctx context.Context, in openapi.PostCreate) (openapi.AdminPost, error) {
	var v apperr.Collector
	slug, name := strings.TrimSpace(in.Slug), text(in.Name)
	checkSlug(&v, slug)
	checkName(&v, "name", name)
	hours := checkHours(&v, in.OpeningHours)
	id := ids.New()
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := s.locationExists(ctx, q, &v, in.LocationId); err != nil {
			return err
		}
		if err := v.Err(); err != nil {
			return err
		}
		now := s.clock.Now()
		if err := q.InsertPost(ctx, sqlc.InsertPostParams{
			ID: id, Slug: slug, Name: name, LocationID: in.LocationId, Phone: phone(in.Phone), OpeningHours: hours, Now: now,
		}); db.IsUniqueViolation(err, "posts_slug_key") {
			return slugTaken(err)
		} else if err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "masterdata.post.create", EntityType: "post", EntityID: id,
			Changes: map[string]any{"slug": slug, "name": name, "location_id": in.LocationId, "phone": phone(in.Phone), "opening_hours": in.OpeningHours}})
	})
	if err != nil {
		return openapi.AdminPost{}, err
	}
	return s.adminPost(ctx, id)
}

func (s *Service) UpdatePost(ctx context.Context, id uuid.UUID, in openapi.PostUpdate) (openapi.AdminPost, error) {
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		p, err := q.PostForUpdate(ctx, id)
		if db.IsNoRows(err) {
			return apperr.NotFound()
		}
		if err != nil {
			return err
		}
		before := postFields(p)
		var v apperr.Collector
		if in.Name != nil {
			p.Name = text(*in.Name)
			checkName(&v, "name", p.Name)
		}
		if in.Phone != nil {
			p.Phone = phone(in.Phone)
		}
		if in.OpeningHours != nil {
			p.OpeningHours = checkHours(&v, *in.OpeningHours)
		}
		if in.Active != nil {
			p.Active = *in.Active
		}
		if in.LocationId != nil && *in.LocationId != p.LocationID {
			p.LocationID = *in.LocationId
			if err := s.locationExists(ctx, q, &v, p.LocationID); err != nil {
				return err
			}
		}
		if err := v.Err(); err != nil {
			return err
		}
		now := s.clock.Now()
		if err := q.UpdatePost(ctx, sqlc.UpdatePostParams{
			ID: id, Name: p.Name, LocationID: p.LocationID, Phone: p.Phone, OpeningHours: p.OpeningHours, Active: p.Active, Now: now,
		}); err != nil {
			return err
		}
		return audit.Record(ctx, q, now, audit.Entry{Action: "masterdata.post.update", EntityType: "post", EntityID: id, Changes: changes(before, postFields(p))})
	})
	if err != nil {
		return openapi.AdminPost{}, err
	}
	return s.adminPost(ctx, id)
}

func (h *Handlers) AdminListCategories(ctx context.Context, _ openapi.AdminListCategoriesRequestObject) (openapi.AdminListCategoriesResponseObject, error) {
	items, err := h.svc.AdminCategories(ctx)
	if err != nil {
		return nil, err
	}
	return openapi.AdminListCategories200JSONResponse{Items: items}, nil
}

func (h *Handlers) AdminCreateCategory(ctx context.Context, req openapi.AdminCreateCategoryRequestObject) (openapi.AdminCreateCategoryResponseObject, error) {
	c, err := h.svc.CreateCategory(ctx, *req.Body)
	if err != nil {
		return nil, err
	}
	return openapi.AdminCreateCategory201JSONResponse(c), nil
}

func (h *Handlers) AdminUpdateCategory(ctx context.Context, req openapi.AdminUpdateCategoryRequestObject) (openapi.AdminUpdateCategoryResponseObject, error) {
	c, err := h.svc.UpdateCategory(ctx, req.Id, *req.Body)
	if err != nil {
		return nil, err
	}
	return openapi.AdminUpdateCategory200JSONResponse(c), nil
}

func (h *Handlers) AdminListCompatibility(ctx context.Context, _ openapi.AdminListCompatibilityRequestObject) (openapi.AdminListCompatibilityResponseObject, error) {
	items, err := h.svc.AdminCompatibility(ctx)
	if err != nil {
		return nil, err
	}
	return openapi.AdminListCompatibility200JSONResponse{Items: items}, nil
}

func (h *Handlers) AdminPutCompatibility(ctx context.Context, req openapi.AdminPutCompatibilityRequestObject) (openapi.AdminPutCompatibilityResponseObject, error) {
	p, err := h.svc.PutCompatibility(ctx, *req.Body)
	if err != nil {
		return nil, err
	}
	return openapi.AdminPutCompatibility200JSONResponse(p), nil
}

func (h *Handlers) AdminListZones(ctx context.Context, _ openapi.AdminListZonesRequestObject) (openapi.AdminListZonesResponseObject, error) {
	items, err := h.svc.AdminZones(ctx)
	if err != nil {
		return nil, err
	}
	return openapi.AdminListZones200JSONResponse{Items: items}, nil
}

func (h *Handlers) AdminCreateZone(ctx context.Context, req openapi.AdminCreateZoneRequestObject) (openapi.AdminCreateZoneResponseObject, error) {
	z, err := h.svc.CreateZone(ctx, *req.Body)
	if err != nil {
		return nil, err
	}
	return openapi.AdminCreateZone201JSONResponse(z), nil
}

func (h *Handlers) AdminUpdateZone(ctx context.Context, req openapi.AdminUpdateZoneRequestObject) (openapi.AdminUpdateZoneResponseObject, error) {
	z, err := h.svc.UpdateZone(ctx, req.Id, *req.Body)
	if err != nil {
		return nil, err
	}
	return openapi.AdminUpdateZone200JSONResponse(z), nil
}

func (h *Handlers) AdminListLocations(ctx context.Context, req openapi.AdminListLocationsRequestObject) (openapi.AdminListLocationsResponseObject, error) {
	items, err := h.svc.AdminLocations(ctx, req.Params.Q, req.Params.ZoneId)
	if err != nil {
		return nil, err
	}
	return openapi.AdminListLocations200JSONResponse{Items: items}, nil
}

func (h *Handlers) AdminCreateLocation(ctx context.Context, req openapi.AdminCreateLocationRequestObject) (openapi.AdminCreateLocationResponseObject, error) {
	l, err := h.svc.CreateLocation(ctx, *req.Body)
	if err != nil {
		return nil, err
	}
	return openapi.AdminCreateLocation201JSONResponse(l), nil
}

func (h *Handlers) AdminUpdateLocation(ctx context.Context, req openapi.AdminUpdateLocationRequestObject) (openapi.AdminUpdateLocationResponseObject, error) {
	l, err := h.svc.UpdateLocation(ctx, req.Id, *req.Body)
	if err != nil {
		return nil, err
	}
	return openapi.AdminUpdateLocation200JSONResponse(l), nil
}

func (h *Handlers) AdminListPosts(ctx context.Context, _ openapi.AdminListPostsRequestObject) (openapi.AdminListPostsResponseObject, error) {
	items, err := h.svc.AdminPosts(ctx)
	if err != nil {
		return nil, err
	}
	return openapi.AdminListPosts200JSONResponse{Items: items}, nil
}

func (h *Handlers) AdminCreatePost(ctx context.Context, req openapi.AdminCreatePostRequestObject) (openapi.AdminCreatePostResponseObject, error) {
	p, err := h.svc.CreatePost(ctx, *req.Body)
	if err != nil {
		return nil, err
	}
	return openapi.AdminCreatePost201JSONResponse(p), nil
}

func (h *Handlers) AdminUpdatePost(ctx context.Context, req openapi.AdminUpdatePostRequestObject) (openapi.AdminUpdatePostResponseObject, error) {
	p, err := h.svc.UpdatePost(ctx, req.Id, *req.Body)
	if err != nil {
		return nil, err
	}
	return openapi.AdminUpdatePost200JSONResponse(p), nil
}
