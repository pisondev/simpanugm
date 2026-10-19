package assist

import (
	"context"
	"math"
	"math/bits"
	"slices"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/matching/fuzzy"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/geo"
)

func (s *Service) Duplicates(ctx context.Context, in openapi.AssistDuplicatesRequest) (openapi.AssistDuplicatesResponse, error) {
	cat, ok, err := s.md.Leaf(ctx, s.q, in.CategoryId)
	if err != nil {
		return openapi.AssistDuplicatesResponse{Items: []openapi.DuplicateCandidate{}}, err
	}
	parentID := in.CategoryId
	if ok && cat.ParentID != nil {
		parentID = *cat.ParentID
	}

	loc, err := s.q.GetLocation(ctx, in.FoundLocationId)
	if err != nil {
		return openapi.AssistDuplicatesResponse{Items: []openapi.DuplicateCandidate{}}, err
	}

	candidates, err := s.q.ListDuplicateFoundCandidates(ctx, sqlc.ListDuplicateFoundCandidatesParams{
		ParentCategoryID: &parentID,
		FoundAt:          in.FoundAt,
	})
	if err != nil {
		return openapi.AssistDuplicatesResponse{Items: []openapi.DuplicateCandidate{}}, err
	}
	if len(candidates) == 0 {
		return openapi.AssistDuplicatesResponse{Items: []openapi.DuplicateCandidate{}}, nil
	}

	var nearCandidates []sqlc.ListDuplicateFoundCandidatesRow
	var candIDs []uuid.UUID
	for _, cand := range candidates {
		dist := geo.HaversineM(loc.Lat, loc.Lng, cand.Lat, cand.Lng)
		if dist <= 500.0 {
			nearCandidates = append(nearCandidates, cand)
			candIDs = append(candIDs, cand.ID)
		}
	}
	if len(nearCandidates) == 0 {
		return openapi.AssistDuplicatesResponse{Items: []openapi.DuplicateCandidate{}}, nil
	}

	candMedia, _ := s.q.ListFoundMediaPhashes(ctx, candIDs)
	candPhashMap := make(map[uuid.UUID][]int64)
	for _, cm := range candMedia {
		if cm.FoundReportID != nil {
			candPhashMap[*cm.FoundReportID] = append(candPhashMap[*cm.FoundReportID], cm.Phash)
		}
	}

	var stagedPhashes []int64
	if in.MediaIds != nil && len(*in.MediaIds) > 0 {
		staged, _ := s.q.ListStagedMediaPhashes(ctx, *in.MediaIds)
		for _, sm := range staged {
			stagedPhashes = append(stagedPhashes, sm.Phash)
		}
	}

	desc := ""
	if in.Description != nil {
		desc = *in.Description
	}
	draftText := fuzzy.Normalize(in.ItemName + " " + desc)

	type scoredCandidate struct {
		cand  sqlc.ListDuplicateFoundCandidatesRow
		score float64
	}
	var scored []scoredCandidate

	for _, cand := range nearCandidates {
		isPhashDup := false
		candPhashes := candPhashMap[cand.ID]
		for _, p1 := range stagedPhashes {
			for _, p2 := range candPhashes {
				if bits.OnesCount64(uint64(p1^p2)) <= 6 {
					isPhashDup = true
					break
				}
			}
			if isPhashDup {
				break
			}
		}

		var sDup float64
		if isPhashDup {
			sDup = 1.0
		} else {
			candText := fuzzy.Normalize(cand.ItemName + " " + cand.Description)
			sSem := fuzzy.TokenSetRatio(draftText, candText) / 100.0
			if len(stagedPhashes) > 0 && len(candPhashes) > 0 {
				minDist := 64
				for _, p1 := range stagedPhashes {
					for _, p2 := range candPhashes {
						d := bits.OnesCount64(uint64(p1 ^ p2))
						if d < minDist {
							minDist = d
						}
					}
				}
				imgSim := math.Max(0, 1.0-float64(minDist)/32.0)
				sDup = 0.6*imgSim + 0.4*sSem
			} else {
				sDup = sSem
			}
		}

		if sDup >= 0.80 {
			scored = append(scored, scoredCandidate{cand: cand, score: sDup})
		}
	}

	slices.SortFunc(scored, func(a, b scoredCandidate) int {
		if a.score > b.score {
			return -1
		}
		if a.score < b.score {
			return 1
		}
		return 0
	})

	if len(scored) > 3 {
		scored = scored[:3]
	}

	items := make([]openapi.DuplicateCandidate, 0, len(scored))
	for _, s := range scored {
		var thumb *uuid.UUID
		if s.cand.ThumbMediaID != uuid.Nil {
			id := s.cand.ThumbMediaID
			thumb = &id
		}
		items = append(items, openapi.DuplicateCandidate{
			Id:             s.cand.ID,
			Code:           s.cand.Code,
			ItemName:       s.cand.ItemName,
			CategoryName:   s.cand.CategoryName,
			LocationName:   s.cand.LocationName,
			FoundAt:        s.cand.FoundAt,
			DuplicateScore: float32(s.score),
			ThumbMediaId:   thumb,
		})
	}

	return openapi.AssistDuplicatesResponse{Items: items}, nil
}
