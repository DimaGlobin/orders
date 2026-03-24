package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dimaglobin/order-service/internal/config"
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
	// db := postgres.Connect(cfg.DB.DSN())
	// repo := postgres.NewOrderRepository(db)  // writes order + outbox event in one TX

	// svc := orders.NewService(repo, slog.Default())
	// handler := orders.NewHandler(svc, slog.Default())

	// TODO: start outbox relay
	// outboxStore := postgres.NewOutboxStore(db)
	// kafkaPublisher := kafka.NewPublisher(cfg.Kafka)
	// relay := outbox.NewRelay(outboxStore, kafkaPublisher, slog.Default(), 5*time.Second)
	// go relay.Run(ctx)

	// TODO: start HTTP server
	// mux := http.NewServeMux()
	// mux.HandleFunc("POST /orders", handler.Create)
	// mux.HandleFunc("GET /orders/{id}", handler.GetByID)

	slog.Info("starting order-service",
		"http_host", cfg.HTTP.Host,
		"http_port", cfg.HTTP.Port,
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutting down", "signal", sig.String())
}
