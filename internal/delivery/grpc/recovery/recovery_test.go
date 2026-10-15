package recovery_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/recovery"
)

const testMethod = "/financecore.v1.ExpenseService/ListExpenses"

// captureLogs redirects the default slog logger into a buffer for the rest of the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

type record struct{ field string }

// nilRecord returns a nil pointer, so dereferencing it causes a real runtime panic.
func nilRecord() *record { return nil }

func call(handler grpc.UnaryHandler) (any, error) {
	interceptor := recovery.UnaryServerInterceptor()
	return interceptor(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: testMethod}, handler)
}

func TestUnaryServerInterceptorRecoversPanics(t *testing.T) {
	tests := []struct {
		name    string
		handler grpc.UnaryHandler
		inLog   string
	}{
		{
			name:    "panic with a string",
			handler: func(ctx context.Context, req any) (any, error) { panic("boom") },
			inLog:   "boom",
		},
		{
			name:    "panic with an error",
			handler: func(ctx context.Context, req any) (any, error) { panic(errors.New("db exploded")) },
			inLog:   "db exploded",
		},
		{
			name: "runtime panic (nil pointer dereference)",
			handler: func(ctx context.Context, req any) (any, error) {
				return nilRecord().field, nil
			},
			inLog: "nil pointer dereference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := captureLogs(t)

			resp, err := call(tt.handler)

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("got non-status error %v, want a gRPC status", err)
			}
			if st.Code() != codes.Internal {
				t.Fatalf("code: got %v, want %v", st.Code(), codes.Internal)
			}
			if st.Message() != "internal error" {
				t.Fatalf("message: got %q, want %q (the panic value must not reach the client)", st.Message(), "internal error")
			}
			if resp != nil {
				t.Fatalf("response: got %v, want nil", resp)
			}

			logged := logs.String()
			for _, want := range []string{"grpc handler panicked", testMethod, tt.inLog, "stack="} {
				if !strings.Contains(logged, want) {
					t.Fatalf("log is missing %q:\n%s", want, logged)
				}
			}
		})
	}
}

func TestUnaryServerInterceptorPassesThrough(t *testing.T) {
	t.Run("successful response is returned unchanged", func(t *testing.T) {
		logs := captureLogs(t)

		resp, err := call(func(ctx context.Context, req any) (any, error) { return "ok", nil })

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp != "ok" {
			t.Fatalf("response: got %v, want %q", resp, "ok")
		}
		if logs.Len() != 0 {
			t.Fatalf("nothing should be logged without a panic, got:\n%s", logs.String())
		}
	})

	t.Run("handler error is returned unchanged", func(t *testing.T) {
		want := status.Error(codes.NotFound, "get expense 1: not found")

		_, err := call(func(ctx context.Context, req any) (any, error) { return nil, want })

		if err != want {
			t.Fatalf("error: got %v, want %v", err, want)
		}
	})

	t.Run("request and context reach the handler", func(t *testing.T) {
		type ctxKey struct{}
		interceptor := recovery.UnaryServerInterceptor()
		ctx := context.WithValue(context.Background(), ctxKey{}, "value")

		var gotReq, gotValue any
		_, err := interceptor(ctx, "request", &grpc.UnaryServerInfo{FullMethod: testMethod}, func(ctx context.Context, req any) (any, error) {
			gotReq, gotValue = req, ctx.Value(ctxKey{})
			return nil, nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotReq != "request" || gotValue != "value" {
			t.Fatalf("handler got req %v and ctx value %v, want %q and %q", gotReq, gotValue, "request", "value")
		}
	})
}

func TestUnaryServerInterceptorCoversLaterInterceptors(t *testing.T) {
	// Recovery is first in the chain, so a panic in a later interceptor (e.g. auth)
	// must be caught too. Simulate the chain: recovery -> panicking interceptor -> handler.
	captureLogs(t)
	panicking := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		panic("auth interceptor bug")
	}
	info := &grpc.UnaryServerInfo{FullMethod: testMethod}
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }

	_, err := recovery.UnaryServerInterceptor()(context.Background(), "request", info, func(ctx context.Context, req any) (any, error) {
		return panicking(ctx, req, info, handler)
	})

	if status.Code(err) != codes.Internal {
		t.Fatalf("code: got %v, want %v", status.Code(err), codes.Internal)
	}
}
