package grpc_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"

	// Generated
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/grpc/financecore/v1"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

// serverSpans returns the server spans recorded for the full method name.
func serverSpans(t *testing.T, method string) []tracetest.SpanStub {
	t.Helper()
	var found []tracetest.SpanStub
	for _, s := range testutil.Spans(t) {
		if s.Name == method && s.SpanKind == trace.SpanKindServer {
			found = append(found, s)
		}
	}
	return found
}

func attr(s tracetest.SpanStub, key string) string {
	for _, a := range s.Attributes {
		if string(a.Key) == key {
			return a.Value.String()
		}
	}
	return ""
}

func TestServerTracing(t *testing.T) {
	testutil.InstallTracing(t)
	conn := startServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("an RPC gets a server span with its method and status code", func(t *testing.T) {
		_, _ = financecorev1.NewCategoryServiceClient(conn).ListCategories(ctx, &financecorev1.ListCategoriesRequest{})

		spans := serverSpans(t, "financecore.v1.CategoryService/ListCategories")
		if len(spans) != 1 {
			t.Fatalf("%d ListCategories spans, want 1", len(spans))
		}
		if got := attr(spans[0], "rpc.response.status_code"); got != "UNAUTHENTICATED" {
			t.Fatalf("rpc.response.status_code %q, want UNAUTHENTICATED", got)
		}
	})

	t.Run("the span joins the caller's trace from traceparent", func(t *testing.T) {
		const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
		callCtx := metadata.AppendToOutgoingContext(ctx, "traceparent", "00-"+traceID+"-00f067aa0ba902b7-01")
		_, _ = financecorev1.NewExpenseServiceClient(conn).ListExpenses(callCtx, &financecorev1.ListExpensesRequest{})

		spans := serverSpans(t, "financecore.v1.ExpenseService/ListExpenses")
		if len(spans) != 1 {
			t.Fatalf("%d ListExpenses spans, want 1", len(spans))
		}
		if got := spans[0].SpanContext.TraceID().String(); got != traceID {
			t.Fatalf("trace ID %s, want the caller's %s", got, traceID)
		}
	})

	t.Run("health checks aren't traced", func(t *testing.T) {
		if _, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{}); err != nil {
			t.Fatalf("health check: %v", err)
		}
		for _, s := range testutil.Spans(t) {
			if strings.HasPrefix(s.Name, "grpc.health.") {
				t.Fatalf("health check traced as %q", s.Name)
			}
		}
	})
}
