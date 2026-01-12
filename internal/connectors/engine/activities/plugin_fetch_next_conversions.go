package activities

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type FetchNextConversionsRequest struct {
	ConnectorID models.ConnectorID
	Req         models.FetchNextConversionsRequest
	Periodic    bool
}

func (a Activities) PluginFetchNextConversions(ctx context.Context, request FetchNextConversionsRequest) (*models.FetchNextConversionsResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.FetchNextConversions(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginPollingError(ctx, err, request.Periodic)
	}
	return &resp, nil
}

var PluginFetchNextConversionsActivity = Activities{}.PluginFetchNextConversions

func PluginFetchNextConversions(ctx workflow.Context, connectorID models.ConnectorID, fromPayload, state json.RawMessage, pageSize int, periodic bool) (*models.FetchNextConversionsResponse, error) {
	ret := models.FetchNextConversionsResponse{}
	if err := executeActivity(ctx, PluginFetchNextConversionsActivity, &ret, FetchNextConversionsRequest{
		ConnectorID: connectorID,
		Req: models.FetchNextConversionsRequest{
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
