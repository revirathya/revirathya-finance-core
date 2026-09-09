package domain

import (
	"context"

	"github.com/google/uuid"
)

// Entity: Category
type Category struct {
	ID           uuid.UUID
	CategoryName string
}

type CategoryForm struct {
	CategoryName string
}

// Repository: Category
type CategoryRepository interface {
	CreateCategory(ctx context.Context, input CategoryForm) (*Category, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*Category, error)
	ListCategories(ctx context.Context) ([]Category, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, patch CategoryForm) (*Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}
