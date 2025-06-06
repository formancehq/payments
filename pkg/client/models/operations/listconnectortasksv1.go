// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
	"github.com/formancehq/payments/pkg/client/models/components"
)

type ListConnectorTasksV1Request struct {
	// The name of the connector.
	Connector components.Connector `pathParam:"style=simple,explode=false,name=connector"`
	// The connector ID.
	ConnectorID string `pathParam:"style=simple,explode=false,name=connectorId"`
	// The maximum number of results to return per page.
	//
	PageSize *int64 `default:"15" queryParam:"style=form,explode=true,name=pageSize"`
	// Parameter used in pagination requests. Maximum page size is set to 15.
	// Set to the value of next for the next page of results.
	// Set to the value of previous for the previous page of results.
	// No other parameters can be set when this parameter is set.
	//
	Cursor *string `queryParam:"style=form,explode=true,name=cursor"`
}

func (l ListConnectorTasksV1Request) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(l, "", false)
}

func (l *ListConnectorTasksV1Request) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &l, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *ListConnectorTasksV1Request) GetConnector() components.Connector {
	if o == nil {
		return components.Connector("")
	}
	return o.Connector
}

func (o *ListConnectorTasksV1Request) GetConnectorID() string {
	if o == nil {
		return ""
	}
	return o.ConnectorID
}

func (o *ListConnectorTasksV1Request) GetPageSize() *int64 {
	if o == nil {
		return nil
	}
	return o.PageSize
}

func (o *ListConnectorTasksV1Request) GetCursor() *string {
	if o == nil {
		return nil
	}
	return o.Cursor
}

type ListConnectorTasksV1Response struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// OK
	TasksCursor *components.TasksCursor
	// Error
	PaymentsErrorResponse *components.PaymentsErrorResponse
}

func (o *ListConnectorTasksV1Response) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *ListConnectorTasksV1Response) GetTasksCursor() *components.TasksCursor {
	if o == nil {
		return nil
	}
	return o.TasksCursor
}

func (o *ListConnectorTasksV1Response) GetPaymentsErrorResponse() *components.PaymentsErrorResponse {
	if o == nil {
		return nil
	}
	return o.PaymentsErrorResponse
}
