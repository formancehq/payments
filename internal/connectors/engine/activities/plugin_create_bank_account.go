package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type CreateBankAccountRequest struct {
	ConnectorID models.ConnectorID
	Req         models.CreateBankAccountRequest
}

func (a Activities) PluginCreateBankAccount(ctx context.Context, request CreateBankAccountRequest) (*models.CreateBankAccountResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.CreateBankAccount(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}
	return &resp, nil
}

var PluginCreateBankAccountActivity = Activities{}.PluginCreateBankAccount

func PluginCreateBankAccount(ctx workflow.Context, connectorID models.ConnectorID, request models.CreateBankAccountRequest) (*models.CreateBankAccountResponse, error) {
	ret := models.CreateBankAccountResponse{}
	if err := executeActivity(ctx, PluginCreateBankAccountActivity, &ret, CreateBankAccountRequest{
		ConnectorID: connectorID,
		Req:         request,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
