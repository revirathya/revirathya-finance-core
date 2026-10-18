package expense

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
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
	input := domain.ExpenseForm{
		CategoryID:    categoryID,
		TransactionDt: transactionDt,
		Description:   description,
		Amount:        amount,
	}
	return uc.expRepo.CreateExpense(ctx, userID, input)
}

func (uc *ExpenseUsecase) GetExpenseByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Expense, error) {
	return uc.expRepo.GetExpenseByID(ctx, id, userID)
}

func (uc *ExpenseUsecase) ListExpenses(ctx context.Context, userID uuid.UUID, transactionDt *time.Time) ([]domain.Expense, error) {
	return uc.expRepo.ListExpenses(ctx, userID, transactionDt)
}

func (uc *ExpenseUsecase) UpdateExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID, categoryID uuid.UUID, transactionDt time.Time, description *string, amount decimal.Decimal) (*domain.Expense, error) {
	patch := domain.ExpenseForm{
		CategoryID:    categoryID,
		TransactionDt: transactionDt,
		Description:   description,
		Amount:        amount,
	}
	return uc.expRepo.UpdateExpense(ctx, id, userID, patch)
}

func (uc *ExpenseUsecase) DeleteExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return uc.expRepo.DeleteExpense(ctx, id, userID)
}
