package metricscollector

import (
	"net/http"
	"time"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"

	// Internal - HTTP
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware"
)

// Middleware records count and latency for one route; handler is a fixed name like "graphql"
func Middleware(metricsColl *metrics.Metrics, handler string) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				start := time.Now()
				rec := middleware.NewStatusRecorder(w)

				next.ServeHTTP(rec, r)

				metricsColl.ObserveHTTP(handler, rec.Code(), time.Since(start))
			},
		)
	}
}
