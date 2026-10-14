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

func (uc *CategoryUsecase) CreateCategory(ctx context.Context, userID uuid.UUID, categoryName string) (*domain.Category, error) {
	input := domain.CategoryForm{
		CategoryName: categoryName,
	}
	return uc.ctgRepo.CreateCategory(ctx, userID, input)
}

func (uc *CategoryUsecase) GetCategoryByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Category, error) {
	return uc.ctgRepo.GetCategoryByID(ctx, id, userID)
}

func (uc *CategoryUsecase) ListCategories(ctx context.Context, userID uuid.UUID) ([]domain.Category, error) {
	return uc.ctgRepo.ListCategories(ctx, userID)
}

func (uc *CategoryUsecase) UpdateCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID, categoryName string) (*domain.Category, error) {
	patch := domain.CategoryForm{
		CategoryName: categoryName,
	}
	return uc.ctgRepo.UpdateCategory(ctx, id, userID, patch)
}

func (uc *CategoryUsecase) DeleteCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return uc.ctgRepo.DeleteCategory(ctx, id, userID)
}
