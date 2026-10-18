package error_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	errGql "github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/error"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestErrorPresenterCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode string
		wantMsg  string
	}{
		{"not found", domain.ErrNotFound, "NOT_FOUND", "resource not found"},
		{"wrapped not found", fmt.Errorf("repo: %w", domain.ErrNotFound), "NOT_FOUND", "resource not found"},
		{"unauthenticated", domain.ErrUnauthenticated, "UNAUTHENTICATED", "authentication required"},
		{"forbidden", domain.ErrForbidden, "FORBIDDEN", "permission denied"},
		{"conflict", domain.ErrConflict, "CONFLICT", "resource already exists"},
		{"domain invalid input", domain.ErrInvalidInput, "BAD_USER_INPUT", "invalid argument"},
		{"resolver input error keeps its message", errGql.NewInvalidInputError(errors.New("invalid id UUID format")), "BAD_USER_INPUT", "invalid id UUID format"},
		{"unknown error is masked", resolverError(errors.New(`pq: relation "app.expenses" does not exist`)), "INTERNAL", "internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.CaptureLogs(t)

			got := errGql.ErrorPresenter(context.Background(), tt.err)

			if code := got.Extensions["code"]; code != tt.wantCode {
				t.Fatalf("code %v, want %s", code, tt.wantCode)
			}
			if got.Message != tt.wantMsg {
				t.Fatalf("message %q, want %q", got.Message, tt.wantMsg)
			}
		})
	}
}

// resolverError wraps err the way gqlgen does for an error returned by a resolver:
// with the field path set.
func resolverError(err error) *gqlerror.Error {
	return &gqlerror.Error{Err: err, Message: err.Error(), Path: ast.Path{ast.PathName("expenses")}}
}

func TestErrorPresenterPassThrough(t *testing.T) {
	for _, code := range []string{errcode.ValidationFailed, errcode.ParseFailed, "COMPLEXITY_LIMIT_EXCEEDED"} {
		t.Run(code, func(t *testing.T) {
			logs := testutil.CaptureLogs(t)
			in := gqlerror.Errorf("Cannot query field %q on type %q.", "emial", "User")
			errcode.Set(in, code)

			got := errGql.ErrorPresenter(context.Background(), in)

			if got.Extensions["code"] != code || got.Message != in.Message {
				t.Fatalf("got %q %v, want the original message and code %s", got.Message, got.Extensions, code)
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
		logs := testutil.CaptureLogs(t)

		_ = errGql.ErrorPresenter(ctx, resolverError(errors.New("connection reset")))

		lines := testutil.LogLines(t, logs)
		if len(lines) != 1 {
			t.Fatalf("got %d log lines, want 1: %s", len(lines), logs.String())
		}
		line := lines[0]
		if line["level"] != "ERROR" || line["msg"] != "graphql request failed" || line["request_id"] != "web-console-42" {
			t.Fatalf("unexpected line: %v", line)
		}
		if !strings.Contains(fmt.Sprint(line["err"]), "connection reset") {
			t.Fatalf("err %v, want the original error in the log", line["err"])
		}
	})

	t.Run("known errors are not logged", func(t *testing.T) {
		for _, err := range []error{
			domain.ErrNotFound, domain.ErrUnauthenticated, domain.ErrForbidden,
			domain.ErrConflict, domain.ErrInvalidInput, errGql.NewInvalidInputError(errors.New("bad")),
		} {
			logs := testutil.CaptureLogs(t)

			_ = errGql.ErrorPresenter(ctx, err)

			if logs.Len() != 0 {
				t.Fatalf("%v was logged: %s", err, logs.String())
			}
		}
	})
}

func TestErrorPresenterRequestErrors(t *testing.T) {
	ctx := logging.WithRequestID(context.Background(), "web-console-42")

	t.Run("transport errors become a generic BAD_REQUEST and are not logged", func(t *testing.T) {
		logs := testutil.CaptureLogs(t)
		// What gqlgen's POST transport produces for a body that isn't valid JSON.
		in := gqlerror.Errorf(`json request body could not be decoded: unexpected EOF body:{"query":"mutation { loginWithGoogle(idToken: \"super-secret-token\") { token } }"`)

		got := errGql.ErrorPresenter(ctx, in)

		if got.Extensions["code"] != "BAD_REQUEST" || got.Message != "invalid request" {
			t.Fatalf("got %q %v, want invalid request / BAD_REQUEST", got.Message, got.Extensions)
		}
		if strings.Contains(got.Message, "super-secret-token") {
			t.Fatalf("response leaks the body: %+v", got)
		}
		if logs.Len() != 0 {
			t.Fatalf("transport error was logged: %s", logs.String())
		}
	})

	t.Run("errors raised inside a resolver are not treated as request errors", func(t *testing.T) {
		testutil.CaptureLogs(t)
		in := gqlerror.Errorf("internal system error")
		in.Path = ast.Path{ast.PathName("categories")}

		if got := errGql.ErrorPresenter(ctx, in); got.Extensions["code"] != "INTERNAL" {
			t.Fatalf("code %v, want INTERNAL for a resolver-level error", got.Extensions["code"])
		}
	})
}

func TestRecoverFunc(t *testing.T) {
	ctx := logging.WithRequestID(context.Background(), "web-console-42")

	t.Run("logs the panic with stack trace and request ID", func(t *testing.T) {
		logs := testutil.CaptureLogs(t)

		if err := errGql.RecoverFunc(ctx, "boom"); err == nil {
			t.Fatal("RecoverFunc returned nil, want an error")
		}

		lines := testutil.LogLines(t, logs)
		if len(lines) != 1 {
			t.Fatalf("got %d log lines, want 1", len(lines))
		}
		line := lines[0]
		if line["level"] != "ERROR" || line["panic"] != "boom" || line["request_id"] != "web-console-42" {
			t.Fatalf("unexpected line: %v", line)
		}
		if stack, _ := line["stack"].(string); !strings.Contains(stack, "goroutine") {
			t.Fatalf("no stack trace in %v", line)
		}
	})

	t.Run("ErrorPresenter reports it as INTERNAL without logging it again", func(t *testing.T) {
		testutil.CaptureLogs(t)
		recovered := errGql.RecoverFunc(ctx, "boom")
		logs := testutil.CaptureLogs(t) // only what the presenter writes

		got := errGql.ErrorPresenter(ctx, resolverError(recovered))

		if got.Extensions["code"] != "INTERNAL" || got.Message != "internal error" {
			t.Fatalf("got %q %v, want internal error / INTERNAL", got.Message, got.Extensions)
		}
		if logs.Len() != 0 {
			t.Fatalf("panic was logged again: %s", logs.String())
		}
	})
}
