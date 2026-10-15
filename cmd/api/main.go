package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/revirathya/revirathya-finance-core/internal/config"
	"github.com/revirathya/revirathya-finance-core/internal/platform/database"
	"github.com/revirathya/revirathya-finance-core/internal/platform/google"

	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	appAuth "github.com/revirathya/revirathya-finance-core/internal/auth"

	pgRepo "github.com/revirathya/revirathya-finance-core/internal/repository/postgres"

	ucAuth "github.com/revirathya/revirathya-finance-core/internal/usecase/auth"
	ucCtg "github.com/revirathya/revirathya-finance-core/internal/usecase/category"
	ucExp "github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
	ucTbc "github.com/revirathya/revirathya-finance-core/internal/usecase/telegrambindcode"
	ucUsr "github.com/revirathya/revirathya-finance-core/internal/usecase/user"
	ucUtel "github.com/revirathya/revirathya-finance-core/internal/usecase/usercontext"
)

func main() {
	// Get config
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Get Postgres Pool
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewDBPool(ctx, cfg.DatabaseURI)
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
	tbcRepo := pgRepo.NewTelegramBindCodeRepositoryImpl(pool)
	usrRepo := pgRepo.NewUserRepositoryImpl(pool)
	utelRepo := pgRepo.NewUserTelegramContextRepositoryImpl(pool)

	// Setup Auth service
	verifier := google.NewGoogleVerifier(cfg.GoogleClientID)
	tokens := appAuth.NewTokenManager(cfg.JwtSecret, cfg.JwtTtl)

	// Gen Usecase
	authUc := ucAuth.NewAuthUsecase(verifier, usrRepo)
	ctgUc := ucCtg.NewCategoryUsecase(ctgRepo)
	expUc := ucExp.NewExpenseUsecase(expRepo)
	tbcUc := ucTbc.NewTelegramBindCodeUsecase(tbcRepo)
	usrUc := ucUsr.NewUserUsecase(usrRepo)
	utelUc := ucUtel.NewUserTelegramContextUsecase(utelRepo)

	// Setup server - gRPC
	grpcServer, lis, err := setupGRPCServer(
		cfg.GrpcPort,
		cfg.EnableGRPCReflection,
		cfg.GrpcApiKey,
		ctgUc,
		expUc,
		tbcUc,
		utelUc,
	)
	if err != nil {
		panic(err)
	}

	// Setup server - gRPC health
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	// Setup server - GraphQL
	gqlServer := setupGraphQLServer(
		cfg.GqlPort,
		cfg.AllowedOrigins,
		tokens,
		authUc,
		ctgUc,
		expUc,
		tbcUc,
		usrUc,
	)

	// Setup server - integration
	g := setupServers(lis, ctx, grpcServer, healthServer, gqlServer)
	if err = g.Wait(); err != nil {
		panic(err)
	}
}
