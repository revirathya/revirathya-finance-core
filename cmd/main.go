package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Apps
	"github.com/revirathya/revirathya-finance-core/cmd/api"
	"github.com/revirathya/revirathya-finance-core/cmd/sub"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/config"
	dlvGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
	dlvHttp "github.com/revirathya/revirathya-finance-core/internal/delivery/http"
	httpGql "github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql"
	dlvMetrics "github.com/revirathya/revirathya-finance-core/internal/delivery/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/platform/database"
	"github.com/revirathya/revirathya-finance-core/internal/platform/google"
	logApp "github.com/revirathya/revirathya-finance-core/internal/platform/logging"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/platform/ratelimit"
	"github.com/revirathya/revirathya-finance-core/internal/platform/tracing"
	"github.com/revirathya/revirathya-finance-core/internal/repository"
	"github.com/revirathya/revirathya-finance-core/internal/usecase"
)

// version is set at build time: go build -ldflags "-X main.version=0.2.0"
var version = "dev"

func main() {
	// Early version, needs no config
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version)
		return
	}

	// Get Config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	// Early Healtcheck
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(sub.RunHealthcheck(cfg.HttpPort))
	}

	// Run
	if err := run(cfg); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	// Setup logger
	logger, err := logApp.NewLogger(os.Stdout, cfg.LogFormat, cfg.LogLevel)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)

	// Define context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Setup tracing
	tracingShutdown, err := tracing.Setup(ctx, version)
	if err != nil {
		return fmt.Errorf("setup tracing: %w", err)
	}

	defer tracing.RunShutdown(tracingShutdown)

	// Get Postgres Pool
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

	// Setup Auth service
	gVeri := google.NewGoogleVerifier(cfg.GoogleClientID)
	tokenManager := auth.NewTokenManager(cfg.JwtSecret, cfg.JwtTtl)

	// Setup metrics collector
	metricsColl := metrics.NewMetrics(pool)
	metricsColl.SetBuildInfo(version)

	// Setup write limiter, shared by gRPC and GraphQL so the budget is per user across both
	writeLimiter := ratelimit.NewLimiter(cfg.WriteRateLimit, cfg.WriteRateWindow)

	// Setup repo and usecase
	repo := repository.BootstrapRepository(pool)
	uc := usecase.BootstrapUsecase(gVeri, repo)

	// Setup server - gRPC
	grpcSrvLayer, err := dlvGrpc.Setup(
		cfg.GrpcPort,
		cfg.GrpcEnableReflection,
		cfg.GrpcApiKey,
		metricsColl,
		writeLimiter,
		uc,
	)
	if err != nil {
		return fmt.Errorf("setup grpc server: %w", err)
	}

	// Setup server - HTTP
	httpSrvLayer := dlvHttp.Setup(cfg.AppEnv, cfg.HttpPort, version, pool.Ping)

	// Configure GraphQL
	httpGql.RegisterHandler(
		httpSrvLayer,
		cfg.AllowedOrigins,
		cfg.GqlEnableIntrospection,
		tokenManager,
		metricsColl,
		writeLimiter,
		uc,
	)

	// Setup metrics server
	metricsSrvLayer := dlvMetrics.Setup(cfg.MetricsPort, metricsColl)

	// Setup server - integration
	g := api.RunServers(ctx, grpcSrvLayer, httpSrvLayer, metricsSrvLayer)
	slog.Info(
		"server started",
		"version", version,
		"grpc_addr", cfg.GrpcPort,
		"http_port", cfg.HttpPort,
		"metrics_port", cfg.MetricsPort,
	)
	if err := g.Wait(); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}
