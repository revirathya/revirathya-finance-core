package main

import (
	"context"
	"os"
	"time"

	"github.com/revirathya/revirathya-finance-core/internal/platform/database"

	pgRepo "github.com/revirathya/revirathya-finance-core/internal/repository/postgres"

	ucCtg "github.com/revirathya/revirathya-finance-core/internal/usecase/category"
	ucExp "github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
	ucUtel "github.com/revirathya/revirathya-finance-core/internal/usecase/usercontext"
)

func main() {
	// Get config
	dsn := os.Getenv("DATABASE_URL")
	grpcAddr := os.Getenv("GRPC_ADDR")
	gqlAddr := os.Getenv("GRAPHQL_ADDR")

	// Get Postgres Pool
	ctx := context.Background()
	pool, err := database.NewDBPool(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	// Test Connection
	tmCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(tmCtx); err != nil {
		panic(err)
	}

	// Gen Repository
	ctgRepo := pgRepo.NewCategoryRepositoryImpl(pool)
	expRepo := pgRepo.NewExpenseRepositoryImpl(pool)
	utelRepo := pgRepo.NewUserTelegramContextRepositoryImpl(pool)

	// Gen Usecase
	ctgUc := ucCtg.NewCategoryUsecase(ctgRepo)
	expUc := ucExp.NewExpenseUsecase(expRepo)
	utelUc := ucUtel.NewUserTelegramContextUsecase(utelRepo)

	// Setup servers
	grpcServer, lis, err := setupGRPCServer(grpcAddr, ctgUc, expUc, utelUc)
	if err != nil {
		panic(err)
	}
	gqlServer := setupGraphQLServer(gqlAddr, ctgUc, expUc)

	g := setupServers(lis, ctx, grpcServer, gqlServer)
	if err = g.Wait(); err != nil {
		panic(err)
	}
}
