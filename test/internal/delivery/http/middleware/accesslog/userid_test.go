package accesslog_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/revirathya/revirathya-finance-core/internal/auth"
)

func TestMiddlewareUserID(t *testing.T) {
	t.Run("a principal set by inner middleware is logged as user_id", func(t *testing.T) {
		userID := uuid.New()
		// Like withauth: the principal goes on a derived request the access log never sees.
		lines := serve(t, httptest.NewRequest(http.MethodPost, "/graphql", nil), func(w http.ResponseWriter, r *http.Request) {
			_ = r.WithContext(auth.WithPrincipal(r.Context(), auth.Principal{UserID: userID}))
		})
		if len(lines) != 1 || lines[0]["user_id"] != userID.String() {
			t.Fatalf("lines %v, want one with user_id %s", lines, userID)
		}
	})

	t.Run("an anonymous request has no user_id", func(t *testing.T) {
		lines := serve(t, httptest.NewRequest(http.MethodPost, "/graphql", nil), status(http.StatusOK))
		if _, ok := lines[0]["user_id"]; len(lines) != 1 || ok {
			t.Fatalf("lines %v, want one without user_id", lines)
		}
	})
}
