package withauth

import (
	"net/http"
	"strings"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"

	// Internal - Http
	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware"
)

func Middleware(tokenManager *auth.TokenManager) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
				if userId, err := tokenManager.Parse(raw); err == nil {
					r = r.WithContext(auth.WithPrincipal(r.Context(), auth.Principal{UserID: userId}))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
