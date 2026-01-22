package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type CreateOrderRequest struct {
	ConnectorID models.ConnectorID
	Req         models.CreateOrderRequest
}

func (a Activities) PluginCreateOrder(ctx context.Context, request CreateOrderRequest) (*models.CreateOrderResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.CreateOrder(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}
	return &resp, nil
}

var PluginCreateOrderActivity = Activities{}.PluginCreateOrder

func PluginCreateOrder(ctx workflow.Context, connectorID models.ConnectorID, request models.CreateOrderRequest) (*models.CreateOrderResponse, error) {
	ret := models.CreateOrderResponse{}
	if err := executeActivity(ctx, PluginCreateOrderActivity, &ret, CreateOrderRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
