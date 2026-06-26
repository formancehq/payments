package moneycorp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/payments/ce/plugins/moneycorp/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
	"github.com/formancehq/payments/pkg/domain/plugins"
)

type paymentsState struct {
	LastCreatedAt time.Time `json:"lastCreatedAt"`
	// LastProcessedID is the transaction ID of the last transaction emitted at
	// exactly LastCreatedAt. It lets the watermark filter be inclusive (>=)
	// without re-skipping distinct transactions that share that timestamp: only
	// the exact already-processed row is excluded. Keyed on transaction.ID (the
	// iteration identity), which differs from payment.Reference for "Payment"
	// types.
	LastProcessedID string `json:"lastProcessedID"`
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
		LastCreatedAt:   oldState.LastCreatedAt,
		LastProcessedID: oldState.LastProcessedID,
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	processedIDs := make([]string, 0, req.PageSize)
	needMore := false
	hasMore := false
	for page := 0; ; page++ {
		pagedTransactions, err := p.client.GetTransactions(ctx, from.Reference, page, req.PageSize, oldState.LastCreatedAt)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payments, processedIDs, err = p.toPSPPayments(ctx, oldState.LastCreatedAt, oldState.LastProcessedID, payments, processedIDs, pagedTransactions)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(payments, pagedTransactions, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		// Trim both slices in lockstep so the watermark and LastProcessedID come
		// from the last EMITTED payment.
		payments = payments[:req.PageSize]
		processedIDs = processedIDs[:req.PageSize]
	}

	if len(payments) > 0 {
		newState.LastCreatedAt = payments[len(payments)-1].CreatedAt
		newState.LastProcessedID = processedIDs[len(processedIDs)-1]
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

func (p *Plugin) toPSPPayments(
	ctx context.Context,
	lastCreatedAt time.Time,
	lastProcessedID string,
	payments []models.PSPPayment,
	processedIDs []string,
	transactions []*client.Transaction,
) ([]models.PSPPayment, []string, error) {
	for _, transaction := range transactions {
		createdAt, err := time.Parse("2006-01-02T15:04:05.999999999", transaction.Attributes.CreatedAt)
		if err != nil {
			return payments, processedIDs, fmt.Errorf("failed to parse transaction date: %v", err)
		}

		// Inclusive watermark: skip transactions strictly before it, and the
		// single already-processed transaction at exactly the watermark. Distinct
		// transactions sharing that timestamp are kept (M-CON2).
		cmp := createdAt.Compare(lastCreatedAt)
		if cmp < 0 || (cmp == 0 && transaction.ID == lastProcessedID) {
			continue
		}

		payment, err := p.transactionToPayment(ctx, transaction)
		if err != nil {
			if errors.Is(err, plugins.ErrCurrencyNotSupported) {
				// Skip unsupported currencies rather than failing: a retryable
				// error here would freeze ingestion for the whole account.
				p.logger.WithField("transaction_id", transaction.ID).Info("skipping transaction with unsupported currency")
				continue
			}
			return payments, processedIDs, err
		}
		if payment == nil {
			continue
		}

		payments = append(payments, *payment)
		processedIDs = append(processedIDs, transaction.ID)
	}
	return payments, processedIDs, nil
}

func (p *Plugin) transactionToPayment(ctx context.Context, transaction *client.Transaction) (*models.PSPPayment, error) {
	switch transaction.Attributes.Type {
	case "Transfer":
		if transaction.Attributes.Direction == "Debit" {
			return p.fetchAndTranslateTransfer(ctx, transaction)
		} else {
			// Do not fetch the transfer, it does not exists if we're trying to
			// fetch it with the destination account.
			return nil, nil
		}
	}

	rawData, err := json.Marshal(transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	paymentType, shouldBeRecorded := matchPaymentType(transaction.Attributes.Type, transaction.Attributes.Direction)
	if !shouldBeRecorded {
		return nil, nil
	}

	createdAt, err := time.Parse("2006-01-02T15:04:05.999999999", transaction.Attributes.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction date: %w", err)
	}

	c, err := currency.GetPrecision(supportedCurrenciesWithDecimal, transaction.Attributes.Currency)
	if err != nil {
		if errors.Is(err, currency.ErrMissingCurrencies) {
			return nil, fmt.Errorf("%w: %w", plugins.ErrCurrencyNotSupported, err)
		}
		return nil, err
	}

	amount, err := currency.GetAmountWithPrecisionFromString(transaction.Attributes.Amount.String(), c)
	if err != nil {
		return nil, err
	}

	reference := transaction.ID
	if transaction.Attributes.Type == "Payment" {
		// In case of payments (related to payouts), we want to take the real
		// object id as a reference
		reference = transaction.Relationships.Data.ID
	}

	payment := models.PSPPayment{
		Reference: reference,
		CreatedAt: createdAt,
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.Attributes.Currency),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Raw:       rawData,
	}

	switch paymentType {
	case models.PAYMENT_TYPE_PAYIN:
		payment.DestinationAccountReference = pointer.For(strconv.Itoa(int(transaction.Attributes.AccountID)))
	case models.PAYMENT_TYPE_PAYOUT:
		payment.SourceAccountReference = pointer.For(strconv.Itoa(int(transaction.Attributes.AccountID)))
	default:
		if transaction.Attributes.Direction == "Debit" {
			payment.SourceAccountReference = pointer.For(strconv.Itoa(int(transaction.Attributes.AccountID)))
		} else {
			payment.DestinationAccountReference = pointer.For(strconv.Itoa(int(transaction.Attributes.AccountID)))
		}
	}

	return &payment, nil
}

func (p *Plugin) fetchAndTranslateTransfer(ctx context.Context, transaction *client.Transaction) (*models.PSPPayment, error) {
	transfer, err := p.client.GetTransfer(ctx, fmt.Sprint(transaction.Attributes.AccountID), transaction.Relationships.Data.ID)
	if err != nil {
		return nil, err
	}

	return transferToPayment(transfer)
}

func matchPaymentType(transactionType string, transactionDirection string) (models.PaymentType, bool) {
	switch transactionType {
	case "Transfer":
		return models.PAYMENT_TYPE_TRANSFER, true
	case "Payment", "Exchange", "Charge", "Refund":
		switch transactionDirection {
		case "Debit":
			return models.PAYMENT_TYPE_PAYOUT, true
		case "Credit":
			return models.PAYMENT_TYPE_PAYIN, true
		}
	}

	return models.PAYMENT_TYPE_OTHER, false
}
