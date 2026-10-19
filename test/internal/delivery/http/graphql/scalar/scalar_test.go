package scalar_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	// Internal
	errGql "github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/error"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/scalar"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
)

// marshal renders a gqlgen Marshaler the way it appears in the JSON response.
func marshal(m graphql.Marshaler) string {
	var buf bytes.Buffer
	m.MarshalGQL(&buf)
	return buf.String()
}

// wantInvalidInput fails unless err is an InvalidInputError, which ErrorPresenter reports as BAD_USER_INPUT
// with the message kept (a plain error would become INTERNAL).
func wantInvalidInput(t *testing.T, err error) {
	t.Helper()
	if !errors.As(err, &errGql.InvalidInputError{}) {
		t.Fatalf("got %v (%T), want an errGql.InvalidInputError", err, err)
	}
}

func TestDate(t *testing.T) {
	t.Run("marshals as a quoted YYYY-MM-DD string", func(t *testing.T) {
		if got := marshal(scalar.MarshalDate(time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC))); got != `"2026-10-17"` {
			t.Fatalf("got %s, want \"2026-10-17\"", got)
		}
	})

	t.Run("unmarshals YYYY-MM-DD", func(t *testing.T) {
		got, err := scalar.UnmarshalDate("2026-10-17")
		if err != nil || !got.Equal(time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("got %v, %v; want 2026-10-17 UTC", got, err)
		}
	})

	t.Run("round-trips", func(t *testing.T) {
		in := `"2024-02-29"`
		var s string
		_ = json.Unmarshal([]byte(in), &s)
		d, err := scalar.UnmarshalDate(s)
		if err != nil || marshal(scalar.MarshalDate(d)) != in {
			t.Fatalf("round-trip of %s failed: %v", in, err)
		}
	})

	for name, v := range map[string]any{
		"impossible day": "2026-02-30",
		"wrong layout":   "17-10-2026",
		"with a time":    "2026-10-17T10:00:00Z",
		"empty":          "",
		"a JSON number":  json.Number("20261017"),
		"an int":         20261017,
		"null":           nil,
		"a list":         []any{"2026-10-17"},
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := scalar.UnmarshalDate(v)
			wantInvalidInput(t, err)
		})
	}
}

func TestDecimal(t *testing.T) {
	t.Run("marshals as a quoted string, never a JSON number", func(t *testing.T) {
		if got := marshal(scalar.MarshalDecimal(decimal.RequireFromString("12500.50"))); got != `"12500.5"` {
			t.Fatalf("got %s, want \"12500.5\"", got)
		}
	})

	t.Run("keeps precision a float would lose", func(t *testing.T) {
		got, err := scalar.UnmarshalDecimal("0.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		sum := got.Add(decimal.RequireFromString("0.2"))
		if !sum.Equal(decimal.RequireFromString("0.3")) {
			t.Fatalf("0.1 + 0.2 = %s, want exactly 0.3", sum)
		}
	})

	for _, s := range []string{"12500.50", "-3", "0", "999999999999.99"} {
		t.Run("unmarshals "+s, func(t *testing.T) {
			got, err := scalar.UnmarshalDecimal(s)
			if err != nil || !got.Equal(decimal.RequireFromString(s)) {
				t.Fatalf("got %v, %v; want %s", got, err, s)
			}
		})
	}

	for name, v := range map[string]any{
		"text":          "twelve",
		"empty":         "",
		"a JSON number": json.Number("12500.5"),
		"a float":       12500.5,
		"an int":        12500,
		"null":          nil,
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := scalar.UnmarshalDecimal(v)
			wantInvalidInput(t, err)
		})
	}
}

func TestUUID(t *testing.T) {
	id := uuid.MustParse("9b2f6c1e-4a7d-4c3e-8f1a-2d5e6b7c8a90")

	t.Run("marshals as a quoted lowercase string", func(t *testing.T) {
		if got := marshal(scalar.MarshalUUID(id)); got != `"9b2f6c1e-4a7d-4c3e-8f1a-2d5e6b7c8a90"` {
			t.Fatalf("got %s", got)
		}
	})

	for _, s := range []string{"9b2f6c1e-4a7d-4c3e-8f1a-2d5e6b7c8a90", "9B2F6C1E-4A7D-4C3E-8F1A-2D5E6B7C8A90"} {
		t.Run("unmarshals "+s, func(t *testing.T) {
			got, err := scalar.UnmarshalUUID(s)
			if err != nil || got != id {
				t.Fatalf("got %v, %v; want %v", got, err, id)
			}
		})
	}

	for name, v := range map[string]any{
		"not a UUID":    "not-a-uuid",
		"too short":     "9b2f6c1e-4a7d-4c3e-8f1a",
		"empty":         "",
		"an int":        42,
		"a JSON number": json.Number("42"),
		"null":          nil,
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := scalar.UnmarshalUUID(v)
			wantInvalidInput(t, err)
		})
	}
}

func TestScalarErrorsThroughPresenter(t *testing.T) {
	present := errGql.NewErrorPresenter(metrics.NewMetrics(nil))
	_, dateErr := scalar.UnmarshalDate("2026-13-01")
	_, decErr := scalar.UnmarshalDecimal(12.5)
	_, uuidErr := scalar.UnmarshalUUID("not-a-uuid")

	for name, err := range map[string]error{"Date": dateErr, "Decimal": decErr, "UUID": uuidErr} {
		t.Run(name+" error is BAD_USER_INPUT with its message", func(t *testing.T) {
			got := present(context.Background(), err)
			if got.Extensions["code"] != "BAD_USER_INPUT" || got.Message != err.Error() {
				t.Fatalf("got %q %v, want BAD_USER_INPUT and message %q", got.Message, got.Extensions, err.Error())
			}
		})
	}
}
