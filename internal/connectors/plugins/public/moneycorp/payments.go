package moneycorp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
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
	// Page is the next page to fetch within the current LastCreatedAt second
	// (0-indexed). It advances while the watermark second is unchanged (a
	// same-second group larger than one page) and resets to 0 once the watermark
	// moves to a newer second, so a same-second group spanning pages is walked
	// without re-scanning from page 0 each cycle (which a single LastProcessedID
	// cannot do).
	Page int `json:"page"`
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
		Page:            oldState.Page,
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	processedIDs := make([]string, 0, req.PageSize)
	needMore := false
	hasMore := false
	// Resume at the persisted page and walk forward (no trim-and-restart, which
	// would skip the trimmed rows); the page cursor below records how far we got.
	page := oldState.Page
	for {
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
		page++
	}

	if len(payments) > 0 {
		newState.LastCreatedAt = payments[len(payments)-1].CreatedAt
		newState.LastProcessedID = processedIDs[len(processedIDs)-1]
		// Advance past the consumed pages only while there is definitely a full
		// next page (hasMore). If the same-second group drained on a short final
		// page, keep the cursor there — a newer row appended to that second's
		// >= watermark query lands on this very page, so advancing past it would
		// strand it forever. When the watermark moved to a newer second, re-anchor
		// at page 0.
		if newState.LastCreatedAt.Equal(oldState.LastCreatedAt) {
			if hasMore {
				newState.Page = page + 1
			} else {
				newState.Page = page
			}
		} else {
			newState.Page = 0
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
