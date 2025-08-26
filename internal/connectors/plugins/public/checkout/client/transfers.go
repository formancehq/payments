package client

import (
	"context"
	"fmt"
	"time"

	"github.com/checkout/checkout-sdk-go/common"
	"github.com/checkout/checkout-sdk-go/transfers"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type TransferRequest struct {
	Reference string `json:"reference,omitempty"`
	Reason    string `json:"reason,omitempty"`

	Source struct {
		EntityID string `json:"entity_id"`
		Currency string `json:"currency"`
	} `json:"source"`

	Destination struct {
		EntityID string `json:"entity_id"`
		Currency string `json:"currency"`
	} `json:"destination"`

	Amount int64  `json:"amount"`
	IdempotencyKey string `json:"-"`
}

type TransferResponse struct {
	ID        string          `json:"id"`
	Status    string          `json:"status,omitempty"`
	CreatedOn *time.Time      `json:"created_on,omitempty"`
	Raw       any             `json:"raw,omitempty"`
}

type sdkTransferRequest = transfers.TransferRequest

func (c *client) InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_transfer")

	req := transfers.TransferRequest{
		Reference: tr.Reference,
		TransferType: "commission",
		Source: &transfers.TransferSourceRequest{
			Id: tr.Source.EntityID,
			Amount: tr.Amount,
			Currency: common.Currency(tr.Source.Currency),
		},
		Destination: &transfers.TransferDestinationRequest{
			Id: tr.Destination.EntityID,
		},
	}

	var idem *string
	if tr.IdempotencyKey != "" {
		idem = &tr.IdempotencyKey
	}

	resp, err := c.sdk.Transfers.InitiateTransferOfFounds(req, idem)
	if err != nil {
		return nil, fmt.Errorf("checkout.accounts.transfers: %w", err)
	}

	out := &TransferResponse{
		ID:        resp.Id,
		Status:    resp.Status,
		Raw:       resp,
	}
	return out, nil
}
