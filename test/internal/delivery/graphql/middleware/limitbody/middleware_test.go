package limitbody_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware/limitbody"
)

const limit = 16

// readAll is a next handler that reads the whole body, like gqlgen does, and reports
// what happened: 200 with the body, or 400 if reading failed.
var readAll = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read failed", http.StatusBadRequest)
		return
	}
	_, _ = w.Write(body)
})

func TestMiddleware(t *testing.T) {
	t.Run("body within the limit passes through", func(t *testing.T) {
		rec := httptest.NewRecorder()
		limitbody.Middleware(limit)(readAll).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader("small")))

		if rec.Code != http.StatusOK || rec.Body.String() != "small" {
			t.Fatalf("status %d body %q, want 200 small", rec.Code, rec.Body.String())
		}
	})

	t.Run("body over the limit with Content-Length is rejected with 413", func(t *testing.T) {
		var called bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
		rec := httptest.NewRecorder()
		limitbody.Middleware(limit)(next).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(strings.Repeat("x", limit+1))))

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
		}
		if called {
			t.Fatal("oversized request reached the next handler")
		}
	})

	t.Run("body over the limit without Content-Length can't be read past the limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(strings.Repeat("x", limit+1)))
		req.ContentLength = -1 // e.g. chunked transfer encoding: length unknown
		rec := httptest.NewRecorder()
		limitbody.Middleware(limit)(readAll).ServeHTTP(rec, req)

		if rec.Code == http.StatusOK {
			t.Fatalf("oversized body without Content-Length was read in full: %q", rec.Body.String())
		}
	})
}
