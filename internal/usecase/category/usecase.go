package category

import (
	"context"

	"github.com/google/uuid"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
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

func (uc *CategoryUsecase) ListCategories(ctx context.Context, userID uuid.UUID) ([]domain.Category, error) {
	return uc.ctgRepo.ListCategories(ctx, userID)
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
