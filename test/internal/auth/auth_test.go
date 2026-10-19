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

func TestTelegramUserIDFrom(t *testing.T) {
	t.Run("returns the ID set by WithTelegramUserID", func(t *testing.T) {
		got, err := auth.TelegramUserIDFrom(auth.WithTelegramUserID(context.Background(), "123456789"))
		if err != nil || got != "123456789" {
			t.Fatalf("got %q, %v; want 123456789", got, err)
		}
	})

	t.Run("missing ID is unauthenticated", func(t *testing.T) {
		if _, err := auth.TelegramUserIDFrom(context.Background()); !errors.Is(err, errDom.ErrUnauthenticated) {
			t.Fatalf("got %v, want errDom.ErrUnauthenticated", err)
		}
	})

	t.Run("a plain string key from another package is not mistaken for the ID", func(t *testing.T) {
		//nolint:staticcheck // SA1029: deliberately the colliding key this test guards against
		ctx := context.WithValue(context.Background(), "telegramID", "999")
		if _, err := auth.TelegramUserIDFrom(ctx); !errors.Is(err, errDom.ErrUnauthenticated) {
			t.Fatalf("got %v, want errDom.ErrUnauthenticated", err)
		}
	})
}
