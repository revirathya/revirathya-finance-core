//go:build integration

package testutil

import (
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/repository"
	"github.com/revirathya/revirathya-finance-core/internal/usecase"
)

// NewUsecases wires the production repositories (on TestPool) and usecases, as cmd/main.go does, with the given
// Google verifier.
func NewUsecases(verifier domain.GoogleVerifier) usecase.Usecase {
	return usecase.BootstrapUsecase(verifier, repository.BootstrapRepository(TestPool))
}
