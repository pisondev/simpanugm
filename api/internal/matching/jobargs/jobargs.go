// Package jobargs holds the River arguments of matching-engine section 10, so reports can enqueue them without importing the engine
package jobargs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

const (
	QueueML      = "ml"
	QueueDefault = "default"
)

// jobs of one report coalesce while waiting or running, so a burst of edits scores once. River requires the running
// state here, so a worker that sees its report changed while it ran snoozes itself to run once more
var uniqueWhileWaiting = river.UniqueOpts{
	ByArgs: true,
	ByState: []rivertype.JobState{
		rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning, rivertype.JobStateScheduled, rivertype.JobStateRetryable,
	},
}

type EmbedText struct {
	ReportType string    `json:"report_type"`
	ReportID   uuid.UUID `json:"report_id"`
}

func (EmbedText) Kind() string { return "ai.embed_text" }

func (EmbedText) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueML, MaxAttempts: 8, UniqueOpts: uniqueWhileWaiting}
}

type EmbedImage struct {
	MediaID    uuid.UUID `json:"media_id"`
	ReportType string    `json:"report_type"`
	ReportID   uuid.UUID `json:"report_id"`
}

func (EmbedImage) Kind() string { return "ai.embed_image" }

func (EmbedImage) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueML, MaxAttempts: 8, UniqueOpts: uniqueWhileWaiting}
}

type AnalyzeImage struct {
	MediaID    uuid.UUID `json:"media_id"`
	ReportType string    `json:"report_type"`
	ReportID   uuid.UUID `json:"report_id"`
}

func (AnalyzeImage) Kind() string { return "ai.analyze_image" }

func (AnalyzeImage) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueML, MaxAttempts: 8, UniqueOpts: uniqueWhileWaiting}
}

type MatchReport struct {
	ReportType string    `json:"report_type"`
	ReportID   uuid.UUID `json:"report_id"`
}

func (MatchReport) Kind() string { return "match.report" }

func (MatchReport) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueDefault, MaxAttempts: 10, UniqueOpts: uniqueWhileWaiting}
}

type MatchNotify struct {
	LostReportID uuid.UUID `json:"lost_report_id"`
}

func (MatchNotify) Kind() string { return "match.notify" }

func (MatchNotify) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueDefault, MaxAttempts: 5, UniqueOpts: uniqueWhileWaiting}
}

type MatchSweep struct{}

func (MatchSweep) Kind() string { return "match.sweep" }

func (MatchSweep) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueDefault, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByPeriod: 12 * time.Hour}}
}

// what reports call after a report is created, edited, reopened or closed
type Enqueuer struct {
	Jobs *river.Client[pgx.Tx]
	// false when no inference service is configured, matching then runs on the structured signals only
	Inference bool
}

func (e *Enqueuer) ReportChanged(ctx context.Context, tx pgx.Tx, reportType string, reportID uuid.UUID, newMedia []uuid.UUID) error {
	if e == nil || e.Jobs == nil {
		return nil
	}
	var params []river.InsertManyParams
	if e.Inference {
		for _, m := range newMedia {
			params = append(params,
				river.InsertManyParams{Args: EmbedImage{MediaID: m, ReportType: reportType, ReportID: reportID}},
				river.InsertManyParams{Args: AnalyzeImage{MediaID: m, ReportType: reportType, ReportID: reportID}},
			)
		}
		params = append(params, river.InsertManyParams{Args: EmbedText{ReportType: reportType, ReportID: reportID}})
	}
	params = append(params, river.InsertManyParams{Args: MatchReport{ReportType: reportType, ReportID: reportID}})
	if _, err := e.Jobs.InsertManyTx(ctx, tx, params); err != nil {
		return fmt.Errorf("jobargs: enqueue matching for %s %s: %w", reportType, reportID, err)
	}
	return nil
}

// a closed or hidden report only needs its candidates superseded, which match.report does when it finds the report shut
func (e *Enqueuer) ReportClosed(ctx context.Context, tx pgx.Tx, reportType string, reportID uuid.UUID) error {
	if e == nil || e.Jobs == nil {
		return nil
	}
	if _, err := e.Jobs.InsertTx(ctx, tx, MatchReport{ReportType: reportType, ReportID: reportID}, nil); err != nil {
		return fmt.Errorf("jobargs: enqueue close for %s %s: %w", reportType, reportID, err)
	}
	return nil
}
