package domain

import (
	"context"

	"github.com/google/uuid"
)

// Entity
type Category struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	CategoryName string
}

type CategoryForm struct {
	CategoryName string
}

// Repository
type CategoryRepository interface {
	CreateCategory(ctx context.Context, userID uuid.UUID, input CategoryForm) (*Category, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Category, error)
	ListCategories(ctx context.Context, userID uuid.UUID) ([]Category, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch CategoryForm) (*Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
