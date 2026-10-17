package mail

import (
	"context"
	"fmt"
	"sync"

	gomail "github.com/wneessen/go-mail"
)

type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

type Sender interface {
	Send(ctx context.Context, m Message) error
}

type SMTP struct {
	from   string
	client *gomail.Client
}

func NewSMTP(host string, port int, username, password, tlsMode, from string) (*SMTP, error) {
	opts := []gomail.Option{gomail.WithPort(port)}
	switch tlsMode {
	case "tls":
		opts = append(opts, gomail.WithSSL())
	case "starttls":
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	default:
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	}
	if username != "" {
		opts = append(opts, gomail.WithSMTPAuth(gomail.SMTPAuthPlain), gomail.WithUsername(username), gomail.WithPassword(password))
	}
	c, err := gomail.NewClient(host, opts...)
	if err != nil {
		return nil, fmt.Errorf("mail: client: %w", err)
	}
	return &SMTP{from: from, client: c}, nil
}

func (s *SMTP) Send(ctx context.Context, m Message) error {
	msg := gomail.NewMsg()
	if err := msg.From(s.from); err != nil {
		return fmt.Errorf("mail: from: %w", err)
	}
	if err := msg.To(m.To); err != nil {
		return fmt.Errorf("mail: to: %w", err)
	}
	msg.Subject(m.Subject)
	msg.SetBodyString(gomail.TypeTextPlain, m.Text)
	if m.HTML != "" {
		msg.AddAlternativeString(gomail.TypeTextHTML, m.HTML)
	}
	if err := s.client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("mail: send: %w", err)
	}
	return nil
}

type Recorder struct {
	mu   sync.Mutex
	sent []Message
}

func (r *Recorder) Send(_ context.Context, m Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, m)
	return nil
}

func (r *Recorder) Sent() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Message(nil), r.sent...)
}
