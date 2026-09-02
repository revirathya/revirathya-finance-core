package domain

import (
	"context"

	"github.com/google/uuid"
)

// Entity: User
type User struct {
	ID             uuid.UUID
	FirstName      string
	LastName       *string
	TelegramChatID *string
}

// Repository: User
type UserRepository interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
}
