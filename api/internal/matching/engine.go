package matching

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/matching/fusion"
	"github.com/pisondev/simpanugm/api/internal/matching/fuzzy"
	"github.com/pisondev/simpanugm/api/internal/matching/jobargs"
	"github.com/pisondev/simpanugm/api/internal/mlclient"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/geo"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/settings"
	"github.com/pisondev/simpanugm/api/internal/platform/storage"
)

const (
	TypeLost  = "lost"
	TypeFound = "found"
)

// the report changed while it was being scored, the job runs once more
var ErrChangedMeanwhile = errors.New("matching: report changed while scoring")

type Engine struct {
	pool     *pgxpool.Pool
	q        *sqlc.Queries
	clock    clock.Clock
	ml       *mlclient.Client
	store    storage.Store
	notifier *notify.Notifier
	settings *settings.Store
	jobs     *river.Client[pgx.Tx]
}

func NewEngine(pool *pgxpool.Pool, c clock.Clock, ml *mlclient.Client, store storage.Store, n *notify.Notifier, st *settings.Store, jobs *river.Client[pgx.Tx]) *Engine {
	return &Engine{pool: pool, q: sqlc.New(pool), clock: c, ml: ml, store: store, notifier: n, settings: st, jobs: jobs}
}

type place struct {
	Lat, Lng float64
	ZoneOnly bool
	Name     string
}

// one side of a pair with everything scoring needs, loaded in bulk
type report struct {
	Type         string
	ID           uuid.UUID
	Code         string
	Open         bool
	CategoryID   uuid.UUID
	ParentID     uuid.UUID
	CategoryName string
	ParentName   string
	Sensitivity  string
	ItemName     string
	Description  string
	Private      string
	Colours      []string
	Brand        string
	LostFrom     time.Time
	LostUntil    time.Time
	Unknown      bool
	Places       []place
	FoundAt      time.Time
	Lat, Lng     float64
	LocationName string
	OwnerName    string
	ReporterID   *uuid.UUID
	FinderID     *uuid.UUID
	Posts        []uuid.UUID
	VisibleText  string
	HasPhotos    bool
	UpdatedAt    time.Time
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nilID(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}

func (e *Engine) loadLost(ctx context.Context, q *sqlc.Queries, idList []uuid.UUID) (map[uuid.UUID]*report, error) {
	out := map[uuid.UUID]*report{}
	if len(idList) == 0 {
		return out, nil
	}
	rows, err := q.LostForMatching(ctx, idList)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = &report{
			Type: TypeLost, ID: r.ID, Code: r.Code,
			Open:       r.HiddenAt == nil && slices.Contains([]string{"reported", "potential_match", "under_verification"}, r.Status),
			CategoryID: r.CategoryID, ParentID: nilID(r.ParentID), CategoryName: r.CategoryName, ParentName: r.ParentName,
			Sensitivity: str(r.Sensitivity), ItemName: r.ItemName, Description: r.Description, Private: str(r.PrivateDetails),
			Colours: r.Colours, Brand: str(r.Brand), LostFrom: r.LostFrom, LostUntil: r.LostUntil, Unknown: r.LocationUnknown,
			OwnerName: r.OwnerName, ReporterID: r.ReporterID, HasPhotos: r.HasPhotos, UpdatedAt: r.UpdatedAt,
		}
	}
	places, err := q.LostPlacesForMatching(ctx, idList)
	if err != nil {
		return nil, err
	}
	for _, p := range places {
		if r := out[p.LostReportID]; r != nil {
			r.Places = append(r.Places, place{Lat: p.Lat, Lng: p.Lng, ZoneOnly: p.ZoneOnly, Name: p.Name})
		}
	}
	return out, nil
}

