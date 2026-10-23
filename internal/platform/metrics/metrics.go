package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	// Prometheus
	grpcProm "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	// Internal - HTTP
)

type Metrics struct {
	Registry *prometheus.Registry
	Grpc     *grpcProm.ServerMetrics

	httpRequests  *prometheus.CounterVec
	httpDuration  *prometheus.HistogramVec
	graphqlErrors *prometheus.CounterVec
	rateLimited   *prometheus.CounterVec
}

// Private
func registerPoolMetrics(reg *prometheus.Registry, pool *pgxpool.Pool) {
	gauge := func(name string, help string, f func(*pgxpool.Stat) float64) prometheus.Collector {
		return prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name: name,
				Help: help,
			},
			func() float64 { return f(pool.Stat()) },
		)
	}

	counter := func(name string, help string, f func(*pgxpool.Stat) float64) prometheus.Collector {
		return prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Name: name,
				Help: help,
			},
			func() float64 { return f(pool.Stat()) },
		)
	}

	reg.MustRegister(
		gauge("db_pool_connections_max", "Max pool size.", func(s *pgxpool.Stat) float64 { return float64(s.MaxConns()) }),
		gauge("db_pool_connections_total", "Open connections.", func(s *pgxpool.Stat) float64 { return float64(s.TotalConns()) }),
		gauge("db_pool_connections_acquired", "Connection in use.", func(s *pgxpool.Stat) float64 { return float64(s.AcquiredConns()) }),
		gauge("db_pool_connections_idle", "Idle connections.", func(s *pgxpool.Stat) float64 { return float64(s.IdleConns()) }),
		counter("db_pool_acquires_total", "Connection acquires.", func(s *pgxpool.Stat) float64 { return float64(s.AcquireCount()) }),
		counter("db_pool_acquire_wait_total", "Acquires that had to wait for a connection.", func(s *pgxpool.Stat) float64 { return float64(s.EmptyAcquireCount()) }),
		counter("db_pool_acquire_wait_seconds_total", "Total time spent acquiring connections.", func(s *pgxpool.Stat) float64 { return s.AcquireDuration().Seconds() }),
	)
}

// Public
func NewMetrics(pool *pgxpool.Pool) *Metrics {
	// Init registry
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	// Init grpc server metrics
	grpcSrvMet := grpcProm.NewServerMetrics(
		grpcProm.WithServerHandlingTimeHistogram(),
	)

	// Init http requests collectors - counter
	httpRequestsColl := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by handler and status code.",
		},
		[]string{"handler", "code"},
	)

	// Init http duration collectors - histogram
	httpDurationColl := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"handler"},
	)

	// Init graphql errors collectors - counter
	graphqlErrorsColl := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "graphql_errors_total",
			Help: "GraphQL errors by code (e.g. BAD_USER_INPUT, INTERNAL)",
		},
		[]string{"code"},
	)

	// Init rate limit collectors - counter
	rateLimitedColl := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limited_total",
			Help: "Writes rejected by the per-user write limit, by API (grpc, graphql)",
		},
		[]string{"api"},
	)

	// Register metrics
	m := &Metrics{
		Registry:      reg,
		Grpc:          grpcSrvMet,
		httpRequests:  httpRequestsColl,
		httpDuration:  httpDurationColl,
		graphqlErrors: graphqlErrorsColl,
		rateLimited:   rateLimitedColl,
	}

	reg.MustRegister(m.Grpc, m.httpRequests, m.httpDuration, m.graphqlErrors, m.rateLimited)
	if pool != nil {
		registerPoolMetrics(reg, pool)
	}

	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

// GraphQLError counts one GraphQL error by its extensions.code
func (m *Metrics) GraphQLError(code string) {
	m.graphqlErrors.WithLabelValues(code).Inc()
}

// platform/metrics: no net/http middleware here any more
func (m *Metrics) ObserveHTTP(handler string, code int, d time.Duration) {
	m.httpRequests.WithLabelValues(handler, strconv.Itoa(code)).Inc()
	m.httpDuration.WithLabelValues(handler).Observe(d.Seconds())
}

// RateLimited counts one write rejected by the write limit; api is "grpc" or "graphql"
func (m *Metrics) RateLimited(api string) {
	m.rateLimited.WithLabelValues(api).Inc()
}
