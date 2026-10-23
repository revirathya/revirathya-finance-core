package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/playground"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

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
	env string,
	port string,
	ping func(ctx context.Context) error,
) *ServerLayer {
	// Configure server
	mux := http.NewServeMux()

	// Set Handler
	mux.HandleFunc("GET /healthz", health.Healthz())
	mux.HandleFunc("GET /readyz", health.Readyz(ping))

	if env == "dev" {
		mux.Handle("GET /playground", playground.Handler("Finance Core", "/graphql"))
	}

	// Set Server
	handler := otelhttp.NewHandler(
		middleware.ChainMiddleware(
			mux,
			requestid.Middleware(),
			accesslog.Middleware(),
		),
		"http",
		otelhttp.WithFilter(
			func(r *http.Request) bool {
				return r.URL.Path != "/healthz" && r.URL.Path != "/readyz"
			},
		),
		otelhttp.WithSpanNameFormatter(
			func(_ string, r *http.Request) string {
				if _, pattern := mux.Handler(r); pattern != "" {
					return r.Method + " " + strings.TrimPrefix(pattern, r.Method+" ")
				}
				return r.Method
			},
		),
	)

	httpServer := &http.Server{
		Addr:              port,
		Handler:           handler,
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
