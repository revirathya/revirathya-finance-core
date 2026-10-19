package timeout_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/interceptor/timeout"
)

var info = &grpc.UnaryServerInfo{FullMethod: "/financecore.v1.ExpenseService/ListExpenses"}

// run calls the interceptor and returns the context the handler saw.
func run(t *testing.T, ctx context.Context, d time.Duration) context.Context {
	t.Helper()
	var got context.Context
	_, err := timeout.UnaryServerInterceptor(d)(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		got = ctx
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return got
}

func TestUnaryServerInterceptor(t *testing.T) {
	t.Run("call without a client deadline gets the default one", func(t *testing.T) {
		start := time.Now()
		ctx := run(t, context.Background(), 10*time.Second)

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("handler context has no deadline")
		}
		if d := deadline.Sub(start); d < 9*time.Second || d > 11*time.Second {
			t.Fatalf("deadline in %v, want about 10s", d)
		}
	})

	t.Run("a client deadline is kept, even when longer than the default", func(t *testing.T) {
		want := time.Now().Add(time.Minute)
		parent, cancel := context.WithDeadline(context.Background(), want)
		defer cancel()

		deadline, _ := run(t, parent, 10*time.Second).Deadline()
		if !deadline.Equal(want) {
			t.Fatalf("deadline %v, want the client's %v", deadline, want)
		}
	})

	t.Run("the default deadline's context is released when the call returns", func(t *testing.T) {
		ctx := run(t, context.Background(), time.Hour)

		select {
		case <-ctx.Done():
		default:
			t.Fatal("context still live after the call returned (cancel not called)")
		}
	})

	t.Run("the handler sees the deadline expire", func(t *testing.T) {
		_, err := timeout.UnaryServerInterceptor(20*time.Millisecond)(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("got %v, want context.DeadlineExceeded", err)
		}
	})
}
