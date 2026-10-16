package requestid

import (
	"context"
	"strings"

	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	key := strings.ToLower(logging.RequestIDHeader)

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		var incoming string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get(key); len(values) == 1 {
				incoming = values[0]
			}
		}

		id := logging.EnsureRequestID(incoming)
		_ = grpc.SetHeader(ctx, metadata.Pairs(key, id))
		return handler(logging.WithRequestID(ctx, id), req)
	}
}
