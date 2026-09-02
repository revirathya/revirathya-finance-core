package graphql

import (
	ucCtg "github.com/revirathya/revirathya-finance-core/internal/usecase/category"
	ucExp "github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	ctgUc *ucCtg.CategoryUsecase
	expUc *ucExp.ExpenseUsecase
}

func NewResolver(ctgUc *ucCtg.CategoryUsecase, expUc *ucExp.ExpenseUsecase) *Resolver {
	return &Resolver{
		ctgUc: ctgUc,
		expUc: expUc,
	}
}
