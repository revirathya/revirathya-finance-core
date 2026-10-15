//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/repository/postgres"
)

func TestExpenseRepository(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewExpenseRepositoryImpl(testPool)

	t.Run("create and get round-trip all fields", func(t *testing.T) {
		alice := newUser(t)
		food := newCategory(t, alice.ID, "Food")

		created, err := repo.CreateExpense(ctx, alice.ID, domain.ExpenseForm{
			CategoryID:    food.ID,
			TransactionDt: day(t, "2026-01-15"),
			Description:   ptr("Lunch"),
			Amount:        decimal.RequireFromString("12500.50"),
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		got, err := repo.GetExpenseByID(ctx, created.ID, alice.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.UserID != alice.ID || got.CategoryID != food.ID || got.CategoryName != "Food" {
			t.Fatalf("got owner %v, category %v %q; want %v, %v %q", got.UserID, got.CategoryID, got.CategoryName, alice.ID, food.ID, "Food")
		}
		if got.TransactionDt.Format("2006-01-02") != "2026-01-15" {
			t.Fatalf("date %v, want 2026-01-15", got.TransactionDt)
		}
		if got.Description == nil || *got.Description != "Lunch" {
			t.Fatalf("description %v, want %q", got.Description, "Lunch")
		}
		if !got.Amount.Equal(decimal.RequireFromString("12500.50")) {
			t.Fatalf("amount %s, want 12500.50", got.Amount)
		}
	})

	t.Run("description is optional", func(t *testing.T) {
		alice := newUser(t)
		expense := newExpense(t, alice.ID, newCategory(t, alice.ID, "Food").ID, "2026-01-15", "1")

		got, err := repo.GetExpenseByID(ctx, expense.ID, alice.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.Description != nil {
			t.Fatalf("description %q, want nil", *got.Description)
		}
	})

	t.Run("creating with another user's category is invalid input", func(t *testing.T) {
		alice, bob := newUser(t), newUser(t)
		bobsFood := newCategory(t, bob.ID, "Food")

		_, err := repo.CreateExpense(ctx, alice.ID, domain.ExpenseForm{
			CategoryID:    bobsFood.ID,
			TransactionDt: day(t, "2026-01-15"),
			Amount:        decimal.RequireFromString("10"),
		})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("got %v, want domain.ErrInvalidInput", err)
		}
	})

	t.Run("updating to another user's category is invalid input", func(t *testing.T) {
		alice, bob := newUser(t), newUser(t)
		expense := newExpense(t, alice.ID, newCategory(t, alice.ID, "Food").ID, "2026-01-15", "10")
		bobsFood := newCategory(t, bob.ID, "Food")

		_, err := repo.UpdateExpense(ctx, expense.ID, alice.ID, domain.ExpenseForm{
			CategoryID:    bobsFood.ID,
			TransactionDt: day(t, "2026-01-15"),
			Amount:        decimal.RequireFromString("10"),
		})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("got %v, want domain.ErrInvalidInput", err)
		}
	})

	t.Run("owner can update and delete", func(t *testing.T) {
		alice := newUser(t)
		food, bills := newCategory(t, alice.ID, "Food"), newCategory(t, alice.ID, "Bills")
		expense := newExpense(t, alice.ID, food.ID, "2026-01-15", "10")

		updated, err := repo.UpdateExpense(ctx, expense.ID, alice.ID, domain.ExpenseForm{
			CategoryID:    bills.ID,
			TransactionDt: day(t, "2026-01-20"),
			Amount:        decimal.RequireFromString("99.99"),
		})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if updated.CategoryID != bills.ID || !updated.Amount.Equal(decimal.RequireFromString("99.99")) {
			t.Fatalf("got category %v amount %s, want %v and 99.99", updated.CategoryID, updated.Amount, bills.ID)
		}

		if err := repo.DeleteExpense(ctx, expense.ID, alice.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}
		if err := repo.DeleteExpense(ctx, expense.ID, alice.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("delete twice: got %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("another user's expense is not found", func(t *testing.T) {
		alice, bob := newUser(t), newUser(t)
		expense := newExpense(t, alice.ID, newCategory(t, alice.ID, "Food").ID, "2026-01-15", "10")
		bobsFood := newCategory(t, bob.ID, "Food")

		if _, err := repo.GetExpenseByID(ctx, expense.ID, bob.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("get: got %v, want domain.ErrNotFound", err)
		}
		_, err := repo.UpdateExpense(ctx, expense.ID, bob.ID, domain.ExpenseForm{
			CategoryID:    bobsFood.ID,
			TransactionDt: day(t, "2026-01-15"),
			Amount:        decimal.RequireFromString("0"),
		})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("update: got %v, want domain.ErrNotFound", err)
		}
		if err := repo.DeleteExpense(ctx, expense.ID, bob.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("delete: got %v, want domain.ErrNotFound", err)
		}

		// Bob's attempts must not have changed anything.
		got, err := repo.GetExpenseByID(ctx, expense.ID, alice.ID)
		if err != nil {
			t.Fatalf("alice get: %v", err)
		}
		if !got.Amount.Equal(decimal.RequireFromString("10")) {
			t.Fatalf("amount %s, want unchanged 10", got.Amount)
		}
	})

	t.Run("list returns only the user's expenses, newest first", func(t *testing.T) {
		alice, bob := newUser(t), newUser(t)
		food := newCategory(t, alice.ID, "Food")
		newExpense(t, alice.ID, food.ID, "2026-01-10", "1")
		newExpense(t, alice.ID, food.ID, "2026-01-20", "2")
		newExpense(t, alice.ID, food.ID, "2026-01-15", "3")
		newExpense(t, bob.ID, newCategory(t, bob.ID, "Food").ID, "2026-01-25", "4")

		got, err := repo.ListExpenses(ctx, alice.ID, nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var dates []string
		for _, e := range got {
			if e.UserID != alice.ID {
				t.Fatalf("expense %v belongs to %v, want %v", e.ID, e.UserID, alice.ID)
			}
			dates = append(dates, e.TransactionDt.Format("2006-01-02"))
		}
		want := []string{"2026-01-20", "2026-01-15", "2026-01-10"}
		if len(dates) != len(want) || dates[0] != want[0] || dates[1] != want[1] || dates[2] != want[2] {
			t.Fatalf("dates %v, want %v", dates, want)
		}
	})

	t.Run("list filters by date", func(t *testing.T) {
		alice := newUser(t)
		food := newCategory(t, alice.ID, "Food")
		newExpense(t, alice.ID, food.ID, "2026-01-10", "1")
		newExpense(t, alice.ID, food.ID, "2026-01-15", "2")
		newExpense(t, alice.ID, food.ID, "2026-01-15", "3")

		filter := day(t, "2026-01-15")
		got, err := repo.ListExpenses(ctx, alice.ID, &filter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d expenses, want 2 on 2026-01-15", len(got))
		}
		for _, e := range got {
			if e.TransactionDt.Format("2006-01-02") != "2026-01-15" {
				t.Fatalf("expense on %v, want only 2026-01-15", e.TransactionDt)
			}
		}
	})

	t.Run("deleting a user deletes their expenses and categories", func(t *testing.T) {
		alice := newUser(t)
		food := newCategory(t, alice.ID, "Food")
		expense := newExpense(t, alice.ID, food.ID, "2026-01-15", "10")

		if _, err := testPool.Exec(ctx, "DELETE FROM app.users WHERE id = $1", alice.ID); err != nil {
			t.Fatalf("delete user: %v", err)
		}

		if _, err := repo.GetExpenseByID(ctx, expense.ID, alice.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expense: got %v, want domain.ErrNotFound", err)
		}
		if _, err := postgres.NewCategoryRepositoryImpl(testPool).GetCategoryByID(ctx, food.ID, alice.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("category: got %v, want domain.ErrNotFound", err)
		}
	})
}
