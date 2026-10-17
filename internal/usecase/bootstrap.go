package usecase

import (
	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/repository"

	// Usecase
	"github.com/revirathya/revirathya-finance-core/internal/usecase/auth"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/category"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/telegrambindcode"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/user"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/usertelegramcontext"
)

// Bootstraping usecase
type Usecase struct {
	Auth                *auth.AuthUsecase
	Category            *category.CategoryUsecase
	Expense             *expense.ExpenseUsecase
	TelegramBindCode    *telegrambindcode.TelegramBindCodeUsecase
	User                *user.UserUsecase
	UserTelegramContext *usertelegramcontext.UserTelegramContextUsecase
}

func BootstrapUsecase(gVeri domain.GoogleVerifier, repo repository.Repository) Usecase {
	return Usecase{
		Auth:                auth.NewAuthUsecase(gVeri, repo.User),
		Category:            category.NewCategoryUsecase(repo.Category),
		Expense:             expense.NewExpenseUsecase(repo.Expense),
		TelegramBindCode:    telegrambindcode.NewTelegramBindCodeUsecase(repo.TelegramBindCode),
		User:                user.NewUserUsecase(repo.User),
		UserTelegramContext: usertelegramcontext.NewUserTelegramContextUsecase(repo.UserTelegramContext),
	}
}
