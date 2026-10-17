//go:build integration

package telegrambindcode_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"

	// Repository
	rTelegramBindCode "github.com/revirathya/revirathya-finance-core/internal/repository/postgres/telegrambindcode"
	rUserTelegramContext "github.com/revirathya/revirathya-finance-core/internal/repository/postgres/usertelegramcontext"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithPostgres(m))
}

func TestTelegramBindCodeRepository(t *testing.T) {
	ctx := context.Background()
	repo := rTelegramBindCode.NewTelegramBindCodeRepositoryImpl(testutil.TestPool)
	contexts := rUserTelegramContext.NewUserTelegramContextRepositoryImpl(testutil.TestPool)
	inTenMinutes := func() time.Time { return time.Now().Add(10 * time.Minute) }

	t.Run("binds a valid code", func(t *testing.T) {
		alice, tg, hash := testutil.NewUser(t), testutil.NewTelegramID(), testutil.NewCodeHash()
		if err := repo.CreateBindCode(ctx, alice.ID, hash, inTenMinutes()); err != nil {
			t.Fatalf("create code: %v", err)
		}

		user, err := repo.BindCode(ctx, hash, tg)

		if err != nil {
			t.Fatalf("bind: %v", err)
		}
		if user.ID != alice.ID {
			t.Fatalf("bound user %v, want %v", user.ID, alice.ID)
		}
		if user.TelegramChatID == nil || *user.TelegramChatID != tg {
			t.Fatalf("telegram ID %v, want %q", user.TelegramChatID, tg)
		}
	})

	t.Run("a code can only be used once", func(t *testing.T) {
		alice, hash := testutil.NewUser(t), testutil.NewCodeHash()
		if err := repo.CreateBindCode(ctx, alice.ID, hash, inTenMinutes()); err != nil {
			t.Fatalf("create code: %v", err)
		}
		if _, err := repo.BindCode(ctx, hash, testutil.NewTelegramID()); err != nil {
			t.Fatalf("first bind: %v", err)
		}

		if _, err := repo.BindCode(ctx, hash, testutil.NewTelegramID()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("second bind: got %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("an expired code is rejected", func(t *testing.T) {
		alice, hash := testutil.NewUser(t), testutil.NewCodeHash()
		if err := repo.CreateBindCode(ctx, alice.ID, hash, time.Now().Add(-time.Minute)); err != nil {
			t.Fatalf("create code: %v", err)
		}

		if _, err := repo.BindCode(ctx, hash, testutil.NewTelegramID()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("got %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("an unknown code is rejected", func(t *testing.T) {
		if _, err := repo.BindCode(ctx, testutil.NewCodeHash(), testutil.NewTelegramID()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("got %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("a new code replaces the previous one", func(t *testing.T) {
		alice, oldHash, newHash := testutil.NewUser(t), testutil.NewCodeHash(), testutil.NewCodeHash()
		if err := repo.CreateBindCode(ctx, alice.ID, oldHash, inTenMinutes()); err != nil {
			t.Fatalf("create old code: %v", err)
		}
		if err := repo.CreateBindCode(ctx, alice.ID, newHash, inTenMinutes()); err != nil {
			t.Fatalf("create new code: %v", err)
		}

		if _, err := repo.BindCode(ctx, oldHash, testutil.NewTelegramID()); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("old code: got %v, want domain.ErrNotFound", err)
		}
		if _, err := repo.BindCode(ctx, newHash, testutil.NewTelegramID()); err != nil {
			t.Fatalf("new code: %v", err)
		}
	})

	t.Run("telegram account bound to another user is a conflict, and the code stays valid", func(t *testing.T) {
		alice, bob, tg := testutil.NewUser(t), testutil.NewUser(t), testutil.NewTelegramID()
		aliceHash, bobHash := testutil.NewCodeHash(), testutil.NewCodeHash()
		if err := repo.CreateBindCode(ctx, alice.ID, aliceHash, inTenMinutes()); err != nil {
			t.Fatalf("create alice code: %v", err)
		}
		if _, err := repo.BindCode(ctx, aliceHash, tg); err != nil {
			t.Fatalf("bind alice: %v", err)
		}
		if err := repo.CreateBindCode(ctx, bob.ID, bobHash, inTenMinutes()); err != nil {
			t.Fatalf("create bob code: %v", err)
		}

		if _, err := repo.BindCode(ctx, bobHash, tg); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("bind bob to alice's telegram: got %v, want domain.ErrConflict", err)
		}

		// The failed bind must have rolled back, so bob's code is still unused.
		if err := repo.UnbindTelegram(ctx, alice.ID); err != nil {
			t.Fatalf("unbind alice: %v", err)
		}
		user, err := repo.BindCode(ctx, bobHash, tg)
		if err != nil {
			t.Fatalf("bind bob after alice unbound: got %v, want success (code must still be valid)", err)
		}
		if user.ID != bob.ID {
			t.Fatalf("bound user %v, want %v", user.ID, bob.ID)
		}
	})

	t.Run("re-binding to a new telegram account resets conversation state", func(t *testing.T) {
		alice, firstTG, secondTG := testutil.NewUser(t), testutil.NewTelegramID(), testutil.NewTelegramID()
		firstHash, secondHash := testutil.NewCodeHash(), testutil.NewCodeHash()
		if err := repo.CreateBindCode(ctx, alice.ID, firstHash, inTenMinutes()); err != nil {
			t.Fatalf("create first code: %v", err)
		}
		if _, err := repo.BindCode(ctx, firstHash, firstTG); err != nil {
			t.Fatalf("first bind: %v", err)
		}
		if _, err := contexts.UpdateUserTelegramContext(ctx, alice.ID, testutil.Ptr("add_expense"), testutil.Ptr("ask_amount"), testutil.Ptr("1"), testutil.Ptr("2")); err != nil {
			t.Fatalf("set conversation: %v", err)
		}
		if err := repo.CreateBindCode(ctx, alice.ID, secondHash, inTenMinutes()); err != nil {
			t.Fatalf("create second code: %v", err)
		}

		if _, err := repo.BindCode(ctx, secondHash, secondTG); err != nil {
			t.Fatalf("second bind: %v", err)
		}

		got, err := contexts.GetUserTelegramContext(ctx, alice.ID)
		if err != nil {
			t.Fatalf("get context: %v", err)
		}
		if got.TelegramChatID == nil || *got.TelegramChatID != secondTG {
			t.Fatalf("telegram ID %v, want %q", got.TelegramChatID, secondTG)
		}
		if got.Conversation != nil || got.State != nil || got.InteractionChatID != nil || got.InteractionMessageID != nil {
			t.Fatalf("conversation state not reset: %+v", got)
		}
	})

	t.Run("unbinding clears the binding and conversation state", func(t *testing.T) {
		alice, tg, hash := testutil.NewUser(t), testutil.NewTelegramID(), testutil.NewCodeHash()
		if err := repo.CreateBindCode(ctx, alice.ID, hash, inTenMinutes()); err != nil {
			t.Fatalf("create code: %v", err)
		}
		if _, err := repo.BindCode(ctx, hash, tg); err != nil {
			t.Fatalf("bind: %v", err)
		}
		if _, err := contexts.UpdateUserTelegramContext(ctx, alice.ID, testutil.Ptr("add_expense"), testutil.Ptr("ask_amount"), nil, nil); err != nil {
			t.Fatalf("set conversation: %v", err)
		}

		if err := repo.UnbindTelegram(ctx, alice.ID); err != nil {
			t.Fatalf("unbind: %v", err)
		}

		got, err := contexts.GetUserTelegramContext(ctx, alice.ID)
		if err != nil {
			t.Fatalf("get context: %v", err)
		}
		if got.TelegramChatID != nil || got.Conversation != nil || got.State != nil {
			t.Fatalf("binding not cleared: %+v", got)
		}
		if _, err := contexts.GetUserTelegramContextByTelegramID(ctx, tg); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("lookup by old telegram ID: got %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("unbinding a never-bound user succeeds", func(t *testing.T) {
		if err := repo.UnbindTelegram(ctx, testutil.NewUser(t).ID); err != nil {
			t.Fatalf("unbind: %v", err)
		}
	})
}

func TestUserTelegramContextRepository(t *testing.T) {
	ctx := context.Background()
	codes := rTelegramBindCode.NewTelegramBindCodeRepositoryImpl(testutil.TestPool)
	repo := rUserTelegramContext.NewUserTelegramContextRepositoryImpl(testutil.TestPool)

	bind := func(t *testing.T) (*domain.User, string) {
		t.Helper()
		user, tg, hash := testutil.NewUser(t), testutil.NewTelegramID(), testutil.NewCodeHash()
		if err := codes.CreateBindCode(ctx, user.ID, hash, time.Now().Add(10*time.Minute)); err != nil {
			t.Fatalf("create code: %v", err)
		}
		if _, err := codes.BindCode(ctx, hash, tg); err != nil {
			t.Fatalf("bind: %v", err)
		}
		return user, tg
	}

	t.Run("a user without a binding has no context", func(t *testing.T) {
		if _, err := repo.GetUserTelegramContext(ctx, testutil.NewUser(t).ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("got %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("lookup by telegram ID finds the bound user", func(t *testing.T) {
		alice, tg := bind(t)

		got, err := repo.GetUserTelegramContextByTelegramID(ctx, tg)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.UserID != alice.ID {
			t.Fatalf("user %v, want %v", got.UserID, alice.ID)
		}
	})

	t.Run("update changes only the fields that are set", func(t *testing.T) {
		alice, _ := bind(t)
		if _, err := repo.UpdateUserTelegramContext(ctx, alice.ID, testutil.Ptr("add_expense"), testutil.Ptr("ask_amount"), nil, nil); err != nil {
			t.Fatalf("first update: %v", err)
		}

		got, err := repo.UpdateUserTelegramContext(ctx, alice.ID, nil, testutil.Ptr("ask_category"), nil, nil)

		if err != nil {
			t.Fatalf("second update: %v", err)
		}
		if got.Conversation == nil || *got.Conversation != "add_expense" {
			t.Fatalf("conversation %v, want unchanged %q", got.Conversation, "add_expense")
		}
		if got.State == nil || *got.State != "ask_category" {
			t.Fatalf("state %v, want %q", got.State, "ask_category")
		}
	})

	t.Run("updating a user without a context is not found", func(t *testing.T) {
		if _, err := repo.UpdateUserTelegramContext(ctx, testutil.NewUser(t).ID, testutil.Ptr("x"), nil, nil, nil); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("got %v, want domain.ErrNotFound", err)
		}
	})
}
