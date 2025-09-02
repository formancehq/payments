package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type ReversePayoutRequest struct {
	ConnectorID models.ConnectorID
	Req         models.ReversePayoutRequest
}

func (a Activities) PluginReversePayout(ctx context.Context, request ReversePayoutRequest) (*models.ReversePayoutResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.ReversePayout(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	return &resp, nil
}

var PluginReversePayoutActivity = Activities{}.PluginReversePayout

func PluginReversePayout(ctx workflow.Context, connectorID models.ConnectorID, request models.ReversePayoutRequest) (*models.ReversePayoutResponse, error) {
	ret := models.ReversePayoutResponse{}
	if err := executeActivity(ctx, PluginReversePayoutActivity, &ret, ReversePayoutRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
