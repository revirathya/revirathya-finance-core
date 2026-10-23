package error

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("conflict")
	ErrInvalidInput    = errors.New("invalid input")
	ErrRateLimited     = errors.New("rate limited")
)

// ValidationError
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

func (e *ValidationError) Unwrap() error { return ErrInvalidInput }

// RateLimitError is returned when a user has used up their write budget; RetryAfter is the time until it resets.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return "too many write requests, try again later"
}

func (e *RateLimitError) Unwrap() error {
	return ErrRateLimited
}

// RetryAfterSeconds is RetryAfter in whole seconds, rounded up and at least 1.
func (e *RateLimitError) RetryAfterSeconds() int {
	secs := int((e.RetryAfter + time.Second - 1) / time.Second)
	return max(secs, 1)
}
