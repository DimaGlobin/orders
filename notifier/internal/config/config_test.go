package config_test

import (
	"errors"
	"os"
	"testing"

	"github.com/dimaglobin/notifier/internal/apperrors"
	"github.com/dimaglobin/notifier/internal/config"
)

func clearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

var envKeys = []string{
	"LOG_LEVEL", "LOG_FORMAT",
	"KAFKA_BROKERS", "KAFKA_TOPIC", "KAFKA_GROUP_ID",
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t, envKeys...)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Logger.Level != "info" {
		t.Errorf("Logger.Level = %q, want %q", cfg.Logger.Level, "info")
	}
	if cfg.Logger.Format != "json" {
		t.Errorf("Logger.Format = %q, want %q", cfg.Logger.Format, "json")
	}
	if cfg.Kafka.Brokers != "localhost:9092" {
		t.Errorf("Kafka.Brokers = %q, want %q", cfg.Kafka.Brokers, "localhost:9092")
	}
	if cfg.Kafka.GroupID != "notifier" {
		t.Errorf("Kafka.GroupID = %q, want %q", cfg.Kafka.GroupID, "notifier")
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	clearEnv(t, envKeys...)
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("KAFKA_BROKERS", "broker1:9092,broker2:9092")
	t.Setenv("KAFKA_TOPIC", "my-orders")
	t.Setenv("KAFKA_GROUP_ID", "my-group")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Logger.Level != "debug" {
		t.Errorf("Logger.Level = %q, want %q", cfg.Logger.Level, "debug")
	}
	if cfg.Logger.Format != "text" {
		t.Errorf("Logger.Format = %q, want %q", cfg.Logger.Format, "text")
	}
	if cfg.Kafka.Brokers != "broker1:9092,broker2:9092" {
		t.Errorf("Kafka.Brokers = %q, want %q", cfg.Kafka.Brokers, "broker1:9092,broker2:9092")
	}
	if cfg.Kafka.Topic != "my-orders" {
		t.Errorf("Kafka.Topic = %q, want %q", cfg.Kafka.Topic, "my-orders")
	}
	if cfg.Kafka.GroupID != "my-group" {
		t.Errorf("Kafka.GroupID = %q, want %q", cfg.Kafka.GroupID, "my-group")
	}
}

func TestLoad_MissingConfigFile(t *testing.T) {
	_, err := config.Load("/nonexistent/path.yml")
	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		env   map[string]string
		field string
	}{
		{
			name:  "invalid log level",
			env:   map[string]string{"LOG_LEVEL": "verbose"},
			field: "LOG_LEVEL",
		},
		{
			name:  "invalid log format",
			env:   map[string]string{"LOG_FORMAT": "xml"},
			field: "LOG_FORMAT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t, envKeys...)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			_, err := config.Load("")
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !errors.Is(err, apperrors.ErrValidation) {
				t.Errorf("errors.Is(err, ErrValidation) = false, err: %v", err)
			}

			var ve *apperrors.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("errors.As(err, *ValidationError) = false, err: %v", err)
			}
			if ve.Field != tt.field {
				t.Errorf("ValidationError.Field = %q, want %q", ve.Field, tt.field)
			}
		})
	}
}
