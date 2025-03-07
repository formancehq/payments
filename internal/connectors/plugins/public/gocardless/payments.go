package gocardless

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	After string `url:"after,omitempty" json:"after,omitempty"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState

	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	newState := paymentsState{
		After: oldState.After,
	}

	var payments []models.PSPPayment

	hasMore := false

	pagedPayments, nextCursor, err := p.client.GetPayments(ctx, req.PageSize, oldState.After)

	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	newState.After = nextCursor.After

	payments, err = fillPayments(pagedPayments, payments)

	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	hasMore = nextCursor.After != ""

	if !hasMore && len(payments) > 0 {
		newState.After = payments[len(payments)-1].Reference
	}

	if len(payments) > req.PageSize {
		payments = payments[:req.PageSize]
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func fillPayments(
	pagedPayments []client.GocardlessPayment,
	payments []models.PSPPayment,
) ([]models.PSPPayment, error) {

	for _, payment := range pagedPayments {

		createdAt := time.Unix(payment.CreatedAt, 0)
		raw, err := json.Marshal(payment)

		if err != nil {
			return []models.PSPPayment{}, err
		}

		payments = append(payments, models.PSPPayment{
			Reference:                   payment.ID,
			Amount:                      big.NewInt(int64(payment.Amount)),
			CreatedAt:                   createdAt,
			Status:                      mapPaymentStatus(payment.Status),
			Asset:                       currency.FormatAsset(SupportedCurrenciesWithDecimal, payment.Asset),
			Metadata:                    payment.Metadata,
			SourceAccountReference:      &payment.SourceAccountReference,
			DestinationAccountReference: &payment.DestinationAccountReference,
			Raw:                         raw,
		})

	}

	return payments, nil
}

func mapPaymentStatus(gcStatus string) models.PaymentStatus {
	switch gcStatus {
	case "pending":
		return models.PAYMENT_STATUS_PENDING
	case "paid":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "failed":
		return models.PAYMENT_STATUS_FAILED
	default:
		return models.PAYMENT_STATUS_OTHER
	}
}
