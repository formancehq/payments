package client

import (
	"context"
	"time"

	"github.com/get-momo/atlar-v1-go-client/client/third_parties"
)

func (c *client) GetV1BetaThirdPartiesID(ctx context.Context, id string) (*third_parties.GetV1betaThirdPartiesIDOK, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "get_third_party")

	params := third_parties.GetV1betaThirdPartiesIDParams{
		Context: ctx,
		ID:      id,
	}

	resp, err := c.client.ThirdParties.GetV1betaThirdPartiesID(&params)
	return resp, wrapSDKErr(err)
}
