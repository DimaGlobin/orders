package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dimaglobin/notifier/internal/model"
)

type Service struct {
	sender Sender
	log    *slog.Logger
}

func NewService(sender Sender, log *slog.Logger) *Service {
	return &Service{sender: sender, log: log}
}

func (s *Service) HandleOrderEvent(ctx context.Context, evt model.OrderEvent) error {
	s.log.Info("handling order event",
		"event_type", evt.Type,
		"order_id", evt.OrderID,
		"user_id", evt.UserID,
		"status", evt.Status,
	)

	subject, body, ok := renderForEvent(evt)
	if !ok {
		s.log.Warn("unknown event type, skipping", "type", evt.Type)
		return nil
	}

	notification := &model.Notification{
		OrderID: evt.OrderID,
		UserID:  evt.UserID,
		Type:    model.TypeEmail,
		Status:  model.StatusPending,
		Subject: subject,
		Body:    body,
	}

	return s.sender.Send(ctx, notification)
}

// renderForEvent returns email subject and body for the given event,
// plus a flag indicating whether this event type is known.
func renderForEvent(evt model.OrderEvent) (subject, body string, ok bool) {
	switch evt.Type {
	case model.EventOrderCreated:
		subject = fmt.Sprintf("Order #%d confirmed", evt.OrderID)
		body = fmt.Sprintf(
			"Hello!\n\n"+
				"Your order #%d has been received and is being processed.\n"+
				"We'll let you know as soon as it ships.\n\n"+
				"Best,\nOrders Team",
			evt.OrderID,
		)
		return subject, body, true

	case model.EventOrderCancelled:
		subject = fmt.Sprintf("Order #%d cancelled", evt.OrderID)
		body = fmt.Sprintf(
			"Hello,\n\n"+
				"Your order #%d has been cancelled.\n"+
				"If this was unexpected, please contact support.\n\n"+
				"Best,\nOrders Team",
			evt.OrderID,
		)
		return subject, body, true

	default:
		return "", "", false
	}
}
