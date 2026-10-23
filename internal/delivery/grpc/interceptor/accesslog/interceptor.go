package accesslog

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"
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

		ctx = auth.WithPrincipalSlot(ctx)
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err)

		attrs := []any{
			"method", info.FullMethod,
			"code", code.String(),
			"duration_ms", time.Since(start).Milliseconds(),
		}

		if userID, ok := auth.SlotUserID(ctx); ok {
			attrs = append(attrs, "user_id", userID.String())
		}

		slog.Log(ctx, grpcLogLevel(code), "grpc request", attrs...)
		return resp, err
	}
}
