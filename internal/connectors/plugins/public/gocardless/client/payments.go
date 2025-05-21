package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
)

type GocardlessPayment struct {
	ID                          string                 `json:"id,omitempty"`
	PayoutID                    string                 `json:"payout_id,omitempty"`
	CreatedAt                   time.Time              `json:"created_at,omitempty"`
	Amount                      int                    `json:"amount,omitempty"`
	Status                      string                 `json:"status,omitempty"`
	Asset                       string                 `json:"asset,omitempty"`
	Metadata                    map[string]interface{} `json:"metadata,omitempty"`
	SourceAccountReference      string                 `json:"sourceAccountReference,omitempty"`
	DestinationAccountReference string                 `json:"destinationAccountReference,omitempty"`
	Raw                         json.RawMessage        `json:"raw"`
}

func (c *client) GetPayments(ctx context.Context, pageSize int, after string) (
	[]GocardlessPayment, Cursor, error,
) {

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_payments")

	paymentsResponse, err := c.service.GetGocardlessPayments(ctx, gocardless.PaymentListParams{
		Limit:         pageSize,
		After:         after,
		SortDirection: "asc",
	})

	if err != nil {
		return nil, Cursor{}, err
	}

	var payments []GocardlessPayment
	for _, payment := range paymentsResponse.Payments {

		GocardlessMetadata := make(map[string]interface{})

		parsedTime, err := time.Parse(time.RFC3339Nano, payment.CreatedAt)
		if err != nil {
			return []GocardlessPayment{}, Cursor{}, fmt.Errorf("failed to parse creation time: %w", err)
		}

		var sourceAccountReference string
		var destinationAccountReference string

		if c.shouldFetchMandate {

			mandate, err := c.GetMandate(ctx, payment.Links.Mandate)

			if err != nil {
				return []GocardlessPayment{}, Cursor{}, err
			}

			sourceAccountReference = mandate.Links.CustomerBankAccount

			if payment.Links.Payout != "" {
				payout, err := c.service.GetGocardlessPayout(ctx, payment.Links.Payout)

				if err != nil {
					return []GocardlessPayment{}, Cursor{}, err
				}

				destinationAccountReference = payout.Links.CreditorBankAccount

			}
		}

		GocardlessMetadata[GocardlessFxEstimatedExchangeRateMetadataKey] = payment.Fx.EstimatedExchangeRate
		GocardlessMetadata[GocardlessFxExchangeRateMetadataKey] = payment.Fx.ExchangeRate
		GocardlessMetadata[GoCardlessFxAmountMetadataKey] = payment.Fx.FxAmount
		GocardlessMetadata[GoCardlessFxCurrencyMetadataKey] = payment.Fx.FxCurrency

		GocardlessMetadata[GocardlessLinkCreditorMetadataKey] = payment.Links.Creditor
		GocardlessMetadata[GocardlessLinkInstalmentScheduleMetadataKey] = payment.Links.InstalmentSchedule
		GocardlessMetadata[GocardlessMandateMetadataKey] = payment.Links.Mandate
		GocardlessMetadata[GocardlessPayoutMetadataKey] = payment.Links.Payout
		GocardlessMetadata[GocardlessSubscriptionMetadataKey] = payment.Links.Subscription

		GocardlessMetadata[GocardlessAmountRefundedMetadataKey] = payment.AmountRefunded
		GocardlessMetadata[GocardlessChargeDateMetadataKey] = payment.ChargeDate
		GocardlessMetadata[GocardlessDescriptionMetadataKey] = payment.Description
		GocardlessMetadata[GocardlessFasterAchMetadataKey] = payment.FasterAch
		GocardlessMetadata[GocardlessRetryIfPossibleMetadataKey] = payment.RetryIfPossible
		GocardlessMetadata[GocardlessReferenceMetadataKey] = payment.Reference

		raw, err := json.Marshal(payment)

		if err != nil {
			return []GocardlessPayment{}, Cursor{}, err
		}

		payments = append(payments, GocardlessPayment{
			ID:                          payment.Id,
			PayoutID:                    payment.Links.Payout,
			CreatedAt:                   parsedTime,
			Amount:                      payment.Amount,
			Status:                      payment.Status,
			Asset:                       payment.Currency,
			Metadata:                    GocardlessMetadata,
			SourceAccountReference:      sourceAccountReference,
			DestinationAccountReference: destinationAccountReference,
			Raw:                         raw,
		})
	}

	nextCursor := Cursor{
		After: paymentsResponse.Meta.Cursors.After,
	}
	return payments, nextCursor, nil
}
