package matching

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/matching/jobargs"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/reports"
)

// the read and feedback side of matching, the engine writes the rows
type Service struct {
	e       *Engine
	enqueue *jobargs.Enqueuer
}

func NewService(e *Engine, enq *jobargs.Enqueuer) *Service { return &Service{e: e, enqueue: enq} }

func refOf(idOrCode string) (*uuid.UUID, *string, error) {
	if id, err := uuid.Parse(idOrCode); err == nil {
		return &id, nil, nil
	}
	if reports.IsCode(idOrCode) {
		return nil, &idOrCode, nil
	}
	return nil, nil, apperr.NotFound()
}

func (s *Service) lostRef(ctx context.Context, p *access.Principal, ref string) (sqlc.LostRefForMatchesRow, error) {
	id, code, err := refOf(ref)
	if err != nil {
		return sqlc.LostRefForMatchesRow{}, err
	}
	r, err := s.e.q.LostRefForMatches(ctx, sqlc.LostRefForMatchesParams{ID: id, Code: code})
	if db.IsNoRows(err) {
		return r, apperr.NotFound()
	}
	if err != nil {
		return r, err
	}
	owner := r.ReporterID != nil && *r.ReporterID == p.UserID
	if (r.HiddenAt != nil && !p.IsAdmin()) || (!owner && !p.IsStaff()) {
		return r, apperr.NotFound()
	}
	return r, nil
}

func (s *Service) foundRef(ctx context.Context, p *access.Principal, ref string) (sqlc.FoundRefForMatchesRow, error) {
	id, code, err := refOf(ref)
	if err != nil {
		return sqlc.FoundRefForMatchesRow{}, err
	}
	r, err := s.e.q.FoundRefForMatches(ctx, sqlc.FoundRefForMatchesParams{ID: id, Code: code})
	if db.IsNoRows(err) {
		return r, apperr.NotFound()
	}
	if err != nil {
		return r, err
	}
	finder := r.FinderID != nil && *r.FinderID == p.UserID
	if (r.HiddenAt != nil && !p.IsAdmin()) || (!finder && !p.IsStaff()) {
		return r, apperr.NotFound()
	}
	return r, nil
}

// BR-VIS-01: an owner sees active medium and high candidates that rank in the top ten of the report
func ownerSees(m sqlc.MatchViewsRow) bool {
	return m.Status == "active" && m.Tier != nil && (*m.Tier == "high" || *m.Tier == "medium") && m.RankForLost != nil && m.OwnerDismissedAt == nil
}

func officerSees(m sqlc.MatchViewsRow, includeLow bool) bool {
	if m.Tier == nil {
		return false
	}
	return includeLow || *m.Tier != "low"
}

var officerStatuses = []string{"active", "confirmed"}

func (s *Service) photos(ctx context.Context, rows []sqlc.MatchViewsRow) (map[uuid.UUID][]uuid.UUID, error) {
	idList := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		idList = append(idList, r.FoundReportID)
	}
	out := map[uuid.UUID][]uuid.UUID{}
	if len(idList) == 0 {
		return out, nil
	}
	ph, err := s.e.q.CandidatePhotos(ctx, idList)
	if err != nil {
		return nil, err
	}
	for _, p := range ph {
		if p.FoundReportID != nil {
			out[*p.FoundReportID] = append(out[*p.FoundReportID], p.ID)
		}
	}
	return out, nil
}

func (s *Service) ListForLost(ctx context.Context, p *access.Principal, ref string, includeLow bool) ([]openapi.MatchCandidate, error) {
	lost, err := s.lostRef(ctx, p, ref)
	if err != nil {
		return nil, err
	}
	staff := p.IsStaff()
	rows, err := s.e.q.MatchViews(ctx, sqlc.MatchViewsParams{LostID: &lost.ID, Statuses: officerStatuses})
	if err != nil {
		return nil, err
	}
	kept := rows[:0]
	for _, r := range rows {
		if (staff && officerSees(r, includeLow)) || (!staff && ownerSees(r)) {
			kept = append(kept, r)
		}
	}
	if !staff {
		sortByRank(kept, func(r sqlc.MatchViewsRow) *int16 { return r.RankForLost })
	}
	return s.project(ctx, p, kept, false)
}

