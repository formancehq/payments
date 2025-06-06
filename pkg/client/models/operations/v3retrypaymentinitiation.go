// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type V3RetryPaymentInitiationRequest struct {
	// The payment initiation ID
	PaymentInitiationID string `pathParam:"style=simple,explode=false,name=paymentInitiationID"`
}

func (o *V3RetryPaymentInitiationRequest) GetPaymentInitiationID() string {
	if o == nil {
		return ""
	}
	return o.PaymentInitiationID
}

type V3RetryPaymentInitiationResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Accepted
	V3RetryPaymentInitiationResponse *components.V3RetryPaymentInitiationResponse
	// Error
	V3ErrorResponse *components.V3ErrorResponse
}

func (o *V3RetryPaymentInitiationResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *V3RetryPaymentInitiationResponse) GetV3RetryPaymentInitiationResponse() *components.V3RetryPaymentInitiationResponse {
	if o == nil {
		return nil
	}
	return o.V3RetryPaymentInitiationResponse
}

func (o *V3RetryPaymentInitiationResponse) GetV3ErrorResponse() *components.V3ErrorResponse {
	if o == nil {
		return nil
	}
	return o.V3ErrorResponse
}
