// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type GetServerInfoResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Server information
	ServerInfo *components.ServerInfo
	// Error
	PaymentsErrorResponse *components.PaymentsErrorResponse
}

func (o *GetServerInfoResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *GetServerInfoResponse) GetServerInfo() *components.ServerInfo {
	if o == nil {
		return nil
	}
	return o.ServerInfo
}

func (o *GetServerInfoResponse) GetPaymentsErrorResponse() *components.PaymentsErrorResponse {
	if o == nil {
		return nil
	}
	return o.PaymentsErrorResponse
}
