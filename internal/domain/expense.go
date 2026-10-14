package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Entity: Expense
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
	CategoryID    uuid.UUID
	TransactionDt time.Time
	Description   *string
	Amount        decimal.Decimal
}

// Repository
type ExpenseRepository interface {
	CreateExpense(ctx context.Context, userID uuid.UUID, input ExpenseForm) (*Expense, error)
	GetExpenseByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Expense, error)
	ListExpenses(ctx context.Context, userID uuid.UUID, transactionDt *time.Time) ([]Expense, error)
	UpdateExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch ExpenseForm) (*Expense, error)
	DeleteExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
