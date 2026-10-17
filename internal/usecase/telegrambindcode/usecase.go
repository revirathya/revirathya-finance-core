package telegrambindcode

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
)

type TelegramBindCodeUsecase struct {
	tbcRepo domain.TelegramBindCodeRepository
}

func NewTelegramBindCodeUsecase(tbcRepo domain.TelegramBindCodeRepository) *TelegramBindCodeUsecase {
	return &TelegramBindCodeUsecase{
		tbcRepo: tbcRepo,
	}
}

func (u *TelegramBindCodeUsecase) CreateBindCode(ctx context.Context, userID uuid.UUID) (string, time.Time, error) {
	// Generate code
	code, err := auth.GenBindCode()
	if err != nil {
		return "", time.Time{}, err
	}

	// Prepare code
	codeHash := auth.HashCode(code)
	expiresAt := time.Now().Add(10 * time.Minute)

	// Save bind code
	if err := u.tbcRepo.CreateBindCode(ctx, userID, codeHash, expiresAt); err != nil {
		return "", time.Time{}, err
	}
	return code, expiresAt, nil
}

func (u *TelegramBindCodeUsecase) BindTelegram(ctx context.Context, code string, telegramID string) (*domain.User, error) {
	// Validate code pattern
	if matched := auth.ValidateCode(code); (!matched) || (telegramID == "") {
		return nil, domain.ErrInvalidInput
	}

	// Bind code
	codeHash := auth.HashCode(code)
	user, err := u.tbcRepo.BindCode(ctx, codeHash, telegramID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidInput
		}
		return nil, err
	}

	return user, nil
}

func (u *TelegramBindCodeUsecase) UnbindTelegram(ctx context.Context, userID uuid.UUID) error {
	return u.tbcRepo.UnbindTelegram(ctx, userID)
}
