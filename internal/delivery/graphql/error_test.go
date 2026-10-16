package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
)

// useLogger installs the service's JSON logger (which adds request_id) for the rest of the test.
func useLogger(t *testing.T) *bytes.Buffer {
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

func TestErrorPresenterCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode string
		wantMsg  string // "" means: keep whatever the error said
	}{
		{"not found", domain.ErrNotFound, "NOT_FOUND", "resource not found"},
		{"wrapped not found", fmt.Errorf("repo: %w", domain.ErrNotFound), "NOT_FOUND", "resource not found"},
		{"unauthenticated", domain.ErrUnauthenticated, "UNAUTHENTICATED", "authentication required"},
		{"forbidden", domain.ErrForbidden, "FORBIDDEN", "permission denied"},
		{"conflict", domain.ErrConflict, "CONFLICT", "resource already exists"},
		{"domain invalid input", domain.ErrInvalidInput, "BAD_USER_INPUT", "invalid argument"},
		{"resolver input error keeps its message", invalidInputError{errors.New("invalid id UUID format")}, "BAD_USER_INPUT", "invalid id UUID format"},
		{"unknown error is masked", errors.New(`pq: relation "app.expenses" does not exist`), "INTERNAL", "internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useLogger(t)

			got := ErrorPresenter(context.Background(), tt.err)

			if code := got.Extensions["code"]; code != tt.wantCode {
				t.Fatalf("code %v, want %s", code, tt.wantCode)
			}
			if got.Message != tt.wantMsg {
				t.Fatalf("message %q, want %q", got.Message, tt.wantMsg)
			}
		})
	}
}

func TestErrorPresenterPassThrough(t *testing.T) {
	// Errors gqlgen produces before any resolver runs keep their message and code.
	for _, code := range []string{errcode.ValidationFailed, errcode.ParseFailed, "COMPLEXITY_LIMIT_EXCEEDED"} {
		t.Run(code, func(t *testing.T) {
			logs := useLogger(t)
			in := gqlerror.Errorf("Cannot query field %q on type %q.", "emial", "User")
			errcode.Set(in, code)

			got := ErrorPresenter(context.Background(), in)

			if got.Extensions["code"] != code {
				t.Fatalf("code %v, want %s", got.Extensions["code"], code)
			}
			if got.Message != in.Message {
				t.Fatalf("message %q, want the original %q", got.Message, in.Message)
			}
			if logs.Len() != 0 {
				t.Fatalf("client error was logged: %s", logs.String())
			}
		})
	}
}

func TestErrorPresenterLogging(t *testing.T) {
	ctx := logging.WithRequestID(context.Background(), "web-console-42")

	t.Run("unknown error is logged with the request ID", func(t *testing.T) {
		logs := useLogger(t)

		_ = ErrorPresenter(ctx, errors.New("connection reset"))

		var line map[string]any
		if err := json.Unmarshal(logs.Bytes(), &line); err != nil {
			t.Fatalf("want one JSON log line, got %q: %v", logs.String(), err)
		}
		if line["level"] != "ERROR" || line["msg"] != "graphql request failed" {
			t.Fatalf("unexpected line: %v", line)
		}
		if line["request_id"] != "web-console-42" {
			t.Fatalf("request_id %v, want web-console-42", line["request_id"])
		}
		if line["err"] != "connection reset" {
			t.Fatalf("err %v, want the original error in the log", line["err"])
		}
	})

	t.Run("known errors are not logged", func(t *testing.T) {
		for _, err := range []error{
			domain.ErrNotFound, domain.ErrUnauthenticated, domain.ErrForbidden,
			domain.ErrConflict, domain.ErrInvalidInput, invalidInputError{errors.New("bad")},
		} {
			logs := useLogger(t)

			_ = ErrorPresenter(ctx, err)

			if logs.Len() != 0 {
				t.Fatalf("%v was logged: %s", err, logs.String())
			}
		}
	})
}

func TestErrorPresenterRequestErrors(t *testing.T) {
	ctx := logging.WithRequestID(context.Background(), "web-console-42")

	t.Run("transport errors become a generic BAD_REQUEST and are not logged", func(t *testing.T) {
		logs := useLogger(t)
		// What gqlgen's POST transport produces for a body that isn't valid JSON.
		in := gqlerror.Errorf(`json request body could not be decoded: unexpected EOF body:{"query":"mutation { loginWithGoogle(idToken: \"super-secret-token\") { token } }"`)

		got := ErrorPresenter(ctx, in)

		if got.Extensions["code"] != "BAD_REQUEST" || got.Message != "invalid request" {
			t.Fatalf("got %q %v, want invalid request / BAD_REQUEST", got.Message, got.Extensions)
		}
		if strings.Contains(got.Message, "super-secret-token") || len(got.Path) != 0 {
			t.Fatalf("response leaks the body: %+v", got)
		}
		if logs.Len() != 0 {
			t.Fatalf("transport error was logged: %s", logs.String())
		}
	})

	t.Run("errors raised inside a resolver are not treated as request errors", func(t *testing.T) {
		useLogger(t)
		// gqlgen sets the field path on every error raised while resolving a field.
		in := gqlerror.Errorf("internal system error")
		in.Path = ast.Path{ast.PathName("categories")}

		got := ErrorPresenter(ctx, in)

		if got.Extensions["code"] != "INTERNAL" {
			t.Fatalf("code %v, want INTERNAL for a resolver-level error", got.Extensions["code"])
		}
	})
}

func TestRecoverFunc(t *testing.T) {
	ctx := logging.WithRequestID(context.Background(), "web-console-42")

	t.Run("logs the panic with stack trace and request ID", func(t *testing.T) {
		logs := useLogger(t)

		err := RecoverFunc(ctx, "boom")

		if !errors.Is(err, errResolverPanic) {
			t.Fatalf("returned %v, want errResolverPanic", err)
		}
		var line map[string]any
		if err := json.Unmarshal(logs.Bytes(), &line); err != nil {
			t.Fatalf("want one JSON log line, got %q: %v", logs.String(), err)
		}
		if line["level"] != "ERROR" || line["panic"] != "boom" || line["request_id"] != "web-console-42" {
			t.Fatalf("unexpected line: %v", line)
		}
		if stack, _ := line["stack"].(string); !strings.Contains(stack, "goroutine") {
			t.Fatalf("no stack trace in %v", line)
		}
	})

	t.Run("ErrorPresenter reports it as INTERNAL without logging it again", func(t *testing.T) {
		logs := useLogger(t)
		// How gqlgen hands the recovered error to the presenter: wrapped, with the field path.
		in := &gqlerror.Error{Err: errResolverPanic, Message: errResolverPanic.Error(), Path: ast.Path{ast.PathName("categories")}}

		got := ErrorPresenter(ctx, in)

		if got.Extensions["code"] != "INTERNAL" || got.Message != "internal error" {
			t.Fatalf("got %q %v, want internal error / INTERNAL", got.Message, got.Extensions)
		}
		if logs.Len() != 0 {
			t.Fatalf("panic was logged again: %s", logs.String())
		}
	})
}
