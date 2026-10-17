package accesslog_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware/accesslog"
	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
	"github.com/revirathya/revirathya-finance-core/testutil"
)

// serve sends one request through the access log middleware and returns the log lines.
func serve(t *testing.T, req *http.Request, handler http.HandlerFunc) []map[string]any {
	t.Helper()
	logs := testutil.CaptureLogs(t)
	accesslog.Middleware()(handler).ServeHTTP(httptest.NewRecorder(), req)
	return testutil.LogLines(t, logs)
}

func status(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(code) }
}

func TestMiddleware(t *testing.T) {
	t.Run("logs method, path, status, duration and request ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
		req = req.WithContext(logging.WithRequestID(req.Context(), "web-console-42"))

		lines := serve(t, req, status(http.StatusOK))

		if len(lines) != 1 {
			t.Fatalf("got %d lines, want 1", len(lines))
		}
		line := lines[0]
		if line["msg"] != "http request" || line["method"] != "POST" || line["path"] != "/graphql" {
			t.Fatalf("unexpected line: %v", line)
		}
		if line["status"] != float64(http.StatusOK) || line["level"] != "INFO" {
			t.Fatalf("status %v level %v, want 200 INFO", line["status"], line["level"])
		}
		if line["request_id"] != "web-console-42" {
			t.Fatalf("request_id %v, want web-console-42", line["request_id"])
		}
		if _, ok := line["duration_ms"]; !ok {
			t.Fatalf("missing duration_ms: %v", line)
		}
	})

	t.Run("status and level by outcome", func(t *testing.T) {
		cases := []struct {
			name    string
			handler http.HandlerFunc
			status  int
			level   string
		}{
			{"write without WriteHeader is 200", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) }, 200, "INFO"},
			{"handler writes nothing is 200", func(w http.ResponseWriter, r *http.Request) {}, 200, "INFO"},
			{"redirect", status(http.StatusFound), 302, "INFO"},
			{"not found", status(http.StatusNotFound), 404, "WARN"},
			{"payload too large", status(http.StatusRequestEntityTooLarge), 413, "WARN"},
			{"internal error", status(http.StatusInternalServerError), 500, "ERROR"},
			{"timeout", status(http.StatusServiceUnavailable), 503, "ERROR"},
			{"first WriteHeader wins", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.WriteHeader(http.StatusOK) // ignored by net/http
			}, 400, "WARN"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				lines := serve(t, httptest.NewRequest(http.MethodPost, "/graphql", nil), tc.handler)

				if len(lines) != 1 {
					t.Fatalf("got %d lines, want 1", len(lines))
				}
				if lines[0]["status"] != float64(tc.status) || lines[0]["level"] != tc.level {
					t.Fatalf("status %v level %v, want %d %s", lines[0]["status"], lines[0]["level"], tc.status, tc.level)
				}
			})
		}
	})

	t.Run("never logs query strings or headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/graphql?token=query-secret", nil)
		req.Header.Set("Authorization", "Bearer header-secret")

		lines := serve(t, req, status(http.StatusOK))

		raw, _ := json.Marshal(lines)
		for _, secret := range []string{"query-secret", "header-secret"} {
			if strings.Contains(string(raw), secret) {
				t.Fatalf("log contains %q: %s", secret, raw)
			}
		}
		if lines[0]["path"] != "/graphql" {
			t.Fatalf("path %v, want /graphql without query", lines[0]["path"])
		}
	})

	t.Run("successful health checks are not logged", func(t *testing.T) {
		for _, path := range []string{"/healthz", "/readyz"} {
			if lines := serve(t, httptest.NewRequest(http.MethodGet, path, nil), status(http.StatusOK)); len(lines) != 0 {
				t.Fatalf("%s 200 was logged: %v", path, lines)
			}
		}
	})

	t.Run("failing readiness check is logged", func(t *testing.T) {
		lines := serve(t, httptest.NewRequest(http.MethodGet, "/readyz", nil), status(http.StatusServiceUnavailable))

		if len(lines) != 1 || lines[0]["level"] != "ERROR" {
			t.Fatalf("want one ERROR line for /readyz 503, got %v", lines)
		}
	})
}
