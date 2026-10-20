//go:build integration

package expense_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
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
		if got.Description != "Lunch" {
			t.Fatalf("description %q, want %q", got.Description, "Lunch")
		}
		if !got.Amount.Equal(decimal.RequireFromString("12500.50")) {
			t.Fatalf("amount %s, want 12500.50", got.Amount)
		}
	})

	t.Run("create without a description stores an empty one", func(t *testing.T) {
		alice := testutil.NewUser(t)
		expense := testutil.NewExpense(t, alice.ID, testutil.NewCategory(t, alice.ID, "Food").ID, "2026-01-15", "1")

		got, err := repo.GetExpenseByID(ctx, expense.ID, alice.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.Description != "" {
			t.Fatalf("description %q, want empty", got.Description)
		}
	})

	t.Run("update keeps the description when omitted and clears it with an empty string", func(t *testing.T) {
		alice := testutil.NewUser(t)
		food := testutil.NewCategory(t, alice.ID, "Food")
		created, err := repo.CreateExpense(ctx, domain.ExpenseForm{
			UserID: alice.ID, CategoryID: food.ID, TransactionDt: testutil.Day(t, "2026-01-15"),
			Description: testutil.Ptr("Lunch"), Amount: decimal.RequireFromString("10"),
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		patch := domain.ExpensePatch{CategoryID: testutil.Ptr(food.ID), TransactionDt: testutil.Ptr(testutil.Day(t, "2026-01-15")), Amount: testutil.Ptr(decimal.RequireFromString("20"))}

		kept, err := repo.UpdateExpense(ctx, created.ID, alice.ID, patch)
		if err != nil {
			t.Fatalf("update without description: %v", err)
		}
		if kept.Description != "Lunch" {
			t.Fatalf("description %q after omitting it, want unchanged %q", kept.Description, "Lunch")
		}

		patch.Description = testutil.Ptr("")
		cleared, err := repo.UpdateExpense(ctx, created.ID, alice.ID, patch)
		if err != nil {
			t.Fatalf("update with empty description: %v", err)
		}
		if cleared.Description != "" {
			t.Fatalf("description %q after sending \"\", want empty", cleared.Description)
		}
	})

	t.Run("description column rejects NULL", func(t *testing.T) {
		alice := testutil.NewUser(t)
		expense := testutil.NewExpense(t, alice.ID, testutil.NewCategory(t, alice.ID, "Food").ID, "2026-01-15", "1")

		if _, err := testutil.TestPool.Exec(ctx, "UPDATE app.expenses SET description = NULL WHERE id = $1", expense.ID); err == nil {
			t.Fatal("NULL description was stored, want a not-null violation")
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
			CategoryID:    testutil.Ptr(bobsFood.ID),
			TransactionDt: testutil.Ptr(testutil.Day(t, "2026-01-15")),
			Amount:        testutil.Ptr(decimal.RequireFromString("10")),
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
			CategoryID:    testutil.Ptr(bills.ID),
			TransactionDt: testutil.Ptr(testutil.Day(t, "2026-01-20")),
			Amount:        testutil.Ptr(decimal.RequireFromString("99.99")),
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
			CategoryID:    testutil.Ptr(bobsFood.ID),
			TransactionDt: testutil.Ptr(testutil.Day(t, "2026-01-15")),
			Amount:        testutil.Ptr(decimal.RequireFromString("0")),
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

		got, err := repo.ListExpenses(ctx, alice.ID, domain.ExpenseListQuery{Limit: 50})

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
		got, err := repo.ListExpenses(ctx, alice.ID, domain.ExpenseListQuery{TransactionDt: &filter, Limit: 50})

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
			CategoryID:    testutil.Ptr(food.ID),
			TransactionDt: testutil.Ptr(testutil.Day(t, "2026-01-15")),
			Amount:        testutil.Ptr(decimal.RequireFromString("1000000000000")),
		})
		if !errors.Is(err, errDom.ErrInvalidInput) {
			t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
		}
	})
}

func TestExpenseRepositoryPagination(t *testing.T) {
	ctx := context.Background()
	repo := rExpense.NewExpenseRepositoryImpl(testutil.TestPool)

	// walk reads every page of size n the way the usecase does (n+1 rows, cursor = last kept row).
	walk := func(t *testing.T, userID uuid.UUID, filter *time.Time, n int) []domain.Expense {
		t.Helper()
		var all []domain.Expense
		query := domain.ExpenseListQuery{TransactionDt: filter, Limit: n + 1}
		for range 100 {
			rows, err := repo.ListExpenses(ctx, userID, query)
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			if len(rows) <= n {
				return append(all, rows...)
			}
			rows = rows[:n]
			all = append(all, rows...)
			last := rows[n-1]
			query.After = &domain.ExpenseCursor{TransactionDt: last.TransactionDt, ID: last.ID}
		}
		t.Fatal("pagination did not end")
		return nil
	}

	// Seven expenses, several on the same day, so the id tiebreaker matters.
	alice := testutil.NewUser(t)
	food := testutil.NewCategory(t, alice.ID, "Food")
	dates := []string{"2026-01-20", "2026-01-15", "2026-01-15", "2026-01-15", "2026-01-10", "2026-01-10", "2026-01-05"}
	created := map[uuid.UUID]bool{}
	for _, d := range dates {
		created[testutil.NewExpense(t, alice.ID, food.ID, d, "1").ID] = true
	}

	for _, size := range []int{1, 2, 3, 7, 50} {
		t.Run(fmt.Sprintf("pages of %d return every expense once, in order", size), func(t *testing.T) {
			got := walk(t, alice.ID, nil, size)

			if len(got) != len(dates) {
				t.Fatalf("got %d expenses, want %d", len(got), len(dates))
			}
			seen := map[uuid.UUID]bool{}
			for i, e := range got {
				if !created[e.ID] || seen[e.ID] {
					t.Fatalf("expense %v is unknown or repeated", e.ID)
				}
				seen[e.ID] = true
				if i > 0 {
					prev := got[i-1]
					after := prev.TransactionDt.After(e.TransactionDt) ||
						(prev.TransactionDt.Equal(e.TransactionDt) && bytes.Compare(prev.ID[:], e.ID[:]) > 0)
					if !after {
						t.Fatalf("row %d (%s %v) is not after row %d (%s %v) in (date, id) DESC order",
							i, e.TransactionDt.Format("2006-01-02"), e.ID, i-1, prev.TransactionDt.Format("2006-01-02"), prev.ID)
					}
				}
			}
		})
	}

	t.Run("date filter applies on every page", func(t *testing.T) {
		filter := testutil.Day(t, "2026-01-15")

		got := walk(t, alice.ID, &filter, 2)

		if len(got) != 3 {
			t.Fatalf("got %d expenses, want the 3 on 2026-01-15", len(got))
		}
		for _, e := range got {
			if !e.TransactionDt.Equal(filter) {
				t.Fatalf("expense on %v, want only 2026-01-15", e.TransactionDt)
			}
		}
	})

	t.Run("a cursor from another user's expense only returns the caller's own rows", func(t *testing.T) {
		bob := testutil.NewUser(t)
		bobs := testutil.NewExpense(t, bob.ID, testutil.NewCategory(t, bob.ID, "Food").ID, "2026-02-01", "1")

		got, err := repo.ListExpenses(ctx, alice.ID, domain.ExpenseListQuery{
			After: &domain.ExpenseCursor{TransactionDt: bobs.TransactionDt, ID: bobs.ID},
			Limit: 50,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != len(dates) {
			t.Fatalf("got %d expenses, want alice's %d", len(got), len(dates))
		}
		for _, e := range got {
			if e.UserID != alice.ID {
				t.Fatalf("expense %v belongs to %v, want only alice's", e.ID, e.UserID)
			}
		}
	})

	t.Run("limit caps the rows returned", func(t *testing.T) {
		got, err := repo.ListExpenses(ctx, alice.ID, domain.ExpenseListQuery{Limit: 3})

		if err != nil || len(got) != 3 {
			t.Fatalf("got %d rows, %v; want 3", len(got), err)
		}
	})
}

// Update is a patch: each field left nil keeps its stored value.
func TestExpenseRepositoryPatch(t *testing.T) {
	ctx := context.Background()
	repo := rExpense.NewExpenseRepositoryImpl(testutil.TestPool)
	alice := testutil.NewUser(t)
	food, bills := testutil.NewCategory(t, alice.ID, "Food"), testutil.NewCategory(t, alice.ID, "Bills")
	created, err := repo.CreateExpense(ctx, domain.ExpenseForm{
		UserID: alice.ID, CategoryID: food.ID, TransactionDt: testutil.Day(t, "2026-01-15"),
		Description: testutil.Ptr("Lunch"), Amount: decimal.RequireFromString("10"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Each step changes one field; every other field must still hold the previous value.
	steps := []struct {
		name  string
		patch domain.ExpensePatch
		check func(e *domain.Expense) bool
	}{
		{"only the amount", domain.ExpensePatch{Amount: testutil.Ptr(decimal.RequireFromString("20"))}, func(e *domain.Expense) bool {
			return e.Amount.Equal(decimal.RequireFromString("20")) && e.CategoryID == food.ID && e.Description == "Lunch" && e.TransactionDt.Equal(testutil.Day(t, "2026-01-15"))
		}},
		{"only the category", domain.ExpensePatch{CategoryID: testutil.Ptr(bills.ID)}, func(e *domain.Expense) bool {
			return e.CategoryID == bills.ID && e.CategoryName == "Bills" && e.Amount.Equal(decimal.RequireFromString("20")) && e.Description == "Lunch"
		}},
		{"only the date", domain.ExpensePatch{TransactionDt: testutil.Ptr(testutil.Day(t, "2026-01-20"))}, func(e *domain.Expense) bool {
			return e.TransactionDt.Equal(testutil.Day(t, "2026-01-20")) && e.CategoryID == bills.ID && e.Amount.Equal(decimal.RequireFromString("20"))
		}},
		{"only the description", domain.ExpensePatch{Description: testutil.Ptr("Dinner")}, func(e *domain.Expense) bool {
			return e.Description == "Dinner" && e.TransactionDt.Equal(testutil.Day(t, "2026-01-20")) && e.CategoryID == bills.ID
		}},
		{"nothing", domain.ExpensePatch{}, func(e *domain.Expense) bool {
			return e.Description == "Dinner" && e.Amount.Equal(decimal.RequireFromString("20")) && e.CategoryID == bills.ID
		}},
	}
	for _, step := range steps {
		got, err := repo.UpdateExpense(ctx, created.ID, alice.ID, step.patch)
		if err != nil {
			t.Fatalf("update %s: %v", step.name, err)
		}
		if !step.check(got) {
			t.Fatalf("update %s: got %+v, want only that field changed", step.name, got)
		}
	}
}
