package error

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// Internal
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

// ToStatus converts a usecase error into a gRPC status error.
// Known domain errors map to specific codes; anything else is logged with
// full detail and returned to the client as a generic Internal error.
func ToStatus(ctx context.Context, op string, err error) error {
	var vErr *errDom.ValidationError
	switch {
	case errors.Is(err, errDom.ErrNotFound):
		return status.Errorf(codes.NotFound, "%s: not found", op)
	case errors.Is(err, errDom.ErrUnauthenticated):
		return status.Error(codes.Unauthenticated, "authentication required")
	case errors.Is(err, errDom.ErrForbidden):
		return status.Errorf(codes.PermissionDenied, "%s: permission denied", op)
	case errors.Is(err, errDom.ErrConflict):
		return status.Errorf(codes.AlreadyExists, "%s: resource already exists", op)
	case errors.As(err, &vErr):
		return status.Error(codes.InvalidArgument, vErr.Error())
	case errors.Is(err, errDom.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, "invalid argument")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request timeout")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	default:
		slog.ErrorContext(ctx, "grpc request failed", "op", op, "err", err)
		return status.Error(codes.Internal, "internal error")
	}
}
