package category_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/category"
)

func TestCategoryValidation(t *testing.T) {
	for _, name := range []string{"", "   ", strings.Repeat("a", 101)} {
		t.Run("create rejects invalid name before the repository", func(t *testing.T) {
			repo := &fakeCategoryRepo{}
			uc := category.NewCategoryUsecase(repo)

			_, err := uc.CreateCategory(context.Background(), uuid.New(), domain.CategoryForm{CategoryName: name})

			if !errors.Is(err, errDom.ErrInvalidInput) {
				t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
			}
			if repo.gotCreateInput.CategoryName != "" {
				t.Fatalf("repository was called with %q", repo.gotCreateInput.CategoryName)
			}
		})

		t.Run("update rejects invalid name before the repository", func(t *testing.T) {
			repo := &fakeCategoryRepo{}
			uc := category.NewCategoryUsecase(repo)

			_, err := uc.UpdateCategory(context.Background(), uuid.New(), uuid.New(), domain.CategoryPatch{CategoryName: &name})

			if !errors.Is(err, errDom.ErrInvalidInput) {
				t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
			}
			if repo.gotUpdateID != uuid.Nil {
				t.Fatal("repository was called")
			}
		})
	}

	t.Run("create and update pass the trimmed name to the repository", func(t *testing.T) {
		repo := &fakeCategoryRepo{}
		uc := category.NewCategoryUsecase(repo)

		if _, err := uc.CreateCategory(context.Background(), uuid.New(), domain.CategoryForm{CategoryName: "  Food  "}); err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := uc.UpdateCategory(context.Background(), uuid.New(), uuid.New(), domain.CategoryPatch{CategoryName: ptr("  Groceries  ")}); err != nil {
			t.Fatalf("update: %v", err)
		}
		if repo.gotCreateInput.CategoryName != "Food" || *repo.gotUpdatePatch.CategoryName != "Groceries" {
			t.Fatalf("repository received %q and %q, want trimmed names", repo.gotCreateInput.CategoryName, *repo.gotUpdatePatch.CategoryName)
		}
	})
}
