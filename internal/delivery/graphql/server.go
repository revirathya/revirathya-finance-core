package graphql

import (
	"context"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/vektah/gqlparser/v2/ast"

	// Generated
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	schema "github.com/revirathya/revirathya-finance-core/gen/graphql"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/usecase"

	// Internal - GraphQL
	errGql "github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/error"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/health"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/resolver"

	// Middlewares
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware/accesslog"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware/limitbody"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware/requestid"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware/withauth"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware/withcors"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware/withtimeout"
)

const (
	readHeaderTimeout  = 5 * time.Second
	readTimeout        = 10 * time.Second
	writeTimeout       = 15 * time.Second
	idleTimeout        = 60 * time.Second
	handlerTimeout     = 10 * time.Second // val < writeTimeout
	maxRequestBodySize = 1 * 1024 * 1024  // 1 MB
)

// Public
func Setup(
	port string,
	allowedOrigins []string,
	ping func(ctx context.Context) error,
	enableInstropection bool,
	tokens *auth.TokenManager,
	usecase usecase.Usecase,
) *http.Server {
	// Init resolver and server
	resolver := resolver.NewResolver(tokens, usecase)
	gqlSrv := handler.New(schema.NewExecutableSchema(
		schema.Config{
			Resolvers: resolver,
		},
	))

	// Configure server
	gqlSrv.AddTransport(transport.POST{})
	gqlSrv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	gqlSrv.SetParserTokenLimit(5000)
	gqlSrv.SetDisableSuggestion(!enableInstropection)
	gqlSrv.Use(extension.FixedComplexityLimit(200))
	if enableInstropection {
		gqlSrv.Use(extension.Introspection{})
	}
	gqlSrv.SetErrorPresenter(errGql.ErrorPresenter)
	gqlSrv.SetRecoverFunc(errGql.RecoverFunc)

	mux := http.NewServeMux()
	httpServer := &http.Server{
		Addr: port,
		Handler: middleware.ChainMiddleware(
			mux,
			requestid.Middleware(),
			accesslog.Middleware(),
		),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// Set routing
	mux.Handle(
		"POST /graphql",
		middleware.ChainMiddleware(
			gqlSrv,
			limitbody.Middleware(maxRequestBodySize),
			withcors.Middleware(allowedOrigins),
			withauth.Middleware(tokens),
			withtimeout.Middleware(handlerTimeout),
		),
	)

	// Healthcheck
	mux.HandleFunc("GET /healthz", health.Healthz())
	mux.HandleFunc("GET /readyz", health.Readyz(ping))

	return httpServer
}
