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

func TestRequestNormalizedSize(t *testing.T) {
	got, err := pagination.Request{Size: 0, Token: "anything"}.NormalizedSize()
	if err != nil || got != constant.DefaultPageSize {
		t.Fatalf("got %d, %v; want the default %d", got, err, constant.DefaultPageSize)
	}
	if _, err := (pagination.Request{Size: -1}).NormalizedSize(); err == nil {
		t.Fatal("negative size: want an error")
	}
}

func TestCut(t *testing.T) {
	rows := []string{"a", "b", "c"}

	t.Run("size + 1 rows: the page is the first size rows, more follow, last is the last one kept", func(t *testing.T) {
		page, last, more := pagination.Cut(rows, 2)
		if len(page) != 2 || page[1] != "b" || last != "b" || !more {
			t.Fatalf("got %v, last %q, more %v; want [a b], b, true", page, last, more)
		}
	})

	t.Run("exactly size rows is the last page", func(t *testing.T) {
		page, last, more := pagination.Cut(rows, 3)
		if len(page) != 3 || more || last != "" {
			t.Fatalf("got %v, last %q, more %v; want all 3, no last, false", page, last, more)
		}
	})

	t.Run("fewer rows than size, and no rows, are a last page", func(t *testing.T) {
		if page, _, more := pagination.Cut(rows, 10); len(page) != 3 || more {
			t.Fatalf("got %v, more %v; want all 3, false", page, more)
		}
		if page, _, more := pagination.Cut([]string{}, 2); len(page) != 0 || more {
			t.Fatalf("got %v, more %v; want empty, false", page, more)
		}
	})

	t.Run("works for any element type", func(t *testing.T) {
		type row struct{ ID int }
		_, last, more := pagination.Cut([]row{{1}, {2}}, 1)
		if !more || last.ID != 1 {
			t.Fatalf("got last %+v, more %v; want {1}, true", last, more)
		}
	})
}
