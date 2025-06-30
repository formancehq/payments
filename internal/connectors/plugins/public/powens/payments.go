package powens

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	LastUpdate time.Time
}

func (p *Plugin) fetchPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
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

	var webhook client.AccountSyncedWebhook
	if err := json.Unmarshal(from.FromPayload, &webhook); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	newState := paymentsState{
		LastUpdate: webhook.LastUpdate,
	}

	resp, err := p.client.ListTransactions(ctx, from.PSUBankBridge.AccessToken.Token, oldState.LastUpdate, req.PageSize)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, len(resp.Transactions))
	for _, transaction := range resp.Transactions {
		bankAccount, err := p.client.GetBankAccount(ctx, from.PSUBankBridge.AccessToken.Token, transaction.AccountID)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payment, err := translateTransactionToPSPPayment(transaction, bankAccount)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
		payments = append(payments, payment)
		newState.LastUpdate = transaction.LastUpdate
	}

	hasMore := resp.Links.Next.Href != ""

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

func translateTransactionToPSPPayment(transaction client.Transaction, bankAccount client.BankAccount) (models.PSPPayment, error) {
	paymentType := models.PAYMENT_TYPE_PAYIN
	if transaction.Value < 0 {
		paymentType = models.PAYMENT_TYPE_PAYOUT
	}

	amountString := strconv.FormatFloat(math.Abs(transaction.Value), 'f', -1, 64)

	precision := bankAccount.Currency.Precision

	amount, err := currency.GetAmountWithPrecisionFromString(amountString, precision)
	if err != nil {
		return models.PSPPayment{}, err
	}

	raw, err := json.Marshal(transaction)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return models.PSPPayment{
		Reference: strconv.Itoa(transaction.ID),
		CreatedAt: transaction.Date,
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAssetWithPrecision(bankAccount.Currency.ID, precision),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Raw:       raw,
	}, nil
}
