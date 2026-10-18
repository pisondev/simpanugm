package claims

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/media"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/idem"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/textx"
	"github.com/pisondev/simpanugm/api/internal/reports"
)

const (
	maxEvidence = 3
	maxAnswer   = 500
)

var (
	foundClaimable = []string{"reported", "potential_match", "under_verification", "unclaimed"}
	lostClaimable  = []string{"reported", "potential_match", "under_verification"}
)

type AnswerInput struct {
	Key    string `json:"key"`
	Answer string `json:"answer"`
}

type CreateInput struct {
	FoundID        uuid.UUID          `json:"found_report_id"`
	LostID         *uuid.UUID         `json:"lost_report_id"`
	MatchID        *uuid.UUID         `json:"match_id"`
	NewLost        *reports.LostInput `json:"lost_report"`
	Statement      string             `json:"statement"`
	Answers        []AnswerInput      `json:"answers"`
	Evidence       []uuid.UUID        `json:"evidence_media_ids"`
	PreferredVisit *string            `json:"preferred_visit"`
}

func ref(idOrCode string) (*uuid.UUID, *string, error) {
	if id, err := uuid.Parse(idOrCode); err == nil {
		return &id, nil, nil
	}
	if reports.IsCode(idOrCode) {
		return nil, &idOrCode, nil
	}
	return nil, nil, apperr.NotFound()
}

func unprocessable(code, title, detail string) *apperr.Error {
	return apperr.Unprocessable(code, title, detail)
}

// the questions of a category with the answers a claimant gave, copied with their text so a later edit of the category
// never rewrites what was asked (US-07 criterion 2)
func answersOf(raw []byte, given []AnswerInput, v *apperr.Collector) []byte {
	var questions []masterdata.Question
	_ = json.Unmarshal(raw, &questions)
	byKey := map[string]string{}
	for _, a := range given {
		byKey[a.Key] = strings.TrimSpace(a.Answer)
	}
	out := make([]openapi.ClaimAnswer, 0, len(questions))
	for _, q := range questions {
		answer := byKey[q.Key]
		if q.Required && answer == "" {
			v.Add("answers."+q.Key, "required", "Wajib diisi")
		}
		if len([]rune(answer)) > maxAnswer {
			v.Add("answers."+q.Key, "too_long", "Maksimal 500 karakter")
		}
		out = append(out, openapi.ClaimAnswer{Key: q.Key, Question: q.Question, Answer: answer})
	}
	for key := range byKey {
		if !slices.ContainsFunc(questions, func(q masterdata.Question) bool { return q.Key == key }) {
			v.Add("answers."+key, "unknown", "Pertanyaan tidak dikenal")
		}
	}
	raw, _ = json.Marshal(out)
	return raw
}

