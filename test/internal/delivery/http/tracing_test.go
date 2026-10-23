package http_test

import (
	"testing"

	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

// spansForPath returns the server spans recorded for path.
func spansForPath(t *testing.T, path string) []tracetest.SpanStub {
	t.Helper()
	var found []tracetest.SpanStub
	for _, s := range testutil.Spans(t) {
		for _, a := range s.Attributes {
			if a.Key == "url.path" && a.Value.AsString() == path && s.SpanKind == trace.SpanKindServer {
				found = append(found, s)
			}
		}
	}
	return found
}

func TestSetupTracing(t *testing.T) {
	testutil.InstallTracing(t)

	t.Run("a request gets a server span named after its route", func(t *testing.T) {
		get(t, "dev", "/playground")
		spans := spansForPath(t, "/playground")
		if len(spans) != 1 {
			t.Fatalf("%d spans for /playground, want 1", len(spans))
		}
		if spans[0].Name != "GET /playground" {
			t.Fatalf("span name %q, want %q", spans[0].Name, "GET /playground")
		}
	})

	t.Run("an unknown path is named by method only", func(t *testing.T) {
		path := "/no-such-route/123"
		get(t, "dev", path)
		spans := spansForPath(t, path)
		if len(spans) != 1 || spans[0].Name != "GET" {
			t.Fatalf("spans %v, want one named GET (paths would make names unbounded)", spans)
		}
	})

	t.Run("health checks aren't traced", func(t *testing.T) {
		get(t, "dev", "/healthz")
		get(t, "dev", "/readyz")
		if n := len(spansForPath(t, "/healthz")) + len(spansForPath(t, "/readyz")); n != 0 {
			t.Fatalf("%d spans for health checks, want 0", n)
		}
	})
}
