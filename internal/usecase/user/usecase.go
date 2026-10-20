package user

import (
	"context"

	"github.com/google/uuid"

	// Internal
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

func (uc *UserUsecase) GetUserByGoogleSub(ctx context.Context, googleSub string) (*domain.User, error) {
	return uc.usrRepo.GetUserByGoogleSub(ctx, googleSub)
}

func (uc *UserUsecase) CreateUser(ctx context.Context, firstName string, lastName *string, email string, googleSub string) (*domain.User, error) {
	form := domain.UserForm{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		GoogleSub: googleSub,
	}
	if err := form.Validate(); err != nil {
		return nil, err
	}

	return uc.usrRepo.CreateUser(ctx, form)
}

func (uc *UserUsecase) UpdateUser(ctx context.Context, id uuid.UUID, patch domain.UserPatch) (*domain.User, error) {
	if err := patch.Validate(); err != nil {
		return nil, err
	}
	return uc.usrRepo.UpdateUser(ctx, id, patch)
}
