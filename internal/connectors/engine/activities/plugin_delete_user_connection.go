package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type DeleteUserConnectionRequest struct {
	ConnectorID models.ConnectorID
	Req         models.DeleteUserConnectionRequest
}

func (a Activities) PluginDeleteUserConnection(ctx context.Context, request DeleteUserConnectionRequest) (*models.DeleteUserConnectionResponse, error) {
	plugin, err := a.plugins.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.DeleteUserConnection(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	return &resp, nil
}

var PluginDeleteUserConnectionActivity = Activities{}.PluginDeleteUserConnection

func PluginDeleteUserConnection(ctx workflow.Context, connectorID models.ConnectorID, request models.DeleteUserConnectionRequest) (*models.DeleteUserConnectionResponse, error) {
	ret := models.DeleteUserConnectionResponse{}
	if err := executeActivity(ctx, PluginDeleteUserConnectionActivity, &ret, DeleteUserConnectionRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
