package category_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/category"
)

// Fake
type fakeCategoryRepo struct {
	categories   []domain.Category
	categoryByID *domain.Category
	err          error
}

func (f *fakeCategoryRepo) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return f.categories, f.err
}

func (f *fakeCategoryRepo) GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	return f.categoryByID, f.err
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

func TestGetCategoryByID(t *testing.T) {
	id := uuid.New()
	want := &domain.Category{ID: id, CategoryName: "Food"}

	t.Run("returns category", func(t *testing.T) {
		repo := &fakeCategoryRepo{categoryByID: want}
		uc := category.NewCategoryUsecase(repo)

		got, err := uc.GetCategoryByID(context.Background(), id)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &fakeCategoryRepo{err: errors.New("not found")}
		uc := category.NewCategoryUsecase(repo)

		if _, err := uc.GetCategoryByID(context.Background(), id); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
