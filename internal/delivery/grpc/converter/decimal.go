package converter

import (
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/shopspring/decimal"
	grpcDec "google.golang.org/genproto/googleapis/type/decimal"
)

func ToProtoDecimal(d decimal.Decimal) *grpcDec.Decimal {
	return &grpcDec.Decimal{Value: d.String()}
}

func FromProtoDecimal(field string, d *grpcDec.Decimal) (decimal.Decimal, error) {
	if d.GetValue() == "" {
		return decimal.Decimal{}, &errDom.ValidationError{Field: field, Reason: "is required"}
	}

	v, err := decimal.NewFromString(d.GetValue())
	if err != nil {
		return decimal.Decimal{}, &errDom.ValidationError{Field: field, Reason: "must be a decimal number"}
	}

	return v, err
}

func FromOptionalProtoDecimal(field string, d *grpcDec.Decimal) (*decimal.Decimal, error) {
	if d == nil {
		return nil, nil
	}
	v, err := FromProtoDecimal(field, d)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
