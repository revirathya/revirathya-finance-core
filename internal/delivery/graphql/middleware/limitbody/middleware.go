package limitbody

import (
	"net/http"

	// Internal - GraphQL
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware"
)

func Middleware(max int64) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > max {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, max)
			next.ServeHTTP(w, r)
		})
	}
}
