package handler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/shopspring/decimal"
)

// Category
type categoryUsecase interface {
	GetCategoryByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Category, error)
	ListCategories(ctx context.Context, userID uuid.UUID, pageSize int, pageToken string) (*domain.CategoryPage, error)
}

// Expense
type expenseUsecase interface {
	CreateExpense(ctx context.Context, userID uuid.UUID, categoryID uuid.UUID, transactionDt time.Time, description *string, amount decimal.Decimal) (*domain.Expense, error)
	GetExpenseByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Expense, error)
	ListExpenses(ctx context.Context, userID uuid.UUID, transactionDt *time.Time, pageSize int, pageToken string) (*domain.ExpensePage, error)
	UpdateExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch domain.ExpensePatch) (*domain.Expense, error)
	DeleteExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

// Telegram Bind Code
type telegramBindCodeUsecase interface {
	BindTelegram(ctx context.Context, code string, telegramID string) (*domain.User, error)
}

// User Telegram Context
type userTelegramContextUsecase interface {
	GetUserTelegramContext(ctx context.Context, userID uuid.UUID) (*domain.UserTelegramContext, error)
	UpdateUserTelegramContext(ctx context.Context, userID uuid.UUID, patch domain.UserTelegramContextPatch) (*domain.UserTelegramContext, error)
}
