package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware"
)

// record returns a middleware that appends name to order when a request passes through it.
func record(name string, order *[]string) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*order = append(*order, name)
			next.ServeHTTP(w, r)
		})
	}
}

func TestChainMiddleware(t *testing.T) {
	tests := []struct {
		name  string
		names []string
		want  []string
	}{
		{"no middleware", nil, []string{"handler"}},
		{"one middleware", []string{"A"}, []string{"A", "handler"}},
		{"two middlewares", []string{"A", "B"}, []string{"A", "B", "handler"}},
		{"four middlewares", []string{"A", "B", "C", "D"}, []string{"A", "B", "C", "D", "handler"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var order []string
			var mws []middleware.Middleware
			for _, n := range tt.names {
				mws = append(mws, record(n, &order))
			}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "handler")
			})

			middleware.ChainMiddleware(handler, mws...).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

			// Every middleware must run, in the order listed (the first one is outermost).
			if !slices.Equal(order, tt.want) {
				t.Fatalf("run order %v, want %v", order, tt.want)
			}
		})
	}
}
