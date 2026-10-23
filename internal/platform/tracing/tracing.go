package tracing

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

const defaultServiceName = "financecore"

// Install sets up the provider with the given exporter; tests pass an in-memory one.
// OTEL_SERVICE_NAME, OTEL_RESOURCE_ATTRIBUTES and OTEL_TRACES_SAMPLER are read from the environment.
func Install(ctx context.Context, exporter sdktrace.SpanExporter) (func(context.Context) error, error) {
	res, err := resource.New(
		ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(defaultServiceName),
		),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
	return tp.Shutdown, nil
}

// Setup installs the global tracer provider when OTEL_EXPORTER_OTLP_ENDPOINT is set, and returns a shutdown
// function that flushes pending spans. With no endpoint, tracing stays off (OTel's default no-op provider).
func Setup(ctx context.Context) (func(context.Context) error, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return func(context.Context) error {
			return nil
		}, nil
	}

	exporter, err := otlptracegrpc.New(ctx) // reads OTEL_EXPORTER_OTLP_* (an http:// endpoint means no TLS)
	if err != nil {
		return nil, fmt.Errorf("otlp exporter: %w", err)
	}
	return Install(ctx, exporter)
}

// RunShutdown run the shutdown function with timeout context.
// Used as defer with the generated tracing shutdown function from Install / Setup
func RunShutdown(shutdownFunc func(context.Context) error) {
	shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := shutdownFunc(shCtx); err != nil {
		slog.Error("shutdown tracing", "err", err)
	}
}
