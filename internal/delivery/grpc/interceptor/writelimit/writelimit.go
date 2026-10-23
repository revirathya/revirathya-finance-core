package writelimit

import (
	"context"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/platform/ratelimit"

	// Internal - gRPC
	errGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"
)

const retryAfterKey = "retry-after"

// UnaryServerInterceptor counts each call to one of methods (full method names) against the caller's write budget.
// It must run after telegramuser, which sets the principal; calls without one aren't counted.
func UnaryServerInterceptor(limiter *ratelimit.Limiter, metricsColl *metrics.Metrics, methods ...string) grpc.UnaryServerInterceptor {
	counted := make(map[string]bool, len(methods))
	for _, m := range methods {
		counted[m] = true
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !counted[info.FullMethod] {
			return handler(ctx, req)
		}

		userID, err := auth.UserIDFrom(ctx)
		if err != nil {
			return handler(ctx, req)
		}

		if ok, retryAfter := limiter.Take(userID.String()); !ok {
			rlErr := &errDom.RateLimitError{RetryAfter: retryAfter}
			_ = grpc.SetHeader(ctx, metadata.Pairs(
				retryAfterKey, strconv.Itoa(rlErr.RetryAfterSeconds()),
			))
			metricsColl.RateLimited("grpc")
			return nil, errGrpc.ToStatus(ctx, info.FullMethod, rlErr)
		}
		return handler(ctx, req)
	}
}