// K1. A member claims from a candidate or from the public listing, an officer files one for a walk in visitor
func (s *Service) Create(ctx context.Context, p *access.Principal, key string, in CreateInput) (openapi.Claim, error) {
	staff := p.IsStaff()
	if !staff && p.ClaimRestricted {
		return openapi.Claim{}, apperr.New(403, "claim_restricted", "Klaim dibatasi", "Akunmu tidak dapat mengajukan klaim sampai administrator membukanya.")
	}
	source := "candidate"
	lostID := in.LostID
	switch {
	case staff && lostID == nil:
		return openapi.Claim{}, apperr.Validation(apperr.FieldError{Field: "lost_report_id", Code: "required", Message: "Pilih laporan kehilangan pengunjung"})
	case staff:
		source = "proxy"
	case lostID == nil:
		if in.NewLost == nil {
			return openapi.Claim{}, apperr.Validation(apperr.FieldError{Field: "lost_report", Code: "required", Message: "Isi data kehilanganmu"})
		}
		created, err := s.reports.CreateLost(ctx, p, key+"-lost", *in.NewLost)
		if err != nil {
			return openapi.Claim{}, err
		}
		lostID, source = &created.Id, "listing"
	}
	now := s.clock.Now()
	var id uuid.UUID
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		k := idem.Key{UserID: p.UserID, Key: key, Endpoint: "createClaim"}
		existing, err := idem.Begin(ctx, q, k, in, now)
		if err != nil || existing != nil {
			if existing != nil {
				id = *existing
			}
			return err
		}
		found, err := q.GetFoundReport(ctx, sqlc.GetFoundReportParams{ID: &in.FoundID})
		if db.IsNoRows(err) || (err == nil && found.FoundReport.HiddenAt != nil) {
			return apperr.NotFound()
		}
		if err != nil {
			return err
		}
		f := found.FoundReport
		if !slices.Contains(foundClaimable, f.Status) || (f.CustodyStatus != "with_finder" && f.CustodyStatus != "in_custody") {
			return unprocessable("found_not_claimable", "Barang tidak dapat diklaim", "Barang ini sudah tidak tersedia untuk diklaim.")
		}
		lost, err := q.GetLostReport(ctx, sqlc.GetLostReportParams{ID: lostID})
		if db.IsNoRows(err) {
			return apperr.NotFound()
		}
		if err != nil {
			return err
		}
		l := lost.LostReport
		var claimant *uuid.UUID
		if staff {
			if l.ReporterID != nil || l.HiddenAt != nil {
				return apperr.NotFound()
			}
		} else {
			if l.ReporterID == nil || *l.ReporterID != p.UserID || l.HiddenAt != nil {
				return apperr.NotFound()
			}
			claimant = &p.UserID
			if f.FinderID != nil && *f.FinderID == p.UserID {
				return apperr.New(403, "own_found_report", "Laporan milikmu", "Kamu tidak dapat mengklaim barang yang kamu laporkan sendiri.")
			}
			open, err := q.CountOpenClaimsOf(ctx, claimant)
			if err != nil {
				return err
			}
			if int(open) >= s.settings.Int(ctx, "claim.max_open") {
				return unprocessable("claim_limit_reached", "Batas klaim tercapai", "Kamu sudah memiliki 3 klaim yang masih berjalan. Tarik salah satu atau tunggu hasilnya.")
			}
			rejected, err := q.HasRejectedClaim(ctx, sqlc.HasRejectedClaimParams{ClaimantID: claimant, FoundReportID: in.FoundID})
			if err != nil {
				return err
			}
			if rejected {
				return unprocessable("claim_rejected_before", "Klaim sudah ditolak", "Klaimmu atas barang ini pernah ditolak. Hubungi petugas pos bila ada bukti baru.")
			}
		}
		if !slices.Contains(lostClaimable, l.Status) {
			return unprocessable("lost_not_open", "Laporan kehilangan sudah selesai", "Laporan kehilangan ini sudah tidak terbuka untuk klaim baru.")
		}
		cat, err := q.GetCategory(ctx, f.CategoryID)
		if err != nil {
			return err
		}
		var v apperr.Collector
		statement := strings.TrimSpace(in.Statement)
		v.Check(len([]rune(statement)) >= 30 && len([]rune(statement)) <= 1000, "statement", "length", "Pernyataan 30 sampai 1000 karakter")
		v.Check(len(in.Evidence) <= maxEvidence, "evidence_media_ids", "too_many", "Maksimal 3 gambar bukti")
		answers := answersOf(cat.VerificationQuestions, in.Answers, &v)
		if err := v.Err(); err != nil {
			return err
		}
		code, err := reports.NextCode(ctx, q, "CLM", now)
		if err != nil {
			return err
		}
		id = ids.New()
		matchID := in.MatchID
		if pair, err := q.PairScores(ctx, sqlc.PairScoresParams{LostReportID: l.ID, FoundReportID: f.ID}); err == nil {
			matchID = &pair.ID
		} else if !db.IsNoRows(err) {
			return err
		}
		visit := textx.Opt(in.PreferredVisit)
		if err := q.InsertClaim(ctx, sqlc.InsertClaimParams{
			ID: id, Code: code, ClaimantID: claimant, CreatedBy: p.UserID, LostReportID: l.ID, FoundReportID: f.ID, MatchID: matchID,
			Source: source, Statement: statement, Answers: answers, PreferredVisit: visit, Now: now,
		}); db.IsUniqueViolation(err, "claims_one_open") {
			return unprocessable("claim_exists", "Klaim sudah ada", "Kamu sudah memiliki klaim yang berjalan untuk barang ini.")
		} else if err != nil {
			return err
		}
		if len(in.Evidence) > 0 {
			if _, err := s.media.Attach(ctx, q, p, media.Target{Purpose: "claim_evidence", ClaimID: &id}, in.Evidence); err != nil {
				return err
			}
		}
		if err := s.step(ctx, q, id, "", "submit", "submitted", nil); err != nil {
			return err
		}
		if err := s.resync(ctx, q, l.ID, f.ID); err != nil {
			return err
		}
		view, err := q.ClaimView(ctx, sqlc.ClaimViewParams{ID: &id})
		if err != nil {
			return err
		}
		if err := s.buildChecks(ctx, q, view, len(in.Evidence), now); err != nil {
			return err
		}
		if claimant != nil {
			if err := s.tell(ctx, tx, notify.ClaimSubmittedAck, *claimant, memberLink(code), map[string]string{"claim_code": code}, id); err != nil {
				return err
			}
		}
		officers, err := s.officersOf(ctx, q, f.HoldingPostID, f.IntendedPostID)
		if err != nil {
			return err
		}
		for _, o := range officers {
			if err := s.tell(ctx, tx, notify.ClaimSubmitted, o, officerLink(code), map[string]string{"claim_code": code, "found_code": f.Code}, id); err != nil {
				return err
			}
			// BR-CLM-07: a claim on an item still with its finder escalates the custody reminder to the 48 hour stage
			if f.CustodyStatus == "with_finder" {
				if err := s.tell(ctx, tx, notify.FoundOverdue, o, "/petugas/laporan/temuan/"+f.Code, map[string]string{"found_code": f.Code}, id); err != nil {
					return err
				}
			}
		}
		if err := audit.Record(ctx, q, now, audit.Entry{Action: "claim.create", EntityType: "claim", EntityID: id,
			Changes: map[string]any{"code": code, "source": source, "found": f.Code, "lost": l.Code}}); err != nil {
			return err
		}
		return idem.Complete(ctx, q, k, id)
	})
	if err != nil {
		return openapi.Claim{}, err
	}
	return s.view(ctx, p, id.String())
}

