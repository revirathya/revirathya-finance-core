package metricscollector_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware/metricscollector"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestMiddleware(t *testing.T) {
	for name, tt := range map[string]struct {
		handler http.HandlerFunc
		code    string
	}{
		"explicit status":           {func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusRequestEntityTooLarge) }, "413"},
		"write without WriteHeader": {func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) }, "200"},
		"nothing written":           {func(w http.ResponseWriter, r *http.Request) {}, "200"},
	} {
		t.Run(name+" is counted as "+tt.code, func(t *testing.T) {
			m := metrics.NewMetrics(nil)
			rec := httptest.NewRecorder()

			metricscollector.Middleware(m, "graphql")(tt.handler).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/graphql", nil))

			if v, _ := testutil.MetricValue(t, m.Registry, "http_requests_total", map[string]string{"handler": "graphql", "code": tt.code}); v != 1 {
				t.Fatalf("http_requests_total{code=%q} %v, want 1", tt.code, v)
			}
			if v, _ := testutil.MetricValue(t, m.Registry, "http_request_duration_seconds", map[string]string{"handler": "graphql"}); v != 1 {
				t.Fatalf("latency samples %v, want 1", v)
			}
		})
	}
}
