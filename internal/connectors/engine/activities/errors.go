package activities

import (
	"go.temporal.io/sdk/temporal"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ErrTypeDefault            = "DEFAULT"
	ErrTypeFailedPrecondition = "FAILED_PRECONDITON"
	ErrTypeInvalidArgument    = "INVALID_ARGUMENT"
	ErrTypePermissionDenied   = "PERMISSION_DENIED"
	ErrTypeUnimplemented      = "UNIMPLEMENTED"
	ErrTypeUnauthenticated    = "UNAUTHENTICATED"
)

var nonRetryableErrorTypes = map[codes.Code]string{
	codes.FailedPrecondition: ErrTypeFailedPrecondition,
	codes.InvalidArgument:    ErrTypeInvalidArgument,
	codes.PermissionDenied:   ErrTypePermissionDenied,
	codes.Unimplemented:      ErrTypeUnimplemented,
	codes.Unauthenticated:    ErrTypeUnauthenticated,
}

func temporalError(err error) error {
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

	errorType, ok := nonRetryableErrorTypes[code]
	if !ok {
		return temporal.NewApplicationErrorWithCause(reason, ErrTypeDefault, err)
	}
	return temporal.NewNonRetryableApplicationError(reason, errorType, err)
}
