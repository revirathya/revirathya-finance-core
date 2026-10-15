package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

func runHealthcheck() int {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get("http://localhost:8080/readyz")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

const readyTimeout = 2 * time.Second

func readyz(ping func(ctx context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()

		if err := ping(ctx); err != nil {
			slog.Warn("readiness check failed", "err", err)
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