func (e *Engine) loadFound(ctx context.Context, q *sqlc.Queries, idList []uuid.UUID) (map[uuid.UUID]*report, error) {
	out := map[uuid.UUID]*report{}
	if len(idList) == 0 {
		return out, nil
	}
	rows, err := q.FoundForMatching(ctx, idList)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		var posts []uuid.UUID
		for _, p := range []*uuid.UUID{r.HoldingPostID, r.IntendedPostID} {
			if p != nil && !slices.Contains(posts, *p) {
				posts = append(posts, *p)
			}
		}
		out[r.ID] = &report{
			Type: TypeFound, ID: r.ID, Code: r.Code,
			Open:       r.HiddenAt == nil && slices.Contains([]string{"reported", "potential_match", "under_verification", "unclaimed"}, r.Status),
			CategoryID: r.CategoryID, ParentID: nilID(r.ParentID), CategoryName: r.CategoryName, ParentName: r.ParentName,
			Sensitivity: str(r.Sensitivity), ItemName: r.ItemName, Description: r.Description, Private: str(r.HiddenDetails),
			Colours: r.Colours, Brand: str(r.Brand), FoundAt: r.FoundAt, Lat: r.Lat, Lng: r.Lng, LocationName: r.LocationName,
			FinderID: r.FinderID, Posts: posts, VisibleText: r.VisibleText, HasPhotos: r.HasPhotos, UpdatedAt: r.UpdatedAt,
		}
	}
	return out, nil
}

func (e *Engine) load(ctx context.Context, q *sqlc.Queries, typ string, id uuid.UUID) (*report, error) {
	var m map[uuid.UUID]*report
	var err error
	if typ == TypeLost {
		m, err = e.loadLost(ctx, q, []uuid.UUID{id})
	} else {
		m, err = e.loadFound(ctx, q, []uuid.UUID{id})
	}
	if err != nil {
		return nil, err
	}
	return m[id], nil
}

type encoders struct {
	Image, XModal, Text uuid.UUID
}

func (e *Engine) encoders(ctx context.Context, q *sqlc.Queries) (encoders, error) {
	var v encoders
	var err error
	if v.Image, err = activeEncoder(ctx, q, ComponentImage); err != nil {
		return v, err
	}
	if v.XModal, err = activeEncoder(ctx, q, ComponentXModal); err != nil {
		return v, err
	}
	v.Text, err = activeEncoder(ctx, q, ComponentText)
	return v, err
}

func opposite(typ string) string {
	if typ == TypeLost {
		return TypeFound
	}
	return TypeLost
}

func (e *Engine) poolWindow(cfg fusion.Config, src *report) (time.Time, time.Time) {
	return src.LostFrom.Add(-time.Duration(cfg.Time.PoolBeforeH * float64(time.Hour))),
		src.LostUntil.Add(time.Duration(cfg.Time.PoolAfterD*24) * time.Hour)
}

