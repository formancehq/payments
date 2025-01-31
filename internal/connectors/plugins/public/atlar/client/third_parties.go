package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/get-momo/atlar-v1-go-client/client/third_parties"
)

func (c *client) GetV1BetaThirdPartiesID(ctx context.Context, id string) (*third_parties.GetV1betaThirdPartiesIDOK, error) {
	params := third_parties.GetV1betaThirdPartiesIDParams{
		Context:    metrics.OperationContext(ctx, "get_third_party"),
		ID:         id,
		HTTPClient: c.httpClient,
	}

	resp, err := c.client.ThirdParties.GetV1betaThirdPartiesID(&params)
	return resp, wrapSDKErr(err, &third_parties.GetV1betaThirdPartiesIDNotFound{})
}
