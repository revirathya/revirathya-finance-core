package grpc

import (
	"context"
	"net"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	// Generated
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/go/financecore/v1"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/platform/ratelimit"
	"github.com/revirathya/revirathya-finance-core/internal/usecase"

	// Internal - gRPC
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/handler"

	// Interceptors
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/interceptor/accesslog"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/interceptor/apikeyauth"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/interceptor/recovery"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/interceptor/requestid"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/interceptor/telegramuser"
)

// Private
func userLookup(usecase usecase.Usecase) func(ctx context.Context, telegramID string) (uuid.UUID, error) {
	return func(ctx context.Context, telegramID string) (uuid.UUID, error) {
		utel, err := usecase.UserTelegramContext.GetUserTelegramContextByTelegramID(ctx, telegramID)
		if err != nil {
			return uuid.Nil, err
		}
		return utel.UserID, nil
	}
}

// Public
type ServerLayer struct {
	Server       *grpc.Server
	HealthServer *health.Server
	Listener     net.Listener
}

func Setup(
	port string,
	enableReflection bool,
	apiKey string,
	usecase usecase.Usecase,
) (*ServerLayer, error) {
	// Set limiter
	bindLimiter := ratelimit.NewLimiter(10, 5*time.Minute)

	// Setup handler
	ctgHan := handler.NewCategoryHandler(usecase.Category)
	expHan := handler.NewExpenseHandler(usecase.Expense)
	utelHan := handler.NewUserTelegramContextHandler(usecase.TelegramBindCode, usecase.UserTelegramContext, bindLimiter)

	// Initiate server
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			requestid.UnaryServerInterceptor(),
			accesslog.UnaryServerInterceptor(),
			recovery.UnaryServerInterceptor(),
			apikeyauth.UnaryServerInterceptor(apiKey),
			telegramuser.UnaryServerInterceptor(
				userLookup(usecase),
				financecorev1.UserTelegramContextService_BindTelegramAccount_FullMethodName,
			),
		),
	)

	// Register service server
	financecorev1.RegisterCategoryServiceServer(srv, ctgHan)
	financecorev1.RegisterExpenseServiceServer(srv, expHan)
	financecorev1.RegisterUserTelegramContextServiceServer(srv, utelHan)

	// Setup listener
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return nil, err
	}

	if enableReflection {
		reflection.Register(srv)
	}

	// Setup health server
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)

	return &ServerLayer{
		Server:       srv,
		HealthServer: healthSrv,
		Listener:     lis,
	}, nil
}
