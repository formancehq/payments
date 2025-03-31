// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type V3ListPaymentInitiationAdjustmentsRequest struct {
	// The payment initiation ID
	PaymentInitiationID string `pathParam:"style=simple,explode=false,name=paymentInitiationID"`
	// The number of items to return
	PageSize *int64 `queryParam:"style=form,explode=true,name=pageSize"`
	// Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.
	//
	Cursor      *string        `queryParam:"style=form,explode=true,name=cursor"`
	RequestBody map[string]any `request:"mediaType=application/json"`
}

func (o *V3ListPaymentInitiationAdjustmentsRequest) GetPaymentInitiationID() string {
	if o == nil {
		return ""
	}
	return o.PaymentInitiationID
}

func (o *V3ListPaymentInitiationAdjustmentsRequest) GetPageSize() *int64 {
	if o == nil {
		return nil
	}
	return o.PageSize
}

func (o *V3ListPaymentInitiationAdjustmentsRequest) GetCursor() *string {
	if o == nil {
		return nil
	}
	return o.Cursor
}

func (o *V3ListPaymentInitiationAdjustmentsRequest) GetRequestBody() map[string]any {
	if o == nil {
		return nil
	}
	return o.RequestBody
}

type V3ListPaymentInitiationAdjustmentsResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// OK
	V3PaymentInitiationAdjustmentsCursorResponse *components.V3PaymentInitiationAdjustmentsCursorResponse
	// Error
	V3ErrorResponse *components.V3ErrorResponse
}

func (o *V3ListPaymentInitiationAdjustmentsResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *V3ListPaymentInitiationAdjustmentsResponse) GetV3PaymentInitiationAdjustmentsCursorResponse() *components.V3PaymentInitiationAdjustmentsCursorResponse {
	if o == nil {
		return nil
	}
	return o.V3PaymentInitiationAdjustmentsCursorResponse
}

func (o *V3ListPaymentInitiationAdjustmentsResponse) GetV3ErrorResponse() *components.V3ErrorResponse {
	if o == nil {
		return nil
	}
	return o.V3ErrorResponse
}
