package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type CreatePayoutRequest struct {
	ConnectorID models.ConnectorID
	Req         models.CreatePayoutRequest
}

func (a Activities) PluginCreatePayout(ctx context.Context, request CreatePayoutRequest) (*models.CreatePayoutResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.CreatePayout(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}
	return &resp, nil
}

var PluginCreatePayoutActivity = Activities{}.PluginCreatePayout

func PluginCreatePayout(ctx workflow.Context, connectorID models.ConnectorID, request models.CreatePayoutRequest) (*models.CreatePayoutResponse, error) {
	ret := models.CreatePayoutResponse{}
	if err := executeActivity(ctx, PluginCreatePayoutActivity, &ret, CreatePayoutRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
