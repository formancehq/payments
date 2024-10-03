package plugins

import (
	"errors"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/models"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	FailureReasonInvalidRequest       = "INVALID_REQUEST"
	FailureReasonInvalidConfig        = "INVALID_CONFIG"
	FailureReasonBadRequestToUpstream = "BAD_REQUEST_TO_UPSTREAM"
	FailureReasonUnimplemented        = "UNIMPLEMENTED"
)

var (
	ErrNotImplemented  = errors.New("not implemented")
	ErrNotYetInstalled = errors.New("not yet installed")
)

type Error struct {
	RawMessage string
	Status     *status.Status
}

func NewError(code codes.Code, reason string, err error) error {
	st := status.Newf(code, err.Error())
	var dtErr error
	st, dtErr = st.WithDetails(&errdetails.ErrorInfo{
		Reason: reason,
	})
	if dtErr != nil {
		return Error{
			RawMessage: fmt.Sprintf("%s (%s)", err.Error(), dtErr.Error()),
			Status:     status.Newf(code, err.Error()),
		}
	}

	return Error{
		RawMessage: err.Error(),
		Status:     st,
	}
}

func (e Error) Error() string {
	return fmt.Sprintf("PLUGIN ERROR: %d, %s", e.Status.Code(), e.RawMessage)
}

func (e Error) GRPCStatus() *status.Status {
	return e.Status
}

func translateErrorToGRPC(err error) error {
	var (
		code   codes.Code
		reason string
	)

	switch {
	case errors.Is(err, ErrNotImplemented):
		code = codes.Unimplemented
		reason = FailureReasonUnimplemented
	case errors.Is(err, models.ErrMissingFromPayloadInRequest),
		errors.Is(err, models.ErrMissingAccountInMetadata):
		code = codes.FailedPrecondition
		reason = FailureReasonInvalidRequest
	case errors.Is(err, models.ErrInvalidConfig):
		code = codes.FailedPrecondition
		reason = FailureReasonInvalidConfig
	case errors.Is(err, httpwrapper.ErrStatusCodeClientError):
		code = codes.InvalidArgument
		reason = FailureReasonBadRequestToUpstream
	default:
		code = codes.Internal
	}

	return NewError(code, reason, err)
}