// the union of the four generators of section 5.3, plus every pair already stored so a rescoring updates it
func (e *Engine) candidates(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, cfg fusion.Config, src *report, enc encoders) ([]uuid.UUID, error) {
	if _, err := tx.Exec(ctx, "SET LOCAL hnsw.ef_search = 100"); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, "SET LOCAL hnsw.iterative_scan = relaxed_order"); err != nil {
		return nil, err
	}
	k := int32(cfg.AnnK)
	set := map[uuid.UUID]bool{}
	add := func(list []uuid.UUID, err error) error {
		if err != nil {
			return err
		}
		for _, id := range list {
			set[id] = true
		}
		return nil
	}
	if src.Type == TypeLost {
		start, end := e.poolWindow(cfg, src)
		if err := add(q.FoundPoolStructured(ctx, sqlc.FoundPoolStructuredParams{WindowStart: start, WindowEnd: end, CategoryID: src.CategoryID, Anchor: src.LostUntil, K: k})); err != nil {
			return nil, err
		}
		if enc.Image != uuid.Nil {
			vectors, err := q.ImageVectors(ctx, sqlc.ImageVectorsParams{ReportType: TypeLost, ReportID: src.ID, ModelVersionID: enc.Image})
			if err != nil {
				return nil, err
			}
			for _, v := range vectors {
				if err := add(q.FoundNearestImages(ctx, sqlc.FoundNearestImagesParams{Query: v.Embedding, VersionID: enc.Image, WindowStart: start, WindowEnd: end, K: k})); err != nil {
					return nil, err
				}
			}
			if enc.XModal != uuid.Nil {
				if clip, err := q.ClipTextVector(ctx, sqlc.ClipTextVectorParams{ReportType: TypeLost, ReportID: src.ID, ModelVersionID: enc.XModal}); err == nil {
					if err := add(q.FoundNearestImages(ctx, sqlc.FoundNearestImagesParams{Query: clip, VersionID: enc.Image, WindowStart: start, WindowEnd: end, K: k})); err != nil {
						return nil, err
					}
				} else if !db.IsNoRows(err) {
					return nil, err
				}
			}
		}
		if enc.Text != uuid.Nil {
			if v, err := q.E5Vector(ctx, sqlc.E5VectorParams{ReportType: TypeLost, ReportID: src.ID, ModelVersionID: enc.Text}); err == nil {
				if err := add(q.FoundNearestE5(ctx, sqlc.FoundNearestE5Params{Query: v, VersionID: enc.Text, WindowStart: start, WindowEnd: end, K: k})); err != nil {
					return nil, err
				}
			} else if !db.IsNoRows(err) {
				return nil, err
			}
		}
		if err := add(q.ExistingFoundOfLost(ctx, src.ID)); err != nil {
			return nil, err
		}
	} else {
		before, after := int32(cfg.Time.PoolBeforeH), int32(cfg.Time.PoolAfterD)
		if err := add(q.LostPoolStructured(ctx, sqlc.LostPoolStructuredParams{BeforeH: before, FoundAt: src.FoundAt, AfterD: after, CategoryID: src.CategoryID, K: k})); err != nil {
			return nil, err
		}
		if enc.Image != uuid.Nil {
			vectors, err := q.ImageVectors(ctx, sqlc.ImageVectorsParams{ReportType: TypeFound, ReportID: src.ID, ModelVersionID: enc.Image})
			if err != nil {
				return nil, err
			}
			for _, v := range vectors {
				if err := add(q.LostNearestImages(ctx, sqlc.LostNearestImagesParams{Query: v.Embedding, VersionID: enc.Image, BeforeH: before, FoundAt: src.FoundAt, AfterD: after, K: k})); err != nil {
					return nil, err
				}
				if enc.XModal != uuid.Nil {
					if err := add(q.LostNearestClipText(ctx, sqlc.LostNearestClipTextParams{Query: v.Embedding, VersionID: enc.XModal, BeforeH: before, FoundAt: src.FoundAt, AfterD: after, K: k})); err != nil {
						return nil, err
					}
				}
			}
		}
		if enc.Text != uuid.Nil {
			if v, err := q.E5Vector(ctx, sqlc.E5VectorParams{ReportType: TypeFound, ReportID: src.ID, ModelVersionID: enc.Text}); err == nil {
				if err := add(q.LostNearestE5(ctx, sqlc.LostNearestE5Params{Query: v, VersionID: enc.Text, BeforeH: before, FoundAt: src.FoundAt, AfterD: after, K: k})); err != nil {
					return nil, err
				}
			} else if !db.IsNoRows(err) {
				return nil, err
			}
		}
		if err := add(q.ExistingLostOfFound(ctx, src.ID)); err != nil {
			return nil, err
		}
	}
	out := make([]uuid.UUID, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return out, nil
}

// cosines keyed by the candidate id, computed in PostgreSQL so vectors never travel to Go
type signals struct {
	image    map[uuid.UUID]float64
	cross    map[uuid.UUID]float64
	semantic map[uuid.UUID]float64
	withImgs map[uuid.UUID]bool
}

