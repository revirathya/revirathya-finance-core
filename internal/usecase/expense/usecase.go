package expense

import (
	"context"

	"github.com/google/uuid"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/domain/pagination"
)

// Usecase: Expense
type ExpenseUsecase struct {
	expense domain.ExpenseRepository
}

func NewExpenseUsecase(expense domain.ExpenseRepository) *ExpenseUsecase {
	return &ExpenseUsecase{
		expense: expense,
	}
}

func (uc *ExpenseUsecase) CreateExpense(ctx context.Context, userID uuid.UUID, form domain.ExpenseForm) (*domain.Expense, error) {
	if err := form.Validate(); err != nil {
		return nil, err
	}

	expense, err := uc.expense.CreateExpense(ctx, userID, form)
	if err != nil {
		return nil, err
	}

	if form.RequestID != nil && !sameExpense(form, *expense) {
		return nil, &errDom.ValidationError{Field: "request_id", Reason: "was already used for a different expense"}
	}
	return expense, nil
}

func (uc *ExpenseUsecase) GetExpenseByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Expense, error) {
	return uc.expense.GetExpenseByID(ctx, id, userID)
}

func (uc *ExpenseUsecase) ListExpenses(ctx context.Context, userID uuid.UUID, filter domain.ExpenseFilter, pageReq pagination.Request) (*domain.ExpensePage, error) {
	// Define page size
	size, err := pageReq.NormalizedSize()
	if err != nil {
		return nil, err
	}

	// Decode token if exists
	query := domain.ExpenseListQuery{
		TransactionDt: filter.TransactionDt,
		Limit:         size + 1,
	}
	if pageReq.Token != "" {
		// Not the first page
		cursor, err := domain.DecodeExpenseCursor(pageReq.Token)
		if err != nil {
			return nil, err
		}
		query.After = &cursor
	}

	// List expenses
	expenses, err := uc.expense.ListExpenses(ctx, userID, query)
	if err != nil {
		return nil, err
	}

	// Parse response
	items, last, more := pagination.Cut(expenses, size)
	response := &domain.ExpensePage{
		Expenses: items,
	}
	if more {
		response.NextCursor = &domain.ExpenseCursor{
			TransactionDt: last.TransactionDt,
			ID:            last.ID,
		}
	}
	return response, nil
}

func (uc *ExpenseUsecase) UpdateExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch domain.ExpensePatch) (*domain.Expense, error) {
	if err := patch.Validate(); err != nil {
		return nil, err
	}

	return uc.expense.UpdateExpense(ctx, id, userID, patch)
}

func (uc *ExpenseUsecase) DeleteExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return uc.expense.DeleteExpense(ctx, id, userID)
}
