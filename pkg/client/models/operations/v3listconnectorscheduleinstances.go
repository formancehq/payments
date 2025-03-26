// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type V3ListConnectorScheduleInstancesRequest struct {
	// The connector ID
	ConnectorID string `pathParam:"style=simple,explode=false,name=connectorID"`
	// The schedule ID
	ScheduleID string `pathParam:"style=simple,explode=false,name=scheduleID"`
	// The number of items to return
	PageSize *int64 `queryParam:"style=form,explode=true,name=pageSize"`
	// Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.
	//
	Cursor *string `queryParam:"style=form,explode=true,name=cursor"`
}

func (o *V3ListConnectorScheduleInstancesRequest) GetConnectorID() string {
	if o == nil {
		return ""
	}
	return o.ConnectorID
}

func (o *V3ListConnectorScheduleInstancesRequest) GetScheduleID() string {
	if o == nil {
		return ""
	}
	return o.ScheduleID
}

func (o *V3ListConnectorScheduleInstancesRequest) GetPageSize() *int64 {
	if o == nil {
		return nil
	}
	return o.PageSize
}

func (o *V3ListConnectorScheduleInstancesRequest) GetCursor() *string {
	if o == nil {
		return nil
	}
	return o.Cursor
}

type V3ListConnectorScheduleInstancesResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// OK
	V3ConnectorScheduleInstancesCursorResponse *components.V3ConnectorScheduleInstancesCursorResponse
	// Error
	V3ErrorResponse *components.V3ErrorResponse
}

func (o *V3ListConnectorScheduleInstancesResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *V3ListConnectorScheduleInstancesResponse) GetV3ConnectorScheduleInstancesCursorResponse() *components.V3ConnectorScheduleInstancesCursorResponse {
	if o == nil {
		return nil
	}
	return o.V3ConnectorScheduleInstancesCursorResponse
}

func (o *V3ListConnectorScheduleInstancesResponse) GetV3ErrorResponse() *components.V3ErrorResponse {
	if o == nil {
		return nil
	}
	return o.V3ErrorResponse
}
