package expense

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/domain/pagination"
)

// Usecase: Expense
type ExpenseUsecase struct {
	expRepo domain.ExpenseRepository
}

func NewExpenseUsecase(expRepo domain.ExpenseRepository) *ExpenseUsecase {
	return &ExpenseUsecase{
		expRepo: expRepo,
	}
}

func (uc *ExpenseUsecase) CreateExpense(ctx context.Context, userID uuid.UUID, categoryID uuid.UUID, transactionDt time.Time, description *string, amount decimal.Decimal) (*domain.Expense, error) {
	form := domain.ExpenseForm{
		UserID:        userID,
		CategoryID:    categoryID,
		TransactionDt: transactionDt,
		Description:   description,
		Amount:        amount,
	}
	if err := form.Validate(); err != nil {
		return nil, err
	}

	return uc.expRepo.CreateExpense(ctx, form)
}

func (uc *ExpenseUsecase) GetExpenseByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Expense, error) {
	return uc.expRepo.GetExpenseByID(ctx, id, userID)
}

func (uc *ExpenseUsecase) ListExpenses(
	ctx context.Context,
	userID uuid.UUID,
	transactionDt *time.Time,
	pageSize int,
	pageToken string,
) (*domain.ExpensePage, error) {
	// Define page size
	ps, err := pagination.SetPageSize(pageSize)
	if err != nil {
		return nil, err
	}

	// Decode token if exists
	query := domain.ExpenseListQuery{
		TransactionDt: transactionDt,
		Limit:         ps + 1,
	}
	if pageToken != "" {
		// Not the first page
		cursor, err := domain.DecodeExpenseCursor(pageToken)
		if err != nil {
			return nil, err
		}
		query.After = &cursor
	}

	expenses, err := uc.expRepo.ListExpenses(ctx, userID, query)
	if err != nil {
		return nil, err
	}

	page := &domain.ExpensePage{
		Expenses: expenses,
	}
	if len(expenses) > ps {
		page.Expenses = expenses[:ps]
		last := page.Expenses[ps-1]
		page.NextCursor = &domain.ExpenseCursor{
			TransactionDt: last.TransactionDt,
			ID:            last.ID,
		}
	}

	return page, nil
}

func (uc *ExpenseUsecase) UpdateExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch domain.ExpensePatch) (*domain.Expense, error) {
	if err := patch.Validate(); err != nil {
		return nil, err
	}

	return uc.expRepo.UpdateExpense(ctx, id, userID, patch)
}

func (uc *ExpenseUsecase) DeleteExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return uc.expRepo.DeleteExpense(ctx, id, userID)
}
