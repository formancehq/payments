package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type CancelOrderRequest struct {
	ConnectorID models.ConnectorID
	Req         models.CancelOrderRequest
}

func (a Activities) PluginCancelOrder(ctx context.Context, request CancelOrderRequest) (*models.CancelOrderResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.CancelOrder(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}
	return &resp, nil
}

var PluginCancelOrderActivity = Activities{}.PluginCancelOrder

func PluginCancelOrder(ctx workflow.Context, connectorID models.ConnectorID, request models.CancelOrderRequest) (*models.CancelOrderResponse, error) {
	ret := models.CancelOrderResponse{}
	if err := executeActivity(ctx, PluginCancelOrderActivity, &ret, CancelOrderRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
