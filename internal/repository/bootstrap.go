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
type repositoryImpl struct {
	repoCategory            domain.CategoryRepository
	repoExpense             domain.ExpenseRepository
	repoTelegramBindCode    domain.TelegramBindCodeRepository
	repoUser                domain.UserRepository
	repoUserTelegramContext domain.UserTelegramContextRepository
}

func BootstrapRepository(pool *pgxpool.Pool) domain.Repository {
	return &repositoryImpl{
		repoCategory:            category.NewCategoryRepositoryImpl(pool),
		repoExpense:             expense.NewExpenseRepositoryImpl(pool),
		repoTelegramBindCode:    telegrambindcode.NewTelegramBindCodeRepositoryImpl(pool),
		repoUser:                user.NewUserRepositoryImpl(pool),
		repoUserTelegramContext: usertelegramcontext.NewUserTelegramContextRepositoryImpl(pool),
	}
}

func (r *repositoryImpl) Category() domain.CategoryRepository {
	return r.repoCategory
}
func (r *repositoryImpl) Expense() domain.ExpenseRepository {
	return r.repoExpense
}
func (r *repositoryImpl) TelegramBindCode() domain.TelegramBindCodeRepository {
	return r.repoTelegramBindCode
}
func (r *repositoryImpl) User() domain.UserRepository {
	return r.repoUser
}
func (r *repositoryImpl) UserTelegramContext() domain.UserTelegramContextRepository {
	return r.repoUserTelegramContext
}
