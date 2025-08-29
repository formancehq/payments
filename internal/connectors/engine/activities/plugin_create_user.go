package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type CreateUserRequest struct {
	ConnectorID models.ConnectorID
	Req         models.CreateUserRequest
}

func (a Activities) PluginCreateUser(ctx context.Context, request CreateUserRequest) (*models.CreateUserResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.CreateUser(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	return &resp, nil
}

var PluginCreateUserActivity = Activities{}.PluginCreateUser

func PluginCreateUser(ctx workflow.Context, connectorID models.ConnectorID, request models.CreateUserRequest) (*models.CreateUserResponse, error) {
	ret := models.CreateUserResponse{}
	if err := executeActivity(ctx, PluginCreateUserActivity, &ret, CreateUserRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
