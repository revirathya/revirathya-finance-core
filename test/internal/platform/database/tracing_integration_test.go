//go:build integration

package database_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/platform/database"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

// querySpans returns the spans whose query text contains marker.
func querySpans(t *testing.T, marker string) []tracetest.SpanStub {
	t.Helper()
	var found []tracetest.SpanStub
	for _, s := range testutil.Spans(t) {
		for _, a := range s.Attributes {
			if a.Key == "db.query.text" && strings.Contains(a.Value.String(), marker) {
				found = append(found, s)
			}
		}
	}
	return found
}

func TestDBPoolTracing(t *testing.T) {
	testutil.InstallTracing(t)
	ctx := context.Background()
	pool, err := database.NewDBPool(ctx, testutil.TestDSN)
	if err != nil {
		t.Fatalf("NewDBPool: %v", err)
	}
	t.Cleanup(pool.Close)

	t.Run("a query inside a request gets a span with its SQL text but not its parameters", func(t *testing.T) {
		reqCtx, parent := otel.Tracer("test").Start(ctx, "request")
		param := "param-" + uuid.NewString()
		var got string
		err := pool.QueryRow(reqCtx, "SELECT $1::text AS traced_query", param).Scan(&got)
		parent.End()
		if err != nil {
			t.Fatalf("query: %v", err)
		}

		spans := querySpans(t, "traced_query")
		if len(spans) == 0 {
			t.Fatal("no span carries the query text")
		}
		for _, s := range spans {
			if s.SpanContext.TraceID() != parent.SpanContext().TraceID() {
				t.Fatalf("query span in trace %s, want the request's %s", s.SpanContext.TraceID(), parent.SpanContext().TraceID())
			}
			for _, a := range s.Attributes {
				if strings.Contains(a.Value.String(), param) {
					t.Fatalf("query parameter recorded in %s", a.Key)
				}
			}
		}
	})

	t.Run("a query outside any request isn't traced", func(t *testing.T) {
		if _, err := pool.Exec(ctx, "SELECT 1 AS untraced_query"); err != nil {
			t.Fatalf("query: %v", err)
		}
		if n := len(querySpans(t, "untraced_query")); n != 0 {
			t.Fatalf("%d spans for a query without a parent span, want 0", n)
		}
	})
}
