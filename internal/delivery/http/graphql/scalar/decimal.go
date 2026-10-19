package scalar

import (
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/shopspring/decimal"

	// Internal
	errGql "github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/error"
)

func MarshalDecimal(d decimal.Decimal) graphql.Marshaler {
	return graphql.MarshalString(d.String())
}

func UnmarshalDecimal(v any) (decimal.Decimal, error) {
	s, ok := v.(string)
	if !ok {
		return decimal.Decimal{}, errGql.NewInvalidInputError(
			fmt.Errorf("'Decimal' must be a string, e.g. \"12500.50\""),
		)
	}

	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Decimal{}, errGql.NewInvalidInputError(
			fmt.Errorf("invalid 'Decimal' %s", s),
		)
	}
	return d, nil
}
