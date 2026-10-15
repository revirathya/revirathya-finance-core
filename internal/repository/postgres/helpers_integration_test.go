//go:build integration

package postgres_test

import (
	"context"
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/repository/postgres"
	"github.com/shopspring/decimal"
)

func newUser(t *testing.T) *domain.User {
	t.Helper()

	id := uuid.NewString()
	user, err := postgres.NewUserRepositoryImpl(testPool).CreateUser(context.Background(), domain.UserForm{
		FirstName: "Test",
		Email:     id + "@test.local",
		GoogleSub: "google-" + id,
	})
	if err != nil {
		t.Fatalf("newUser: %v", err)
	}
	return user
}

func newCategory(t *testing.T, userID uuid.UUID, name string) *domain.Category {
	t.Helper()

	category, err := postgres.NewCategoryRepositoryImpl(testPool).CreateCategory(
		context.Background(), userID, domain.CategoryForm{CategoryName: name},
	)
	if err != nil {
		t.Fatalf("newCategory(%q): %v", name, err)
	}
	return category
}

// newExpense creates an expense owned by userID on date (YYYY-MM-DD) with the given amount.
func newExpense(t *testing.T, userID, categoryID uuid.UUID, date, amount string) *domain.Expense {
	t.Helper()

	expense, err := postgres.NewExpenseRepositoryImpl(testPool).CreateExpense(context.Background(), userID, domain.ExpenseForm{
		CategoryID:    categoryID,
		TransactionDt: day(t, date),
		Amount:        decimal.RequireFromString(amount),
	})
	if err != nil {
		t.Fatalf("newExpense(%s, %s): %v", date, amount, err)
	}
	return expense
}

// day parses a YYYY-MM-DD date.
func day(t *testing.T, date string) time.Time {
	t.Helper()

	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatalf("day(%q): %v", date, err)
	}
	return d
}

// newTelegramID returns a random Telegram user ID. telegram_chat_id is UNIQUE, so
// every test needs its own.
func newTelegramID() string {
	return strconv.FormatInt(rand.Int64N(1_000_000_000_000)+1, 10)
}

// newCodeHash returns a unique stand-in for a bind-code hash. The repository only
// stores and compares hashes, so any unique string up to 64 characters works.
func newCodeHash() string {
	return "hash-" + uuid.NewString()
}

func ptr[T any](v T) *T { return &v }
