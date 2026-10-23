package accesslog

import (
	"log/slog"
	"net/http"
	"time"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"

	// Internal - HTTP
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
			r = r.WithContext(auth.WithPrincipalSlot((r.Context())))

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

			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration_ms", time.Since(start).Milliseconds(),
			}

			if userID, ok := auth.SlotUserID(r.Context()); ok {
				attrs = append(attrs, "user_id", userID.String())
			}

			slog.Log(r.Context(), httpLogLevel(status), "http request", attrs...)
		})
	}
}
