package client

import (
	"context"

	"github.com/Increase/increase-go"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

func (c *client) InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_transfer")

	params := &increase.ACHTransferCreateParams{
		AccountID:            increase.F(tr.SourceAccountID),
		Amount:              increase.F(tr.Amount),
		StatementDescriptor: increase.F(tr.Description),
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

func (c *client) InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_payout")

	params := &increase.ACHTransferCreateParams{
		AccountID:            increase.F(pr.AccountID),
		Amount:              increase.F(pr.Amount),
		StatementDescriptor: increase.F(pr.Description),
	}

	resp, err := c.increaseClient.ACHTransfers.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return &PayoutResponse{
		ID:          string(resp.ID),
		Status:      string(resp.Status),
		Amount:      resp.Amount,
		Currency:    string(resp.Currency),
		Description: resp.StatementDescriptor,
		CreatedAt:   resp.CreatedAt.String(),
	}, nil
}
