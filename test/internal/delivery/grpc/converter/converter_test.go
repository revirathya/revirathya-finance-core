package converter_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/genproto/googleapis/type/date"
	gdecimal "google.golang.org/genproto/googleapis/type/decimal"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/converter"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

// wantValidation fails unless err is a ValidationError for field whose reason contains reason, so that
// errGrpc.ToStatus reports it as INVALID_ARGUMENT with a "field: reason" message.
func wantValidation(t *testing.T, err error, field, reason string) {
	t.Helper()
	var vErr *errDom.ValidationError
	if !errors.As(err, &vErr) || vErr.Field != field || !strings.Contains(vErr.Reason, reason) {
		t.Fatalf("got %v, want a ValidationError %s: %s", err, field, reason)
	}
}

func TestDate(t *testing.T) {
	t.Run("ToProtoDate takes the calendar date", func(t *testing.T) {
		got := converter.ToProtoDate(time.Date(2026, 10, 17, 23, 59, 0, 0, time.UTC))
		if got.GetYear() != 2026 || got.GetMonth() != 10 || got.GetDay() != 17 {
			t.Fatalf("got %v, want 2026-10-17", got)
		}
	})

	t.Run("FromProtoDate gives UTC midnight, like time.Parse(\"2006-01-02\")", func(t *testing.T) {
		got, err := converter.FromProtoDate("transaction_dt", &date.Date{Year: 2024, Month: 2, Day: 29})
		want, _ := time.Parse("2006-01-02", "2024-02-29")
		if err != nil || !got.Equal(want) || got.Location() != time.UTC {
			t.Fatalf("got %v, %v; want %v UTC", got, err, want)
		}
	})

	t.Run("round-trips", func(t *testing.T) {
		in := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
		got, err := converter.FromProtoDate("transaction_dt", converter.ToProtoDate(in))
		if err != nil || !got.Equal(in) {
			t.Fatalf("got %v, %v; want %v", got, err, in)
		}
	})

	t.Run("missing is required", func(t *testing.T) {
		_, err := converter.FromProtoDate("transaction_dt", nil)
		wantValidation(t, err, "transaction_dt", "is required")
	})

	for name, d := range map[string]*date.Date{
		"30 February (time.Date would roll it to 2 March)": {Year: 2026, Month: 2, Day: 30},
		"29 February in a non-leap year":                   {Year: 2026, Month: 2, Day: 29},
		"month 13":                                         {Year: 2026, Month: 13, Day: 1},
		"day 32":                                           {Year: 2026, Month: 1, Day: 32},
		"negative day":                                     {Year: 2026, Month: 1, Day: -1},
		"partial: no year":                                 {Month: 10, Day: 17},
		"partial: no day":                                  {Year: 2026, Month: 10},
		"partial: no month":                                {Year: 2026, Day: 17},
		"empty":                                            {},
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := converter.FromProtoDate("transaction_dt", d)
			wantValidation(t, err, "transaction_dt", "must be a valid calendar date")
		})
	}

	t.Run("FromOptionalProtoDate: nil is not set; a value is converted and validated", func(t *testing.T) {
		if got, err := converter.FromOptionalProtoDate("transaction_dt", nil); got != nil || err != nil {
			t.Fatalf("nil: got %v, %v; want nil, nil", got, err)
		}
		got, err := converter.FromOptionalProtoDate("transaction_dt", &date.Date{Year: 2026, Month: 10, Day: 17})
		if err != nil || got == nil || got.Format("2006-01-02") != "2026-10-17" {
			t.Fatalf("got %v, %v; want 2026-10-17", got, err)
		}
		_, err = converter.FromOptionalProtoDate("transaction_dt", &date.Date{Year: 2026, Month: 2, Day: 30})
		wantValidation(t, err, "transaction_dt", "must be a valid calendar date")
	})
}

func TestDecimal(t *testing.T) {
	t.Run("ToProtoDecimal keeps the exact value", func(t *testing.T) {
		if got := converter.ToProtoDecimal(decimal.RequireFromString("999999999999.99")); got.GetValue() != "999999999999.99" {
			t.Fatalf("got %q", got.GetValue())
		}
	})

	for _, v := range []string{"12500.50", "0.01", "-3", "999999999999.99", "1e3"} {
		t.Run("FromProtoDecimal parses "+v, func(t *testing.T) {
			got, err := converter.FromProtoDecimal("amount", &gdecimal.Decimal{Value: v})
			if err != nil || !got.Equal(decimal.RequireFromString(v)) {
				t.Fatalf("got %v, %v; want %s", got, err, v)
			}
		})
	}

	t.Run("no float rounding: 0.1 + 0.2 is exactly 0.3", func(t *testing.T) {
		a, _ := converter.FromProtoDecimal("amount", &gdecimal.Decimal{Value: "0.1"})
		b, _ := converter.FromProtoDecimal("amount", &gdecimal.Decimal{Value: "0.2"})
		if !a.Add(b).Equal(decimal.RequireFromString("0.3")) {
			t.Fatalf("0.1 + 0.2 = %s", a.Add(b))
		}
	})

	t.Run("missing message or empty value is required", func(t *testing.T) {
		_, err := converter.FromProtoDecimal("amount", nil)
		wantValidation(t, err, "amount", "is required")
		_, err = converter.FromProtoDecimal("amount", &gdecimal.Decimal{})
		wantValidation(t, err, "amount", "is required")
	})

	for _, v := range []string{"ten", "12,500.50", "1.2.3", " 12"} {
		t.Run("rejects "+v, func(t *testing.T) {
			_, err := converter.FromProtoDecimal("amount", &gdecimal.Decimal{Value: v})
			wantValidation(t, err, "amount", "must be a decimal number")
		})
	}
}
