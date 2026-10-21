package domain

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/domain/validation"
)

// Entity
type Expense struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	CategoryID    uuid.UUID
	CategoryName  string
	TransactionDt time.Time
	Description   string
	Amount        decimal.Decimal
}

// Form
type ExpenseForm struct {
	CategoryID    uuid.UUID
	TransactionDt time.Time
	Description   *string
	Amount        decimal.Decimal
	RequestID     *uuid.UUID
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

// Patch
type ExpensePatch struct {
	CategoryID    *uuid.UUID
	TransactionDt *time.Time
	Description   *string
	Amount        *decimal.Decimal
}

func (p *ExpensePatch) Validate() error {
	// Amount
	if p.Amount != nil {
		if err := validation.ValidateDecimal("amount", *p.Amount, true, 14, 2); err != nil {
			return err
		}
	}

	// Transaction Date
	if p.TransactionDt != nil {
		if err := validation.ValidateDate("transaction_dt", *p.TransactionDt); err != nil {
			return err
		}
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

// Cursor
type ExpenseCursor struct {
	TransactionDt time.Time
	ID            uuid.UUID
}

func (c ExpenseCursor) Encode() string {
	raw := c.TransactionDt.Format("2006-01-02") + "|" + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeExpenseCursor(token string) (ExpenseCursor, error) {
	invalid := &errDom.ValidationError{Field: "page_token", Reason: "is invalid"}

	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return ExpenseCursor{}, invalid
	}

	transactionDt, id, ok := strings.Cut(string(raw), "|")
	if !ok {
		return ExpenseCursor{}, invalid
	}

	timeTransactionDt, err := time.Parse("2006-01-02", transactionDt)
	if err != nil {
		return ExpenseCursor{}, invalid
	}

	uuidID, err := uuid.Parse(id)
	if err != nil {
		return ExpenseCursor{}, invalid
	}

	return ExpenseCursor{
		TransactionDt: timeTransactionDt,
		ID:            uuidID,
	}, nil
}

// Pagination
type ExpenseListQuery struct {
	TransactionDt *time.Time
	After         *ExpenseCursor
	Limit         int
}

type ExpenseFilter struct {
	TransactionDt *time.Time
}

type ExpensePage struct {
	Expenses   []Expense
	NextCursor *ExpenseCursor
}

// Repository
type ExpenseRepository interface {
	CreateExpense(ctx context.Context, userID uuid.UUID, form ExpenseForm) (*Expense, error)
	GetExpenseByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Expense, error)
	ListExpenses(ctx context.Context, userID uuid.UUID, query ExpenseListQuery) ([]Expense, error)
	UpdateExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch ExpensePatch) (*Expense, error)
	DeleteExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
