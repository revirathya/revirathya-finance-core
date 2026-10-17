package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	appAuth "github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/vektah/gqlparser/v2/ast"

	ucAuth "github.com/revirathya/revirathya-finance-core/internal/usecase/auth"
	ucCtg "github.com/revirathya/revirathya-finance-core/internal/usecase/category"
	ucExp "github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
	ucTbc "github.com/revirathya/revirathya-finance-core/internal/usecase/telegrambindcode"
	ucUsr "github.com/revirathya/revirathya-finance-core/internal/usecase/user"

	"golang.org/x/sync/errgroup"

	gqlHdl "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	gqlTrn "github.com/99designs/gqlgen/graphql/handler/transport"
	gqlSch "github.com/revirathya/revirathya-finance-core/gen/graphql"
	gqlDlv "github.com/revirathya/revirathya-finance-core/internal/delivery/graphql"

	dlvGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
	// grpcFincoreV1 "github.com/revirathya/revirathya-finance-core/gen/go/financecore/v1"
	// grpcDlv "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
	// grpcAlog "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/accesslog"
	// grpcAuth "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/auth"
	// grpcRec "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/recovery"
	// grpcRqi "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/requestid"
	// "google.golang.org/grpc/reflection"
)

const (
	readHeaderTimeout  = 5 * time.Second
	readTimeout        = 10 * time.Second
	writeTimeout       = 15 * time.Second
	idleTimeout        = 60 * time.Second
	handlerTimeout     = 10 * time.Second // val < writeTimeout
	shutdownTimeout    = 10 * time.Second
	maxRequestBodySize = 1 * 1024 * 1024 // 1 MB
)

func setupGraphQLServer(
	port string,
	allowedOrigins []string,
	ping func(ctx context.Context) error,
	enableInstropection bool,
	tokens *appAuth.TokenManager,
	authUc *ucAuth.AuthUsecase,
	ctgUc *ucCtg.CategoryUsecase,
	expUc *ucExp.ExpenseUsecase,
	tbcUc *ucTbc.TelegramBindCodeUsecase,
	usrUc *ucUsr.UserUsecase,
) *http.Server {
	// Init resolver and server
	resolver := gqlDlv.NewResolver(tokens, authUc, ctgUc, expUc, tbcUc, usrUc)
	gqlServer := gqlHdl.New(gqlSch.NewExecutableSchema(
		gqlSch.Config{
			Resolvers: resolver,
		},
	))

	// Configure server
	gqlServer.AddTransport(gqlTrn.POST{})
	gqlServer.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	gqlServer.SetParserTokenLimit(5000)
	gqlServer.SetDisableSuggestion(!enableInstropection)
	gqlServer.Use(extension.FixedComplexityLimit(200))
	if enableInstropection {
		gqlServer.Use(extension.Introspection{})
	}
	gqlServer.SetErrorPresenter(gqlDlv.ErrorPresenter)
	gqlServer.SetRecoverFunc(gqlDlv.RecoverFunc)

	// Set routing
	mux := http.NewServeMux()
	mux.Handle(
		"/graphql",
		limitBody(
			maxRequestBodySize,
			withCORS(
				allowedOrigins,
				gqlDlv.WithAuth(tokens,
					http.TimeoutHandler(
						gqlServer,
						handlerTimeout,
						`{"errors":[{"message":"request timeout"}]}`,
					),
				),
			),
		),
	)

	// Healthcheck
	mux.HandleFunc("GET /healthz", healthz())
	mux.HandleFunc("GET /readyz", readyz(ping))

	httpServer := &http.Server{
		Addr:              port,
		Handler:           withRequestID(withAccessLog(mux)),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	return httpServer
}

// [NEW]
func runServers(ctx context.Context, grpcSrvLayer *dlvGrpc.ServerLayer, gqlServer *http.Server) *errgroup.Group {
	// Init errgroup
	g, gctx := errgroup.WithContext(ctx)

	// Serve gRPC
	g.Go(func() error {
		return grpcSrvLayer.Server.Serve(grpcSrvLayer.Listener)
	})

	// Serve GraphQL
	g.Go(func() error {
		if err := gqlServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	// Gracefull shutdown
	g.Go(func() error {
		<-gctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		stopped := make(chan struct{})
		go func() {
			grpcSrvLayer.HealthServer.Shutdown()
			grpcSrvLayer.Server.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-shutdownCtx.Done():
			grpcSrvLayer.Server.Stop()
		}

		return gqlServer.Shutdown(shutdownCtx)
	})

	return g
}
