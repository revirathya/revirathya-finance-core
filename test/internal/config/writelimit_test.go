package config_test

import (
	"testing"
	"time"

	"github.com/revirathya/revirathya-finance-core/internal/config"
)

func TestLoadWriteRateLimit(t *testing.T) {
	t.Run("defaults to 15 per minute", func(t *testing.T) {
		setRequired(t)
		t.Setenv("WRITE_RATE_LIMIT", "")
		t.Setenv("WRITE_RATE_WINDOW", "")

		cfg, err := config.Load()

		if err != nil || cfg.WriteRateLimit != 15 || cfg.WriteRateWindow != time.Minute {
			t.Fatalf("got %d per %v, %v; want 15 per 1m", cfg.WriteRateLimit, cfg.WriteRateWindow, err)
		}
	})

	t.Run("explicit values are used", func(t *testing.T) {
		setRequired(t)
		t.Setenv("WRITE_RATE_LIMIT", "30")
		t.Setenv("WRITE_RATE_WINDOW", "30s")

		cfg, err := config.Load()

		if err != nil || cfg.WriteRateLimit != 30 || cfg.WriteRateWindow != 30*time.Second {
			t.Fatalf("got %d per %v, %v; want 30 per 30s", cfg.WriteRateLimit, cfg.WriteRateWindow, err)
		}
	})

	t.Run("invalid values are startup errors", func(t *testing.T) {
		for _, tt := range []struct{ key, value string }{
			{"WRITE_RATE_LIMIT", "0"},
			{"WRITE_RATE_LIMIT", "-1"},
			{"WRITE_RATE_LIMIT", "many"},
			{"WRITE_RATE_WINDOW", "0s"},
			{"WRITE_RATE_WINDOW", "-1m"},
			{"WRITE_RATE_WINDOW", "60"},
		} {
			setRequired(t)
			t.Setenv("WRITE_RATE_LIMIT", "")
			t.Setenv("WRITE_RATE_WINDOW", "")
			t.Setenv(tt.key, tt.value)

			if _, err := config.Load(); err == nil {
				t.Fatalf("%s=%q: no error, want one", tt.key, tt.value)
			}
		}
	})
}
