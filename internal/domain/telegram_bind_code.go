package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository
type TelegramBindCodeRepository interface {
	CreateBindCode(ctx context.Context, userID uuid.UUID, codeHash string, expiresAt time.Time) error
	BindCode(ctx context.Context, codeHash string, telegramID string) (*User, error)
	UnbindTelegram(ctx context.Context, userID uuid.UUID) error
}
