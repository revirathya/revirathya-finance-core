package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/revirathya/revirathya-finance-core/cmd/api"
	dlvGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
	dlvHttp "github.com/revirathya/revirathya-finance-core/internal/delivery/http"
	dlvMetrics "github.com/revirathya/revirathya-finance-core/internal/delivery/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/platform/ratelimit"
	"github.com/revirathya/revirathya-finance-core/internal/usecase"
)

func TestRunServers(t *testing.T) {
	t.Run("all servers stop and Wait returns once the context is cancelled", func(t *testing.T) {
		m := metrics.NewMetrics(nil)
		grpcLayer, err := dlvGrpc.Setup("127.0.0.1:0", false, "test-api-key", m, ratelimit.NewLimiter(1000, time.Minute), usecase.Usecase{})
		if err != nil {
			t.Fatalf("grpc setup: %v", err)
		}
		httpLayer := dlvHttp.Setup("prod", "127.0.0.1:0", "test", func(ctx context.Context) error { return nil })
		metricsLayer := dlvMetrics.Setup("127.0.0.1:0", m)

		ctx, cancel := context.WithCancel(context.Background())
		g := api.RunServers(ctx, grpcLayer, httpLayer, metricsLayer)
		time.Sleep(100 * time.Millisecond) // let the servers start listening
		cancel()

		done := make(chan error, 1)
		go func() { done <- g.Wait() }()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("Wait: %v, want a clean shutdown", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Wait did not return after cancellation: a server was not shut down")
		}
	})
}