// the claim row as the caller may see it, or not found
func (s *Service) load(ctx context.Context, q *sqlc.Queries, p *access.Principal, idOrCode string) (sqlc.ClaimViewRow, error) {
	id, code, err := ref(idOrCode)
	if err != nil {
		return sqlc.ClaimViewRow{}, err
	}
	v, err := q.ClaimView(ctx, sqlc.ClaimViewParams{ID: id, Code: code})
	if db.IsNoRows(err) {
		return v, apperr.NotFound()
	}
	if err != nil {
		return v, err
	}
	if !p.IsStaff() && (v.ClaimantID == nil || *v.ClaimantID != p.UserID) {
		return v, apperr.NotFound()
	}
	return v, nil
}

// GET /claims/{id}. An officer opening it reads the hidden details of the item, which is logged (FR-AUDIT-03)
func (s *Service) Get(ctx context.Context, p *access.Principal, idOrCode string) (openapi.Claim, error) {
	v, err := s.load(ctx, s.q, p, idOrCode)
	if err != nil {
		return openapi.Claim{}, err
	}
	if p.IsStaff() && (v.HiddenDetails != nil || v.PrivateDetails != nil) {
		if err := audit.Record(ctx, s.q, s.clock.Now(), audit.Entry{Action: "private.read", EntityType: "claim", EntityID: v.ID,
			Changes: map[string]any{"fields": []string{"hidden_details", "private_details"}}}); err != nil {
			return openapi.Claim{}, err
		}
	}
	return s.project(ctx, s.q, p, v)
}

func (s *Service) view(ctx context.Context, p *access.Principal, idOrCode string) (openapi.Claim, error) {
	v, err := s.load(ctx, s.q, p, idOrCode)
	if err != nil {
		return openapi.Claim{}, err
	}
	return s.project(ctx, s.q, p, v)
}

func named(id *uuid.UUID, name *string) *openapi.NamedRef {
	if id == nil || name == nil {
		return nil
	}
	return &openapi.NamedRef{Id: *id, Name: *name}
}

func (s *Service) assignedTo(p *access.Principal, v sqlc.ClaimViewRow) bool {
	if p.IsAdmin() {
		return true
	}
	return v.HoldingPostID != nil && p.AssignedTo(*v.HoldingPostID)
}

// what the caller may do next, so the pages draw buttons from the api instead of repeating the state machine
func (s *Service) actions(p *access.Principal, v sqlc.ClaimViewRow) []string {
	out := []string{}
	open := slices.Contains(Open, v.Status)
	if p.IsStaff() {
		if open {
			out = append(out, "message")
		}
		post := s.assignedTo(p, v)
		switch v.Status {
		case "submitted":
			out = append(out, "pick_up")
		case "under_review":
			out = append(out, "request_info", "reject")
			if post {
				out = append(out, "approve")
			}
		case "info_requested":
			out = append(out, "reject")
		case "approved":
			if post {
				out = append(out, "revoke", "regenerate_code")
				if v.CustodyStatus == "in_custody" {
					out = append(out, "handover")
				}
			}
		case "rejected":
			out = append(out, "reopen")
		}
		return out
	}
	if open {
		out = append(out, "withdraw", "message")
	}
	if v.Status == "info_requested" {
		out = append(out, "respond")
	}
	return out
}

