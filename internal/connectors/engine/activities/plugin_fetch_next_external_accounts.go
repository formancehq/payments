package activities

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type FetchNextExternalAccountsRequest struct {
	ConnectorID models.ConnectorID
	Req         models.FetchNextExternalAccountsRequest
	Periodic    bool
}

func (a Activities) PluginFetchNextExternalAccounts(ctx context.Context, request FetchNextExternalAccountsRequest) (*models.FetchNextExternalAccountsResponse, error) {
	plugin, err := a.connectors.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.FetchNextExternalAccounts(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginPollingError(ctx, err, request.Periodic)
	}

	return &resp, nil
}

var PluginFetchNextExternalAccountsActivity = Activities{}.PluginFetchNextExternalAccounts

func PluginFetchNextExternalAccounts(ctx workflow.Context, connectorID models.ConnectorID, fromPayload, state json.RawMessage, pageSize int, periodic bool) (*models.FetchNextExternalAccountsResponse, error) {
	ret := models.FetchNextExternalAccountsResponse{}
	if err := executeActivity(ctx, PluginFetchNextExternalAccountsActivity, &ret, FetchNextExternalAccountsRequest{
		ConnectorID: connectorID,
		Req: models.FetchNextExternalAccountsRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    pageSize,
		},
		Periodic: periodic,
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
