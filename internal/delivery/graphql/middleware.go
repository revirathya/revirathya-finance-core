package graphql

import (
	"net/http"
	"strings"

	"github.com/revirathya/revirathya-finance-core/internal/auth"
)

// WithAuth attaches the session's principal to the request context when a valid
// "Authorization: Bearer <token>" header is present. Requests without a valid
// token pass through unauthenticated; resolvers decide whether that's allowed.
func WithAuth(tokens *auth.TokenManager, next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
				if userId, err := tokens.Parse(raw); err == nil {
					r = r.WithContext(auth.WithPrincipal(r.Context(), auth.Principal{UserID: userId}))
				}
			}
			next.ServeHTTP(w, r)
		},
	)
}
