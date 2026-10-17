package withauth

import (
	"net/http"
	"strings"

	"github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware"
)

func Middleware(tokens *auth.TokenManager) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
				if userId, err := tokens.Parse(raw); err == nil {
					r = r.WithContext(auth.WithPrincipal(r.Context(), auth.Principal{UserID: userId}))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
