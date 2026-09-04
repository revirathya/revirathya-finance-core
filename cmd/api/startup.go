package main

import (
	"context"
	"net"
	"net/http"

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
)

func setupGraphQLServer(
	addr string,
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
	mux.Handle("/graphql", withCORS(allowedOrigins, gqlServer))

	if addr == "" {
		addr = ":8080"
	}

	httpServer := &http.Server{Addr: addr, Handler: mux}
	return httpServer
}

func setupGRPCServer(addr string, ctgUc *ucCtg.CategoryUsecase, expUc *ucExp.ExpenseUsecase, utelUc *ucUtel.UserTelegramContextUsecase) (*grpc.Server, net.Listener, error) {
	ctgHan := grpcDlv.NewCategoryHandler(ctgUc)
	expHan := grpcDlv.NewExpenseHandler(expUc)
	utelHan := grpcDlv.NewUserTelegramContextHandler(utelUc)

	grpcServer := grpc.NewServer()
	grpcFincoreV1.RegisterCategoryServiceServer(grpcServer, ctgHan)
	grpcFincoreV1.RegisterExpenseServiceServer(grpcServer, expHan)
	grpcFincoreV1.RegisterUserTelegramContextServiceServer(grpcServer, utelHan)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	return grpcServer, lis, err
}

func setupServers(lis net.Listener, ctx context.Context, grpcServer *grpc.Server, gqlServer *http.Server) *errgroup.Group {
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return grpcServer.Serve(lis)
	})

	g.Go(func() error {
		return gqlServer.ListenAndServe()
	})

	g.Go(func() error {
		<-gctx.Done()
		grpcServer.GracefulStop()
		return gqlServer.Shutdown(context.Background())
	})

	return g
}
