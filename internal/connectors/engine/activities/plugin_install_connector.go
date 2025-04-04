package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type InstallConnectorRequest struct {
	ConnectorID models.ConnectorID
	Req         models.InstallRequest
}

func (a Activities) PluginInstallConnector(ctx context.Context, request InstallConnectorRequest) (*models.InstallResponse, error) {
	plugin, err := a.plugins.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.Install(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	return &resp, nil
}

var PluginInstallConnectorActivity = Activities{}.PluginInstallConnector

func PluginInstallConnector(ctx workflow.Context, connectorID models.ConnectorID) (*models.InstallResponse, error) {
	ret := models.InstallResponse{}
	if err := executeActivity(ctx, PluginInstallConnectorActivity, &ret, InstallConnectorRequest{
		ConnectorID: connectorID,
		Req:         models.InstallRequest{
			ConnectorID: connectorID.String(),
		},
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
