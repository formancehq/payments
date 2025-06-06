// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type AddAccountToPoolRequest struct {
	// The pool ID.
	PoolID                  string                             `pathParam:"style=simple,explode=false,name=poolId"`
	AddAccountToPoolRequest components.AddAccountToPoolRequest `request:"mediaType=application/json"`
}

func (o *AddAccountToPoolRequest) GetPoolID() string {
	if o == nil {
		return ""
	}
	return o.PoolID
}

func (o *AddAccountToPoolRequest) GetAddAccountToPoolRequest() components.AddAccountToPoolRequest {
	if o == nil {
		return components.AddAccountToPoolRequest{}
	}
	return o.AddAccountToPoolRequest
}

type AddAccountToPoolResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Error
	PaymentsErrorResponse *components.PaymentsErrorResponse
}

func (o *AddAccountToPoolResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *AddAccountToPoolResponse) GetPaymentsErrorResponse() *components.PaymentsErrorResponse {
	if o == nil {
		return nil
	}
	return o.PaymentsErrorResponse
}