func (s *Service) ListForFound(ctx context.Context, p *access.Principal, ref string, includeLow bool) ([]openapi.MatchCandidate, error) {
	if !p.IsStaff() {
		return nil, apperr.Forbidden()
	}
	found, err := s.foundRef(ctx, p, ref)
	if err != nil {
		return nil, err
	}
	rows, err := s.e.q.MatchViews(ctx, sqlc.MatchViewsParams{FoundID: &found.ID, Statuses: officerStatuses})
	if err != nil {
		return nil, err
	}
	kept := rows[:0]
	for _, r := range rows {
		if officerSees(r, includeLow) {
			kept = append(kept, r)
		}
	}
	return s.project(ctx, p, kept, false)
}

func sortByRank(rows []sqlc.MatchViewsRow, rank func(sqlc.MatchViewsRow) *int16) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && *rank(rows[j]) < *rank(rows[j-1]); j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
}

func (s *Service) Get(ctx context.Context, p *access.Principal, id uuid.UUID) (openapi.MatchCandidate, error) {
	rows, err := s.e.q.MatchViews(ctx, sqlc.MatchViewsParams{MatchID: &id, Statuses: []string{"active", "superseded", "dismissed", "confirmed", "rejected"}})
	if err != nil {
		return openapi.MatchCandidate{}, err
	}
	if len(rows) == 0 {
		return openapi.MatchCandidate{}, apperr.NotFound()
	}
	r := rows[0]
	owner := r.ReporterID != nil && *r.ReporterID == p.UserID
	switch {
	case p.IsStaff():
		if r.Tier == nil {
			return openapi.MatchCandidate{}, apperr.NotFound()
		}
	case owner && ownerSees(r):
	default:
		return openapi.MatchCandidate{}, apperr.NotFound()
	}
	out, err := s.project(ctx, p, rows, p.IsStaff())
	if err != nil {
		return openapi.MatchCandidate{}, err
	}
	if p.IsStaff() && r.HiddenDetails != nil {
		if err := audit.Record(ctx, s.e.q, s.e.clock.Now(), audit.Entry{Action: "private.read", EntityType: "found_report", EntityID: r.FoundReportID,
			Changes: map[string]any{"fields": []string{"hidden_details"}, "match_id": r.ID}}); err != nil {
			return openapi.MatchCandidate{}, err
		}
	}
	return out[0], nil
}

func line(l Line) openapi.ExplanationLine {
	out := openapi.ExplanationLine{Reason: l.Reason, Missing: l.Missing}
	if l.Level != nil {
		v := openapi.ExplanationLineLevel(*l.Level)
		out.Level = &v
	}
	if len(l.Chips) > 0 {
		chips := l.Chips
		out.Chips = &chips
	}
	return out
}

func explanationOf(r sqlc.MatchViewsRow) openapi.MatchExplanation {
	var f Facts
	_ = json.Unmarshal(r.Explanation, &f)
	nameHit := false
	for _, fl := range r.Flags {
		nameHit = nameHit || fl == "name_hit"
	}
	e := Explain(r.VisualMode, r.SVisual, r.SText, r.SLocation, nameHit, f)
	return openapi.MatchExplanation{
		VisualMode: openapi.MatchExplanationVisualMode(r.VisualMode),
		Visual:     line(e.Visual), Text: line(e.Text), Location: line(e.Location), Time: line(e.Time), Category: line(e.Category),
	}
}

func sensitivity(s *string) openapi.Sensitivity {
	if s == nil {
		return openapi.Sensitivity("normal")
	}
	return openapi.Sensitivity(*s)
}

