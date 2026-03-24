package transport

import (
	"context"
	"log/slog"
)

type Consumer struct {
	handler EventHandler
	log     *slog.Logger
}

func NewConsumer(handler EventHandler, log *slog.Logger) *Consumer {
	return &Consumer{handler: handler, log: log}
}

func (c *Consumer) Run(ctx context.Context) error {
	c.log.Info("consumer started")
	<-ctx.Done()
	return nil
}
