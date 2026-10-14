package error_test

import (
	"errors"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	grpcErr "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
)

func TestToStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
		wantMsg  string
	}{
		{"not found", domain.ErrNotFound, codes.NotFound, "get expense 1: not found"},
		{"wrapped not found", fmt.Errorf("repo: %w", domain.ErrNotFound), codes.NotFound, "get expense 1: not found"},
		{"unauthenticated", domain.ErrUnauthenticated, codes.Unauthenticated, "authentication required"},
		{"forbidden", domain.ErrForbidden, codes.PermissionDenied, "get expense 1: permission denied"},
		{"conflict", domain.ErrConflict, codes.AlreadyExists, "get expense 1: resource already exists"},
		{"unknown error is hidden", errors.New(`pq: relation "app.expenses" does not exist`), codes.Internal, "internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, ok := status.FromError(grpcErr.ToStatus("get expense 1", tt.err))
			if !ok {
				t.Fatalf("not a gRPC status error")
			}
			if st.Code() != tt.wantCode {
				t.Fatalf("code: got %v, want %v", st.Code(), tt.wantCode)
			}
			if st.Message() != tt.wantMsg {
				t.Fatalf("message: got %q, want %q", st.Message(), tt.wantMsg)
			}
		})
	}
}
