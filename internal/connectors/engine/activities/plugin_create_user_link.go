package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type CreateUserLinkRequest struct {
	ConnectorID models.ConnectorID
	Req         models.CreateUserLinkRequest
}

func (a Activities) PluginCreateUserLink(ctx context.Context, request CreateUserLinkRequest) (*models.CreateUserLinkResponse, error) {
	plugin, err := a.plugins.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.CreateUserLink(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	return &resp, nil
}

var PluginCreateUserLinkActivity = Activities{}.PluginCreateUserLink

func PluginCreateUserLink(ctx workflow.Context, connectorID models.ConnectorID, request models.CreateUserLinkRequest) (*models.CreateUserLinkResponse, error) {
	ret := models.CreateUserLinkResponse{}
	if err := executeActivity(ctx, PluginCreateUserLinkActivity, &ret, CreateUserLinkRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
