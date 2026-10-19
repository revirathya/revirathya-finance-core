package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	// Internal
	dlvGql "github.com/revirathya/revirathya-finance-core/internal/delivery/graphql"
	dlvGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
)

const (
	shutdownTimeout = 10 * time.Second
)

func RunServers(ctx context.Context, grpcSrvLayer *dlvGrpc.ServerLayer, gqlServerLayer *dlvGql.ServerLayer) *errgroup.Group {
	// Init errgroup
	g, gctx := errgroup.WithContext(ctx)

	// Serve gRPC
	g.Go(func() error {
		return grpcSrvLayer.Server.Serve(grpcSrvLayer.Listener)
	})

	// Serve GraphQL
	g.Go(func() error {
		if err := gqlServerLayer.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	// Graceful shutdown
	g.Go(func() error {
		<-gctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		stopped := make(chan struct{})
		go func() {
			grpcSrvLayer.HealthServer.Shutdown()
			grpcSrvLayer.Server.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-shutdownCtx.Done():
			grpcSrvLayer.Server.Stop()
		}

		return gqlServerLayer.Server.Shutdown(shutdownCtx)
	})

	return g
}
