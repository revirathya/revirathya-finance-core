//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"

	// Repository
	rCategory "github.com/revirathya/revirathya-finance-core/internal/repository/postgres/category"
	rExpense "github.com/revirathya/revirathya-finance-core/internal/repository/postgres/expense"
)

func TestCategoryRepository(t *testing.T) {
	ctx := context.Background()
	repo := rCategory.NewCategoryRepositoryImpl(testPool)

	t.Run("each user sees only their own categories, ordered by name", func(t *testing.T) {
		alice, bob := newUser(t), newUser(t)
		newCategory(t, alice.ID, "Transport")
		newCategory(t, alice.ID, "Food")
		newCategory(t, bob.ID, "Bills")

		got, err := repo.ListCategories(ctx, alice.ID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 || got[0].CategoryName != "Food" || got[1].CategoryName != "Transport" {
			t.Fatalf("got %+v, want alice's Food then Transport", got)
		}
		for _, c := range got {
			if c.UserID != alice.ID {
				t.Fatalf("category %q belongs to %v, want %v", c.CategoryName, c.UserID, alice.ID)
			}
		}
	})

	t.Run("two users can use the same name", func(t *testing.T) {
		newCategory(t, newUser(t).ID, "Food")
		newCategory(t, newUser(t).ID, "Food")
	})

	t.Run("duplicate name for the same user is a conflict", func(t *testing.T) {
		alice := newUser(t)
		newCategory(t, alice.ID, "Food")

		if _, err := repo.CreateCategory(ctx, alice.ID, domain.CategoryForm{CategoryName: "Food"}); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("got %v, want domain.ErrConflict", err)
		}
	})

	t.Run("renaming to an existing name is a conflict", func(t *testing.T) {
		alice := newUser(t)
		newCategory(t, alice.ID, "Food")
		bills := newCategory(t, alice.ID, "Bills")

		if _, err := repo.UpdateCategory(ctx, bills.ID, alice.ID, domain.CategoryForm{CategoryName: "Food"}); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("got %v, want domain.ErrConflict", err)
		}
	})

	t.Run("owner can get, rename and delete", func(t *testing.T) {
		alice := newUser(t)
		food := newCategory(t, alice.ID, "Food")

		if _, err := repo.GetCategoryByID(ctx, food.ID, alice.ID); err != nil {
			t.Fatalf("get: %v", err)
		}
		renamed, err := repo.UpdateCategory(ctx, food.ID, alice.ID, domain.CategoryForm{CategoryName: "Groceries"})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if renamed.CategoryName != "Groceries" {
			t.Fatalf("name %q, want %q", renamed.CategoryName, "Groceries")
		}
		if err := repo.DeleteCategory(ctx, food.ID, alice.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}
		if _, err := repo.GetCategoryByID(ctx, food.ID, alice.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("get after delete: got %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("another user's category is not found", func(t *testing.T) {
		alice, bob := newUser(t), newUser(t)
		food := newCategory(t, alice.ID, "Food")

		if _, err := repo.GetCategoryByID(ctx, food.ID, bob.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("get: got %v, want domain.ErrNotFound", err)
		}
		if _, err := repo.UpdateCategory(ctx, food.ID, bob.ID, domain.CategoryForm{CategoryName: "Hacked"}); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("update: got %v, want domain.ErrNotFound", err)
		}
		if err := repo.DeleteCategory(ctx, food.ID, bob.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("delete: got %v, want domain.ErrNotFound", err)
		}

		// Bob's attempts must not have changed anything.
		got, err := repo.GetCategoryByID(ctx, food.ID, alice.ID)
		if err != nil {
			t.Fatalf("alice get: %v", err)
		}
		if got.CategoryName != "Food" {
			t.Fatalf("name %q, want unchanged %q", got.CategoryName, "Food")
		}
	})

	t.Run("a category used by an expense cannot be deleted", func(t *testing.T) {
		alice := newUser(t)
		food := newCategory(t, alice.ID, "Food")
		expense := newExpense(t, alice.ID, food.ID, "2026-01-15", "10.00")

		if err := repo.DeleteCategory(ctx, food.ID, alice.ID); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("delete in-use category: got %v, want domain.ErrConflict", err)
		}

		if err := rExpense.NewExpenseRepositoryImpl(testPool).DeleteExpense(ctx, expense.ID, alice.ID); err != nil {
			t.Fatalf("delete expense: %v", err)
		}
		if err := repo.DeleteCategory(ctx, food.ID, alice.ID); err != nil {
			t.Fatalf("delete unused category: %v", err)
		}
	})
}
