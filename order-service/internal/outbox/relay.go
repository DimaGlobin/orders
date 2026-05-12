package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	kafka "github.com/segmentio/kafka-go"

	"github.com/dimaglobin/order-service/internal/config"
	"github.com/dimaglobin/order-service/internal/metrics"
)

// Relay polls the outbox table and forwards pending events to Kafka.
// On publish failure it increments per-row attempts and marks rows dead
// once OutboxConfig.MaxAttempts is reached, so a poison message never
// blocks the queue.
type Relay struct {
	pool   *pgxpool.Pool
	writer *kafka.Writer
	cfg    config.OutboxConfig
	log    *slog.Logger
}

func NewRelay(pool *pgxpool.Pool, writer *kafka.Writer, cfg config.OutboxConfig, log *slog.Logger) *Relay {
	return &Relay{pool: pool, writer: writer, cfg: cfg, log: log}
}

func (r *Relay) Run(ctx context.Context) error {
	r.log.Info("outbox relay started",
		"interval", r.cfg.PollInterval,
		"batch", r.cfg.BatchSize,
		"max_attempts", r.cfg.MaxAttempts,
	)
	ticker := time.NewTicker(r.cfg.PollInterval)
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

type outboxRecord struct {
	id      uuid.UUID
	topic   string
	key     string
	payload []byte
}

func (r *Relay) processBatch(parent context.Context) error {
	start := time.Now()
	defer func() { metrics.OutboxBatchDuration.Observe(time.Since(start).Seconds()) }()

	ctx, cancel := context.WithTimeout(parent, r.cfg.BatchTimeout)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	records, err := r.lockBatch(ctx, tx)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return tx.Commit(ctx)
	}

	msgs := make([]kafka.Message, len(records))
	for i, rec := range records {
		msgs[i] = kafka.Message{Topic: rec.topic, Key: []byte(rec.key), Value: rec.payload}
	}

	ids := make([]uuid.UUID, len(records))
	for i, rec := range records {
		ids[i] = rec.id
	}

	if writeErr := r.writer.WriteMessages(ctx, msgs...); writeErr != nil {
		metrics.OutboxPublishErrors.Inc()
		if err := r.markFailed(ctx, tx, ids, writeErr); err != nil {
			return fmt.Errorf("mark failed: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit failed batch: %w", err)
		}
		r.log.Warn("outbox: batch publish failed",
			"count", len(records),
			"error", writeErr,
		)
		return nil
	}

	if _, err := tx.Exec(ctx,
		`UPDATE outbox SET sent_at = NOW() WHERE id = ANY($1)`,
		ids,
	); err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	metrics.OutboxPublished.Add(float64(len(records)))
	r.log.Info("outbox: published events", "count", len(records))
	return nil
}

func (r *Relay) lockBatch(ctx context.Context, tx pgx.Tx) ([]outboxRecord, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, topic, key, payload FROM outbox
		 WHERE sent_at IS NULL AND dead_at IS NULL
		 ORDER BY id
		 LIMIT $1
		 FOR UPDATE SKIP LOCKED`,
		r.cfg.BatchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("select outbox: %w", err)
	}
	defer rows.Close()

	records := make([]outboxRecord, 0, r.cfg.BatchSize)
	for rows.Next() {
		var rec outboxRecord
		if err := rows.Scan(&rec.id, &rec.topic, &rec.key, &rec.payload); err != nil {
			return nil, fmt.Errorf("scan outbox row: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *Relay) markFailed(ctx context.Context, tx pgx.Tx, ids []uuid.UUID, cause error) error {
	_, err := tx.Exec(ctx,
		`UPDATE outbox
		 SET attempts   = attempts + 1,
		     last_error = $1,
		     dead_at    = CASE WHEN attempts + 1 >= $2 THEN NOW() ELSE NULL END
		 WHERE id = ANY($3)`,
		cause.Error(), r.cfg.MaxAttempts, ids,
	)
	return err
}

