package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/revirathya/revirathya-finance-core/internal/config"
	"github.com/revirathya/revirathya-finance-core/internal/platform/database"
	"github.com/revirathya/revirathya-finance-core/internal/platform/google"

	appAuth "github.com/revirathya/revirathya-finance-core/internal/auth"
	appLog "github.com/revirathya/revirathya-finance-core/internal/platform/logging"
	appRepo "github.com/revirathya/revirathya-finance-core/internal/repository"
	appUc "github.com/revirathya/revirathya-finance-core/internal/usecase"

	pgRepo "github.com/revirathya/revirathya-finance-core/internal/repository/postgres"

	ucAuth "github.com/revirathya/revirathya-finance-core/internal/usecase/auth"
	ucCtg "github.com/revirathya/revirathya-finance-core/internal/usecase/category"
	ucExp "github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
	ucTbc "github.com/revirathya/revirathya-finance-core/internal/usecase/telegrambindcode"
	ucUsr "github.com/revirathya/revirathya-finance-core/internal/usecase/user"

	dlvGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
)

func main() {
	// Early Healtcheck
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}

	// Run
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	// Get config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Setup logger
	logger, err := appLog.NewLogger(os.Stdout, cfg.LogFormat, cfg.LogLevel)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)

	// Get Postgres Pool
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewDBPool(ctx, cfg.DatabaseURI)
	if err != nil {
		return fmt.Errorf("create db pool: %w", err)
	}
	defer pool.Close()

	// Test Connection
	tmCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(tmCtx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	// Gen Repository
	ctgRepo := pgRepo.NewCategoryRepositoryImpl(pool)
	expRepo := pgRepo.NewExpenseRepositoryImpl(pool)
	tbcRepo := pgRepo.NewTelegramBindCodeRepositoryImpl(pool)
	usrRepo := pgRepo.NewUserRepositoryImpl(pool)
	// utelRepo := pgRepo.NewUserTelegramContextRepositoryImpl(pool)

	// Setup Auth service
	gVeri := google.NewGoogleVerifier(cfg.GoogleClientID)
	tokenManager := appAuth.NewTokenManager(cfg.JwtSecret, cfg.JwtTtl)

	// Gen Usecase
	authUc := ucAuth.NewAuthUsecase(gVeri, usrRepo)
	ctgUc := ucCtg.NewCategoryUsecase(ctgRepo)
	expUc := ucExp.NewExpenseUsecase(expRepo)
	tbcUc := ucTbc.NewTelegramBindCodeUsecase(tbcRepo)
	usrUc := ucUsr.NewUserUsecase(usrRepo)

	// [NEW] Setup repo and usecase
	repo := appRepo.BootstrapRepository(pool)
	uc := appUc.BootstrapUsecase(gVeri, repo)

	// Setup server - gRPC
	grpcSrvLayer, err := dlvGrpc.Setup(cfg.GrpcPort, cfg.GrpcEnableReflection, cfg.GrpcApiKey, uc)
	if err != nil {
		return fmt.Errorf("setup grpc server: %w", err)
	}

	// Setup server - GraphQL
	gqlServer := setupGraphQLServer(
		cfg.GqlPort,
		cfg.AllowedOrigins,
		pool.Ping,
		cfg.GqlEnableIntrospection,
		tokenManager,
		authUc,
		ctgUc,
		expUc,
		tbcUc,
		usrUc,
	)

	// Setup server - integration
	g := runServers(ctx, grpcSrvLayer, gqlServer)
	slog.Info(
		"server started",
		"grpc_addr", cfg.GrpcPort,
		"gql_port", cfg.GqlPort,
	)
	if err := g.Wait(); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
