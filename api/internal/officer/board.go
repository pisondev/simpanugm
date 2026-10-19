package officer

import (
	"context"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
)

type ReportsGetter interface {
	GetLost(ctx context.Context, p *access.Principal, id string) (openapi.LostReport, error)
	GetFound(ctx context.Context, p *access.Principal, id string) (openapi.FoundReport, error)
}

type MatchesLister interface {
	ListForLost(ctx context.Context, p *access.Principal, ref string, includeLow bool) ([]openapi.MatchCandidate, error)
	ListForFound(ctx context.Context, p *access.Principal, ref string, includeLow bool) ([]openapi.MatchCandidate, error)
}

func (s *Service) WithReportsAndMatching(rep ReportsGetter, mat MatchesLister) *Service {
	s.reports = rep
	s.matching = mat
	return s
}

func (s *Service) Board(ctx context.Context, p *access.Principal, reportType string, reportID string) (*openapi.OfficerBoard, error) {
	if !p.IsStaff() {
		return nil, apperr.Forbidden()
	}
	if s.reports == nil || s.matching == nil {
		return nil, apperr.BadRequest("board services not configured")
	}
	switch reportType {
	case "lost":
		lost, err := s.reports.GetLost(ctx, p, reportID)
		if err != nil {
			return nil, err
		}
		candidates, err := s.matching.ListForLost(ctx, p, reportID, true)
		if err != nil {
			return nil, err
		}
		return &openapi.OfficerBoard{
			ReportType: openapi.OfficerBoardReportTypeLost,
			LostReport: &lost,
			Candidates: candidates,
		}, nil
	case "found":
		found, err := s.reports.GetFound(ctx, p, reportID)
		if err != nil {
			return nil, err
		}
		candidates, err := s.matching.ListForFound(ctx, p, reportID, true)
		if err != nil {
			return nil, err
		}
		return &openapi.OfficerBoard{
			ReportType:  openapi.OfficerBoardReportTypeFound,
			FoundReport: &found,
			Candidates:  candidates,
		}, nil
	default:
		return nil, apperr.BadRequest("Parameter report_type harus bernilai lost atau found.")
	}
}
