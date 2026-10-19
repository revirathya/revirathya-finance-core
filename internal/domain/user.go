package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain/validation"
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

func (f *UserForm) Validate() error {
	// First Name
	firstName, err := validation.ValidateString("first_name", f.FirstName, 100, true)
	if err != nil {
		return err
	}
	f.FirstName = firstName

	// Last Name
	if f.LastName != nil {
		lastName, err := validation.ValidateString("last_name", *f.LastName, 100, false)
		if err != nil {
			return err
		}
		f.LastName = &lastName
	}

	// Email
	email, err := validation.ValidateString("email", f.Email, 100, true)
	if err != nil {
		return err
	}
	f.Email = email

	// GoogleSub
	googleSub, err := validation.ValidateString("google_sub", f.GoogleSub, 100, true)
	if err != nil {
		return err
	}
	f.GoogleSub = googleSub

	return nil
}

type UserPatch struct {
	FirstName string
	LastName  *string
}

func (p *UserPatch) Validate() error {
	// First Name
	firstName, err := validation.ValidateString("first_name", p.FirstName, 100, true)
	if err != nil {
		return err
	}
	p.FirstName = firstName

	// Last Name
	if p.LastName != nil {
		lastName, err := validation.ValidateString("last_name", *p.LastName, 100, false)
		if err != nil {
			return err
		}

		if lastName == "" {
			p.LastName = nil
		} else {
			p.LastName = &lastName
		}
	}

	return nil
}

// Repository: User
type UserRepository interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByGoogleSub(ctx context.Context, googleSub string) (*User, error)
	CreateUser(ctx context.Context, form UserForm) (*User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, patch UserPatch) (*User, error)
}
