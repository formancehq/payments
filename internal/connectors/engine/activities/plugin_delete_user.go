package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type DeleteUserRequest struct {
	ConnectorID models.ConnectorID
	Req         models.DeleteUserRequest
}

func (a Activities) PluginDeleteUser(ctx context.Context, request DeleteUserRequest) (*models.DeleteUserResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.DeleteUser(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	return &resp, nil
}

var PluginDeleteUserActivity = Activities{}.PluginDeleteUser

func PluginDeleteUser(ctx workflow.Context, connectorID models.ConnectorID, request models.DeleteUserRequest) (*models.DeleteUserResponse, error) {
	ret := models.DeleteUserResponse{}
	if err := executeActivity(ctx, PluginDeleteUserActivity, &ret, DeleteUserRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
