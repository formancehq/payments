package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type PollOrderStatusRequest struct {
	ConnectorID models.ConnectorID
	Req         models.PollOrderStatusRequest
}

func (a Activities) PluginPollOrderStatus(ctx context.Context, request PollOrderStatusRequest) (*models.PollOrderStatusResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.PollOrderStatus(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}
	return &resp, nil
}

var PluginPollOrderStatusActivity = Activities{}.PluginPollOrderStatus

func PluginPollOrderStatus(ctx workflow.Context, connectorID models.ConnectorID, request models.PollOrderStatusRequest) (*models.PollOrderStatusResponse, error) {
	ret := models.PollOrderStatusResponse{}
	if err := executeActivity(ctx, PluginPollOrderStatusActivity, &ret, PollOrderStatusRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
