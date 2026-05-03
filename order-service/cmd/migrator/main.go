package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/dimaglobin/order-service/internal/config"
	"github.com/dimaglobin/order-service/internal/migrator"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: up | down")
	cfgPath := flag.String("config", "", "path to yaml config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.SetDefault(config.SetupLogger(cfg.Logger))

	slog.Info("running migrations", "direction", *direction, "db", cfg.DB.Name)

	switch *direction {
	case "up":
		err = migrator.Up(cfg.DB.DSN())
	case "down":
		err = migrator.Down(cfg.DB.DSN())
	default:
		slog.Error("unknown direction, use: up | down", "direction", *direction)
		os.Exit(1)
	}

	if err != nil {
		slog.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations completed", "direction", *direction)
}
