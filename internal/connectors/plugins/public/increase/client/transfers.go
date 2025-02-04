package client

import (
	"context"

	"github.com/Increase/increase-go"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

func (c *client) InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_transfer")

	params := increase.ACHTransferNewParams{
		AccountID:           increase.F(tr.SourceAccountID),
		Amount:              increase.F(tr.Amount),
		StatementDescriptor: increase.F(tr.Description),
		AccountNumber:       increase.F(tr.DestinationAccountID),
	}

	resp, err := c.increaseClient.ACHTransfers.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return &TransferResponse{
		ID:          string(resp.ID),
		Status:      string(resp.Status),
		Amount:      resp.Amount,
		Currency:    string(resp.Currency),
		Description: resp.StatementDescriptor,
		CreatedAt:   resp.CreatedAt.String(),
	}, nil
}
