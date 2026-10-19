package error

import (
	"context"
	"errors"
	"log/slog"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/vektah/gqlparser/v2/gqlerror"

	// Internal
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
)

type InvalidInputError struct{ err error }

func (e InvalidInputError) Error() string { return e.err.Error() }
func (e InvalidInputError) Unwrap() error { return e.err }

func NewInvalidInputError(err error) InvalidInputError {
	return InvalidInputError{
		err: err,
	}
}

var passThroughCodes = map[string]bool{
	errcode.ParseFailed:         true,
	errcode.ValidationFailed:    true,
	"COMPLEXITY_LIMIT_EXCEEDED": true, // from gqlgen extension.ComplexityLimit
}

func NewErrorPresenter(metricsColl *metrics.Metrics) graphql.ErrorPresenterFunc {
	return func(ctx context.Context, err error) *gqlerror.Error {
		gqlErr := present(ctx, err)
		if code, ok := gqlErr.Extensions["code"].(string); ok {
			metricsColl.GraphQLError(code)
		}
		return gqlErr
	}
}

func present(ctx context.Context, err error) *gqlerror.Error {
	var vErr *errDom.ValidationError
	gqlErr := graphql.DefaultErrorPresenter(ctx, err)

	if code, _ := gqlErr.Extensions["code"].(string); passThroughCodes[code] {
		return gqlErr
	}

	var raw *gqlerror.Error
	if (errors.As(err, &raw)) && (raw.Err == nil && len(raw.Path) == 0) {
		return &gqlerror.Error{
			Message: "invalid request",
			Extensions: map[string]any{
				"code": "BAD_REQUEST",
			},
		}
	}

	switch {
	case errors.Is(err, errResolverPanic):
		// Already logged with its stack trace
		gqlErr.Message = "internal error"
		gqlErr.Extensions = map[string]any{
			"code": "INTERNAL",
		}
	case errors.Is(err, errDom.ErrNotFound):
		gqlErr.Message = "resource not found"
		gqlErr.Extensions = map[string]any{"code": "NOT_FOUND"}
	case errors.As(err, &vErr):
		gqlErr.Message = vErr.Error()
		gqlErr.Extensions = map[string]any{"code": "BAD_USER_INPUT"}
	case errors.As(err, &InvalidInputError{}):
		gqlErr.Extensions = map[string]any{"code": "BAD_USER_INPUT"}
	case errors.Is(err, errDom.ErrUnauthenticated):
		gqlErr.Message = "authentication required"
		gqlErr.Extensions = map[string]any{
			"code": "UNAUTHENTICATED",
		}
	case errors.Is(err, errDom.ErrForbidden):
		gqlErr.Message = "permission denied"
		gqlErr.Extensions = map[string]any{
			"code": "FORBIDDEN",
		}
	case errors.Is(err, errDom.ErrConflict):
		gqlErr.Message = "resource already exists"
		gqlErr.Extensions = map[string]any{
			"code": "CONFLICT",
		}
	case errors.Is(err, errDom.ErrInvalidInput):
		gqlErr.Message = "invalid argument"
		gqlErr.Extensions = map[string]any{
			"code": "BAD_USER_INPUT",
		}
	default:
		slog.ErrorContext(ctx, "graphql request failed", "path", graphql.GetPath(ctx), "err", err)
		gqlErr.Message = "internal error"
		gqlErr.Extensions = map[string]any{"code": "INTERNAL"}
	}

	return gqlErr
}
