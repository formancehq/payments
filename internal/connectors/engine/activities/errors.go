package activities

import (
	"errors"

	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ErrTypeStorage            = "STORAGE"
	ErrTypeDefault            = "DEFAULT"
	ErrTypeFailedPrecondition = "FAILED_PRECONDITON"
	ErrTypeInvalidArgument    = "INVALID_ARGUMENT"
	ErrTypePermissionDenied   = "PERMISSION_DENIED"
	ErrTypeUnimplemented      = "UNIMPLEMENTED"
	ErrTypeUnauthenticated    = "UNAUTHENTICATED"
	ErrTypeNotYetInstalled    = "NOT_YET_INSTALLED"
)

var nonRetryableErrorTypes = map[codes.Code]string{
	codes.FailedPrecondition: ErrTypeFailedPrecondition,
	codes.InvalidArgument:    ErrTypeInvalidArgument,
	codes.PermissionDenied:   ErrTypePermissionDenied,
	codes.Unimplemented:      ErrTypeUnimplemented,
	codes.Unauthenticated:    ErrTypeUnauthenticated,
}

func temporalPluginError(err error) error {
	var reason string

	code := status.Code(err)
	if code == codes.OK {
		return nil
	}

	if converted := status.Convert(err); converted != nil {
		for _, d := range converted.Details() {
			switch info := d.(type) {
			case *errdetails.ErrorInfo:
				reason = info.Reason
			}
		}
	}

	if code == codes.Internal && reason == ErrTypeNotYetInstalled {
		// Special case when the plugin is not yet installed
		return temporal.NewApplicationErrorWithCause(reason, ErrTypeNotYetInstalled, err)
	}

	errorType, ok := nonRetryableErrorTypes[code]
	if !ok {
		return temporal.NewApplicationErrorWithCause(reason, ErrTypeDefault, err)
	}
	return temporal.NewNonRetryableApplicationError(reason, errorType, err)
}

func temporalStorageError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, storage.ErrNotFound),
		errors.Is(err, storage.ErrDuplicateKeyValue),
		errors.Is(err, storage.ErrValidation):
		// Do not retry these errors
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeStorage, err)
	default:
		return temporal.NewApplicationErrorWithCause(err.Error(), ErrTypeStorage, err)
	}
}
