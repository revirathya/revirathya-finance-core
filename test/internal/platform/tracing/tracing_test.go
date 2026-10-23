package tracing_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/platform/tracing"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

// installed runs tracing.Install with an in-memory exporter, records one span and returns it.
func installed(t *testing.T) tracetest.SpanStub {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := tracing.Install(context.Background(), exporter)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	_, span := otel.Tracer("test").Start(context.Background(), "op")
	span.End()
	if err := otel.GetTracerProvider().(*sdktrace.TracerProvider).ForceFlush(context.Background()); err != nil {
		t.Fatalf("ForceFlush: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("%d spans exported, want 1", len(spans))
	}
	return spans[0]
}

func resourceValue(span tracetest.SpanStub, key string) string {
	v, _ := span.Resource.Set().Value(attribute.Key(key))
	return v.AsString()
}

func TestSetup(t *testing.T) {
	t.Run("no endpoint leaves tracing off", func(t *testing.T) {
		t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
		before := otel.GetTracerProvider()

		shutdown, err := tracing.Setup(context.Background())
		if err != nil {
			t.Fatalf("Setup: %v", err)
		}
		if err := shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown: %v", err)
		}
		if otel.GetTracerProvider() != before {
			t.Fatal("global tracer provider changed without an endpoint")
		}
	})

	t.Run("an endpoint installs the SDK provider without connecting", func(t *testing.T) {
		t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1") // nothing listens; export isn't attempted
		shutdown, err := tracing.Setup(context.Background())
		if err != nil {
			t.Fatalf("Setup: %v", err)
		}
		t.Cleanup(func() { _ = shutdown(context.Background()) })

		if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
			t.Fatalf("global provider is %T, want *sdktrace.TracerProvider", otel.GetTracerProvider())
		}
	})
}

func TestInstall(t *testing.T) {
	t.Run("service name defaults to financecore", func(t *testing.T) {
		t.Setenv("OTEL_SERVICE_NAME", "")
		t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
		if got := resourceValue(installed(t), "service.name"); got != "financecore" {
			t.Fatalf("service.name %q, want financecore", got)
		}
	})

	t.Run("environment sets the version and overrides the name", func(t *testing.T) {
		t.Setenv("OTEL_SERVICE_NAME", "financecore-staging")
		t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.version=1.2.3")
		span := installed(t)
		if got := resourceValue(span, "service.name"); got != "financecore-staging" {
			t.Fatalf("service.name %q, want financecore-staging", got)
		}
		if got := resourceValue(span, "service.version"); got != "1.2.3" {
			t.Fatalf("service.version %q, want 1.2.3", got)
		}
	})

	t.Run("W3C trace context is propagated", func(t *testing.T) {
		parent := installed(t).SpanContext
		carrier := propagation.MapCarrier{
			"traceparent": "00-" + parent.TraceID().String() + "-" + parent.SpanID().String() + "-01",
		}
		ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
		_, child := otel.Tracer("test").Start(ctx, "child")
		defer child.End()

		if child.SpanContext().TraceID() != parent.TraceID() {
			t.Fatal("span started from an incoming traceparent has a new trace ID, want the caller's")
		}
	})
}

func TestRunShutdown(t *testing.T) {
	t.Run("a failed shutdown is logged", func(t *testing.T) {
		logs := testutil.CaptureLogs(t)
		tracing.RunShutdown(func(context.Context) error { return errors.New("collector unreachable") })

		if !strings.Contains(logs.String(), "shutdown tracing") || !strings.Contains(logs.String(), "collector unreachable") {
			t.Fatalf("log %q, want the shutdown error", logs.String())
		}
	})

	t.Run("a clean shutdown logs nothing", func(t *testing.T) {
		logs := testutil.CaptureLogs(t)
		tracing.RunShutdown(func(context.Context) error { return nil })
		if logs.Len() != 0 {
			t.Fatalf("log %q, want nothing", logs.String())
		}
	})
}
