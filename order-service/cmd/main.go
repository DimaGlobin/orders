package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	kafka "github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"

	"github.com/dimaglobin/order-service/internal/config"
	"github.com/dimaglobin/order-service/internal/outbox"
	"github.com/dimaglobin/order-service/internal/repository"
	"github.com/dimaglobin/order-service/internal/service"
	"github.com/dimaglobin/order-service/internal/transport"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath := flag.String("config", "", "path to yaml config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	slog.SetDefault(config.SetupLogger(cfg.Logger))
	log := slog.Default()

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := openPool(rootCtx, cfg.DB.DSN())
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer pool.Close()
	log.Info("connected to postgres", "host", cfg.DB.Host, "db", cfg.DB.Name)

	if err := ensureTopic(rootCtx, cfg.Kafka); err != nil {
		return fmt.Errorf("ensure kafka topic: %w", err)
	}
	log.Info("kafka topic ready", "topic", cfg.Kafka.Topic)

	writer := newKafkaWriter(cfg.Kafka)
	defer writer.Close()
	log.Info("kafka writer ready", "brokers", cfg.Kafka.Brokers, "topic", cfg.Kafka.Topic)

	repo := repository.NewPostgres(pool)
	svc := service.NewService(repo, log)
	orderHandler := transport.NewHandler(svc, log)
	healthHandler := transport.NewHealthHandler(pool)
	metricsHandler := transport.NewMetricsHandler()

	mux := http.NewServeMux()
	orderHandler.RegisterRoutes(mux)
	healthHandler.RegisterRoutes(mux)
	metricsHandler.RegisterRoutes(mux)

	httpHandler := transport.Chain(mux,
		transport.RequestID,
		transport.Recover(log),
		transport.Logging(log),
	)

	addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      httpHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	relay := outbox.NewRelay(pool, writer, cfg.Outbox, log)
	cleaner := outbox.NewCleaner(pool, cfg.Outbox, log)
	gauges := outbox.NewGaugeUpdater(pool, cfg.Outbox, log)

	g, gCtx := errgroup.WithContext(rootCtx)

	g.Go(func() error {
		log.Info("order-service started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	g.Go(func() error { return relay.Run(gCtx) })
	g.Go(func() error { return cleaner.Run(gCtx) })
	g.Go(func() error { return gauges.Run(gCtx) })

	g.Go(func() error {
		<-gCtx.Done()
		log.Info("shutting down")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}
	log.Info("shutdown complete")
	return nil
}

func openPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(dialCtx, dsn)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}
	if err := pool.Ping(dialCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

func newKafkaWriter(cfg config.KafkaConfig) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(cfg.BrokerList()...),
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
		BatchTimeout:           50 * time.Millisecond,
		RequiredAcks:           kafka.RequireAll,
	}
}

// ensureTopic creates the Kafka topic if it doesn't exist yet.
// Treating "already exists" as success makes the call idempotent — safe to run
// on every startup. Creating the topic explicitly (rather than relying on
// auto-creation by the writer) guarantees it exists *before* any consumer
// connects, so consumer groups register at offset 0 instead of at the
// then-current end of a non-existent topic.
func ensureTopic(ctx context.Context, cfg config.KafkaConfig) error {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client := &kafka.Client{
		Addr:    kafka.TCP(cfg.BrokerList()...),
		Timeout: 10 * time.Second,
	}

	resp, err := client.CreateTopics(dialCtx, &kafka.CreateTopicsRequest{
		Topics: []kafka.TopicConfig{{
			Topic:             cfg.Topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		}},
	})
	if err != nil {
		return fmt.Errorf("create topics request: %w", err)
	}

	if topicErr, ok := resp.Errors[cfg.Topic]; ok && topicErr != nil {
		if errors.Is(topicErr, kafka.TopicAlreadyExists) {
			return nil
		}
		return fmt.Errorf("create topic %q: %w", cfg.Topic, topicErr)
	}
	return nil
}
