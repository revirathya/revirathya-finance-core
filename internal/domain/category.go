package domain

import (
	"context"
	"encoding/base64"
	"unicode/utf8"

	"github.com/google/uuid"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/domain/validation"
)

// Entity
type Category struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	CategoryName string
}

// Form
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

// Patch
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

// Cursor
type CategoryCursor struct {
	CategoryName string
}

func (c CategoryCursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.CategoryName))
}

func DecodeCategoryCursor(token string) (CategoryCursor, error) {
	invalid := &errDom.ValidationError{Field: "page_token", Reason: "is invalid"}

	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return CategoryCursor{}, invalid
	}

	if !utf8.Valid(raw) {
		return CategoryCursor{}, invalid
	}

	categoryName := string(raw)
	return CategoryCursor{
		CategoryName: categoryName,
	}, nil
}

// Pagination
type CategoryListQuery struct {
	After *CategoryCursor
	Limit int
}

type CategoryPage struct {
	Categories []Category
	NextCursor *CategoryCursor
}

// Repository
type CategoryRepository interface {
	CreateCategory(ctx context.Context, userID uuid.UUID, form CategoryForm) (*Category, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Category, error)
	ListCategories(ctx context.Context, userID uuid.UUID, query CategoryListQuery) ([]Category, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch CategoryPatch) (*Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