// the role projection of section 3.2 of the api page. Members never receive numbers beyond the overall score, hidden
// details, visible text, flags, the finder or the versions (BR-VIS-04)
func (s *Service) project(ctx context.Context, p *access.Principal, rows []sqlc.MatchViewsRow, detail bool) ([]openapi.MatchCandidate, error) {
	photos, err := s.photos(ctx, rows)
	if err != nil {
		return nil, err
	}
	staff := p.IsStaff()
	out := make([]openapi.MatchCandidate, 0, len(rows))
	for _, r := range rows {
		found := openapi.FoundCandidate{
			Id: r.FoundReportID, Code: r.FoundCode, ItemName: r.FoundItemName, Description: r.FoundDescription,
			Colours: r.FoundColours, Brand: r.FoundBrand, FoundAt: r.FoundAt, CustodyStatus: openapi.CustodyStatus(r.CustodyStatus),
			Category: openapi.CategoryRef{
				Id: r.FoundCategoryID, Slug: r.FoundCategorySlug, Name: r.FoundCategoryName, ParentName: r.FoundCategoryParent,
				Sensitivity: sensitivity(r.FoundSensitivity),
			},
			FoundLocation: openapi.LocationRef{Id: r.LocationID, Name: r.LocationName, ZoneId: r.ZoneID, ZoneName: r.ZoneName},
			Photos:        []openapi.CandidatePhoto{},
		}
		if found.Colours == nil {
			found.Colours = []string{}
		}
		if r.HoldingPostID != nil && r.HoldingPostName != nil {
			found.HoldingPost = &openapi.NamedRef{Id: *r.HoldingPostID, Name: *r.HoldingPostName}
		}
		variant := openapi.CandidatePhotoVariant("display")
		if !staff && found.Category.Sensitivity == "identity" {
			variant = "thumb_blur"
		}
		for _, m := range photos[r.FoundReportID] {
			found.Photos = append(found.Photos, openapi.CandidatePhoto{MediaId: m, Variant: variant})
		}
		lost := openapi.LostCandidate{
			Id: r.LostReportID, Code: r.LostCode, ItemName: r.LostItemName, LostFrom: r.LostFrom, LostUntil: r.LostUntil,
			Category: openapi.CategoryRef{
				Id: r.LostCategoryID, Slug: r.LostCategorySlug, Name: r.LostCategoryName, ParentName: r.LostCategoryParent,
				Sensitivity: sensitivity(r.LostSensitivity),
			},
		}
		c := openapi.MatchCandidate{
			Id: r.ID, Tier: openapi.MatchTier(*r.Tier), Overall: float32(r.Overall), Status: openapi.MatchCandidateStatus(r.Status),
			FoundReport: found, LostReport: lost, Explanation: explanationOf(r), ScoredAt: r.ScoredAt,
		}
		if r.RankForLost != nil {
			rank := int(*r.RankForLost)
			c.Rank = &rank
		}
		if staff {
			first := r.FirstScoredAt
			flags := append([]string{}, r.Flags...)
			versions := map[string]string{}
			_ = json.Unmarshal(r.ComponentVersions, &versions)
			owner := r.OwnerName
			c.FirstScoredAt, c.Flags, c.ComponentVersions = &first, &flags, &versions
			c.LostReport.OwnerName = &owner
			c.Scores = &openapi.MatchScores{
				Visual: f32(r.SVisual), Semantic: f32(r.SSemantic), Attributes: f32(r.SAttr), Text: f32(r.SText),
				Location: f32(r.SLocation), Time: f32(r.STime), CategoryFactor: float32(r.CategoryFactor),
			}
			if detail {
				c.FoundReport.HiddenDetails = r.HiddenDetails
			}
		}
		out = append(out, c)
	}
	return out, nil
}

func f32(v *float64) *float32 {
	if v == nil {
		return nil
	}
	f := float32(*v)
	return &f
}

func roleOf(p *access.Principal) string {
	switch {
	case p.IsAdmin():
		return "admin"
	case p.IsStaff():
		return "officer"
	default:
		return "member"
	}
}

func trimReason(r *string) *string {
	if r == nil {
		return nil
	}
	t := strings.TrimSpace(*r)
	if t == "" {
		return nil
	}
	return &t
}

// the owner or an officer may label a candidate they can see, the label feeds the evaluation set (FR-AI-09)
func (s *Service) action(ctx context.Context, tx pgx.Tx, p *access.Principal, id uuid.UUID) (sqlc.MatchForActionRow, error) {
	m, err := s.e.q.WithTx(tx).MatchForAction(ctx, id)
	if db.IsNoRows(err) {
		return m, apperr.NotFound()
	}
	if err != nil {
		return m, err
	}
	owner := m.ReporterID != nil && *m.ReporterID == p.UserID
	visibleToOwner := m.Status == "active" && m.Tier != nil && *m.Tier != "low" && m.RankForLost != nil && m.OwnerDismissedAt == nil
	if !p.IsStaff() && (!owner || !visibleToOwner) {
		return m, apperr.NotFound()
	}
	return m, nil
}

