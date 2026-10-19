package metrics

import (
	"net/http"
	"time"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
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
	metricsColl *metrics.Metrics,
) *ServerLayer {
	// Configure server
	mux := http.NewServeMux()

	// Set Handler
	mux.Handle("GET /metrics", metricsColl.Handler())

	// Set Server
	httpServer := &http.Server{
		Addr:              port,
		Handler:           mux,
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
