package accesslog

import (
	"log/slog"
	"net/http"
	"time"

	// Internal - GraphQL
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware"
)

func httpLogLevel(code int) slog.Level {
	switch {
	case code >= 500:
		return slog.LevelError
	case code >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

func Middleware() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := middleware.NewStatusRecorder(w)
			next.ServeHTTP(rec, r)

			status := rec.Code()
			if status == 0 {
				status = http.StatusOK
			}
			if status == http.StatusOK && (r.URL.Path == "/healthz" || r.URL.Path == "/readyz") {
				return
			}

			slog.Log(r.Context(),
				httpLogLevel(status), "http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
