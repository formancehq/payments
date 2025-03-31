// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type ListConfigsAvailableConnectorsResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// OK
	ConnectorsConfigsResponse *components.ConnectorsConfigsResponse
	// Error
	PaymentsErrorResponse *components.PaymentsErrorResponse
}

func (o *ListConfigsAvailableConnectorsResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *ListConfigsAvailableConnectorsResponse) GetConnectorsConfigsResponse() *components.ConnectorsConfigsResponse {
	if o == nil {
		return nil
	}
	return o.ConnectorsConfigsResponse
}

func (o *ListConfigsAvailableConnectorsResponse) GetPaymentsErrorResponse() *components.PaymentsErrorResponse {
	if o == nil {
		return nil
	}
	return o.PaymentsErrorResponse
}
