package usercontext

import (
	"context"

	"github.com/google/uuid"

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

func (uc *UserTelegramContextUsecase) UpdateUserTelegramContext(ctx context.Context, userID uuid.UUID, conversation *string, state *string, interactionChatID *string, interactionMessageID *string) (*domain.UserTelegramContext, error) {
	return uc.utelRepo.UpdateUserTelegramContext(
		ctx,
		userID,
		conversation,
		state,
		interactionChatID,
		interactionMessageID,
	)
}
