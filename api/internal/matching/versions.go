package matching

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/matching/fusion"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
)

const (
	ComponentImage  = "image_encoder"
	ComponentXModal = "xmodal_text_encoder"
	ComponentText   = "text_encoder"
	ComponentFusion = "fusion"
)

// ml serves a revision other than the active row, vectors of two models must never be compared (model-lifecycle 2)
var ErrVersionMismatch = errors.New("matching: model_version_mismatch")

// installs the embedded fusion document as the active version when no fusion version is active yet
func EnsureDefaultFusion(ctx context.Context, q *sqlc.Queries, now time.Time) error {
	_ = EnsureCandidateFusion(ctx, q, now)
	if _, err := q.ActiveVersion(ctx, ComponentFusion); err == nil {
		return nil
	} else if !db.IsNoRows(err) {
		return err
	}
	cfg, doc := fusion.Default()
	activated := now
	return q.InsertVersion(ctx, sqlc.InsertVersionParams{
		ID: ids.New(), Component: ComponentFusion, Name: cfg.ID, Identity: []byte(`{"kind":"weighted_mean"}`),
		Config: doc, ConfigSha256: fusion.Digest(doc), Status: "active", ActivatedAt: &activated, CreatedAt: now,
	})
}

// EnsureCandidateFusion ensures a candidate fusion version is registered for admin activation and rollback testing
func EnsureCandidateFusion(ctx context.Context, q *sqlc.Queries, now time.Time) error {
	if _, err := q.VersionByName(ctx, "fusion-learned-2026.11.0"); err == nil {
		return nil
	} else if !db.IsNoRows(err) {
		return err
	}
	candidateDoc := []byte(`{
  "id": "fusion-learned-2026.11.0",
  "kind": "weighted_mean",
  "weights": { "visual_image": 0.38, "visual_cross_modal": 0.22, "text": 0.32, "location": 0.18, "time": 0.12 },
  "bands": { "image": [0.60, 0.95], "cross_modal": [0.18, 0.33], "e5": [0.78, 0.93] },
  "text_mix": { "semantic": 0.75, "attributes": 0.25 },
  "colour_partial": 0.5,
  "location": { "scale_m": 400, "zone_cap": 0.6 },
  "time": { "tolerance_before_h": 2, "half_life_h": 72, "pool_before_h": 24, "pool_after_d": 60 },
  "category": { "same_parent": 0.85, "other": 0.35 },
  "identity_name_hit": { "min_ratio": 85, "text_floor": 0.95 },
  "tiers": { "high": 0.72, "medium": 0.55, "floor": 0.40 },
  "top_k": 10,
  "ann_k": 50
}`)
	return q.InsertVersion(ctx, sqlc.InsertVersionParams{
		ID: ids.New(), Component: ComponentFusion, Name: "fusion-learned-2026.11.0", Identity: []byte(`{"kind":"learned_fusion","model":"logistic_regression"}`),
		Config: candidateDoc, ConfigSha256: fusion.Digest(candidateDoc), Status: "candidate", ActivatedAt: nil, CreatedAt: now,
	})
}

type activeFusion struct {
	ID     uuid.UUID
	Config fusion.Config
}

func loadFusion(ctx context.Context, q *sqlc.Queries) (activeFusion, error) {
	row, err := q.ActiveVersion(ctx, ComponentFusion)
	if err != nil {
		return activeFusion{}, fmt.Errorf("matching: no active fusion version: %w", err)
	}
	cfg, err := fusion.Parse(row.Config)
	if err != nil {
		return activeFusion{}, err
	}
	return activeFusion{ID: row.ID, Config: cfg}, nil
}

// the active version id of an encoder, uuid.Nil when none is registered yet
func activeEncoder(ctx context.Context, q *sqlc.Queries, component string) (uuid.UUID, error) {
	row, err := q.ActiveVersion(ctx, component)
	if db.IsNoRows(err) {
		return uuid.Nil, nil
	}
	return row.ID, err
}

// registers what ml reports it serves. The first version of a component becomes active, a later different one stays a
// candidate until an administrator activates it, and the caller is told the vectors cannot be stored
func registerEncoder(ctx context.Context, q *sqlc.Queries, component, model, revision string, now time.Time) (uuid.UUID, error) {
	if model == "" || revision == "" {
		return uuid.Nil, fmt.Errorf("matching: %s answered without a model identity", component)
	}
	sum := sha256.Sum256([]byte(model + "@" + revision))
	name := component + "-" + hex.EncodeToString(sum[:4])
	if row, err := q.VersionByName(ctx, name); err == nil {
		if row.Status != "active" {
			return uuid.Nil, ErrVersionMismatch
		}
		return row.ID, nil
	} else if !db.IsNoRows(err) {
		return uuid.Nil, err
	}
	identity, _ := json.Marshal(map[string]string{"model": model, "revision": revision})
	_, activeErr := q.ActiveVersion(ctx, component)
	status := "candidate"
	var activated *time.Time
	if db.IsNoRows(activeErr) {
		status, activated = "active", &now
	} else if activeErr != nil {
		return uuid.Nil, activeErr
	}
	if err := q.InsertVersion(ctx, sqlc.InsertVersionParams{
		ID: ids.New(), Component: component, Name: name, Identity: identity, Status: status, ActivatedAt: activated, CreatedAt: now,
	}); err != nil {
		return uuid.Nil, err
	}
	row, err := q.VersionByName(ctx, name)
	if err != nil {
		return uuid.Nil, err
	}
	if row.Status != "active" {
		return uuid.Nil, ErrVersionMismatch
	}
	return row.ID, nil
}
