package auth

import (
	"context"

	"github.com/google/uuid"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

// Authenticated user for current request
type Principal struct {
	UserID uuid.UUID
}

type principalKey struct{}

// WithPrincipal returns a copy of ctx carrying p.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// WithTelegramUserID returns a copy of ctx carrying telegramID.
func WithTelegramUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, "telegramID", id)
}

// UserIDFrom returns authenticated user's ID, or errDom.ErrUnauthenticated (if missing principal)
func UserIDFrom(ctx context.Context) (uuid.UUID, error) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	if !ok {
		return uuid.Nil, errDom.ErrUnauthenticated
	}
	return p.UserID, nil
}

func TelegramUserIDFrom(ctx context.Context) (string, error) {
	telegramID, ok := ctx.Value("telegramID").(string)
	if !ok {
		return "", errDom.ErrUnauthenticated
	}
	return telegramID, nil
}
