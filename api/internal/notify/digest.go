package notify

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
)

type DigestArgs struct{}

func (DigestArgs) Kind() string { return "notify.officer_digest" }

func (DigestArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: emailQueue, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByPeriod: 12 * time.Hour}}
}

// BR-NTF-02: one email per officer with the digest events of the last 24 hours, nothing when there is nothing
type DigestWorker struct {
	river.WorkerDefaults[DigestArgs]
	Q       *sqlc.Queries
	Jobs    *river.Client[pgx.Tx]
	Clock   clock.Clock
	BaseURL string
}

func (w *DigestWorker) Work(ctx context.Context, _ *river.Job[DigestArgs]) error {
	until := w.Clock.Now()
	since := until.Add(-24 * time.Hour)
	events := DigestEvents()
	recipients, err := w.Q.ListDigestRecipients(ctx, sqlc.ListDigestRecipientsParams{Since: since, Until: until, Events: events})
	if err != nil {
		return err
	}
	base := strings.TrimRight(w.BaseURL, "/")
	for _, r := range recipients {
		rows, err := w.Q.ListDigestItems(ctx, sqlc.ListDigestItemsParams{RecipientID: r.RecipientID, Since: since, Until: until, Events: events})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			continue
		}
		d := Data{Name: r.FullName, AppURL: base, Link: base + "/petugas"}
		for _, row := range rows {
			d.Items = append(d.Items, DigestItem{Title: row.Title, Body: row.Body, Link: base + row.Link})
		}
		var text, html bytes.Buffer
		if err := textTemplates.ExecuteTemplate(&text, "digest.txt.tmpl", d); err != nil {
			return fmt.Errorf("notify: digest text: %w", err)
		}
		if err := htmlTemplates.ExecuteTemplate(&html, "digest.html.tmpl", d); err != nil {
			return fmt.Errorf("notify: digest html: %w", err)
		}
		args := EmailArgs{Event: "officer.digest", To: r.Email, Subject: fmt.Sprintf("Ringkasan SimpanUGM, %d hal perlu ditindaklanjuti", len(rows)), Text: text.String(), HTML: html.String()}
		if _, err := w.Jobs.Insert(ctx, args, nil); err != nil {
			return err
		}
	}
	return nil
}

// a fixed wall clock time every day in WIB
type DailyAt struct {
	Hour, Minute int
}

func (d DailyAt) Next(current time.Time) time.Time {
	local := current.In(clock.WIB())
	next := time.Date(local.Year(), local.Month(), local.Day(), d.Hour, d.Minute, 0, 0, clock.WIB())
	if !next.After(local) {
		next = next.AddDate(0, 0, 1)
	}
	return next.UTC()
}

func Periodic() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(DailyAt{Hour: 7}, func() (river.JobArgs, *river.InsertOpts) {
			return DigestArgs{}, nil
		}, nil),
	}
}
