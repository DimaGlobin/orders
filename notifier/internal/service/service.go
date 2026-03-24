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

func (s *Service) HandleOrderCreated(ctx context.Context, evt model.OrderCreated) error {
	s.log.Info("handling order created event",
		"order_id", evt.OrderID,
		"user_id", evt.UserID,
	)

	notification := &model.Notification{
		OrderID: evt.OrderID,
		UserID:  evt.UserID,
		Type:    model.TypeEmail,
		Status:  model.StatusPending,
	}

	return s.sender.Send(ctx, notification)
}
