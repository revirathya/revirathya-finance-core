package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	appAuth "github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
)

const testOrigin = "http://localhost:3000"

// newTestGraphQLHandler builds the production HTTP handler (/graphql with CORS, auth,
// timeout, body limit and all gqlgen settings). Usecases are nil: every request in
// these tests is rejected, or answered, before any resolver touches them.
func newTestGraphQLHandler(t *testing.T, introspection bool) http.Handler {
	t.Helper()
	tokens := appAuth.NewTokenManager("0123456789abcdef0123456789abcdef", time.Hour)
	srv := setupGraphQLServer(":0", []string{testOrigin}, nil, introspection, tokens, nil, nil, nil, nil, nil)
	return srv.Handler
}

type gqlResponse struct {
	Data   map[string]any `json:"data"`
	Errors []struct {
		Message    string         `json:"message"`
		Extensions map[string]any `json:"extensions"`
	} `json:"errors"`
}

func postQuery(t *testing.T, h http.Handler, query string) (*httptest.ResponseRecorder, gqlResponse) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var resp gqlResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec, resp
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

func errorCode(resp gqlResponse) string {
	if len(resp.Errors) == 0 {
		return ""
	}
	code, _ := resp.Errors[0].Extensions["code"].(string)
	return code
}

func TestGraphQLTransports(t *testing.T) {
	h := newTestGraphQLHandler(t, false)

	t.Run("POST with JSON works", func(t *testing.T) {
		rec, resp := postQuery(t, h, "{ __typename }")

		if rec.Code != http.StatusOK {
			t.Fatalf("status %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		if resp.Data["__typename"] != "Query" {
			t.Fatalf("data %v, want __typename Query", resp.Data)
		}
	})

	t.Run("GET is not supported", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/graphql?query=%7B__typename%7D", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), `"__typename"`) {
			t.Fatalf("GET was answered (status %d): %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("websocket upgrade is not supported", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
		req.Header.Set("Sec-WebSocket-Protocol", "graphql-transport-ws")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code == http.StatusSwitchingProtocols || rec.Code == http.StatusOK {
			t.Fatalf("websocket upgrade was accepted (status %d)", rec.Code)
		}
	})

	t.Run("multipart uploads are not supported", func(t *testing.T) {
		body := "--x\r\nContent-Disposition: form-data; name=\"operations\"\r\n\r\n{\"query\":\"{ __typename }\"}\r\n--x--\r\n"
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), `"__typename"`) {
			t.Fatalf("multipart was answered (status %d): %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("CORS preflight still works", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/graphql", nil)
		req.Header.Set("Origin", testOrigin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status %d, want %d", rec.Code, http.StatusNoContent)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
			t.Fatalf("Access-Control-Allow-Origin %q, want %q", got, testOrigin)
		}
	})
}

