package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"github.com/dimaglobin/notifier/internal/model"
)

// mailDialer abstracts the SMTP delivery call so tests can substitute a fake.
// The signature mirrors net/smtp.SendMail — pluggable without changing call sites.
type mailDialer func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// SMTPSender delivers notifications via plain SMTP (no auth, no TLS).
// Designed for local development with MailHog. For real SMTP servers, swap
// auth=nil for smtp.PlainAuth and use a TLS-enabled dial.
type SMTPSender struct {
	addr string // "host:port"
	from string // From: header
	to   string // recipient (demo: hardcoded — real systems look up by UserID)
	dial mailDialer
	log  *slog.Logger
}

func NewSMTPSender(addr, from, to string, log *slog.Logger) *SMTPSender {
	return &SMTPSender{
		addr: addr,
		from: from,
		to:   to,
		dial: smtp.SendMail, // real network call in production
		log:  log,
	}
}

func (s *SMTPSender) Send(_ context.Context, n *model.Notification) error {
	msg := buildMessage(s.from, s.to, n.Subject, n.Body)

	if err := s.dial(s.addr, nil, s.from, []string{s.to}, msg); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}

	s.log.Info("email sent",
		"to", s.to,
		"order_id", n.OrderID,
		"user_id", n.UserID,
		"subject", n.Subject,
	)
	return nil
}

// buildMessage assembles a minimal RFC 5322 message: headers, blank line, body.
// CRLF line endings are required by the SMTP protocol.
func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\n")
	b.WriteString("To: ")
	b.WriteString(to)
	b.WriteString("\r\n")
	b.WriteString("Subject: ")
	b.WriteString(subject)
	b.WriteString("\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n") // header/body separator
	b.WriteString(body)
	return []byte(b.String())
}
