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
	tokens  *auth.TokenManager
	usecase usecase.Usecase
}

func NewResolver(tokens *auth.TokenManager, usecase usecase.Usecase) *Resolver {
	return &Resolver{tokens: tokens, usecase: usecase}
}
