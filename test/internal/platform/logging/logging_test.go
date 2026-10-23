package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
)

// newJSONLogger returns a JSON logger at level that writes into the returned buffer.
func newJSONLogger(t *testing.T, level slog.Level) (*slog.Logger, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	logger, err := logging.NewLogger(&buf, "json", level)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	return logger, &buf
}

// decode parses the single JSON log line in buf.
func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log line is not valid JSON: %v\n%s", err, buf.String())
	}
	return line
}

func TestRequestIDContext(t *testing.T) {
	t.Run("stored ID is returned", func(t *testing.T) {
		ctx := logging.WithRequestID(context.Background(), "abc")
		if got := logging.RequestIDFrom(ctx); got != "abc" {
			t.Fatalf("got %q, want %q", got, "abc")
		}
	})

	t.Run("no ID returns empty string", func(t *testing.T) {
		if got := logging.RequestIDFrom(context.Background()); got != "" {
			t.Fatalf("got %q, want empty", got)
		}
	})
}

func TestNewLogger(t *testing.T) {
	ctxWithID := logging.WithRequestID(context.Background(), "req-123")

	t.Run("adds the request ID from the context", func(t *testing.T) {
		logger, buf := newJSONLogger(t, slog.LevelInfo)

		logger.InfoContext(ctxWithID, "hello", "k", "v")

		line := decode(t, buf)
		if line["request_id"] != "req-123" {
			t.Fatalf("request_id %v, want %q: %s", line["request_id"], "req-123", buf.String())
		}
		if line["msg"] != "hello" || line["k"] != "v" || line["level"] != "INFO" {
			t.Fatalf("unexpected line: %s", buf.String())
		}
	})

	t.Run("no request ID without one in the context", func(t *testing.T) {
		logger, buf := newJSONLogger(t, slog.LevelInfo)

		logger.InfoContext(context.Background(), "hello")

		if _, ok := decode(t, buf)["request_id"]; ok {
			t.Fatalf("unexpected request_id: %s", buf.String())
		}
	})

	t.Run("methods without a context have no request ID", func(t *testing.T) {
		logger, buf := newJSONLogger(t, slog.LevelInfo)

		logger.Info("hello")

		if _, ok := decode(t, buf)["request_id"]; ok {
			t.Fatalf("unexpected request_id: %s", buf.String())
		}
	})

	t.Run("loggers derived with With keep the request ID", func(t *testing.T) {
		logger, buf := newJSONLogger(t, slog.LevelInfo)

		logger.With("component", "grpc").InfoContext(ctxWithID, "hello")

		line := decode(t, buf)
		if line["request_id"] != "req-123" {
			t.Fatalf("request_id lost after With: %s", buf.String())
		}
		if line["component"] != "grpc" {
			t.Fatalf("attribute from With lost: %s", buf.String())
		}
	})

	t.Run("loggers derived with WithGroup keep the request ID", func(t *testing.T) {
		logger, buf := newJSONLogger(t, slog.LevelInfo)

		logger.WithGroup("http").InfoContext(ctxWithID, "hello", "status", 200)

		if !strings.Contains(buf.String(), `"request_id":"req-123"`) {
			t.Fatalf("request_id lost after WithGroup: %s", buf.String())
		}
	})

	t.Run("records below the level are dropped", func(t *testing.T) {
		logger, buf := newJSONLogger(t, slog.LevelInfo)

		logger.DebugContext(ctxWithID, "noisy")

		if buf.Len() != 0 {
			t.Fatalf("debug line written at info level: %s", buf.String())
		}
	})

	t.Run("text format", func(t *testing.T) {
		var buf bytes.Buffer
		logger, err := logging.NewLogger(&buf, "text", slog.LevelInfo)
		if err != nil {
			t.Fatalf("NewLogger: %v", err)
		}

		logger.InfoContext(ctxWithID, "hello")

		if !strings.Contains(buf.String(), "request_id=req-123") || !strings.Contains(buf.String(), "msg=hello") {
			t.Fatalf("unexpected text line: %s", buf.String())
		}
	})

	t.Run("unknown format is an error", func(t *testing.T) {
		if _, err := logging.NewLogger(&bytes.Buffer{}, "xml", slog.LevelInfo); err == nil {
			t.Fatal("expected an error for format xml")
		}
	})
}

func TestEnsureRequestID(t *testing.T) {
	t.Run("valid incoming IDs are kept", func(t *testing.T) {
		for _, id := range []string{
			"a",
			"abc-123",
			"my_request.42",
			"9f1c2b7e-4c1d-4e0b-9a3e-2f6d8c1b5a70",
			strings.Repeat("x", 64),
		} {
			if got := logging.EnsureRequestID(id); got != id {
				t.Fatalf("EnsureRequestID(%q) = %q, want it kept", id, got)
			}
		}
	})

	t.Run("invalid incoming IDs are replaced with a new UUID", func(t *testing.T) {
		for _, id := range []string{
			"",
			strings.Repeat("x", 65),
			"abc\nlevel=ERROR msg=forged",
			"has space",
			"tab\there",
			"quote\"d",
			"ünïcode",
			"semi;colon",
		} {
			got := logging.EnsureRequestID(id)
			if got == id {
				t.Fatalf("EnsureRequestID(%q) kept an invalid ID", id)
			}
			if _, err := uuid.Parse(got); err != nil {
				t.Fatalf("EnsureRequestID(%q) = %q, want a UUID", id, got)
			}
		}
	})

	t.Run("each generated ID is new", func(t *testing.T) {
		if a, b := logging.EnsureRequestID(""), logging.EnsureRequestID(""); a == b {
			t.Fatalf("two generated IDs are equal: %q", a)
		}
	})

	t.Run("a generated ID is itself valid", func(t *testing.T) {
		id := logging.EnsureRequestID("")
		if again := logging.EnsureRequestID(id); again != id {
			t.Fatalf("generated ID %q is not accepted back (got %q)", id, again)
		}
	})
}

func TestTraceIDs(t *testing.T) {
	t.Run("a span in the context adds trace_id and span_id", func(t *testing.T) {
		traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
		spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
		ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		}))
		logger, buf := newJSONLogger(t, slog.LevelInfo)

		logger.InfoContext(ctx, "hello")

		line := decode(t, buf)
		if line["trace_id"] != traceID.String() || line["span_id"] != spanID.String() {
			t.Fatalf("trace_id %v, span_id %v; want %s and %s", line["trace_id"], line["span_id"], traceID, spanID)
		}
	})

	t.Run("no span adds neither", func(t *testing.T) {
		logger, buf := newJSONLogger(t, slog.LevelInfo)

		logger.InfoContext(context.Background(), "hello")

		line := decode(t, buf)
		if _, ok := line["trace_id"]; ok {
			t.Fatalf("trace_id %v logged without a span", line["trace_id"])
		}
		if _, ok := line["span_id"]; ok {
			t.Fatalf("span_id %v logged without a span", line["span_id"])
		}
	})
}
