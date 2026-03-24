package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dimaglobin/notifier/internal/apperrors"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Kafka  KafkaConfig  `yaml:"kafka"`
	Logger LoggerConfig `yaml:"logger"`
}

type KafkaConfig struct {
	Brokers string `yaml:"brokers"  env:"KAFKA_BROKERS"  env-default:"localhost:9092"`
	Topic   string `yaml:"topic"    env:"KAFKA_TOPIC"    env-default:"orders"`
	GroupID string `yaml:"group_id" env:"KAFKA_GROUP_ID"  env-default:"notifier"`
}

func (c KafkaConfig) BrokerList() []string {
	return strings.Split(c.Brokers, ",")
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
		validateLogger(cfg.Logger),
		validateKafka(cfg.Kafka),
	)
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

func validateKafka(cfg KafkaConfig) error {
	if cfg.Brokers == "" {
		return apperrors.NewValidationError("KAFKA_BROKERS", "must not be empty")
	}
	if cfg.GroupID == "" {
		return apperrors.NewValidationError("KAFKA_GROUP_ID", "must not be empty")
	}
	return nil
}
