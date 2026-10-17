package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"

	// Repository
	repoPg "github.com/revirathya/revirathya-finance-core/internal/repository/postgres"
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
		Category:            repoPg.NewCategoryRepositoryImpl(pool),
		Expense:             repoPg.NewExpenseRepositoryImpl(pool),
		TelegramBindCode:    repoPg.NewTelegramBindCodeRepositoryImpl(pool),
		User:                repoPg.NewUserRepositoryImpl(pool),
		UserTelegramContext: repoPg.NewUserTelegramContextRepositoryImpl(pool),
	}
}
