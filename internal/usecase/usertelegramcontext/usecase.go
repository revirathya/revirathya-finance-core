package usertelegramcontext

import (
	"context"

	"github.com/google/uuid"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
)

// Usecase: User Telegram Context
type UserTelegramContextUsecase struct {
	utelRepo domain.UserTelegramContextRepository
}

func NewUserTelegramContextUsecase(utelRepo domain.UserTelegramContextRepository) *UserTelegramContextUsecase {
	return &UserTelegramContextUsecase{
		utelRepo: utelRepo,
	}
}

func (uc *UserTelegramContextUsecase) GetUserTelegramContext(ctx context.Context, userID uuid.UUID) (*domain.UserTelegramContext, error) {
	return uc.utelRepo.GetUserTelegramContext(ctx, userID)
}

func (uc *UserTelegramContextUsecase) GetUserTelegramContextByTelegramID(ctx context.Context, telegramChatID string) (*domain.UserTelegramContext, error) {
	return uc.utelRepo.GetUserTelegramContextByTelegramID(ctx, telegramChatID)
}

func (uc *UserTelegramContextUsecase) UpdateUserTelegramContext(ctx context.Context, userID uuid.UUID, conversation *string, state *string, interactionChatID *string, interactionMessageID *string) (*domain.UserTelegramContext, error) {
	patch := domain.UserTelegramContextPatch{
		Conversation:         conversation,
		State:                state,
		InteractionChatID:    interactionChatID,
		InteractionMessageID: interactionMessageID,
	}
	if err := patch.Validate(); err != nil {
		return nil, err
	}

	return uc.utelRepo.UpdateUserTelegramContext(ctx, userID, patch)
}
