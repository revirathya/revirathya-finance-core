package timeout

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// UnaryServerInterceptor gives calls without a client deadline a default one.
// A client deadline is kept as is, even if it's longer than d.
func UnaryServerInterceptor(d time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, d)
			defer cancel()
		}
		return handler(ctx, req)
	}
}
