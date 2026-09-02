package category

import (
	"context"

	"github.com/google/uuid"

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

func (uc *CategoryUsecase) CreateCategory(ctx context.Context, categoryName string) (*domain.Category, error) {
	input := domain.CategoryForm{
		CategoryName: categoryName,
	}
	return uc.ctgRepo.CreateCategory(ctx, input)
}

func (uc *CategoryUsecase) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return uc.ctgRepo.ListCategories(ctx)
}

func (uc *CategoryUsecase) GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	return uc.ctgRepo.GetCategoryByID(ctx, id)
}

func (uc *CategoryUsecase) UpdateCategory(ctx context.Context, id uuid.UUID, categoryName string) (*domain.Category, error) {
	patch := domain.CategoryForm{
		CategoryName: categoryName,
	}
	return uc.ctgRepo.UpdateCategory(ctx, id, patch)
}

func (uc *CategoryUsecase) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return uc.ctgRepo.DeleteCategory(ctx, id)
}
