package requestid

import (
	"net/http"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware"
	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
)

func Middleware() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := logging.EnsureRequestID(r.Header.Get(logging.RequestIDHeader))
			w.Header().Set(logging.RequestIDHeader, id)

			rLog := r.WithContext(logging.WithRequestID(r.Context(), id))
			next.ServeHTTP(w, rLog)
		})
	}
}
