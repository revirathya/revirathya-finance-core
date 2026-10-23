package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dlvHttp "github.com/revirathya/revirathya-finance-core/internal/delivery/http"
)

func get(t *testing.T, env, path string) *httptest.ResponseRecorder {
	t.Helper()
	layer := dlvHttp.Setup(env, ":0", "test", func(ctx context.Context) error { return nil })
	rec := httptest.NewRecorder()
	layer.Server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestSetupPlayground(t *testing.T) {
	t.Run("dev serves the GraphQL playground, pointed at /graphql", func(t *testing.T) {
		rec := get(t, "dev", "/playground")

		if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			t.Fatalf("status %d, content type %q; want 200 HTML", rec.Code, rec.Header().Get("Content-Type"))
		}
		if !strings.Contains(rec.Body.String(), "/graphql") {
			t.Fatal("playground page doesn't point at /graphql")
		}
	})

	t.Run("prod doesn't serve it", func(t *testing.T) {
		if rec := get(t, "prod", "/playground"); rec.Code != http.StatusNotFound {
			t.Fatalf("status %d, want 404", rec.Code)
		}
	})

	t.Run("health checks are served in both", func(t *testing.T) {
		for _, env := range []string{"dev", "prod"} {
			if rec := get(t, env, "/healthz"); rec.Code != http.StatusOK {
				t.Fatalf("%s: /healthz status %d, want 200", env, rec.Code)
			}
		}
	})
}
