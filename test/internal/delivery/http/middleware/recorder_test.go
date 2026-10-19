package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revirathya/revirathya-finance-core/internal/delivery/http/middleware"
)

func TestStatusRecorder(t *testing.T) {
	t.Run("records the status sent", func(t *testing.T) {
		rec := middleware.NewStatusRecorder(httptest.NewRecorder())
		rec.WriteHeader(http.StatusTeapot)
		if rec.Code() != http.StatusTeapot {
			t.Fatalf("code %d, want %d", rec.Code(), http.StatusTeapot)
		}
	})

	t.Run("keeps the first status, as net/http does", func(t *testing.T) {
		under := httptest.NewRecorder()
		rec := middleware.NewStatusRecorder(under)
		rec.WriteHeader(http.StatusNotFound)
		rec.WriteHeader(http.StatusInternalServerError) // superfluous: net/http ignores it

		if rec.Code() != http.StatusNotFound || under.Code != http.StatusNotFound {
			t.Fatalf("recorded %d, client got %d; want both %d", rec.Code(), under.Code, http.StatusNotFound)
		}
	})

	t.Run("a write without WriteHeader is 200", func(t *testing.T) {
		rec := middleware.NewStatusRecorder(httptest.NewRecorder())
		_, _ = rec.Write([]byte("ok"))
		rec.WriteHeader(http.StatusInternalServerError) // too late: headers are already sent
		if rec.Code() != http.StatusOK {
			t.Fatalf("code %d, want 200", rec.Code())
		}
	})

	t.Run("nothing written is 200", func(t *testing.T) {
		if code := middleware.NewStatusRecorder(httptest.NewRecorder()).Code(); code != http.StatusOK {
			t.Fatalf("code %d, want 200", code)
		}
	})

	t.Run("Unwrap returns the wrapped writer, for http.ResponseController", func(t *testing.T) {
		under := httptest.NewRecorder()
		if middleware.NewStatusRecorder(under).Unwrap() != under {
			t.Fatal("Unwrap did not return the wrapped writer")
		}
	})
}
