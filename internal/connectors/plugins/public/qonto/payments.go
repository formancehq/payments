package qonto

import (
	"context"
	"encoding/json"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type paymentsState struct {
	LastPage      int       `json:"lastPage"`
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	// TODO Parent refernece => ID of the payout/payin etc (what's related to the transaction)
	// Reference: id of the transaction
	// Test if transaction ID is a reference to a payout / payin

	if req.PageSize == 0 {
		return models.FetchNextPaymentsResponse{}, models.ErrMissingPageSize
	}
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}
	if oldState.LastPage == 0 {
		oldState.LastPage = 1
	}
	if oldState.LastUpdatedAt.IsZero() {
		oldState.LastUpdatedAt = time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC) // Qonto returns an error for date < 2017
	}

	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextPaymentsResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	newState := paymentsState{
		LastPage:      oldState.LastPage,
		LastUpdatedAt: oldState.LastUpdatedAt,
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	needMore := false
	hasMore := false
	for page := oldState.LastPage; ; page++ {
		newState.LastPage = page
		pagedTransactions, err := p.client.GetTransactions(
			ctx,
			from.Reference,
			oldState.LastUpdatedAt,
			page,
			req.PageSize,
		)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payments, err = p.transactionsToPSPPayments(oldState.LastUpdatedAt, payments, pagedTransactions)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(payments, pagedTransactions, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		payments = payments[:req.PageSize]
	}

	// TODO updating both updatedAt & page will probably not work
	if len(payments) > 0 {
		var err error
		newState.LastUpdatedAt, err = time.ParseInLocation(client.QONTO_TIMEFORMAT, payments[len(payments)-1].Metadata["updated_at"], time.UTC)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
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

func (p *Plugin) transactionsToPSPPayments(
	oldUpdatedAt time.Time,
	payments []models.PSPPayment,
	transactions []client.Transactions,
) ([]models.PSPPayment, error) {
	for _, transaction := range transactions {
		updatedAt, err := time.ParseInLocation(client.QONTO_TIMEFORMAT, transaction.UpdatedAt, time.UTC)
		if err != nil {
			return payments, err
		}
		if updatedAt.Before(oldUpdatedAt) || updatedAt.Equal(oldUpdatedAt) {
			continue
		}

		emittedAt, err := time.ParseInLocation(client.QONTO_TIMEFORMAT, transaction.EmittedAt, time.UTC)
		if err != nil {
			return payments, err
		}
		raw, err := json.Marshal(transaction)
		if err != nil {
			return payments, err
		}

		payment := models.PSPPayment{
			ParentReference:             "",
			Reference:                   transaction.Id,
			CreatedAt:                   emittedAt,
			Type:                        mapQontoTransactionType(transaction.SubjectType),
			Amount:                      big.NewInt(transaction.AmountCents),
			Asset:                       currency.FormatAsset(supportedCurrenciesForInternalAccounts, transaction.Currency),
			Scheme:                      mapQontoTransactionScheme(transaction.SubjectType),
			Status:                      mapQontoPaymentStatus(transaction.Status),
			SourceAccountReference:      &transaction.BankAccountId,
			DestinationAccountReference: &transaction.Transfer.CounterpartyAccountNumber, // TODO, it's not always a transfer
			Raw:                         raw,
			Metadata: map[string]string{
				"updated_at": transaction.UpdatedAt,
			},
		}
		payments = append(payments, payment)
	}
	return payments, nil
}

func mapQontoPaymentStatus(status string) models.PaymentStatus {
	switch status {
	case "completed":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "declined":
		return models.PAYMENT_STATUS_FAILED
	case "pending":
		return models.PAYMENT_STATUS_PENDING
	}
	return models.PAYMENT_STATUS_UNKNOWN
}

func mapQontoTransactionType(transactionType string) models.PaymentType {

	switch transactionType {
	case "Income", "SwiftIncome", "DirectDebitCollection", "FinancingIncome":
		return models.PAYMENT_TYPE_PAYIN

	case "Check", "DirectDebit", "BillingTransfer", "FinancingInstallment", "Transfer", "Card", "PagodaPayment", "F24Payment":
		return models.PAYMENT_TYPE_PAYOUT

	case "WalletToWallet":
		return models.PAYMENT_TYPE_TRANSFER

	case "DirectDebitHold":
	case "Other":
		return models.PAYMENT_TYPE_OTHER
	}
	return models.PAYMENT_TYPE_UNKNOWN
}

func mapQontoTransactionScheme(transactionType string) models.PaymentScheme {
	switch transactionType {
	case "DirectDebit":
		return models.PAYMENT_SCHEME_SEPA_DEBIT
	case "DirectDebitCollection":
		return models.PAYMENT_SCHEME_SEPA_CREDIT
	}

	return models.PAYMENT_SCHEME_UNKNOWN
}
