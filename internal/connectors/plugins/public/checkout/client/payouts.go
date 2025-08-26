package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"

	"github.com/checkout/checkout-sdk-go/common"
	"github.com/checkout/checkout-sdk-go/payments/nas"
	"github.com/checkout/checkout-sdk-go/payments/nas/sources"
)

type PayoutRequest struct {
	SourceEntityID string
	DestinationInstrumentID string

	Amount        		 int64
	Currency      		 string
	BillingDescriptor    string
	Reference     		 string
	IdempotencyKey		 string
}

type PayoutResponse struct {
	ID        string
	Status    string
	Reference string
}

func (c *client) InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_payout")

	src := sources.NewRequestIdSource()
	src.Id = pr.SourceEntityID
	dest := nas.NewRequestIdDestination()
	dest.Id = pr.DestinationInstrumentID

	var billing *nas.PayoutBillingDescriptor
	if pr.BillingDescriptor != "" {
		billing = &nas.PayoutBillingDescriptor{Reference: pr.BillingDescriptor}
	}

	req := nas.PayoutRequest{
		Source:    src,
		Destination: dest,
		Amount:    pr.Amount,
		Currency:  common.Currency(pr.Currency),
		Reference: pr.Reference,
		ProcessingChannelId: c.processingChannelId,
		BillingDescriptor:   billing,
	}

	res, err := c.sdk.Payments.RequestPayout(req, &pr.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	out := &PayoutResponse{
		ID:        res.Id,
		Status:    string(res.Status),
		Reference: res.Reference,
	}
	return out, nil
}
