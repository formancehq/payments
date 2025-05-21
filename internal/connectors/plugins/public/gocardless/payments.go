package gocardless

import (
	"context"
	"encoding/json"
	"math/big"

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

	if len(payments) > req.PageSize {
		payments = payments[:req.PageSize]
	}

	if len(payments) > 0 {
		newState.After = payments[len(payments)-1].Reference
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

		pspPayment := models.PSPPayment{
			Reference:       payment.ID,
			ParentReference: payment.ID,
			Amount:          big.NewInt(int64(payment.Amount)),
			CreatedAt:       payment.CreatedAt,
			Status:          mapPaymentStatus(payment.Status),
			Asset:           currency.FormatAsset(SupportedCurrenciesWithDecimal, payment.Asset),
			Metadata:        extractExternalAccountMetadata(payment.Metadata),
			Raw:             payment.Raw,
			Type:            models.PAYMENT_TYPE_PAYOUT,
		}

		if payment.SourceAccountReference != "" {
			pspPayment.SourceAccountReference = &payment.SourceAccountReference
		}

		if payment.DestinationAccountReference != "" {
			pspPayment.DestinationAccountReference = &payment.DestinationAccountReference
		}

		payments = append(payments, pspPayment)

	}

	return payments, nil
}

func mapPaymentStatus(gcStatus string) models.PaymentStatus {
	switch gcStatus {
	case "submitted", "pending_customer_approval", "pending_submission":
		return models.PAYMENT_STATUS_PENDING
	case "confirmed", "paid_out":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "cancelled":
		return models.PAYMENT_STATUS_CANCELLED
	case "customer_approval_denied", "failed":
		return models.PAYMENT_STATUS_FAILED
	case "charged_back":
		return models.PAYMENT_STATUS_REFUNDED
	default:
		return models.PAYMENT_STATUS_OTHER
	}
}
