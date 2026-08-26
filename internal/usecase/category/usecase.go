package category

import (
	"context"

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

func (uc *CategoryUsecase) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return uc.ctgRepo.ListCategories(ctx)
}
