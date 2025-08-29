package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type TranslateWebhookRequest struct {
	ConnectorID models.ConnectorID
	Req         models.TranslateWebhookRequest
}

func (a Activities) PluginTranslateWebhook(ctx context.Context, request TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.TranslateWebhook(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}
	return &resp, nil
}

var PluginTranslateWebhookActivity = Activities{}.PluginTranslateWebhook

func PluginTranslateWebhook(ctx workflow.Context, connectorID models.ConnectorID, request models.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
	ret := models.TranslateWebhookResponse{}
	if err := executeActivity(ctx, PluginTranslateWebhookActivity, &ret, TranslateWebhookRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
