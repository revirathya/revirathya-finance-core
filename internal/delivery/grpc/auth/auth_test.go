package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	appAuth "github.com/revirathya/revirathya-finance-core/internal/auth"
	grpcAuth "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/auth"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
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

func TestAPIKeyAuthHealthcheck(t *testing.T) {
	interceptor := grpcAuth.APIKeyAuth("secret")
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }

	t.Run("healthcheck", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
		_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/"}, handler)
		if got := status.Code(err); got != codes.OK {
			t.Fatalf("got %v, want %v", got, codes.OK)
		}
	})
}

func TestTelegramUser(t *testing.T) {
	const (
		bindMethod  = "/financecore.v1.UserTelegramContextService/BindTelegramAccount"
		otherMethod = "/financecore.v1.ExpenseService/ListExpenses"
	)
	userID := uuid.New()

	// "111" is bound, "999" makes the lookup fail, every other ID is unbound.
	lookup := func(ctx context.Context, telegramID string) (uuid.UUID, error) {
		switch telegramID {
		case "111":
			return userID, nil
		case "999":
			return uuid.Nil, errors.New("db down")
		default:
			return uuid.Nil, domain.ErrNotFound
		}
	}
	interceptor := grpcAuth.TelegramUser(lookup, bindMethod)

	tests := []struct {
		name           string
		md             metadata.MD
		method         string
		wantCode       codes.Code
		wantPrincipal  uuid.UUID // uuid.Nil: no principal expected
		wantTelegramID string    // only checked when the handler runs
	}{
		{"no metadata", metadata.MD{}, otherMethod, codes.Unauthenticated, uuid.Nil, ""},
		{"empty ID", metadata.Pairs("x-telegram-user-id", ""), otherMethod, codes.Unauthenticated, uuid.Nil, ""},
		{"letters", metadata.Pairs("x-telegram-user-id", "abc"), otherMethod, codes.Unauthenticated, uuid.Nil, ""},
		{"negative", metadata.Pairs("x-telegram-user-id", "-5"), otherMethod, codes.Unauthenticated, uuid.Nil, ""},
		{"leading zero", metadata.Pairs("x-telegram-user-id", "0123"), otherMethod, codes.Unauthenticated, uuid.Nil, ""},
		{"too long", metadata.Pairs("x-telegram-user-id", "123456789012345678901"), otherMethod, codes.Unauthenticated, uuid.Nil, ""},
		{"repeated header", metadata.Pairs("x-telegram-user-id", "111", "x-telegram-user-id", "222"), otherMethod, codes.Unauthenticated, uuid.Nil, ""},
		{"bound user, any method", metadata.Pairs("x-telegram-user-id", "111"), otherMethod, codes.OK, userID, "111"},
		{"bound user, bind method", metadata.Pairs("x-telegram-user-id", "111"), bindMethod, codes.OK, userID, "111"},
		{"unbound user, bind method", metadata.Pairs("x-telegram-user-id", "222"), bindMethod, codes.OK, uuid.Nil, "222"},
		{"unbound user, other method", metadata.Pairs("x-telegram-user-id", "222"), otherMethod, codes.Unauthenticated, uuid.Nil, ""},
		{"lookup fails", metadata.Pairs("x-telegram-user-id", "999"), otherMethod, codes.Internal, uuid.Nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				called        bool
				gotPrincipal  uuid.UUID
				principalErr  error
				gotTelegramID string
			)
			handler := func(ctx context.Context, req any) (any, error) {
				called = true
				gotPrincipal, principalErr = appAuth.UserIDFrom(ctx)
				gotTelegramID, _ = appAuth.TelegramUserIDFrom(ctx)
				return "ok", nil
			}

			ctx := metadata.NewIncomingContext(context.Background(), tt.md)
			_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: tt.method}, handler)

			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("got code %v, want %v (err: %v)", got, tt.wantCode, err)
			}
			if tt.wantCode != codes.OK {
				if called {
					t.Fatal("handler was called for a rejected request")
				}
				return
			}
			if !called {
				t.Fatal("handler was not called")
			}
			if gotTelegramID != tt.wantTelegramID {
				t.Fatalf("telegram ID in context: got %q, want %q", gotTelegramID, tt.wantTelegramID)
			}
			if tt.wantPrincipal == uuid.Nil {
				if !errors.Is(principalErr, domain.ErrUnauthenticated) {
					t.Fatalf("got principal %v, want none", gotPrincipal)
				}
				return
			}
			if principalErr != nil || gotPrincipal != tt.wantPrincipal {
				t.Fatalf("principal: got %v (err %v), want %v", gotPrincipal, principalErr, tt.wantPrincipal)
			}
		})
	}
}

func TestTelegramUserHealthcheck(t *testing.T) {
	lookup := func(ctx context.Context, telegramID string) (uuid.UUID, error) {
		return uuid.Nil, nil
	}
	const bindMethod = "/financecore.v1.UserTelegramContextService/BindTelegramAccount"
	interceptor := grpcAuth.TelegramUser(lookup, bindMethod)
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }

	t.Run("healthcheck", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
		_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/"}, handler)
		if got := status.Code(err); got != codes.OK {
			t.Fatalf("got %v, want %v", got, codes.OK)
		}
	})
}
