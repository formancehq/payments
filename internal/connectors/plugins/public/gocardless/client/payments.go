package client

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
)

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

func (c *client) GetPayments(ctx context.Context, pageSize int, after string) (
	[]GocardlessPayment, Cursor, error,
) {

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_payments")

	paymentsResponse, err := c.service.GetGocardlessPayments(ctx, gocardless.PaymentListParams{
		Limit: pageSize,
		After: after,
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

		sourceAccountReference := ""
		destinationAccountReference := ""

		if c.shouldFetchMandate {

			mandate, err := c.GetMandate(ctx, payment.Links.Mandate)

			if err != nil {
				return []GocardlessPayment{}, Cursor{}, err
			}

			sourceAccountReference = mandate.Links.Creditor
			destinationAccountReference = mandate.Links.Customer

		}

		payments = append(payments, GocardlessPayment{
			ID:                          payment.Id,
			CreatedAt:                   parsedTime.Unix(),
			Amount:                      payment.Amount,
			Status:                      payment.Status,
			Asset:                       payment.Currency,
			Metadata:                    convertGocardlessMetadata(payment.Metadata),
			SourceAccountReference:      sourceAccountReference,
			DestinationAccountReference: destinationAccountReference,
		})
	}
	nextCursor := Cursor{
		After: paymentsResponse.Meta.Cursors.After,
	}
	return payments, nextCursor, nil
}

func convertGocardlessMetadata(in map[string]interface{}) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		out[k] = fmt.Sprint(v)
	}
	return out
}
