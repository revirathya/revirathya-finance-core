package domain

import (
	"context"

	"github.com/google/uuid"
)

// Entity: User Telegram Context
type UserTelegramContext struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	TelegramChatID       string
	Conversation         *string
	State                *string
	InteractionChatID    *string
	InteractionMessageID *string
}

// Repository: User Telegram Context
type UserTelegramContextRepository interface {
	GetUserTelegramContext(ctx context.Context, userID uuid.UUID) (*UserTelegramContext, error)
	GetUserTelegramContextByTelegramID(ctx context.Context, telegramChatID string) (*UserTelegramContext, error)
	UpdateUserTelegramContext(ctx context.Context, userID uuid.UUID, conversation *string, state *string, interactionChatID *string, interactionMessageID *string) (*UserTelegramContext, error)
}
