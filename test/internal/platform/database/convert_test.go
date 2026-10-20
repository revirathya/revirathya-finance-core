package database_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/revirathya/revirathya-finance-core/internal/platform/database"
)

func TestToOptionalPgxUUID(t *testing.T) {
	if got := database.ToOptionalPgxUUID(nil); got.Valid {
		t.Fatalf("nil: got %+v, want an invalid (NULL) UUID", got)
	}

	id := uuid.New()
	got := database.ToOptionalPgxUUID(&id)
	if !got.Valid || uuid.UUID(got.Bytes) != id {
		t.Fatalf("got %+v, want valid %v", got, id)
	}
}
