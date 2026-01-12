package activities

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type FetchNextOrdersRequest struct {
	ConnectorID models.ConnectorID
	Req         models.FetchNextOrdersRequest
	Periodic    bool
}

func (a Activities) PluginFetchNextOrders(ctx context.Context, request FetchNextOrdersRequest) (*models.FetchNextOrdersResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.FetchNextOrders(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginPollingError(ctx, err, request.Periodic)
	}
	return &resp, nil
}

var PluginFetchNextOrdersActivity = Activities{}.PluginFetchNextOrders

func PluginFetchNextOrders(ctx workflow.Context, connectorID models.ConnectorID, fromPayload, state json.RawMessage, pageSize int, periodic bool) (*models.FetchNextOrdersResponse, error) {
	ret := models.FetchNextOrdersResponse{}
	if err := executeActivity(ctx, PluginFetchNextOrdersActivity, &ret, FetchNextOrdersRequest{
		ConnectorID: connectorID,
		Req: models.FetchNextOrdersRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    pageSize,
		},
		Periodic: periodic,
	},
	); err != nil {
		return nil, err
	}
	return &ret, nil
}
