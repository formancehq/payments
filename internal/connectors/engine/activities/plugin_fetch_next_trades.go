package activities

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type FetchNextTradesRequest struct {
	ConnectorID models.ConnectorID
	Req         models.FetchNextTradesRequest
	Periodic    bool
}

func (a Activities) PluginFetchNextTrades(ctx context.Context, request FetchNextTradesRequest) (*models.FetchNextTradesResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.FetchNextTrades(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginPollingError(ctx, err, request.Periodic)
	}

	return &resp, nil
}

var PluginFetchNextTradesActivity = Activities{}.PluginFetchNextTrades

func PluginFetchNextTrades(ctx workflow.Context, connectorID models.ConnectorID, fromPayload, state json.RawMessage, pageSize int, periodic bool) (*models.FetchNextTradesResponse, error) {
	ret := models.FetchNextTradesResponse{}
	if err := executeActivity(ctx, PluginFetchNextTradesActivity, &ret, FetchNextTradesRequest{
		ConnectorID: connectorID,
		Req: models.FetchNextTradesRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    pageSize,
		},
		Periodic: periodic,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}