func (e *Engine) signals(ctx context.Context, q *sqlc.Queries, src *report, cands map[uuid.UUID]*report, enc encoders) (signals, error) {
	s := signals{image: map[uuid.UUID]float64{}, cross: map[uuid.UUID]float64{}, semantic: map[uuid.UUID]float64{}, withImgs: map[uuid.UUID]bool{}}
	other := opposite(src.Type)
	candIDs := make([]uuid.UUID, 0, len(cands))
	for id := range cands {
		candIDs = append(candIDs, id)
	}
	if len(candIDs) == 0 {
		return s, nil
	}
	if enc.Image != uuid.Nil {
		withSrc, err := q.ReportsWithImageEmbeddings(ctx, sqlc.ReportsWithImageEmbeddingsParams{ReportType: src.Type, Ids: []uuid.UUID{src.ID}, VersionID: enc.Image})
		if err != nil {
			return s, err
		}
		withCands, err := q.ReportsWithImageEmbeddings(ctx, sqlc.ReportsWithImageEmbeddingsParams{ReportType: other, Ids: candIDs, VersionID: enc.Image})
		if err != nil {
			return s, err
		}
		for _, id := range withCands {
			s.withImgs[id] = true
		}
		srcHasImages := len(withSrc) > 0
		if srcHasImages {
			rows, err := q.ImageCosines(ctx, sqlc.ImageCosinesParams{CandidateType: other, CandidateIds: candIDs, SourceType: src.Type, SourceID: src.ID, VersionID: enc.Image})
			if err != nil {
				return s, err
			}
			for _, r := range rows {
				s.image[r.CandidateID] = r.Cosine
			}
		}
		if enc.XModal != uuid.Nil {
			// the side without photos lends its CLIP text, the other side its images (section 6.1)
			var textType, imageType string
			var textIDs, imageIDs []uuid.UUID
			if !src.HasPhotos {
				textType, textIDs, imageType = src.Type, []uuid.UUID{src.ID}, other
				for _, id := range candIDs {
					if s.withImgs[id] {
						imageIDs = append(imageIDs, id)
					}
				}
			} else if srcHasImages {
				imageType, imageIDs, textType = src.Type, []uuid.UUID{src.ID}, other
				for _, id := range candIDs {
					if !cands[id].HasPhotos {
						textIDs = append(textIDs, id)
					}
				}
			}
			if len(textIDs) > 0 && len(imageIDs) > 0 {
				rows, err := q.CrossCosines(ctx, sqlc.CrossCosinesParams{
					ImageType: imageType, ImageIds: imageIDs, ImageVersionID: enc.Image, TextType: textType, TextIds: textIDs, TextVersionID: enc.XModal,
				})
				if err != nil {
					return s, err
				}
				for _, r := range rows {
					key := r.ImageReportID
					if textType == other {
						key = r.TextReportID
					}
					s.cross[key] = r.Cosine
				}
			}
		}
	}
	if enc.Text != uuid.Nil {
		rows, err := q.E5Cosines(ctx, sqlc.E5CosinesParams{CandidateType: other, CandidateIds: candIDs, SourceType: src.Type, SourceID: src.ID, VersionID: enc.Text})
		if err != nil {
			return s, err
		}
		for _, r := range rows {
			s.semantic[r.CandidateID] = r.Cosine
		}
	}
	return s, nil
}

type scoredPair struct {
	lost, found *report
	result      fusion.Result
	facts       Facts
}

