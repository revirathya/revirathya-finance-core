package config_test

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/revirathya/revirathya-finance-core/internal/config"
)

// setRequired sets every required variable to a valid value for the rest of the test.
// t.Setenv restores the previous values afterwards.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URI", "postgres://test:test@localhost:5432/test?sslmode=disable")
	t.Setenv("GRPC_API_KEY", "test-api-key")
	t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
}

func TestLoadLogging(t *testing.T) {
	t.Run("defaults to json at info", func(t *testing.T) {
		setRequired(t)
		t.Setenv("LOG_FORMAT", "")
		t.Setenv("LOG_LEVEL", "")

		cfg, err := config.Load()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.LogFormat != "json" {
			t.Fatalf("LogFormat %q, want %q", cfg.LogFormat, "json")
		}
		if cfg.LogLevel != slog.LevelInfo {
			t.Fatalf("LogLevel %v, want %v", cfg.LogLevel, slog.LevelInfo)
		}
	})

	t.Run("accepts text format and every level, in any case", func(t *testing.T) {
		levels := map[string]slog.Level{
			"debug": slog.LevelDebug,
			"INFO":  slog.LevelInfo,
			"Warn":  slog.LevelWarn,
			"error": slog.LevelError,
		}
		for input, want := range levels {
			setRequired(t)
			t.Setenv("LOG_FORMAT", "text")
			t.Setenv("LOG_LEVEL", input)

			cfg, err := config.Load()

			if err != nil {
				t.Fatalf("LOG_LEVEL=%q: unexpected error: %v", input, err)
			}
			if cfg.LogFormat != "text" || cfg.LogLevel != want {
				t.Fatalf("LOG_LEVEL=%q: got format %q level %v, want text %v", input, cfg.LogFormat, cfg.LogLevel, want)
			}
		}
	})

	t.Run("invalid format is rejected", func(t *testing.T) {
		setRequired(t)
		t.Setenv("LOG_FORMAT", "xml")

		_, err := config.Load()

		if err == nil || !strings.Contains(err.Error(), "LOG_FORMAT") {
			t.Fatalf("got %v, want an error about LOG_FORMAT", err)
		}
	})

	t.Run("invalid level is rejected", func(t *testing.T) {
		setRequired(t)
		t.Setenv("LOG_LEVEL", "loud")

		_, err := config.Load()

		if err == nil || !strings.Contains(err.Error(), "LOG_LEVEL") {
			t.Fatalf("got %v, want an error about LOG_LEVEL", err)
		}
	})
}

func TestLoadRequired(t *testing.T) {
	for _, name := range []string{"DATABASE_URI", "GRPC_API_KEY", "GOOGLE_CLIENT_ID", "JWT_SECRET"} {
		t.Run(name+" is required", func(t *testing.T) {
			setRequired(t)
			t.Setenv(name, "")

			_, err := config.Load()

			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("got %v, want an error naming %s", err, name)
			}
		})
	}

	t.Run("short JWT_SECRET is rejected", func(t *testing.T) {
		setRequired(t)
		t.Setenv("JWT_SECRET", "too-short")

		_, err := config.Load()

		if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
			t.Fatalf("got %v, want an error about JWT_SECRET", err)
		}
	})

	t.Run("error messages never contain secret values", func(t *testing.T) {
		setRequired(t)
		t.Setenv("JWT_SECRET", "short-secret-value")

		_, err := config.Load()

		if err == nil || strings.Contains(err.Error(), "short-secret-value") {
			t.Fatalf("got %v, want an error that does not echo the secret", err)
		}
	})
}
