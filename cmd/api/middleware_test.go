package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
)

func TestWithRequestID(t *testing.T) {
	// serve sends a request through withRequestID and returns the ID the next
	// handler saw in its context, plus the response.
	serve := func(t *testing.T, header string) (string, *httptest.ResponseRecorder) {
		t.Helper()
		var seen string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen = logging.RequestIDFrom(r.Context())
		})

		req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
		if header != "" {
			req.Header.Set(logging.RequestIDHeader, header)
		}
		rec := httptest.NewRecorder()
		withRequestID(next).ServeHTTP(rec, req)
		return seen, rec
	}

	t.Run("generates an ID when none is sent", func(t *testing.T) {
		seen, rec := serve(t, "")

		if _, err := uuid.Parse(seen); err != nil {
			t.Fatalf("context ID %q, want a generated UUID", seen)
		}
		if got := rec.Header().Get(logging.RequestIDHeader); got != seen {
			t.Fatalf("response header %q, want the context ID %q", got, seen)
		}
	})

	t.Run("keeps a valid incoming ID", func(t *testing.T) {
		seen, rec := serve(t, "web-console-42")

		if seen != "web-console-42" {
			t.Fatalf("context ID %q, want %q", seen, "web-console-42")
		}
		if got := rec.Header().Get(logging.RequestIDHeader); got != "web-console-42" {
			t.Fatalf("response header %q, want %q", got, "web-console-42")
		}
	})

	t.Run("replaces an invalid incoming ID", func(t *testing.T) {
		const forged = "abc level=ERROR msg=forged"
		seen, rec := serve(t, forged)

		if seen == forged {
			t.Fatal("invalid ID reached the context")
		}
		if _, err := uuid.Parse(seen); err != nil {
			t.Fatalf("context ID %q, want a generated UUID", seen)
		}
		if got := rec.Header().Get(logging.RequestIDHeader); got != seen {
			t.Fatalf("response header %q, want the context ID %q", got, seen)
		}
	})
}

func TestWithCORSRequestID(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := withCORS([]string{testOrigin}, next)

	t.Run("preflight allows the request ID header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/graphql", nil)
		req.Header.Set("Origin", testOrigin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "content-type, x-request-id")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		allowed := strings.ToLower(rec.Header().Get("Access-Control-Allow-Headers"))
		if !strings.Contains(allowed, "x-request-id") {
			t.Fatalf("Access-Control-Allow-Headers %q does not allow X-Request-ID", allowed)
		}
	})

	t.Run("actual responses expose the request ID header", func(t *testing.T) {
		// Browsers only let JavaScript read X-Request-ID on the real response
		// (not the preflight) if it is listed in Access-Control-Expose-Headers.
		req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
		req.Header.Set("Origin", testOrigin)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		exposed := strings.ToLower(rec.Header().Get("Access-Control-Expose-Headers"))
		if !strings.Contains(exposed, "x-request-id") {
			t.Fatalf("Access-Control-Expose-Headers %q on a POST response, want it to include X-Request-ID", exposed)
		}
	})
}

func TestWithAccessLog(t *testing.T) {
	// serve sends one request through withRequestID(withAccessLog(handler)), the
	// same order as production, and returns the decoded log lines.
	serve := func(t *testing.T, req *http.Request, handler http.HandlerFunc) []map[string]any {
		t.Helper()
		var buf bytes.Buffer
		logger, err := logging.NewLogger(&buf, "json", slog.LevelDebug)
		if err != nil {
			t.Fatalf("NewLogger: %v", err)
		}
		previous := slog.Default()
		slog.SetDefault(logger)
		t.Cleanup(func() { slog.SetDefault(previous) })

		withRequestID(withAccessLog(handler)).ServeHTTP(httptest.NewRecorder(), req)

		var lines []map[string]any
		dec := json.NewDecoder(&buf)
		for dec.More() {
			var line map[string]any
			if err := dec.Decode(&line); err != nil {
				t.Fatalf("invalid log line: %v", err)
			}
			lines = append(lines, line)
		}
		return lines
	}
	status := func(code int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(code) }
	}

	t.Run("logs method, path, status, duration and request ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
		req.Header.Set(logging.RequestIDHeader, "web-console-42")

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
