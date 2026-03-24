package service

import (
	"context"

	"github.com/dimaglobin/notifier/internal/model"
)

type Sender interface {
	Send(ctx context.Context, notification *model.Notification) error
}
