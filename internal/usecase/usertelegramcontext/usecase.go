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

func (uc *UserTelegramContextUsecase) UpdateUserTelegramContext(ctx context.Context, userID uuid.UUID, patch domain.UserTelegramContextPatch) (*domain.UserTelegramContext, error) {
	if err := patch.Validate(); err != nil {
		return nil, err
	}
	return uc.utelRepo.UpdateUserTelegramContext(ctx, userID, patch)
}
