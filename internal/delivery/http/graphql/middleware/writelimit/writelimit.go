package writelimit

import (
	"context"

	"github.com/99designs/gqlgen/graphql"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/platform/ratelimit"
)

// FieldMiddleware counts each top-level Mutation field in fields against the caller's write budget, so a document
// with three mutations uses three writes. Requests without a principal aren't counted (the resolver rejects them).
func FieldMiddleware(limiter *ratelimit.Limiter, metricsColl *metrics.Metrics, fields ...string) graphql.FieldMiddleware {
	counted := make(map[string]bool, len(fields))
	for _, f := range fields {
		counted[f] = true
	}

	return func(ctx context.Context, next graphql.Resolver) (any, error) {
		fc := graphql.GetFieldContext(ctx)
		if fc == nil || fc.Object != "Mutation" || !counted[fc.Field.Name] {
			return next(ctx)
		}

		userID, err := auth.UserIDFrom(ctx)
		if err != nil {
			return next(ctx)
		}

		if ok, retryAfter := limiter.Take(userID.String()); !ok {
			metricsColl.RateLimited("graphql")
			return nil, &errDom.RateLimitError{RetryAfter: retryAfter}
		}
		return next(ctx)
	}
}