func (e *Engine) score(cfg fusion.Config, lost, found *report, compat map[uuid.UUID]float64, sig signals, candID uuid.UUID) scoredPair {
	in := fusion.Input{
		FoundAt: found.FoundAt, LostFrom: lost.LostFrom, LostUntil: lost.LostUntil,
		Colour: fusion.Colours(lost.Colours, found.Colours), Brand: fusion.Brands(lost.Brand, found.Brand),
	}
	var c *float64
	if f, ok := compat[lost.CategoryID]; ok && lost.CategoryID != found.CategoryID {
		c = &f
	} else if f, ok := compat[found.CategoryID]; ok && lost.CategoryID != found.CategoryID {
		c = &f
	}
	in.Category = cfg.CategoryFactor(lost.CategoryID == found.CategoryID, lost.ParentID == found.ParentID, c)
	facts := Facts{SameCategory: lost.CategoryID == found.CategoryID, CategoryLost: lost.CategoryName, CategoryFound: found.CategoryName}
	if !lost.Unknown && len(lost.Places) > 0 {
		best, bestTerm := place{}, -1.0
		var bestDist float64
		for _, p := range lost.Places {
			d := geo.HaversineM(p.Lat, p.Lng, found.Lat, found.Lng)
			term := cfg.LocationTerm(d, p.ZoneOnly)
			if term > bestTerm {
				best, bestTerm, bestDist = p, term, d
			}
		}
		in.LocationScore = &bestTerm
		facts.DistanceM, facts.Place = &bestDist, best.Name
	}
	switch {
	case found.FoundAt.After(lost.LostUntil):
		facts.TimeRelation, facts.HoursAfter = "after_window", found.FoundAt.Sub(lost.LostUntil).Hours()
	case found.FoundAt.Before(lost.LostFrom.Add(-time.Duration(cfg.Time.ToleranceBeforeH * float64(time.Hour)))):
		facts.TimeRelation = "before_window"
	default:
		facts.TimeRelation = "in_window"
	}
	if in.Colour == fusion.ColourShared {
		for _, a := range lost.Colours {
			for _, b := range found.Colours {
				if strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) && facts.ColourShared == "" {
					facts.ColourShared = strings.ToLower(strings.TrimSpace(b))
				}
			}
		}
	}
	if in.Brand == fusion.BrandSame {
		facts.BrandSame = strings.TrimSpace(found.Brand)
	}
	if v, ok := sig.image[candID]; ok {
		in.Mode, in.VisualCosine = fusion.ModeImage, &v
	} else if v, ok := sig.cross[candID]; ok {
		in.Mode, in.VisualCosine = fusion.ModeCrossModal, &v
	} else {
		in.Mode = fusion.ModeNone
	}
	if v, ok := sig.semantic[candID]; ok {
		in.SemanticCosine = &v
	}
	// a card that carries the owner's name, section 6.2
	if found.Sensitivity == "identity" && found.VisibleText != "" && lost.OwnerName != "" {
		ratio := fuzzy.TokenSetRatio(fuzzy.Normalize(found.VisibleText), fuzzy.Normalize(lost.OwnerName))
		in.NameHit = ratio >= cfg.IdentityNameHit.MinRatio
	}
	return scoredPair{lost: lost, found: found, result: cfg.Score(in), facts: facts}
}

func round4(v *float64) *float64 {
	if v == nil {
		return nil
	}
	r := math.Round(*v*10000) / 10000
	return &r
}

type change struct {
	matchID    uuid.UUID
	prevStatus string
	prevTier   string
	overall    float64
	lost       *report
	found      *report
}

