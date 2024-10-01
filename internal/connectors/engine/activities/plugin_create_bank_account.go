package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type CreateBankAccountRequest struct {
	ConnectorID models.ConnectorID
	Req         models.CreateBankAccountRequest
}

func (a Activities) PluginCreateBankAccount(ctx context.Context, request CreateBankAccountRequest) (*models.CreateBankAccountResponse, error) {
	plugin, err := a.plugins.Get(request.ConnectorID)
	if err != nil {
		return nil, err
	}

	resp, err := plugin.CreateBankAccount(ctx, request.Req)
	if err != nil {
		if plgErr, ok := err.(*models.PluginError); ok {
			return nil, plgErr.TemporalError()
		}
		return nil, temporal.NewApplicationErrorWithCause(err.Error(), ErrTypeUnhandled, err)
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
