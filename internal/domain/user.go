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
	Email          string
	TelegramChatID *string
}

type UserForm struct {
	FirstName string
	LastName  *string
	Email     string
	GoogleSub string
}

// Repository: User
type UserRepository interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByGoogleSub(ctx context.Context, googleSub string) (*User, error)
	CreateUser(ctx context.Context, form UserForm) (*User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, firstName string, lastName *string) (*User, error)
}
