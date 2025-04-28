package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type PayoutRequest struct {}

type PayoutResponse struct {}

func (c *client) InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_payout")

	// TODO: call PSP to create payout
    return nil, nil
}
