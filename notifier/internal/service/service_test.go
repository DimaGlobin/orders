package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/dimaglobin/notifier/internal/model"
)

// ── Test doubles ─────────────────────────────────────────────────────────────

type fakeSender struct {
	sent []model.Notification
	err  error
}

func (f *fakeSender) Send(_ context.Context, n *model.Notification) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, *n) // copy: protects against later mutation
	return nil
}

// discardLogger returns a slog.Logger that writes nowhere. Tests that don't
// care about log output use this to keep test output clean.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestService_HandleOrderEvent_KnownTypes_SendNotification(t *testing.T) {
	tests := []struct {
		name             string
		evt              model.OrderEvent
		wantSubjectPart  string // Subject must contain this
		wantBodyPart     string // Body must contain this
	}{
		{
			name: "order.created → confirmation email",
			evt: model.OrderEvent{
				Type:    model.EventOrderCreated,
				OrderID: "44444444-4444-4444-4444-444444444042",
				UserID:  "77777777-7777-7777-7777-777777777777",
				Status:  "new",
			},
			wantSubjectPart: "confirmed",
			wantBodyPart:    "received and is being processed",
		},
		{
			name: "order.cancelled → cancellation email",
			evt: model.OrderEvent{
				Type:    model.EventOrderCancelled,
				OrderID: "44444444-4444-4444-4444-444444444042",
				UserID:  "77777777-7777-7777-7777-777777777777",
				Status:  "cancelled",
			},
			wantSubjectPart: "cancelled",
			wantBodyPart:    "has been cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &fakeSender{}
			svc := NewService(sender, discardLogger())

			if err := svc.HandleOrderEvent(context.Background(), tt.evt); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(sender.sent) != 1 {
				t.Fatalf("expected 1 notification sent, got %d", len(sender.sent))
			}
			got := sender.sent[0]

			if got.OrderID != tt.evt.OrderID {
				t.Errorf("OrderID = %q, want %q", got.OrderID, tt.evt.OrderID)
			}
			if got.UserID != tt.evt.UserID {
				t.Errorf("UserID = %q, want %q", got.UserID, tt.evt.UserID)
			}
			if got.Type != model.TypeEmail {
				t.Errorf("Type = %q, want %q", got.Type, model.TypeEmail)
			}
			if got.Status != model.StatusPending {
				t.Errorf("Status = %q, want %q", got.Status, model.StatusPending)
			}
			if !strings.Contains(got.Subject, tt.wantSubjectPart) {
				t.Errorf("Subject = %q, must contain %q", got.Subject, tt.wantSubjectPart)
			}
			if !strings.Contains(got.Body, tt.wantBodyPart) {
				t.Errorf("Body = %q, must contain %q", got.Body, tt.wantBodyPart)
			}
		})
	}
}

func TestService_HandleOrderEvent_UnknownType_NothingSent(t *testing.T) {
	sender := &fakeSender{}
	svc := NewService(sender, discardLogger())

	evt := model.OrderEvent{
		Type:    "order.weird", // not a known type
		OrderID: "44444444-4444-4444-4444-444444444042",
	}

	if err := svc.HandleOrderEvent(context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sender.sent) != 0 {
		t.Errorf("expected 0 notifications for unknown type, got %d", len(sender.sent))
	}
}

func TestService_HandleOrderEvent_SenderError_Propagates(t *testing.T) {
	wantErr := errors.New("smtp dead")
	sender := &fakeSender{err: wantErr}
	svc := NewService(sender, discardLogger())

	evt := model.OrderEvent{
		Type:    model.EventOrderCreated,
		OrderID: "44444444-4444-4444-4444-444444444042",
	}

	err := svc.HandleOrderEvent(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error from sender, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error not wrapping sender err: got %v, want %v", err, wantErr)
	}
}

func TestRenderForEvent_OrderIDInSubjectAndBody(t *testing.T) {
	// Sanity check: the order ID must appear in both subject and body for
	// every supported event type, so the user can identify their order.
	knownTypes := []string{
		model.EventOrderCreated,
		model.EventOrderCancelled,
	}

	for _, et := range knownTypes {
		t.Run(et, func(t *testing.T) {
			subject, body, ok := renderForEvent(model.OrderEvent{
				Type:    et,
				OrderID: "abcdabcd-abcd-abcd-abcd-abcdabcd1234",
			})
			if !ok {
				t.Fatal("renderForEvent returned ok=false for known type")
			}
			if !strings.Contains(subject, "abcdabcd-abcd-abcd-abcd-abcdabcd1234") {
				t.Errorf("subject must contain order ID: %q", subject)
			}
			if !strings.Contains(body, "abcdabcd-abcd-abcd-abcd-abcdabcd1234") {
				t.Errorf("body must contain order ID: %q", body)
			}
		})
	}
}
