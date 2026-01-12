package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/genericclient"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type paymentsState struct {
	LastUpdatedAtFrom time.Time `json:"lastUpdatedAtFrom"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	newState := paymentsState{
		LastUpdatedAtFrom: oldState.LastUpdatedAtFrom,
	}

	payments := make([]models.PSPPayment, 0)
	updatedAts := make([]time.Time, 0)
	needMore := false
	hasMore := false
	for page := 1; ; page++ {
		pagedPayments, err := p.client.ListTransactions(ctx, int64(page), int64(req.PageSize), oldState.LastUpdatedAtFrom)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payments, updatedAts, err = fillPayments(pagedPayments, payments, updatedAts, oldState)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(payments, pagedPayments, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		payments = payments[:req.PageSize]
		updatedAts = updatedAts[:req.PageSize]
	}

	if len(updatedAts) > 0 {
		newState.LastUpdatedAtFrom = updatedAts[len(payments)-1]
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
	pagedPayments []genericclient.Transaction,
	payments []models.PSPPayment,
	updatedAts []time.Time,
	oldState paymentsState,
) ([]models.PSPPayment, []time.Time, error) {
	for _, payment := range pagedPayments {
		switch payment.UpdatedAt.Compare(oldState.LastUpdatedAtFrom) {
		case -1, 0:
			// Payment already ingested, skip
			continue
		default:
		}

		raw, err := json.Marshal(payment)
		if err != nil {
			return nil, nil, err
		}

		paymentType := matchPaymentType(payment.Type)
		paymentStatus := matchPaymentStatus(payment.Status)

		var amount big.Int
		_, ok := amount.SetString(payment.Amount, 10)
		if !ok {
			return nil, nil, fmt.Errorf("failed to parse amount: %s", payment.Amount)
		}

		p := models.PSPPayment{
			Reference: payment.Id,
			CreatedAt: payment.CreatedAt,
			Type:      paymentType,
			Amount:    &amount,
			Asset:     payment.Currency, // UMN format from PSP: "USD/2", "BTC/8"
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    paymentStatus,
			Metadata:  payment.Metadata,
			Raw:       raw,
		}

		if payment.RelatedTransactionID != nil {
			p.Reference = *payment.RelatedTransactionID
		}

		if payment.SourceAccountID != nil {
			p.SourceAccountReference = payment.SourceAccountID
		}

		if payment.DestinationAccountID != nil {
			p.DestinationAccountReference = payment.DestinationAccountID
		}

		payments = append(payments, p)
		updatedAts = append(updatedAts, payment.UpdatedAt)
	}

	return payments, updatedAts, nil
}

func matchPaymentType(
	transactionType genericclient.TransactionType,
) models.PaymentType {
	switch transactionType {
	case genericclient.PAYIN:
		return models.PAYMENT_TYPE_PAYIN
	case genericclient.PAYOUT:
		return models.PAYMENT_TYPE_PAYOUT
	case genericclient.TRANSFER:
		return models.PAYMENT_TYPE_TRANSFER
	default:
		return models.PAYMENT_TYPE_OTHER
	}
}

func matchPaymentStatus(
	status genericclient.TransactionStatus,
) models.PaymentStatus {
	switch status {
	case genericclient.PENDING:
		return models.PAYMENT_STATUS_PENDING
	case genericclient.PROCESSING:
		return models.PAYMENT_STATUS_PROCESSING
	case genericclient.FAILED:
		return models.PAYMENT_STATUS_FAILED
	case genericclient.SUCCEEDED:
		return models.PAYMENT_STATUS_SUCCEEDED
	default:
		return models.PAYMENT_STATUS_OTHER
	}
}
