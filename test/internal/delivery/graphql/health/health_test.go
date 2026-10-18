package health_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/health"
	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	health.Healthz().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestReadyz(t *testing.T) {
	hang := func(ctx context.Context) error { // a database that never answers
		<-ctx.Done()
		return ctx.Err()
	}

	t.Run("database answers", func(t *testing.T) {
		logs := testutil.CaptureLogs(t)
		rec := httptest.NewRecorder()
		health.Readyz(func(ctx context.Context) error { return nil }).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

		if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
			t.Fatalf("status %d body %q, want 200 ok", rec.Code, rec.Body.String())
		}
		if logs.Len() != 0 {
			t.Fatalf("nothing should be logged when ready: %s", logs.String())
		}
	})

	t.Run("database down: 503, error logged with request ID but not returned", func(t *testing.T) {
		logs := testutil.CaptureLogs(t)
		const detail = "dial tcp 10.0.0.5:5432: connection refused"
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		req = req.WithContext(logging.WithRequestID(req.Context(), "probe-1"))
		rec := httptest.NewRecorder()
		health.Readyz(func(ctx context.Context) error { return errors.New(detail) }).ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
		if strings.Contains(rec.Body.String(), "10.0.0.5") {
			t.Fatalf("body leaks the database error: %q", rec.Body.String())
		}
		lines := testutil.LogLines(t, logs)
		if len(lines) != 1 || lines[0]["err"] != detail || lines[0]["request_id"] != "probe-1" {
			t.Fatalf("want one warning with the error and request_id, got %v", lines)
		}
	})

	t.Run("ping gets a deadline", func(t *testing.T) {
		testutil.CaptureLogs(t)
		var hasDeadline bool
		health.Readyz(func(ctx context.Context) error {
			_, hasDeadline = ctx.Deadline()
			return nil
		}).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/readyz", nil))

		if !hasDeadline {
			t.Fatal("ping context has no deadline; a hanging database would hang the health check")
		}
	})

	t.Run("database hangs: cut off by the timeout", func(t *testing.T) {
		testutil.CaptureLogs(t)
		start := time.Now()
		rec := httptest.NewRecorder()
		health.Readyz(hang).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
		if elapsed := time.Since(start); elapsed > 3*time.Second {
			t.Fatalf("took %v, want about 2s (the timeout did not stop the ping)", elapsed)
		}
	})

	t.Run("client gives up first: stops immediately", func(t *testing.T) {
		testutil.CaptureLogs(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		start := time.Now()
		rec := httptest.NewRecorder()
		health.Readyz(hang).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil).WithContext(ctx))

		if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
			t.Fatalf("took %v; the ping should stop as soon as the request is cancelled", elapsed)
		}
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})
}
