package domain_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
)

func TestExpenseCursor(t *testing.T) {
	t.Run("encode and decode round-trip", func(t *testing.T) {
		want := domain.ExpenseCursor{TransactionDt: time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC), ID: uuid.New()}

		got, err := domain.DecodeExpenseCursor(want.Encode())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.TransactionDt.Equal(want.TransactionDt) || got.ID != want.ID {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("token is URL-safe without padding", func(t *testing.T) {
		token := domain.ExpenseCursor{TransactionDt: time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC), ID: uuid.New()}.Encode()
		for _, c := range token {
			if c == '+' || c == '/' || c == '=' {
				t.Fatalf("token %q contains %q, want URL-safe base64 without padding", token, c)
			}
		}
	})

	enc := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	for name, token := range map[string]string{
		"not base64":         "garbage!",
		"standard padding":   base64.StdEncoding.EncodeToString([]byte("2026-10-17|" + uuid.NewString())),
		"no separator":       enc("2026-10-17" + uuid.NewString()),
		"bad date":           enc("2026-13-40|" + uuid.NewString()),
		"bad id":             enc("2026-10-17|not-a-uuid"),
		"empty after decode": enc(""),
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := domain.DecodeExpenseCursor(token)
			wantField(t, err, "page_token")
		})
	}
}

func TestCategoryCursor(t *testing.T) {
	for _, name := range []string{"Food", "Makan & Minum", "Café ☕", "a|b", ""} {
		t.Run("round-trips "+name, func(t *testing.T) {
			got, err := domain.DecodeCategoryCursor(domain.CategoryCursor{CategoryName: name}.Encode())
			if err != nil || got.CategoryName != name {
				t.Fatalf("got %q, %v; want %q", got.CategoryName, err, name)
			}
		})
	}

	t.Run("token is URL-safe without padding", func(t *testing.T) {
		token := domain.CategoryCursor{CategoryName: "Food & Drinks???"}.Encode()
		if strings.ContainsAny(token, "+/=") {
			t.Fatalf("token %q is not URL-safe base64 without padding", token)
		}
	})

	enc := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	for name, token := range map[string]string{
		"not base64":       "garbage!",
		"standard padding": base64.StdEncoding.EncodeToString([]byte("Food")),
		// Not UTF-8: Postgres would reject it as a TEXT parameter (22021), surfacing as an internal error.
		"invalid UTF-8": enc([]byte{0xff, 0xfe}),
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := domain.DecodeCategoryCursor(token)
			wantField(t, err, "page_token")
		})
	}
}
