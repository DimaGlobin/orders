package service

import (
	"context"
	"errors"
	"net/smtp"
	"strings"
	"testing"

	"github.com/dimaglobin/notifier/internal/model"
)

func TestBuildMessage_StructureAndCRLF(t *testing.T) {
	msg := buildMessage("alice@x.test", "bob@y.test", "Hello", "Body line")
	got := string(msg)

	// Required headers.
	wantHeaders := []string{
		"From: alice@x.test",
		"To: bob@y.test",
		"Subject: Hello",
		"Content-Type: text/plain; charset=UTF-8",
	}
	for _, h := range wantHeaders {
		if !strings.Contains(got, h) {
			t.Errorf("missing header %q in:\n%s", h, got)
		}
	}

	// Body present.
	if !strings.Contains(got, "Body line") {
		t.Errorf("missing body content in:\n%s", got)
	}

	// CRLF line endings — required by SMTP. There must be no lone \n.
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Errorf("found a lone LF without CR (SMTP requires CRLF):\n%q", got)
	}

	// Headers and body must be separated by a blank CRLF line.
	if !strings.Contains(got, "\r\n\r\n") {
		t.Errorf("missing blank line between headers and body")
	}
}

func TestBuildMessage_EmptyBodyStillValid(t *testing.T) {
	msg := buildMessage("a@x", "b@y", "subj", "")
	got := string(msg)

	// Must still have header/body separator even with empty body.
	if !strings.HasSuffix(got, "\r\n\r\n") {
		t.Errorf("empty body: must end with header/body separator, got:\n%q", got)
	}
}

// ── Send tests with mocked dialer ────────────────────────────────────────────

// captured records what arguments the dialer was called with.
type captured struct {
	addr    string
	auth    smtp.Auth
	from    string
	to      []string
	message []byte
}

func TestSMTPSender_Send_PassesCorrectArgsToDialer(t *testing.T) {
	var got captured
	mock := func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		got = captured{addr, a, from, to, msg}
		return nil
	}

	s := &SMTPSender{
		addr: "mail.test:1025",
		from: "sender@x.test",
		to:   "user@y.test",
		dial: mock,
		log:  discardLogger(),
	}

	notif := &model.Notification{
		OrderID: 42,
		UserID:  7,
		Subject: "Your order #42",
		Body:    "Thanks!",
	}

	if err := s.Send(context.Background(), notif); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.addr != "mail.test:1025" {
		t.Errorf("addr = %q, want %q", got.addr, "mail.test:1025")
	}
	if got.auth != nil {
		t.Errorf("auth = %v, want nil (MailHog accepts no-auth)", got.auth)
	}
	if got.from != "sender@x.test" {
		t.Errorf("from = %q, want %q", got.from, "sender@x.test")
	}
	if len(got.to) != 1 || got.to[0] != "user@y.test" {
		t.Errorf("to = %v, want [user@y.test]", got.to)
	}

	body := string(got.message)
	if !strings.Contains(body, "Subject: Your order #42") {
		t.Errorf("subject not in message: %s", body)
	}
	if !strings.Contains(body, "Thanks!") {
		t.Errorf("body not in message: %s", body)
	}
}

func TestSMTPSender_Send_PropagatesDialerError(t *testing.T) {
	wantErr := errors.New("connection refused")
	mock := func(string, smtp.Auth, string, []string, []byte) error {
		return wantErr
	}

	s := &SMTPSender{
		addr: "doesnt.matter:25",
		from: "a@a",
		to:   "b@b",
		dial: mock,
		log:  discardLogger(),
	}

	err := s.Send(context.Background(), &model.Notification{Subject: "x", Body: "y"})
	if err == nil {
		t.Fatal("expected error from dialer, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error must wrap dialer err: got %v, want %v", err, wantErr)
	}
}
