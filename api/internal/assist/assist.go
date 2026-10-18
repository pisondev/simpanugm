// Package assist is the report assistant of AI-03 and AI-04: suggestions from a photo and a drafted description
package assist

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/mlclient"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/storage"
	"github.com/pisondev/simpanugm/api/internal/privacy"
)

const (
	maxCategories = 3
	maxColours    = 2
	// a brand is suggested only when the OCR tokens match it closely
	minBrandScore = 0.85
	// FR-ASSIST: a visible text sentence is added only when it is short
	maxDraftText = 40
	// the variant sent to ml, the same one embedding uses
	variant = "display"
)

type Service struct {
	q     *sqlc.Queries
	ml    *mlclient.Client
	store storage.Store
	md    *masterdata.Service
}

func NewService(pool *pgxpool.Pool, ml *mlclient.Client, store storage.Store, md *masterdata.Service) *Service {
	return &Service{q: sqlc.New(pool), ml: ml, store: store, md: md}
}

// FR-ASSIST-04: the form carries on without suggestions, the client shows its fixed note for this code
func unavailable() *apperr.Error {
	return apperr.New(http.StatusServiceUnavailable, "assist_unavailable", "Saran otomatis tidak tersedia", "Saran otomatis sedang tidak tersedia.")
}

// the staged photo of the caller, still unattached or already part of their own report
func (s *Service) photo(ctx context.Context, p *access.Principal, id uuid.UUID, reportType string) ([]byte, error) {
	row, err := s.q.GetMediaForAccess(ctx, sqlc.GetMediaForAccessParams{ID: id, Viewer: p.UserID})
	if db.IsNoRows(err) {
		return nil, apperr.NotFound()
	}
	if err != nil {
		return nil, err
	}
	if row.UploadedBy != p.UserID || row.State == "deleted" || row.Purpose != reportType+"_report" {
		return nil, apperr.NotFound()
	}
	var keys map[string]string
	if err := json.Unmarshal(row.StorageKeys, &keys); err != nil || keys[variant] == "" {
		return nil, apperr.NotFound()
	}
	body, _, err := s.store.Get(ctx, keys[variant])
	if err != nil {
		return nil, apperr.NotFound().WithCause(err)
	}
	defer func() { _ = body.Close() }()
	return io.ReadAll(body)
}

func (s *Service) Analyze(ctx context.Context, p *access.Principal, mediaID uuid.UUID, reportType string) (openapi.AssistSuggestions, error) {
	if s.ml == nil {
		return openapi.AssistSuggestions{}, unavailable()
	}
	img, err := s.photo(ctx, p, mediaID, reportType)
	if err != nil {
		return openapi.AssistSuggestions{}, err
	}
	tree, err := s.md.Categories(ctx)
	if err != nil {
		return openapi.AssistSuggestions{}, err
	}
	known := map[string]masterdata.Category{}
	var set []mlclient.CategoryPrompts
	for _, parent := range tree {
		for _, leaf := range parent.Children {
			known[leaf.Slug] = leaf
			if len(leaf.ClipPrompts) > 0 {
				set = append(set, mlclient.CategoryPrompts{Slug: leaf.Slug, Prompts: leaf.ClipPrompts})
			}
		}
	}
	slices.SortFunc(set, func(a, b mlclient.CategoryPrompts) int { return strings.Compare(a.Slug, b.Slug) })
	parents := map[uuid.UUID]string{}
	for _, parent := range tree {
		parents[parent.ID] = parent.Name
	}
	raw, _ := json.Marshal(set)
	out, err := s.ml.AnalyzeImage(ctx, img, set, hashOf(raw))
	if err != nil {
		// a missing endpoint, an outage and a timeout all read the same to the user
		return openapi.AssistSuggestions{}, unavailable().WithCause(err)
	}
	res := openapi.AssistSuggestions{Categories: []openapi.CategorySuggestion{}, Colours: []string{}}
	for _, c := range out.Categories {
		leaf, ok := known[c.Slug]
		if !ok || len(res.Categories) == maxCategories {
			continue
		}
		sens := openapi.Sensitivity("normal")
		if leaf.Sensitivity != nil {
			sens = openapi.Sensitivity(*leaf.Sensitivity)
		}
		parentName := ""
		if leaf.ParentID != nil {
			parentName = parents[*leaf.ParentID]
		}
		res.Categories = append(res.Categories, openapi.CategorySuggestion{
			Score: float32(c.Score),
			Category: openapi.CategoryRef{
				Id: leaf.ID, Slug: leaf.Slug, Name: leaf.Name, ParentName: parentName, Sensitivity: sens,
			},
		})
	}
	palette, err := s.md.Colours(ctx)
	if err != nil {
		return openapi.AssistSuggestions{}, err
	}
	for _, c := range out.Colours {
		if len(res.Colours) == maxColours {
			break
		}
		for _, pc := range palette {
			if strings.EqualFold(pc.Name, c.Name) && !slices.Contains(res.Colours, pc.Name) {
				res.Colours = append(res.Colours, pc.Name)
			}
		}
	}
	for _, b := range out.Brands {
		if b.Score >= minBrandScore {
			name := b.Name
			res.Brand = &name
			break
		}
	}
	// ml masks already, masking again keeps a card number out when it did not (FR-ASSIST-05)
	if t := strings.TrimSpace(privacy.MaskNumbers(out.Text.Value)); t != "" {
		res.VisibleText = &t
	}
	return res, nil
}

// AI-04: the template of ai-features, the item name in lower case and one sentence for the text on the item
func Draft(item string, colours []string, brand, visibleText string, identity bool) string {
	var b strings.Builder
	item = strings.TrimSpace(item)
	// an acronym such as KTM keeps its capitals, any other name is lower case
	if item != strings.ToUpper(item) {
		item = strings.ToLower(item)
	}
	b.WriteString(item)
	switch len(colours) {
	case 0:
	case 1:
		b.WriteString(" berwarna " + colours[0])
	default:
		b.WriteString(" berwarna " + colours[0] + " dan " + colours[1])
	}
	if brand = strings.TrimSpace(brand); brand != "" {
		b.WriteString(", merek " + brand)
	}
	b.WriteString(".")
	text := strings.TrimSpace(privacy.MaskNumbers(visibleText))
	if text != "" && !identity && !strings.Contains(text, "*") && !strings.Contains(text, `"`) && utf8.RuneCountInString(text) <= maxDraftText {
		b.WriteString(` Terdapat tulisan "` + text + `".`)
	}
	out := b.String()
	r, size := utf8.DecodeRuneInString(out)
	return strings.ToUpper(string(r)) + out[size:]
}

func (s *Service) Describe(ctx context.Context, in openapi.AssistDescribeRequest) (openapi.AssistDescription, error) {
	leaf, ok, err := s.md.Leaf(ctx, s.q, in.CategoryId)
	if err != nil {
		return openapi.AssistDescription{}, err
	}
	if !ok {
		return openapi.AssistDescription{}, apperr.Validation(apperr.FieldError{Field: "category_id", Code: "unknown", Message: "Kategori tidak ditemukan"})
	}
	var colours []string
	if in.Colours != nil {
		colours = *in.Colours
	}
	brand, text := "", ""
	if in.Brand != nil {
		brand = *in.Brand
	}
	if in.VisibleText != nil {
		text = *in.VisibleText
	}
	return openapi.AssistDescription{Description: Draft(leaf.Name, colours, brand, text, leaf.SensitivityValue() == "identity")}, nil
}
