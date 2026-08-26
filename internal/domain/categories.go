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

// Repository: Category
type CategoryRepository interface {
	ListCategories(ctx context.Context) ([]Category, error)
}
