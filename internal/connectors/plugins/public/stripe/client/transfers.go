package client

import (
    "context"

    pluginsdkmetrics "github.com/formancehq/payments/pkg/pluginsdk/metrics"
    "github.com/stripe/stripe-go/v79"
)

type CreateTransferRequest struct {
	IdempotencyKey string
	Amount         int64
	Currency       string
	Source         *string
	Destination    string
	Description    string
	Metadata       map[string]string
}

func (c *client) CreateTransfer(ctx context.Context, createTransferRequest *CreateTransferRequest) (*stripe.Transfer, error) {
	params := &stripe.TransferParams{
        Params: stripe.Params{
            Context:       pluginsdkmetrics.OperationContext(ctx, "initiate_transfer"),
			StripeAccount: createTransferRequest.Source,
		},
		Amount:      stripe.Int64(createTransferRequest.Amount),
		Currency:    stripe.String(createTransferRequest.Currency),
		Destination: stripe.String(createTransferRequest.Destination),
		Metadata:    createTransferRequest.Metadata,
	}

	params.AddExpand("balance_transaction")

	if createTransferRequest.IdempotencyKey != "" {
		params.IdempotencyKey = stripe.String(createTransferRequest.IdempotencyKey)
	}

	if createTransferRequest.Description != "" {
		params.Description = stripe.String(createTransferRequest.Description)
	}

	transferResponse, err := c.transferClient.New(params)
	if err != nil {
		return nil, wrapSDKErr(err)
	}

	return transferResponse, nil
}
