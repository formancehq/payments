package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/stripe/stripe-go/v80"
)

type ReverseTransferRequest struct {
	IdempotencyKey   string
	StripeTransferID string
	Account          *string
	Amount           int64
	Description      string
	Metadata         map[string]string
}

func (c *client) ReverseTransfer(ctx context.Context, reverseTransferRequest ReverseTransferRequest) (*stripe.TransferReversal, error) {
	params := &stripe.TransferReversalParams{
		Params: stripe.Params{
			Context:       metrics.OperationContext(ctx, "reverse_transfer"),
			StripeAccount: reverseTransferRequest.Account,
		},
		ID:          stripe.String(reverseTransferRequest.StripeTransferID),
		Amount:      stripe.Int64(reverseTransferRequest.Amount),
		Description: stripe.String(reverseTransferRequest.Description),
		Metadata:    reverseTransferRequest.Metadata,
	}

	params.AddExpand("balance_transaction")
	params.AddExpand("transfer")
	params.AddExpand("transfer.balance_transaction")
	if reverseTransferRequest.IdempotencyKey != "" {
		params.IdempotencyKey = stripe.String(reverseTransferRequest.IdempotencyKey)
	}

	if reverseTransferRequest.Description != "" {
		params.Description = stripe.String(reverseTransferRequest.Description)
	}

	reversalResponse, err := c.transferReversalClient.New(params)
	if err != nil {
		return nil, wrapSDKErr(err)
	}

	return reversalResponse, nil
}
