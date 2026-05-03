package transport

import (
	"context"

	"github.com/dimaglobin/notifier/internal/model"
)

type EventHandler interface {
	HandleOrderEvent(ctx context.Context, evt model.OrderEvent) error
}
