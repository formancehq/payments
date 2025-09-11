package plaid

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/plaid/plaid-go/v34/plaid"
)

type paymentsState struct {
	LastCursor string `json:"lastCursor"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	var from models.OpenBankingForwardedUserFromPayload
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	var baseWebhook client.BaseWebhooks
	if err := json.Unmarshal(from.FromPayload, &baseWebhook); err != nil {
		return models.FetchNextPaymentsResponse{}, err
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
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	paymentsToDelete := make([]models.PSPPaymentsToDelete, 0, req.PageSize)

	for _, transaction := range resp.Added {
		payment, err := translatePlaidPaymentToPSPPayment(transaction, from.PSUID, from.OpenBankingConnection.ConnectionID)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
		payments = append(payments, payment)
	}

	for _, transaction := range resp.Modified {
		payment, err := translatePlaidPaymentToPSPPayment(transaction, from.PSUID, from.OpenBankingConnection.ConnectionID)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
		payments = append(payments, payment)
	}

	for _, transaction := range resp.Removed {
		paymentsToDelete = append(paymentsToDelete, models.PSPPaymentsToDelete{
			Reference: transaction.TransactionId,
		})
	}

	newState.LastCursor = resp.NextCursor
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments:         payments,
		PaymentsToDelete: paymentsToDelete,
		NewState:         payload,
		HasMore:          resp.HasMore,
	}, nil
}

func translatePlaidPaymentToPSPPayment(transaction plaid.Transaction, psuID uuid.UUID, connectionID string) (models.PSPPayment, error) {
	var sourceAccountReference *string
	var destinationAccountReference *string
	var paymentType models.PaymentType
	switch {
	case transaction.Amount > 0:
		paymentType = models.PAYMENT_TYPE_PAYOUT
		sourceAccountReference = &transaction.AccountId
	default:
		paymentType = models.PAYMENT_TYPE_PAYIN
		destinationAccountReference = &transaction.AccountId
	}

	amountString := strconv.FormatFloat(math.Abs(transaction.Amount), 'f', -1, 64)

	var curr string
	if transaction.IsoCurrencyCode.IsSet() {
		curr = *transaction.IsoCurrencyCode.Get()
	} else {
		curr = transaction.GetUnofficialCurrencyCode()
	}

	precision, err := currency.GetPrecision(currency.ISO4217Currencies, curr)
	if err != nil {
		return models.PSPPayment{}, err
	}

	amount, err := currency.GetAmountWithPrecisionFromString(amountString, precision)
	if err != nil {
		return models.PSPPayment{}, err
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
			return models.PSPPayment{}, err
		}
	case okDate:
		createdAt, err = time.Parse(time.DateOnly, *date)
		if err != nil {
			return models.PSPPayment{}, err
		}
	}

	raw, err := json.Marshal(transaction)
	if err != nil {
		return models.PSPPayment{}, err
	}

	payment := models.PSPPayment{
		Reference:                   transaction.TransactionId,
		CreatedAt:                   createdAt.UTC(),
		Type:                        paymentType,
		Amount:                      amount,
		Asset:                       currency.FormatAsset(currency.ISO4217Currencies, curr),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      models.PAYMENT_STATUS_SUCCEEDED,
		SourceAccountReference:      sourceAccountReference,
		DestinationAccountReference: destinationAccountReference,
		PsuID:                       &psuID,
		OpenBankingConnectionID:     &connectionID,
		Raw:                         raw,
	}

	return payment, nil
}
