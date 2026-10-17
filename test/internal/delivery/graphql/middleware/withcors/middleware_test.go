package withcors_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware/withcors"
)

const allowed = "http://localhost:3000"

func newHandler(called *bool) http.Handler {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
	return withcors.Middleware([]string{allowed})(next)
}

func TestMiddleware(t *testing.T) {
	t.Run("preflight from an allowed origin is answered here", func(t *testing.T) {
		var called bool
		req := httptest.NewRequest(http.MethodOptions, "/graphql", nil)
		req.Header.Set("Origin", allowed)
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		newHandler(&called).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status %d, want %d", rec.Code, http.StatusNoContent)
		}
		if called {
			t.Fatal("preflight reached the next handler")
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowed {
			t.Fatalf("Access-Control-Allow-Origin %q, want %q", got, allowed)
		}
		headers := strings.ToLower(rec.Header().Get("Access-Control-Allow-Headers"))
		for _, h := range []string{"content-type", "authorization", "x-request-id"} {
			if !strings.Contains(headers, h) {
				t.Fatalf("Access-Control-Allow-Headers %q is missing %s", headers, h)
			}
		}
	})

	t.Run("actual request from an allowed origin passes through and exposes X-Request-ID", func(t *testing.T) {
		var called bool
		req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
		req.Header.Set("Origin", allowed)
		rec := httptest.NewRecorder()
		newHandler(&called).ServeHTTP(rec, req)

		if !called {
			t.Fatal("request did not reach the next handler")
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowed {
			t.Fatalf("Access-Control-Allow-Origin %q, want %q", got, allowed)
		}
		if exposed := strings.ToLower(rec.Header().Get("Access-Control-Expose-Headers")); !strings.Contains(exposed, "x-request-id") {
			t.Fatalf("Access-Control-Expose-Headers %q, want it to include X-Request-ID", exposed)
		}
	})

	t.Run("unknown origin gets no CORS headers", func(t *testing.T) {
		var called bool
		req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
		req.Header.Set("Origin", "https://evil.example")
		rec := httptest.NewRecorder()
		newHandler(&called).ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("Access-Control-Allow-Origin %q for an unknown origin, want none", got)
		}
		if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "" {
			t.Fatalf("Access-Control-Expose-Headers %q for an unknown origin, want none", got)
		}
	})

	t.Run("request without Origin passes through unchanged", func(t *testing.T) {
		var called bool
		rec := httptest.NewRecorder()
		newHandler(&called).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/graphql", nil))

		if !called || rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("called %v, Allow-Origin %q; want passed through without CORS headers", called, rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}
