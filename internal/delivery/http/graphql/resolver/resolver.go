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
	tokenManager *auth.TokenManager
	usecase      usecase.Usecase
}

func NewResolver(tokenManager *auth.TokenManager, usecase usecase.Usecase) *Resolver {
	return &Resolver{tokenManager: tokenManager, usecase: usecase}
}
