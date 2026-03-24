package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

	// TODO: initialize infrastructure
	// sender := email.NewSender(...)
	// svc := notifier.NewService(sender, slog.Default())
	// consumer := notifier.NewConsumer(svc, slog.Default())
	// go consumer.Run(ctx)

	slog.Info("starting notifier",
		"kafka_brokers", cfg.Kafka.Brokers,
		"kafka_topic", cfg.Kafka.Topic,
		"kafka_group_id", cfg.Kafka.GroupID,
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutting down", "signal", sig.String())
}
