package graphql

import (
	"context"
	"errors"
	"log/slog"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type invalidInputError struct{ err error }

func (e invalidInputError) Error() string { return e.err.Error() }
func (e invalidInputError) Unwrap() error { return e.err }

func ErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	gqlErr := graphql.DefaultErrorPresenter(ctx, err)

	if code, _ := gqlErr.Extensions["code"].(string); code == errcode.ValidationFailed || code == errcode.ParseFailed {
		return gqlErr
	}

	switch {
	case errors.Is(err, domain.ErrNotFound):
		gqlErr.Message = "resource not found"
		gqlErr.Extensions = map[string]any{"code": "NOT_FOUND"}
	case errors.As(err, &invalidInputError{}):
		gqlErr.Extensions = map[string]any{"code": "BAD_USER_INPUT"}
	case errors.Is(err, domain.ErrUnauthenticated):
		gqlErr.Message = "authentication required"
		gqlErr.Extensions = map[string]any{
			"code": "UNAUTHENTICATED",
		}
	case errors.Is(err, domain.ErrForbidden):
		gqlErr.Message = "permission denied"
		gqlErr.Extensions = map[string]any{
			"code": "FORBIDDEN",
		}
	case errors.Is(err, domain.ErrConflict):
		gqlErr.Message = "resource already exists"
		gqlErr.Extensions = map[string]any{
			"code": "CONFLICT",
		}
	default:
		gqlErr.Message = "internal error"
		gqlErr.Extensions = map[string]any{"code": "INTERNAL"}
		slog.Error("graphql request failed", "path", graphql.GetPath(ctx), "err", err)
	}

	return gqlErr
}
