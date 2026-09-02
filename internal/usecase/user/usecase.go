package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
)

// Usecase: User
type UserUsecase struct {
	usrRepo domain.UserRepository
}

func NewUserUsecase(usrRepo domain.UserRepository) *UserUsecase {
	return &UserUsecase{
		usrRepo: usrRepo,
	}
}

func (uc *UserUsecase) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return uc.usrRepo.GetUserByID(ctx, id)
}
