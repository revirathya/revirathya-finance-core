package error_test

import (
	"errors"
	"testing"
	"time"

	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

func TestRateLimitError(t *testing.T) {
	t.Run("unwraps to ErrRateLimited", func(t *testing.T) {
		var err error = &errDom.RateLimitError{RetryAfter: time.Second}
		if !errors.Is(err, errDom.ErrRateLimited) {
			t.Fatal("errors.Is(err, ErrRateLimited) = false")
		}
	})

	t.Run("RetryAfterSeconds rounds up, at least 1", func(t *testing.T) {
		for _, tt := range []struct {
			in   time.Duration
			want int
		}{
			{0, 1},
			{time.Millisecond, 1},
			{time.Second, 1},
			{1500 * time.Millisecond, 2},
			{59*time.Second + time.Nanosecond, 60},
		} {
			if got := (&errDom.RateLimitError{RetryAfter: tt.in}).RetryAfterSeconds(); got != tt.want {
				t.Fatalf("RetryAfter %v: got %d, want %d", tt.in, got, tt.want)
			}
		}
	})
}
