package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"

	"github.com/dimaglobin/notifier/internal/config"
	"github.com/dimaglobin/notifier/internal/service"
	"github.com/dimaglobin/notifier/internal/transport"
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

	reader := newKafkaReader(cfg.Kafka)
	defer reader.Close()
	log.Info("kafka reader ready",
		"brokers", cfg.Kafka.Brokers,
		"topic", cfg.Kafka.Topic,
		"group_id", cfg.Kafka.GroupID,
	)

	sender := service.NewSMTPSender(cfg.SMTP.Addr(), cfg.SMTP.From, cfg.SMTP.To, log)
	log.Info("smtp sender ready",
		"addr", cfg.SMTP.Addr(),
		"from", cfg.SMTP.From,
		"to", cfg.SMTP.To,
	)

	svc := service.NewService(sender, log)
	consumer := transport.NewConsumer(reader, svc, log)

	g, gCtx := errgroup.WithContext(rootCtx)

	g.Go(func() error {
		return consumer.Run(gCtx)
	})

	if err := g.Wait(); err != nil {
		return err
	}
	log.Info("shutdown complete")
	return nil
}

func newKafkaReader(cfg config.KafkaConfig) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.BrokerList(),
		Topic:   cfg.Topic,
		GroupID: cfg.GroupID,

		// Manual commits — see Consumer.Run.
		CommitInterval: 0,

		// Reasonable defaults for an event-driven service.
		MinBytes:    1,
		MaxBytes:    10e6, // 10 MB
		MaxWait:     500 * time.Millisecond,
		StartOffset: kafka.FirstOffset, // first-time start: read from beginning
	})
}
