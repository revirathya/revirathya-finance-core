package graphql_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/revirathya/revirathya-finance-core/internal/auth"
	dlvGql "github.com/revirathya/revirathya-finance-core/internal/delivery/graphql"
	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
	"github.com/revirathya/revirathya-finance-core/internal/usecase"
	"github.com/revirathya/revirathya-finance-core/testutil"
)

const (
	testOrigin = "http://localhost:3000"
	testSecret = "0123456789abcdef0123456789abcdef"
)

// newServer builds the production HTTP handler (all middleware, routes and gqlgen
// settings). Usecases are nil: the requests here are rejected, or answered, before a
// resolver needs one, except the panic test, which relies on that nil.
func newServer(t *testing.T, introspection bool) (http.Handler, *auth.TokenManager) {
	t.Helper()
	tokens := auth.NewTokenManager(testSecret, time.Hour)
	ping := func(ctx context.Context) error { return nil }
	layer := dlvGql.Setup(":0", []string{testOrigin}, ping, introspection, tokens, usecase.Usecase{})
	return layer.Server.Handler, tokens
}

type gqlResponse struct {
	Data   map[string]any `json:"data"`
	Errors []struct {
		Message    string         `json:"message"`
		Extensions map[string]any `json:"extensions"`
	} `json:"errors"`
}

func (r gqlResponse) code() string {
	if len(r.Errors) == 0 {
		return ""
	}
	code, _ := r.Errors[0].Extensions["code"].(string)
	return code
}

func do(t *testing.T, h http.Handler, req *http.Request) (*httptest.ResponseRecorder, gqlResponse) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var resp gqlResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec, resp
}

func queryRequest(t *testing.T, query string) *http.Request {
	t.Helper()
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// aliases repeats field n times under different aliases: "{ a0: f a1: f ... }".
func aliases(n int, field string) string {
	var b strings.Builder
	b.WriteString("{ ")
	for i := range n {
		fmt.Fprintf(&b, "a%d: %s ", i, field)
	}
	b.WriteString("}")
	return b.String()
}

func TestServerTransports(t *testing.T) {
	h, _ := newServer(t, false)

	t.Run("POST with JSON works", func(t *testing.T) {
		testutil.CaptureLogs(t)
		rec, resp := do(t, h, queryRequest(t, "{ __typename }"))

		if rec.Code != http.StatusOK || resp.Data["__typename"] != "Query" {
			t.Fatalf("status %d, data %v; want 200 with __typename Query: %s", rec.Code, resp.Data, rec.Body.String())
		}
	})

	t.Run("GET is not supported", func(t *testing.T) {
		testutil.CaptureLogs(t)
		rec, _ := do(t, h, httptest.NewRequest(http.MethodGet, "/graphql?query=%7B__typename%7D", nil))

		if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), `"__typename"`) {
			t.Fatalf("GET was answered (status %d): %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("multipart uploads are not supported", func(t *testing.T) {
		testutil.CaptureLogs(t)
		body := "--x\r\nContent-Disposition: form-data; name=\"operations\"\r\n\r\n{\"query\":\"{ __typename }\"}\r\n--x--\r\n"
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
		rec, _ := do(t, h, req)

		if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), `"__typename"`) {
			t.Fatalf("multipart was answered (status %d): %s", rec.Code, rec.Body.String())
		}
	})
}

func TestServerMiddleware(t *testing.T) {
	h, _ := newServer(t, false)

	t.Run("CORS preflight from the Web Console is answered", func(t *testing.T) {
		testutil.CaptureLogs(t)
		req := httptest.NewRequest(http.MethodOptions, "/graphql", nil)
		req.Header.Set("Origin", testOrigin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "content-type, authorization")
		rec, _ := do(t, h, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("preflight status %d, want %d; browsers would block every request", rec.Code, http.StatusNoContent)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
			t.Fatalf("Access-Control-Allow-Origin %q, want %q", got, testOrigin)
		}
	})

	t.Run("every response carries a request ID", func(t *testing.T) {
		testutil.CaptureLogs(t)
		for _, req := range []*http.Request{
			queryRequest(t, "{ __typename }"),
			httptest.NewRequest(http.MethodGet, "/healthz", nil),
		} {
			rec, _ := do(t, h, req)
			if rec.Header().Get(logging.RequestIDHeader) == "" {
				t.Fatalf("%s %s: no %s response header", req.Method, req.URL.Path, logging.RequestIDHeader)
			}
		}
	})

	t.Run("each request writes one access log line with its request ID", func(t *testing.T) {
		logs := testutil.CaptureLogs(t)
		req := queryRequest(t, "{ __typename }")
		req.Header.Set(logging.RequestIDHeader, "access-log-1")
		do(t, h, req)

		var found bool
		for _, line := range testutil.LogLines(t, logs) {
			if line["msg"] == "http request" {
				found = true
				if line["request_id"] != "access-log-1" || line["path"] != "/graphql" {
					t.Fatalf("access log line %v, want request_id access-log-1 and path /graphql", line)
				}
			}
		}
		if !found {
			t.Fatalf("no access log line was written: %s", logs.String())
		}
	})

	t.Run("health endpoints are reachable without auth", func(t *testing.T) {
		testutil.CaptureLogs(t)
		for _, path := range []string{"/healthz", "/readyz"} {
			if rec, _ := do(t, h, httptest.NewRequest(http.MethodGet, path, nil)); rec.Code != http.StatusOK {
				t.Fatalf("%s: status %d, want 200", path, rec.Code)
			}
		}
	})
}

