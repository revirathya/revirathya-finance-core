package version_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	// Internal - HTTP
	dlvHttp "github.com/revirathya/revirathya-finance-core/internal/delivery/http"
)

func TestVersion(t *testing.T) {
	layer := dlvHttp.Setup("prod", ":0", "0.2.0", func(ctx context.Context) error { return nil })

	t.Run("GET /version returns the build version as JSON", func(t *testing.T) {
		rec := httptest.NewRecorder()
		layer.Server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("body %q is not JSON: %v", rec.Body.String(), err)
		}
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" || body["version"] != "0.2.0" {
			t.Fatalf("status %d, type %q, body %v; want 200 JSON with version 0.2.0", rec.Code, rec.Header().Get("Content-Type"), body)
		}
	})

	t.Run("other methods are rejected", func(t *testing.T) {
		rec := httptest.NewRecorder()
		layer.Server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/version", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status %d, want 405", rec.Code)
		}
	})
}
