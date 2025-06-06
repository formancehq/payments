// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type DeleteTransferInitiationRequest struct {
	// The transfer ID.
	TransferID string `pathParam:"style=simple,explode=false,name=transferId"`
}

func (o *DeleteTransferInitiationRequest) GetTransferID() string {
	if o == nil {
		return ""
	}
	return o.TransferID
}

type DeleteTransferInitiationResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Error
	PaymentsErrorResponse *components.PaymentsErrorResponse
}

func (o *DeleteTransferInitiationResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *DeleteTransferInitiationResponse) GetPaymentsErrorResponse() *components.PaymentsErrorResponse {
	if o == nil {
		return nil
	}
	return o.PaymentsErrorResponse
}
