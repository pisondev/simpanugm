package notify

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	htmltemplate "html/template"
	texttemplate "text/template"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/pisondev/simpanugm/api/internal/platform/mail"
)

type Event string

const (
	VerifyEmail      Event = "account.verify_email"
	RegisterAttempt  Event = "account.register_attempt"
	PasswordReset    Event = "account.password_reset"
	PasswordChanged  Event = "account.password_changed"
	Invite           Event = "account.invite"
	emailQueue             = "email"
	emailMaxAttempts       = 5
)

var subjects = map[Event]string{ //nolint:gosec // email subjects, not credentials
	VerifyEmail:     "Verifikasi email SimpanUGM",
	RegisterAttempt: "Percobaan pendaftaran dengan emailmu",
	PasswordReset:   "Atur ulang kata sandi SimpanUGM",
	PasswordChanged: "Kata sandi SimpanUGM diubah",
	Invite:          "Undangan akun SimpanUGM",
}

//go:embed templates/*.tmpl
var templateFS embed.FS

var (
	textTemplates = texttemplate.Must(texttemplate.ParseFS(templateFS, "templates/*.txt.tmpl"))
	htmlTemplates = htmltemplate.Must(htmltemplate.ParseFS(templateFS, "templates/*.html.tmpl"))
)

type Data struct {
	Name    string
	Link    string
	AppURL  string
	Expires string
}

type EmailArgs struct {
	Event   string `json:"event"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	HTML    string `json:"html"`
}

func (EmailArgs) Kind() string { return "email.send" }

func (EmailArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: emailQueue, MaxAttempts: emailMaxAttempts}
}

type Mailer struct {
	jobs    *river.Client[pgx.Tx]
	baseURL string
}

func NewMailer(jobs *river.Client[pgx.Tx], baseURL string) *Mailer {
	return &Mailer{jobs: jobs, baseURL: baseURL}
}

func (m *Mailer) BaseURL() string { return m.baseURL }

// rendered now and sent after commit, so a rolled back action emails nobody
func (m *Mailer) Enqueue(ctx context.Context, tx pgx.Tx, event Event, to string, d Data) error {
	args, err := Render(event, to, d, m.baseURL)
	if err != nil {
		return err
	}
	if _, err := m.jobs.InsertTx(ctx, tx, args, nil); err != nil {
		return fmt.Errorf("notify: enqueue %s: %w", event, err)
	}
	return nil
}

func Render(event Event, to string, d Data, baseURL string) (EmailArgs, error) {
	subject, ok := subjects[event]
	if !ok {
		return EmailArgs{}, fmt.Errorf("notify: unknown event %q", event)
	}
	d.AppURL = baseURL
	var text, html bytes.Buffer
	if err := textTemplates.ExecuteTemplate(&text, string(event)+".txt.tmpl", d); err != nil {
		return EmailArgs{}, fmt.Errorf("notify: text template %s: %w", event, err)
	}
	if err := htmlTemplates.ExecuteTemplate(&html, string(event)+".html.tmpl", d); err != nil {
		return EmailArgs{}, fmt.Errorf("notify: html template %s: %w", event, err)
	}
	return EmailArgs{Event: string(event), To: to, Subject: subject, Text: text.String(), HTML: html.String()}, nil
}

type EmailWorker struct {
	river.WorkerDefaults[EmailArgs]
	Sender mail.Sender
}

func (w *EmailWorker) Work(ctx context.Context, job *river.Job[EmailArgs]) error {
	a := job.Args
	return w.Sender.Send(ctx, mail.Message{To: a.To, Subject: a.Subject, Text: a.Text, HTML: a.HTML})
}
