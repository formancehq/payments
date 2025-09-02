package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type UpdateUserLinkRequest struct {
	ConnectorID models.ConnectorID
	Req         models.UpdateUserLinkRequest
}

func (a Activities) PluginUpdateUserLink(ctx context.Context, request UpdateUserLinkRequest) (*models.UpdateUserLinkResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.UpdateUserLink(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	return &resp, nil
}

var PluginUpdateUserLinkActivity = Activities{}.PluginUpdateUserLink

func PluginUpdateUserLink(ctx workflow.Context, connectorID models.ConnectorID, request models.UpdateUserLinkRequest) (*models.UpdateUserLinkResponse, error) {
	ret := models.UpdateUserLinkResponse{}
	if err := executeActivity(ctx, PluginUpdateUserLinkActivity, &ret, UpdateUserLinkRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
