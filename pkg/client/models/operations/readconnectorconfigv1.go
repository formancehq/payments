// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type ReadConnectorConfigV1Request struct {
	// The name of the connector.
	Connector components.Connector `pathParam:"style=simple,explode=false,name=connector"`
	// The connector ID.
	ConnectorID string `pathParam:"style=simple,explode=false,name=connectorId"`
}

func (o *ReadConnectorConfigV1Request) GetConnector() components.Connector {
	if o == nil {
		return components.Connector("")
	}
	return o.Connector
}

func (o *ReadConnectorConfigV1Request) GetConnectorID() string {
	if o == nil {
		return ""
	}
	return o.ConnectorID
}

type ReadConnectorConfigV1Response struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// OK
	ConnectorConfigResponse *components.ConnectorConfigResponse
	// Error
	PaymentsErrorResponse *components.PaymentsErrorResponse
}

func (o *ReadConnectorConfigV1Response) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *ReadConnectorConfigV1Response) GetConnectorConfigResponse() *components.ConnectorConfigResponse {
	if o == nil {
		return nil
	}
	return o.ConnectorConfigResponse
}

func (o *ReadConnectorConfigV1Response) GetPaymentsErrorResponse() *components.PaymentsErrorResponse {
	if o == nil {
		return nil
	}
	return o.PaymentsErrorResponse
}
