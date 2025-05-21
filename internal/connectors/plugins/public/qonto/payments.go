package qonto

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"maps"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type paymentsState struct {
	LastUpdatedAt            map[string]time.Time `json:"lastUpdatedAt"`
	TransactionStatusToFetch string               `json:"transactionStatusToFetch"`
	LastProcessedId          map[string]string    `json:"lastProcessedId"`
	Page                     map[string]int       `json:"page"`
}

/*
*
A fair bit of complexity here on the pagination.
Qonto doesn't let us get the list of all transactions, regardless of their status. For that reason, we need to fetch
transaction by status -- first all the pending ones, then declined, then completed.
In addition to that, we are using the lastUpdatedAt for state management. However, as in the ExternalAccount fetcher, there
is a possible edge case when a lot of transactions are changed at the same time (more than a page full), so we also
keep track of the current page in addition. 99% of the cases, the page in the state will be 1.

Another complexity is around transfer -- Qonto does not provide any common ID between a transfer and the subsequent
transaction, wo we have to make use of the Reference field to detect it, allowing us to populate the parentID accurately.
*/
func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	// Validation / Initialization
	from, err := validateAndGetAccount(req)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	oldState, err := initializeOldState(req)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	lastUpdatedAt := oldState.LastUpdatedAt[oldState.TransactionStatusToFetch]

	newState := paymentsState{
		LastUpdatedAt:            maps.Clone(oldState.LastUpdatedAt),
		TransactionStatusToFetch: oldState.TransactionStatusToFetch,
		LastProcessedId:          maps.Clone(oldState.LastProcessedId),
		Page:                     maps.Clone(oldState.Page),
	}
	// Run
	payments := make([]models.PSPPayment, 0, req.PageSize)
	hasMore := false

	transactions, err := p.client.GetTransactions(
		ctx,
		from.Reference,
		lastUpdatedAt,
		oldState.TransactionStatusToFetch,
		oldState.Page[oldState.TransactionStatusToFetch],
		req.PageSize,
	)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	lastProcessedId := oldState.LastProcessedId[oldState.TransactionStatusToFetch]
	payments, err = p.transactionsToPSPPayments(lastUpdatedAt, lastProcessedId, payments, transactions)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	_, hasMore = pagination.ShouldFetchMore(payments, transactions, req.PageSize)

	// State update
	if len(payments) > 0 {
		var err error
		newState.LastUpdatedAt[oldState.TransactionStatusToFetch], err = time.ParseInLocation(
 		client.QontoTimeformat,
			payments[len(payments)-1].Metadata["updated_at"],
			time.UTC,
		)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
		newState.LastProcessedId[oldState.TransactionStatusToFetch] = payments[len(payments)-1].Reference
	}

	// If the lastUpdatedAt is the same, increment the page number
	if len(payments) > 0 && newState.LastUpdatedAt[oldState.TransactionStatusToFetch].Equal(oldState.LastUpdatedAt[oldState.TransactionStatusToFetch]) {
		newState.Page[oldState.TransactionStatusToFetch]++
	} else {
		newState.Page[oldState.TransactionStatusToFetch] = 1
	}

	if !hasMore {
		switch oldState.TransactionStatusToFetch {
		case client.TransactionStatusPending:
			newState.TransactionStatusToFetch = client.TransactionStatusDeclined
			hasMore = true
		case client.TransactionStatusDeclined:
			newState.TransactionStatusToFetch = client.TransactionStatusCompleted
			hasMore = true
		case client.TransactionStatusCompleted:
			newState.TransactionStatusToFetch = client.TransactionStatusPending
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

func initializeOldState(req models.FetchNextPaymentsRequest) (paymentsState, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			err := errorsutils.NewWrappedError(
				fmt.Errorf("failed to unmarshall state"),
				err,
			)
			return paymentsState{}, err
		}
	}

	if oldState.TransactionStatusToFetch == "" {
		oldState.TransactionStatusToFetch = client.TransactionStatusPending
	}
	if oldState.LastUpdatedAt == nil {
		oldState.LastUpdatedAt = make(map[string]time.Time)
	}
	if oldState.LastProcessedId == nil {
		oldState.LastProcessedId = make(map[string]string)
	}
	if oldState.Page == nil {
		oldState.Page = make(map[string]int)
	}
	if oldState.Page[oldState.TransactionStatusToFetch] == 0 {
		oldState.Page[oldState.TransactionStatusToFetch] = 1
	}
	return oldState, nil
}

