package accesslog

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func grpcLogLevel(code codes.Code) slog.Level {
	switch code {
	case codes.OK:
		return slog.LevelInfo
	case codes.Unknown, codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unimplemented:
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err)

		slog.Log(
			ctx, grpcLogLevel(code), "grpc request",
			"method", info.FullMethod,
			"code", code.String(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return resp, err
	}
}
