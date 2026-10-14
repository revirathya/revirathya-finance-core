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
	createResp   *domain.Category
	updateResp   *domain.Category
	err          error

	gotCreateInput domain.CategoryForm
	gotUpdateID    uuid.UUID
	gotUpdatePatch domain.CategoryForm
	gotDeleteID    uuid.UUID
}

func (f *fakeCategoryRepo) ListCategories(ctx context.Context, userID uuid.UUID) ([]domain.Category, error) {
	return f.categories, f.err
}

func (f *fakeCategoryRepo) GetCategoryByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Category, error) {
	return f.categoryByID, f.err
}

func (f *fakeCategoryRepo) CreateCategory(ctx context.Context, userID uuid.UUID, input domain.CategoryForm) (*domain.Category, error) {
	f.gotCreateInput = input
	return f.createResp, f.err
}

func (f *fakeCategoryRepo) UpdateCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch domain.CategoryForm) (*domain.Category, error) {
	f.gotUpdateID = id
	f.gotUpdatePatch = patch
	return f.updateResp, f.err
}

func (f *fakeCategoryRepo) DeleteCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	f.gotDeleteID = id
	return f.err
}

// Test
func TestCreateCategory(t *testing.T) {
	want := &domain.Category{ID: uuid.New(), CategoryName: "Food"}

	t.Run("creates and returns category", func(t *testing.T) {
		userID := uuid.New()
		repo := &fakeCategoryRepo{createResp: want}
		uc := category.NewCategoryUsecase(repo)

		got, err := uc.CreateCategory(context.Background(), userID, "Food")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if repo.gotCreateInput.CategoryName != "Food" {
			t.Fatalf("repo received category_name %q, want %q", repo.gotCreateInput.CategoryName, "Food")
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		userID := uuid.New()
		repo := &fakeCategoryRepo{err: errors.New("db down")}
		uc := category.NewCategoryUsecase(repo)

		if _, err := uc.CreateCategory(context.Background(), userID, "Food"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetCategoryByID(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	want := &domain.Category{ID: id, CategoryName: "Food"}

	t.Run("returns category", func(t *testing.T) {
		repo := &fakeCategoryRepo{categoryByID: want}
		uc := category.NewCategoryUsecase(repo)

		got, err := uc.GetCategoryByID(context.Background(), id, userID)

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

		if _, err := uc.GetCategoryByID(context.Background(), id, userID); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

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

	userID := uuid.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := category.NewCategoryUsecase(tt.repo)
			got, err := uc.ListCategories(context.Background(), userID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateCategory(t *testing.T) {
	id := uuid.New()
	want := &domain.Category{ID: id, CategoryName: "Groceries"}

	t.Run("updates and returns category", func(t *testing.T) {
		userID := uuid.New()
		repo := &fakeCategoryRepo{updateResp: want}
		uc := category.NewCategoryUsecase(repo)

		got, err := uc.UpdateCategory(context.Background(), id, userID, "Groceries")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if repo.gotUpdateID != id {
			t.Fatalf("repo received id %v, want %v", repo.gotUpdateID, id)
		}
		if repo.gotUpdatePatch.CategoryName != "Groceries" {
			t.Fatalf("repo received category_name %q, want %q", repo.gotUpdatePatch.CategoryName, "Groceries")
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		userID := uuid.New()
		repo := &fakeCategoryRepo{err: errors.New("not found")}
		uc := category.NewCategoryUsecase(repo)

		if _, err := uc.UpdateCategory(context.Background(), id, userID, "Groceries"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDeleteCategory(t *testing.T) {
	id := uuid.New()

	t.Run("deletes category", func(t *testing.T) {
		userID := uuid.New()
		repo := &fakeCategoryRepo{}
		uc := category.NewCategoryUsecase(repo)

		if err := uc.DeleteCategory(context.Background(), id, userID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotDeleteID != id {
			t.Fatalf("repo received id %v, want %v", repo.gotDeleteID, id)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		userID := uuid.New()
		repo := &fakeCategoryRepo{err: errors.New("fk violation")}
		uc := category.NewCategoryUsecase(repo)

		if err := uc.DeleteCategory(context.Background(), id, userID); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