func (s *Service) project(ctx context.Context, q *sqlc.Queries, p *access.Principal, v sqlc.ClaimViewRow) (openapi.Claim, error) {
	staff := p.IsStaff()
	sens := openapi.Sensitivity("normal")
	if v.FoundSensitivity != nil {
		sens = openapi.Sensitivity(*v.FoundSensitivity)
	}
	found := openapi.ClaimFound{
		Id: v.FoundReportID, Code: v.FoundCode, ItemName: v.FoundItemName, Description: v.FoundDescription,
		Colours: v.FoundColours, Brand: v.FoundBrand, FoundAt: v.FoundAt, FoundLocationName: v.FoundLocationName,
		CustodyStatus: openapi.CustodyStatus(v.CustodyStatus), Status: openapi.FoundStatus(v.FoundStatus),
		Category:    openapi.CategoryRef{Id: v.FoundCategoryID, Slug: v.FoundCategorySlug, Name: v.FoundCategoryName, ParentName: v.FoundCategoryParent, Sensitivity: sens},
		HoldingPost: named(v.HoldingPostID, v.HoldingPostName), IntendedPost: named(v.IntendedPostID, v.IntendedPostName),
		Photos: []openapi.CandidatePhoto{},
	}
	if found.Colours == nil {
		found.Colours = []string{}
	}
	photos, err := q.ListReportMedia(ctx, sqlc.ListReportMediaParams{FoundReportID: &v.FoundReportID})
	if err != nil {
		return openapi.Claim{}, err
	}
	variant := openapi.CandidatePhotoVariant("display")
	if !staff && sens == "identity" {
		variant = "thumb_blur"
	}
	for _, m := range photos {
		found.Photos = append(found.Photos, openapi.CandidatePhoto{MediaId: m.ID, Variant: variant})
	}
	lost := openapi.ClaimLost{
		Id: v.LostReportID, Code: v.LostCode, ItemName: v.LostItemName, Colours: v.LostColours, Brand: v.LostBrand,
		LostFrom: v.LostFrom, LostUntil: v.LostUntil,
	}
	if lost.Colours == nil {
		lost.Colours = []string{}
	}
	var answers []openapi.ClaimAnswer
	_ = json.Unmarshal(v.Answers, &answers)
	if answers == nil {
		answers = []openapi.ClaimAnswer{}
	}
	evidence := []openapi.PhotoRef{}
	media, err := s.media.Evidence(ctx, q, v.ID)
	if err != nil {
		return openapi.Claim{}, err
	}
	for _, m := range media {
		evidence = append(evidence, openapi.PhotoRef{MediaId: m.ID, Position: int(m.Position)})
	}
	out := openapi.Claim{
		Id: v.ID, Code: v.Code, Status: openapi.ClaimStatus(v.Status), Source: openapi.ClaimSource(v.Source),
		FoundReport: found, LostReport: lost, Statement: v.Statement, Answers: answers, PreferredVisit: v.PreferredVisit, Evidence: evidence,
		DecisionNote: v.DecisionNote, Reviewer: named(v.ReviewerID, v.ReviewerName), HandoverLocked: v.HandoverLockedAt != nil,
		ApprovedExpiresAt: v.ApprovedExpiresAt, InfoRequestedAt: v.InfoRequestedAt, HandedOverAt: v.HandedOverAt,
		Actions: s.actions(p, v), Version: int(v.Version), CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
	}
	if v.DecisionReason != nil {
		r := openapi.RejectionReason(*v.DecisionReason)
		out.DecisionReason = &r
	}
	if v.ClaimantID != nil && v.ClaimantName != nil {
		person := openapi.ClaimPerson{Id: v.ClaimantID, FullName: *v.ClaimantName}
		if staff {
			person.Email = deref(v.ClaimantEmail)
			person.Phone = v.ClaimantPhone
		}
		out.Claimant = &person
	}
	if v.ProxyName != nil && v.ClaimantID == nil {
		out.VisitorName = v.ProxyName
	}
	// FR-CLAIM-12: only the claimant sees the code, and only while the claim is approved
	if !staff && v.Status == "approved" && len(v.HandoverNonce) > 0 {
		code := handoverCode(s.pepper, v.ID, v.HandoverNonce)
		out.HandoverCode = &code
	}
	if staff {
		out.InternalNote = v.InternalNote
		found.HiddenDetails, found.StorageSlot = v.HiddenDetails, v.StorageSlot
		lost.PrivateDetails = v.PrivateDetails
		out.FoundReport, out.LostReport = found, lost
	}
	return out, nil
}

func deref(s *string) *string {
	if s == nil {
		return nil
	}
	return s
}

type ListFilter struct {
	Statuses []string
	PostID   *uuid.UUID
	FoundID  *uuid.UUID
	Cursor   *struct {
		At time.Time
		ID uuid.UUID
	}
	Limit int
}

func (s *Service) List(ctx context.Context, p *access.Principal, f ListFilter) ([]sqlc.ListClaimsRow, error) {
	params := sqlc.ListClaimsParams{Statuses: nonNil(f.Statuses), RowLimit: int32(f.Limit + 1), PostID: f.PostID, FoundID: f.FoundID}
	if !p.IsStaff() {
		params.ClaimantID = &p.UserID
	}
	if f.Cursor != nil {
		at, id := f.Cursor.At, f.Cursor.ID
		params.CursorCreated, params.CursorID = &at, &id
	}
	return s.q.ListClaims(ctx, params)
}

func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
