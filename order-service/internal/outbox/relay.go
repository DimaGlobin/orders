package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	kafka "github.com/segmentio/kafka-go"
)

const (
	defaultInterval = 1 * time.Second
	defaultBatch    = 100
)

type Relay struct {
	pool     *pgxpool.Pool
	writer   *kafka.Writer
	interval time.Duration
	batch    int
	log      *slog.Logger
}

func NewRelay(pool *pgxpool.Pool, writer *kafka.Writer, log *slog.Logger) *Relay {
	return &Relay{
		pool:     pool,
		writer:   writer,
		interval: defaultInterval,
		batch:    defaultBatch,
		log:      log,
	}
}

// Run polls the outbox table until ctx is cancelled.
func (r *Relay) Run(ctx context.Context) error {
	r.log.Info("outbox relay started", "interval", r.interval, "batch", r.batch)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.log.Info("outbox relay stopped")
			return nil
		case <-ticker.C:
			if err := r.processBatch(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				r.log.Error("outbox relay: process batch", "error", err)
			}
		}
	}
}

func (r *Relay) processBatch(ctx context.Context) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	type record struct {
		id      int64
		topic   string
		key     string
		payload []byte
	}

	rows, err := tx.Query(ctx,
		`SELECT id, topic, key, payload FROM outbox
		 WHERE sent_at IS NULL
		 ORDER BY id
		 LIMIT $1
		 FOR UPDATE SKIP LOCKED`,
		r.batch,
	)
	if err != nil {
		return fmt.Errorf("select outbox: %w", err)
	}

	var records []record
	for rows.Next() {
		var rec record
		if err := rows.Scan(&rec.id, &rec.topic, &rec.key, &rec.payload); err != nil {
			rows.Close()
			return fmt.Errorf("scan outbox row: %w", err)
		}
		records = append(records, rec)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	if len(records) == 0 {
		return tx.Commit(ctx)
	}

	msgs := make([]kafka.Message, len(records))
	for i, rec := range records {
		msgs[i] = kafka.Message{
			Topic: rec.topic,
			Key:   []byte(rec.key),
			Value: rec.payload,
		}
	}
	if err := r.writer.WriteMessages(ctx, msgs...); err != nil {
		return fmt.Errorf("kafka write: %w", err)
	}

	ids := make([]int64, len(records))
	for i, rec := range records {
		ids[i] = rec.id
	}
	if _, err := tx.Exec(ctx,
		`UPDATE outbox SET sent_at = NOW() WHERE id = ANY($1)`,
		ids,
	); err != nil {
		return fmt.Errorf("mark outbox sent: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	r.log.Info("outbox: published events", "count", len(records))
	return nil
}
