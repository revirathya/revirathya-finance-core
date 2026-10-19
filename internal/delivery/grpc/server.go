package grpc

import (
	"context"
	"net"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	// Generated
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/go/financecore/v1"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
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
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/interceptor/timeout"
)

// Const
const (
	maxRecvMsgSize              = 1 << 20 // 1 MB
	maxConcurrentStreams        = 100
	connectionTimeout           = 10 * time.Second
	rpcTimeout                  = 10 * time.Second
	keepaliveMaxConnectionIdle  = 5 * time.Minute  // close connections idle this long
	keepaliveTime               = 2 * time.Hour    // ping an idle client after this
	keepaliveTimeout            = 20 * time.Second // drop it if the ping isn't answered
	keepaliveEnforcementMinTime = 30 * time.Second // clients may ping at most this often
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
	metricsColl *metrics.Metrics,
	usecase usecase.Usecase,
) (*ServerLayer, error) {
	// Set limiter
	bindLimiter := ratelimit.NewLimiter(10, 5*time.Minute)

	// Setup handler
	ctgHan := handler.NewCategoryHandler(usecase.Category)
	expHan := handler.NewExpenseHandler(usecase.Expense)
	utelHan := handler.NewUserTelegramContextHandler(usecase.TelegramBindCode, usecase.UserTelegramContext, bindLimiter)

	// Initiate server
	grpcKeepaliveParams := grpc.KeepaliveParams(
		keepalive.ServerParameters{
			MaxConnectionIdle: keepaliveMaxConnectionIdle,
			Time:              keepaliveTime,
			Timeout:           keepaliveTimeout,
		},
	)

	grpcKeepaliveEnforcementPolicy := grpc.KeepaliveEnforcementPolicy(
		keepalive.EnforcementPolicy{
			MinTime:             keepaliveEnforcementMinTime,
			PermitWithoutStream: true, // allow pings between calls
		},
	)

	srv := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxRecvMsgSize),
		grpc.MaxConcurrentStreams(maxConcurrentStreams),
		grpc.ConnectionTimeout(connectionTimeout),
		grpcKeepaliveParams,
		grpcKeepaliveEnforcementPolicy,
		grpc.ChainUnaryInterceptor(
			requestid.UnaryServerInterceptor(),
			accesslog.UnaryServerInterceptor(),
			metricsColl.Grpc.UnaryServerInterceptor(),
			recovery.UnaryServerInterceptor(),
			timeout.UnaryServerInterceptor(rpcTimeout),
			apikeyauth.UnaryServerInterceptor(apiKey),
			telegramuser.UnaryServerInterceptor(
				userLookup(usecase),
				financecorev1.UserTelegramContextService_BindTelegramAccount_FullMethodName,
			),
		),
	)

	// Startup metrics collector
	metricsColl.Grpc.InitializeMetrics(srv)

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
