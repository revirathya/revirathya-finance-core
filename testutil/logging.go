package testutil

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
)

// CaptureLogs installs the service's JSON logger (which adds request_id from the
// context) as the slog default for the rest of the test, writing into the returned
// buffer. The previous default is restored when the test ends.
func CaptureLogs(t *testing.T) *bytes.Buffer {
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

// LogLines decodes every JSON log line written to buf.
func LogLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	for dec.More() {
		var line map[string]any
		if err := dec.Decode(&line); err != nil {
			t.Fatalf("invalid log line: %v\n%s", err, buf.String())
		}
		lines = append(lines, line)
	}
	return lines
}
