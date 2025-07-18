package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type CompleteUserLinkRequest struct {
	ConnectorID models.ConnectorID
	Req         models.CompleteUserLinkRequest
}

func (a Activities) PluginCompleteUserLink(ctx context.Context, request CompleteUserLinkRequest) (*models.CompleteUserLinkResponse, error) {
	plugin, err := a.plugins.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.CompleteUserLink(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	return &resp, nil
}

var PluginCompleteUserLinkActivity = Activities{}.PluginCompleteUserLink

func PluginCompleteUserLink(ctx workflow.Context, connectorID models.ConnectorID, request models.CompleteUserLinkRequest) (*models.CompleteUserLinkResponse, error) {
	ret := models.CompleteUserLinkResponse{}
	if err := executeActivity(ctx, PluginCompleteUserLinkActivity, &ret, CompleteUserLinkRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
