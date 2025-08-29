package activities

import (
	context "context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type VerifyWebhookRequest struct {
	ConnectorID models.ConnectorID
	Req         models.VerifyWebhookRequest
}

func (a Activities) PluginVerifyWebhook(ctx context.Context, request VerifyWebhookRequest) (*models.VerifyWebhookResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.VerifyWebhook(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}
	return &resp, nil
}

var PluginVerifyWebhookActivity = Activities{}.PluginVerifyWebhook

func PluginVerifyWebhook(ctx workflow.Context, connectorID models.ConnectorID, request models.VerifyWebhookRequest) (*models.VerifyWebhookResponse, error) {
	ret := models.VerifyWebhookResponse{}
	if err := executeActivity(ctx, PluginVerifyWebhookActivity, &ret, VerifyWebhookRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
