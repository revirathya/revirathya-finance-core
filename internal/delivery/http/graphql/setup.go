package graphql

import (
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/vektah/gqlparser/v2/ast"

	// Generated
	schema "github.com/revirathya/revirathya-finance-core/gen/graphql"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"
	dlvHttp "github.com/revirathya/revirathya-finance-core/internal/delivery/http"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/platform/ratelimit"
	"github.com/revirathya/revirathya-finance-core/internal/usecase"

	// Internal - GraphQL
	errGql "github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/error"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/middleware/limitbody"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/middleware/withauth"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/middleware/withcors"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/middleware/withtimeout"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/middleware/writelimit"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/resolver"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/tracing"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware/metricscollector"
)

const (
	handlerTimeout     = 10 * time.Second // val < writeTimeout
	maxRequestBodySize = 1 * 1024 * 1024  // 1 MB
)

// RateLimitedMutations are the Mutation fields counted against the user's write budget. loginWithGoogle is
// unauthenticated and ok writes nothing (docs/rate-limiting.md).
var RateLimitedMutations = []string{
	"createCategory", "updateCategory", "deleteCategory",
	"createExpense", "updateExpense", "deleteExpense",
	"updateUser", "createTelegramBindCode", "unbindTelegram",
}

func RegisterHandler(
	srvLayer *dlvHttp.ServerLayer,
	allowedOrigins []string,
	enableInstropection bool,
	tokenManager *auth.TokenManager,
	metricsColl *metrics.Metrics,
	writeLimiter *ratelimit.Limiter,
	usecase usecase.Usecase,
) {
	// Init resolver and server
	resolver := resolver.NewResolver(tokenManager, usecase)
	gqlHdl := handler.New(schema.NewExecutableSchema(
		schema.Config{
			Resolvers: resolver,
		},
	))

	// Configure Handler
	gqlHdl.AddTransport(transport.POST{})
	gqlHdl.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	gqlHdl.SetParserTokenLimit(5000)
	gqlHdl.SetDisableSuggestion(!enableInstropection)
	gqlHdl.Use(extension.FixedComplexityLimit(200))
	if enableInstropection {
		gqlHdl.Use(extension.Introspection{})
	}
	gqlHdl.Use(tracing.New())
	gqlHdl.AroundFields(
		writelimit.FieldMiddleware(writeLimiter, metricsColl, RateLimitedMutations...),
	)
	gqlHdl.SetErrorPresenter(errGql.NewErrorPresenter(metricsColl))
	gqlHdl.SetRecoverFunc(errGql.RecoverFunc)

	// Register handler
	// Set routing
	srvLayer.Mux.Handle(
		"/graphql",
		middleware.ChainMiddleware(
			gqlHdl,
			metricscollector.Middleware(metricsColl, "graphql"),
			limitbody.Middleware(maxRequestBodySize),
			withcors.Middleware(allowedOrigins),
			withauth.Middleware(tokenManager),
			withtimeout.Middleware(handlerTimeout),
		),
	)
}
