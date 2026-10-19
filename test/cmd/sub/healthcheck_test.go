package sub_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revirathya/revirathya-finance-core/cmd/sub"
)

// serve starts a server on a free localhost port answering /readyz with status, and returns the port in the
// form config.Load produces for HTTP_PORT (":<port>").
func serve(t *testing.T, status int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	_, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("split address: %v", err)
	}
	return ":" + port
}

func TestRunHealthcheck(t *testing.T) {
	t.Run("ready is exit code 0", func(t *testing.T) {
		if code := sub.RunHealthcheck(serve(t, http.StatusOK)); code != 0 {
			t.Fatalf("exit code %d, want 0 (is the port turned into a valid URL?)", code)
		}
	})

	t.Run("not ready is exit code 1", func(t *testing.T) {
		if code := sub.RunHealthcheck(serve(t, http.StatusServiceUnavailable)); code != 1 {
			t.Fatalf("exit code %d, want 1", code)
		}
	})

	t.Run("nothing listening is exit code 1", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		_, free, _ := net.SplitHostPort(l.Addr().String())
		_ = l.Close() // the port is free again: nothing answers

		if code := sub.RunHealthcheck(":" + free); code != 1 {
			t.Fatalf("exit code %d, want 1", code)
		}
	})
}
