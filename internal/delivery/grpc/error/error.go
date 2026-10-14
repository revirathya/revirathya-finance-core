package error

import (
	"errors"
	"log/slog"

	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToStatus converts a usecase error into a gRPC status error.
// Known domain errors map to specific codes; anything else is logged with
// full detail and returned to the client as a generic Internal error.
func ToStatus(op string, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return status.Errorf(codes.NotFound, "%s: not found", op)
	case errors.Is(err, domain.ErrUnauthenticated):
		return status.Error(codes.Unauthenticated, "authentication required")
	case errors.Is(err, domain.ErrForbidden):
		return status.Errorf(codes.PermissionDenied, "%s: permission denied", op)
	case errors.Is(err, domain.ErrConflict):
		return status.Errorf(codes.AlreadyExists, "%s: resource already exists", op)
	case errors.Is(err, domain.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, "invalid argument")
	default:
		slog.Error("grpc request failed", "op", op, "err", err)
		return status.Error(codes.Internal, "internal error")
	}
}
