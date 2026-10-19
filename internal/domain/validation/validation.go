package validation

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/shopspring/decimal"
)

var tzLoc = time.FixedZone("Asia/Jakarta", 7*60*60)
var minDate = time.Date(2000, time.January, 1, 0, 0, 0, 0, tzLoc)

func ValidateString(field string, value string, maxLength int, isRequired bool) (string, error) {
	v := strings.TrimSpace(value)
	switch {
	case (isRequired) && (v == ""):
		return "", &errDom.ValidationError{Field: field, Reason: "is required"}
	case utf8.RuneCountInString(v) > maxLength:
		return "", &errDom.ValidationError{Field: field, Reason: fmt.Sprintf("must be at most %d characters", maxLength)}
	}
	return v, nil
}

func ValidateDate(field string, value time.Time) error {
	y, m, d := time.Now().In(tzLoc).Date()
	maxDate := time.Date(y, m, d+1, 0, 0, 0, 0, time.UTC)
	switch {
	case value.Before(minDate):
		return &errDom.ValidationError{Field: field, Reason: fmt.Sprintf("must be on or after %s", minDate.Format("2006-01-02"))}
	case value.After(maxDate):
		return &errDom.ValidationError{Field: field, Reason: "at most 1 day from today"}
	}
	return nil
}

func ValidateDecimal(field string, value decimal.Decimal, mustPositive bool, precision int32, scale int32) error {
	limit := decimal.New(1, precision-scale)

	switch {
	case (mustPositive) && (!value.IsPositive()):
		return &errDom.ValidationError{Field: field, Reason: "must be greater than 0"}
	case !value.Equal(value.Round(scale)):
		return &errDom.ValidationError{Field: field, Reason: fmt.Sprintf("at most %d decimal places", scale)}
	case value.Abs().GreaterThanOrEqual(limit):
		return &errDom.ValidationError{Field: field, Reason: fmt.Sprintf("must be less than %s", limit)}
	}
	return nil
}
