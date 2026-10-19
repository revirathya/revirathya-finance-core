package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dlvMetrics "github.com/revirathya/revirathya-finance-core/internal/delivery/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
)

func TestSetup(t *testing.T) {
	m := metrics.NewMetrics(nil)
	m.GraphQLError("INTERNAL")
	h := dlvMetrics.Setup(":0", m).Server.Handler

	t.Run("GET /metrics serves the registry", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `graphql_errors_total{code="INTERNAL"} 1`) {
			t.Fatalf("status %d, body missing the recorded metric", rec.Code)
		}
	})

	t.Run("nothing else is served", func(t *testing.T) {
		for _, path := range []string{"/", "/graphql", "/healthz"} {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("GET %s: status %d, want 404", path, rec.Code)
			}
		}
	})

	t.Run("the server has a read-header timeout", func(t *testing.T) {
		if dlvMetrics.Setup(":0", m).Server.ReadHeaderTimeout <= 0 {
			t.Fatal("ReadHeaderTimeout not set (slowloris)")
		}
	})
}
