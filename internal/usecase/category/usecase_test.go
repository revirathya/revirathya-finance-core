package category_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/category"
)

// Fake
type fakeCategoryRepo struct {
	categories []domain.Category
	err        error
}

func (f *fakeCategoryRepo) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return f.categories, f.err
}

// Test
func TestListCategories(t *testing.T) {
	tests := []struct {
		name    string
		repo    *fakeCategoryRepo
		want    []domain.Category
		wantErr bool
	}{
		{
			"returns categories",
			&fakeCategoryRepo{
				categories: []domain.Category{{CategoryName: "Food"}},
			},
			[]domain.Category{{CategoryName: "Food"}},
			false,
		},
		{
			"propagates repo error",
			&fakeCategoryRepo{
				err: errors.New("db down"),
			},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := category.NewCategoryUsecase(tt.repo)
			got, err := uc.ListCategories(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
