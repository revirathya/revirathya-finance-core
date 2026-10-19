package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Private
func Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

const readyTimeout = 2 * time.Second

func Readyz(ping func(ctx context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()

		if err := ping(ctx); err != nil {
			slog.WarnContext(ctx, "readiness check failed", "err", err)
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