func (p *Plugin) transactionsToPSPPayments(
	oldUpdatedAt time.Time,
	lastProcessedId string,
	payments []models.PSPPayment,
	transactions []client.Transactions,
) ([]models.PSPPayment, error) {
	for _, transaction := range transactions {
		updatedAt, err := time.ParseInLocation(client.QontoTimeformat, transaction.UpdatedAt, time.UTC)
		if err != nil {
			err := errorsutils.NewWrappedError(
				fmt.Errorf("invalid time format for updatedAt transaction"),
				err,
			)
			return payments, err
		}
		if updatedAt.Before(oldUpdatedAt) || (updatedAt.Equal(oldUpdatedAt) && transaction.Id == lastProcessedId) {
			continue
		}

		emittedAt, err := time.ParseInLocation(client.QontoTimeformat, transaction.EmittedAt, time.UTC)
		if err != nil {
			err := errorsutils.NewWrappedError(
				fmt.Errorf("invalid time format for emittedAt transaction"),
				err,
			)
			return payments, err
		}
		raw, err := json.Marshal(transaction)
		if err != nil {
			return payments, err
		}

		payment := models.PSPPayment{
			ParentReference:             transaction.Id,
			Reference:                   transaction.Id,
			CreatedAt:                   emittedAt,
			Type:                        mapQontoTransactionType(transaction.SubjectType),
			Amount:                      big.NewInt(transaction.AmountCents),
			Asset:                       currency.FormatAsset(supportedCurrenciesForInternalAccounts, transaction.Currency),
			Scheme:                      mapQontoTransactionScheme(transaction.SubjectType),
			Status:                      mapQontoPaymentStatus(transaction.Status),
			SourceAccountReference:      &transaction.BankAccountId,
			DestinationAccountReference: nil,
			Raw:                         raw,
			Metadata: map[string]string{
				"updated_at": transaction.UpdatedAt,
			},
		}

		// Set DestinationAccountReference, which needs to match the externalAccount's format (see generateAccountReference in external_accounts.go)
		// Worth noting that we don't have the intermediaryBankBic information here, but it's not necessary for account uniqueness
		var destinationAccountDetails *client.CounterpartyDetails
		switch transaction.SubjectType {
		case "Transfer":
			destinationAccountDetails = transaction.Transfer
		case "DirectDebit":
			destinationAccountDetails = transaction.DirectDebit
		case "DirectDebitCollection":
			destinationAccountDetails = transaction.DirectDebitCollection
		case "Income":
			destinationAccountDetails = transaction.Income
		case "SwiftIncome":
			destinationAccountDetails = transaction.SwiftIncome
		}
		if destinationAccountDetails != nil {
			payment.DestinationAccountReference = pointer.For(
				destinationAccountDetails.CounterpartyAccountNumber + "-" + destinationAccountDetails.CounterpartyBankIdentifier,
			)
		}

		payments = append(payments, payment)
	}
	return payments, nil
}

func mapQontoPaymentStatus(status string) models.PaymentStatus {
	switch status {
	case client.TransactionStatusCompleted:
		return models.PAYMENT_STATUS_SUCCEEDED
	case client.TransactionStatusDeclined:
		return models.PAYMENT_STATUS_FAILED
	case client.TransactionStatusPending:
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

func validateAndGetAccount(req models.FetchNextPaymentsRequest) (models.PSPAccount, error) {
	if req.PageSize == 0 {
		return models.PSPAccount{}, models.ErrMissingPageSize
	}

	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.PSPAccount{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		err := errorsutils.NewWrappedError(
			fmt.Errorf("failed to unmarshall FromPayload"),
			err,
		)
		return models.PSPAccount{}, err
	}
	return from, nil
}
