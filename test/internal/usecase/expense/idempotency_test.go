package expense_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
)

// TestCreateExpenseRequestID covers the check after the repository returns an expense for a request_id: a new row
// always matches its form, and a replayed one must match too, or the key was reused for a different expense.
func TestCreateExpenseRequestID(t *testing.T) {
	userID, categoryID, requestID := uuid.New(), uuid.New(), uuid.New()
	dt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	stored := func(description string, amount string) *domain.Expense {
		return &domain.Expense{ID: uuid.New(), UserID: userID, CategoryID: categoryID, TransactionDt: dt,
			Description: description, Amount: decimal.RequireFromString(amount)}
	}
	form := func(description *string, amount string) domain.ExpenseForm {
		return domain.ExpenseForm{CategoryID: categoryID, TransactionDt: dt, Description: description,
			Amount: decimal.RequireFromString(amount), RequestID: &requestID}
	}
	create := func(t *testing.T, repoReturns *domain.Expense, f domain.ExpenseForm) (*domain.Expense, error) {
		t.Helper()
		return expense.NewExpenseUsecase(&fakeExpenseRepo{createResp: repoReturns}).CreateExpense(context.Background(), userID, f)
	}

	matching := []struct {
		name   string
		stored *domain.Expense
		form   domain.ExpenseForm
	}{
		{"no description (stored as \"\")", stored("", "10"), form(nil, "10")},
		{"the same description", stored("Lunch", "10"), form(ptr("Lunch"), "10")},
		{"a description that is trimmed before comparing", stored("Lunch", "10"), form(ptr("  Lunch "), "10")},
		{"the same amount written differently (10 vs 10.00)", stored("", "10.00"), form(nil, "10")},
	}
	for _, tt := range matching {
		t.Run("returns the expense when the payload matches: "+tt.name, func(t *testing.T) {
			got, err := create(t, tt.stored, tt.form)
			if err != nil || got != tt.stored {
				t.Fatalf("got %v, %v; want the repository's expense", got, err)
			}
		})
	}

	otherCategory, otherDay := uuid.New(), dt.AddDate(0, 0, 1)
	mismatching := map[string]domain.ExpenseForm{
		"another category":    {CategoryID: otherCategory, TransactionDt: dt, Amount: decimal.RequireFromString("10"), RequestID: &requestID},
		"another date":        {CategoryID: categoryID, TransactionDt: otherDay, Amount: decimal.RequireFromString("10"), RequestID: &requestID},
		"another amount":      {CategoryID: categoryID, TransactionDt: dt, Amount: decimal.RequireFromString("11"), RequestID: &requestID},
		"another description": {CategoryID: categoryID, TransactionDt: dt, Amount: decimal.RequireFromString("10"), Description: ptr("Dinner"), RequestID: &requestID},
	}
	for name, f := range mismatching {
		t.Run("a reused request_id with "+name+" is a request_id ValidationError", func(t *testing.T) {
			_, err := create(t, stored("", "10"), f)

			var vErr *errDom.ValidationError
			if !errors.As(err, &vErr) || vErr.Field != "request_id" || !errors.Is(err, errDom.ErrInvalidInput) {
				t.Fatalf("got %v, want a ValidationError for request_id", err)
			}
		})
	}

	t.Run("without a request_id nothing is compared", func(t *testing.T) {
		f := form(nil, "10")
		f.RequestID = nil
		different := stored("Something else", "99")

		if got, err := create(t, different, f); err != nil || got != different {
			t.Fatalf("got %v, %v; want the repository's expense, unchecked", got, err)
		}
	})
}
