package usercontext_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/usercontext"
)

// Fake
type fakeUserContextRepo struct {
	getResp    *domain.UserTelegramContext
	updateResp *domain.UserTelegramContext
	err        error

	gotGetUserID uuid.UUID

	gotUpdateUserID               uuid.UUID
	gotUpdateConversation         *string
	gotUpdateState                *string
	gotUpdateInteractionChatID    *string
	gotUpdateInteractionMessageID *string
}

func (f *fakeUserContextRepo) GetUserTelegramContext(ctx context.Context, userID uuid.UUID) (*domain.UserTelegramContext, error) {
	f.gotGetUserID = userID
	return f.getResp, f.err
}

func (f *fakeUserContextRepo) UpdateUserTelegramContext(ctx context.Context, userID uuid.UUID, conversation *string, state *string, interactionChatID *string, interactionMessageID *string) (*domain.UserTelegramContext, error) {
	f.gotUpdateUserID = userID
	f.gotUpdateConversation = conversation
	f.gotUpdateState = state
	f.gotUpdateInteractionChatID = interactionChatID
	f.gotUpdateInteractionMessageID = interactionMessageID
	return f.updateResp, f.err
}

// Tests

func TestGetUserTelegramContext(t *testing.T) {
	userID := uuid.New()
	state := "awaiting_amount"
	want := &domain.UserTelegramContext{UserID: userID, State: &state}

	t.Run("returns context", func(t *testing.T) {
		repo := &fakeUserContextRepo{getResp: want}
		uc := usercontext.NewUserTelegramContextUsecase(repo)

		got, err := uc.GetUserTelegramContext(context.Background(), userID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if repo.gotGetUserID != userID {
			t.Fatalf("repo received userID %v, want %v", repo.gotGetUserID, userID)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &fakeUserContextRepo{err: errors.New("not found")}
		uc := usercontext.NewUserTelegramContextUsecase(repo)

		if _, err := uc.GetUserTelegramContext(context.Background(), userID); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestUpdateUserTelegramContext(t *testing.T) {
	userID := uuid.New()
	state := "confirmed"
	want := &domain.UserTelegramContext{UserID: userID, State: &state}

	t.Run("updates and returns context", func(t *testing.T) {
		repo := &fakeUserContextRepo{updateResp: want}
		uc := usercontext.NewUserTelegramContextUsecase(repo)

		conversation := "adding_expense"
		state := "confirmed"
		chatID := "123"
		messageID := "456"
		got, err := uc.UpdateUserTelegramContext(context.Background(), userID, &conversation, &state, &chatID, &messageID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if repo.gotUpdateUserID != userID {
			t.Fatalf("repo received userID %v, want %v", repo.gotUpdateUserID, userID)
		}
		if *repo.gotUpdateConversation != "adding_expense" || *repo.gotUpdateState != "confirmed" {
			t.Fatalf("repo received wrong conversation/state: %v/%v", repo.gotUpdateConversation, repo.gotUpdateState)
		}
		if *repo.gotUpdateInteractionChatID != "123" || *repo.gotUpdateInteractionMessageID != "456" {
			t.Fatalf("repo received wrong interaction ids: %v/%v", repo.gotUpdateInteractionChatID, repo.gotUpdateInteractionMessageID)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &fakeUserContextRepo{err: errors.New("not found")}
		uc := usercontext.NewUserTelegramContextUsecase(repo)

		conversation := "adding_expense"
		state := "confirmed"
		chatID := "123"
		messageID := "456"
		if _, err := uc.UpdateUserTelegramContext(context.Background(), userID, &conversation, &state, &chatID, &messageID); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
