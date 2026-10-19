package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain/validation"
	"github.com/shopspring/decimal"
)

// Entity
type Expense struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	CategoryID    uuid.UUID
	CategoryName  string
	TransactionDt time.Time
	Description   *string
	Amount        decimal.Decimal
}

type ExpenseForm struct {
	UserID        uuid.UUID
	CategoryID    uuid.UUID
	TransactionDt time.Time
	Description   *string
	Amount        decimal.Decimal
}

func (f *ExpenseForm) Validate() error {
	// Amount
	if err := validation.ValidateDecimal("amount", f.Amount, true, 14, 2); err != nil {
		return err
	}

	// Date
	if err := validation.ValidateDate("transaction_dt", f.TransactionDt); err != nil {
		return err
	}

	// Description
	if f.Description != nil {
		description, err := validation.ValidateString("description", *f.Description, 500, false)
		if err != nil {
			return err
		}
		f.Description = &description
	}

	return nil
}

type ExpensePatch struct {
	CategoryID    uuid.UUID
	TransactionDt time.Time
	Description   *string
	Amount        decimal.Decimal
}

func (p *ExpensePatch) Validate() error {
	// Amount
	if err := validation.ValidateDecimal("amount", p.Amount, true, 14, 2); err != nil {
		return err
	}

	// Date
	if err := validation.ValidateDate("transaction_dt", p.TransactionDt); err != nil {
		return err
	}

	// Description
	if p.Description != nil {
		description, err := validation.ValidateString("description", *p.Description, 500, false)
		if err != nil {
			return err
		}
		p.Description = &description
	}

	return nil
}

// Repository
type ExpenseRepository interface {
	CreateExpense(ctx context.Context, form ExpenseForm) (*Expense, error)
	GetExpenseByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Expense, error)
	ListExpenses(ctx context.Context, userID uuid.UUID, transactionDt *time.Time) ([]Expense, error)
	UpdateExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch ExpensePatch) (*Expense, error)
	DeleteExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
