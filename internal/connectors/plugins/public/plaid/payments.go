package plaid

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
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

	var from models.BankBridgeFromPayload
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
		from.PSUBankBridgeConnection.AccessToken.Token,
		oldState.LastCursor,
		req.PageSize,
	)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)

	// TODO(polo): handle deleted and modfied payments
	for _, transaction := range resp.Added {
		payment, err := translatePlaidPaymentToPSPPayment(transaction)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
		payments = append(payments, payment)
	}

	newState.LastCursor = resp.NextCursor
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  resp.HasMore,
	}, nil
}

func translatePlaidPaymentToPSPPayment(transaction plaid.Transaction) (models.PSPPayment, error) {
	paymentType := models.PAYMENT_TYPE_PAYIN
	if transaction.Amount < 0 {
		paymentType = models.PAYMENT_TYPE_PAYOUT
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

	// TODO(polo): source/destination account
	payment := models.PSPPayment{
		Reference: transaction.TransactionId,
		CreatedAt: createdAt.UTC(),
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAsset(currency.ISO4217Currencies, curr),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Raw:       raw,
	}

	return payment, nil
}
