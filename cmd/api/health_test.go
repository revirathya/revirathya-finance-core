package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
)

// captureLogs redirects the default slog logger into a buffer for the rest of the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	healthz().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestReadyz(t *testing.T) {
	t.Run("database answers", func(t *testing.T) {
		logs := captureLogs(t)
		ping := func(ctx context.Context) error { return nil }

		rec := httptest.NewRecorder()
		readyz(ping).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
		}
		if body := rec.Body.String(); body != "ok" {
			t.Fatalf("body: got %q, want %q", body, "ok")
		}
		if logs.Len() != 0 {
			t.Fatalf("nothing should be logged when ready, got:\n%s", logs.String())
		}
	})

	t.Run("database down", func(t *testing.T) {
		logs := captureLogs(t)
		const detail = "dial tcp 10.0.0.5:5432: connection refused"
		ping := func(ctx context.Context) error { return errors.New(detail) }

		rec := httptest.NewRecorder()
		readyz(ping).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status: got %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
		if body := rec.Body.String(); strings.Contains(body, "10.0.0.5") {
			t.Fatalf("body leaks the database error: %q", body)
		}
		if !strings.Contains(logs.String(), detail) {
			t.Fatalf("the database error should be logged, got:\n%s", logs.String())
		}
	})

	t.Run("ping gets a deadline", func(t *testing.T) {
		captureLogs(t)
		var hasDeadline bool
		var until time.Duration
		ping := func(ctx context.Context) error {
			var deadline time.Time
			deadline, hasDeadline = ctx.Deadline()
			until = time.Until(deadline)
			return nil
		}

		readyz(ping).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/readyz", nil))

		if !hasDeadline {
			t.Fatal("ping context has no deadline; a hanging database would hang the health check")
		}
		if until <= 0 || until > readyTimeout {
			t.Fatalf("deadline is %v away, want within %v", until, readyTimeout)
		}
	})

	t.Run("database hangs", func(t *testing.T) {
		captureLogs(t)
		// Waits until the context is cancelled, like a database that never answers.
		ping := func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}

		start := time.Now()
		rec := httptest.NewRecorder()
		readyz(ping).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		elapsed := time.Since(start)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status: got %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
		if elapsed > readyTimeout+time.Second {
			t.Fatalf("took %v, want about %v (the timeout did not stop the ping)", elapsed, readyTimeout)
		}
	})

	t.Run("client gives up first", func(t *testing.T) {
		captureLogs(t)
		ping := func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}

		// The request's own context is already cancelled, e.g. the checker disconnected.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil).WithContext(ctx)

		start := time.Now()
		rec := httptest.NewRecorder()
		readyz(ping).ServeHTTP(rec, req)

		if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
			t.Fatalf("took %v; the ping should stop as soon as the request is cancelled", elapsed)
		}
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status: got %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})
}

func TestReadyzLogsRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger, err := logging.NewLogger(&buf, "json", slog.LevelDebug)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	previous := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(previous) })

	ping := func(ctx context.Context) error { return errors.New("connection refused") }
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	req = req.WithContext(logging.WithRequestID(req.Context(), "probe-1"))
	readyz(ping).ServeHTTP(httptest.NewRecorder(), req)

	if !strings.Contains(buf.String(), `"request_id":"probe-1"`) {
		t.Fatalf("readiness warning has no request_id: %s", buf.String())
	}
}
