package requestid_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/requestid"
	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
)

// call runs the interceptor with md as incoming metadata (nil: no metadata at all)
// and returns the request ID the handler saw.
func call(t *testing.T, md metadata.MD) string {
	t.Helper()
	ctx := context.Background()
	if md != nil {
		ctx = metadata.NewIncomingContext(ctx, md)
	}

	var seen string
	handler := func(ctx context.Context, req any) (any, error) {
		seen = logging.RequestIDFrom(ctx)
		return "ok", nil
	}

	resp, err := requestid.UnaryServerInterceptor()(ctx, "request", &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("response %v, want the handler's response", resp)
	}
	return seen
}

func TestUnaryServerInterceptor(t *testing.T) {
	t.Run("generates an ID without metadata", func(t *testing.T) {
		if _, err := uuid.Parse(call(t, nil)); err != nil {
			t.Fatal("want a generated UUID when there is no metadata")
		}
	})

	t.Run("generates an ID when the key is missing", func(t *testing.T) {
		if _, err := uuid.Parse(call(t, metadata.Pairs("authorization", "Bearer x"))); err != nil {
			t.Fatal("want a generated UUID when x-request-id is missing")
		}
	})

	t.Run("keeps a valid incoming ID", func(t *testing.T) {
		if got := call(t, metadata.Pairs("x-request-id", "bot-update-981")); got != "bot-update-981" {
			t.Fatalf("got %q, want %q", got, "bot-update-981")
		}
	})

	t.Run("replaces an invalid incoming ID", func(t *testing.T) {
		got := call(t, metadata.Pairs("x-request-id", "abc\nlevel=ERROR"))
		if _, err := uuid.Parse(got); err != nil {
			t.Fatalf("got %q, want a generated UUID", got)
		}
	})

	t.Run("replaces a repeated ID", func(t *testing.T) {
		got := call(t, metadata.Pairs("x-request-id", "one", "x-request-id", "two"))
		if got == "one" || got == "two" {
			t.Fatalf("got %q, want a new ID when the key is sent twice", got)
		}
	})

	t.Run("each call without an ID gets a different one", func(t *testing.T) {
		if a, b := call(t, nil), call(t, nil); a == b {
			t.Fatalf("two calls got the same ID %q", a)
		}
	})
}
