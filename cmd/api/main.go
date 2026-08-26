package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/go/financecore/v1"
	grpcDlv "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
	"github.com/revirathya/revirathya-finance-core/internal/platform/database"
	pgRepo "github.com/revirathya/revirathya-finance-core/internal/repository/postgres"
	"google.golang.org/grpc"

	ucCtg "github.com/revirathya/revirathya-finance-core/internal/usecase/category"
	ucExp "github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
	ucUtel "github.com/revirathya/revirathya-finance-core/internal/usecase/usercontext"
)

func main() {
	// Get config
	dsn := os.Getenv("DATABASE_URL")
	grpcAddr := os.Getenv("GRPC_ADDR")

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

	// Setup gRPC
	ctgHan := grpcDlv.NewCategoryHandler(ctgUc)
	expHan := grpcDlv.NewExpenseHandler(expUc)
	utelHan := grpcDlv.NewUserTelegramContextHandler(utelUc)

	grpcServer := grpc.NewServer()
	financecorev1.RegisterCategoryServiceServer(grpcServer, ctgHan)
	financecorev1.RegisterExpenseServiceServer(grpcServer, expHan)
	financecorev1.RegisterUserTelegramContextServiceServer(grpcServer, utelHan)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		panic(err)
	}

	log.Printf("gRPC server listening on %s", grpcAddr)
	if err := grpcServer.Serve(lis); err != nil {
		panic(err)
	}
}
