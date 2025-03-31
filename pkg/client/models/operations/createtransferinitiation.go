// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type CreateTransferInitiationResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// OK
	TransferInitiationResponse *components.TransferInitiationResponse
	// Error
	PaymentsErrorResponse *components.PaymentsErrorResponse
}

func (o *CreateTransferInitiationResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *CreateTransferInitiationResponse) GetTransferInitiationResponse() *components.TransferInitiationResponse {
	if o == nil {
		return nil
	}
	return o.TransferInitiationResponse
}

func (o *CreateTransferInitiationResponse) GetPaymentsErrorResponse() *components.PaymentsErrorResponse {
	if o == nil {
		return nil
	}
	return o.PaymentsErrorResponse
}
