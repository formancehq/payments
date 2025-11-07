package errors

import "fmt"

type wrappedError struct {
    err error
}

// NewWrappedError creates a new error that wraps the cause error with the new error.
// Matches the behavior used by connectors to retain cause classification.
func NewWrappedError(cause error, newError error) error {
    return &wrappedError{err: fmt.Errorf("%w: %w", cause, newError)}
}

func (e *wrappedError) Error() string {
    return e.err.Error()
}

// Unwrap chain support so errors.Is works properly
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
            err = nexts[0]
        default:
            return err
        }
    }
}

