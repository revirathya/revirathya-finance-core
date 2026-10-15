package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"

	"github.com/google/uuid"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	appAuth "github.com/revirathya/revirathya-finance-core/internal/auth"
	grpcErr "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
)

const telegramUserIDKey = "x-telegram-user-id"

func APIKeyAuth(apiKey string) grpc.UnaryServerInterceptor {

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

type TelegramUserLookup func(ctx context.Context, telegramID string) (uuid.UUID, error)

func TelegramUser(lookup TelegramUserLookup, unboundMethods ...string) grpc.UnaryServerInterceptor {
	allowUnbound := make(map[string]bool, len(unboundMethods))
	for _, m := range unboundMethods {
		allowUnbound[m] = true
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Bypass healthcheck
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		// Read and validate Telegram user ID
		md, _ := metadata.FromIncomingContext(ctx)
		values := md.Get(telegramUserIDKey)
		if len(values) != 1 || !isTelegramID(values[0]) {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid telegram user")
		}
		telegramID := values[0]

		// Store to context
		ctx = appAuth.WithTelegramUserID(ctx, telegramID)

		// Find bounded users
		userID, err := lookup(ctx, telegramID)
		switch {
		case err == nil:
			ctx = appAuth.WithPrincipal(ctx, appAuth.Principal{UserID: userID})
		case errors.Is(err, domain.ErrNotFound) && allowUnbound[info.FullMethod]:
			// Not bounded
		case errors.Is(err, domain.ErrNotFound):
			return nil, status.Error(codes.Unauthenticated, "telegram accoount not bound")
		default:
			return nil, grpcErr.ToStatus("look up telegram user", err)
		}

		// Continue with enriched context
		return handler(ctx, req)
	}
}

func isTelegramID(s string) bool {
	if s == "" || len(s) > 20 || s[0] == '0' {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
