package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type PollTransferStatusRequest struct {
	ConnectorID models.ConnectorID
	Req         models.PollTransferStatusRequest
}

func (a Activities) PluginPollTransferStatus(ctx context.Context, request PollTransferStatusRequest) (*models.PollTransferStatusResponse, error) {
	plugin, err := a.plugins.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(err)
	}

	resp, err := plugin.PollTransferStatus(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(err)
	}
	return &resp, nil
}

var PluginPollTransferStatusActivity = Activities{}.PluginPollTransferStatus

func PluginPollTransferStatus(ctx workflow.Context, connectorID models.ConnectorID, request models.PollTransferStatusRequest) (*models.PollTransferStatusResponse, error) {
	ret := models.PollTransferStatusResponse{}
	if err := executeActivity(ctx, PluginPollTransferStatusActivity, &ret, PollTransferStatusRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
