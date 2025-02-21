package client

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
)

type PaymentPayload struct {
	Mandate string `url:"mandate,omitempty" json:"mandate,omitempty"`
}

type GocardlessPayment struct {
	ID                          string            `json:"id,omitempty"`
	CreatedAt                   int64             `json:"created_at,omitempty"`
	Amount                      int               `json:"amount,omitempty"`
	Status                      string            `json:"status,omitempty"`
	Asset                       string            `json:"asset,omitempty"`
	Metadata                    map[string]string `json:"metadata,omitempty"`
	SourceAccountReference      string            `json:"sourceAccountReference,omitempty"`
	DestinationAccountReference string            `json:"destinationAccountReference,omitempty"`
}

func (c *client) GetPayments(ctx context.Context, payload PaymentPayload, pageSize int, after string, before string) (
	[]GocardlessPayment, Cursor, error,
) {

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_payments")

	paymentsResponse, err := c.service.Payments.List(ctx, gocardless.PaymentListParams{
		Limit:   pageSize,
		After:   after,
		Before:  before,
		Mandate: payload.Mandate,
	})

	if err != nil {
		return nil, Cursor{}, err
	}

	var payments []GocardlessPayment
	for _, payment := range paymentsResponse.Payments {
		parsedTime, err := time.Parse(time.RFC3339Nano, payment.CreatedAt)
		if err != nil {
			return []GocardlessPayment{}, Cursor{}, fmt.Errorf("failed to parse creation time: %w", err)
		}

		mandate, err := c.GetMandate(ctx, payment.Links.Mandate)

		if err != nil {
			return []GocardlessPayment{}, Cursor{}, err
		}

		payments = append(payments, GocardlessPayment{
			ID:                          payment.Id,
			CreatedAt:                   parsedTime.Unix(),
			Amount:                      payment.Amount,
			Status:                      payment.Status,
			Asset:                       payment.Currency,
			Metadata:                    convertMetadata(payment.Metadata),
			SourceAccountReference:      mandate.Links.Creditor,
			DestinationAccountReference: mandate.Links.Customer,
		})
	}
	nextCursor := Cursor{
		After:  paymentsResponse.Meta.Cursors.After,
		Before: paymentsResponse.Meta.Cursors.Before,
	}
	return payments, nextCursor, nil
}

func convertMetadata(in map[string]interface{}) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		out[k] = fmt.Sprint(v)
	}
	return out
}
