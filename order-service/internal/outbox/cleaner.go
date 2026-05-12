package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dimaglobin/order-service/internal/config"
	"github.com/dimaglobin/order-service/internal/metrics"
)

// Cleaner periodically deletes outbox rows that were published longer
// than OutboxConfig.Retention ago. Dead rows are kept for forensics —
// they need a separate decision and are not auto-pruned.
type Cleaner struct {
	pool *pgxpool.Pool
	cfg  config.OutboxConfig
	log  *slog.Logger
}

func NewCleaner(pool *pgxpool.Pool, cfg config.OutboxConfig, log *slog.Logger) *Cleaner {
	return &Cleaner{pool: pool, cfg: cfg, log: log}
}

func (c *Cleaner) Run(ctx context.Context) error {
	c.log.Info("outbox cleaner started",
		"interval", c.cfg.CleanupInterval,
		"retention", c.cfg.Retention,
	)
	ticker := time.NewTicker(c.cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("outbox cleaner stopped")
			return nil
		case <-ticker.C:
			if err := c.cleanup(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				c.log.Error("outbox cleanup", "error", err)
			}
		}
	}
}

func (c *Cleaner) cleanup(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, c.cfg.BatchTimeout)
	defer cancel()

	cutoff := time.Now().Add(-c.cfg.Retention)
	tag, err := c.pool.Exec(ctx,
		`DELETE FROM outbox WHERE sent_at IS NOT NULL AND sent_at < $1`,
		cutoff,
	)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	deleted := tag.RowsAffected()
	if deleted > 0 {
		metrics.OutboxCleanedUp.Add(float64(deleted))
		c.log.Info("outbox cleanup", "deleted", deleted, "older_than", cutoff.Format(time.RFC3339))
	}
	return nil
}
