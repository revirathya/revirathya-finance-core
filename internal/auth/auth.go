package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
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

// UserIDFrom returns authenticated user's ID, or domain.ErrUnauthenticated (if missing principal)
func UserIDFrom(ctx context.Context) (uuid.UUID, error) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	if !ok {
		return uuid.Nil, domain.ErrUnauthenticated
	}
	return p.UserID, nil
}
