package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain/validation"
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

func (f *CategoryForm) Validate() error {
	categoryName, err := validation.ValidateString("category_name", f.CategoryName, 100, true)
	if err != nil {
		return err
	}
	f.CategoryName = categoryName
	return nil
}

type CategoryPatch struct {
	CategoryName *string
}

func (f *CategoryPatch) Validate() error {
	if f.CategoryName != nil {
		categoryName, err := validation.ValidateString("category_name", *f.CategoryName, 100, true)
		if err != nil {
			return err
		}
		f.CategoryName = &categoryName
	}
	return nil
}

// Repository
type CategoryRepository interface {
	CreateCategory(ctx context.Context, userID uuid.UUID, form CategoryForm) (*Category, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Category, error)
	ListCategories(ctx context.Context, userID uuid.UUID) ([]Category, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch CategoryPatch) (*Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
