package resolver

import (
	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/usecase"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	tokenManager     *auth.TokenManager
	auth             authUsecase
	category         categoryUsecase
	expense          expenseUsecase
	telegramBindCode telegramBindCodeUsecase
	user             userUsecase
}

func NewResolver(
	tokenManager *auth.TokenManager,
	usecase usecase.Usecase,
) *Resolver {
	return &Resolver{
		tokenManager:     tokenManager,
		auth:             usecase.Auth,
		category:         usecase.Category,
		expense:          usecase.Expense,
		telegramBindCode: usecase.TelegramBindCode,
		user:             usecase.User,
	}
}
