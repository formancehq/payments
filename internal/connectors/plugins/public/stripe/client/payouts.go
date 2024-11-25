package client

import (
	"context"
	"time"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/payout"
)

type CreatePayoutRequest struct {
	IdempotencyKey string
	Amount         int64
	Currency       string
	Source         *string
	Destination    string
	Description    string
	Metadata       map[string]string
}

func (c *client) CreatePayout(ctx context.Context, createPayoutRequest *CreatePayoutRequest) (*stripe.Payout, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "initiate_payout")

	params := &stripe.PayoutParams{
		Params: stripe.Params{
			Context:       ctx,
			StripeAccount: createPayoutRequest.Source,
		},
		Amount:      stripe.Int64(createPayoutRequest.Amount),
		Currency:    stripe.String(createPayoutRequest.Currency),
		Destination: stripe.String(createPayoutRequest.Destination),
		Metadata:    createPayoutRequest.Metadata,
		Method:      stripe.String("standard"),
	}

	params.AddExpand("balance_transaction")

	if createPayoutRequest.IdempotencyKey != "" {
		params.IdempotencyKey = stripe.String(createPayoutRequest.IdempotencyKey)
	}

	if createPayoutRequest.Description != "" {
		params.Description = stripe.String(createPayoutRequest.Description)
	}

	payoutResponse, err := payout.New(params)
	if err != nil {
		return nil, wrapSDKErr(err)
	}

	return payoutResponse, nil
}
