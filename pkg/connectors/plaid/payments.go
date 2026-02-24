package plaid

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/formancehq/payments/pkg/connectors/plaid/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/google/uuid"
	"github.com/plaid/plaid-go/v34/plaid"
)

type paymentsState struct {
	LastCursor string `json:"lastCursor"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req connector.FetchNextPaymentsRequest) (connector.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}
	}

	var from connector.OpenBankingForwardedUserFromPayload
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	var baseWebhook client.BaseWebhooks
	if err := json.Unmarshal(from.FromPayload, &baseWebhook); err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	newState := paymentsState{
		LastCursor: oldState.LastCursor,
	}

	resp, err := p.client.ListTransactions(
		ctx,
		from.OpenBankingConnection.AccessToken.Token,
		oldState.LastCursor,
		req.PageSize,
	)
	if err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	payments := make([]connector.PSPPayment, 0, req.PageSize)
	paymentsToDelete := make([]connector.PSPPaymentsToDelete, 0, req.PageSize)

	for _, transaction := range resp.Added {
		payment, err := translatePlaidPaymentToPSPPayment(transaction, from.PSUID, from.OpenBankingConnection.ConnectionID)
		if err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}
		payments = append(payments, payment)
	}

	for _, transaction := range resp.Modified {
		payment, err := translatePlaidPaymentToPSPPayment(transaction, from.PSUID, from.OpenBankingConnection.ConnectionID)
		if err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}
		payments = append(payments, payment)
	}

	for _, transaction := range resp.Removed {
		paymentsToDelete = append(paymentsToDelete, connector.PSPPaymentsToDelete{
			Reference: transaction.TransactionId,
		})
	}

	newState.LastCursor = resp.NextCursor
	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	return connector.FetchNextPaymentsResponse{
		Payments:         payments,
		PaymentsToDelete: paymentsToDelete,
		NewState:         payload,
		HasMore:          resp.HasMore,
	}, nil
}

func translatePlaidPaymentToPSPPayment(transaction plaid.Transaction, psuID uuid.UUID, connectionID string) (connector.PSPPayment, error) {
	var sourceAccountReference *string
	var destinationAccountReference *string
	var paymentType connector.PaymentType
	switch {
	case transaction.Amount > 0:
		paymentType = connector.PAYMENT_TYPE_PAYOUT
		sourceAccountReference = &transaction.AccountId
	default:
		paymentType = connector.PAYMENT_TYPE_PAYIN
		destinationAccountReference = &transaction.AccountId
	}

	var curr string
	if transaction.IsoCurrencyCode.IsSet() {
		curr = *transaction.IsoCurrencyCode.Get()
	} else {
		curr = transaction.GetUnofficialCurrencyCode()
	}

	amount, assetName, err := client.TranslatePlaidAmount(math.Abs(transaction.Amount), curr)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	dateTime, okDateTime := transaction.GetDatetimeOk()
	authorizedDateTime, okAuthorizedDateTime := transaction.GetAuthorizedDatetimeOk()
	date, okDate := transaction.GetDateOk()
	authorizedDate, okAuthorizedDate := transaction.GetAuthorizedDateOk()
	var createdAt time.Time
	switch {
	case okAuthorizedDateTime:
		createdAt = *authorizedDateTime
	case okDateTime:
		createdAt = *dateTime
	case okAuthorizedDate:
		createdAt, err = time.Parse(time.DateOnly, *authorizedDate)
		if err != nil {
			return connector.PSPPayment{}, err
		}
	case okDate:
		createdAt, err = time.Parse(time.DateOnly, *date)
		if err != nil {
			return connector.PSPPayment{}, err
		}
	}

	raw, err := json.Marshal(transaction)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	payment := connector.PSPPayment{
		Reference:                   transaction.TransactionId,
		CreatedAt:                   createdAt.UTC(),
		Type:                        paymentType,
		Amount:                      amount,
		Asset:                       assetName,
		Scheme:                      connector.PAYMENT_SCHEME_OTHER,
		Status:                      connector.PAYMENT_STATUS_SUCCEEDED,
		SourceAccountReference:      sourceAccountReference,
		DestinationAccountReference: destinationAccountReference,
		PsuID:                       &psuID,
		OpenBankingConnectionID:     &connectionID,
		Raw:                         raw,
	}

	return payment, nil
}
