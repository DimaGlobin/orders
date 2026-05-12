package service

import (
	"context"
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

	switch evt.Type {
	case model.EventOrderCreated:
		s.log.Info("order created — will notify user",
			"order_id", evt.OrderID,
			"user_id", evt.UserID,
		)
	case model.EventOrderCancelled:
		s.log.Info("order cancelled — will notify user",
			"order_id", evt.OrderID,
			"user_id", evt.UserID,
		)
	default:
		s.log.Warn("unknown event type, skipping", "type", evt.Type)
		return nil
	}

	notification := &model.Notification{
		OrderID: evt.OrderID,
		UserID:  evt.UserID,
		Type:    model.TypeEmail,
		Status:  model.StatusPending,
	}

	return s.sender.Send(ctx, notification)
}
