package currencycloud

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/currencycloud/client"
	"github.com/formancehq/payments/pkg/connector"
)

type paymentsState struct {
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req connector.FetchNextPaymentsRequest) (connector.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}
	}

	newState := paymentsState{
		LastUpdatedAt: oldState.LastUpdatedAt,
	}

	var payments []connector.PSPPayment
	var updatedAts []time.Time
	hasMore := false
	page := 1
	for {
		pagedTransactions, nextPage, err := p.client.GetTransactions(ctx, page, req.PageSize, newState.LastUpdatedAt)
		if err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}

		if len(pagedTransactions) == 0 {
			break
		}

		payments, updatedAts, err = fillPayments(payments, updatedAts, pagedTransactions, newState)
		if err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}

		var needMore bool
		needMore, hasMore, payments = shouldFetchMore(payments, nextPage, req.PageSize)

		if len(payments) > 0 {
			newState.LastUpdatedAt = updatedAts[len(payments)-1]
		}

		if !needMore {
			break
		}

		if len(payments) > 0 {
			newState.LastUpdatedAt = updatedAts[len(payments)-1]
		}

		page = nextPage
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	return connector.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func fillPayments(
	payments []connector.PSPPayment,
	updatedAts []time.Time,
	pagedTransactions []client.Transaction,
	newState paymentsState,
) ([]connector.PSPPayment, []time.Time, error) {
	for _, transaction := range pagedTransactions {
		switch transaction.UpdatedAt.Compare(newState.LastUpdatedAt) {
		case -1, 0:
			continue
		default:
		}

		payment, err := transactionToPayment(transaction)
		if err != nil {
			return nil, nil, err
		}

		if payment != nil {
			payments = append(payments, *payment)
			updatedAts = append(updatedAts, transaction.UpdatedAt)
		}
	}

	return payments, updatedAts, nil
}

func transactionToPayment(transaction client.Transaction) (*connector.PSPPayment, error) {
	raw, err := json.Marshal(transaction)
	if err != nil {
		return nil, err
	}

	precision, ok := supportedCurrenciesWithDecimal[transaction.Currency]
	if !ok {
		return nil, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(transaction.Amount.String(), precision)
	if err != nil {
		return nil, err
	}

	paymentType := matchTransactionType(transaction.RelatedEntityType, transaction.Type)

	reference := transaction.RelatedEntityID
	if reference == "" {
		reference = transaction.ID
	}
	payment := &connector.PSPPayment{
		Reference: reference,
		CreatedAt: transaction.CreatedAt,
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.Currency),
		Scheme:    connector.PAYMENT_SCHEME_OTHER,
		Status:    matchTransactionStatus(transaction.Status),
		Raw:       raw,
	}

	switch paymentType {
	case connector.PAYMENT_TYPE_PAYOUT:
		payment.SourceAccountReference = &transaction.AccountID
	case connector.PAYMENT_TYPE_PAYIN:
		payment.DestinationAccountReference = &transaction.AccountID
	}

	return payment, nil
}

func matchTransactionType(entityType string, transactionType string) connector.PaymentType {
	switch entityType {
	case "inbound_funds":
		return connector.PAYMENT_TYPE_PAYIN
	case "payment":
		return connector.PAYMENT_TYPE_PAYOUT
	case "transfer", "balance_transfer":
		return connector.PAYMENT_TYPE_TRANSFER
	default:
		switch transactionType {
		case "credit":
			return connector.PAYMENT_TYPE_PAYIN
		case "debit":
			return connector.PAYMENT_TYPE_PAYOUT
		}
	}

	return connector.PAYMENT_TYPE_OTHER
}

func matchTransactionStatus(transactionStatus string) connector.PaymentStatus {
	switch transactionStatus {
	case "completed":
		return connector.PAYMENT_STATUS_SUCCEEDED
	case "pending", "ready_to_send":
		return connector.PAYMENT_STATUS_PENDING
	case "deleted":
		return connector.PAYMENT_STATUS_FAILED
	case "cancelled":
		return connector.PAYMENT_STATUS_CANCELLED
	}
	return connector.PAYMENT_STATUS_OTHER
}
