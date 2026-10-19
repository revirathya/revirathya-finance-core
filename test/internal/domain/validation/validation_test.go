package validation_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	// Internal
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/domain/validation"
)

// wantValidationError fails unless err is a *ValidationError for field whose reason
// contains reason, and unwraps to ErrInvalidInput (so the error mappers report it as
// bad input, not an internal error).
func wantValidationError(t *testing.T, err error, field, reason string) {
	t.Helper()
	var vErr *errDom.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("got %v, want a *ValidationError", err)
	}
	if vErr.Field != field {
		t.Fatalf("field %q, want %q", vErr.Field, field)
	}
	if !strings.Contains(vErr.Reason, reason) {
		t.Fatalf("reason %q, want it to contain %q", vErr.Reason, reason)
	}
	if !errors.Is(err, errDom.ErrInvalidInput) {
		t.Fatalf("%v does not unwrap to ErrInvalidInput", err)
	}
}

// day is a date as the handlers parse it: time.Parse("2006-01-02", ...), so UTC midnight.
func day(t *testing.T, date string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatalf("bad date %q: %v", date, err)
	}
	return d
}

// todayWIB is today's date in Asia/Jakarta, as UTC midnight like a parsed date.
func todayWIB() time.Time {
	y, m, d := time.Now().In(time.FixedZone("Asia/Jakarta", 7*60*60)).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestValidationErrorMessage(t *testing.T) {
	err := &errDom.ValidationError{Field: "amount", Reason: "must be greater than 0"}
	if got := err.Error(); got != "amount: must be greater than 0" {
		t.Fatalf("Error() %q, want %q", got, "amount: must be greater than 0")
	}
}

func TestValidateString(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		max      int
		required bool
		want     string
		reason   string // empty: valid
	}{
		{name: "valid value is returned unchanged", value: "Food", max: 100, required: true, want: "Food"},
		{name: "surrounding whitespace is trimmed", value: "  Food \n\t", max: 100, required: true, want: "Food"},
		{name: "exactly max characters is valid", value: strings.Repeat("a", 100), max: 100, required: true, want: strings.Repeat("a", 100)},
		{name: "max counts characters, not bytes", value: strings.Repeat("é", 100), max: 100, required: true, want: strings.Repeat("é", 100)},
		{name: "length is checked after trimming", value: "  " + strings.Repeat("a", 100) + "  ", max: 100, required: true, want: strings.Repeat("a", 100)},
		{name: "one character over max is rejected", value: strings.Repeat("a", 101), max: 100, required: true, reason: "must be at most 100 characters"},
		{name: "message uses the given max", value: strings.Repeat("a", 501), max: 500, required: false, reason: "must be at most 500 characters"},
		{name: "required and empty is rejected", value: "", max: 100, required: true, reason: "is required"},
		{name: "required and only whitespace is rejected", value: "   ", max: 100, required: true, reason: "is required"},
		{name: "optional and empty is valid", value: "", max: 100, required: false, want: ""},
		{name: "optional and only whitespace becomes empty", value: "   ", max: 100, required: false, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validation.ValidateString("category_name", tt.value, tt.max, tt.required)

			if tt.reason != "" {
				wantValidationError(t, err, "category_name", tt.reason)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateDate(t *testing.T) {
	tests := []struct {
		name   string
		value  time.Time
		reason string // empty: valid
	}{
		{name: "2000-01-01 is the earliest valid date", value: day(t, "2000-01-01")},
		{name: "1999-12-31 is rejected", value: day(t, "1999-12-31"), reason: "2000-01-01"},
		{name: "a past date is valid", value: day(t, "2026-01-15")},
		{name: "today is valid", value: todayWIB()},
		// Today and tomorrow are taken in Asia/Jakarta. A user in Jakarta must be able to
		// enter tomorrow's date at any hour, including 00:00-07:00 WIB, when UTC is still
		// on the previous day.
		{name: "tomorrow is valid", value: todayWIB().AddDate(0, 0, 1)},
		{name: "the day after tomorrow is rejected", value: todayWIB().AddDate(0, 0, 2), reason: "today"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateDate("transaction_dt", tt.value)

			if tt.reason != "" {
				wantValidationError(t, err, "transaction_dt", tt.reason)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}

	t.Run("error uses the given field name", func(t *testing.T) {
		wantValidationError(t, validation.ValidateDate("paid_on", day(t, "1990-01-01")), "paid_on", "2000-01-01")
	})
}

func TestValidateDecimal(t *testing.T) {
	// NUMERIC(14, 2), like expenses.amount
	tests := []struct {
		name     string
		value    string
		positive bool
		reason   string // empty: valid
	}{
		{name: "whole amount is valid", value: "100", positive: true},
		{name: "two decimal places are valid", value: "42.50", positive: true},
		{name: "smallest positive amount is valid", value: "0.01", positive: true},
		{name: "trailing zeros beyond the scale are valid", value: "10.500", positive: true},
		{name: "three significant decimal places are rejected", value: "10.505", positive: true, reason: "at most 2 decimal places"},
		{name: "zero is rejected when it must be positive", value: "0", positive: true, reason: "must be greater than 0"},
		{name: "negative is rejected when it must be positive", value: "-1", positive: true, reason: "must be greater than 0"},
		{name: "negative is valid when it need not be positive", value: "-5.25", positive: false},
		{name: "largest value the column holds is valid", value: "999999999999.99", positive: true},
		{name: "10^12 is rejected", value: "1000000000000", positive: true, reason: "must be less than 1000000000000"},
		{name: "upper limit applies to negative values too", value: "-1000000000000", positive: false, reason: "must be less than 1000000000000"},
		{name: "too many decimals is reported before the limit", value: "999999999999.999", positive: true, reason: "at most 2 decimal places"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateDecimal("amount", decimal.RequireFromString(tt.value), tt.positive, 14, 2)

			if tt.reason != "" {
				wantValidationError(t, err, "amount", tt.reason)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
