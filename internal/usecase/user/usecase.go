package user

import (
	"context"

	"github.com/google/uuid"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
)

// Usecase: User
type UserUsecase struct {
	user domain.UserRepository
}

func NewUserUsecase(usrRepo domain.UserRepository) *UserUsecase {
	return &UserUsecase{
		user: usrRepo,
	}
}

func (uc *UserUsecase) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return uc.user.GetUserByID(ctx, id)
}

func (uc *UserUsecase) UpdateUser(ctx context.Context, id uuid.UUID, patch domain.UserPatch) (*domain.User, error) {
	if err := patch.Validate(); err != nil {
		return nil, err
	}
	return uc.user.UpdateUser(ctx, id, patch)
}
