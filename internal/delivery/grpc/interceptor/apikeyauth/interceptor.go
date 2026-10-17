package apikeyauth

import (
	"context"
	"crypto/subtle"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor(apiKey string) grpc.UnaryServerInterceptor {
	expected := []byte("Bearer " + apiKey)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Bypass healthcheck
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		md, _ := metadata.FromIncomingContext(ctx)
		values := md.Get("authorization")
		if len(values) != 1 || subtle.ConstantTimeCompare([]byte(values[0]), expected) != 1 {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid API key")
		}
		return handler(ctx, req)
	}
}
