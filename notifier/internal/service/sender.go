package service

import (
	"context"
	"log/slog"

	"github.com/dimaglobin/notifier/internal/model"
)

// LogSender is a stub implementation of Sender that just logs notifications.
// In a real system this would be replaced with SMTP/push/SMS clients.
type LogSender struct {
	log *slog.Logger
}

func NewLogSender(log *slog.Logger) *LogSender {
	return &LogSender{log: log}
}

func (s *LogSender) Send(_ context.Context, n *model.Notification) error {
	s.log.Info("notification sent",
		"type", n.Type,
		"order_id", n.OrderID,
		"user_id", n.UserID,
	)
	return nil
}
