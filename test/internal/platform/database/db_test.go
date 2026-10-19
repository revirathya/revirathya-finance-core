package database_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/revirathya/revirathya-finance-core/internal/platform/database"
)

func TestNewDBPool(t *testing.T) {
	t.Run("pool settings come from code, overriding the DSN", func(t *testing.T) {
		// Nothing listens on port 1: the pool is lazy, so creating it needs no database.
		pool, err := database.NewDBPool(context.Background(), "postgres://app:secret@127.0.0.1:1/finance?pool_max_conns=50&pool_min_conns=5")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer pool.Close()

		cfg := pool.Config()
		if cfg.MaxConns != 10 || cfg.MinConns != 1 {
			t.Fatalf("max/min conns %d/%d, want 10/1", cfg.MaxConns, cfg.MinConns)
		}
		if cfg.MaxConnLifetime != time.Hour || cfg.MaxConnIdleTime != 30*time.Minute {
			t.Fatalf("lifetime/idle %v/%v, want 1h/30m", cfg.MaxConnLifetime, cfg.MaxConnIdleTime)
		}
		if cfg.ConnConfig.ConnectTimeout != 5*time.Second {
			t.Fatalf("connect timeout %v, want 5s", cfg.ConnConfig.ConnectTimeout)
		}
		if got := cfg.ConnConfig.RuntimeParams["statement_timeout"]; got != "10s" {
			t.Fatalf("statement_timeout %q, want 10s", got)
		}
		if got := cfg.ConnConfig.RuntimeParams["application_name"]; got != "financecore" {
			t.Fatalf("application_name %q, want financecore", got)
		}
	})

	t.Run("invalid DSN is an error that doesn't leak the password", func(t *testing.T) {
		_, err := database.NewDBPool(context.Background(), "postgres://app:s3cr3t-pass@127.0.0.1:notaport/finance")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if strings.Contains(err.Error(), "s3cr3t-pass") {
			t.Fatalf("error leaks the password: %v", err)
		}
	})
}
