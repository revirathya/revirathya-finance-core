package http

import (
	"context"
	"net/http"
	"time"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/usecase"

	// Internal - HTTP
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/health"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware"

	// Middlewares
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware/accesslog"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware/requestid"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
)

// Public
type ServerLayer struct {
	Server *http.Server
	Mux    *http.ServeMux
}

func Setup(
	port string,
	allowedOrigins []string,
	ping func(ctx context.Context) error,
	enableInstropection bool,
	tokens *auth.TokenManager,
	metricsColl *metrics.Metrics,
	usecase usecase.Usecase,
) *ServerLayer {
	// Configure server
	mux := http.NewServeMux()

	// Set Handler
	mux.HandleFunc("GET /healthz", health.Healthz())
	mux.HandleFunc("GET /readyz", health.Readyz(ping))

	mux.Handle("GET /metrics", metricsColl.Handler())

	// Set Server
	httpServer := &http.Server{
		Addr: port,
		Handler: middleware.ChainMiddleware(
			mux,
			requestid.Middleware(),
			accesslog.Middleware(),
		),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	return &ServerLayer{
		Server: httpServer,
		Mux:    mux,
	}
}
