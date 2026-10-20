package grpc_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	// Generated
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/grpc/financecore/v1"

	// Internal
	dlvGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/usecase"
	"github.com/revirathya/revirathya-finance-core/testutil"
)

// startServer runs the production server from Setup on a free port and returns a client connection.
func startServer(t *testing.T) *grpc.ClientConn {
	t.Helper()
	conn, _ := startServerWithMetrics(t)
	return conn
}

// startServerWithMetrics is startServer, also returning the metrics the server records into.
func startServerWithMetrics(t *testing.T) (*grpc.ClientConn, *metrics.Metrics) {
	t.Helper()
	m := metrics.NewMetrics(nil)
	layer, err := dlvGrpc.Setup("127.0.0.1:0", false, "test-api-key", m, usecase.Usecase{})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	go func() { _ = layer.Server.Serve(layer.Listener) }()
	t.Cleanup(layer.Server.Stop)

	conn, err := grpc.NewClient(layer.Listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn, m
}

func TestServer(t *testing.T) {
	conn := startServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("health check is served without an API key", func(t *testing.T) {
		resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
			t.Fatalf("status %v, want SERVING", resp.GetStatus())
		}
	})

	t.Run("request within the size limit reaches the interceptors", func(t *testing.T) {
		_, err := financecorev1.NewCategoryServiceClient(conn).GetCategoryByID(ctx, &financecorev1.GetCategoryByIDRequest{Id: strings.Repeat("a", 512<<10)})
		if code := status.Code(err); code != codes.Unauthenticated {
			t.Fatalf("code %v (%v), want Unauthenticated from apikeyauth", code, err)
		}
	})

	t.Run("request over 1 MB is rejected before any interceptor", func(t *testing.T) {
		_, err := financecorev1.NewCategoryServiceClient(conn).GetCategoryByID(ctx, &financecorev1.GetCategoryByIDRequest{Id: strings.Repeat("a", 1<<20)})
		if code := status.Code(err); code != codes.ResourceExhausted {
			t.Fatalf("code %v (%v), want ResourceExhausted", code, err)
		}
	})
}

func TestServerMetrics(t *testing.T) {
	conn, m := startServerWithMetrics(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("every registered method is exported at zero before any call", func(t *testing.T) {
		v, ok := testutil.MetricValue(t, m.Registry, "grpc_server_handled_total",
			map[string]string{"grpc_service": "financecore.v1.ExpenseService", "grpc_method": "ListExpenses", "grpc_code": "OK"})
		if !ok || v != 0 {
			t.Fatalf("ListExpenses OK = %v (exported: %v), want 0 and exported", v, ok)
		}
	})

	t.Run("a call is counted by method and code", func(t *testing.T) {
		// No API key: rejected by apikeyauth, which runs inside the metrics interceptor.
		_, err := financecorev1.NewCategoryServiceClient(conn).ListCategories(ctx, &financecorev1.ListCategoriesRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code %v, want Unauthenticated", status.Code(err))
		}

		labels := map[string]string{"grpc_service": "financecore.v1.CategoryService", "grpc_method": "ListCategories", "grpc_code": "Unauthenticated"}
		if v, _ := testutil.MetricValue(t, m.Registry, "grpc_server_handled_total", labels); v != 1 {
			t.Fatalf("grpc_server_handled_total%v = %v, want 1", labels, v)
		}
		delete(labels, "grpc_code")
		if v, _ := testutil.MetricValue(t, m.Registry, "grpc_server_handling_seconds", labels); v != 1 {
			t.Fatalf("grpc_server_handling_seconds samples %v, want 1", v)
		}
	})
}
