package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type CreateTransferRequest struct {
	ConnectorID models.ConnectorID
	Req         models.CreateTransferRequest
}

func (a Activities) PluginCreateTransfer(ctx context.Context, request CreateTransferRequest) (*models.CreateTransferResponse, error) {
	plugin, err := a.plugins.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(err)
	}

	resp, err := plugin.CreateTransfer(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(err)
	}
	return &resp, nil
}

var PluginCreateTransferActivity = Activities{}.PluginCreateTransfer

func PluginCreateTransfer(ctx workflow.Context, connectorID models.ConnectorID, request models.CreateTransferRequest) (*models.CreateTransferResponse, error) {
	ret := models.CreateTransferResponse{}
	if err := executeActivity(ctx, PluginCreateTransferActivity, &ret, CreateTransferRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
