package activities

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type InstallConnectorRequest struct {
	ConnectorID models.ConnectorID
	Req         models.InstallRequest
}

func (a Activities) PluginInstallConnector(ctx context.Context, request InstallConnectorRequest) (*models.InstallResponse, error) {
	plugin, err := a.plugins.Get(request.ConnectorID)
	if err != nil {
		return nil, err
	}

	resp, err := plugin.Install(ctx, request.Req)
	if err != nil {
		if plgErr, ok := err.(*models.PluginError); ok {
			return nil, plgErr.TemporalError()
		}
		return nil, temporal.NewApplicationErrorWithCause(err.Error(), ErrTypeUnhandled, err)
	}

	return &resp, err
}

var PluginInstallConnectorActivity = Activities{}.PluginInstallConnector

func PluginInstallConnector(ctx workflow.Context, connectorID models.ConnectorID, config json.RawMessage) (*models.InstallResponse, error) {
	ret := models.InstallResponse{}
	if err := executeActivity(ctx, PluginInstallConnectorActivity, &ret, InstallConnectorRequest{
		ConnectorID: connectorID,
		Req: models.InstallRequest{
			Config: config,
		},
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
