package auth

import (
	"context"
	"sync/atomic"

	"github.com/google/uuid"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

// Authenticated user for current request
type Principal struct {
	UserID uuid.UUID
}

type principalKey struct{}

type principalSlotKey struct{}

type telegramIDKey struct{}

// WithPrincipal returns a copy of ctx carrying p.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	if slot, ok := ctx.Value(principalSlotKey{}).(*atomic.Pointer[Principal]); ok {
		slot.Store(&p)
	}
	return context.WithValue(ctx, principalKey{}, p)
}

// WithPrincipalSlot returns a copy of ctx with an empty slot that WithPrincipal fills, so an outer middleware (the
// access log) can read the principal that an inner one (auth) sets on its own, derived context.
func WithPrincipalSlot(ctx context.Context) context.Context {
	return context.WithValue(ctx, principalSlotKey{}, new(atomic.Pointer[Principal]))
}

// SlotUserID returns the user ID that WithPrincipal recorded in ctx's slot, if any.
func SlotUserID(ctx context.Context) (uuid.UUID, bool) {
	slot, ok := ctx.Value(principalSlotKey{}).(*atomic.Pointer[Principal])
	if !ok {
		return uuid.Nil, false
	}

	p := slot.Load()
	if p == nil {
		return uuid.Nil, false
	}

	return p.UserID, true
}

// WithTelegramUserID returns a copy of ctx carrying telegramID.
func WithTelegramUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, telegramIDKey{}, id)
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
	telegramID, ok := ctx.Value(telegramIDKey{}).(string)
	if !ok {
		return "", errDom.ErrUnauthenticated
	}
	return telegramID, nil
}
