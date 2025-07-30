package client

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/plaid/plaid-go/v34/plaid"
)

func wrapSDKError(err error) error {
	plaidErr, err := plaid.ToPlaidError(err)
	if err != nil {
		// Return the original error if it cannot be converted to a Plaid error
		return err
	}

	err = fmt.Errorf("%s: %s: %s", plaidErr.ErrorType, plaidErr.ErrorCode, plaidErr.ErrorMessage)
	switch plaidErr.ErrorType {
	case plaid.PLAIDERRORTYPE_INVALID_INPUT, plaid.PLAIDERRORTYPE_INVALID_REQUEST, plaid.PLAIDERRORTYPE_OAUTH_ERROR, plaid.PLAIDERRORTYPE_ITEM_ERROR, plaid.PLAIDERRORTYPE_INSTITUTION_ERROR:
		return errorsutils.NewWrappedError(err, httpwrapper.ErrStatusCodeClientError)
	case plaid.PLAIDERRORTYPE_RATE_LIMIT_EXCEEDED:
		return errorsutils.NewWrappedError(err, httpwrapper.ErrStatusCodeTooManyRequests)
	default:
		err := fmt.Errorf("%s: %s: %s", plaidErr.ErrorType, plaidErr.ErrorCode, plaidErr.ErrorMessage)
		return errorsutils.NewWrappedError(err, httpwrapper.ErrStatusCodeServerError)
	}

}
