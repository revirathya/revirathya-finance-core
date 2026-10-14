package graphql

import (
	appAuth "github.com/revirathya/revirathya-finance-core/internal/auth"
	ucAuth "github.com/revirathya/revirathya-finance-core/internal/usecase/auth"
	ucCtg "github.com/revirathya/revirathya-finance-core/internal/usecase/category"
	ucExp "github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
	ucUsr "github.com/revirathya/revirathya-finance-core/internal/usecase/user"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	tokens *appAuth.TokenManager
	authUc *ucAuth.AuthUsecase
	ctgUc  *ucCtg.CategoryUsecase
	expUc  *ucExp.ExpenseUsecase
	usrUc  *ucUsr.UserUsecase
}

func NewResolver(
	tokens *appAuth.TokenManager,
	authUc *ucAuth.AuthUsecase,
	ctgUc *ucCtg.CategoryUsecase,
	expUc *ucExp.ExpenseUsecase,
	usrUc *ucUsr.UserUsecase,
) *Resolver {
	return &Resolver{
		tokens: tokens,
		authUc: authUc,
		ctgUc:  ctgUc,
		expUc:  expUc,
		usrUc:  usrUc,
	}
}
