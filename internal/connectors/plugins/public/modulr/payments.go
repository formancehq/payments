package modulr

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/modulr/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type paymentsState struct {
	LastTransactionTime time.Time `json:"lastTransactionTime"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextPaymentsResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	newState := paymentsState{
		LastTransactionTime: oldState.LastTransactionTime,
	}

	var payments []models.PSPPayment
	needMore := false
	hasMore := false
	for page := 0; ; page++ {
		pagedTransactions, err := p.client.GetTransactions(ctx, from.Reference, page, req.PageSize, oldState.LastTransactionTime)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payments, err = p.fillPayments(ctx, pagedTransactions, from, payments, oldState, req.PageSize)
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

	if len(payments) > 0 {
		newState.LastTransactionTime = payments[len(payments)-1].CreatedAt
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

func (p *Plugin) fillPayments(
	ctx context.Context,
	pagedTransactions []client.Transaction,
	from models.PSPAccount,
	payments []models.PSPPayment,
	oldState paymentsState,
	pageSize int,
) ([]models.PSPPayment, error) {
	for _, transaction := range pagedTransactions {
		if len(payments) >= pageSize {
			break
		}

		createdTime, err := time.Parse("2006-01-02T15:04:05.999-0700", transaction.TransactionDate)
		if err != nil {
			return nil, err
		}

		switch createdTime.Compare(oldState.LastTransactionTime) {
		case -1, 0:
			// Account already ingested, skip
			continue
		default:
		}

		payment, err := p.transactionToPayment(ctx, transaction, from)
		if err != nil {
			return nil, err
		}

		if payment != nil {
			payments = append(payments, *payment)
		}
	}

	return payments, nil
}

func (p *Plugin) transactionToPayment(
	ctx context.Context,
	transaction client.Transaction,
	from models.PSPAccount,
) (*models.PSPPayment, error) {
	raw, err := json.Marshal(transaction)
	if err != nil {
		return nil, err
	}

	paymentType := matchTransactionType(transaction.Type)
	switch paymentType {
	case models.PAYMENT_TYPE_TRANSFER:
		// We want to fetch the transfer details in order to have the source
		// and destination account references
		return p.fetchAndTranslateTransfer(ctx, transaction)
	default:
	}

	precision, ok := supportedCurrenciesWithDecimal[transaction.Account.Currency]
	if !ok {
		return nil, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(transaction.Amount.String(), precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount %s: %w", transaction.Amount, err)
	}

	createdAt, err := time.Parse("2006-01-02T15:04:05.999-0700", transaction.PostedDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse posted date %s: %w", transaction.PostedDate, err)
	}

	payment := &models.PSPPayment{
		Reference: transaction.SourceID, // Do not take the transaction ID, but the source ID
		CreatedAt: createdAt,
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.Account.Currency),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Raw:       raw,
	}

	switch paymentType {
	case models.PAYMENT_TYPE_PAYIN:
		payment.DestinationAccountReference = &from.Reference
	case models.PAYMENT_TYPE_PAYOUT:
		payment.SourceAccountReference = &from.Reference
	default:
		if transaction.Credit {
			payment.DestinationAccountReference = &from.Reference
		} else {
			payment.SourceAccountReference = &from.Reference
		}
	}

	return payment, nil
}

func (p *Plugin) fetchAndTranslateTransfer(
	ctx context.Context,
	transaction client.Transaction,
) (*models.PSPPayment, error) {
	if !transaction.Credit {
		// Transfer are reprensented as double transactions: one for the source
		// account and one for the destination account. We don't want to generate
		// multiple events for the same transfer, and since we are fetching the
		// whole object, we can safely send it once. Let's ignore the transfer
		// if the transaction is a debit. It will be fetch on the other side (
		// the other account's transaction)
		return nil, nil
	}

	transfer, err := p.client.GetTransfer(ctx, transaction.SourceID)
	if err != nil {
		return nil, err
	}

	return translateTransferToPayment(&transfer)
}

func matchTransactionType(transactionType string) models.PaymentType {
	if transactionType == "PI_REV" ||
		transactionType == "PO_REV" ||
		transactionType == "ADHOC" {
		return models.PAYMENT_TYPE_OTHER
	}

	if transactionType == "INT_INTERC" {
		return models.PAYMENT_TYPE_TRANSFER
	}

	if strings.HasPrefix(transactionType, "PI_") {
		return models.PAYMENT_TYPE_PAYIN
	}

	if strings.HasPrefix(transactionType, "PO_") {
		return models.PAYMENT_TYPE_PAYOUT
	}

	return models.PAYMENT_TYPE_OTHER
}
