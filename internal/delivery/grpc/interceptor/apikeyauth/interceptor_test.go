package apikeyauth_test

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/interceptor/apikeyauth"
)

func TestAPIKeyAuth(t *testing.T) {
	interceptor := apikeyauth.UnaryServerInterceptor("secret")
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

func TestAPIKeyAuthHealthcheck(t *testing.T) {
	interceptor := apikeyauth.UnaryServerInterceptor("secret")
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }

	tests := []struct {
		name     string
		method   string
		wantCode codes.Code
	}{
		{"healthcheck", "/grpc.health.v1.Health/Check", codes.OK},
		{"non healthcheck", "/financecore.v1.ExpenseService/ListExpenses", codes.Unauthenticated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
			_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: tt.method}, handler)
			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("got %v, want %v", got, tt.wantCode)
			}
		})
	}
}
