package transport

import (
	"context"

	"github.com/dimaglobin/notifier/internal/model"
)

type EventHandler interface {
	HandleOrderCreated(ctx context.Context, evt model.OrderCreated) error
}
