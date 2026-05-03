package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/dimaglobin/notifier/internal/config"
)

func main() {
	cfgPath := flag.String("config", "", "path to yaml config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.SetDefault(config.SetupLogger(cfg.Logger))

	// Connect to Kafka
	kafkaCtx, kafkaCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer kafkaCancel()
	kafkaConn, err := kafka.DialContext(kafkaCtx, "tcp", cfg.Kafka.BrokerList()[0])
	if err != nil {
		slog.Error("failed to connect to kafka", "brokers", cfg.Kafka.Brokers, "error", err)
		os.Exit(1)
	}
	defer kafkaConn.Close()
	slog.Info("connected to kafka", "brokers", cfg.Kafka.Brokers)

	// TODO: initialize infrastructure
	// sender := email.NewSender(...)
	// svc := notifier.NewService(sender, slog.Default())
	// consumer := notifier.NewConsumer(svc, slog.Default())
	// go consumer.Run(ctx)

	slog.Info("notifier started",
		"kafka_brokers", cfg.Kafka.Brokers,
		"kafka_topic", cfg.Kafka.Topic,
		"kafka_group_id", cfg.Kafka.GroupID,
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutting down", "signal", sig.String())
}
