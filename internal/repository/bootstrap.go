package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"

	// Repository
	"github.com/revirathya/revirathya-finance-core/internal/repository/postgres/category"
	"github.com/revirathya/revirathya-finance-core/internal/repository/postgres/expense"
	"github.com/revirathya/revirathya-finance-core/internal/repository/postgres/telegrambindcode"
	"github.com/revirathya/revirathya-finance-core/internal/repository/postgres/user"
	"github.com/revirathya/revirathya-finance-core/internal/repository/postgres/usertelegramcontext"
)

// Bootstraping Postgres repositories
type Repository struct {
	Category            domain.CategoryRepository
	Expense             domain.ExpenseRepository
	TelegramBindCode    domain.TelegramBindCodeRepository
	User                domain.UserRepository
	UserTelegramContext domain.UserTelegramContextRepository
}

func BootstrapRepository(pool *pgxpool.Pool) Repository {
	return Repository{
		Category:            category.NewCategoryRepositoryImpl(pool),
		Expense:             expense.NewExpenseRepositoryImpl(pool),
		TelegramBindCode:    telegrambindcode.NewTelegramBindCodeRepositoryImpl(pool),
		User:                user.NewUserRepositoryImpl(pool),
		UserTelegramContext: usertelegramcontext.NewUserTelegramContextRepositoryImpl(pool),
	}
}