func (s *Service) Feedback(ctx context.Context, p *access.Principal, id uuid.UUID, verdict string, reason *string) error {
	return db.InTx(ctx, s.e.pool, func(tx pgx.Tx) error {
		if _, err := s.action(ctx, tx, p, id); err != nil {
			return err
		}
		actor := p.UserID
		return s.e.q.WithTx(tx).InsertMatchFeedback(ctx, sqlc.InsertMatchFeedbackParams{
			ID: ids.New(), MatchID: id, ActorID: &actor, ActorRole: roleOf(p), Verdict: verdict, Reason: trimReason(reason), CreatedAt: s.e.clock.Now(),
		})
	})
}

// US-21: "Bukan milik saya" hides the candidate for the owner, an officer dismissal (M4) hides it for everyone. Both store
// a negative label, and the lists and statuses of both reports are settled again
func (s *Service) Dismiss(ctx context.Context, p *access.Principal, id uuid.UUID, reason *string) error {
	return db.InTx(ctx, s.e.pool, func(tx pgx.Tx) error {
		m, err := s.action(ctx, tx, p, id)
		if err != nil {
			return err
		}
		q := s.e.q.WithTx(tx)
		now := s.e.clock.Now()
		if p.IsStaff() {
			if m.Status != "active" {
				return apperr.InvalidTransition(m.Status)
			}
			if err := q.OfficerDismissMatch(ctx, id); err != nil {
				return err
			}
			if err := audit.Record(ctx, q, now, audit.Entry{Action: "match.dismiss", EntityType: "match", EntityID: id,
				Changes: map[string]any{"status": audit.Change("active", "dismissed")}}); err != nil {
				return err
			}
		} else if err := q.OwnerDismissMatch(ctx, sqlc.OwnerDismissMatchParams{ID: id, OwnerDismissedAt: &now}); err != nil {
			return err
		}
		actor := p.UserID
		if err := q.InsertMatchFeedback(ctx, sqlc.InsertMatchFeedbackParams{
			ID: ids.New(), MatchID: id, ActorID: &actor, ActorRole: roleOf(p), Verdict: "not_relevant", Reason: trimReason(reason), CreatedAt: now,
		}); err != nil {
			return err
		}
		fa, err := loadFusion(ctx, q)
		if err != nil {
			return err
		}
		lostIDs, foundIDs, err := s.e.settle(ctx, q, fa.Config, []uuid.UUID{m.LostReportID}, []uuid.UUID{m.FoundReportID})
		if err != nil {
			return err
		}
		return s.e.syncStatuses(ctx, q, lostIDs, foundIDs, now)
	})
}

// US-14: whether someone may be looking for the item, with the lost item name and category only
func (s *Service) Hint(ctx context.Context, p *access.Principal, ref string) ([]openapi.MatchHintItem, error) {
	found, err := s.foundRef(ctx, p, ref)
	if err != nil {
		return nil, err
	}
	rows, err := s.e.q.FinderHint(ctx, found.ID)
	if err != nil {
		return nil, err
	}
	out := make([]openapi.MatchHintItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, openapi.MatchHintItem{ItemName: r.ItemName, CategoryName: r.CategoryName})
	}
	return out, nil
}

func (s *Service) Rematch(ctx context.Context, p *access.Principal, typ, ref string) error {
	var id uuid.UUID
	if typ == TypeLost {
		r, err := s.lostRef(ctx, p, ref)
		if err != nil {
			return err
		}
		id = r.ID
	} else {
		r, err := s.foundRef(ctx, p, ref)
		if err != nil {
			return err
		}
		id = r.ID
	}
	return db.InTx(ctx, s.e.pool, func(tx pgx.Tx) error {
		if err := s.enqueue.ReportChanged(ctx, tx, typ, id, nil); err != nil {
			return err
		}
		return audit.Record(ctx, s.e.q.WithTx(tx), s.e.clock.Now(), audit.Entry{Action: "match.rematch", EntityType: typ + "_report", EntityID: id})
	})
}
