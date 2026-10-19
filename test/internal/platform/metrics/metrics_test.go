package metrics_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestMetrics(t *testing.T) {
	t.Run("ObserveHTTP counts by handler and code and observes latency", func(t *testing.T) {
		m := metrics.NewMetrics(nil)

		m.ObserveHTTP("graphql", 200, 10*time.Millisecond)
		m.ObserveHTTP("graphql", 200, 20*time.Millisecond)
		m.ObserveHTTP("graphql", 413, time.Millisecond)

		if v, _ := testutil.MetricValue(t, m.Registry, "http_requests_total", map[string]string{"handler": "graphql", "code": "200"}); v != 2 {
			t.Fatalf("200s counted %v, want 2", v)
		}
		if v, _ := testutil.MetricValue(t, m.Registry, "http_requests_total", map[string]string{"handler": "graphql", "code": "413"}); v != 1 {
			t.Fatalf("413s counted %v, want 1", v)
		}
		if v, _ := testutil.MetricValue(t, m.Registry, "http_request_duration_seconds", map[string]string{"handler": "graphql"}); v != 3 {
			t.Fatalf("latency samples %v, want 3", v)
		}
	})

	t.Run("GraphQLError counts by code", func(t *testing.T) {
		m := metrics.NewMetrics(nil)

		m.GraphQLError("BAD_USER_INPUT")
		m.GraphQLError("BAD_USER_INPUT")
		m.GraphQLError("INTERNAL")

		if v, _ := testutil.MetricValue(t, m.Registry, "graphql_errors_total", map[string]string{"code": "BAD_USER_INPUT"}); v != 2 {
			t.Fatalf("BAD_USER_INPUT %v, want 2", v)
		}
		if v, _ := testutil.MetricValue(t, m.Registry, "graphql_errors_total", map[string]string{"code": "INTERNAL"}); v != 1 {
			t.Fatalf("INTERNAL %v, want 1", v)
		}
	})

	t.Run("each instance has its own registry", func(t *testing.T) {
		a, b := metrics.NewMetrics(nil), metrics.NewMetrics(nil)
		a.GraphQLError("INTERNAL")

		if _, ok := testutil.MetricValue(t, b.Registry, "graphql_errors_total", map[string]string{"code": "INTERNAL"}); ok {
			t.Fatal("a metric recorded in one instance shows up in another")
		}
	})

	t.Run("without a pool there are no pool metrics", func(t *testing.T) {
		if _, ok := testutil.MetricValue(t, metrics.NewMetrics(nil).Registry, "db_pool_connections_max", nil); ok {
			t.Fatal("db_pool_connections_max exported without a pool")
		}
	})

	t.Run("Handler serves the registry in Prometheus text format, with Go runtime metrics", func(t *testing.T) {
		m := metrics.NewMetrics(nil)
		m.ObserveHTTP("graphql", 200, time.Millisecond)

		rec := httptest.NewRecorder()
		m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

		body, _ := io.ReadAll(rec.Body)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d, want 200", rec.Code)
		}
		for _, want := range []string{`http_requests_total{code="200",handler="graphql"} 1`, "go_goroutines", "process_"} {
			if !strings.Contains(string(body), want) {
				t.Fatalf("/metrics output is missing %q", want)
			}
		}
	})
}
