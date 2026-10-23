package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"regexp"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type requestIDKey struct{}

const RequestIDHeader = "X-Request-ID"
const maxRequestIDLen = 64

var requestIDPattern = regexp.MustCompile(`^[a-zA-Z0-9-_.]+$`)

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

func isValidRequestID(s string) bool {
	if s == "" || len(s) > maxRequestIDLen {
		return false
	}
	return requestIDPattern.MatchString(s)
}

func EnsureRequestID(incoming string) string {
	if isValidRequestID(incoming) {
		return incoming
	}
	return uuid.NewString()
}

type contextHandler struct {
	slog.Handler
}

func NewLogger(w io.Writer, format string, level slog.Level) (*slog.Logger, error) {
	opts := &slog.HandlerOptions{Level: level}

	var h slog.Handler
	switch format {
	case "json":
		h = slog.NewJSONHandler(w, opts)
	case "text":
		h = slog.NewTextHandler(w, opts)
	default:
		return nil, fmt.Errorf("unknown log format %q", format)
	}
	return slog.New(contextHandler{h}), nil
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestIDFrom(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}

	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}
