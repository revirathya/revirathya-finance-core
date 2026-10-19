package scalar

import (
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"

	// Internal
	errGql "github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/error"
)

func MarshalUUID(id uuid.UUID) graphql.Marshaler {
	return graphql.MarshalString(id.String())
}

func UnmarshalUUID(v any) (uuid.UUID, error) {
	s, ok := v.(string)
	if !ok {
		return uuid.Nil, errGql.NewInvalidInputError(fmt.Errorf("'ID' must be a UUID string"))
	}

	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, errGql.NewInvalidInputError(fmt.Errorf("invalid 'ID' %q: want a UUID", s))
	}

	return id, nil
}
