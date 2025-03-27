package errors

import "fmt"

type wrappedError struct {
	err error
}

// NewWrappedError creates a new error that wraps the cause error with the new error.
// It should be use when you want to have the end cause of an error and not all
// the stack trace, for example when you want to store the transfer/payout creation
// error.
func NewWrappedError(cause error, newError error) error {
	return &wrappedError{
		err: fmt.Errorf("%w: %w", cause, newError),
	}
}

func (e *wrappedError) Error() string {
	return e.err.Error()
}

// This method is needed for all errors to be recognized by the errors.Is function
func (e *wrappedError) Unwrap() []error {
	return e.err.(interface{ Unwrap() []error }).Unwrap()
}

// Cause unwraps the error to the root cause
func Cause(err error) error {
	for {
		switch v := err.(type) {
		case interface{ Unwrap() error }:
			next := v.Unwrap()
			if next == nil {
				return err
			}
			err = next
		case interface{ Unwrap() []error }:
			nexts := v.Unwrap()
			if len(nexts) == 0 {
				return err
			}
			// we assume that the cause error is always the first one
			err = nexts[0]
		default:
			return err
		}
	}
}
