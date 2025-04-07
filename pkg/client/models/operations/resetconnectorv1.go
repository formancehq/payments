// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type ResetConnectorV1Request struct {
	// The name of the connector.
	Connector components.Connector `pathParam:"style=simple,explode=false,name=connector"`
	// The connector ID.
	ConnectorID string `pathParam:"style=simple,explode=false,name=connectorId"`
}

func (o *ResetConnectorV1Request) GetConnector() components.Connector {
	if o == nil {
		return components.Connector("")
	}
	return o.Connector
}

func (o *ResetConnectorV1Request) GetConnectorID() string {
	if o == nil {
		return ""
	}
	return o.ConnectorID
}

type ResetConnectorV1Response struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Error
	PaymentsErrorResponse *components.PaymentsErrorResponse
}

func (o *ResetConnectorV1Response) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *ResetConnectorV1Response) GetPaymentsErrorResponse() *components.PaymentsErrorResponse {
	if o == nil {
		return nil
	}
	return o.PaymentsErrorResponse
}
