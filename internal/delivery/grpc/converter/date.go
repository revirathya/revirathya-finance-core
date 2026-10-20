package converter

import (
	"time"

	"google.golang.org/genproto/googleapis/type/date"

	// Internal
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

func ToProtoDate(t time.Time) *date.Date {
	return &date.Date{
		Year:  int32(t.Year()),  //nolint:gosec // G115: a calendar year
		Month: int32(t.Month()), //nolint:gosec // G115: 1-12
		Day:   int32(t.Day()),   //nolint:gosec // G115: 1-31
	}
}

func FromProtoDate(field string, d *date.Date) (time.Time, error) {
	if d == nil {
		return time.Time{}, &errDom.ValidationError{Field: field, Reason: "is required"}
	}

	t := time.Date(int(d.GetYear()), time.Month(d.GetMonth()), int(d.GetDay()), 0, 0, 0, 0, time.UTC)

	if d.GetYear() < 1 || t.Year() != int(d.GetYear()) || int(t.Month()) != int(d.GetMonth()) || int(t.Day()) != int(d.GetDay()) {
		return time.Time{}, &errDom.ValidationError{Field: field, Reason: "must be a valid calendar date"}
	}
	return t, nil
}

func FromOptionalProtoDate(field string, d *date.Date) (*time.Time, error) {
	if d == nil {
		return nil, nil
	}

	t, err := FromProtoDate(field, d)
	if err != nil {
		return nil, err
	}

	return &t, nil
}
