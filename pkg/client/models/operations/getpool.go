// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type GetPoolRequest struct {
	// The pool ID.
	PoolID string `pathParam:"style=simple,explode=false,name=poolId"`
}

func (o *GetPoolRequest) GetPoolID() string {
	if o == nil {
		return ""
	}
	return o.PoolID
}

type GetPoolResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// OK
	PoolResponse *components.PoolResponse
	// Error
	PaymentsErrorResponse *components.PaymentsErrorResponse
}

func (o *GetPoolResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *GetPoolResponse) GetPoolResponse() *components.PoolResponse {
	if o == nil {
		return nil
	}
	return o.PoolResponse
}

func (o *GetPoolResponse) GetPaymentsErrorResponse() *components.PaymentsErrorResponse {
	if o == nil {
		return nil
	}
	return o.PaymentsErrorResponse
}
