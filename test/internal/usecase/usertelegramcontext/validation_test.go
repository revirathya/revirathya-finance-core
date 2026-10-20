package usertelegramcontext_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/usertelegramcontext"
)

func TestUserTelegramContextValidation(t *testing.T) {
	t.Run("rejects a too long field before the repository", func(t *testing.T) {
		repo := &fakeUserContextRepo{}
		uc := usertelegramcontext.NewUserTelegramContextUsecase(repo)
		state := strings.Repeat("a", 101)

		_, err := uc.UpdateUserTelegramContext(context.Background(), uuid.New(), domain.UserTelegramContextPatch{State: &state})

		if !errors.Is(err, errDom.ErrInvalidInput) {
			t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
		}
		if repo.gotUpdateUserID != uuid.Nil {
			t.Fatal("repository was called")
		}
	})

	t.Run("passes trimmed values to the repository and leaves nil fields nil", func(t *testing.T) {
		repo := &fakeUserContextRepo{}
		uc := usertelegramcontext.NewUserTelegramContextUsecase(repo)
		state, chatID := " ask_amount ", " 123 "

		if _, err := uc.UpdateUserTelegramContext(context.Background(), uuid.New(), domain.UserTelegramContextPatch{State: &state, InteractionChatID: &chatID}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *repo.gotUpdateState != "ask_amount" || *repo.gotUpdateInteractionChatID != "123" {
			t.Fatalf("repository received %q %q, want trimmed values", *repo.gotUpdateState, *repo.gotUpdateInteractionChatID)
		}
		if repo.gotUpdateConversation != nil || repo.gotUpdateInteractionMessageID != nil {
			t.Fatal("nil fields reached the repository as non-nil")
		}
	})
}
