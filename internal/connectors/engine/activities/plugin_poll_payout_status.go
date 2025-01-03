package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type PollPayoutStatusRequest struct {
	ConnectorID models.ConnectorID
	Req         models.PollPayoutStatusRequest
}

func (a Activities) PluginPollPayoutStatus(ctx context.Context, request PollPayoutStatusRequest) (*models.PollPayoutStatusResponse, error) {
	plugin, err := a.plugins.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(err)
	}

	resp, err := plugin.PollPayoutStatus(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(err)
	}
	return &resp, nil
}

var PluginPollPayoutStatusActivity = Activities{}.PluginPollPayoutStatus

func PluginPollPayoutStatus(ctx workflow.Context, connectorID models.ConnectorID, request models.PollPayoutStatusRequest) (*models.PollPayoutStatusResponse, error) {
	ret := models.PollPayoutStatusResponse{}
	if err := executeActivity(ctx, PluginPollPayoutStatusActivity, &ret, PollPayoutStatusRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
