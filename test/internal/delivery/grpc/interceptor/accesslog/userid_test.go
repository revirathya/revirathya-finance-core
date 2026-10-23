package accesslog_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"

	// Internal - gRPC
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/interceptor/accesslog"
)

func TestUnaryServerInterceptorUserID(t *testing.T) {
	// callAs runs the interceptor with a handler that sets the principal on its own context, the way telegramuser
	// does further down the chain, and returns the access log line.
	callAs := func(t *testing.T, userID *uuid.UUID) map[string]any {
		t.Helper()
		logs := captureLogs(t)
		handler := func(ctx context.Context, req any) (any, error) {
			if userID != nil {
				ctx = auth.WithPrincipal(ctx, auth.Principal{UserID: *userID})
			}
			_ = ctx
			return "response", nil
		}
		if _, err := accesslog.UnaryServerInterceptor()(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: testMethod}, handler); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lines := logLines(t, logs)
		if len(lines) != 1 {
			t.Fatalf("%d log lines, want 1", len(lines))
		}
		return lines[0]
	}

	t.Run("a principal set by an inner interceptor is logged as user_id", func(t *testing.T) {
		userID := uuid.New()
		if got := callAs(t, &userID)["user_id"]; got != userID.String() {
			t.Fatalf("user_id %v, want %s", got, userID)
		}
	})

	t.Run("an anonymous call has no user_id", func(t *testing.T) {
		if got, ok := callAs(t, nil)["user_id"]; ok {
			t.Fatalf("user_id %v, want none", got)
		}
	})
}