// ScoreReport is the match.report job of section 10 for one report
func (e *Engine) ScoreReport(ctx context.Context, typ string, id uuid.UUID) error {
	src, err := e.load(ctx, e.q, typ, id)
	if err != nil || src == nil {
		return err
	}
	before, err := e.q.EmbeddingFingerprint(ctx, sqlc.EmbeddingFingerprintParams{ReportType: typ, ReportID: id})
	if err != nil {
		return err
	}
	now := e.clock.Now()
	var changes []change
	err = db.InTx(ctx, e.pool, func(tx pgx.Tx) error {
		q := e.q.WithTx(tx)
		fa, err := loadFusion(ctx, q)
		if err != nil {
			return err
		}
		if !src.Open {
			return e.supersede(ctx, tx, q, fa.Config, src, now)
		}
		enc, err := e.encoders(ctx, q)
		if err != nil {
			return err
		}
		candIDs, err := e.candidates(ctx, tx, q, fa.Config, src, enc)
		if err != nil {
			return err
		}
		var cands map[uuid.UUID]*report
		if src.Type == TypeLost {
			cands, err = e.loadFound(ctx, q, candIDs)
		} else {
			cands, err = e.loadLost(ctx, q, candIDs)
		}
		if err != nil {
			return err
		}
		for id, c := range cands {
			if !c.Open {
				delete(cands, id)
			}
		}
		sig, err := e.signals(ctx, q, src, cands, enc)
		if err != nil {
			return err
		}
		compatRows, err := q.CompatibilityFor(ctx, src.CategoryID)
		if err != nil {
			return err
		}
		compat := map[uuid.UUID]float64{}
		for _, r := range compatRows {
			other := r.CategoryA
			if other == src.CategoryID {
				other = r.CategoryB
			}
			compat[other] = r.Factor
		}
		versions, _ := json.Marshal(map[string]string{ComponentImage: enc.Image.String(), ComponentXModal: enc.XModal.String(), ComponentText: enc.Text.String()})
		lostIDs, foundIDs := map[uuid.UUID]bool{}, map[uuid.UUID]bool{}
		for candID, cand := range cands {
			lost, found := src, cand
			if src.Type == TypeFound {
				lost, found = cand, src
			}
			p := e.score(fa.Config, lost, found, compat, sig, candID)
			ch, stored, err := e.upsert(ctx, q, p, fa.ID, versions, now)
			if err != nil {
				return err
			}
			if stored {
				changes = append(changes, ch)
				lostIDs[lost.ID], foundIDs[found.ID] = true, true
			}
		}
		lostIDs[src.ID] = src.Type == TypeLost
		foundIDs[src.ID] = src.Type == TypeFound
		touchedLost, touchedFound, err := e.settle(ctx, q, fa.Config, keys(lostIDs), keys(foundIDs))
		if err != nil {
			return err
		}
		if err := e.syncStatuses(ctx, q, touchedLost, touchedFound, now); err != nil {
			return err
		}
		return e.announce(ctx, tx, q, changes)
	})
	if err != nil {
		return err
	}
	// an edit or an embedding that committed while this job ran is scored by running once more
	if again, err := e.load(ctx, e.q, typ, id); err == nil && again != nil && !again.UpdatedAt.Equal(src.UpdatedAt) {
		return ErrChangedMeanwhile
	}
	if after, err := e.q.EmbeddingFingerprint(ctx, sqlc.EmbeddingFingerprintParams{ReportType: typ, ReportID: id}); err == nil && after != before {
		return ErrChangedMeanwhile
	}
	return nil
}

func keys(m map[uuid.UUID]bool) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(m))
	for k, v := range m {
		if v {
			out = append(out, k)
		}
	}
	return out
}

// inserts or rescores one pair, rows an officer or a claim settled are never touched (state machine M)
func (e *Engine) upsert(ctx context.Context, q *sqlc.Queries, p scoredPair, fusionID uuid.UUID, versions []byte, now time.Time) (change, bool, error) {
	r := p.result
	var tier *string
	if r.Tier != fusion.NotStored {
		t := r.Tier
		tier = &t
	}
	var flags []string
	if r.NameHit {
		flags = append(flags, "name_hit")
	}
	if flags == nil {
		flags = []string{}
	}
	facts, _ := json.Marshal(p.facts)
	prev, err := q.PairForUpdate(ctx, sqlc.PairForUpdateParams{LostReportID: p.lost.ID, FoundReportID: p.found.ID})
	switch {
	case db.IsNoRows(err):
		if tier == nil {
			return change{}, false, nil
		}
		id := ids.New()
		if err := q.InsertMatch(ctx, sqlc.InsertMatchParams{
			ID: id, LostReportID: p.lost.ID, FoundReportID: p.found.ID, VisualMode: string(r.Mode),
			SVisual: round4(r.Visual), SSemantic: round4(r.Semantic), SAttr: round4(r.Attributes), SText: round4(r.Text),
			SLocation: round4(r.Location), STime: round4(&r.Time), CategoryFactor: r.Category, Overall: r.Overall, Tier: tier,
			Flags: flags, Explanation: facts, FusionVersionID: fusionID, ComponentVersions: versions, FirstScoredAt: now,
		}); err != nil {
			return change{}, false, err
		}
		row, err := q.PairForUpdate(ctx, sqlc.PairForUpdateParams{LostReportID: p.lost.ID, FoundReportID: p.found.ID})
		if err != nil {
			return change{}, false, err
		}
		return change{matchID: row.ID, overall: r.Overall, lost: p.lost, found: p.found}, true, nil
	case err != nil:
		return change{}, false, err
	}
	if prev.Status != "active" && prev.Status != "superseded" {
		return change{}, false, nil
	}
	if err := q.UpdateMatchScores(ctx, sqlc.UpdateMatchScoresParams{
		ID: prev.ID, VisualMode: string(r.Mode), SVisual: round4(r.Visual), SSemantic: round4(r.Semantic), SAttr: round4(r.Attributes),
		SText: round4(r.Text), SLocation: round4(r.Location), STime: round4(&r.Time), CategoryFactor: r.Category, Overall: r.Overall,
		Tier: tier, Flags: flags, Explanation: facts, FusionVersionID: fusionID, ComponentVersions: versions, ScoredAt: now,
	}); err != nil {
		return change{}, false, err
	}
	return change{matchID: prev.ID, prevStatus: prev.Status, prevTier: str(prev.Tier), overall: r.Overall, lost: p.lost, found: p.found}, true, nil
}

