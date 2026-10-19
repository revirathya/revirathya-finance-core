package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain/validation"
)

// Entity
type UserTelegramContext struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	TelegramChatID       *string
	Conversation         *string
	State                *string
	InteractionChatID    *string
	InteractionMessageID *string
}

type UserTelegramContextPatch struct {
	Conversation         *string
	State                *string
	InteractionChatID    *string
	InteractionMessageID *string
}

func (p *UserTelegramContextPatch) Validate() error {
	// Conversation
	if p.Conversation != nil {
		description, err := validation.ValidateString("conversation", *p.Conversation, 100, false)
		if err != nil {
			return err
		}
		p.Conversation = &description
	}

	// State
	if p.State != nil {
		state, err := validation.ValidateString("state", *p.State, 100, false)
		if err != nil {
			return err
		}
		p.State = &state
	}

	// Interaction Chat ID
	if p.InteractionChatID != nil {
		interactionChatID, err := validation.ValidateString("interaction_chat_id", *p.InteractionChatID, 100, false)
		if err != nil {
			return err
		}
		p.InteractionChatID = &interactionChatID
	}

	// Interaction Message ID
	if p.InteractionMessageID != nil {
		interactionMessageID, err := validation.ValidateString("interaction_message_id", *p.InteractionMessageID, 100, false)
		if err != nil {
			return err
		}
		p.InteractionMessageID = &interactionMessageID
	}
	return nil
}

// Repository
type UserTelegramContextRepository interface {
	GetUserTelegramContext(ctx context.Context, userID uuid.UUID) (*UserTelegramContext, error)
	GetUserTelegramContextByTelegramID(ctx context.Context, telegramChatID string) (*UserTelegramContext, error)
	UpdateUserTelegramContext(ctx context.Context, userID uuid.UUID, patch UserTelegramContextPatch) (*UserTelegramContext, error)
}
