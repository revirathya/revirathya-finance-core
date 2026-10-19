package error_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	grpcErr "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
)

func TestToStatus(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
		wantMsg  string
	}{
		{"not found", errDom.ErrNotFound, codes.NotFound, "get expense 1: not found"},
		{"wrapped not found", fmt.Errorf("repo: %w", errDom.ErrNotFound), codes.NotFound, "get expense 1: not found"},
		{"unauthenticated", errDom.ErrUnauthenticated, codes.Unauthenticated, "authentication required"},
		{"forbidden", errDom.ErrForbidden, codes.PermissionDenied, "get expense 1: permission denied"},
		{"conflict", errDom.ErrConflict, codes.AlreadyExists, "get expense 1: resource already exists"},
		{"invalid input", errDom.ErrInvalidInput, codes.InvalidArgument, "invalid argument"},
		{"validation error keeps its field message", &errDom.ValidationError{Field: "amount", Reason: "must be greater than 0"}, codes.InvalidArgument, "amount: must be greater than 0"},
		{"wrapped validation error keeps its field message", fmt.Errorf("create expense: %w", &errDom.ValidationError{Field: "amount", Reason: "must be greater than 0"}), codes.InvalidArgument, "amount: must be greater than 0"},
		{"unknown error is hidden", errors.New(`pq: relation "app.expenses" does not exist`), codes.Internal, "internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, ok := status.FromError(grpcErr.ToStatus(ctx, "get expense 1", tt.err))
			if !ok {
				t.Fatalf("not a gRPC status error")
			}
			if st.Code() != tt.wantCode {
				t.Fatalf("code: got %v, want %v", st.Code(), tt.wantCode)
			}
			if st.Message() != tt.wantMsg {
				t.Fatalf("message: got %q, want %q", st.Message(), tt.wantMsg)
			}
		})
	}
}

func TestToStatusLogging(t *testing.T) {
	// useLogger installs the service's JSON logger (which adds request_id) for the rest of the test.
	useLogger := func(t *testing.T) *bytes.Buffer {
		t.Helper()
		var buf bytes.Buffer
		logger, err := logging.NewLogger(&buf, "json", slog.LevelDebug)
		if err != nil {
			t.Fatalf("NewLogger: %v", err)
		}
		previous := slog.Default()
		slog.SetDefault(logger)
		t.Cleanup(func() { slog.SetDefault(previous) })
		return &buf
	}
	ctx := logging.WithRequestID(context.Background(), "bot-981")

	t.Run("unknown error is logged with the request ID", func(t *testing.T) {
		logs := useLogger(t)

		_ = grpcErr.ToStatus(ctx, "list expenses", errors.New("connection reset"))

		var line map[string]any
		if err := json.Unmarshal(logs.Bytes(), &line); err != nil {
			t.Fatalf("want one JSON log line, got %q: %v", logs.String(), err)
		}
		if line["level"] != "ERROR" || line["request_id"] != "bot-981" || line["op"] != "list expenses" {
			t.Fatalf("unexpected line: %v", line)
		}
		if line["err"] != "connection reset" {
			t.Fatalf("err %v, want the original error in the log", line["err"])
		}
	})

	t.Run("known domain errors are not logged", func(t *testing.T) {
		for _, err := range []error{
			errDom.ErrNotFound, errDom.ErrUnauthenticated, errDom.ErrForbidden,
			errDom.ErrConflict, errDom.ErrInvalidInput,
		} {
			logs := useLogger(t)

			_ = grpcErr.ToStatus(ctx, "get expense", err)

			if logs.Len() != 0 {
				t.Fatalf("%v was logged: %s", err, logs.String())
			}
		}
	})
}