// ranks both sides again and turns rows active or superseded by whether either side still keeps them in its top ten,
// returning every report whose candidate list may have changed so its status is checked
func (e *Engine) settle(ctx context.Context, q *sqlc.Queries, cfg fusion.Config, lostIDs, foundIDs []uuid.UUID) ([]uuid.UUID, []uuid.UUID, error) {
	for _, id := range lostIDs {
		if err := q.RerankLost(ctx, sqlc.RerankLostParams{ReportID: id, TopK: int32(cfg.TopK)}); err != nil {
			return nil, nil, err
		}
	}
	for _, id := range foundIDs {
		if err := q.RerankFound(ctx, sqlc.RerankFoundParams{ReportID: id, TopK: int32(cfg.TopK)}); err != nil {
			return nil, nil, err
		}
	}
	if err := q.ClearRanksBelowFloor(ctx, sqlc.ClearRanksBelowFloorParams{LostIds: lostIDs, FoundIds: foundIDs}); err != nil {
		return nil, nil, err
	}
	rows, err := q.SettleStatuses(ctx, sqlc.SettleStatusesParams{LostIds: lostIDs, FoundIds: foundIDs})
	if err != nil {
		return nil, nil, err
	}
	lost, found := map[uuid.UUID]bool{}, map[uuid.UUID]bool{}
	for _, id := range lostIDs {
		lost[id] = true
	}
	for _, id := range foundIDs {
		found[id] = true
	}
	for _, r := range rows {
		lost[r.LostReportID], found[r.FoundReportID] = true, true
	}
	return keys(lost), keys(found), nil
}

// M2 for every pair of a closed or hidden report, then the counterparts are ranked and checked again
func (e *Engine) supersede(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, cfg fusion.Config, src *report, now time.Time) error {
	var lostIDs, foundIDs []uuid.UUID
	if src.Type == TypeLost {
		others, err := q.SupersedeForLost(ctx, src.ID)
		if err != nil {
			return err
		}
		foundIDs = others
	} else {
		others, err := q.SupersedeForFound(ctx, src.ID)
		if err != nil {
			return err
		}
		lostIDs = others
	}
	if len(lostIDs)+len(foundIDs) == 0 {
		return nil
	}
	touchedLost, touchedFound, err := e.settle(ctx, q, cfg, lostIDs, foundIDs)
	if err != nil {
		return err
	}
	return e.syncStatuses(ctx, q, touchedLost, touchedFound, now)
}

// L2, L3, F2 and F3: an open report shows potential_match exactly while it has an active medium or high candidate
func (e *Engine) syncStatuses(ctx context.Context, q *sqlc.Queries, lostIDs, foundIDs []uuid.UUID, now time.Time) error {
	for _, id := range lostIDs {
		status, err := q.LostStatusForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if status != "reported" && status != "potential_match" {
			continue
		}
		n, err := q.VisibleCandidatesOfLost(ctx, id)
		if err != nil {
			return err
		}
		want, transition := "reported", "L3"
		if n > 0 {
			want, transition = "potential_match", "L2"
		}
		if want == status {
			continue
		}
		if err := q.SetLostStatusBySystem(ctx, sqlc.SetLostStatusBySystemParams{ID: id, Status: want}); err != nil {
			return err
		}
		if err := history(ctx, q, "lost_report", id, status, want, transition, now); err != nil {
			return err
		}
	}
	for _, id := range foundIDs {
		status, err := q.FoundStatusForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if status != "reported" && status != "potential_match" {
			continue
		}
		n, err := q.VisibleCandidatesOfFound(ctx, id)
		if err != nil {
			return err
		}
		want, transition := "reported", "F3"
		if n > 0 {
			want, transition = "potential_match", "F2"
		}
		if want == status {
			continue
		}
		if err := q.SetFoundStatusBySystem(ctx, sqlc.SetFoundStatusBySystemParams{ID: id, Status: want}); err != nil {
			return err
		}
		if err := history(ctx, q, "found_report", id, status, want, transition, now); err != nil {
			return err
		}
	}
	return nil
}

