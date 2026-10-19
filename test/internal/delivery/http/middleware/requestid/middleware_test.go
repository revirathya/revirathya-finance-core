package requestid_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware/requestid"
	"github.com/revirathya/revirathya-finance-core/internal/platform/logging"
)

// serve sends a request through the middleware and returns the ID the next handler
// saw in its context, plus the response.
func serve(t *testing.T, header string) (string, *httptest.ResponseRecorder) {
	t.Helper()
	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = logging.RequestIDFrom(r.Context())
	})

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	if header != "" {
		req.Header.Set(logging.RequestIDHeader, header)
	}
	rec := httptest.NewRecorder()
	requestid.Middleware()(next).ServeHTTP(rec, req)
	return seen, rec
}

func TestMiddleware(t *testing.T) {
	t.Run("generates an ID when none is sent", func(t *testing.T) {
		seen, rec := serve(t, "")

		if _, err := uuid.Parse(seen); err != nil {
			t.Fatalf("context ID %q, want a generated UUID", seen)
		}
		if got := rec.Header().Get(logging.RequestIDHeader); got != seen {
			t.Fatalf("response header %q, want the context ID %q", got, seen)
		}
	})

	t.Run("keeps a valid incoming ID", func(t *testing.T) {
		seen, rec := serve(t, "web-console-42")

		if seen != "web-console-42" || rec.Header().Get(logging.RequestIDHeader) != "web-console-42" {
			t.Fatalf("context %q, header %q, want both web-console-42", seen, rec.Header().Get(logging.RequestIDHeader))
		}
	})

	t.Run("replaces an invalid incoming ID", func(t *testing.T) {
		const forged = "abc level=ERROR msg=forged"
		seen, rec := serve(t, forged)

		if seen == forged {
			t.Fatal("invalid ID reached the context")
		}
		if _, err := uuid.Parse(seen); err != nil {
			t.Fatalf("context ID %q, want a generated UUID", seen)
		}
		if got := rec.Header().Get(logging.RequestIDHeader); got != seen {
			t.Fatalf("response header %q, want the context ID %q", got, seen)
		}
	})
}
