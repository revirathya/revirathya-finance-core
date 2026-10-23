package auth_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/revirathya/revirathya-finance-core/internal/auth"
)

func TestPrincipalSlot(t *testing.T) {
	t.Run("WithPrincipal on a derived context fills the slot", func(t *testing.T) {
		ctx := auth.WithPrincipalSlot(context.Background())
		userID := uuid.New()
		_ = auth.WithPrincipal(ctx, auth.Principal{UserID: userID})

		if got, ok := auth.SlotUserID(ctx); !ok || got != userID {
			t.Fatalf("got %v, %v; want %s", got, ok, userID)
		}
	})

	t.Run("an empty slot reports nothing", func(t *testing.T) {
		if _, ok := auth.SlotUserID(auth.WithPrincipalSlot(context.Background())); ok {
			t.Fatal("ok = true, want false")
		}
	})

	t.Run("no slot reports nothing, and WithPrincipal still works", func(t *testing.T) {
		userID := uuid.New()
		ctx := auth.WithPrincipal(context.Background(), auth.Principal{UserID: userID})
		if _, ok := auth.SlotUserID(ctx); ok {
			t.Fatal("ok = true without a slot, want false")
		}
		if got, err := auth.UserIDFrom(ctx); err != nil || got != userID {
			t.Fatalf("UserIDFrom: %v, %v", got, err)
		}
	})
}
