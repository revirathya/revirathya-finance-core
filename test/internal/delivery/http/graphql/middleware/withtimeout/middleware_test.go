package withtimeout_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/middleware/withtimeout"
)

func TestMiddleware(t *testing.T) {
	t.Run("fast handler responds normally", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
		rec := httptest.NewRecorder()
		withtimeout.Middleware(time.Second)(next).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/graphql", nil))

		if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
			t.Fatalf("status %d body %q, want 200 ok", rec.Code, rec.Body.String())
		}
	})

	t.Run("slow handler gets 503 with a GraphQL-shaped error and a cancelled context", func(t *testing.T) {
		cancelled := make(chan bool, 1)
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
				cancelled <- true
			case <-time.After(2 * time.Second):
				cancelled <- false
			}
		})
		rec := httptest.NewRecorder()
		withtimeout.Middleware(50*time.Millisecond)(next).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/graphql", nil))

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
		if !strings.Contains(rec.Body.String(), "request timeout") {
			t.Fatalf("body %q, want the request timeout error", rec.Body.String())
		}
		if !<-cancelled {
			t.Fatal("handler's context was not cancelled, so slow work (e.g. DB queries) keeps running")
		}
	})
}
