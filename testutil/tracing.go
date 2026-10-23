package testutil

import (
	"context"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/revirathya/revirathya-finance-core/internal/platform/tracing"
)

var (
	tracingOnce     sync.Once
	tracingExporter *tracetest.InMemoryExporter
)

// InstallTracing installs the service's tracer provider (tracing.Install) once per test binary, exporting into
// memory. Instrumentation captures the provider when it is built, so call it before building servers or pools.
// Spans from every test in the package accumulate; find yours by name or attribute.
func InstallTracing(t *testing.T) {
	t.Helper()
	tracingOnce.Do(func() {
		tracingExporter = tracetest.NewInMemoryExporter()
		if _, err := tracing.Install(context.Background(), tracingExporter); err != nil {
			t.Fatalf("tracing.Install: %v", err)
		}
	})
}

// Spans flushes the batcher and returns every span exported so far.
func Spans(t *testing.T) tracetest.SpanStubs {
	t.Helper()
	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	if !ok {
		t.Fatal("InstallTracing was not called")
	}
	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("ForceFlush: %v", err)
	}
	return tracingExporter.GetSpans()
}
