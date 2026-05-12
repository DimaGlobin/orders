// Package metrics defines Prometheus collectors used across the service.
// Metrics are registered with promauto on the default registry; exposing
// them is done by mounting promhttp.Handler() at /metrics in main.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	OutboxPending = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbox_pending_total",
		Help: "Number of outbox messages awaiting delivery (sent_at IS NULL AND dead_at IS NULL).",
	})

	OutboxDead = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbox_dead_total",
		Help: "Number of outbox messages that exceeded max attempts and were marked dead.",
	})

	OutboxOldestPendingSeconds = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbox_oldest_pending_seconds",
		Help: "Age in seconds of the oldest pending outbox message; 0 when queue is empty.",
	})

	OutboxPublished = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_published_total",
		Help: "Cumulative number of outbox messages successfully published to Kafka.",
	})

	OutboxPublishErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_publish_errors_total",
		Help: "Cumulative number of batches that failed during Kafka publish.",
	})

	OutboxCleanedUp = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_cleaned_up_total",
		Help: "Cumulative number of outbox rows removed by the retention cleaner.",
	})

	OutboxBatchDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "outbox_batch_duration_seconds",
		Help:    "Time spent processing one outbox batch (select + publish + mark).",
		Buckets: prometheus.DefBuckets,
	})
)
