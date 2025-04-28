package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type TransferRequest struct {}

type TransferResponse struct {}

func (c *client) InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_transfer")

	// TODO: call PSP to create transfer
    return nil, nil
}
