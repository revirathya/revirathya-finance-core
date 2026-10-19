//go:build integration

package expense_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/shopspring/decimal"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"

	// Repository
	rCategory "github.com/revirathya/revirathya-finance-core/internal/repository/postgres/category"
	rExpense "github.com/revirathya/revirathya-finance-core/internal/repository/postgres/expense"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithPostgres(m))
}

func TestExpenseRepository(t *testing.T) {
	ctx := context.Background()
	repo := rExpense.NewExpenseRepositoryImpl(testutil.TestPool)

	t.Run("create and get round-trip all fields", func(t *testing.T) {
		alice := testutil.NewUser(t)
		food := testutil.NewCategory(t, alice.ID, "Food")

		created, err := repo.CreateExpense(ctx, domain.ExpenseForm{
			UserID:        alice.ID,
			CategoryID:    food.ID,
			TransactionDt: testutil.Day(t, "2026-01-15"),
			Description:   testutil.Ptr("Lunch"),
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
		alice := testutil.NewUser(t)
		expense := testutil.NewExpense(t, alice.ID, testutil.NewCategory(t, alice.ID, "Food").ID, "2026-01-15", "1")

		got, err := repo.GetExpenseByID(ctx, expense.ID, alice.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.Description != nil {
			t.Fatalf("description %q, want nil", *got.Description)
		}
	})

	t.Run("creating with another user's category is invalid input", func(t *testing.T) {
		alice, bob := testutil.NewUser(t), testutil.NewUser(t)
		bobsFood := testutil.NewCategory(t, bob.ID, "Food")

		_, err := repo.CreateExpense(ctx, domain.ExpenseForm{
			UserID:        alice.ID,
			CategoryID:    bobsFood.ID,
			TransactionDt: testutil.Day(t, "2026-01-15"),
			Amount:        decimal.RequireFromString("10"),
		})
		if !errors.Is(err, errDom.ErrInvalidInput) {
			t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
		}
	})

	t.Run("updating to another user's category is invalid input", func(t *testing.T) {
		alice, bob := testutil.NewUser(t), testutil.NewUser(t)
		expense := testutil.NewExpense(t, alice.ID, testutil.NewCategory(t, alice.ID, "Food").ID, "2026-01-15", "10")
		bobsFood := testutil.NewCategory(t, bob.ID, "Food")

		_, err := repo.UpdateExpense(ctx, expense.ID, alice.ID, domain.ExpensePatch{
			CategoryID:    bobsFood.ID,
			TransactionDt: testutil.Day(t, "2026-01-15"),
			Amount:        decimal.RequireFromString("10"),
		})
		if !errors.Is(err, errDom.ErrInvalidInput) {
			t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
		}
	})

	t.Run("owner can update and delete", func(t *testing.T) {
		alice := testutil.NewUser(t)
		food, bills := testutil.NewCategory(t, alice.ID, "Food"), testutil.NewCategory(t, alice.ID, "Bills")
		expense := testutil.NewExpense(t, alice.ID, food.ID, "2026-01-15", "10")

		updated, err := repo.UpdateExpense(ctx, expense.ID, alice.ID, domain.ExpensePatch{
			CategoryID:    bills.ID,
			TransactionDt: testutil.Day(t, "2026-01-20"),
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
		if err := repo.DeleteExpense(ctx, expense.ID, alice.ID); !errors.Is(err, errDom.ErrNotFound) {
			t.Fatalf("delete twice: got %v, want errDom.ErrNotFound", err)
		}
	})

	t.Run("another user's expense is not found", func(t *testing.T) {
		alice, bob := testutil.NewUser(t), testutil.NewUser(t)
		expense := testutil.NewExpense(t, alice.ID, testutil.NewCategory(t, alice.ID, "Food").ID, "2026-01-15", "10")
		bobsFood := testutil.NewCategory(t, bob.ID, "Food")

		if _, err := repo.GetExpenseByID(ctx, expense.ID, bob.ID); !errors.Is(err, errDom.ErrNotFound) {
			t.Fatalf("get: got %v, want errDom.ErrNotFound", err)
		}
		_, err := repo.UpdateExpense(ctx, expense.ID, bob.ID, domain.ExpensePatch{
			CategoryID:    bobsFood.ID,
			TransactionDt: testutil.Day(t, "2026-01-15"),
			Amount:        decimal.RequireFromString("0"),
		})
		if !errors.Is(err, errDom.ErrNotFound) {
			t.Fatalf("update: got %v, want errDom.ErrNotFound", err)
		}
		if err := repo.DeleteExpense(ctx, expense.ID, bob.ID); !errors.Is(err, errDom.ErrNotFound) {
			t.Fatalf("delete: got %v, want errDom.ErrNotFound", err)
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
		alice, bob := testutil.NewUser(t), testutil.NewUser(t)
		food := testutil.NewCategory(t, alice.ID, "Food")
		testutil.NewExpense(t, alice.ID, food.ID, "2026-01-10", "1")
		testutil.NewExpense(t, alice.ID, food.ID, "2026-01-20", "2")
		testutil.NewExpense(t, alice.ID, food.ID, "2026-01-15", "3")
		testutil.NewExpense(t, bob.ID, testutil.NewCategory(t, bob.ID, "Food").ID, "2026-01-25", "4")

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
		alice := testutil.NewUser(t)
		food := testutil.NewCategory(t, alice.ID, "Food")
		testutil.NewExpense(t, alice.ID, food.ID, "2026-01-10", "1")
		testutil.NewExpense(t, alice.ID, food.ID, "2026-01-15", "2")
		testutil.NewExpense(t, alice.ID, food.ID, "2026-01-15", "3")

		filter := testutil.Day(t, "2026-01-15")
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
		alice := testutil.NewUser(t)
		food := testutil.NewCategory(t, alice.ID, "Food")
		expense := testutil.NewExpense(t, alice.ID, food.ID, "2026-01-15", "10")

		if _, err := testutil.TestPool.Exec(ctx, "DELETE FROM app.users WHERE id = $1", alice.ID); err != nil {
			t.Fatalf("delete user: %v", err)
		}

		if _, err := repo.GetExpenseByID(ctx, expense.ID, alice.ID); !errors.Is(err, errDom.ErrNotFound) {
			t.Fatalf("expense: got %v, want errDom.ErrNotFound", err)
		}
		if _, err := rCategory.NewCategoryRepositoryImpl(testutil.TestPool).GetCategoryByID(ctx, food.ID, alice.ID); !errors.Is(err, errDom.ErrNotFound) {
			t.Fatalf("category: got %v, want errDom.ErrNotFound", err)
		}
	})
}

// Backstop for input that skipped Validate(): amount is NUMERIC(14, 2), and an amount
// that doesn't fit comes back as ErrInvalidInput, not as an unknown (internal) error.
func TestExpenseRepositoryLimits(t *testing.T) {
	ctx := context.Background()
	repo := rExpense.NewExpenseRepositoryImpl(testutil.TestPool)

	t.Run("largest amount the column holds is stored", func(t *testing.T) {
		alice := testutil.NewUser(t)
		testutil.NewExpense(t, alice.ID, testutil.NewCategory(t, alice.ID, "Food").ID, "2026-01-15", "999999999999.99")
	})

	t.Run("amount of 10^12 on create is invalid input", func(t *testing.T) {
		alice := testutil.NewUser(t)
		_, err := repo.CreateExpense(ctx, domain.ExpenseForm{
			UserID:        alice.ID,
			CategoryID:    testutil.NewCategory(t, alice.ID, "Food").ID,
			TransactionDt: testutil.Day(t, "2026-01-15"),
			Amount:        decimal.RequireFromString("1000000000000"),
		})
		if !errors.Is(err, errDom.ErrInvalidInput) {
			t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
		}
	})

	t.Run("amount of 10^12 on update is invalid input", func(t *testing.T) {
		alice := testutil.NewUser(t)
		food := testutil.NewCategory(t, alice.ID, "Food")
		expense := testutil.NewExpense(t, alice.ID, food.ID, "2026-01-15", "10")

		_, err := repo.UpdateExpense(ctx, expense.ID, alice.ID, domain.ExpensePatch{
			CategoryID:    food.ID,
			TransactionDt: testutil.Day(t, "2026-01-15"),
			Amount:        decimal.RequireFromString("1000000000000"),
		})
		if !errors.Is(err, errDom.ErrInvalidInput) {
			t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
		}
	})
}
