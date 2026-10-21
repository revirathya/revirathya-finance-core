package resolver

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/shopspring/decimal"
)

// Auth
type authUsecase interface {
	LoginWithGoogle(ctx context.Context, idToken string) (*domain.User, error)
}

// Category
type categoryUsecase interface {
	CreateCategory(ctx context.Context, userID uuid.UUID, categoryName string) (*domain.Category, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Category, error)
	ListCategories(ctx context.Context, userID uuid.UUID, pageSize int, pageToken string) (*domain.CategoryPage, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch domain.CategoryPatch) (*domain.Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
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
	CreateBindCode(ctx context.Context, userID uuid.UUID) (string, time.Time, error)
	UnbindTelegram(ctx context.Context, userID uuid.UUID) error
}

// User
type userUsecase interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, patch domain.UserPatch) (*domain.User, error)
}