func TestGraphQLIntrospection(t *testing.T) {
	const query = "{ __schema { types { name } } }"

	t.Run("disabled: schema is not returned", func(t *testing.T) {
		rec, resp := postQuery(t, newTestGraphQLHandler(t, false), query)

		if len(resp.Errors) == 0 {
			t.Fatalf("no error returned: %s", rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "Expense") {
			t.Fatalf("schema leaked: %s", rec.Body.String())
		}
	})

	t.Run("enabled: schema is returned", func(t *testing.T) {
		rec, resp := postQuery(t, newTestGraphQLHandler(t, true), query)

		if len(resp.Errors) != 0 {
			t.Fatalf("unexpected errors: %s", rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"Expense"`) {
			t.Fatalf("schema types missing: %s", rec.Body.String())
		}
	})
}

func TestGraphQLSuggestions(t *testing.T) {
	const query = "{ me { emial } }"

	t.Run("hidden when introspection is disabled", func(t *testing.T) {
		rec, resp := postQuery(t, newTestGraphQLHandler(t, false), query)

		if code := errorCode(resp); code != "GRAPHQL_VALIDATION_FAILED" {
			t.Fatalf("code %q, want GRAPHQL_VALIDATION_FAILED: %s", code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "Did you mean") {
			t.Fatalf("suggestion leaks field names: %s", rec.Body.String())
		}
	})

	t.Run("shown when introspection is enabled", func(t *testing.T) {
		rec, _ := postQuery(t, newTestGraphQLHandler(t, true), query)

		if !strings.Contains(rec.Body.String(), "Did you mean") {
			t.Fatalf("expected a suggestion in development: %s", rec.Body.String())
		}
	})
}

func TestGraphQLLimits(t *testing.T) {
	h := newTestGraphQLHandler(t, false)

	t.Run("a normal query is within the complexity limit", func(t *testing.T) {
		// Logged out, so the resolver answers UNAUTHENTICATED; the point is that the
		// complexity check let it through.
		rec, resp := postQuery(t, h, "{ me { id firstName lastName email telegramChatId } }")

		if code := errorCode(resp); code == "COMPLEXITY_LIMIT_EXCEEDED" {
			t.Fatalf("normal query rejected as too complex: %s", rec.Body.String())
		}
	})

	t.Run("too complex query is rejected with its own code", func(t *testing.T) {
		rec, resp := postQuery(t, h, aliases(300, "me { id }"))

		if code := errorCode(resp); code != "COMPLEXITY_LIMIT_EXCEEDED" {
			t.Fatalf("code %q, want COMPLEXITY_LIMIT_EXCEEDED (not INTERNAL): %s", code, rec.Body.String())
		}
	})

	t.Run("too many tokens is rejected before parsing", func(t *testing.T) {
		rec, resp := postQuery(t, h, aliases(3000, "__typename"))

		if len(resp.Errors) == 0 || resp.Data != nil {
			t.Fatalf("oversized query was executed (status %d): %.200s", rec.Code, rec.Body.String())
		}
		if code := errorCode(resp); code == "INTERNAL" {
			t.Fatalf("token limit reported as INTERNAL: %.200s", rec.Body.String())
		}
	})

	t.Run("body over the size limit is rejected", func(t *testing.T) {
		padding := strings.Repeat(" ", maxRequestBodySize+1)
		body := `{"query":"{ __typename }"` + padding + `}`
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), `"__typename"`) {
			t.Fatalf("oversized body was answered (status %d): %.200s", rec.Code, rec.Body.String())
		}
	})
}

func TestGraphQLErrorLogging(t *testing.T) {
	// useLogger installs the service's JSON logger for the rest of the test.
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

	t.Run("a malformed body never reaches the response or the logs", func(t *testing.T) {
		logs := useLogger(t)
		h := newTestGraphQLHandler(t, false)
		body := `{"query": "mutation { loginWithGoogle(idToken: \"super-secret-token\") { token } }"` // missing }

		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		var resp gqlResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if code := errorCode(resp); code != "BAD_REQUEST" {
			t.Fatalf("code %q, want BAD_REQUEST: %s", code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "super-secret-token") {
			t.Fatalf("response leaks the body: %s", rec.Body.String())
		}
		if strings.Contains(logs.String(), "super-secret-token") {
			t.Fatalf("logs contain the body: %s", logs.String())
		}
	})

	t.Run("a resolver panic is INTERNAL and logged once, as JSON, with the request ID", func(t *testing.T) {
		logs := useLogger(t)
		stderr := captureStderr(t)
		tokens := appAuth.NewTokenManager("0123456789abcdef0123456789abcdef", time.Hour)
		h := setupGraphQLServer(":0", []string{testOrigin}, nil, false, tokens, nil, nil, nil, nil, nil).Handler
		token, _, err := tokens.Issue(uuid.New())
		if err != nil {
			t.Fatalf("issue token: %v", err)
		}

		// Logged in, so the categories resolver runs and calls its usecase, which is
		// nil in this test: a real nil-pointer panic inside a resolver.
		reqBody, _ := json.Marshal(map[string]string{"query": "{ categories { id } }"})
		req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(logging.RequestIDHeader, "panic-test-1")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		var resp gqlResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if code := errorCode(resp); code != "INTERNAL" {
			t.Fatalf("code %q, want INTERNAL for a resolver panic (not BAD_REQUEST): %s", code, rec.Body.String())
		}

		var panics []map[string]any
		dec := json.NewDecoder(logs)
		for dec.More() {
			var line map[string]any
			if err := dec.Decode(&line); err != nil {
				t.Fatalf("invalid log line: %v", err)
			}
			switch line["msg"] {
			case "graphql resolver panic":
				panics = append(panics, line)
			case "graphql request failed":
				t.Fatalf("panic was logged a second time by ErrorPresenter: %v", line)
			}
		}
		if len(panics) != 1 {
			t.Fatalf("got %d panic log lines, want exactly 1", len(panics))
		}
		line := panics[0]
		if line["level"] != "ERROR" || line["request_id"] != "panic-test-1" {
			t.Fatalf("panic line %v, want ERROR with request_id panic-test-1", line)
		}
		if stack, _ := line["stack"].(string); !strings.Contains(stack, "goroutine") {
			t.Fatalf("panic line has no stack trace: %v", line)
		}
		if path := fmt.Sprint(line["path"]); !strings.Contains(path, "categories") {
			t.Fatalf("path %v, want the field that panicked (categories)", line["path"])
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
