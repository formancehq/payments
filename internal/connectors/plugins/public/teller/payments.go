package teller

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/teller/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

type paymentsState struct {
	LastTransactionID string `json:"lastTransactionID"`
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

	// The accountID is carried in the inner FromPayload as a string
	var accountID string
	if err := json.Unmarshal(from.FromPayload, &accountID); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	accessToken := from.OpenBankingConnection.AccessToken.Token

	transactions, err := p.client.ListTransactions(
		ctx,
		accessToken,
		accountID,
		oldState.LastTransactionID,
		req.PageSize,
	)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, len(transactions))
	for _, tx := range transactions {
		payment, err := translateTellerTransactionToPSPPayment(tx, from.PSUID, from.OpenBankingConnection.ConnectionID)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
		payments = append(payments, payment)
	}

	newState := paymentsState{
		LastTransactionID: oldState.LastTransactionID,
	}
	if len(transactions) > 0 {
		newState.LastTransactionID = transactions[len(transactions)-1].ID
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  len(transactions) == req.PageSize,
	}, nil
}

func translateTellerTransactionToPSPPayment(tx client.Transaction, psuID uuid.UUID, connectionID string) (models.PSPPayment, error) {
	// Parse amount string to determine direction
	// Teller returns amounts as strings like "-100.50" or "200.00"
	amountStr := tx.Amount
	isNegative := strings.HasPrefix(amountStr, "-")
	if isNegative {
		amountStr = amountStr[1:]
	}

	var sourceAccountReference *string
	var destinationAccountReference *string
	var paymentType models.PaymentType
	if isNegative {
		// Negative amount = money leaving the account = PAYOUT
		paymentType = models.PAYMENT_TYPE_PAYOUT
		sourceAccountReference = &tx.AccountID
	} else {
		// Positive amount = money entering the account = PAYIN
		paymentType = models.PAYMENT_TYPE_PAYIN
		destinationAccountReference = &tx.AccountID
	}

	// Teller is USD-only for this prototype
	curr := "USD"
	precision, err := currency.GetPrecision(supportedCurrenciesWithDecimal, curr)
	if err != nil {
		return models.PSPPayment{}, err
	}

	amount, err := currency.GetAmountWithPrecisionFromString(amountStr, precision)
	if err != nil {
		return models.PSPPayment{}, err
	}

	// Ensure amount is non-negative (absolute value)
	if amount.Sign() < 0 {
		amount = new(big.Int).Abs(amount)
	}

	assetName := currency.FormatAssetWithPrecision(curr, precision)

	createdAt, err := time.Parse(time.DateOnly, tx.Date)
	if err != nil {
		createdAt = time.Now().UTC()
	}

	// Map Teller status to payment status
	var status models.PaymentStatus
	switch tx.Status {
	case "posted":
		status = models.PAYMENT_STATUS_SUCCEEDED
	case "pending":
		status = models.PAYMENT_STATUS_PENDING
	default:
		status = models.PAYMENT_STATUS_OTHER
	}

	raw, err := json.Marshal(tx)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return models.PSPPayment{
		Reference:                   tx.ID,
		CreatedAt:                   createdAt.UTC(),
		Type:                        paymentType,
		Amount:                      amount,
		Asset:                       assetName,
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      status,
		SourceAccountReference:      sourceAccountReference,
		DestinationAccountReference: destinationAccountReference,
		PsuID:                       &psuID,
		OpenBankingConnectionID:     &connectionID,
		Raw:                         raw,
	}, nil
}
