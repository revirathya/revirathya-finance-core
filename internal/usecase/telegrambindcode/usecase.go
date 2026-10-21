package telegrambindcode

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

type TelegramBindCodeUsecase struct {
	telegramBindCode domain.TelegramBindCodeRepository
}

func NewTelegramBindCodeUsecase(telegramBindCode domain.TelegramBindCodeRepository) *TelegramBindCodeUsecase {
	return &TelegramBindCodeUsecase{
		telegramBindCode: telegramBindCode,
	}
}

func (uc *TelegramBindCodeUsecase) CreateBindCode(ctx context.Context, userID uuid.UUID) (string, time.Time, error) {
	// Generate code
	code, err := auth.GenBindCode()
	if err != nil {
		return "", time.Time{}, err
	}

	// Prepare code
	codeHash := auth.HashCode(code)
	expiresAt := time.Now().Add(10 * time.Minute)

	// Save bind code
	if err := uc.telegramBindCode.CreateBindCode(ctx, userID, codeHash, expiresAt); err != nil {
		return "", time.Time{}, err
	}
	return code, expiresAt, nil
}

func (uc *TelegramBindCodeUsecase) BindTelegram(ctx context.Context, code string, telegramID string) (*domain.User, error) {
	// Validate code pattern
	if matched := auth.ValidateCode(code); (!matched) || (telegramID == "") {
		return nil, errDom.ErrInvalidInput
	}

	// Bind code
	codeHash := auth.HashCode(code)
	user, err := uc.telegramBindCode.BindCode(ctx, codeHash, telegramID)
	if err != nil {
		if errors.Is(err, errDom.ErrNotFound) {
			return nil, errDom.ErrInvalidInput
		}
		return nil, err
	}

	return user, nil
}

func (uc *TelegramBindCodeUsecase) UnbindTelegram(ctx context.Context, userID uuid.UUID) error {
	return uc.telegramBindCode.UnbindTelegram(ctx, userID)
}