func TestServerIntrospectionAndSuggestions(t *testing.T) {
	const schemaQuery = "{ __schema { types { name } } }"
	const typoQuery = "{ me { emial } }"

	t.Run("disabled: no schema and no suggestions", func(t *testing.T) {
		testutil.CaptureLogs(t)
		h, _ := newServer(t, false)

		rec, resp := do(t, h, queryRequest(t, schemaQuery))
		if len(resp.Errors) == 0 || strings.Contains(rec.Body.String(), "Expense") {
			t.Fatalf("schema returned with introspection off: %.300s", rec.Body.String())
		}

		rec, resp = do(t, h, queryRequest(t, typoQuery))
		if resp.code() != "GRAPHQL_VALIDATION_FAILED" || strings.Contains(rec.Body.String(), "Did you mean") {
			t.Fatalf("want a validation error without suggestions: %s", rec.Body.String())
		}
	})

	t.Run("enabled: schema and suggestions", func(t *testing.T) {
		testutil.CaptureLogs(t)
		h, _ := newServer(t, true)

		rec, resp := do(t, h, queryRequest(t, schemaQuery))
		if len(resp.Errors) != 0 || !strings.Contains(rec.Body.String(), `"Expense"`) {
			t.Fatalf("schema missing with introspection on: %.300s", rec.Body.String())
		}

		rec, _ = do(t, h, queryRequest(t, typoQuery))
		if !strings.Contains(rec.Body.String(), "Did you mean") {
			t.Fatalf("want a suggestion in development: %s", rec.Body.String())
		}
	})
}

func TestServerLimits(t *testing.T) {
	h, _ := newServer(t, false)

	t.Run("a normal query is within the complexity limit", func(t *testing.T) {
		testutil.CaptureLogs(t)
		if _, resp := do(t, h, queryRequest(t, "{ me { id firstName lastName email telegramChatId } }")); resp.code() == "COMPLEXITY_LIMIT_EXCEEDED" {
			t.Fatal("normal query rejected as too complex")
		}
	})

	t.Run("too complex query is rejected with its own code", func(t *testing.T) {
		testutil.CaptureLogs(t)
		if rec, resp := do(t, h, queryRequest(t, aliases(300, "me { id }"))); resp.code() != "COMPLEXITY_LIMIT_EXCEEDED" {
			t.Fatalf("code %q, want COMPLEXITY_LIMIT_EXCEEDED: %.300s", resp.code(), rec.Body.String())
		}
	})

	t.Run("too many tokens is rejected", func(t *testing.T) {
		testutil.CaptureLogs(t)
		rec, resp := do(t, h, queryRequest(t, aliases(3000, "__typename")))
		if len(resp.Errors) == 0 || resp.Data != nil || resp.code() == "INTERNAL" {
			t.Fatalf("oversized query not rejected cleanly (status %d): %.200s", rec.Code, rec.Body.String())
		}
	})

	t.Run("body over 1 MB is rejected with 413", func(t *testing.T) {
		testutil.CaptureLogs(t)
		body := `{"query":"{ __typename }"` + strings.Repeat(" ", 1<<20) + `}`
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if rec, _ := do(t, h, req); rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
		}
	})
}

func TestServerErrorHandling(t *testing.T) {
	t.Run("a malformed body never reaches the response or the logs", func(t *testing.T) {
		logs := testutil.CaptureLogs(t)
		h, _ := newServer(t, false)
		body := `{"query": "mutation { loginWithGoogle(idToken: \"super-secret-token\") { token } }"` // missing }
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec, resp := do(t, h, req)

		if resp.code() != "BAD_REQUEST" {
			t.Fatalf("code %q, want BAD_REQUEST: %s", resp.code(), rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "super-secret-token") || strings.Contains(logs.String(), "super-secret-token") {
			t.Fatalf("body leaked into the response or the logs:\nresponse: %s\nlogs: %s", rec.Body.String(), logs.String())
		}
	})

	t.Run("a resolver panic is INTERNAL and logged once, as JSON, with the request ID", func(t *testing.T) {
		logs := testutil.CaptureLogs(t)
		stderr := captureStderr(t)
		h, tokens := newServer(t, false)
		token, _, err := tokens.Issue(uuid.New())
		if err != nil {
			t.Fatalf("issue token: %v", err)
		}

		// Logged in, so the categories resolver runs and calls its usecase, which is
		// nil here: a real nil-pointer panic inside a resolver.
		req := queryRequest(t, "{ categories { id } }")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(logging.RequestIDHeader, "panic-test-1")
		rec, resp := do(t, h, req)

		if resp.code() != "INTERNAL" {
			t.Fatalf("code %q, want INTERNAL: %s", resp.code(), rec.Body.String())
		}
		var panics int
		for _, line := range testutil.LogLines(t, logs) {
			switch line["msg"] {
			case "graphql resolver panic":
				panics++
				if line["level"] != "ERROR" || line["request_id"] != "panic-test-1" {
					t.Fatalf("panic line %v, want ERROR with request_id panic-test-1", line)
				}
				if stack, _ := line["stack"].(string); !strings.Contains(stack, "goroutine") {
					t.Fatalf("panic line has no stack trace: %v", line)
				}
			case "graphql request failed":
				t.Fatalf("panic was logged a second time: %v", line)
			}
		}
		if panics != 1 {
			t.Fatalf("got %d panic log lines, want exactly 1", panics)
		}
		if out := stderr(); out != "" {
			t.Fatalf("panic was also printed to stderr outside the logger:\n%.300s", out)
		}
	})
}

// captureStderr redirects os.Stderr for the rest of the test. The returned function
// restores it and returns everything written in between.
func captureStderr(t *testing.T) func() string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	original := os.Stderr
	os.Stderr = w

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	var out string
	var once sync.Once
	stop := func() string {
		once.Do(func() {
			os.Stderr = original
			_ = w.Close()
			out = <-done
		})
		return out
	}
	t.Cleanup(func() { stop() })
	return stop
}
