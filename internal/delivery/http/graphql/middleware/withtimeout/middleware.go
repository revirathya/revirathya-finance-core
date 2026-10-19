package withtimeout

import (
	"net/http"
	"time"

	// Internal - Http
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware"
)

func Middleware(timeout time.Duration) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(
			next,
			timeout,
			`{"errors":[{"message":"request timeout"}]}`,
		)
	}
}
