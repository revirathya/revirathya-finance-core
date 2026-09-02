package graphql

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type invalidInputError struct{ err error }

func (e invalidInputError) Error() string { return e.err.Error() }
func (e invalidInputError) Unwrap() error { return e.err }

func ErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	gqlErr := graphql.DefaultErrorPresenter(ctx, err)

	switch {
	case errors.Is(err, domain.ErrNotFound):
		gqlErr.Message = "resource not found"
		gqlErr.Extensions = map[string]interface{}{"code": "NOT_FOUND"}
	case errors.As(err, &invalidInputError{}):
		gqlErr.Extensions = map[string]interface{}{"code": "BAD_USER_INPUT"}
	default:
		gqlErr.Message = "internal error"
		gqlErr.Extensions = map[string]interface{}{"code": "INTERNAL"}
	}

	return gqlErr
}
