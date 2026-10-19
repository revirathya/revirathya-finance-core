package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/revirathya/revirathya-finance-core/internal/auth"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

func TestUserIDFrom(t *testing.T) {
	t.Run("returns the principal's user ID", func(t *testing.T) {
		userID := uuid.New()
		ctx := auth.WithPrincipal(context.Background(), auth.Principal{UserID: userID})

		got, err := auth.UserIDFrom(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != userID {
			t.Fatalf("got %v, want %v", got, userID)
		}
	})

	t.Run("no principal is unauthenticated", func(t *testing.T) {
		_, err := auth.UserIDFrom(context.Background())
		if !errors.Is(err, errDom.ErrUnauthenticated) {
			t.Fatalf("got %v, want errDom.ErrUnauthenticated", err)
		}
	})
}
