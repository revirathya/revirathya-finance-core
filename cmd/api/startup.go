package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	gqlHdl "github.com/99designs/gqlgen/graphql/handler"
	gqlSch "github.com/revirathya/revirathya-finance-core/gen/graphql"
	gqlDlv "github.com/revirathya/revirathya-finance-core/internal/delivery/graphql"
	ucCtg "github.com/revirathya/revirathya-finance-core/internal/usecase/category"
	ucExp "github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
	ucUsr "github.com/revirathya/revirathya-finance-core/internal/usecase/user"
	ucUtel "github.com/revirathya/revirathya-finance-core/internal/usecase/usercontext"
	"golang.org/x/sync/errgroup"

	grpcFincoreV1 "github.com/revirathya/revirathya-finance-core/gen/go/financecore/v1"
	grpcDlv "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	handlerTimeout    = 10 * time.Second // val < writeTimeout
	shutdownTimeout   = 10 * time.Second
)

func setupGraphQLServer(
	port string,
	allowedOrigins []string,
	ctgUc *ucCtg.CategoryUsecase,
	expUc *ucExp.ExpenseUsecase,
	usrUc *ucUsr.UserUsecase,
) *http.Server {
	resolver := gqlDlv.NewResolver(ctgUc, expUc, usrUc)

	gqlServer := gqlHdl.NewDefaultServer(gqlSch.NewExecutableSchema(
		gqlSch.Config{
			Resolvers: resolver,
		},
	))

	gqlServer.SetErrorPresenter(gqlDlv.ErrorPresenter)

	mux := http.NewServeMux()
	mux.Handle(
		"/graphql",
		withCORS(allowedOrigins,
			http.TimeoutHandler(
				gqlServer,
				handlerTimeout,
				`{"errors":[{"message":"request timeout"}]}`,
			),
		),
	)

	httpServer := &http.Server{
		Addr:              port,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	return httpServer
}

func setupGRPCServer(port string, enableReflection bool, ctgUc *ucCtg.CategoryUsecase, expUc *ucExp.ExpenseUsecase, utelUc *ucUtel.UserTelegramContextUsecase) (*grpc.Server, net.Listener, error) {
	ctgHan := grpcDlv.NewCategoryHandler(ctgUc)
	expHan := grpcDlv.NewExpenseHandler(expUc)
	utelHan := grpcDlv.NewUserTelegramContextHandler(utelUc)

	grpcServer := grpc.NewServer()
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

func setupServers(lis net.Listener, ctx context.Context, grpcServer *grpc.Server, gqlServer *http.Server) *errgroup.Group {
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
