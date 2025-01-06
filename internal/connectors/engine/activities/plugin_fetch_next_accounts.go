package activities

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type FetchNextAccountsRequest struct {
	ConnectorID models.ConnectorID
	Req         models.FetchNextAccountsRequest
	Periodic    bool
}

func (a Activities) PluginFetchNextAccounts(ctx context.Context, request FetchNextAccountsRequest) (*models.FetchNextAccountsResponse, error) {
	plugin, err := a.plugins.Get(request.ConnectorID)
	if err != nil {
		return nil, a.temporalPluginError(ctx, err)
	}

	resp, err := plugin.FetchNextAccounts(ctx, request.Req)
	if err != nil {
		return nil, a.temporalPluginPollingError(ctx, err, request.Periodic)
	}
	return &resp, nil
}

var PluginFetchNextAccountsActivity = Activities{}.PluginFetchNextAccounts

func PluginFetchNextAccounts(ctx workflow.Context, connectorID models.ConnectorID, fromPayload, state json.RawMessage, pageSize int, periodic bool) (*models.FetchNextAccountsResponse, error) {
	ret := models.FetchNextAccountsResponse{}
	if err := executeActivity(ctx, PluginFetchNextAccountsActivity, &ret, FetchNextAccountsRequest{
		ConnectorID: connectorID,
		Req: models.FetchNextAccountsRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    pageSize,
		},
		Periodic: periodic,
	},
	); err != nil {
		return nil, err
	}
	return &ret, nil
}
