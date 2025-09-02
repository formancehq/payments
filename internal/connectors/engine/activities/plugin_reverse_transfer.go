package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type ReverseTransferRequest struct {
	ConnectorID models.ConnectorID
	Req         models.ReverseTransferRequest
}

func (a Activities) PluginReverseTransfer(ctx context.Context, request ReverseTransferRequest) (*models.ReverseTransferResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.ReverseTransfer(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	return &resp, nil
}

var PluginReverseTransferActivity = Activities{}.PluginReverseTransfer

func PluginReverseTransfer(ctx workflow.Context, connectorID models.ConnectorID, request models.ReverseTransferRequest) (*models.ReverseTransferResponse, error) {
	ret := models.ReverseTransferResponse{}
	if err := executeActivity(ctx, PluginReverseTransferActivity, &ret, ReverseTransferRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
