package orders

import "errors"

var (
	ErrNotFound         = errors.New("order not found")
	ErrAlreadyExists    = errors.New("order already exists")
	ErrInvalidData      = errors.New("invalid data")
	ErrRetry            = errors.New("retry limit exceeded")
	ErrUnexpected       = errors.New("unexpected error")
	ErrInvalidReference = errors.New("invalid reference")
	ErrConflict         = errors.New("conflict")
	ErrRetryable        = errors.New("retryable")
	ErrTimeout          = errors.New("timeout")
)
