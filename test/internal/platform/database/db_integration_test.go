//go:build integration

package database_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// Internal
	errGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithPostgres(m))
}

func TestDBPoolIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("sessions run with the statement timeout and application name", func(t *testing.T) {
		var timeout, appName string
		if err := testutil.TestPool.QueryRow(ctx, "SHOW statement_timeout").Scan(&timeout); err != nil {
			t.Fatalf("show statement_timeout: %v", err)
		}
		if err := testutil.TestPool.QueryRow(ctx, "SHOW application_name").Scan(&appName); err != nil {
			t.Fatalf("show application_name: %v", err)
		}
		if timeout != "10s" || appName != "financecore" {
			t.Fatalf("statement_timeout %q, application_name %q; want 10s and financecore", timeout, appName)
		}
	})

	t.Run("a query outliving its deadline is cancelled and reported as DeadlineExceeded", func(t *testing.T) {
		short, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := testutil.TestPool.Exec(short, "SELECT pg_sleep(5)")

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("got %v, want an error wrapping context.DeadlineExceeded", err)
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Fatalf("query ran %v after its deadline, want it cancelled", elapsed)
		}
		if code := status.Code(errGrpc.ToStatus(short, "list expenses", err)); code != codes.DeadlineExceeded {
			t.Fatalf("gRPC code %v, want DeadlineExceeded", code)
		}
	})
}
