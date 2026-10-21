package category

import (
	"context"

	"github.com/google/uuid"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/domain/pagination"
)

// Usecase: Category
type CategoryUsecase struct {
	category domain.CategoryRepository
}

func NewCategoryUsecase(category domain.CategoryRepository) *CategoryUsecase {
	return &CategoryUsecase{
		category: category,
	}
}

func (uc *CategoryUsecase) CreateCategory(ctx context.Context, userID uuid.UUID, form domain.CategoryForm) (*domain.Category, error) {
	if err := form.Validate(); err != nil {
		return nil, err
	}
	return uc.category.CreateCategory(ctx, userID, form)
}

func (uc *CategoryUsecase) GetCategoryByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Category, error) {
	return uc.category.GetCategoryByID(ctx, id, userID)
}

func (uc *CategoryUsecase) ListCategories(ctx context.Context, userID uuid.UUID, pageReq pagination.Request) (*domain.CategoryPage, error) {
	// Define page size
	size, err := pagination.SetPageSize(pageReq.Size)
	if err != nil {
		return nil, err
	}

	// Decode token if exists
	query := domain.CategoryListQuery{
		Limit: size + 1,
	}
	if pageReq.Token != "" {
		// Not the first page
		cursor, err := domain.DecodeCategoryCursor(pageReq.Token)
		if err != nil {
			return nil, err
		}
		query.After = &cursor
	}

	// List categories
	categories, err := uc.category.ListCategories(ctx, userID, query)
	if err != nil {
		return nil, err
	}

	// Parse page result
	items, last, more := pagination.Cut(categories, size)
	response := &domain.CategoryPage{
		Categories: items,
	}
	if more {
		response.NextCursor = &domain.CategoryCursor{
			CategoryName: last.CategoryName,
		}
	}
	return response, nil
}

func (uc *CategoryUsecase) UpdateCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch domain.CategoryPatch) (*domain.Category, error) {
	if err := patch.Validate(); err != nil {
		return nil, err
	}
	return uc.category.UpdateCategory(ctx, id, userID, patch)
}

func (uc *CategoryUsecase) DeleteCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return uc.category.DeleteCategory(ctx, id, userID)
}
