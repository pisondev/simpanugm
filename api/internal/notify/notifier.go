package notify

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/settings"
)

type Notifier struct {
	jobs     *river.Client[pgx.Tx]
	baseURL  string
	settings *settings.Store
	clock    clock.Clock
}

func NewNotifier(jobs *river.Client[pgx.Tx], baseURL string, st *settings.Store, c clock.Clock) *Notifier {
	return &Notifier{jobs: jobs, baseURL: strings.TrimRight(baseURL, "/"), settings: st, clock: c}
}

type Message struct {
	Event      Event
	To         uuid.UUID
	Data       map[string]string
	Link       string
	EntityType string
	EntityID   *uuid.UUID
}

// inserts the app row and queues the email in the caller's transaction, so a rolled back action notifies nobody
func (n *Notifier) Send(ctx context.Context, tx pgx.Tx, m Message) error {
	spec, ok := catalog[m.Event]
	if !ok {
		return fmt.Errorf("notify: unknown event %q", m.Event)
	}
	if !strings.HasPrefix(m.Link, "/") {
		return fmt.Errorf("notify: link %q must be a relative path", m.Link)
	}
	q := sqlc.New(tx)
	u, err := q.GetRecipient(ctx, m.To)
	if db.IsNoRows(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if u.Status == "suspended" || u.Status == "deleted" {
		return nil
	}
	app, email := spec.App, spec.Email && !spec.Digest
	if spec.Optional {
		prefs, err := q.ListPreferences(ctx, m.To)
		if err != nil {
			return err
		}
		for _, p := range prefs {
			if p.EventGroup != m.Event.Group() {
				continue
			}
			switch p.Channel {
			case "app":
				app = app && p.Enabled
			case "email":
				email = email && p.Enabled
			}
		}
	}
	if !app && !email {
		return nil
	}
	title, err := render(spec.Title, m.Data)
	if err != nil {
		return err
	}
	body, err := render(spec.Body, m.Data)
	if err != nil {
		return err
	}
	now := n.clock.Now()
	id := ids.New()
	if app {
		status := "none"
		if email {
			status = "pending"
		}
		var entityType *string
		if m.EntityType != "" {
			entityType = &m.EntityType
		}
		if err := q.InsertNotification(ctx, sqlc.InsertNotificationParams{
			ID: id, RecipientID: m.To, Event: string(m.Event), Title: title, Body: body, Link: m.Link,
			EntityType: entityType, EntityID: m.EntityID, EmailStatus: status, CreatedAt: now,
		}); err != nil {
			return fmt.Errorf("notify: insert: %w", err)
		}
		// delivered by PostgreSQL at commit, the hub of every api instance wakes the user's streams
		if err := q.SignalRecipient(ctx, m.To.String()); err != nil {
			return fmt.Errorf("notify: signal: %w", err)
		}
	}
	if email {
		args, err := renderNotificationEmail(u.Email, u.FullName, title, body, n.baseURL+m.Link, n.baseURL)
		if err != nil {
			return err
		}
		args.Event = string(m.Event)
		if app {
			args.NotificationID = &id
		}
		opts := &river.InsertOpts{}
		if spec.Quiet {
			if until, quiet := QuietUntil(now, n.settings.QuietHours(ctx)); quiet {
				opts.ScheduledAt = until
			}
		}
		if _, err := n.jobs.InsertTx(ctx, tx, args, opts); err != nil {
			return fmt.Errorf("notify: enqueue email: %w", err)
		}
	}
	return nil
}

// reports whether now falls inside the quiet window and when it ends, both ends are WIB wall clock
func QuietUntil(now time.Time, q settings.QuietHours) (time.Time, bool) {
	from, okFrom := minutes(q.From)
	to, okTo := minutes(q.To)
	if !okFrom || !okTo || from == to {
		return time.Time{}, false
	}
	local := now.In(clock.WIB())
	m := local.Hour()*60 + local.Minute()
	inside := (from < to && m >= from && m < to) || (from > to && (m >= from || m < to))
	if !inside {
		return time.Time{}, false
	}
	end := time.Date(local.Year(), local.Month(), local.Day(), to/60, to%60, 0, 0, clock.WIB())
	if !end.After(local) {
		end = end.AddDate(0, 0, 1)
	}
	return end.UTC(), true
}

func minutes(hhmm string) (int, bool) {
	h, m, ok := strings.Cut(hhmm, ":")
	if !ok {
		return 0, false
	}
	hi, err1 := strconv.Atoi(h)
	mi, err2 := strconv.Atoi(m)
	if err1 != nil || err2 != nil || hi < 0 || hi > 23 || mi < 0 || mi > 59 {
		return 0, false
	}
	return hi*60 + mi, true
}

func renderNotificationEmail(to, name, title, body, link, baseURL string) (EmailArgs, error) {
	d := Data{Name: name, Title: title, Body: body, Link: link, AppURL: baseURL}
	var text, html bytes.Buffer
	if err := textTemplates.ExecuteTemplate(&text, "notification.txt.tmpl", d); err != nil {
		return EmailArgs{}, fmt.Errorf("notify: text template: %w", err)
	}
	if err := htmlTemplates.ExecuteTemplate(&html, "notification.html.tmpl", d); err != nil {
		return EmailArgs{}, fmt.Errorf("notify: html template: %w", err)
	}
	return EmailArgs{To: to, Subject: title, Text: text.String(), HTML: html.String()}, nil
}
