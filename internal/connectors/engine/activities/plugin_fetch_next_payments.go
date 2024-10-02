package activities

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type FetchNextPaymentsRequest struct {
	ConnectorID models.ConnectorID
	Req         models.FetchNextPaymentsRequest
}

func (a Activities) PluginFetchNextPayments(ctx context.Context, request FetchNextPaymentsRequest) (*models.FetchNextPaymentsResponse, error) {
	plugin, err := a.plugins.Get(request.ConnectorID)
	if err != nil {
		return nil, temporalError(err, request.ConnectorID.Provider)
	}

	resp, err := plugin.FetchNextPayments(ctx, request.Req)
	if err != nil {
		return nil, temporalError(err, request.ConnectorID.Provider)
	}

	return &resp, nil
}

var PluginFetchNextPaymentsActivity = Activities{}.PluginFetchNextPayments

func PluginFetchNextPayments(ctx workflow.Context, connectorID models.ConnectorID, fromPayload, state json.RawMessage, pageSize int) (*models.FetchNextPaymentsResponse, error) {
	ret := models.FetchNextPaymentsResponse{}
	if err := executeActivity(ctx, PluginFetchNextPaymentsActivity, &ret, FetchNextPaymentsRequest{
		ConnectorID: connectorID,
		Req: models.FetchNextPaymentsRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    pageSize,
		},
	}); err != nil {
		return nil, err
	}
	return &ret, nil
}
