package withauth

import (
	"net/http"
	"strings"

	// Internal
	authApp "github.com/revirathya/revirathya-finance-core/internal/auth"

	// Internal - GraphQL
	"github.com/revirathya/revirathya-finance-core/internal/delivery/graphql/middleware"
)

func Middleware(tokens *authApp.TokenManager) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
				if userId, err := tokens.Parse(raw); err == nil {
					r = r.WithContext(authApp.WithPrincipal(r.Context(), authApp.Principal{UserID: userId}))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
