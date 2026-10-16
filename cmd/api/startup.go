package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	appAuth "github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/platform/ratelimit"
	"github.com/vektah/gqlparser/v2/ast"

	ucAuth "github.com/revirathya/revirathya-finance-core/internal/usecase/auth"
	ucCtg "github.com/revirathya/revirathya-finance-core/internal/usecase/category"
	ucExp "github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
	ucTbc "github.com/revirathya/revirathya-finance-core/internal/usecase/telegrambindcode"
	ucUsr "github.com/revirathya/revirathya-finance-core/internal/usecase/user"
	ucUtel "github.com/revirathya/revirathya-finance-core/internal/usecase/usercontext"
	"golang.org/x/sync/errgroup"

	gqlHdl "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	gqlTrn "github.com/99designs/gqlgen/graphql/handler/transport"
	gqlSch "github.com/revirathya/revirathya-finance-core/gen/graphql"
	gqlDlv "github.com/revirathya/revirathya-finance-core/internal/delivery/graphql"

	grpcFincoreV1 "github.com/revirathya/revirathya-finance-core/gen/go/financecore/v1"
	grpcDlv "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
	grpcAlog "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/accesslog"
	grpcAuth "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/auth"
	grpcRec "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/recovery"
	grpcRqi "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/requestid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/reflection"
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

func setupGRPCServer(
	port string,
	enableReflection bool,
	apiKey string,
	ctgUc *ucCtg.CategoryUsecase,
	expUc *ucExp.ExpenseUsecase,
	tbcUc *ucTbc.TelegramBindCodeUsecase,
	utelUc *ucUtel.UserTelegramContextUsecase,
) (*grpc.Server, net.Listener, error) {
	bindLimiter := ratelimit.NewLimiter(10, 5*time.Minute)

	ctgHan := grpcDlv.NewCategoryHandler(ctgUc)
	expHan := grpcDlv.NewExpenseHandler(expUc)
	utelHan := grpcDlv.NewUserTelegramContextHandler(tbcUc, utelUc, bindLimiter)

	lookup := func(ctx context.Context, telegramID string) (uuid.UUID, error) {
		utel, err := utelUc.GetUserTelegramContextByTelegramID(ctx, telegramID)
		if err != nil {
			return uuid.Nil, err
		}
		return utel.UserID, nil
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcRqi.UnaryServerInterceptor(),
			grpcAlog.UnaryServerInterceptor(),
			grpcRec.UnaryServerInterceptor(),
			grpcAuth.APIKeyAuth(apiKey),
			grpcAuth.TelegramUser(
				lookup,
				grpcFincoreV1.UserTelegramContextService_BindTelegramAccount_FullMethodName,
			),
		),
	)
	grpcFincoreV1.RegisterCategoryServiceServer(grpcServer, ctgHan)
	grpcFincoreV1.RegisterExpenseServiceServer(grpcServer, expHan)
	grpcFincoreV1.RegisterUserTelegramContextServiceServer(grpcServer, utelHan)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		return nil, nil, err
	}

	if enableReflection {
		reflection.Register(grpcServer)
	}
	return grpcServer, lis, err
}

func setupServers(
	lis net.Listener,
	ctx context.Context,
	grpcServer *grpc.Server,
	healthServer *health.Server,
	gqlServer *http.Server,
) *errgroup.Group {
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return grpcServer.Serve(lis)
	})

	g.Go(func() error {
		if err := gqlServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-gctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		stopped := make(chan struct{})
		go func() {
			healthServer.Shutdown()
			grpcServer.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-shutdownCtx.Done():
			grpcServer.Stop()
		}

		return gqlServer.Shutdown(shutdownCtx)
	})

	return g
}
