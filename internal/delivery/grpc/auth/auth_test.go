package auth_test

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grpcAuth "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/auth"
)

func TestAPIKeyAuth(t *testing.T) {
	interceptor := grpcAuth.APIKeyAuth("secret")
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }

	tests := []struct {
		name     string
		md       metadata.MD
		wantCode codes.Code
	}{
		{"valid key", metadata.Pairs("authorization", "Bearer secret"), codes.OK},
		{"wrong key", metadata.Pairs("authorization", "Bearer nope"), codes.Unauthenticated},
		{"missing scheme", metadata.Pairs("authorization", "secret"), codes.Unauthenticated},
		{"no metadata", metadata.MD{}, codes.Unauthenticated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), tt.md)
			_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("got %v, want %v", got, tt.wantCode)
			}
		})
	}
}
