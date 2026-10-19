package expense_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	// Internal
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
)

func TestExpenseValidation(t *testing.T) {
	validDt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	invalid := []struct {
		name   string
		dt     time.Time
		amount string
	}{
		{name: "zero amount", dt: validDt, amount: "0"},
		{name: "amount of 10^12", dt: validDt, amount: "1000000000000"},
		{name: "date before 2000", dt: time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC), amount: "1"},
	}
	for _, tt := range invalid {
		t.Run("create rejects "+tt.name+" before the repository", func(t *testing.T) {
			repo := &fakeExpenseRepo{}
			uc := expense.NewExpenseUsecase(repo)

			_, err := uc.CreateExpense(context.Background(), uuid.New(), uuid.New(), tt.dt, nil, decimal.RequireFromString(tt.amount))

			if !errors.Is(err, errDom.ErrInvalidInput) {
				t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
			}
			if repo.gotCreateInput.UserID != uuid.Nil {
				t.Fatal("repository was called")
			}
		})

		t.Run("update rejects "+tt.name+" before the repository", func(t *testing.T) {
			repo := &fakeExpenseRepo{}
			uc := expense.NewExpenseUsecase(repo)

			_, err := uc.UpdateExpense(context.Background(), uuid.New(), uuid.New(), uuid.New(), tt.dt, nil, decimal.RequireFromString(tt.amount))

			if !errors.Is(err, errDom.ErrInvalidInput) {
				t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
			}
			if repo.gotUpdateID != uuid.Nil {
				t.Fatal("repository was called")
			}
		})
	}

	t.Run("create and update pass the trimmed description to the repository", func(t *testing.T) {
		repo := &fakeExpenseRepo{}
		uc := expense.NewExpenseUsecase(repo)
		description := "  Lunch  "

		if _, err := uc.CreateExpense(context.Background(), uuid.New(), uuid.New(), validDt, &description, decimal.RequireFromString("10")); err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := uc.UpdateExpense(context.Background(), uuid.New(), uuid.New(), uuid.New(), validDt, &description, decimal.RequireFromString("10")); err != nil {
			t.Fatalf("update: %v", err)
		}
		if *repo.gotCreateInput.Description != "Lunch" || *repo.gotUpdatePatch.Description != "Lunch" {
			t.Fatalf("repository received %q and %q, want %q", *repo.gotCreateInput.Description, *repo.gotUpdatePatch.Description, "Lunch")
		}
		if description != "  Lunch  " {
			t.Fatalf("caller's description changed to %q", description)
		}
	})
}
