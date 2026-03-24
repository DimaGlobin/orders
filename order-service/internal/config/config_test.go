package config_test

import (
	"errors"
	"os"
	"testing"

	"github.com/dimaglobin/order-service/internal/apperrors"
	"github.com/dimaglobin/order-service/internal/config"
)

func clearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

var envKeys = []string{
	"HTTP_HOST", "HTTP_PORT", "LOG_LEVEL", "LOG_FORMAT",
	"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
	"KAFKA_BROKERS", "KAFKA_TOPIC",
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t, envKeys...)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HTTP.Host != "localhost" {
		t.Errorf("HTTP.Host = %q, want %q", cfg.HTTP.Host, "localhost")
	}
	if cfg.HTTP.Port != 8080 {
		t.Errorf("HTTP.Port = %d, want %d", cfg.HTTP.Port, 8080)
	}
	if cfg.Logger.Level != "info" {
		t.Errorf("Logger.Level = %q, want %q", cfg.Logger.Level, "info")
	}
	if cfg.Logger.Format != "json" {
		t.Errorf("Logger.Format = %q, want %q", cfg.Logger.Format, "json")
	}
	if cfg.DB.Host != "localhost" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "localhost")
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 5432)
	}
	if cfg.Kafka.Brokers != "localhost:9092" {
		t.Errorf("Kafka.Brokers = %q, want %q", cfg.Kafka.Brokers, "localhost:9092")
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	clearEnv(t, envKeys...)
	t.Setenv("HTTP_HOST", "0.0.0.0")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("KAFKA_BROKERS", "broker1:9092,broker2:9092")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HTTP.Host != "0.0.0.0" {
		t.Errorf("HTTP.Host = %q, want %q", cfg.HTTP.Host, "0.0.0.0")
	}
	if cfg.HTTP.Port != 9090 {
		t.Errorf("HTTP.Port = %d, want %d", cfg.HTTP.Port, 9090)
	}
	if cfg.Logger.Level != "debug" {
		t.Errorf("Logger.Level = %q, want %q", cfg.Logger.Level, "debug")
	}
	if cfg.DB.Host != "db.example.com" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "db.example.com")
	}
	if cfg.DB.Port != 5433 {
		t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 5433)
	}
	if cfg.Kafka.Brokers != "broker1:9092,broker2:9092" {
		t.Errorf("Kafka.Brokers = %q, want %q", cfg.Kafka.Brokers, "broker1:9092,broker2:9092")
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
		{
			name:  "invalid HTTP port zero",
			env:   map[string]string{"HTTP_PORT": "0"},
			field: "HTTP_PORT",
		},
		{
			name:  "invalid HTTP port negative",
			env:   map[string]string{"HTTP_PORT": "-1"},
			field: "HTTP_PORT",
		},
		{
			name:  "invalid DB port zero",
			env:   map[string]string{"DB_PORT": "0"},
			field: "DB_PORT",
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
