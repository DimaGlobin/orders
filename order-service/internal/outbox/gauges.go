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

// GaugeUpdater periodically queries the outbox table to refresh the
// pending/dead/oldest-pending Prometheus gauges. Without this background
// task gauges would only update on relay activity.
type GaugeUpdater struct {
	pool *pgxpool.Pool
	cfg  config.OutboxConfig
	log  *slog.Logger
}

func NewGaugeUpdater(pool *pgxpool.Pool, cfg config.OutboxConfig, log *slog.Logger) *GaugeUpdater {
	return &GaugeUpdater{pool: pool, cfg: cfg, log: log}
}

func (g *GaugeUpdater) Run(ctx context.Context) error {
	ticker := time.NewTicker(g.cfg.MetricsInterval)
	defer ticker.Stop()

	// Refresh once on start so /metrics is non-zero immediately.
	_ = g.refresh(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := g.refresh(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				g.log.Error("outbox gauges refresh", "error", err)
			}
		}
	}
}

func (g *GaugeUpdater) refresh(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, g.cfg.BatchTimeout)
	defer cancel()

	var pending, dead int64
	var oldestAgeSeconds float64
	err := g.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE sent_at IS NULL AND dead_at IS NULL),
			COUNT(*) FILTER (WHERE dead_at IS NOT NULL),
			COALESCE(
				EXTRACT(EPOCH FROM (NOW() - MIN(created_at) FILTER (WHERE sent_at IS NULL AND dead_at IS NULL))),
				0
			)
		FROM outbox
	`).Scan(&pending, &dead, &oldestAgeSeconds)
	if err != nil {
		return fmt.Errorf("query gauges: %w", err)
	}

	metrics.OutboxPending.Set(float64(pending))
	metrics.OutboxDead.Set(float64(dead))
	metrics.OutboxOldestPendingSeconds.Set(oldestAgeSeconds)
	return nil
}
