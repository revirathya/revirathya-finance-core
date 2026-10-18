package accesslog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"

	// Internal - gRPC
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/interceptor/accesslog"
)

const testMethod = "/financecore.v1.ExpenseService/GetExpenseByID"

// captureLogs installs the service's JSON logger (which adds request_id) as the
// default for the rest of the test, writing into the returned buffer.
func captureLogs(t *testing.T) *bytes.Buffer {
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

// logLines decodes every JSON line in buf.
func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	dec := json.NewDecoder(buf)
	for dec.More() {
		var line map[string]any
		if err := dec.Decode(&line); err != nil {
			t.Fatalf("invalid log line: %v", err)
		}
		lines = append(lines, line)
	}
	return lines
}

func call(ctx context.Context, method string, handlerErr error) (any, error) {
	handler := func(ctx context.Context, req any) (any, error) {
		if handlerErr != nil {
			return nil, handlerErr
		}
		return "response", nil
	}
	return accesslog.UnaryServerInterceptor()(ctx, "request", &grpc.UnaryServerInfo{FullMethod: method}, handler)
}

func TestUnaryServerInterceptorLevels(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantCode  string
		wantLevel string
	}{
		{"success", nil, "OK", "INFO"},
		{"invalid argument", status.Error(codes.InvalidArgument, "bad"), "InvalidArgument", "WARN"},
		{"not found", status.Error(codes.NotFound, "x"), "NotFound", "WARN"},
		{"already exists", status.Error(codes.AlreadyExists, "x"), "AlreadyExists", "WARN"},
		{"unauthenticated", status.Error(codes.Unauthenticated, "x"), "Unauthenticated", "WARN"},
		{"permission denied", status.Error(codes.PermissionDenied, "x"), "PermissionDenied", "WARN"},
		{"resource exhausted", status.Error(codes.ResourceExhausted, "x"), "ResourceExhausted", "WARN"},
		{"canceled by client", status.Error(codes.Canceled, "x"), "Canceled", "WARN"},
		{"deadline exceeded", status.Error(codes.DeadlineExceeded, "x"), "DeadlineExceeded", "WARN"},
		{"internal", status.Error(codes.Internal, "x"), "Internal", "ERROR"},
		{"unavailable", status.Error(codes.Unavailable, "x"), "Unavailable", "ERROR"},
		{"unimplemented", status.Error(codes.Unimplemented, "x"), "Unimplemented", "ERROR"},
		{"data loss", status.Error(codes.DataLoss, "x"), "DataLoss", "ERROR"},
		{"plain error is unknown", errors.New("boom"), "Unknown", "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := captureLogs(t)

			_, _ = call(context.Background(), testMethod, tt.err)

			lines := logLines(t, logs)
			if len(lines) != 1 {
				t.Fatalf("got %d log lines, want 1", len(lines))
			}
			line := lines[0]
			if line["msg"] != "grpc request" || line["method"] != testMethod {
				t.Fatalf("unexpected line: %v", line)
			}
			if line["code"] != tt.wantCode {
				t.Fatalf("code %v, want %s", line["code"], tt.wantCode)
			}
			if line["level"] != tt.wantLevel {
				t.Fatalf("level %v, want %s for code %s", line["level"], tt.wantLevel, tt.wantCode)
			}
			if _, ok := line["duration_ms"]; !ok {
				t.Fatalf("missing duration_ms: %v", line)
			}
		})
	}
}

func TestUnaryServerInterceptorBehaviour(t *testing.T) {
	t.Run("response and error are returned unchanged", func(t *testing.T) {
		captureLogs(t)

		resp, err := call(context.Background(), testMethod, nil)
		if err != nil || resp != "response" {
			t.Fatalf("got (%v, %v), want (response, nil)", resp, err)
		}

		want := status.Error(codes.NotFound, "get expense 1: not found")
		if _, err := call(context.Background(), testMethod, want); err != want {
			t.Fatalf("error %v, want the handler's error unchanged", err)
		}
	})

	t.Run("includes the request ID", func(t *testing.T) {
		logs := captureLogs(t)
		ctx := logging.WithRequestID(context.Background(), "bot-981")

		_, _ = call(ctx, testMethod, nil)

		lines := logLines(t, logs)
		if len(lines) != 1 || lines[0]["request_id"] != "bot-981" {
			t.Fatalf("want one line with request_id bot-981, got %v", lines)
		}
	})

	t.Run("health checks are not logged", func(t *testing.T) {
		logs := captureLogs(t)

		resp, err := call(context.Background(), "/grpc.health.v1.Health/Check", nil)

		if err != nil || resp != "response" {
			t.Fatalf("health check got (%v, %v), want it passed through", resp, err)
		}
		if logs.Len() != 0 {
			t.Fatalf("health check was logged: %s", logs.String())
		}
	})
}
