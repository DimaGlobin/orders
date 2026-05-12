package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/segmentio/kafka-go"

	"github.com/dimaglobin/notifier/internal/model"
)

// Consumer reads OrderEvent messages from Kafka and dispatches them to the
// handler. Offsets are committed only after successful handling — on error
// the message will be re-delivered on the next fetch (at-least-once).
type Consumer struct {
	reader  *kafka.Reader
	handler EventHandler
	log     *slog.Logger
}

func NewConsumer(reader *kafka.Reader, handler EventHandler, log *slog.Logger) *Consumer {
	return &Consumer{reader: reader, handler: handler, log: log}
}

func (c *Consumer) Run(ctx context.Context) error {
	c.log.Info("consumer started",
		"topic", c.reader.Config().Topic,
		"group_id", c.reader.Config().GroupID,
	)

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			// Normal shutdown paths.
			if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
				c.log.Info("consumer stopped")
				return nil
			}
			c.log.Error("fetch message", "error", err)
			continue
		}

		if err := c.handle(ctx, msg); err != nil {
			// Do not commit — Kafka will re-deliver this message.
			c.log.Error("handle message failed, will retry on next fetch",
				"error", err,
				"offset", msg.Offset,
				"partition", msg.Partition,
			)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.log.Error("commit message", "error", err, "offset", msg.Offset)
		}
	}
}

func (c *Consumer) handle(ctx context.Context, msg kafka.Message) error {
	var evt model.OrderEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		// Poison message — log and skip (do NOT block the partition forever).
		c.log.Error("unmarshal event, skipping poison message",
			"error", err,
			"offset", msg.Offset,
			"value", string(msg.Value),
		)
		return nil
	}

	if err := c.handler.HandleOrderEvent(ctx, evt); err != nil {
		return fmt.Errorf("handle order event: %w", err)
	}
	return nil
}
