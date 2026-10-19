package error_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"google.golang.org/grpc/status"

	grpcErr "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

// Concurrent requests must each get their own validation message. Run with -race.
func TestToStatusConcurrentValidationErrors(t *testing.T) {
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			field := fmt.Sprintf("field_%d", i)
			st, _ := status.FromError(grpcErr.ToStatus(context.Background(), "op", &errDom.ValidationError{Field: field, Reason: "is required"}))
			if want := field + ": is required"; st.Message() != want {
				t.Errorf("message %q, want %q", st.Message(), want)
			}
		}()
	}
	wg.Wait()
}
