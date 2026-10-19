//go:build integration

package metrics_test

import (
	"context"
	"os"
	"testing"

	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithPostgres(m))
}

func TestPoolMetrics(t *testing.T) {
	m := metrics.NewMetrics(testutil.TestPool)
	value := func(name string) float64 {
		t.Helper()
		v, ok := testutil.MetricValue(t, m.Registry, name, nil)
		if !ok {
			t.Fatalf("%s is not exported", name)
		}
		return v
	}

	if got := value("db_pool_connections_max"); got != 10 {
		t.Fatalf("db_pool_connections_max %v, want 10 (the pool's MaxConns)", got)
	}

	before := value("db_pool_acquires_total")
	if _, err := testutil.TestPool.Exec(context.Background(), "SELECT 1"); err != nil {
		t.Fatalf("query: %v", err)
	}
	if after := value("db_pool_acquires_total"); after <= before {
		t.Fatalf("db_pool_acquires_total %v after a query, want more than %v (values are read at scrape time)", after, before)
	}
	if total := value("db_pool_connections_total"); total < 1 {
		t.Fatalf("db_pool_connections_total %v, want at least 1 open connection", total)
	}
	for _, name := range []string{"db_pool_connections_acquired", "db_pool_connections_idle", "db_pool_acquire_wait_total", "db_pool_acquire_wait_seconds_total"} {
		value(name)
	}
}
