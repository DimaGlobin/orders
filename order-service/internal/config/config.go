package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/dimaglobin/order-service/internal/apperrors"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTP   HTTPConfig   `yaml:"http"`
	DB     DBConfig     `yaml:"db"`
	Kafka  KafkaConfig  `yaml:"kafka"`
	Outbox OutboxConfig `yaml:"outbox"`
	Logger LoggerConfig `yaml:"logger"`
}

type OutboxConfig struct {
	PollInterval    time.Duration `yaml:"poll_interval"    env:"OUTBOX_POLL_INTERVAL"    env-default:"1s"`
	BatchSize       int           `yaml:"batch_size"       env:"OUTBOX_BATCH_SIZE"       env-default:"100"`
	BatchTimeout    time.Duration `yaml:"batch_timeout"    env:"OUTBOX_BATCH_TIMEOUT"    env-default:"10s"`
	MaxAttempts     int           `yaml:"max_attempts"     env:"OUTBOX_MAX_ATTEMPTS"     env-default:"5"`
	Retention       time.Duration `yaml:"retention"        env:"OUTBOX_RETENTION"        env-default:"168h"` // 7 days
	CleanupInterval time.Duration `yaml:"cleanup_interval" env:"OUTBOX_CLEANUP_INTERVAL" env-default:"1h"`
	MetricsInterval time.Duration `yaml:"metrics_interval" env:"OUTBOX_METRICS_INTERVAL" env-default:"15s"`
}

type HTTPConfig struct {
	Host string `yaml:"host" env:"HTTP_HOST" env-default:"localhost"`
	Port int    `yaml:"port" env:"HTTP_PORT" env-default:"8080"`
}

type LoggerConfig struct {
	Level  string `yaml:"level"  env:"LOG_LEVEL"  env-default:"info"`
	Format string `yaml:"format" env:"LOG_FORMAT" env-default:"json"`
}

func (c LoggerConfig) LogLevel() slog.Level {
	switch c.Level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type DBConfig struct {
	Host     string `yaml:"host"     env:"DB_HOST"     env-default:"localhost"`
	Port     int    `yaml:"port"     env:"DB_PORT"     env-default:"5432"`
	User     string `yaml:"user"     env:"DB_USER"     env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD"  env-default:"postgres"`
	Name     string `yaml:"name"     env:"DB_NAME"     env-default:"orders"`
	SSLMode  string `yaml:"sslmode"  env:"DB_SSLMODE"  env-default:"disable"`
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode)
}

type KafkaConfig struct {
	Brokers string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092"`
	Topic   string `yaml:"topic"   env:"KAFKA_TOPIC"   env-default:"orders"`
}

func (c KafkaConfig) BrokerList() []string {
	return strings.Split(c.Brokers, ",")
}

var validLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

var validLogFormats = map[string]bool{
	"json": true,
	"text": true,
}

func SetupLogger(cfg LoggerConfig) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel()}
	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

func Load(cfgPath string) (*Config, error) {
	var cfg Config
	if err := load(cfgPath, &cfg); err != nil {
		return nil, err
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func validate(cfg Config) error {
	return errors.Join(
		validatePort("HTTP_PORT", cfg.HTTP.Port),
		validatePort("DB_PORT", cfg.DB.Port),
		validateLogger(cfg.Logger),
		validateKafka(cfg.Kafka),
		validateOutbox(cfg.Outbox),
	)
}

func validateOutbox(cfg OutboxConfig) error {
	if cfg.PollInterval <= 0 {
		return apperrors.NewValidationError("OUTBOX_POLL_INTERVAL", "must be positive")
	}
	if cfg.BatchSize <= 0 {
		return apperrors.NewValidationError("OUTBOX_BATCH_SIZE", "must be positive")
	}
	if cfg.BatchTimeout <= 0 {
		return apperrors.NewValidationError("OUTBOX_BATCH_TIMEOUT", "must be positive")
	}
	if cfg.MaxAttempts <= 0 {
		return apperrors.NewValidationError("OUTBOX_MAX_ATTEMPTS", "must be positive")
	}
	if cfg.Retention <= 0 {
		return apperrors.NewValidationError("OUTBOX_RETENTION", "must be positive")
	}
	if cfg.CleanupInterval <= 0 {
		return apperrors.NewValidationError("OUTBOX_CLEANUP_INTERVAL", "must be positive")
	}
	if cfg.MetricsInterval <= 0 {
		return apperrors.NewValidationError("OUTBOX_METRICS_INTERVAL", "must be positive")
	}
	return nil
}

func load(cfgPath string, cfg any) error {
	if cfgPath != "" {
		if _, err := os.Stat(cfgPath); err != nil {
			return fmt.Errorf("config file %q: %w", cfgPath, err)
		}
		return cleanenv.ReadConfig(cfgPath, cfg)
	}
	return cleanenv.ReadEnv(cfg)
}

func validateLogger(cfg LoggerConfig) error {
	if !validLogLevels[cfg.Level] {
		return apperrors.NewValidationError("LOG_LEVEL",
			fmt.Sprintf("must be one of debug/info/warn/error, got %q", cfg.Level))
	}
	if !validLogFormats[cfg.Format] {
		return apperrors.NewValidationError("LOG_FORMAT",
			fmt.Sprintf("must be json or text, got %q", cfg.Format))
	}
	return nil
}

func validatePort(field string, port int) error {
	if port <= 0 || port > 65535 {
		return apperrors.NewValidationError(field,
			fmt.Sprintf("must be between 1 and 65535, got %d", port))
	}
	return nil
}

func validateKafka(cfg KafkaConfig) error {
	if cfg.Brokers == "" {
		return apperrors.NewValidationError("KAFKA_BROKERS", "must not be empty")
	}
	if cfg.Topic == "" {
		return apperrors.NewValidationError("KAFKA_TOPIC", "must not be empty")
	}
	return nil
}
