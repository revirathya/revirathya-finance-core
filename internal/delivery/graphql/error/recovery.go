package error

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"

	"github.com/99designs/gqlgen/graphql"
)

var errResolverPanic = errors.New("resolver panic")

func RecoverFunc(ctx context.Context, p any) error {
	slog.ErrorContext(
		ctx, "graphql resolver panic",
		"path", graphql.GetPath(ctx),
		"panic", p,
		"stack", string(debug.Stack()),
	)
	return errResolverPanic
}
