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

	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	appAuth "github.com/revirathya/revirathya-finance-core/internal/auth"
	appLog "github.com/revirathya/revirathya-finance-core/internal/platform/logging"

	pgRepo "github.com/revirathya/revirathya-finance-core/internal/repository/postgres"

	ucAuth "github.com/revirathya/revirathya-finance-core/internal/usecase/auth"
	ucCtg "github.com/revirathya/revirathya-finance-core/internal/usecase/category"
	ucExp "github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
	ucTbc "github.com/revirathya/revirathya-finance-core/internal/usecase/telegrambindcode"
	ucUsr "github.com/revirathya/revirathya-finance-core/internal/usecase/user"
	ucUtel "github.com/revirathya/revirathya-finance-core/internal/usecase/usercontext"
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
		cfg.GrpcEnableReflection,
		cfg.GrpcApiKey,
		ctgUc,
		expUc,
		tbcUc,
		utelUc,
	)
	if err != nil {
		return fmt.Errorf("setup grpc server: %w", err)
	}

	// Setup server - gRPC health
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	// Setup server - GraphQL
	gqlServer := setupGraphQLServer(
		cfg.GqlPort,
		cfg.AllowedOrigins,
		pool.Ping,
		cfg.GqlEnableIntrospection,
		tokens,
		authUc,
		ctgUc,
		expUc,
		tbcUc,
		usrUc,
	)

	// Setup server - integration
	g := setupServers(lis, ctx, grpcServer, healthServer, gqlServer)
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
