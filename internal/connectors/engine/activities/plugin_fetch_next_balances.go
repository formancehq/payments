package activities

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type FetchNextBalancesRequest struct {
	ConnectorID models.ConnectorID
	Req         models.FetchNextBalancesRequest
	Periodic    bool
}

func (a Activities) PluginFetchNextBalances(ctx context.Context, request FetchNextBalancesRequest) (*models.FetchNextBalancesResponse, error) {
	plugin, err := a.plugins.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.FetchNextBalances(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginPollingError(ctx, err, request.Periodic)
	}
	return &resp, nil
}

var PluginFetchNextBalancesActivity = Activities{}.PluginFetchNextBalances

func PluginFetchNextBalances(ctx workflow.Context, connectorID models.ConnectorID, fromPayload, state json.RawMessage, pageSize int, periodic bool) (*models.FetchNextBalancesResponse, error) {
	ret := models.FetchNextBalancesResponse{}
	if err := executeActivity(ctx, PluginFetchNextBalancesActivity, &ret, FetchNextBalancesRequest{
		ConnectorID: connectorID,
		Req: models.FetchNextBalancesRequest{
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