func history(ctx context.Context, q *sqlc.Queries, entity string, id uuid.UUID, from, to, transition string, now time.Time) error {
	return q.InsertStatusHistory(ctx, sqlc.InsertStatusHistoryParams{
		EntityType: entity, EntityID: id, FromStatus: &from, ToStatus: to, Transition: transition, ActorKind: "system", CreatedAt: now,
	})
}

// a pair that became an active high candidate in this run tells the finder and the post officers at once, and queues the
// owner's notification, which BR-NTF-01 throttles
func (e *Engine) announce(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, changes []change) error {
	if len(changes) == 0 {
		return nil
	}
	idList := make([]uuid.UUID, 0, len(changes))
	for _, c := range changes {
		idList = append(idList, c.matchID)
	}
	states, err := q.MatchStates(ctx, idList)
	if err != nil {
		return err
	}
	final := map[uuid.UUID]sqlc.MatchStatesRow{}
	for _, s := range states {
		final[s.ID] = s
	}
	queued := map[uuid.UUID]bool{}
	for _, c := range changes {
		s, ok := final[c.matchID]
		if !ok || s.Status != "active" || s.Tier == nil {
			continue
		}
		tier := *s.Tier
		visibleToOwner := (tier == "high" || tier == "medium") && s.RankForLost != nil
		wasSame := c.prevStatus == "active" && c.prevTier == tier
		if wasSame {
			continue
		}
		if visibleToOwner && c.lost.ReporterID != nil && !queued[c.lost.ID] {
			queued[c.lost.ID] = true
			if _, err := e.jobs.InsertTx(ctx, tx, jobargs.MatchNotify{LostReportID: c.lost.ID}, nil); err != nil {
				return err
			}
		}
		if tier != "high" {
			continue
		}
		if c.found.FinderID != nil {
			sent, err := q.NotificationExists(ctx, sqlc.NotificationExistsParams{RecipientID: *c.found.FinderID, Event: string(notify.MatchHintForFinder), EntityID: &c.found.ID})
			if err != nil {
				return err
			}
			if !sent {
				if err := e.notifier.Send(ctx, tx, notify.Message{
					Event: notify.MatchHintForFinder, To: *c.found.FinderID, Link: "/laporan/temuan/" + c.found.Code,
					Data: map[string]string{"found_code": c.found.Code}, EntityType: "found_report", EntityID: &c.found.ID,
				}); err != nil {
					return err
				}
			}
		}
		if len(c.found.Posts) > 0 {
			officers, err := q.ListUserIDsForPosts(ctx, c.found.Posts)
			if err != nil {
				return err
			}
			for _, officer := range officers {
				sent, err := q.NotificationExists(ctx, sqlc.NotificationExistsParams{RecipientID: officer, Event: string(notify.MatchHighForOfficer), EntityID: &c.matchID})
				if err != nil {
					return err
				}
				if sent {
					continue
				}
				if err := e.notifier.Send(ctx, tx, notify.Message{
					Event: notify.MatchHighForOfficer, To: officer, Link: "/petugas/laporan/hilang/" + c.lost.Code,
					Data:       map[string]string{"lost_code": c.lost.Code, "found_code": c.found.Code, "score": fmt.Sprintf("%.0f%%", c.overall*100)},
					EntityType: "match", EntityID: &c.matchID,
				}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
