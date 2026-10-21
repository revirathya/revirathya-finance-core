package pagination_test

import (
	"errors"
	"testing"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain/constant"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/domain/pagination"
)

func TestSetPageSize(t *testing.T) {
	for _, tt := range []struct{ in, want int }{
		{0, constant.DefaultPageSize},
		{1, 1},
		{constant.MaxPageSize, constant.MaxPageSize},
		{constant.MaxPageSize + 1, constant.MaxPageSize},
		{1_000_000, constant.MaxPageSize},
	} {
		got, err := pagination.SetPageSize(tt.in)
		if err != nil || got != tt.want {
			t.Fatalf("SetPageSize(%d) = %d, %v; want %d", tt.in, got, err, tt.want)
		}
	}

	t.Run("negative is a ValidationError for page_size", func(t *testing.T) {
		_, err := pagination.SetPageSize(-1)
		var vErr *errDom.ValidationError
		if !errors.As(err, &vErr) || vErr.Field != "page_size" || vErr.Reason == "" {
			t.Fatalf("got %#v, want a ValidationError for page_size", err)
		}
	})
}
