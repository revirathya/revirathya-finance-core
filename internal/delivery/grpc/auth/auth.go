package auth

import (
	"context"
	"crypto/subtle"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func APIKeyAuth(apiKey string) grpc.UnaryServerInterceptor {
	expected := []byte("Bearer " + apiKey)

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		values := md.Get("authorization")
		if len(values) != 1 || subtle.ConstantTimeCompare([]byte(values[0]), expected) != 1 {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid API key")
		}
		return handler(ctx, req)
	}
}
