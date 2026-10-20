package category

import (
	"context"

	"github.com/google/uuid"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/utils"
)

// Usecase: Category
type CategoryUsecase struct {
	ctgRepo domain.CategoryRepository
}

func NewCategoryUsecase(ctgRepo domain.CategoryRepository) *CategoryUsecase {
	return &CategoryUsecase{
		ctgRepo: ctgRepo,
	}
}

func (uc *CategoryUsecase) CreateCategory(ctx context.Context, userID uuid.UUID, categoryName string) (*domain.Category, error) {
	form := domain.CategoryForm{
		CategoryName: categoryName,
	}
	if err := form.Validate(); err != nil {
		return nil, err
	}

	return uc.ctgRepo.CreateCategory(ctx, userID, form)
}

func (uc *CategoryUsecase) GetCategoryByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Category, error) {
	return uc.ctgRepo.GetCategoryByID(ctx, id, userID)
}

func (uc *CategoryUsecase) ListCategories(ctx context.Context, userID uuid.UUID, pageSize int, pageToken string) (*domain.CategoryPage, error) {
	// Define page size
	ps, err := utils.SetPageSize(pageSize)
	if err != nil {
		return nil, err
	}

	// Decode token if exists
	query := domain.CategoryListQuery{
		Limit: ps + 1,
	}
	if pageToken != "" {
		// Not the first page
		cursor, err := domain.DecodeCategoryCursor(pageToken)
		if err != nil {
			return nil, err
		}
		query.After = &cursor
	}

	// Process
	categories, err := uc.ctgRepo.ListCategories(ctx, userID, query)
	if err != nil {
		return nil, err
	}

	// Parse page result
	page := &domain.CategoryPage{
		Categories: categories,
	}
	if len(categories) > ps {
		page.Categories = categories[:ps]
		last := page.Categories[ps-1]
		page.NextCursor = &domain.CategoryCursor{
			CategoryName: last.CategoryName,
		}
	}
	return page, nil
}

func (uc *CategoryUsecase) UpdateCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch domain.CategoryPatch) (*domain.Category, error) {
	if err := patch.Validate(); err != nil {
		return nil, err
	}
	return uc.ctgRepo.UpdateCategory(ctx, id, userID, patch)
}

func (uc *CategoryUsecase) DeleteCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return uc.ctgRepo.DeleteCategory(ctx, id, userID)
}
