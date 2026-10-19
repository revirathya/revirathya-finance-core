package scalar

import (
	"fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"

	// Internal - GraphQL
	errGql "github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/error"
)

const dateLayout = "2006-01-02"

func MarshalDate(t time.Time) graphql.Marshaler {
	return graphql.MarshalString(t.Format(dateLayout))
}

func UnmarshalDate(v any) (time.Time, error) {
	s, ok := v.(string)
	if !ok {
		return time.Time{}, errGql.NewInvalidInputError(
			fmt.Errorf("'Date' must be a string in YYYY-MM-DD format"),
		)
	}

	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return time.Time{}, errGql.NewInvalidInputError(
			fmt.Errorf("invalid 'Date' %q:  want YYYY-MM-DD", s),
		)
	}
	return t, nil
}
